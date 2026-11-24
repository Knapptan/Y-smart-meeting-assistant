// Package domain содержит доменные модели и интерфейсы приложения.
// Не зависит ни от БД, ни от конкретных внешних API.
package domain

import "time"

// Status — статус обработки встречи/задачи.
type Status string

const (
	StatusCreated     Status = "created"
	StatusProcessing  Status = "processing"
	StatusTranscribed Status = "transcribed"
	StatusSummarized  Status = "summarized"
	StatusCompleted   Status = "completed"
	StatusFailed      Status = "failed"
)

// User — пользователь системы.
type User struct {
	ID         int64
	ExternalID string
	CreatedAt  time.Time
}

// Meeting — загруженная встреча.
type Meeting struct {
	ID        int64
	UserID    int64
	Title     string
	Status    Status
	FilePath  string
	Error     string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Task — задача на обработку встречи.
type Task struct {
	ID        int64
	MeetingID int64
	Status    Status
	Attempts  int
	Error     string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Transcript — транскрипция встречи.
type Transcript struct {
	MeetingID int64
	Text      string
	CreatedAt time.Time
}

// Summary — краткая выжимка встречи.
type Summary struct {
	MeetingID int64
	Text      string
	CreatedAt time.Time
}

// MeetingFull — агрегат для команды get.
type MeetingFull struct {
	Meeting    Meeting
	Transcript string
	Summary    string
}
