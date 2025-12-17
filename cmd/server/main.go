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

	_ "github.com/lib/pq"
)

func main() {
	envConfig := config.Load()

	pool, err := db.Connect(envConfig.DBURL)
	if err != nil {
		log.Fatal("Failed to init DB:", err)
	}
	defer pool.Close()

	chConn, err := db.ConnectClickHouse(envConfig.CLICKHOUSE_URL)
	if err != nil {
		log.Fatal("Failed to init ClickHouse:", err)
	}
	defer chConn.Close()

	router := server.NewRouter(pool, chConn)

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
	log.Println("server stopped")

}
