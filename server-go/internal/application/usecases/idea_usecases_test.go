package usecases

import (
	"context"
	"testing"

	https://github.com/federiconbaez/gogrpc-go-android/server-go/internal/domain/entities"
	https://github.com/federiconbaez/gogrpc-go-android/server-go/internal/domain/ports"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockIdeaRepository es un mock del repositorio de ideas
type MockIdeaRepository struct {
	mock.Mock
}

func (m *MockIdeaRepository) Create(ctx context.Context, idea *entities.Idea) error {
	args := m.Called(ctx, idea)
	return args.Error(0)
}

func (m *MockIdeaRepository) GetByID(ctx context.Context, id uuid.UUID) (*entities.Idea, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.Idea), args.Error(1)
}

func (m *MockIdeaRepository) GetByUserID(ctx context.Context, userID uuid.UUID, filters ports.IdeaFilters) ([]*entities.Idea, int, error) {
	args := m.Called(ctx, userID, filters)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).([]*entities.Idea), args.Int(1), args.Error(2)
}

func (m *MockIdeaRepository) Update(ctx context.Context, idea *entities.Idea) error {
	args := m.Called(ctx, idea)
	return args.Error(0)
}

func (m *MockIdeaRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// MockEventBus es un mock del bus de eventos
type MockEventBus struct {
	mock.Mock
}

func (m *MockEventBus) Publish(ctx context.Context, event interface{}) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

func (m *MockEventBus) Subscribe(eventType string, handler ports.EventHandler) error {
	args := m.Called(eventType, handler)
	return args.Error(0)
}

func TestCreateIdea_Success(t *testing.T) {
	// Arrange
	mockRepo := new(MockIdeaRepository)
	mockEventBus := new(MockEventBus)
	useCase := NewIdeaUseCases(mockRepo, mockEventBus)

	userID := uuid.New()
	title := "Test Idea"
	content := "Test content"
	category := entities.IdeaCategoryBusiness
	tags := []string{"test"}
	priority := int32(5)

	mockRepo.On("Create", mock.Anything, mock.AnythingOfType("*entities.Idea")).Return(nil)
	mockEventBus.On("Publish", mock.Anything, mock.AnythingOfType("*usecases.IdeaCreatedEvent")).Return(nil)

	// Act
	idea, err := useCase.CreateIdea(context.Background(), title, content, category, userID, tags, priority)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, idea)
	assert.Equal(t, title, idea.Title)
	assert.Equal(t, content, idea.Content)
	assert.Equal(t, category, idea.Category)
	assert.Equal(t, userID, idea.UserID)
	assert.Equal(t, tags, idea.Tags)
	assert.Equal(t, priority, idea.Priority)
	
	mockRepo.AssertExpectations(t)
	mockEventBus.AssertExpectations(t)
}

func TestCreateIdea_ValidationError(t *testing.T) {
	// Arrange
	mockRepo := new(MockIdeaRepository)
	mockEventBus := new(MockEventBus)
	useCase := NewIdeaUseCases(mockRepo, mockEventBus)

	userID := uuid.New()
	title := "" // Invalid title
	content := "Test content"
	category := entities.IdeaCategoryBusiness

	// Act
	idea, err := useCase.CreateIdea(context.Background(), title, content, category, userID, []string{}, 5)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, idea)
	assert.Equal(t, entities.ErrIdeaTitleRequired, err)
	
	// Repository and event bus should not be called
	mockRepo.AssertNotCalled(t, "Create")
	mockEventBus.AssertNotCalled(t, "Publish")
}

func TestCreateIdea_RepositoryError(t *testing.T) {
	// Arrange
	mockRepo := new(MockIdeaRepository)
	mockEventBus := new(MockEventBus)
	useCase := NewIdeaUseCases(mockRepo, mockEventBus)

	userID := uuid.New()
	title := "Test Idea"
	content := "Test content"
	category := entities.IdeaCategoryBusiness

	expectedError := assert.AnError
	mockRepo.On("Create", mock.Anything, mock.AnythingOfType("*entities.Idea")).Return(expectedError)

	// Act
	idea, err := useCase.CreateIdea(context.Background(), title, content, category, userID, []string{}, 5)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, idea)
	assert.Equal(t, expectedError, err)
	
	mockRepo.AssertExpectations(t)
	mockEventBus.AssertNotCalled(t, "Publish")
}

