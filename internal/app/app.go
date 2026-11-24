// Package app собирает все слои приложения и запускает нужную команду.
package app

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/Knapptan/Y-smart-meeting-assistant/internal/cli"
	"github.com/Knapptan/Y-smart-meeting-assistant/internal/config"
	"github.com/Knapptan/Y-smart-meeting-assistant/internal/llm"
	"github.com/Knapptan/Y-smart-meeting-assistant/internal/repository"
	"github.com/Knapptan/Y-smart-meeting-assistant/internal/service"
	"github.com/Knapptan/Y-smart-meeting-assistant/internal/speech"
)

// Run собирает зависимости и запускает команду.
// args — os.Args[1:].
func Run(ctx context.Context, cfg *config.Config, args []string) error {
	// 1. БД.
	db, err := repository.Open(cfg.DatabaseDSN)
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	defer db.Close()

	// 2. Миграции.
	if err := db.Migrate(cfg.MigrationsPath); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	slog.Info("migrations applied")

	// 3. Клиенты внешних API (mock).
	speechClient := speech.NewMockClient(cfg.MockTranscriptPath, 0)
	llmClient := llm.NewMockClient(cfg.MockSummaryPath, 0)

	// 4. Репозитории.
	userRepo := repository.NewUserRepo(db)
	meetingRepo := repository.NewMeetingRepo(db)
	taskRepo := repository.NewTaskRepo(db)
	transcriptRepo := repository.NewTranscriptRepo(db)
	summaryRepo := repository.NewSummaryRepo(db)
	searchRepo := repository.NewSearchRepo(db)

	// 5. Сервис встреч.
	meetingSvc := service.NewMeetingService(
		userRepo, meetingRepo, taskRepo,
		transcriptRepo, summaryRepo, searchRepo,
		llmClient, 3,
	)

	// 6. Если команда — worker, запускаем воркер-пул.
	if len(args) > 0 && args[0] == "worker" {
		processingSvc := service.NewProcessingService(
			service.ProcessingConfig{
				Workers:       cfg.MaxWorkers,
				PollInterval:  time.Second,
				SpeechTimeout: time.Duration(cfg.SpeechTimeoutSec) * time.Second,
				LLMTimeout:    time.Duration(cfg.LLMTimeoutSec) * time.Second,
				MaxAttempts:   3,
			},
			taskRepo, meetingRepo, transcriptRepo, summaryRepo,
			speechClient, llmClient,
		)
		slog.Info("starting worker pool", "workers", cfg.MaxWorkers)
		return processingSvc.Run(ctx)
	}

	// 7. Иначе — CLI.
	appCLI := cli.New(meetingSvc, cfg.UserID, os.Stdout, os.Stderr)
	return appCLI.Run(ctx, args)
}
