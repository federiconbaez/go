package usecases

import (
	"context"

	https://github.com/federiconbaez/gogrpc-go-android/server-go/internal/domain/entities"
	https://github.com/federiconbaez/gogrpc-go-android/server-go/internal/domain/ports"
	"github.com/google/uuid"
)

// IdeaUseCases contiene los casos de uso para ideas
type IdeaUseCases struct {
	ideaRepo ports.IdeaRepository
	eventBus ports.EventBus
}

// NewIdeaUseCases crea una nueva instancia de IdeaUseCases
func NewIdeaUseCases(ideaRepo ports.IdeaRepository, eventBus ports.EventBus) *IdeaUseCases {
	return &IdeaUseCases{
		ideaRepo: ideaRepo,
		eventBus: eventBus,
	}
}

// CreateIdea crea una nueva idea
func (uc *IdeaUseCases) CreateIdea(ctx context.Context, title, content string, category entities.IdeaCategory, userID uuid.UUID, tags []string, priority int32) (*entities.Idea, error) {
	idea := entities.NewIdea(title, content, category, userID, tags, priority)
	
	if err := idea.Validate(); err != nil {
		return nil, err
	}
	
	if err := uc.ideaRepo.Create(ctx, idea); err != nil {
		return nil, err
	}
	
	// Publicar evento de idea creada
	if uc.eventBus != nil {
		event := &IdeaCreatedEvent{
			IdeaID: idea.ID,
			UserID: userID,
			Title:  title,
		}
		uc.eventBus.Publish(ctx, event)
	}
	
	return idea, nil
}

// GetIdea obtiene una idea por ID
func (uc *IdeaUseCases) GetIdea(ctx context.Context, id, userID uuid.UUID) (*entities.Idea, error) {
	idea, err := uc.ideaRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	
	if !idea.IsOwnedBy(userID) {
		return nil, entities.ErrIdeaUnauthorized
	}
	
	return idea, nil
}

// ListIdeas obtiene las ideas de un usuario con filtros
func (uc *IdeaUseCases) ListIdeas(ctx context.Context, userID uuid.UUID, filters ports.IdeaFilters) ([]*entities.Idea, int, error) {
	return uc.ideaRepo.GetByUserID(ctx, userID, filters)
}

// UpdateIdea actualiza una idea existente
func (uc *IdeaUseCases) UpdateIdea(ctx context.Context, id, userID uuid.UUID, title, content string, tags []string, category entities.IdeaCategory, status entities.IdeaStatus, priority int32) (*entities.Idea, error) {
	idea, err := uc.ideaRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	
	if !idea.IsOwnedBy(userID) {
		return nil, entities.ErrIdeaUnauthorized
	}
	
	idea.Update(title, content, tags, category, status, priority)
	
	if err := idea.Validate(); err != nil {
		return nil, err
	}
	
	if err := uc.ideaRepo.Update(ctx, idea); err != nil {
		return nil, err
	}
	
	// Publicar evento de idea actualizada
	if uc.eventBus != nil {
		event := &IdeaUpdatedEvent{
			IdeaID: idea.ID,
			UserID: userID,
			Title:  idea.Title,
		}
		uc.eventBus.Publish(ctx, event)
	}
	
	return idea, nil
}

// DeleteIdea elimina una idea
func (uc *IdeaUseCases) DeleteIdea(ctx context.Context, id, userID uuid.UUID) error {
	idea, err := uc.ideaRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	
	if !idea.IsOwnedBy(userID) {
		return entities.ErrIdeaUnauthorized
	}
	
	if err := uc.ideaRepo.Delete(ctx, id); err != nil {
		return err
	}
	
	// Publicar evento de idea eliminada
	if uc.eventBus != nil {
		event := &IdeaDeletedEvent{
			IdeaID: id,
			UserID: userID,
		}
		uc.eventBus.Publish(ctx, event)
	}
	
	return nil
}

// Events
type IdeaCreatedEvent struct {
	IdeaID uuid.UUID
	UserID uuid.UUID
	Title  string
}

type IdeaUpdatedEvent struct {
	IdeaID uuid.UUID
	UserID uuid.UUID
	Title  string
}

type IdeaDeletedEvent struct {
	IdeaID uuid.UUID
	UserID uuid.UUID
}