package manager

import (
	"encoding/json"
	"log"
	"sync"

	"github.com/gorilla/websocket"
)

type WebsocketClient struct {
	Connection *websocket.Conn
	egress     chan Event
	onEvent    func(Event)
}

func NewWebsocketClient(conn *websocket.Conn, onEvent func(Event)) *WebsocketClient {
	return &WebsocketClient{
		Connection: conn,
		egress:     make(chan Event),
		onEvent:    onEvent,
	}
}

func (w *WebsocketClient) ReadMessage(wg *sync.WaitGroup) {
	defer func() {
		close(w.egress)
		w.Connection.Close()
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
		w.Connection.Close()
		wg.Done()
	}()

	for message := range w.egress {

		if message.Type == "close" {
			if err := w.Connection.WriteMessage(websocket.CloseMessage, nil); err != nil {
				log.Println("message write error.Connetion closed:", err)
			}
			log.Println("Connetion closed:")
			return
		}

		data, err := json.Marshal(message)
		if err != nil {
			log.Println(err)
			return
		}

		if err := w.Connection.WriteMessage(websocket.TextMessage, data); err != nil {
			log.Printf("faild to send message: %v", err)
		}
		log.Println("Sent message: ", message.Type)

	}
}
