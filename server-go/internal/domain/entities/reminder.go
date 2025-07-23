package entities

import (
	"time"

	"github.com/google/uuid"
)

// ReminderType representa los tipos de recordatorios
type ReminderType int32

const (
	ReminderTypeUnspecified ReminderType = 0
	ReminderTypeTask        ReminderType = 1
	ReminderTypeMeeting     ReminderType = 2
	ReminderTypeDeadline    ReminderType = 3
	ReminderTypeEvent       ReminderType = 4
	ReminderTypeCall        ReminderType = 5
)

// ReminderStatus representa el estado de un recordatorio
type ReminderStatus int32

const (
	ReminderStatusUnspecified ReminderStatus = 0
	ReminderStatusPending     ReminderStatus = 1
	ReminderStatusActive      ReminderStatus = 2
	ReminderStatusCompleted   ReminderStatus = 3
	ReminderStatusCancelled   ReminderStatus = 4
	ReminderStatusOverdue     ReminderStatus = 5
)

// RecurrencePattern representa el patrón de recurrencia
type RecurrencePattern int32

const (
	RecurrencePatternUnspecified RecurrencePattern = 0
	RecurrencePatternDaily       RecurrencePattern = 1
	RecurrencePatternWeekly      RecurrencePattern = 2
	RecurrencePatternMonthly     RecurrencePattern = 3
	RecurrencePatternYearly      RecurrencePattern = 4
	RecurrencePatternCustom      RecurrencePattern = 5
)

// Reminder representa un recordatorio en el dominio
type Reminder struct {
	ID                    uuid.UUID
	Title                 string
	Description           string
	ScheduledTime         time.Time
	Type                  ReminderType
	Status                ReminderStatus
	Recurring             bool
	RecurrencePattern     RecurrencePattern
	CreatedAt             time.Time
	UpdatedAt             time.Time
	UserID                uuid.UUID
	NotificationChannels  []string
}

// NewReminder crea un nuevo recordatorio
func NewReminder(title, description string, scheduledTime time.Time, reminderType ReminderType, userID uuid.UUID, recurring bool, recurrencePattern RecurrencePattern, channels []string) *Reminder {
	now := time.Now()
	return &Reminder{
		ID:                   uuid.New(),
		Title:                title,
		Description:          description,
		ScheduledTime:        scheduledTime,
		Type:                 reminderType,
		Status:               ReminderStatusPending,
		Recurring:            recurring,
		RecurrencePattern:    recurrencePattern,
		CreatedAt:            now,
		UpdatedAt:            now,
		UserID:               userID,
		NotificationChannels: channels,
	}
}

// Update actualiza los campos modificables del recordatorio
func (r *Reminder) Update(title, description string, scheduledTime time.Time, reminderType ReminderType, status ReminderStatus, recurring bool, recurrencePattern RecurrencePattern) {
	if title != "" {
		r.Title = title
	}
	if description != "" {
		r.Description = description
	}
	if !scheduledTime.IsZero() {
		r.ScheduledTime = scheduledTime
	}
	if reminderType != ReminderTypeUnspecified {
		r.Type = reminderType
	}
	if status != ReminderStatusUnspecified {
		r.Status = status
	}
	r.Recurring = recurring
	if recurrencePattern != RecurrencePatternUnspecified {
		r.RecurrencePattern = recurrencePattern
	}
	r.UpdatedAt = time.Now()
}

// Complete marca el recordatorio como completado
func (r *Reminder) Complete() {
	r.Status = ReminderStatusCompleted
	r.UpdatedAt = time.Now()
}

// Cancel marca el recordatorio como cancelado
func (r *Reminder) Cancel() {
	r.Status = ReminderStatusCancelled
	r.UpdatedAt = time.Now()
}

// MarkAsOverdue marca el recordatorio como vencido
func (r *Reminder) MarkAsOverdue() {
	if r.Status == ReminderStatusPending || r.Status == ReminderStatusActive {
		r.Status = ReminderStatusOverdue
		r.UpdatedAt = time.Now()
	}
}

// IsOverdue verifica si el recordatorio está vencido
func (r *Reminder) IsOverdue() bool {
	return time.Now().After(r.ScheduledTime) && 
		   (r.Status == ReminderStatusPending || r.Status == ReminderStatusActive)
}

// IsOwnedBy verifica si el recordatorio pertenece al usuario especificado
func (r *Reminder) IsOwnedBy(userID uuid.UUID) bool {
	return r.UserID == userID
}

// Validate valida que el recordatorio tenga los campos requeridos
func (r *Reminder) Validate() error {
	if r.Title == "" {
		return ErrReminderTitleRequired
	}
	if r.UserID == uuid.Nil {
		return ErrReminderUserIDRequired
	}
	if r.ScheduledTime.IsZero() {
		return ErrReminderScheduledTimeRequired
	}
	return nil
}