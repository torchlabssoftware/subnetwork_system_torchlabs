package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/cors"
	"github.com/google/uuid"
	"github.com/torchlabssoftware/subnetwork_system/internal/config"
	"github.com/torchlabssoftware/subnetwork_system/internal/repository"

	_ "github.com/lib/pq"
)

func main() {
	dotenvConfig := config.Load()

	router := chi.NewRouter()

	DBURL := dotenvConfig.DBURL

	connection, err := sql.Open("postgres", DBURL)
	if err != nil {
		log.Fatalf("cant connect to the database: %v", err)
	}

	queries := repository.New(connection)
	_, err = queries.CreateUser(context.Background(), repository.CreateUserParams{
		ID:        uuid.New(),
		Email:     sql.NullString{String: "pubkkudu.ppp.lk", Valid: true},
		Username:  "pppk",
		Password:  "ppp",
		DataLimit: sql.NullInt64{Int64: 0, Valid: true},
		DataUsage: sql.NullInt64{Int64: 0, Valid: true},
		Status:    sql.NullString{String: "active", Valid: true},
		CreatedAt: sql.NullTime{Time: time.Now(), Valid: true},
		UpdatedAt: sql.NullTime{Time: time.Now(), Valid: true},
	})

	if err != nil {
		log.Printf("", err)
	}

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
	err = srver.ListenAndServe()
	if err != nil {
		log.Fatal("cannot start http server:", err)
	}
}
