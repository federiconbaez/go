package entities

import (
	"time"

	"github.com/google/uuid"
)

// ProgressMilestone representa un hito en el progreso
type ProgressMilestone struct {
	ID          uuid.UUID
	Name        string
	Description string
	Completed   bool
	DueDate     time.Time
	CompletedAt *time.Time
}

// Progress representa el progreso de un proyecto
type Progress struct {
	ID                     uuid.UUID
	UserID                 uuid.UUID
	ProjectName            string
	Description            string
	CompletionPercentage   float32
	Milestones             []ProgressMilestone
	CreatedAt              time.Time
	UpdatedAt              time.Time
}

// NewProgress crea un nuevo registro de progreso
func NewProgress(userID uuid.UUID, projectName, description string) *Progress {
	now := time.Now()
	return &Progress{
		ID:                   uuid.New(),
		UserID:               userID,
		ProjectName:          projectName,
		Description:          description,
		CompletionPercentage: 0.0,
		Milestones:           make([]ProgressMilestone, 0),
		CreatedAt:            now,
		UpdatedAt:            now,
	}
}

// NewMilestone crea un nuevo hito
func NewMilestone(name, description string, dueDate time.Time) ProgressMilestone {
	return ProgressMilestone{
		ID:          uuid.New(),
		Name:        name,
		Description: description,
		Completed:   false,
		DueDate:     dueDate,
		CompletedAt: nil,
	}
}

// Update actualiza los campos modificables del progreso
func (p *Progress) Update(projectName, description string, completionPercentage float32, milestones []ProgressMilestone) error {
	if projectName != "" {
		p.ProjectName = projectName
	}
	if description != "" {
		p.Description = description
	}
	if completionPercentage >= 0 && completionPercentage <= 100 {
		p.CompletionPercentage = completionPercentage
	} else if completionPercentage != 0 {
		return ErrInvalidCompletionPercentage
	}
	if milestones != nil {
		p.Milestones = milestones
	}
	p.UpdatedAt = time.Now()
	return nil
}

// AddMilestone añade un nuevo hito
func (p *Progress) AddMilestone(milestone ProgressMilestone) {
	p.Milestones = append(p.Milestones, milestone)
	p.UpdatedAt = time.Now()
}

// CompleteMilestone marca un hito como completado
func (p *Progress) CompleteMilestone(milestoneID uuid.UUID) bool {
	for i := range p.Milestones {
		if p.Milestones[i].ID == milestoneID {
			p.Milestones[i].Completed = true
			now := time.Now()
			p.Milestones[i].CompletedAt = &now
			p.UpdatedAt = now
			p.recalculateCompletion()
			return true
		}
	}
	return false
}

// UncompleteMilestone marca un hito como no completado
func (p *Progress) UncompleteMilestone(milestoneID uuid.UUID) bool {
	for i := range p.Milestones {
		if p.Milestones[i].ID == milestoneID {
			p.Milestones[i].Completed = false
			p.Milestones[i].CompletedAt = nil
			p.UpdatedAt = time.Now()
			p.recalculateCompletion()
			return true
		}
	}
	return false
}

// RemoveMilestone elimina un hito
func (p *Progress) RemoveMilestone(milestoneID uuid.UUID) bool {
	for i, milestone := range p.Milestones {
		if milestone.ID == milestoneID {
			p.Milestones = append(p.Milestones[:i], p.Milestones[i+1:]...)
			p.UpdatedAt = time.Now()
			p.recalculateCompletion()
			return true
		}
	}
	return false
}

// recalculateCompletion recalcula el porcentaje de completación basado en los hitos
func (p *Progress) recalculateCompletion() {
	if len(p.Milestones) == 0 {
		return
	}
	
	completed := 0
	for _, milestone := range p.Milestones {
		if milestone.Completed {
			completed++
		}
	}
	
	p.CompletionPercentage = float32(completed) / float32(len(p.Milestones)) * 100
}

// GetCompletedMilestones obtiene los hitos completados
func (p *Progress) GetCompletedMilestones() []ProgressMilestone {
	var completed []ProgressMilestone
	for _, milestone := range p.Milestones {
		if milestone.Completed {
			completed = append(completed, milestone)
		}
	}
	return completed
}

// GetPendingMilestones obtiene los hitos pendientes
func (p *Progress) GetPendingMilestones() []ProgressMilestone {
	var pending []ProgressMilestone
	for _, milestone := range p.Milestones {
		if !milestone.Completed {
			pending = append(pending, milestone)
		}
	}
	return pending
}

// GetOverdueMilestones obtiene los hitos vencidos
func (p *Progress) GetOverdueMilestones() []ProgressMilestone {
	var overdue []ProgressMilestone
	now := time.Now()
	for _, milestone := range p.Milestones {
		if !milestone.Completed && milestone.DueDate.Before(now) {
			overdue = append(overdue, milestone)
		}
	}
	return overdue
}

// IsOwnedBy verifica si el progreso pertenece al usuario especificado
func (p *Progress) IsOwnedBy(userID uuid.UUID) bool {
	return p.UserID == userID
}

// Validate valida que el progreso tenga los campos requeridos
func (p *Progress) Validate() error {
	if p.ProjectName == "" {
		return ErrProgressProjectNameRequired
	}
	if p.UserID == uuid.Nil {
		return ErrProgressUserIDRequired
	}
	if p.CompletionPercentage < 0 || p.CompletionPercentage > 100 {
		return ErrInvalidCompletionPercentage
	}
	return nil
}