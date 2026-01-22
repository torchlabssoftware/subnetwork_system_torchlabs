package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/torchlabssoftware/subnetwork_system/internal/config"
	db "github.com/torchlabssoftware/subnetwork_system/internal/db"
	"github.com/torchlabssoftware/subnetwork_system/internal/server"
	wsm "github.com/torchlabssoftware/subnetwork_system/internal/server/websocket"
)

func main() {
	envConfig := config.Load()

	pgConn, err := db.ConnectPostgres(envConfig.POSTGRES_URL)
	if err != nil {
		log.Fatal("Failed to init PostgresDB:", err)
	}
	defer pgConn.Close()

	chConn, err := db.ConnectClickHouse(envConfig.CLICKHOUSE_URL)
	if err != nil {
		log.Fatal("Failed to init ClickHouse:", err)
	}
	defer chConn.Close()

	websocketManager := wsm.NewWebsocketManager()
	router := server.NewRouter(pgConn, chConn, websocketManager)

	srv := &http.Server{
		Addr:         ":" + envConfig.PORT,
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 20 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		log.Println("server running in port:", envConfig.PORT)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("server failed")
		}
	}()

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
