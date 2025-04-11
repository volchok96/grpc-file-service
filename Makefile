# Параметры
GO = go
GOBUILD = $(GO) build
GOTEST = $(GO) test
GOTAGS =

# Путь до исходных файлов
SRC_DIR = ./cmd/client
SERVER_DIR = ./cmd/server

# Название исполнимого файла
CLIENT_BINARY = client
SERVER_BINARY = server

# Путь к директории для хранения зависимостей
BIN_DIR = ./bin

# Порт для запуска
PORT = 50051

# Основная цель - сборка
all: build

# Сборка клиента
build-client:
	@echo "Building client..."
	$(GOBUILD) -o $(BIN_DIR)/$(CLIENT_BINARY) $(SRC_DIR)/main.go

# Сборка сервера
build-server:
	@echo "Building server..."
	$(GOBUILD) -o $(BIN_DIR)/$(SERVER_BINARY) $(SERVER_DIR)/main.go

# Сборка всего проекта
build: build-client build-server

# Запуск сервера
run-server: build-server
	@echo "Running server..."
	$(BIN_DIR)/$(SERVER_BINARY) -port $(PORT)

# Запуск клиента
run-client: build-client
	@echo "Running client..."
	$(BIN_DIR)/$(CLIENT_BINARY)

# Запуск всех тестов
test:
	@echo "Running tests..."
	$(GOTEST) ./...

# Очистка всех собранных файлов
clean:
	@echo "Cleaning up..."
	rm -rf $(BIN_DIR)/*

# Сборка и запуск сервера и клиента в одном процессе
run-all: run-server run-client

# Запуск клиента с заданными аргументами
run-client-args: build-client
	@echo "Running client with args: $(ARGS)"
	$(BIN_DIR)/$(CLIENT_BINARY) $(ARGS)

# Включение дополнительных тегов Go (например, для сборки в разные архитектуры)
tags:
	$(eval GOTAGS := $(TAG))

.PHONY: build build-client build-server run-server run-client test clean run-all run-client-args tags
