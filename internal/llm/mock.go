// Package llm содержит реализации domain.LLMClient.
package llm

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Knapptan/Y-smart-meeting-assistant/internal/domain"
)

// MockClient — тестовая реализация LLMClient.
// Summarize возвращает заранее подготовленную выжимку из файла,
// Ask — детерминированный ответ на основе ключевых слов вопроса
// и переданного контекста.
type MockClient struct {
	summaryPath string
	delay       time.Duration
}

// NewMockClient создаёт mock-клиент.
// summaryPath — путь к файлу с заранее подготовленной выжимкой.
// delay — искусственная задержка; 0 — без задержки.
func NewMockClient(summaryPath string, delay time.Duration) *MockClient {
	return &MockClient{
		summaryPath: summaryPath,
		delay:       delay,
	}
}

// Проверка на этапе компиляции.
var _ domain.LLMClient = (*MockClient)(nil)

// Summarize возвращает заранее подготовленную выжимку из файла.
// Параметр transcript игнорируется.
func (c *MockClient) Summarize(ctx context.Context, transcript string) (string, error) {
	if c.delay > 0 {
		select {
		case <-time.After(c.delay):
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}

	data, err := os.ReadFile(c.summaryPath)
	if err != nil {
		return "", fmt.Errorf("mock llm: read summary %q: %w", c.summaryPath, err)
	}
	if len(data) == 0 {
		return "", fmt.Errorf("mock llm: summary %q is empty", c.summaryPath)
	}
	return string(data), nil
}

// Ask возвращает детерминированный ответ на вопрос.
// Логика: если в вопросе есть ключевые слова (бюджет, найм, дедлайн и т.п.) —
// отвечаем конкретным фактом из контекста; иначе — общая заглушка,
// в которой упоминается начало контекста.
//
// Это позволяет демонстрировать «работу LLM» без внешнего API.
func (c *MockClient) Ask(ctx context.Context, question, contextText string) (string, error) {
	if c.delay > 0 {
		select {
		case <-time.After(c.delay):
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}

	q := strings.ToLower(question)
	ctxLower := strings.ToLower(contextText)

	switch {
	case strings.Contains(q, "бюджет") || strings.Contains(q, "деньг") || strings.Contains(q, "маркетинг"):
		if i := strings.Index(ctxLower, "бюджет"); i >= 0 {
			return pickSentence(contextText, i), nil
		}
		return "В материалах встречи упоминается бюджет, но точная сумма не найдена.", nil

	case strings.Contains(q, "найм") || strings.Contains(q, "ваканс") || strings.Contains(q, "сотрудник"):
		if i := strings.Index(ctxLower, "найм"); i >= 0 {
			return pickSentence(contextText, i), nil
		}
		return "В материалах встречи обсуждается найм, но детали не найдены.", nil

	case strings.Contains(q, "дедлайн") || strings.Contains(q, "срок") || strings.Contains(q, "когда"):
		if i := strings.Index(ctxLower, "дедлайн"); i >= 0 {
			return pickSentence(contextText, i), nil
		}
		if i := strings.Index(ctxLower, "срок"); i >= 0 {
			return pickSentence(contextText, i), nil
		}
		return "Явных дедлайнов в материалах встречи не зафиксировано.", nil

	case strings.Contains(q, "запуск") || strings.Contains(q, "сервис") || strings.Contains(q, "релиз"):
		if i := strings.Index(ctxLower, "запуск"); i >= 0 {
			return pickSentence(contextText, i), nil
		}
		return "Информации о запуске в материалах встречи нет.", nil
	}

	// Общий ответ: цитируем первое предложение контекста.
	snippet := firstSentence(contextText)
	if snippet == "" {
		return "По материалам встречи ответить не могу: контекст пуст.", nil
	}
	return fmt.Sprintf("По материалам встречи: %s", snippet), nil
}

// pickSentence возвращает предложение, содержащее позицию i.
func pickSentence(text string, i int) string {
	// Найти начало предложения (после '.', '!', '?', '\n').
	start := 0
	for j := i - 1; j >= 0; j-- {
		if text[j] == '.' || text[j] == '!' || text[j] == '?' || text[j] == '\n' {
			start = j + 1
			break
		}
	}
	// Найти конец предложения.
	end := len(text)
	for j := i; j < len(text); j++ {
		if text[j] == '.' || text[j] == '!' || text[j] == '?' || text[j] == '\n' {
			end = j + 1
			break
		}
	}
	return strings.TrimSpace(text[start:end])
}

// firstSentence возвращает первое непустое предложение текста.
func firstSentence(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	for i, r := range text {
		if r == '.' || r == '!' || r == '?' || r == '\n' {
			return strings.TrimSpace(text[:i+1])
		}
	}
	return text
}
