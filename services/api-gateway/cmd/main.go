package main

import (
	"log"

	"ms-ride-sharing/services/api-gateway/internal/config"
)

func main() {
	log.Println("Starting API Gateway")

	config.ConfigServer()
}
