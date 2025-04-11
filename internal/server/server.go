package server

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/volchok96/grpc-file-service/api"
	"github.com/volchok96/grpc-file-service/internal/config"
	"github.com/volchok96/grpc-file-service/internal/service"
	"github.com/volchok96/grpc-file-service/internal/storage"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type Server struct {
	grpcServer *grpc.Server
	config     *config.Config
}

type DiskStorage struct {
	storagePath string
}

func (s *DiskStorage) SaveFile(filename string, data []byte) error {
	filePath := filepath.Join(s.storagePath, filename)
	return os.WriteFile(filePath, data, 0644)
}

func (s *DiskStorage) GetFile(filename string) ([]byte, error) {
	filePath := filepath.Join(s.storagePath, filename)
	return os.ReadFile(filePath)
}

func (s *DiskStorage) ListFiles() ([]storage.FileInfo, error) {
	files, err := os.ReadDir(s.storagePath)
	if err != nil {
		if os.IsNotExist(err) {
			return []storage.FileInfo{}, nil
		}
		return nil, err
	}

	var fileInfos []storage.FileInfo
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		info, err := file.Info()
		if err != nil {
			continue
		}

		fileInfos = append(fileInfos, storage.FileInfo{
			Name:      file.Name(),
			CreatedAt: info.ModTime(),
			UpdatedAt: info.ModTime(),
		})
	}

	return fileInfos, nil
}

func NewServer(cfg *config.Config) *Server {
	diskStorage := &DiskStorage{storagePath: cfg.StoragePath}
	fileStorageService := service.NewFileStorageService(
		diskStorage,
		cfg.UploadLimit,
		cfg.DownloadLimit,
		cfg.ListLimit,
		cfg.MaxChunkSize,
	)

	grpcServer := grpc.NewServer()
	api.RegisterFileStorageServer(grpcServer, fileStorageService)

	reflection.Register(grpcServer)

	return &Server{
		grpcServer: grpcServer,
		config:     cfg,
	}
}

func (s *Server) Run() error {
	lis, err := net.Listen("tcp", s.config.GRPCPort)
	if err != nil {
		return err
	}

	go func() {
		log.Printf("Starting gRPC server on %s", s.config.GRPCPort)
		if err := s.grpcServer.Serve(lis); err != nil {
			log.Fatalf("Failed to serve: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), s.config.ShutdownTimeout)
	defer cancel()

	stopped := make(chan struct{})
	go func() {
		s.grpcServer.GracefulStop()
		close(stopped)
	}()

	select {
	case <-ctx.Done():
		s.grpcServer.Stop()
		return ctx.Err()
	case <-stopped:
		return nil
	}
}
