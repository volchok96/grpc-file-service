package config

import "time"

type Config struct {
	GRPCPort        string
	StoragePath     string
	UploadLimit     int
	DownloadLimit   int
	ListLimit       int
	MaxChunkSize    int
	ShutdownTimeout time.Duration
}

func NewDefaultConfig() *Config {
	return &Config{
		GRPCPort:        ":50051",
		StoragePath:     "files/files_server",
		UploadLimit:     10,
		DownloadLimit:   10,
		ListLimit:       100,
		MaxChunkSize:    64 * 1024, // 64KB
		ShutdownTimeout: 10 * time.Second,
	}
}
