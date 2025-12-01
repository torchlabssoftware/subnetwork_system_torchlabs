package main

import (
	"log"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/go-chi/cors"
	"github.com/torchlabssoftware/subnetwork_system/internal/config"
)

func main() {
	dotenvConfig := config.Load()
	log.Println("port:", dotenvConfig.PORT)

	router := chi.NewRouter()

	router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"https://*", "http://*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	srver := &http.Server{
		Handler: router,
		Addr:    ":" + dotenvConfig.PORT,
	}

	log.Println("Server start on port", dotenvConfig.PORT)
	err := srver.ListenAndServe()
	if err != nil {
		log.Fatal("cannot start http server:", err)
	}
}
