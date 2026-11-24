// Package service содержит бизнес-логику приложения.
package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/Knapptan/Y-smart-meeting-assistant/internal/domain"
)

// Общие ошибки сервисного слоя.
var (
	ErrNotFound        = errors.New("not found")
	ErrForbidden       = errors.New("forbidden")
	ErrInvalidInput    = errors.New("invalid input")
	ErrRetryNotAllowed = errors.New("retry not allowed: task is not failed or attempts exhausted")
)

// MeetingService — бизнес-логика работы со встречами.
// Не знает про CLI, HTTP, конкретные Speech/LLM API — только про интерфейсы.
type MeetingService struct {
	users       domain.UserRepository
	meetings    domain.MeetingRepository
	tasks       domain.TaskRepository
	transcripts domain.TranscriptRepository
	summaries   domain.SummaryRepository
	search      domain.SearchRepository
	llm         domain.LLMClient
	maxAttempts int
}

// NewMeetingService создаёт сервис.
func NewMeetingService(
	users domain.UserRepository,
	meetings domain.MeetingRepository,
	tasks domain.TaskRepository,
	transcripts domain.TranscriptRepository,
	summaries domain.SummaryRepository,
	search domain.SearchRepository,
	llm domain.LLMClient,
	maxAttempts int,
) *MeetingService {
	if maxAttempts <= 0 {
		maxAttempts = 3
	}
	return &MeetingService{
		users:       users,
		meetings:    meetings,
		tasks:       tasks,
		transcripts: transcripts,
		summaries:   summaries,
		search:      search,
		llm:         llm,
		maxAttempts: maxAttempts,
	}
}

// CreateInput — параметры создания встречи.
type CreateInput struct {
	UserExternalID string
	Title          string
	FilePath       string
}

// Create создаёт встречу и задачу обработки в одной транзакции.
// Возвращает встречу и задачу для отображения пользователю.
func (s *MeetingService) Create(ctx context.Context, in CreateInput) (*domain.Meeting, *domain.Task, error) {
	if in.UserExternalID == "" {
		return nil, nil, fmt.Errorf("%w: user_id is required", ErrInvalidInput)
	}
	if in.FilePath == "" {
		return nil, nil, fmt.Errorf("%w: file path is required", ErrInvalidInput)
	}
	if in.Title == "" {
		in.Title = in.FilePath
	}

	user, err := s.users.GetOrCreate(ctx, in.UserExternalID)
	if err != nil {
		return nil, nil, fmt.Errorf("get or create user: %w", err)
	}

	m := &domain.Meeting{
		UserID:   user.ID,
		Title:    in.Title,
		Status:   domain.StatusCreated,
		FilePath: in.FilePath,
	}
	task, err := s.meetings.CreateMeetingWithTask(ctx, m)
	if err != nil {
		return nil, nil, fmt.Errorf("create meeting with task: %w", err)
	}
	slog.Info("meeting: created", "meeting_id", m.ID, "task_id", task.ID, "user_id", user.ID)
	return m, task, nil
}

// List возвращает встречи пользователя.
func (s *MeetingService) List(ctx context.Context, userExternalID string) ([]*domain.Meeting, error) {
	user, err := s.users.GetByExternalID(ctx, userExternalID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrNotFound
	}
	return s.meetings.ListByUser(ctx, user.ID)
}

// Get возвращает встречу с транскрипцией и выжимкой.
// Пользователь получает только свою встречу: проверка user_id обязательна.
func (s *MeetingService) Get(ctx context.Context, userExternalID string, meetingID int64) (*domain.MeetingFull, error) {
	user, err := s.users.GetByExternalID(ctx, userExternalID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrNotFound
	}

	m, err := s.meetings.GetByIDForUser(ctx, meetingID, user.ID)
	if err != nil {
		return nil, err
	}
	if m == nil {
		return nil, ErrNotFound
	}

	full := &domain.MeetingFull{Meeting: *m}
	if t, err := s.transcripts.GetByMeetingID(ctx, meetingID); err == nil && t != nil {
		full.Transcript = t.Text
	}
	if sm, err := s.summaries.GetByMeetingID(ctx, meetingID); err == nil && sm != nil {
		full.Summary = sm.Text
	}
	return full, nil
}

// Status возвращает статус встречи.
func (s *MeetingService) Status(ctx context.Context, userExternalID string, meetingID int64) (*domain.Meeting, error) {
	user, err := s.users.GetByExternalID(ctx, userExternalID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrNotFound
	}
	m, err := s.meetings.GetByIDForUser(ctx, meetingID, user.ID)
	if err != nil {
		return nil, err
	}
	if m == nil {
		return nil, ErrNotFound
	}
	return m, nil
}

// Find ищет встречи пользователя по ключевому слову в транскрипциях и выжимках.
func (s *MeetingService) Find(ctx context.Context, userExternalID, keyword string) ([]*domain.Meeting, error) {
	if strings.TrimSpace(keyword) == "" {
		return nil, fmt.Errorf("%w: keyword is required", ErrInvalidInput)
	}
	user, err := s.users.GetByExternalID(ctx, userExternalID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrNotFound
	}
	return s.search.FindByKeyword(ctx, user.ID, keyword)
}

// Chat отправляет вопрос LLM по материалам конкретной встречи.
// Пользователь получает ответ только по своей встрече.
func (s *MeetingService) Chat(ctx context.Context, userExternalID string, meetingID int64, question string) (string, error) {
	if strings.TrimSpace(question) == "" {
		return "", fmt.Errorf("%w: question is required", ErrInvalidInput)
	}

	user, err := s.users.GetByExternalID(ctx, userExternalID)
	if err != nil {
		return "", err
	}
	if user == nil {
		return "", ErrNotFound
	}

	m, err := s.meetings.GetByIDForUser(ctx, meetingID, user.ID)
	if err != nil {
		return "", err
	}
	if m == nil {
		return "", ErrNotFound
	}

	// Контекст для вопроса — транскрипция, а если её нет — выжимка.
	contextText := ""
	if t, err := s.transcripts.GetByMeetingID(ctx, meetingID); err == nil && t != nil {
		contextText = t.Text
	}
	if contextText == "" {
		if sm, err := s.summaries.GetByMeetingID(ctx, meetingID); err == nil && sm != nil {
			contextText = sm.Text
		}
	}
	if contextText == "" {
		return "", fmt.Errorf("%w: meeting has no transcript or summary yet", ErrInvalidInput)
	}

	return s.llm.Ask(ctx, question, contextText)
}

// Retry перезапускает обработку встречи со статусом failed.
// Разрешено, если задача в failed и attempts < maxAttempts.
func (s *MeetingService) Retry(ctx context.Context, userExternalID string, meetingID int64) (*domain.Task, error) {
	user, err := s.users.GetByExternalID(ctx, userExternalID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrNotFound
	}
	m, err := s.meetings.GetByIDForUser(ctx, meetingID, user.ID)
	if err != nil {
		return nil, err
	}
	if m == nil {
		return nil, ErrNotFound
	}
	if m.Status != domain.StatusFailed {
		return nil, ErrRetryNotAllowed
	}

	task, err := s.tasks.Retry(ctx, meetingID, s.maxAttempts)
	if err != nil {
		return nil, err
	}
	if task == nil {
		return nil, ErrRetryNotAllowed
	}
	if err := s.meetings.UpdateStatus(ctx, meetingID, domain.StatusCreated, ""); err != nil {
		return nil, err
	}
	slog.Info("meeting: retry scheduled", "meeting_id", meetingID, "task_id", task.ID, "attempts", task.Attempts)
	return task, nil
}
