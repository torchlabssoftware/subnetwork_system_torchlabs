package main

import (
	"log"

	"github.com/torchlabssoftware/subnetwork_system/config"
)

func main() {
	dotenvConfig := config.Load()
	log.Println("port:", dotenvConfig.PORT)
}
