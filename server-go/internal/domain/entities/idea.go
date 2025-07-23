package entities

import (
	"time"

	"github.com/google/uuid"
)

// IdeaCategory representa las categorías de ideas
type IdeaCategory int32

const (
	IdeaCategoryUnspecified IdeaCategory = 0
	IdeaCategoryBusiness    IdeaCategory = 1
	IdeaCategoryPersonal    IdeaCategory = 2
	IdeaCategoryTechnical   IdeaCategory = 3
	IdeaCategoryCreative    IdeaCategory = 4
	IdeaCategoryResearch    IdeaCategory = 5
)

// IdeaStatus representa el estado de una idea
type IdeaStatus int32

const (
	IdeaStatusUnspecified IdeaStatus = 0
	IdeaStatusDraft       IdeaStatus = 1
	IdeaStatusActive      IdeaStatus = 2
	IdeaStatusOnHold      IdeaStatus = 3
	IdeaStatusCompleted   IdeaStatus = 4
	IdeaStatusArchived    IdeaStatus = 5
)

// Idea representa una idea en el dominio
type Idea struct {
	ID           uuid.UUID
	Title        string
	Content      string
	Tags         []string
	Category     IdeaCategory
	Status       IdeaStatus
	CreatedAt    time.Time
	UpdatedAt    time.Time
	UserID       uuid.UUID
	RelatedIdeas []uuid.UUID
	Priority     int32
}

// NewIdea crea una nueva idea con valores por defecto
func NewIdea(title, content string, category IdeaCategory, userID uuid.UUID, tags []string, priority int32) *Idea {
	now := time.Now()
	return &Idea{
		ID:           uuid.New(),
		Title:        title,
		Content:      content,
		Tags:         tags,
		Category:     category,
		Status:       IdeaStatusDraft,
		CreatedAt:    now,
		UpdatedAt:    now,
		UserID:       userID,
		RelatedIdeas: make([]uuid.UUID, 0),
		Priority:     priority,
	}
}

// Update actualiza los campos modificables de la idea
func (i *Idea) Update(title, content string, tags []string, category IdeaCategory, status IdeaStatus, priority int32) {
	if title != "" {
		i.Title = title
	}
	if content != "" {
		i.Content = content
	}
	if tags != nil {
		i.Tags = tags
	}
	if category != IdeaCategoryUnspecified {
		i.Category = category
	}
	if status != IdeaStatusUnspecified {
		i.Status = status
	}
	if priority >= 0 {
		i.Priority = priority
	}
	i.UpdatedAt = time.Now()
}

// AddRelatedIdea añade una idea relacionada
func (i *Idea) AddRelatedIdea(ideaID uuid.UUID) {
	for _, id := range i.RelatedIdeas {
		if id == ideaID {
			return // Ya existe
		}
	}
	i.RelatedIdeas = append(i.RelatedIdeas, ideaID)
	i.UpdatedAt = time.Now()
}

// RemoveRelatedIdea elimina una idea relacionada
func (i *Idea) RemoveRelatedIdea(ideaID uuid.UUID) {
	for idx, id := range i.RelatedIdeas {
		if id == ideaID {
			i.RelatedIdeas = append(i.RelatedIdeas[:idx], i.RelatedIdeas[idx+1:]...)
			i.UpdatedAt = time.Now()
			return
		}
	}
}

// IsOwnedBy verifica si la idea pertenece al usuario especificado
func (i *Idea) IsOwnedBy(userID uuid.UUID) bool {
	return i.UserID == userID
}

// Validate valida que la idea tenga los campos requeridos
func (i *Idea) Validate() error {
	if i.Title == "" {
		return ErrIdeaTitleRequired
	}
	if i.Content == "" {
		return ErrIdeaContentRequired
	}
	if i.UserID == uuid.Nil {
		return ErrIdeaUserIDRequired
	}
	return nil
}