# Dockerfile para el servidor Go
FROM golang:1.21-alpine AS builder

# Instalar dependencias del sistema
RUN apk add --no-cache git ca-certificates tzdata

# Establecer el directorio de trabajo
WORKDIR /app

# Copiar archivos de go modules primero para cache de dependencias
COPY go.mod go.sum ./
RUN go mod download

# Copiar el código fuente
COPY . .

# Generar código desde protos si es necesario
RUN if [ -d "../proto" ]; then \
        apk add --no-cache protobuf && \
        go install google.golang.org/protobuf/cmd/protoc-gen-go@latest && \
        go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest && \
        make proto; \
    fi

# Compilar la aplicación
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags="-w -s" -o notebook-server cmd/server/main.go

# Imagen final
FROM alpine:latest

# Instalar certificados CA y timezone data
RUN apk --no-cache add ca-certificates tzdata

# Crear usuario no-root
RUN addgroup -g 1001 appgroup && \
    adduser -D -s /bin/sh -u 1001 -G appgroup appuser

# Establecer directorio de trabajo
WORKDIR /app

# Copiar el binario desde el stage de build
COPY --from=builder /app/notebook-server .

# Crear directorio para uploads
RUN mkdir -p uploads && chown -R appuser:appgroup /app

# Cambiar al usuario no-root
USER appuser

# Exponer el puerto gRPC
EXPOSE 50051

# Variables de entorno por defecto
ENV GRPC_PORT=50051
ENV DB_HOST=localhost
ENV DB_PORT=5432
ENV DB_USER=postgres
ENV DB_PASSWORD=postgres
ENV DB_NAME=notebook
ENV DB_SSL_MODE=disable

# Healthcheck
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD nc -z localhost 50051 || exit 1

# Comando para ejecutar la aplicación
CMD ["./notebook-server"]