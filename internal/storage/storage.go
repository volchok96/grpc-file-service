package storage

import (
	"time"
)

type FileInfo struct {
	Name      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type FileStorage interface {
	SaveFile(filename string, data []byte) error
	GetFile(filename string) ([]byte, error)
	ListFiles() ([]FileInfo, error)
}
