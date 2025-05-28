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
	DownloaderIsDetached bool = false

	Clients = make(map[string]*models.Client)
	Loops   = make(map[string]chan bool)
	Seeks   = make(map[string]chan int)
	IsDone  = make(map[string]chan bool)

	clientsMu sync.Mutex
)

// Receives byte data from ws connection and puts it into the `output` channel, will stop when `true` is sent through the `stop` channel
func RecvByteData(ws *websocket.Conn, output chan []byte, stop <-chan bool) {
	defer func() {
		if r := recover(); r != nil {
			return
		}
		log.Println("[WS-BYTE-RECV] Receiver exiting")
	}()

	for {
		select {
		case <-stop:
			log.Println("[WS-BYTE-RECV] Stop signal received")
			return
		case <-time.After(15 * time.Millisecond):
			var data []byte
			err := websocket.Message.Receive(ws, &data)
			if err != nil {
				// Connection is likely closed
				log.Println("[WS-BYTE-RECV] Receive error:", err)
				return
			}
			output <- data
		}
	}
}

// Sends byte data to ws connection, undocumented.
func sendByteData(identifier string, ws *websocket.Conn, song *models.SongInfo, stop <-chan bool, seek <-chan int, startFrame int, done <-chan bool) {
	ticker := time.NewTicker(15 * time.Millisecond) // 5ms under discords needed send timing to allow for buffering 4 frames at a time
	defer ticker.Stop()
	log.Println("[WS] Streaming connection started")

	var file *os.File
	var err error
	if DownloaderIsDetached { // The file will be mounted in a different directory over WLAN
		file, err = os.Open("mounted/" + song.FilePath) // Should just be mounted at `mounted/`, should end up being `mounted/audio_temp/videoID`
		if err != nil {
			log.Println("Error opening mounted dca file:", err)
			return
		}
	} else {
		file, err = os.Open(song.FilePath)
		if err != nil {
			log.Println("Error opening dca file:", err)
			return
		}
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

	// Start playback from startFrame
	if err := seekToFrame(startFrame); err != nil {
		log.Println("[WS] Initial seek error:", err)
		return
	}

	var pendingSeek *int
	doneFlag := false
	var pendingDone *bool = &doneFlag

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
			// Unset pending done to resume playing
			*pendingDone = false
			pendingSeek = &seconds
		case <-ticker.C:
			if *pendingDone {
				select {
				case <-stop:
				case <-time.After(4 * time.Millisecond):
					continue // keep looping incase seek gets sent

				}
			}
			// If there's a pending seek, perform it now
			if pendingSeek != nil {
				targetFrame := *pendingSeek * frameRateDCA
				log.Printf("[WS] Seeking: currentFrame=%d, requestedSeconds=%d, targetFrame=%d", currentFrame, *pendingSeek, targetFrame)
				if err := seekToFrame(targetFrame); err != nil {
					log.Println("[WS] Seek error:", err)
				}
				pendingSeek = nil
				log.Printf("[WS] Restarting stream at targetFrame")
				sendByteData(identifier, ws, song, stop, seek, targetFrame, done)
				return // Exit
			}
			if currentFrame >= len(frameIndex) {
				log.Println("[WS] Reached end of stream")
				_ = websocket.Message.Send(ws, []byte("DONE")) // Send DONE because they seeked to the end, songs over
				return
			}

			// Read Opus frame length
			var opuslen int16
			if err := binary.Read(file, binary.LittleEndian, &opuslen); err != nil {
				if err == io.EOF || err == io.ErrUnexpectedEOF {
					log.Println("[WS] EOF reached")
					_ = websocket.Message.Send(ws, []byte("DONESTREAM"))
					log.Println("[WS] Waiting for BOT to send DONE back")
					*pendingDone = true
					IsDone[identifier] <- true // Send done to main handler

				} else {
					log.Println("[WS] Error reading frame length:", err)
					return

				}
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

// Handles a websocket connection
func HandleWebSocket(ws *websocket.Conn) {
	log.Println("[WS] Connected: ", ws.RemoteAddr())
	var tempConnection bool = true // Assume temp connection
	var name string = ""
	var identifier string = ""

	for {
		var jsonDataRecv []byte
		if err := websocket.Message.Receive(ws, &jsonDataRecv); err != nil {
			if tempConnection {
				log.Println("[WS] Communication connection Closed:", name, identifier, "-", err)
				break
			} else {
				log.Println("[WS] Streaming connection Closed:", name, identifier, "-", err)
				break

			}
		}
		msgStr := string(jsonDataRecv)

		if msgStr == "DONE" { // Client sent DONE
			log.Println("[WS] Client sent DONE, ensuring Streamer has sent done")
			<-IsDone[identifier] // Block until WS done
			log.Println("[WS] Streamer DONE, sending back DONE")

			_ = websocket.Message.Send(ws, []byte("DONE")) // Send DONE so the bot knows everything is OK and DONE

		} else {
			var msg models.Message
			if err := json.Unmarshal(jsonDataRecv, &msg); err != nil {
				log.Fatalf("Failed to decode JSON: %v", err) // FIXME break from loop
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
				if Clients[identifier].Name == "" && msg.From != "" {
					log.Println("[WS] Client sent identifier: ", msg.From)
					Clients[identifier].Name = msg.From
				}
			}

			clientsMu.Unlock()

			// Process stop FIXME: stopping while queued will make the bot think its playing the next song but the server keeps playing the old one
			if msg.Stop {
				clientsMu.Lock()
				if stop, ok := Loops[identifier]; ok {
					stop <- true
					delete(Loops, identifier)
				}
				clientsMu.Unlock()

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
				// FIXME make only play- not download- not possible currently unless i write another function in fs
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
				clientsMu.Lock()

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

				doneChannel, exists := IsDone[identifier]
				if !exists {
					doneChannel = make(chan bool, 1)
					IsDone[identifier] = doneChannel
				}
				go sendByteData(identifier, Clients[identifier].Conn, msgData, stopChannel, seekChannel, 0, doneChannel)

				clientsMu.Unlock()

			} else if msg.Seek >= 0 {
				log.Println("[WS] Seek received from " + msg.From)
				clientsMu.Lock()
				if seek, ok := Seeks[identifier]; ok {
					seek <- msg.Seek
					log.Println("[WS] Seek received from " + msg.From + " sent to channel")

				}
				clientsMu.Unlock()
			}
		}

	}

	// Remove client on disconnect if not temp connection
	if !tempConnection {
		clientsMu.Lock()
		delete(Clients, identifier)
		if stop, ok := Loops[identifier]; ok {
			stop <- true
			close(stop)
			delete(Loops, identifier)
		}
		if seek, ok := Seeks[identifier]; ok {
			close(seek)
			delete(Seeks, identifier)
		}
		clientsMu.Unlock()
	}

	// FIXME closes early during bot sending back DONE
	defer ws.Close() // should be fixed with manual closing

}
