package server

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/cors"
	"github.com/torchlabssoftware/subnetwork_system/internal/db/repository"
	handlers "github.com/torchlabssoftware/subnetwork_system/internal/server/handlers"
	service "github.com/torchlabssoftware/subnetwork_system/internal/server/service"
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
	u := handlers.NewUserHandler(q, pool, service.NewUserService(q, pool))
	w := handlers.NewWorkerHandler(q, pool)
	p := handlers.NewPoolHandler(q, pool)

	router.Route("/admin", func(r chi.Router) {
		r.Mount("/users", u.AdminRoutes())
		r.Mount("/pools", p.AdminRoutes())
		r.Mount("/worker", w.AdminRoutes())
	})

	router.Route("/worker", func(r chi.Router) {
		r.Mount("/", w.WorkerRoutes())
	})

	return router

}
