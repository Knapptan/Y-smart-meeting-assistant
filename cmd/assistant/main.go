// Package main — точка входа CLI-утилиты «Умный помощник для конспектирования встреч».
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/Knapptan/Y-smart-meeting-assistant/internal/app"
	"github.com/Knapptan/Y-smart-meeting-assistant/internal/config"
	"github.com/Knapptan/Y-smart-meeting-assistant/internal/logger"
)

func main() {
	cfg := config.Load()
	logger.Init(cfg.LogLevel)

	if cfg.DatabaseDSN == "" {
		fmt.Fprintln(os.Stderr, "DB_DSN не задан (например: -d 'postgres://user:pass@localhost:5432/db?sslmode=disable')")
		os.Exit(2)
	}

	// Контекст, отменяемый по SIGTERM/SIGINT/SIGQUIT.
	ctx, stop := signal.NotifyContext(context.Background(),
		syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)
	defer stop()

	if err := app.Run(ctx, cfg, os.Args[1:]); err != nil {
		slog.Error("app failed", "error", err)
		os.Exit(1)
	}
}
