// Package speech содержит реализации domain.SpeechClient.
package speech

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/Knapptan/Y-smart-meeting-assistant/internal/domain"
)

// MockClient — тестовая реализация SpeechClient.
// Читает заранее подготовленную транскрипцию из файла и имитирует
// задержку распознавания. Используется, когда реальный Speech API недоступен.
type MockClient struct {
	transcriptPath string
	delay          time.Duration
}

// NewMockClient создаёт mock-клиент.
// transcriptPath — путь к файлу с заранее подготовленной транскрипцией.
// delay — искусственная задержка (имитация работы API); 0 — без задержки.
func NewMockClient(transcriptPath string, delay time.Duration) *MockClient {
	return &MockClient{
		transcriptPath: transcriptPath,
		delay:          delay,
	}
}

// Проверка на этапе компиляции, что MockClient реализует интерфейс.
var _ domain.SpeechClient = (*MockClient)(nil)

// Recognize «распознаёт» файл: возвращает содержимое заранее
// подготовленной транскрипции. Параметр filePath игнорируется —
// mock не читает сам аудиофайл.
func (c *MockClient) Recognize(ctx context.Context, filePath string) (string, error) {
	// Имитируем задержку внешнего API, уважая отмену контекста.
	if c.delay > 0 {
		select {
		case <-time.After(c.delay):
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}

	data, err := os.ReadFile(c.transcriptPath)
	if err != nil {
		return "", fmt.Errorf("mock speech: read transcript %q: %w", c.transcriptPath, err)
	}
	if len(data) == 0 {
		return "", fmt.Errorf("mock speech: transcript %q is empty", c.transcriptPath)
	}
	return string(data), nil
}
