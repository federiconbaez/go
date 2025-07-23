package main

import (
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	https://github.com/federiconbaez/gogrpc-go-android/server-go/internal/application/usecases"
	grpcAdapter https://github.com/federiconbaez/gogrpc-go-android/server-go/internal/infrastructure/adapters/grpc"
	https://github.com/federiconbaez/gogrpc-go-android/server-go/internal/infrastructure/adapters/postgres"
	https://github.com/federiconbaez/gogrpc-go-android/server-go/internal/infrastructure/services"
	pb https://github.com/federiconbaez/gogrpc-go-android/proto"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	// Configurar logger
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Sync()

	// Configuración de la base de datos
	dbConfig := postgres.Config{
		Host:     getEnv("DB_HOST", "localhost"),
		Port:     getEnv("DB_PORT", "5432"),
		User:     getEnv("DB_USER", "postgres"),
		Password: getEnv("DB_PASSWORD", "postgres"),
		DBName:   getEnv("DB_NAME", "notebook"),
		SSLMode:  getEnv("DB_SSL_MODE", "disable"),
	}

	// Inicializar repositorios
	db, err := postgres.NewConnection(dbConfig)
	if err != nil {
		logger.Fatal("Failed to connect to database", zap.Error(err))
	}
	defer db.Close()

	ideaRepo := postgres.NewIdeaRepository(db)
	reminderRepo := postgres.NewReminderRepository(db)
	fileRepo := postgres.NewFileRepository(db)
	progressRepo := postgres.NewProgressRepository(db)

	// Inicializar servicios
	fileStorageService := services.NewLocalFileStorageService("./uploads")
	compressionService := services.NewCompressionService()
	eventBus := services.NewInMemoryEventBus()
	notificationService := services.NewNotificationService(eventBus)

	// Inicializar casos de uso
	ideaUseCases := usecases.NewIdeaUseCases(ideaRepo, eventBus)
	reminderUseCases := usecases.NewReminderUseCases(reminderRepo, notificationService, eventBus)
	fileUseCases := usecases.NewFileUseCases(fileRepo, fileStorageService, eventBus)
	progressUseCases := usecases.NewProgressUseCases(progressRepo, eventBus)

	// Crear el servidor gRPC
	notebookServer := grpcAdapter.NewNotebookServer(
		ideaUseCases,
		reminderUseCases,
		fileUseCases,
		progressUseCases,
		notificationService,
	)

	// Configurar el servidor gRPC
	port := getEnv("GRPC_PORT", "50051")
	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		logger.Fatal("Failed to listen", zap.Error(err))
	}

	s := grpc.NewServer()
	pb.RegisterNotebookServiceServer(s, notebookServer)
	
	// Habilitar reflection para herramientas como grpcurl
	reflection.Register(s)

	logger.Info("Starting gRPC server", zap.String("port", port))

	// Manejar señales para shutdown graceful
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
		
		logger.Info("Shutting down gRPC server...")
		s.GracefulStop()
	}()

	// Iniciar el servidor
	if err := s.Serve(listener); err != nil {
		logger.Fatal("Failed to serve gRPC server", zap.Error(err))
	}
}

// getEnv obtiene una variable de entorno con un valor por defecto
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}