package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Config contiene la configuración de la base de datos
type Config struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

// NewConnection crea una nueva conexión a la base de datos PostgreSQL
func NewConnection(config Config) (*pgxpool.Pool, error) {
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		config.Host, config.Port, config.User, config.Password, config.DBName, config.SSLMode)

	poolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database config: %w", err)
	}

	// Configurar pool de conexiones
	poolConfig.MaxConns = 25
	poolConfig.MinConns = 5
	poolConfig.MaxConnLifetime = time.Hour
	poolConfig.MaxConnIdleTime = time.Minute * 30

	pool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Verificar conexión
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return pool, nil
}

// NewReminderRepository crea un nuevo repositorio de recordatorios
func NewReminderRepository(db *pgxpool.Pool) *reminderRepository {
	return &reminderRepository{db: db}
}

// NewFileRepository crea un nuevo repositorio de archivos
func NewFileRepository(db *pgxpool.Pool) *fileRepository {
	return &fileRepository{db: db}
}

// NewProgressRepository crea un nuevo repositorio de progreso
func NewProgressRepository(db *pgxpool.Pool) *progressRepository {
	return &progressRepository{db: db}
}

// Estructuras placeholder para los repositorios
type reminderRepository struct {
	db *pgxpool.Pool
}

type fileRepository struct {
	db *pgxpool.Pool
}

type progressRepository struct {
	db *pgxpool.Pool
}