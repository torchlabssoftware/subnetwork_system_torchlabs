package main

import (
	"log"
	"net/http"

	"github.com/torchlabssoftware/subnetwork_system/internal/config"
)

func main() {
	dotenvConfig := config.Load()
	log.Println("port:", dotenvConfig.PORT)

	http.ListenAndServe(":8080", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("rrr"))
	}))
}
