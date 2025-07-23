package entities

import "errors"

// Domain errors for Ideas
var (
	ErrIdeaTitleRequired   = errors.New("idea title is required")
	ErrIdeaContentRequired = errors.New("idea content is required")
	ErrIdeaUserIDRequired  = errors.New("idea user ID is required")
	ErrIdeaNotFound        = errors.New("idea not found")
	ErrIdeaUnauthorized    = errors.New("unauthorized to access idea")
)

// Domain errors for Reminders
var (
	ErrReminderTitleRequired       = errors.New("reminder title is required")
	ErrReminderUserIDRequired      = errors.New("reminder user ID is required")
	ErrReminderScheduledTimeRequired = errors.New("reminder scheduled time is required")
	ErrReminderNotFound            = errors.New("reminder not found")
	ErrReminderUnauthorized        = errors.New("unauthorized to access reminder")
)

// Domain errors for Files
var (
	ErrFileNameRequired    = errors.New("file name is required")
	ErrFileUserIDRequired  = errors.New("file user ID is required")
	ErrFileNotFound        = errors.New("file not found")
	ErrFileUnauthorized    = errors.New("unauthorized to access file")
	ErrFileSizeExceeded    = errors.New("file size exceeded maximum allowed")
	ErrInvalidFileType     = errors.New("invalid file type")
)

// Domain errors for Progress
var (
	ErrProgressProjectNameRequired = errors.New("progress project name is required")
	ErrProgressUserIDRequired      = errors.New("progress user ID is required")
	ErrProgressNotFound            = errors.New("progress not found")
	ErrProgressUnauthorized        = errors.New("unauthorized to access progress")
	ErrInvalidCompletionPercentage = errors.New("completion percentage must be between 0 and 100")
)

// General domain errors
var (
	ErrInvalidUUID        = errors.New("invalid UUID format")
	ErrInvalidPagination  = errors.New("invalid pagination parameters")
	ErrInvalidSortField   = errors.New("invalid sort field")
)