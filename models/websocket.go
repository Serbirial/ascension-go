package models

import (
	"fmt"
	"log"
	"time"

	"golang.org/x/net/websocket"
)

// JSON message structure
type Message struct {
	From     string `json:"from"`
	URL      string `json:"content"`
	Download bool   `json:"download"`
	Stop     bool   `json:"stop"`
	Seek     int    `json:"seek"`
}

// Struct to hold a connected client
type Client struct {
	Name string
	Conn *websocket.Conn
}

func (client *Client) Send(data any) {

	if err := websocket.JSON.Send(client.Conn, data); err != nil {
		log.Println("[]Send error:", err)
	}
}

func (client *Client) StreamByteData(dataToSend chan []byte, stop <-chan bool) {
	for {
		select {
		case <-stop:
			log.Println("[WS-STREAM-BYTE] Stop recognized")
			return
		case <-time.After(5 * time.Millisecond):
		case data, ok := <-dataToSend:
			if !ok {
				OnError("[WS-STREAM-BYTE]", "Channel closed", nil)
				return
			}
			client.Send(data)
		}
	}
}

// OnError gets called by dgvoice when an error is encountered.
// By default logs to STDERR
var OnError = func(prefix string, str string, err error) {
	prefix = prefix + ": " + str

	if err != nil {
		fmt.Println(prefix + ": " + err.Error())
	} else {
		fmt.Println(prefix)
	}
}
