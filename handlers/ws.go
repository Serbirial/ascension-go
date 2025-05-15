package handlers

import (
	"encoding/binary"
	"encoding/json"
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
				OnError("[WS-BYTE-RECV]", "Receive error:", err)
				return
			}
			output <- data
		}
	}
}
func sendByteData(ws *websocket.Conn, song *models.SongInfo, stop <-chan bool, seek <-chan int) {
	log.Println("[WS] Streaming started")
	file, err := os.Open(song.FilePath)
	if err != nil {
		log.Println("Error opening dca file :", err)
		return
	}
	defer file.Close()

	// Constants and state
	const frameDuration = 20 * time.Millisecond
	const frameRateDCA = int(time.Second / frameDuration) // 50 fps

	var (
		opuslen      int16
		currentFrame int
		smu          sync.Mutex
		frameBufPool = sync.Pool{
			New: func() any { return make([]byte, 2048) }, // max expected opus frame
		}
	)

	// Build frame index
	frameIndex, err := buildFrameIndex(file)
	if err != nil {
		log.Println("[WS] Error building frame index:", err)
		return
	}
	if len(frameIndex) == 0 {
		log.Println("[WS] Frame index is empty, aborting playback")
		return
	}

	for {
		select {
		case <-stop:
			log.Println("[WS] Streaming stopped")
			return

		case seconds := <-seek: // Seek through file if the seek signal has been sent
			smu.Lock()
			log.Println("[WS] Seeking.")

			frameDelta := int(seconds * frameRateDCA)
			targetFrame := currentFrame + frameDelta
			if targetFrame < 0 {
				targetFrame = 0
			}
			if targetFrame >= len(frameIndex) {
				targetFrame = len(frameIndex) - 1
			}

			_, err := file.Seek(frameIndex[targetFrame], io.SeekStart)
			if err != nil {
				log.Println("[WS] Seek error:", err)
				smu.Unlock()
				return
			}
			currentFrame = targetFrame
			smu.Unlock()

		case <-time.After(5 * time.Millisecond): // Send the next frame
			smu.Lock()
			err := binary.Read(file, binary.LittleEndian, &opuslen)
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				smu.Unlock()
				_ = websocket.Message.Send(ws, []byte("DONE")) // send DONE to client

				return // End of file
			}
			if err != nil {
				log.Println("[WS] Error reading frame length:", err)
				smu.Unlock()
				return
			}

			buf := frameBufPool.Get().([]byte)
			if int(opuslen) > cap(buf) {
				buf = make([]byte, opuslen)
			}
			frame := buf[:opuslen]

			err = binary.Read(file, binary.LittleEndian, &frame)
			if err != nil {
				log.Println("[WS] Error reading frame data:", err)
				smu.Unlock()
				return
			}

			err = websocket.Message.Send(ws, frame)
			if err != nil {
				log.Println("[WS] Error sending data:", err)
				smu.Unlock()
				return
			}
			currentFrame++
			frameBufPool.Put(buf)
			smu.Unlock()
		}
	}
}

func HandleWebSocket(ws *websocket.Conn) {
	defer ws.Close()

	log.Println("[WS] Connected: ", ws.RemoteAddr())
	var tempConnection bool = true // Assume temp connection
	var reference string = ""

	for {
		var msg models.Message
		// Read JSON message
		if err := websocket.JSON.Receive(ws, &msg); err != nil {
			log.Println("[WS] Disconnected: ", ws.RemoteAddr(), "-", err)
			break
		}

		// Set the reference
		reference = msg.From

		// Register new clients after they send identifier (first recv)
		clientsMu.Lock()

		// Client already might exist (ex: is streaming from the server, but opened temporary WS connection)
		_, exists := Clients[msg.From]
		if !exists { // First time connection from a client means its main WS connection, dont replace that
			tempConnection = false // First time connection from a client means its main WS connection
			Clients[msg.From] = &models.Client{Conn: ws}
			// Set client's name if first message
			if Clients[msg.From].Name == "" {
				log.Println("[WS] Client sent identifier: ", msg.From)

				Clients[msg.From].Name = msg.From
			}
		}

		clientsMu.Unlock()

		// Process stop
		if msg.Stop {
			loopsMu.Lock()
			if stop, ok := Loops[msg.From]; ok {
				stop <- true
				delete(Loops, msg.From)
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

			stopChannel := make(chan bool, 1)
			seekChannel := make(chan int, 1)

			go sendByteData(Clients[msg.From].Conn, msgData, stopChannel, seekChannel)
			Loops[msg.From] = stopChannel
			Seeks[msg.From] = seekChannel

			loopsMu.Unlock()
			seeksMu.Unlock()

		} else if msg.Seek != 0 {
			log.Println("[WS] Seek received from " + msg.From)
			seeksMu.Lock()
			if seek, ok := Seeks[msg.From]; ok {
				seek <- msg.Seek
			}
			seeksMu.Unlock()
			log.Println("[WS] Seek received from " + msg.From + " sent to channel")

		}

	}

	// Remove client on disconnect if not temp connection
	if !tempConnection {
		clientsMu.Lock()
		delete(Clients, reference)
		clientsMu.Unlock()
		loopsMu.Lock()
		if stop, ok := Loops[reference]; ok {
			stop <- true
			close(stop)
			delete(Loops, reference)
		}
		loopsMu.Unlock()
		seeksMu.Lock()
		if seek, ok := Seeks[reference]; ok {
			close(seek)
			delete(Seeks, reference)
		}
		seeksMu.Unlock()
	}

}
