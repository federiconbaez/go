package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	https://github.com/federiconbaez/gogrpc-go-android/server-go/internal/domain/entities"
	https://github.com/federiconbaez/gogrpc-go-android/server-go/internal/domain/ports"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lib/pq"
)

type ideaRepository struct {
	db *pgxpool.Pool
}

// NewIdeaRepository crea una nueva instancia del repositorio de ideas
func NewIdeaRepository(db *pgxpool.Pool) ports.IdeaRepository {
	return &ideaRepository{db: db}
}

// Create crea una nueva idea en la base de datos
func (r *ideaRepository) Create(ctx context.Context, idea *entities.Idea) error {
	query := `
		INSERT INTO ideas (id, title, content, tags, category, status, created_at, updated_at, user_id, related_ideas, priority)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`
	
	relatedIdeaStrings := make([]string, len(idea.RelatedIdeas))
	for i, id := range idea.RelatedIdeas {
		relatedIdeaStrings[i] = id.String()
	}

	_, err := r.db.Exec(ctx, query,
		idea.ID,
		idea.Title,
		idea.Content,
		pq.Array(idea.Tags),
		int(idea.Category),
		int(idea.Status),
		idea.CreatedAt,
		idea.UpdatedAt,
		idea.UserID,
		pq.Array(relatedIdeaStrings),
		idea.Priority,
	)

	if err != nil {
		return fmt.Errorf("failed to create idea: %w", err)
	}

	return nil
}

// GetByID obtiene una idea por su ID
func (r *ideaRepository) GetByID(ctx context.Context, id uuid.UUID) (*entities.Idea, error) {
	query := `
		SELECT id, title, content, tags, category, status, created_at, updated_at, user_id, related_ideas, priority
		FROM ideas
		WHERE id = $1
	`

	var idea entities.Idea
	var tags pq.StringArray
	var relatedIdeas pq.StringArray
	var category, status int

	err := r.db.QueryRow(ctx, query, id).Scan(
		&idea.ID,
		&idea.Title,
		&idea.Content,
		&tags,
		&category,
		&status,
		&idea.CreatedAt,
		&idea.UpdatedAt,
		&idea.UserID,
		&relatedIdeas,
		&idea.Priority,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, entities.ErrIdeaNotFound
		}
		return nil, fmt.Errorf("failed to get idea: %w", err)
	}

	idea.Tags = []string(tags)
	idea.Category = entities.IdeaCategory(category)
	idea.Status = entities.IdeaStatus(status)

	// Convertir related ideas de strings a UUIDs
	idea.RelatedIdeas = make([]uuid.UUID, len(relatedIdeas))
	for i, idStr := range relatedIdeas {
		relatedID, err := uuid.Parse(idStr)
		if err != nil {
			continue // Skip invalid UUIDs
		}
		idea.RelatedIdeas[i] = relatedID
	}

	return &idea, nil
}

// GetByUserID obtiene las ideas de un usuario con filtros
func (r *ideaRepository) GetByUserID(ctx context.Context, userID uuid.UUID, filters ports.IdeaFilters) ([]*entities.Idea, int, error) {
	// Construir query base
	baseQuery := `FROM ideas WHERE user_id = $1`
	countQuery := `SELECT COUNT(*) ` + baseQuery
	selectQuery := `
		SELECT id, title, content, tags, category, status, created_at, updated_at, user_id, related_ideas, priority
	` + baseQuery

	args := []interface{}{userID}
	argIndex := 2

	// Aplicar filtros
	if filters.Category != entities.IdeaCategoryUnspecified {
		baseQuery += fmt.Sprintf(" AND category = $%d", argIndex)
		selectQuery = strings.Replace(selectQuery, baseQuery[:len(baseQuery)-len(fmt.Sprintf(" AND category = $%d", argIndex))], baseQuery, 1)
		countQuery = strings.Replace(countQuery, baseQuery[:len(baseQuery)-len(fmt.Sprintf(" AND category = $%d", argIndex))], baseQuery, 1)
		args = append(args, int(filters.Category))
		argIndex++
	}

	if filters.Status != entities.IdeaStatusUnspecified {
		filter := fmt.Sprintf(" AND status = $%d", argIndex)
		baseQuery += filter
		selectQuery += filter
		countQuery += filter
		args = append(args, int(filters.Status))
		argIndex++
	}

	if len(filters.Tags) > 0 {
		filter := fmt.Sprintf(" AND tags && $%d", argIndex)
		baseQuery += filter
		selectQuery += filter
		countQuery += filter
		args = append(args, pq.Array(filters.Tags))
		argIndex++
	}

	// Obtener conteo total
	var totalCount int
	err := r.db.QueryRow(ctx, countQuery, args...).Scan(&totalCount)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count ideas: %w", err)
	}

	// Aplicar ordenamiento y paginación
	orderBy := "created_at"
	if filters.SortBy != "" {
		orderBy = filters.SortBy
	}
	
	direction := "DESC"
	if !filters.SortDesc {
		direction = "ASC"
	}

	selectQuery += fmt.Sprintf(" ORDER BY %s %s", orderBy, direction)

	// Paginación
	if filters.PageSize > 0 {
		offset := (filters.Page - 1) * filters.PageSize
		selectQuery += fmt.Sprintf(" LIMIT %d OFFSET %d", filters.PageSize, offset)
	}

	// Ejecutar query principal
	rows, err := r.db.Query(ctx, selectQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query ideas: %w", err)
	}
	defer rows.Close()

	var ideas []*entities.Idea
	for rows.Next() {
		var idea entities.Idea
		var tags pq.StringArray
		var relatedIdeas pq.StringArray
		var category, status int

		err := rows.Scan(
			&idea.ID,
			&idea.Title,
			&idea.Content,
			&tags,
			&category,
			&status,
			&idea.CreatedAt,
			&idea.UpdatedAt,
			&idea.UserID,
			&relatedIdeas,
			&idea.Priority,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan idea: %w", err)
		}

		idea.Tags = []string(tags)
		idea.Category = entities.IdeaCategory(category)
		idea.Status = entities.IdeaStatus(status)

		// Convertir related ideas
		idea.RelatedIdeas = make([]uuid.UUID, len(relatedIdeas))
		for i, idStr := range relatedIdeas {
			if relatedID, err := uuid.Parse(idStr); err == nil {
				idea.RelatedIdeas[i] = relatedID
			}
		}

		ideas = append(ideas, &idea)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating ideas: %w", err)
	}

	return ideas, totalCount, nil
}

// Update actualiza una idea existente
func (r *ideaRepository) Update(ctx context.Context, idea *entities.Idea) error {
	query := `
		UPDATE ideas 
		SET title = $2, content = $3, tags = $4, category = $5, status = $6, 
		    updated_at = $7, related_ideas = $8, priority = $9
		WHERE id = $1
	`

	relatedIdeaStrings := make([]string, len(idea.RelatedIdeas))
	for i, id := range idea.RelatedIdeas {
		relatedIdeaStrings[i] = id.String()
	}

	result, err := r.db.Exec(ctx, query,
		idea.ID,
		idea.Title,
		idea.Content,
		pq.Array(idea.Tags),
		int(idea.Category),
		int(idea.Status),
		idea.UpdatedAt,
		pq.Array(relatedIdeaStrings),
		idea.Priority,
	)

	if err != nil {
		return fmt.Errorf("failed to update idea: %w", err)
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return entities.ErrIdeaNotFound
	}

	return nil
}

// Delete elimina una idea
func (r *ideaRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM ideas WHERE id = $1`

	result, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete idea: %w", err)
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return entities.ErrIdeaNotFound
	}

	return nil
}