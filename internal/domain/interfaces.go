package domain

import "context"

// SpeechClient — клиент распознавания речи.
// Реализации: speech.MockClient, (speech.YandexClient).
type SpeechClient interface {
	// Recognize отправляет файл на распознавание и возвращает текст.
	Recognize(ctx context.Context, filePath string) (string, error)
}

// LLMClient — клиент языковой модели.
// Реализации: llm.MockClient, (llm.GigaChatClient).
type LLMClient interface {
	// Summarize возвращает краткую выжимку по транскрипции.
	Summarize(ctx context.Context, transcript string) (string, error)
	// Ask отвечает на вопрос по контексту (материалам встречи).
	Ask(ctx context.Context, question, context string) (string, error)
}

// MeetingRepository — работа с встречами в БД.
type MeetingRepository interface {
	// CreateMeetingWithTask атомарно создаёт встречу и задачу (транзакция).
	CreateMeetingWithTask(ctx context.Context, m *Meeting) (*Task, error)
	// GetByID возвращает встречу по ID (без проверки пользователя).
	GetByID(ctx context.Context, id int64) (*Meeting, error)
	// GetByIDForUser возвращает встречу, если она принадлежит userID.
	GetByIDForUser(ctx context.Context, id, userID int64) (*Meeting, error)
	// ListByUser возвращает встречи пользователя.
	ListByUser(ctx context.Context, userID int64) ([]*Meeting, error)
	// UpdateStatus обновляет статус встречи и timestamp.
	UpdateStatus(ctx context.Context, id int64, s Status, errMsg string) error
}

// TaskRepository — работа с задачами обработки.
type TaskRepository interface {
	// NextPending атомарно забирает одну задачу в статусе created и переводит
	// её в processing (SKIP LOCKED). Возвращает nil, если очередь пуста.
	NextPending(ctx context.Context) (*Task, *Meeting, error)
	// UpdateStatus обновляет статус задачи.
	UpdateStatus(ctx context.Context, id int64, s Status, errMsg string) error
	// IncAttempts увеличивает счётчик попыток.
	IncAttempts(ctx context.Context, id int64) error
}

// TranscriptRepository — транскрипции.
type TranscriptRepository interface {
	// Save атомарно сохраняет транскрипцию и переводит встречу в статус transcribed.
	Save(ctx context.Context, t *Transcript) error
	GetByMeetingID(ctx context.Context, meetingID int64) (*Transcript, error)
}

// SummaryRepository — краткие выжимки.
type SummaryRepository interface {
	// Save атомарно сохраняет выжимку и переводит встречу в статус summarized.
	Save(ctx context.Context, s *Summary) error
	GetByMeetingID(ctx context.Context, meetingID int64) (*Summary, error)
}

// UserRepository — пользователи.
type UserRepository interface {
	// GetOrCreate возвращает пользователя по external_id,
	// создаёт, если не существует.
	GetOrCreate(ctx context.Context, externalID string) (*User, error)
}

// SearchRepository — поиск по встречам.
type SearchRepository interface {
	// FindByKeyword ищет встречи пользователя, содержащие keyword
	// в транскрипции или выжимке.
	FindByKeyword(ctx context.Context, userID int64, keyword string) ([]*Meeting, error)
}
