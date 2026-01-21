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
	models "github.com/torchlabssoftware/subnetwork_system/internal/server/models"
	service "github.com/torchlabssoftware/subnetwork_system/internal/server/service"
)

func NewRouter(pool *sql.DB, clickHouseConn driver.Conn, websocketManager models.WebsocketManagerInterface) http.Handler {
	router := chi.NewRouter()

	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(middleware.Recoverer)
	router.Use(middleware.Logger)
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

	analyticsService := service.NewAnalyticsService(clickHouseConn)
	analyticsService.StartWorkers()
	a := handlers.NewAnalyticsHandler(analyticsService)

	websocketManager.SetAnalyticsandQueries(q, analyticsService)

	u := handlers.NewUserHandler(service.NewUserService(q, pool, websocketManager))

	p := handlers.NewPoolHandler(service.NewPoolService(q, pool, websocketManager))

	w := handlers.NewWorkerHandler(service.NewWorkerService(q, pool, websocketManager))

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
