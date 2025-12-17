package server

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/cors"
	"github.com/torchlabssoftware/subnetwork_system/internal/db/repository"
	handlers "github.com/torchlabssoftware/subnetwork_system/internal/server/handlers"
	service "github.com/torchlabssoftware/subnetwork_system/internal/server/service"
)

func NewRouter(pool *sql.DB, clickHouseConn driver.Conn) http.Handler {
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
	analyticsService := service.NewAnalyticsService(clickHouseConn)
	w := handlers.NewWorkerHandler(q, pool, analyticsService)
	p := handlers.NewPoolHandler(q, pool)

	a := handlers.NewAnalyticsHandler(analyticsService)

	router.Route("/admin", func(r chi.Router) {
		r.Mount("/users", u.AdminRoutes())
		r.Mount("/pools", p.AdminRoutes())
		r.Mount("/worker", w.AdminRoutes())
		r.Route("/analytics", func(r chi.Router) {
			a.RegisterRoutes(r)
		})
	})

	router.Route("/worker", func(r chi.Router) {
		r.Mount("/", w.WorkerRoutes())
	})

	return router

}
