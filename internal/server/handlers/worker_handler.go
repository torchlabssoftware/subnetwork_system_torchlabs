package server

import (
	"database/sql"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/torchlabssoftware/subnetwork_system/internal/db/repository"
	middleware "github.com/torchlabssoftware/subnetwork_system/internal/server/middleware"
	wsm "github.com/torchlabssoftware/subnetwork_system/internal/server/websocket"
)

type WorkerHandler struct {
	queries   *repository.Queries
	db        *sql.DB
	wsManager *wsm.WebsocketManager
}

func NewWorkerHandler(q *repository.Queries, db *sql.DB) *WorkerHandler {
	w := &WorkerHandler{
		queries: q,
		db:      db,
	}
	w.wsManager = wsm.NewWebsocketManager(q)
	return w
}

func (ws *WorkerHandler) Routes() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.WorkerAuthentication)

	r.Get("/ws", ws.serveWS)
	return r
}

func (ws *WorkerHandler) serveWS(w http.ResponseWriter, r *http.Request) {
	ws.wsManager.ServeWS(w, r)
}
