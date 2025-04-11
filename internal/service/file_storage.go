package service

import (
	"context"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/volchok96/grpc-file-service/api"
	"github.com/volchok96/grpc-file-service/internal/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type FileStorageService struct {
	api.UnimplementedFileStorageServer
	storage      storage.FileStorage
	uploadSem    chan struct{}
	downloadSem  chan struct{}
	listSem      chan struct{}
	maxChunkSize int
	mu           sync.Mutex
}

func NewFileStorageService(
	storage storage.FileStorage,
	uploadLimit, downloadLimit, listLimit, maxChunkSize int,
) *FileStorageService {
	return &FileStorageService{
		storage:      storage,
		uploadSem:    make(chan struct{}, uploadLimit),
		downloadSem:  make(chan struct{}, downloadLimit),
		listSem:      make(chan struct{}, listLimit),
		maxChunkSize: maxChunkSize,
	}
}

func (s *FileStorageService) UploadFile(stream api.FileStorage_UploadFileServer) error {
	s.mu.Lock()
	s.uploadSem <- struct{}{}
	s.mu.Unlock()
	defer func() { <-s.uploadSem }()

	var filename string
	var data []byte

	for {
		req, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return status.Errorf(codes.Unknown, "cannot receive chunk: %v", err)
		}

		if filename == "" {
			filename = req.GetFilename()
			if filename == "" {
				return status.Error(codes.InvalidArgument, "filename cannot be empty")
			}
			filename = filepath.Base(filename) // защита от вложенных путей
		}

		data = append(data, req.GetChunk()...)
	}

	if err := s.storage.SaveFile(filename, data); err != nil {
		return status.Errorf(codes.Internal, "cannot save file: %v", err)
	}

	log.Printf("Saved file: %s (%d bytes)", filename, len(data))

	return stream.SendAndClose(&api.UploadResponse{
		Filename: filename,
		Size:     uint32(len(data)),
	})
}

func (s *FileStorageService) DownloadFile(req *api.DownloadRequest, stream api.FileStorage_DownloadFileServer) error {
	s.mu.Lock()
	s.downloadSem <- struct{}{}
	s.mu.Unlock()
	defer func() { <-s.downloadSem }()

	filename := req.GetFilename()
	if filename == "" {
		return status.Error(codes.InvalidArgument, "filename cannot be empty")
	}

	data, err := s.storage.GetFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return status.Errorf(codes.NotFound, "file not found: %s", filename)
		}
		return status.Errorf(codes.Internal, "cannot read file: %v", err)
	}

	for currentByte := 0; currentByte < len(data); currentByte += s.maxChunkSize {
		end := currentByte + s.maxChunkSize
		if end > len(data) {
			end = len(data)
		}

		if err := stream.Send(&api.DownloadResponse{
			Chunk: data[currentByte:end],
		}); err != nil {
			return status.Errorf(codes.Internal, "cannot send chunk: %v", err)
		}
	}

	return nil
}

func (s *FileStorageService) ListFiles(ctx context.Context, req *api.ListRequest) (*api.ListResponse, error) {
	s.mu.Lock()
	s.listSem <- struct{}{}
	s.mu.Unlock()
	defer func() { <-s.listSem }()

	files, err := s.storage.ListFiles()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "cannot list files: %v", err)
	}

	response := &api.ListResponse{
		Files: make([]*api.FileInfo, 0, len(files)),
	}

	for _, file := range files {
		response.Files = append(response.Files, &api.FileInfo{
			Filename:  file.Name,
			CreatedAt: file.CreatedAt.Format(time.RFC3339),
			UpdatedAt: file.UpdatedAt.Format(time.RFC3339),
		})
	}

	return response, nil
}
