// Package main — точка входа CLI-утилиты «Умный помощник для конспектирования встреч».
package main

import (
	"fmt"
	"os"

	"github.com/Knapptan/Y-smart-meeting-assistant/internal/config"
	"github.com/Knapptan/Y-smart-meeting-assistant/internal/logger"
)

func main() {
	cfg := config.Load()
	logger.Init(cfg.LogLevel)

	// TODO: реализовать CLI.
	fmt.Fprintln(os.Stderr, "CLI не реализован. Запустите с -h для флагов.")
	os.Exit(0)
}
