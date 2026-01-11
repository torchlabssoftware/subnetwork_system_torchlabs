package manager

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait     = 10 * time.Second
	egressBufSize = 100
)

type WebsocketClient struct {
	Connection *websocket.Conn
	egress     chan Event
	onEvent    func(Event)
	done       chan struct{}
	closeOnce  sync.Once
}

func NewWebsocketClient(conn *websocket.Conn, onEvent func(Event)) *WebsocketClient {
	return &WebsocketClient{
		Connection: conn,
		egress:     make(chan Event, egressBufSize),
		onEvent:    onEvent,
		done:       make(chan struct{}),
	}
}

// Close gracefully closes the WebSocket client
func (w *WebsocketClient) Close() {
	w.closeOnce.Do(func() {
		close(w.done)
		w.Connection.Close()
	})
}

func (w *WebsocketClient) ReadMessage(wg *sync.WaitGroup) {
	defer func() {
		w.Close()
		wg.Done()
	}()

	for {
		_, payload, err := w.Connection.ReadMessage()
		if err != nil {
			log.Println("message read error.Connetion closed:", err)
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("Error reading message: %v", err)
			}
			break
		}

		var event Event

		if err := json.Unmarshal(payload, &event); err != nil {
			log.Printf("errror marshelling event: %v", err)
			break
		}

		w.onEvent(event)
	}
}

func (w *WebsocketClient) WriteMessage(wg *sync.WaitGroup) {
	defer func() {
		w.Close()
		wg.Done()
	}()

	for {
		select {
		case <-w.done:
			return
		case message, ok := <-w.egress:
			if !ok {
				return
			}

			if message.Type == "close" {
				w.Connection.SetWriteDeadline(time.Now().Add(writeWait))
				if err := w.Connection.WriteMessage(websocket.CloseMessage, nil); err != nil {
					log.Println("[WebSocket] Close message write error:", err)
				}
				log.Println("[WebSocket] Connection closed")
				return
			}

			data, err := json.Marshal(message)
			if err != nil {
				log.Printf("[WebSocket] Failed to marshal message: %v", err)
				continue
			}

			w.Connection.SetWriteDeadline(time.Now().Add(writeWait))
			if err := w.Connection.WriteMessage(websocket.TextMessage, data); err != nil {
				log.Printf("[WebSocket] Failed to send message: %v", err)
				return
			}
			log.Println("[WebSocket] Sent message:", message.Type)
		}
	}
}