func TestGetIdea_Success(t *testing.T) {
	// Arrange
	mockRepo := new(MockIdeaRepository)
	mockEventBus := new(MockEventBus)
	useCase := NewIdeaUseCases(mockRepo, mockEventBus)

	ideaID := uuid.New()
	userID := uuid.New()
	expectedIdea := &entities.Idea{
		ID:     ideaID,
		Title:  "Test Idea",
		UserID: userID,
	}

	mockRepo.On("GetByID", mock.Anything, ideaID).Return(expectedIdea, nil)

	// Act
	idea, err := useCase.GetIdea(context.Background(), ideaID, userID)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, expectedIdea, idea)
	
	mockRepo.AssertExpectations(t)
}

func TestGetIdea_NotFound(t *testing.T) {
	// Arrange
	mockRepo := new(MockIdeaRepository)
	mockEventBus := new(MockEventBus)
	useCase := NewIdeaUseCases(mockRepo, mockEventBus)

	ideaID := uuid.New()
	userID := uuid.New()

	mockRepo.On("GetByID", mock.Anything, ideaID).Return(nil, entities.ErrIdeaNotFound)

	// Act
	idea, err := useCase.GetIdea(context.Background(), ideaID, userID)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, idea)
	assert.Equal(t, entities.ErrIdeaNotFound, err)
	
	mockRepo.AssertExpectations(t)
}

func TestGetIdea_Unauthorized(t *testing.T) {
	// Arrange
	mockRepo := new(MockIdeaRepository)
	mockEventBus := new(MockEventBus)
	useCase := NewIdeaUseCases(mockRepo, mockEventBus)

	ideaID := uuid.New()
	userID := uuid.New()
	differentUserID := uuid.New()
	
	existingIdea := &entities.Idea{
		ID:     ideaID,
		Title:  "Test Idea",
		UserID: differentUserID, // Different user
	}

	mockRepo.On("GetByID", mock.Anything, ideaID).Return(existingIdea, nil)

	// Act
	idea, err := useCase.GetIdea(context.Background(), ideaID, userID)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, idea)
	assert.Equal(t, entities.ErrIdeaUnauthorized, err)
	
	mockRepo.AssertExpectations(t)
}

func TestListIdeas_Success(t *testing.T) {
	// Arrange
	mockRepo := new(MockIdeaRepository)
	mockEventBus := new(MockEventBus)
	useCase := NewIdeaUseCases(mockRepo, mockEventBus)

	userID := uuid.New()
	filters := ports.IdeaFilters{
		Category: entities.IdeaCategoryBusiness,
		Page:     1,
		PageSize: 10,
	}

	expectedIdeas := []*entities.Idea{
		{ID: uuid.New(), Title: "Idea 1", UserID: userID},
		{ID: uuid.New(), Title: "Idea 2", UserID: userID},
	}
	expectedCount := 2

	mockRepo.On("GetByUserID", mock.Anything, userID, filters).Return(expectedIdeas, expectedCount, nil)

	// Act
	ideas, count, err := useCase.ListIdeas(context.Background(), userID, filters)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, expectedIdeas, ideas)
	assert.Equal(t, expectedCount, count)
	
	mockRepo.AssertExpectations(t)
}

func TestUpdateIdea_Success(t *testing.T) {
	// Arrange
	mockRepo := new(MockIdeaRepository)
	mockEventBus := new(MockEventBus)
	useCase := NewIdeaUseCases(mockRepo, mockEventBus)

	ideaID := uuid.New()
	userID := uuid.New()
	existingIdea := &entities.Idea{
		ID:     ideaID,
		Title:  "Original Title",
		UserID: userID,
	}

	newTitle := "Updated Title"
	
	mockRepo.On("GetByID", mock.Anything, ideaID).Return(existingIdea, nil)
	mockRepo.On("Update", mock.Anything, mock.AnythingOfType("*entities.Idea")).Return(nil)
	mockEventBus.On("Publish", mock.Anything, mock.AnythingOfType("*usecases.IdeaUpdatedEvent")).Return(nil)

	// Act
	updatedIdea, err := useCase.UpdateIdea(context.Background(), ideaID, userID, newTitle, "", []string{}, entities.IdeaCategoryUnspecified, entities.IdeaStatusUnspecified, 0)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, updatedIdea)
	assert.Equal(t, newTitle, updatedIdea.Title)
	
	mockRepo.AssertExpectations(t)
	mockEventBus.AssertExpectations(t)
}

