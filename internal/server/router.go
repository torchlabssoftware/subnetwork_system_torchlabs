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

	router.Route("/admin", func(r chi.Router) {
		q := repository.New(pool)

		h := server.NewUserHandler(q)
		r.Mount("/user", h.Routes())
	})

	return router

}
