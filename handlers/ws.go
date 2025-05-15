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
	Clients = make(map[*websocket.Conn]*models.Client)
	Loops   = make(map[*websocket.Conn]chan bool)
	Seeks   = make(map[*websocket.Conn]chan int)

	clientsMu sync.Mutex
	loopsMu   sync.Mutex
	seeksMu   sync.Mutex
)

func RecvByteData(ws *websocket.Conn, output chan []byte, stop <-chan bool) {
	for {
		var data []byte
		err := websocket.Message.Receive(ws, &data)
		if err != nil {
			OnError("[WS-BYTE-RECV]", "Receive error:", err)
		}
		output <- data
	}
}

func sendByteData(ws *websocket.Conn, song *models.SongInfo, stop <-chan bool, seek <-chan int) {

	file, err := os.Open(song.FilePath)
	if err != nil {
		log.Println("Error opening dca file :", err)
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
			log.Println("[WS] Stop recognized")
			return

		case seconds := <-seek: // Seek through file if the seek signal has been sent
			smu.Lock()
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

		case <-time.After(10 * time.Millisecond): // Send the next frame after 10ms
			smu.Lock()
			err := binary.Read(file, binary.LittleEndian, &opuslen)
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				smu.Unlock()
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

	// Register new client
	clientsMu.Lock()
	Clients[ws] = &models.Client{Conn: ws}
	clientsMu.Unlock()

	log.Println("[WS] Connected: ", ws.RemoteAddr())

	for {
		var msg models.Message

		// Read JSON message
		if err := websocket.JSON.Receive(ws, &msg); err != nil {
			log.Println("[WS] Disconnected: ", ws.RemoteAddr(), "-", err)
			break
		}

		// Set client's name if first message
		clientsMu.Lock()
		if Clients[ws].Name == "" {
			log.Println("[WS] SetName: ", msg.From)

			Clients[ws].Name = msg.From
		}
		clientsMu.Unlock()

		// Process stop
		if msg.Stop {
			loopsMu.Lock()
			if stop, ok := Loops[ws]; ok {
				stop <- true
				delete(Loops, ws)
			}
			loopsMu.Unlock()

		} else if msg.URL != "" {
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
			loopsMu.Lock()
			stopChannel := make(chan bool, 1)
			seekChannel := make(chan int, 1)

			go sendByteData(ws, msg, stopChannel, seekChannel)
			Loops[ws] = stopChannel
			loopsMu.Unlock()
		} else if msg.Seek != 0 {
			seeksMu.Lock()
			if seek, ok := Seeks[ws]; ok {
				seek <- msg.Seek
			}
			seeksMu.Unlock()

		}

	}

	// Remove client on disconnect
	clientsMu.Lock()
	delete(Clients, ws)
	clientsMu.Unlock()
	loopsMu.Lock()
	if stop, ok := Loops[ws]; ok {
		stop <- true
		close(stop)
		delete(Loops, ws)
	}
	loopsMu.Unlock()
	seeksMu.Lock()
	if seek, ok := Seeks[ws]; ok {
		close(seek)
		delete(Seeks, ws)
	}
	seeksMu.Unlock()
}
