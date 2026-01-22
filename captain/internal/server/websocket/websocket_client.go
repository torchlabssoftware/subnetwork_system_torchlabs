package server

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var (
	pongWait     = 10 * time.Second
	pingInterval = (pongWait * 9) / 10
)

type Worker struct {
	Connection *websocket.Conn
	Manager    *WebsocketManager
	egress     chan Event
	ID         uuid.UUID
	Name       string
	PoolId     uuid.UUID
}

type WorkerList map[uuid.UUID]*Worker

func NewWorker(conn *websocket.Conn, manager *WebsocketManager) *Worker {
	return &Worker{
		Connection: conn,
		Manager:    manager,
		egress:     make(chan Event, 100),
	}
}

func (w *Worker) ReadMessage() {
	defer func() {
		w.Manager.RemoveWorker(w)
		w.Connection.Close()
		close(w.egress)
	}()
	if err := w.Connection.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
		log.Println("[websocket] failed to set read deadline:", err)
		return
	}
	w.Connection.SetPongHandler(w.PongHandler)
	for {
		_, payload, err := w.Connection.ReadMessage()
		if err != nil {
			log.Println("[websocket] message read error.Connection closed:", err)
			return
		}
		var request Event
		if err := json.Unmarshal(payload, &request); err != nil {
			log.Println("[websocket] error marshaling event:", err)
			return
		}
		log.Println("[websocket] Received message", request.Type)
		if err := w.Manager.RouteEvent(request, w); err != nil {
			log.Println("[websocket] Error handling message: ", err)
			w.egress <- Event{
				Type:    "error",
				Payload: replyPayload{Success: false, Payload: err.Error()},
			}
		}
	}
}

func (w *Worker) WriteMessage() {
	ticker := time.NewTicker(pingInterval)
	defer func() {
		ticker.Stop()
		w.Connection.Close()
		w.Manager.RemoveWorker(w)
	}()
	for {
		select {
		case message, ok := <-w.egress:
			if !ok {
				if err := w.Connection.WriteMessage(websocket.CloseMessage, nil); err != nil {
					log.Println("[websocket] failed to send message:", err)
				}
				log.Println("[websocket] message write error.Connection closed:")
				return
			}
			data, err := json.Marshal(message)
			if err != nil {
				log.Println("[websocket] error marshaling event:", err)
				return
			}
			if err := w.Connection.SetWriteDeadline(time.Now().Add(10 * time.Second)); err != nil {
				log.Println("[websocket] failed to set write deadline:", err)
				return
			}
			if err := w.Connection.WriteMessage(websocket.TextMessage, data); err != nil {
				log.Println("[websocket] failed to send message:", err)
				return
			}
			log.Println("[websocket] Sent message", message.Type)
		case <-ticker.C:
			log.Println("[websocket] ping to", w.Name)
			if err := w.Connection.SetWriteDeadline(time.Now().Add(10 * time.Second)); err != nil {
				log.Println("[websocket] failed to set write deadline:", err)
				return
			}
			if err := w.Connection.WriteMessage(websocket.PingMessage, []byte(``)); err != nil {
				log.Println("[websocket] ping message error")
				return
			}
		}
	}
}

func (w *Worker) PongHandler(pongMsg string) error {
	log.Println("[websocket] pong from", w.Name)
	if err := w.Manager.queries.UpdateWorkerLastSeen(context.Background(), w.ID); err != nil {
		log.Printf("[websocket] failed to update last seen for worker %s: %v", w.Name, err)
	}
	return w.Connection.SetReadDeadline(time.Now().Add(pongWait))
}
