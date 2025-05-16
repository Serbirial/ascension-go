package handlers

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"sync"
	"time"

	"ascension/models"
	"ascension/utils/fs"

	"golang.org/x/net/websocket"
)

var (
	Clients = make(map[string]*models.Client)
	Loops   = make(map[string]chan bool)
	Seeks   = make(map[string]chan int)

	clientsMu sync.Mutex
	loopsMu   sync.Mutex
	seeksMu   sync.Mutex
)

func RecvByteData(ws *websocket.Conn, output chan []byte, stop <-chan bool) {
	for {
		select {
		case <-stop:
			log.Println("[WS-BYTE-RECV] Stop signal received")
			return
		default:
			var data []byte
			err := websocket.Message.Receive(ws, &data)
			if err != nil {
				fmt.Println("[WS-BYTE-RECV]", "Receive error:", err)
				return
			}
			output <- data
		}
	}
}

func sendByteData(ws *websocket.Conn, song *models.SongInfo, stop <-chan bool, seek <-chan int) {
	ticker := time.NewTicker(5 * time.Millisecond)
	defer ticker.Stop()
	log.Println("[WS] Streaming started")

	file, err := os.Open(song.FilePath)
	if err != nil {
		log.Println("Error opening dca file:", err)
		return
	}
	defer file.Close()

	const frameDuration = 20 * time.Millisecond
	const frameRateDCA = int(time.Second / frameDuration) // 50 fps

	// Build frame index once at start
	frameIndex, err := buildFrameIndex(file)
	if err != nil {
		log.Println("[WS] Error building frame index:", err)
		return
	}
	if len(frameIndex) == 0 {
		log.Println("[WS] Frame index is empty, aborting playback")
		return
	}

	// Buffer pool for Opus frames
	frameBufPool := sync.Pool{
		New: func() any { return make([]byte, 2048) },
	}

	// Playback state variables
	var currentFrame int = 0

	// Helper to seek to a frame in file & update currentFrame
	seekToFrame := func(targetFrame int) error {
		if targetFrame < 0 {
			targetFrame = 0
		}
		if targetFrame >= len(frameIndex) {
			targetFrame = len(frameIndex) - 1
		}

		seekPos := frameIndex[targetFrame]
		pos, err := file.Seek(seekPos, io.SeekStart)
		if err != nil {
			return err
		}
		currentFrame = targetFrame
		log.Printf("[WS] Seeked to frame %d (byte offset %d, actual file pos %d)", targetFrame, seekPos, pos)
		return nil
	}

	// Start playback from beginning
	if err := seekToFrame(0); err != nil {
		log.Println("[WS] Initial seek error:", err)
		return
	}

	var pendingSeek *int

	for {
		select {
		case <-stop:
			log.Println("[WS] Streaming stopped")
			return

		case seconds := <-seek:
			// Drain further seek requests to get the latest one
		drain:
			for {
				select {
				case s := <-seek:
					seconds = s
				default:
					break drain
				}
			}
			pendingSeek = &seconds
		case <-ticker.C:
			// If there's a pending seek, perform it now
			if pendingSeek != nil {
				targetFrame := *pendingSeek * frameRateDCA
				log.Printf("[WS] Seeking: currentFrame=%d, requestedSeconds=%d, targetFrame=%d", currentFrame, *pendingSeek, targetFrame)
				if err := seekToFrame(targetFrame); err != nil {
					log.Println("[WS] Seek error:", err)
				}
				pendingSeek = nil
				continue
			}
			if currentFrame >= len(frameIndex) {
				log.Println("[WS] Reached end of stream")
				_ = websocket.Message.Send(ws, []byte("DONE"))
				return
			}

			// Read Opus frame length
			var opuslen int16
			if err := binary.Read(file, binary.LittleEndian, &opuslen); err != nil {
				if err == io.EOF || err == io.ErrUnexpectedEOF {
					log.Println("[WS] EOF reached")
					_ = websocket.Message.Send(ws, []byte("DONE"))
					return
				}
				log.Println("[WS] Error reading frame length:", err)
				return
			}

			// Get buffer from pool or allocate if too small
			buf := frameBufPool.Get().([]byte)
			if int(opuslen) > cap(buf) {
				buf = make([]byte, opuslen)
			}
			frame := buf[:opuslen]

			// Read Opus frame data
			if _, err := io.ReadFull(file, frame); err != nil {
				log.Println("[WS] Error reading frame data:", err)
				return
			}

			// Send frame to client
			if err := websocket.Message.Send(ws, frame); err != nil {
				log.Println("[WS] Error sending frame:", err)
				return
			}

			currentFrame++
			frameBufPool.Put(buf)
		}
	}
}

