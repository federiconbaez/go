# Makefile para el proyecto Notebook Server

# Variables
PROTO_DIR=../proto
PROTO_FILES=$(PROTO_DIR)/*.proto
GO_OUT=./proto
BINARY_NAME=notebook-server
DOCKER_IMAGE=notebook-server
VERSION=1.0.0

# Colores para output
GREEN=\033[0;32m
YELLOW=\033[1;33m
RED=\033[0;31m
NC=\033[0m # No Color

.PHONY: help install-tools proto build run test clean docker lint vet fmt deps

help: ## Mostrar ayuda
	@echo "$(GREEN)Comandos disponibles:$(NC)"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "$(YELLOW)%-15s$(NC) %s\n", $$1, $$2}'

install-tools: ## Instalar herramientas necesarias
	@echo "$(GREEN)Instalando herramientas...$(NC)"
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

proto: ## Generar código Go desde archivos .proto
	@echo "$(GREEN)Generando código desde archivos .proto...$(NC)"
	@mkdir -p $(GO_OUT)
	protoc --proto_path=$(PROTO_DIR) \
		--go_out=$(GO_OUT) \
		--go_opt=paths=source_relative \
		--go-grpc_out=$(GO_OUT) \
		--go-grpc_opt=paths=source_relative \
		$(PROTO_FILES)

deps: ## Descargar dependencias
	@echo "$(GREEN)Descargando dependencias...$(NC)"
	go mod download
	go mod tidy

fmt: ## Formatear código
	@echo "$(GREEN)Formateando código...$(NC)"
	go fmt ./...

vet: ## Ejecutar go vet
	@echo "$(GREEN)Ejecutando go vet...$(NC)"
	go vet ./...

lint: ## Ejecutar linter
	@echo "$(GREEN)Ejecutando linter...$(NC)"
	golangci-lint run

test: ## Ejecutar tests
	@echo "$(GREEN)Ejecutando tests...$(NC)"
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

build: deps fmt vet ## Compilar el servidor
	@echo "$(GREEN)Compilando servidor...$(NC)"
	CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bin/$(BINARY_NAME) cmd/server/main.go

run: ## Ejecutar el servidor
	@echo "$(GREEN)Ejecutando servidor...$(NC)"
	go run cmd/server/main.go

run-dev: ## Ejecutar en modo desarrollo con hot reload
	@echo "$(GREEN)Ejecutando en modo desarrollo...$(NC)"
	air -c .air.toml

docker-build: ## Construir imagen Docker
	@echo "$(GREEN)Construyendo imagen Docker...$(NC)"
	docker build -t $(DOCKER_IMAGE):$(VERSION) .
	docker tag $(DOCKER_IMAGE):$(VERSION) $(DOCKER_IMAGE):latest

docker-run: ## Ejecutar contenedor Docker
	@echo "$(GREEN)Ejecutando contenedor Docker...$(NC)"
	docker run -p 50051:50051 --env-file .env $(DOCKER_IMAGE):latest

docker-compose-up: ## Levantar servicios con docker-compose
	@echo "$(GREEN)Levantando servicios con docker-compose...$(NC)"
	docker-compose up -d

docker-compose-down: ## Bajar servicios de docker-compose
	@echo "$(GREEN)Bajando servicios de docker-compose...$(NC)"
	docker-compose down

migrate-up: ## Ejecutar migraciones hacia arriba
	@echo "$(GREEN)Ejecutando migraciones...$(NC)"
	goose -dir migrations postgres "$(DB_URL)" up

migrate-down: ## Ejecutar migraciones hacia abajo
	@echo "$(GREEN)Revirtiendo migraciones...$(NC)"
	goose -dir migrations postgres "$(DB_URL)" down

migrate-status: ## Ver estado de migraciones
	@echo "$(GREEN)Estado de migraciones:$(NC)"
	goose -dir migrations postgres "$(DB_URL)" status

clean: ## Limpiar archivos generados
	@echo "$(GREEN)Limpiando archivos generados...$(NC)"
	rm -rf bin/
	rm -f coverage.out coverage.html
	rm -rf $(GO_OUT)
	docker image prune -f

benchmark: ## Ejecutar benchmarks
	@echo "$(GREEN)Ejecutando benchmarks...$(NC)"
	go test -bench=. -benchmem ./...

security: ## Ejecutar análisis de seguridad
	@echo "$(GREEN)Ejecutando análisis de seguridad...$(NC)"
	gosec ./...

all: clean deps proto fmt vet lint test build ## Ejecutar pipeline completo

# Variables de entorno por defecto
DB_URL ?= postgres://postgres:postgres@localhost:5432/notebook?sslmode=disable
GRPC_PORT ?= 50051
DB_HOST ?= localhost
DB_PORT ?= 5432
DB_USER ?= postgres
DB_PASSWORD ?= postgres
DB_NAME ?= notebook