package server

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/cors"
	"github.com/torchlabssoftware/subnetwork_system/internal/db/repository"
	server "github.com/torchlabssoftware/subnetwork_system/internal/server/handlers"
)

func NewRouter(pool *sql.DB) http.Handler {
	router := chi.NewRouter()

	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(middleware.Recoverer)
	router.Use(middleware.Timeout(30 * time.Second))

	router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"https://*", "http://*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	q := repository.New(pool)

	router.Route("/api", func(r chi.Router) {
		h := server.NewUserHandler(q, pool)
		r.Mount("/users", h.Routes())

		p := server.NewPoolHandler(q, pool)
		r.Mount("/pools", p.Routes())

		w := server.NewWorkerHandler(q, pool)
		r.Mount("/worker", w.Routes())
	})

	return router

}
