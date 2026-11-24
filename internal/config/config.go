// Package config парсит конфигурацию из флагов и переменных окружения.
package config

import (
	"flag"
	"os"
)

// Config — настройки приложения.
type Config struct {
	DatabaseDSN        string
	SpeechProvider     string // "mock"
	LLMProvider        string // "mock"
	MaxWorkers         int
	SpeechTimeoutSec   int
	LLMTimeoutSec      int
	MockTranscriptPath string
	MockSummaryPath    string
	LogLevel           string
}

// Load парсит флаги и переменные окружения.
// Приоритет: env > flag > default.
func Load() *Config {
	dbDSN := flag.String("d", "", "PostgreSQL DSN (or DB_DSN env)")
	speechProvider := flag.String("speech", "mock", "speech provider: mock")
	llmProvider := flag.String("llm", "mock", "llm provider: mock")
	maxWorkers := flag.Int("workers", 3, "max concurrent workers")
	speechTimeout := flag.Int("speech-timeout", 60, "speech timeout, seconds")
	llmTimeout := flag.Int("llm-timeout", 60, "llm timeout, seconds")
	mockTranscript := flag.String("mock-transcript", "testdata/transcript_ru.txt", "path to mock transcript")
	mockSummary := flag.String("mock-summary", "testdata/summary_ru.txt", "path to mock summary")
	logLevel := flag.String("log-level", "info", "log level: debug|info|warn|error")
	flag.Parse()

	return &Config{
		DatabaseDSN:        pick("DB_DSN", *dbDSN),
		SpeechProvider:     pick("SPEECH_PROVIDER", *speechProvider),
		LLMProvider:        pick("LLM_PROVIDER", *llmProvider),
		MaxWorkers:         *maxWorkers,
		SpeechTimeoutSec:   *speechTimeout,
		LLMTimeoutSec:      *llmTimeout,
		MockTranscriptPath: *mockTranscript,
		MockSummaryPath:    *mockSummary,
		LogLevel:           pick("LOG_LEVEL", *logLevel),
	}
}

func pick(envKey, def string) string {
	if v := os.Getenv(envKey); v != "" {
		return v
	}
	return def
}