func HandleWebSocket(ws *websocket.Conn) {
	defer ws.Close()

	log.Println("[WS] Connected: ", ws.RemoteAddr())
	var tempConnection bool = true // Assume temp connection
	var name string = ""
	var identifier string = ""

	for {
		var msg models.Message
		// Read JSON message
		if err := websocket.JSON.Receive(ws, &msg); err != nil {
			if tempConnection {
				log.Println("[WS] Communication connection Closed:", name, identifier, "-", err)
				break
			} else {
				log.Println("[WS] Streaming connection Closed:", name, identifier, "-", err)
				break

			}
		}

		// Set the reference and identifier
		name = msg.From
		identifier = msg.Identifier

		// Register new clients after they send identifier (first recv)
		clientsMu.Lock()

		// Client already might exist (ex: is streaming from the server, but opened temporary WS connection)
		_, exists := Clients[identifier]
		if !exists { // First time connection from a client means its main WS connection, dont replace that
			tempConnection = false // First time connection from a client means its main WS connection
			Clients[identifier] = &models.Client{Conn: ws}
			// Set client's name if first message
			if Clients[identifier].Name == "" {
				log.Println("[WS] Client sent identifier: ", msg.From)

				Clients[identifier].Name = msg.From
			}
		}

		clientsMu.Unlock()

		// Process stop
		if msg.Stop {
			loopsMu.Lock()
			if stop, ok := Loops[identifier]; ok {
				stop <- true
				delete(Loops, identifier)
			}
			loopsMu.Unlock()

		} else if msg.URL != "" && msg.Download == true { // If theres a URL and download is true, only download
			log.Println("[WS] Download received from " + msg.From)

			data, err := fs.DownloadYoutubeURLToFile(msg.URL, "audio_temp")
			if err != nil {
				log.Fatal("[WS] Error while downloading song and info:", err)
			}
			msg := &models.SongInfo{
				FilePath: data.FilePath,
				Title:    data.Title,
				Uploader: data.Uploader,
				ID:       data.ID,
				Duration: data.Duration,
			}
			jsonData, err := json.Marshal(msg)
			err = websocket.Message.Send(ws, jsonData)
			if err != nil {
				log.Fatal("[WS] Error while sending song info:", err)
			}
		} else if msg.URL != "" && msg.Download == false { // If theres a URL but download is false, that means download&play
			log.Println("[WS] Play received from " + msg.From)

			data, err := fs.DownloadYoutubeURLToFile(msg.URL, "audio_temp")
			if err != nil {
				log.Fatal("[WS] Error while downloading song and info:", err)
			}
			msgData := &models.SongInfo{
				FilePath: data.FilePath,
				Title:    data.Title,
				Uploader: data.Uploader,
				ID:       data.ID,
				Duration: data.Duration,
			}
			jsonData, err := json.Marshal(msgData)
			err = websocket.Message.Send(ws, jsonData)
			if err != nil {
				log.Fatal("[WS] Error while sending song info:", err)
			}
			loopsMu.Lock()
			seeksMu.Lock()

			seekChannel, exists := Seeks[identifier]
			if !exists {
				seekChannel = make(chan int)
				Seeks[identifier] = seekChannel
			}

			stopChannel, exists := Loops[identifier]
			if !exists {
				stopChannel = make(chan bool, 1)
				Loops[identifier] = stopChannel
			}
			go sendByteData(Clients[identifier].Conn, msgData, stopChannel, seekChannel)
			Loops[identifier] = stopChannel
			Seeks[identifier] = seekChannel

			loopsMu.Unlock()
			seeksMu.Unlock()

		} else if msg.Seek >= 0 {
			log.Println("[WS] Seek received from " + msg.From)
			seeksMu.Lock()
			if seek, ok := Seeks[identifier]; ok {
				seek <- msg.Seek
				log.Println("[WS] Seek received from " + msg.From + " sent to channel")

			}
			seeksMu.Unlock()
		}

	}

	// Remove client on disconnect if not temp connection
	if !tempConnection {
		clientsMu.Lock()
		delete(Clients, identifier)
		clientsMu.Unlock()
		loopsMu.Lock()
		if stop, ok := Loops[identifier]; ok {
			stop <- true
			close(stop)
			delete(Loops, identifier)
		}
		loopsMu.Unlock()
		seeksMu.Lock()
		if seek, ok := Seeks[identifier]; ok {
			close(seek)
			delete(Seeks, identifier)
		}
		seeksMu.Unlock()
	}

}
