package entities

import (
	"time"

	"github.com/google/uuid"
)

// FileInfo representa la información de un archivo en el dominio
type FileInfo struct {
	ID              uuid.UUID
	Filename        string
	ContentType     string
	Size            int64
	Checksum        string
	CreatedAt       time.Time
	UserID          uuid.UUID
	Compressed      bool
	CompressionType string
	Path            string
}

// NewFileInfo crea una nueva información de archivo
func NewFileInfo(filename, contentType, checksum, path string, size int64, userID uuid.UUID, compressed bool, compressionType string) *FileInfo {
	return &FileInfo{
		ID:              uuid.New(),
		Filename:        filename,
		ContentType:     contentType,
		Size:            size,
		Checksum:        checksum,
		CreatedAt:       time.Now(),
		UserID:          userID,
		Compressed:      compressed,
		CompressionType: compressionType,
		Path:            path,
	}
}

// IsOwnedBy verifica si el archivo pertenece al usuario especificado
func (f *FileInfo) IsOwnedBy(userID uuid.UUID) bool {
	return f.UserID == userID
}

// Validate valida que la información del archivo sea correcta
func (f *FileInfo) Validate() error {
	if f.Filename == "" {
		return ErrFileNameRequired
	}
	if f.UserID == uuid.Nil {
		return ErrFileUserIDRequired
	}
	if f.Size < 0 {
		return ErrFileSizeExceeded
	}
	return nil
}

// IsImage verifica si el archivo es una imagen
func (f *FileInfo) IsImage() bool {
	return f.ContentType == "image/jpeg" || 
		   f.ContentType == "image/png" || 
		   f.ContentType == "image/gif" || 
		   f.ContentType == "image/webp"
}

// IsDocument verifica si el archivo es un documento
func (f *FileInfo) IsDocument() bool {
	return f.ContentType == "application/pdf" || 
		   f.ContentType == "application/msword" || 
		   f.ContentType == "application/vnd.openxmlformats-officedocument.wordprocessingml.document" ||
		   f.ContentType == "text/plain"
}

// GetFileExtension obtiene la extensión del archivo basada en el content type
func (f *FileInfo) GetFileExtension() string {
	switch f.ContentType {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/gif":
		return ".gif"
	case "image/webp":
		return ".webp"
	case "application/pdf":
		return ".pdf"
	case "text/plain":
		return ".txt"
	case "application/json":
		return ".json"
	default:
		return ""
	}
}