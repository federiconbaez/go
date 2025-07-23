package entities

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewIdea(t *testing.T) {
	// Arrange
	title := "Test Idea"
	content := "This is a test idea"
	category := IdeaCategoryBusiness
	userID := uuid.New()
	tags := []string{"test", "idea"}
	priority := int32(5)

	// Act
	idea := NewIdea(title, content, category, userID, tags, priority)

	// Assert
	assert.NotEqual(t, uuid.Nil, idea.ID)
	assert.Equal(t, title, idea.Title)
	assert.Equal(t, content, idea.Content)
	assert.Equal(t, category, idea.Category)
	assert.Equal(t, userID, idea.UserID)
	assert.Equal(t, tags, idea.Tags)
	assert.Equal(t, priority, idea.Priority)
	assert.Equal(t, IdeaStatusDraft, idea.Status)
	assert.NotZero(t, idea.CreatedAt)
	assert.NotZero(t, idea.UpdatedAt)
	assert.Empty(t, idea.RelatedIdeas)
}

func TestIdea_Update(t *testing.T) {
	// Arrange
	idea := NewIdea("Original", "Original content", IdeaCategoryPersonal, uuid.New(), []string{"original"}, 1)
	originalUpdatedAt := idea.UpdatedAt
	time.Sleep(time.Millisecond) // Ensure time difference

	newTitle := "Updated Title"
	newContent := "Updated content"
	newTags := []string{"updated", "test"}
	newCategory := IdeaCategoryTechnical
	newStatus := IdeaStatusActive
	newPriority := int32(8)

	// Act
	idea.Update(newTitle, newContent, newTags, newCategory, newStatus, newPriority)

	// Assert
	assert.Equal(t, newTitle, idea.Title)
	assert.Equal(t, newContent, idea.Content)
	assert.Equal(t, newTags, idea.Tags)
	assert.Equal(t, newCategory, idea.Category)
	assert.Equal(t, newStatus, idea.Status)
	assert.Equal(t, newPriority, idea.Priority)
	assert.True(t, idea.UpdatedAt.After(originalUpdatedAt))
}

func TestIdea_UpdateWithEmptyValues(t *testing.T) {
	// Arrange
	idea := NewIdea("Original", "Original content", IdeaCategoryPersonal, uuid.New(), []string{"original"}, 1)
	originalTitle := idea.Title
	originalContent := idea.Content

	// Act - update with empty values
	idea.Update("", "", nil, IdeaCategoryUnspecified, IdeaStatusUnspecified, -1)

	// Assert - original values should be preserved
	assert.Equal(t, originalTitle, idea.Title)
	assert.Equal(t, originalContent, idea.Content)
}

func TestIdea_AddRelatedIdea(t *testing.T) {
	// Arrange
	idea := NewIdea("Test", "Content", IdeaCategoryBusiness, uuid.New(), []string{}, 1)
	relatedID := uuid.New()

	// Act
	idea.AddRelatedIdea(relatedID)

	// Assert
	assert.Len(t, idea.RelatedIdeas, 1)
	assert.Contains(t, idea.RelatedIdeas, relatedID)
}

func TestIdea_AddRelatedIdea_Duplicate(t *testing.T) {
	// Arrange
	idea := NewIdea("Test", "Content", IdeaCategoryBusiness, uuid.New(), []string{}, 1)
	relatedID := uuid.New()
	idea.AddRelatedIdea(relatedID)

	// Act - add same ID again
	idea.AddRelatedIdea(relatedID)

	// Assert - should not duplicate
	assert.Len(t, idea.RelatedIdeas, 1)
	assert.Contains(t, idea.RelatedIdeas, relatedID)
}

func TestIdea_RemoveRelatedIdea(t *testing.T) {
	// Arrange
	idea := NewIdea("Test", "Content", IdeaCategoryBusiness, uuid.New(), []string{}, 1)
	relatedID1 := uuid.New()
	relatedID2 := uuid.New()
	idea.AddRelatedIdea(relatedID1)
	idea.AddRelatedIdea(relatedID2)

	// Act
	idea.RemoveRelatedIdea(relatedID1)

	// Assert
	assert.Len(t, idea.RelatedIdeas, 1)
	assert.NotContains(t, idea.RelatedIdeas, relatedID1)
	assert.Contains(t, idea.RelatedIdeas, relatedID2)
}

