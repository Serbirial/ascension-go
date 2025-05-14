package handlers

import (
	"encoding/binary"
	"encoding/json"
	"io"
	"log"
	"os"
	"sync"
	"time"

	"gobot/models"
	"gobot/utils/fs"

	"golang.org/x/net/websocket"
)

var (
	Clients = make(map[*websocket.Conn]*models.Client)
	Loops   = make(map[*websocket.Conn]chan bool)

	clientsMu sync.Mutex
	loopsMu   sync.Mutex
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

func sendByteData(ws *websocket.Conn, song *models.SongInfo, stop <-chan bool) {

	file, err := os.Open(song.FilePath)
	if err != nil {
		log.Println("Error opening dca file :", err)
	}
	defer file.Close()
	var opuslen int16

	for {
		select {
		case <-stop:
			log.Println("[WS] Stop recognized")
			return
		case <-time.After(10 * time.Millisecond):
			err := binary.Read(file, binary.LittleEndian, &opuslen)
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				return
			}
			if err != nil {
				log.Println("[WS] Error reading frame length:", err)
				return
			}

			data := make([]byte, opuslen)
			err = binary.Read(file, binary.LittleEndian, &data)
			if err != nil {
				log.Println("[WS] Error reading frame data:", err)
				return
			}

			err = websocket.Message.Send(ws, data)
			if err != nil {
				log.Println("[WS] Error sending data:", err)
				return
			}
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
			Clients[ws].Name = msg.From
		}
		clientsMu.Unlock()

		// Process stop
		if msg.Stop {
			loopsMu.Lock()
			channel := Loops[ws]
			channel <- true
			delete(Loops, ws)
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
			}
			jsonData, err := json.Marshal(msg)
			err = websocket.Message.Send(ws, jsonData)
			if err != nil {
				log.Fatal("[WS] Error while sending song info:", err)
			}
			loopsMu.Lock()
			stopChannel := make(chan bool, 1)
			go sendByteData(ws, msg, stopChannel)
			Loops[ws] = stopChannel
			loopsMu.Unlock()
		}

	}

	// Remove client on disconnect
	clientsMu.Lock()
	delete(Clients, ws)
	clientsMu.Unlock()
}
