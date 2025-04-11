package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/volchok96/grpc-file-service/api"
	"google.golang.org/grpc"
)

// Глобальная переменная для пути к файлам
var fileBasePath = "files/files_client"

func main() {
	uploadFlag := flag.String("upload", "", "Comma-separated list of filenames to upload (from files/files_client)")
	downloadFlag := flag.String("download", "", "Comma-separated list of filenames to download (to files/files_client)")
	flag.Parse()

	conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()

	client := api.NewFileStorageClient(conn)

	switch {
	case *uploadFlag != "":
		files := strings.Split(*uploadFlag, ",")
		uploadFiles(client, files)

	case *downloadFlag != "":
		files := strings.Split(*downloadFlag, ",")
		downloadFiles(client, files)

	default:
		listFiles(client)
	}
}

// uploadFiles загружает указанные файлы из глобальной директории (fileBasePath) на сервер
func uploadFiles(client api.FileStorageClient, filenames []string) {
	for _, name := range filenames {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}

		filePath := filepath.Join(fileBasePath, name)

		// Проверка на существование файла на клиенте перед загрузкой
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			log.Printf("file %s does not exist on client, skipping upload", filePath)
			continue
		}

		// Проверка, если файл уже существует на сервере, чтобы избежать перезагрузки
		// Можно запросить сервер перед загрузкой файла, если он уже существует
		if fileExistsOnServer(client, name) {
			log.Printf("file %s already exists on server, skipping upload", name)
			continue
		}

		// Открытие файла для загрузки
		file, err := os.Open(filePath)
		if err != nil {
			log.Printf("failed to open file %s: %v", filePath, err)
			continue
		}
		defer file.Close()

		stream, err := client.UploadFile(context.Background())
		if err != nil {
			log.Printf("failed to initiate upload for file %s: %v", name, err)
			continue
		}

		// отправка имени файла
		if err := stream.Send(&api.UploadRequest{Filename: name}); err != nil {
			log.Printf("failed to send filename %s: %v", name, err)
			continue
		}

		// отправка содержимого файла по частям
		buf := make([]byte, 64*1024)
		for {
			n, err := file.Read(buf)
			if err == io.EOF {
				break
			}
			if err != nil {
				log.Printf("failed to read file %s: %v", filePath, err)
				break
			}
			if err := stream.Send(&api.UploadRequest{Chunk: buf[:n]}); err != nil {
				log.Printf("failed to send chunk for file %s: %v", name, err)
				break
			}
		}

		resp, err := stream.CloseAndRecv()
		if err != nil {
			log.Printf("upload failed for file %s: %v", name, err)
			continue
		}

		fmt.Printf("Uploaded: %s (%d bytes)\n", resp.Filename, resp.Size)
	}
}

// Проверка, существует ли файл на сервере
func fileExistsOnServer(client api.FileStorageClient, filename string) bool {
	resp, err := client.ListFiles(context.Background(), &api.ListRequest{})
	if err != nil {
		log.Printf("failed to list files on server: %v", err)
		return false
	}

	// Проходим по списку файлов и проверяем, есть ли нужный
	for _, file := range resp.Files {
		if file.Filename == filename {
			return true
		}
	}
	return false
}

// downloadFiles скачивает указанные файлы с сервера в глобальную директорию (fileBasePath)
func downloadFiles(client api.FileStorageClient, filenames []string) {
	for _, name := range filenames {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}

		savePath := filepath.Join(fileBasePath, name)

		// проверка на существование файла
		if _, err := os.Stat(savePath); err == nil {
			log.Printf("file %s already exists, skipping download to avoid overwriting", savePath)
			continue
		}

		// Проверка на наличие файла на сервере
		if !fileExistsOnServer(client, name) {
			log.Printf("file %s not found on the server, skipping download", name)
			continue
		}

		// Создание файла для сохранения на клиенте
		file, err := os.Create(savePath)
		if err != nil {
			log.Printf("failed to create file %s: %v", savePath, err)
			continue
		}
		defer file.Close()

		// Получение и запись содержимого файла
		stream, err := client.DownloadFile(context.Background(), &api.DownloadRequest{Filename: name})
		if err != nil {
			log.Printf("failed to initiate download for file %s: %v", name, err)
			continue
		}

		for {
			resp, err := stream.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				log.Printf("error receiving chunk for file %s: %v", name, err)
				break
			}
			if _, err := file.Write(resp.GetChunk()); err != nil {
				log.Printf("error writing to file %s: %v", name, err)
				break
			}
		}

		log.Printf("Downloaded: %s", name)
	}
}

// listFiles выводит список всех файлов на сервере
func listFiles(client api.FileStorageClient) {
	resp, err := client.ListFiles(context.Background(), &api.ListRequest{})
	if err != nil {
		log.Fatalf("failed to list files: %v", err)
	}

	if len(resp.Files) == 0 {
		fmt.Println("No files found on server.")
		return
	}

	fmt.Println("Files on server:")
	for _, file := range resp.Files {
		fmt.Printf("Filename: %s, Created At: %s, Updated At: %s\n",
			file.Filename, file.CreatedAt, file.UpdatedAt)
	}
}