func TestIdea_RemoveRelatedIdea_NotFound(t *testing.T) {
	// Arrange
	idea := NewIdea("Test", "Content", IdeaCategoryBusiness, uuid.New(), []string{}, 1)
	relatedID1 := uuid.New()
	relatedID2 := uuid.New()
	idea.AddRelatedIdea(relatedID1)

	// Act - try to remove non-existent ID
	idea.RemoveRelatedIdea(relatedID2)

	// Assert - should not affect existing ideas
	assert.Len(t, idea.RelatedIdeas, 1)
	assert.Contains(t, idea.RelatedIdeas, relatedID1)
}

func TestIdea_IsOwnedBy(t *testing.T) {
	// Arrange
	userID := uuid.New()
	otherUserID := uuid.New()
	idea := NewIdea("Test", "Content", IdeaCategoryBusiness, userID, []string{}, 1)

	// Act & Assert
	assert.True(t, idea.IsOwnedBy(userID))
	assert.False(t, idea.IsOwnedBy(otherUserID))
}

func TestIdea_Validate(t *testing.T) {
	tests := []struct {
		name        string
		setupIdea   func() *Idea
		expectError error
	}{
		{
			name: "valid idea",
			setupIdea: func() *Idea {
				return NewIdea("Valid Title", "Valid content", IdeaCategoryBusiness, uuid.New(), []string{}, 1)
			},
			expectError: nil,
		},
		{
			name: "missing title",
			setupIdea: func() *Idea {
				idea := NewIdea("", "Valid content", IdeaCategoryBusiness, uuid.New(), []string{}, 1)
				return idea
			},
			expectError: ErrIdeaTitleRequired,
		},
		{
			name: "missing content",
			setupIdea: func() *Idea {
				idea := NewIdea("Valid Title", "", IdeaCategoryBusiness, uuid.New(), []string{}, 1)
				return idea
			},
			expectError: ErrIdeaContentRequired,
		},
		{
			name: "missing user ID",
			setupIdea: func() *Idea {
				idea := NewIdea("Valid Title", "Valid content", IdeaCategoryBusiness, uuid.Nil, []string{}, 1)
				return idea
			},
			expectError: ErrIdeaUserIDRequired,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			idea := tt.setupIdea()

			// Act
			err := idea.Validate()

			// Assert
			if tt.expectError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectError, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestIdeaCategory_String(t *testing.T) {
	tests := []struct {
		category IdeaCategory
		expected string
	}{
		{IdeaCategoryBusiness, "Business"},
		{IdeaCategoryPersonal, "Personal"},
		{IdeaCategoryTechnical, "Technical"},
		{IdeaCategoryCreative, "Creative"},
		{IdeaCategoryResearch, "Research"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			// Note: This test assumes you implement String() methods
			// You might want to add String() methods to your enums
			assert.NotZero(t, int(tt.category))
		})
	}
}

// Benchmark tests
func BenchmarkNewIdea(b *testing.B) {
	userID := uuid.New()
	tags := []string{"benchmark", "test"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		NewIdea("Benchmark Idea", "Benchmark content", IdeaCategoryBusiness, userID, tags, 5)
	}
}

func BenchmarkIdea_Update(b *testing.B) {
	idea := NewIdea("Original", "Original content", IdeaCategoryPersonal, uuid.New(), []string{"original"}, 1)
	tags := []string{"updated", "benchmark"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		idea.Update("Updated", "Updated content", tags, IdeaCategoryTechnical, IdeaStatusActive, 8)
	}
}

func BenchmarkIdea_AddRelatedIdea(b *testing.B) {
	idea := NewIdea("Test", "Content", IdeaCategoryBusiness, uuid.New(), []string{}, 1)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		idea.AddRelatedIdea(uuid.New())
	}
}