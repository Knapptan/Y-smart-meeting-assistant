// Package service содержит бизнес-логику приложения.
package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/Knapptan/Y-smart-meeting-assistant/internal/domain"
)

// ProcessingConfig — настройки фоновой обработки.
type ProcessingConfig struct {
	Workers       int           // количество одновременных воркеров
	PollInterval  time.Duration // пауза между опросами пустой очереди
	SpeechTimeout time.Duration // таймаут speech-клиента на один файл
	LLMTimeout    time.Duration // таймаут LLM-клиента
	MaxAttempts   int           // максимальное количество попыток (для retry)
}

// ProcessingService — фоновая обработка встреч.
// Реализует worker pool: N воркеров забирают задачи из БД через SKIP LOCKED
// и обрабатывают их параллельно.
type ProcessingService struct {
	cfg         ProcessingConfig
	tasks       domain.TaskRepository
	meetings    domain.MeetingRepository
	transcripts domain.TranscriptRepository
	summaries   domain.SummaryRepository
	speech      domain.SpeechClient
	llm         domain.LLMClient
}

// NewProcessingService создаёт сервис обработки.
func NewProcessingService(
	cfg ProcessingConfig,
	tasks domain.TaskRepository,
	meetings domain.MeetingRepository,
	transcripts domain.TranscriptRepository,
	summaries domain.SummaryRepository,
	speech domain.SpeechClient,
	llm domain.LLMClient,
) *ProcessingService {
	if cfg.Workers <= 0 {
		cfg.Workers = 3
	}
	if cfg.PollInterval <= 0 {
		cfg.PollInterval = time.Second
	}
	if cfg.SpeechTimeout <= 0 {
		cfg.SpeechTimeout = 60 * time.Second
	}
	if cfg.LLMTimeout <= 0 {
		cfg.LLMTimeout = 60 * time.Second
	}
	if cfg.MaxAttempts <= 0 {
		cfg.MaxAttempts = 3
	}
	return &ProcessingService{
		cfg:         cfg,
		tasks:       tasks,
		meetings:    meetings,
		transcripts: transcripts,
		summaries:   summaries,
		speech:      speech,
		llm:         llm,
	}
}

// Run запускает worker pool и блокируется до отмены ctx.
// При отмене перестаёт брать новые задачи и дожидается завершения активных
// (с таймаутом 30 секунд), затем возвращает управление.
func (s *ProcessingService) Run(ctx context.Context) error {
	slog.Info("processing: starting worker pool", "workers", s.cfg.Workers)

	var wg sync.WaitGroup
	for i := 0; i < s.cfg.Workers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			s.worker(ctx, workerID)
		}(i)
	}

	// Ждём отмену контекста.
	<-ctx.Done()
	slog.Info("processing: shutdown requested, waiting for active workers")

	// Даём воркерам время завершить текущую задачу.
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	shutdownTimeout := 30 * time.Second
	select {
	case <-done:
		slog.Info("processing: all workers stopped gracefully")
		return nil
	case <-time.After(shutdownTimeout):
		slog.Warn("processing: shutdown timeout, some workers may have been killed")
		return errors.New("processing shutdown timeout")
	}
}

// worker — один воркер: опрашивает очередь, обрабатывает одну задачу за раз.
func (s *ProcessingService) worker(ctx context.Context, workerID int) {
	logger := slog.With("worker_id", workerID)
	for {
		select {
		case <-ctx.Done():
			logger.Info("worker: context cancelled, exiting")
			return
		default:
		}

		// Пытаемся забрать задачу.
		task, meeting, err := s.tasks.NextPending(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return
			}
			logger.Error("worker: NextPending failed", "error", err)
			sleepCtx(ctx, s.cfg.PollInterval)
			continue
		}
		if task == nil {
			// Очередь пуста — ждём.
			sleepCtx(ctx, s.cfg.PollInterval)
			continue
		}

		logger.Info("worker: task picked", "task_id", task.ID, "meeting_id", task.MeetingID)
		s.processOne(ctx, logger, task, meeting)
	}
}

// processOne выполняет полный пайплайн одной задачи:
// speech → transcripts → llm → summaries → статус completed.
func (s *ProcessingService) processOne(ctx context.Context, logger *slog.Logger, task *domain.Task, meeting *domain.Meeting) {
	// Обёртка: при ошибке — перевести в failed и сохранить текст ошибки.
	fail := func(stage string, err error) {
		logger.Error("worker: processing failed", "stage", stage, "error", err, "task_id", task.ID)
		errMsg := fmt.Sprintf("%s: %v", stage, err)

		// Задача — failed.
		if e := s.tasks.UpdateStatus(context.Background(), task.ID, domain.StatusFailed, errMsg); e != nil {
			logger.Error("worker: update task status failed", "error", e)
		}
		// Встреча — failed.
		if e := s.meetings.UpdateStatus(context.Background(), meeting.ID, domain.StatusFailed, errMsg); e != nil {
			logger.Error("worker: update meeting status failed", "error", e)
		}
	}

	// 1. Speech: распознавание.
	speechCtx, speechCancel := context.WithTimeout(ctx, s.cfg.SpeechTimeout)
	transcript, err := s.speech.Recognize(speechCtx, meeting.FilePath)
	speechCancel()
	if err != nil {
		fail("speech", err)
		return
	}

	// 2. Сохранение транскрипции + перевод встречи в transcribed.
	if err := s.transcripts.Save(ctx, &domain.Transcript{
		MeetingID: meeting.ID,
		Text:      transcript,
	}); err != nil {
		fail("save transcript", err)
		return
	}

	// 3. LLM: выжимка.
	llmCtx, llmCancel := context.WithTimeout(ctx, s.cfg.LLMTimeout)
	summary, err := s.llm.Summarize(llmCtx, transcript)
	llmCancel()
	if err != nil {
		fail("llm summarize", err)
		return
	}

	// 4. Сохранение выжимки + перевод встречи в summarized.
	if err := s.summaries.Save(ctx, &domain.Summary{
		MeetingID: meeting.ID,
		Text:      summary,
	}); err != nil {
		fail("save summary", err)
		return
	}

	// 5. Всё готово — встреча и задача в completed.
	if err := s.tasks.UpdateStatus(ctx, task.ID, domain.StatusCompleted, ""); err != nil {
		logger.Error("worker: update task status failed", "error", err)
	}
	if err := s.meetings.UpdateStatus(ctx, meeting.ID, domain.StatusCompleted, ""); err != nil {
		logger.Error("worker: update meeting status failed", "error", err)
	}
	logger.Info("worker: task completed", "task_id", task.ID, "meeting_id", meeting.ID)
}

// sleepCtx — пауза, уважающая отмену контекста.
func sleepCtx(ctx context.Context, d time.Duration) {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-t.C:
	case <-ctx.Done():
	}
}
