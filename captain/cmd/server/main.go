package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/torchlabssoftware/subnetwork_system/internal/config"
	db "github.com/torchlabssoftware/subnetwork_system/internal/db"
	"github.com/torchlabssoftware/subnetwork_system/internal/server"
	wsm "github.com/torchlabssoftware/subnetwork_system/internal/server/websocket"
)

func main() {
	//select workspace and load environment variables
	appEnv := os.Getenv("APP_ENV")
	log.Println("APP_ENV:", appEnv)
	if strings.ToLower(appEnv) == "dev" || strings.ToLower(appEnv) == "" {
		if err := godotenv.Load(".env.dev"); err != nil {
			log.Println("Error in loading .env.dev file:", err)
		}
	} else {
		log.Println("Running in production mode, using environment variables from Docker")
	}
	envConfig := config.Load()

	//connect to postgres
	pgConn, err := db.ConnectPostgres(envConfig.POSTGRES_URL)
	if err != nil {
		log.Fatal("Failed to init PostgresDB:", err)
	}
	defer pgConn.Close()

	//connect to clickhouse
	chConn, err := db.ConnectClickHouse(envConfig.CLICKHOUSE_URL, envConfig.CLICKHOUSE_DB, envConfig.CLICKHOUSE_USER, envConfig.CLICKHOUSE_PASSWORD)
	if err != nil {
		log.Fatal("Failed to init ClickHouse:", err)
	}
	defer chConn.Close()

	//create websocket manager
	websocketManager := wsm.NewWebsocketManager()

	//create router
	router := server.NewRouter(pgConn, chConn, websocketManager)

	//create server
	srv := &http.Server{
		Addr:         ":" + envConfig.PORT,
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 20 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	//start server
	go func() {
		log.Println("server running in port:", envConfig.PORT)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("server failed")
		}
	}()

	//graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	<-stop

	log.Println("shutdown initiated")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("server shutdown error: %v", err)
	}
	log.Println("closing database connections...")
	if pgConn != nil {
		if err := pgConn.Close(); err != nil {
			log.Printf("error closing Postgres: %v", err)
		}
	}
	if chConn != nil {
		if err := chConn.Close(); err != nil {
			log.Printf("error closing ClickHouse: %v", err)
		}
	}
	websocketManager.Shutdown()
	log.Println("server stopped")
}