func TestDeleteIdea_Success(t *testing.T) {
	// Arrange
	mockRepo := new(MockIdeaRepository)
	mockEventBus := new(MockEventBus)
	useCase := NewIdeaUseCases(mockRepo, mockEventBus)

	ideaID := uuid.New()
	userID := uuid.New()
	existingIdea := &entities.Idea{
		ID:     ideaID,
		Title:  "Test Idea",
		UserID: userID,
	}

	mockRepo.On("GetByID", mock.Anything, ideaID).Return(existingIdea, nil)
	mockRepo.On("Delete", mock.Anything, ideaID).Return(nil)
	mockEventBus.On("Publish", mock.Anything, mock.AnythingOfType("*usecases.IdeaDeletedEvent")).Return(nil)

	// Act
	err := useCase.DeleteIdea(context.Background(), ideaID, userID)

	// Assert
	require.NoError(t, err)
	
	mockRepo.AssertExpectations(t)
	mockEventBus.AssertExpectations(t)
}

func TestDeleteIdea_Unauthorized(t *testing.T) {
	// Arrange
	mockRepo := new(MockIdeaRepository)
	mockEventBus := new(MockEventBus)
	useCase := NewIdeaUseCases(mockRepo, mockEventBus)

	ideaID := uuid.New()
	userID := uuid.New()
	differentUserID := uuid.New()
	
	existingIdea := &entities.Idea{
		ID:     ideaID,
		Title:  "Test Idea",
		UserID: differentUserID, // Different user
	}

	mockRepo.On("GetByID", mock.Anything, ideaID).Return(existingIdea, nil)

	// Act
	err := useCase.DeleteIdea(context.Background(), ideaID, userID)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, entities.ErrIdeaUnauthorized, err)
	
	mockRepo.AssertExpectations(t)
	mockRepo.AssertNotCalled(t, "Delete")
	mockEventBus.AssertNotCalled(t, "Publish")
}

// Integration-style test
func TestIdeaUseCases_IntegrationFlow(t *testing.T) {
	// Arrange
	mockRepo := new(MockIdeaRepository)
	mockEventBus := new(MockEventBus)
	useCase := NewIdeaUseCases(mockRepo, mockEventBus)

	userID := uuid.New()
	
	// Setup mocks for create
	mockRepo.On("Create", mock.Anything, mock.AnythingOfType("*entities.Idea")).Return(nil)
	mockEventBus.On("Publish", mock.Anything, mock.AnythingOfType("*usecases.IdeaCreatedEvent")).Return(nil)

	// Act 1: Create idea
	idea, err := useCase.CreateIdea(context.Background(), "Test Idea", "Content", entities.IdeaCategoryBusiness, userID, []string{"test"}, 5)
	require.NoError(t, err)
	require.NotNil(t, idea)

	// Setup mocks for get
	mockRepo.On("GetByID", mock.Anything, idea.ID).Return(idea, nil)

	// Act 2: Get idea
	retrievedIdea, err := useCase.GetIdea(context.Background(), idea.ID, userID)
	require.NoError(t, err)
	assert.Equal(t, idea.Title, retrievedIdea.Title)

	// Assert all expectations
	mockRepo.AssertExpectations(t)
	mockEventBus.AssertExpectations(t)
}

// Benchmark tests
func BenchmarkCreateIdea(b *testing.B) {
	mockRepo := new(MockIdeaRepository)
	mockEventBus := new(MockEventBus)
	useCase := NewIdeaUseCases(mockRepo, mockEventBus)

	userID := uuid.New()
	mockRepo.On("Create", mock.Anything, mock.AnythingOfType("*entities.Idea")).Return(nil)
	mockEventBus.On("Publish", mock.Anything, mock.AnythingOfType("*usecases.IdeaCreatedEvent")).Return(nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		useCase.CreateIdea(context.Background(), "Benchmark Idea", "Content", entities.IdeaCategoryBusiness, userID, []string{}, 5)
	}
}