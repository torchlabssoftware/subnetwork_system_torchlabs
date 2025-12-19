package server

import (
	"encoding/json"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var (
	pongWait     = 10 * time.Second
	pingIntervel = (pongWait * 9) / 10
)

type Worker struct {
	Connection *websocket.Conn
	Manager    *WebsocketManager
	egress     chan Event
	ID         uuid.UUID
	Name       string
}

type WorkerList map[*Worker]bool

func NewWorker(conn *websocket.Conn, manager *WebsocketManager) *Worker {
	return &Worker{
		Connection: conn,
		Manager:    manager,
		egress:     make(chan Event),
	}
}

func (w *Worker) ReadMessage() {
	defer func() {
		w.Manager.RemoveWorker(w)
		w.Connection.Close()
	}()

	if err := w.Connection.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
		log.Println(err)
		return
	}

	w.Connection.SetReadLimit(4096)

	w.Connection.SetPongHandler(w.PongHandler)

	for {
		_, payload, err := w.Connection.ReadMessage()
		if err != nil {
			log.Println("message read error.Connetion closed:", err)
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("Error reading message: %v", err)
			}
			break
		}

		var request Event

		if err := json.Unmarshal(payload, &request); err != nil {
			log.Printf("errror marshelling event: %v", err)
			break
		}

		if err := w.Manager.RouteEvent(request, w); err != nil {
			log.Println("Error handleing message: ", err)
			w.egress <- Event{
				Type:    "error",
				Payload: errorPayload{Success: false, Payload: err.Error()},
			}
		}
	}
}

func (w *Worker) WriteMessage() {
	defer func() {
		close(w.egress)
		w.Connection.Close()
		w.Manager.RemoveWorker(w)
	}()

	ticker := time.NewTicker(pingIntervel)

	for {
		select {
		case message, ok := <-w.egress:
			if !ok {
				if err := w.Connection.WriteMessage(websocket.CloseMessage, nil); err != nil {
					log.Println("Connetion closed:", err)
				}
				log.Println("message write error.Connetion closed:")
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
			log.Println("Sent message")
		case <-ticker.C:
			log.Println("ping")
			if err := w.Connection.WriteMessage(websocket.PingMessage, []byte(``)); err != nil {
				log.Println("write message error")
				return
			}
		}
	}
}

func (w *Worker) PongHandler(pongMsg string) error {
	log.Println("pong")
	return w.Connection.SetReadDeadline(time.Now().Add(pongWait))
}
