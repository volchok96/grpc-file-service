package main

import (
	"log"
	"os"

	"github.com/volchok96/grpc-file-service/internal/config"
	"github.com/volchok96/grpc-file-service/internal/server"
)

func main() {
	cfg := config.NewDefaultConfig()

	if err := os.MkdirAll(cfg.StoragePath, 0755); err != nil {
		log.Fatalf("Failed to create storage directory: %v", err)
	}

	srv := server.NewServer(cfg)

	if err := srv.Run(); err != nil {
		log.Fatalf("Failed to run server: %v", err)
	}
}
