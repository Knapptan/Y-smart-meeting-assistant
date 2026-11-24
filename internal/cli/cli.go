// Package cli реализует CLI-интерфейс приложения.
// Отвечает только за разбор команд и форматированный вывод.
// Вся бизнес-логика — в MeetingService.
package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/Knapptan/Y-smart-meeting-assistant/internal/domain"
	"github.com/Knapptan/Y-smart-meeting-assistant/internal/service"
)

// CLI — обработчик команд.
type CLI struct {
	svc    *service.MeetingService
	userID string
	stdout io.Writer
	stderr io.Writer
}

// New создаёт CLI.
func New(svc *service.MeetingService, userID string, stdout, stderr io.Writer) *CLI {
	return &CLI{svc: svc, userID: userID, stdout: stdout, stderr: stderr}
}

// Run разбирает аргументы и выполняет команду.
// args — os.Args[1:].
func (c *CLI) Run(ctx context.Context, args []string) error {
	if len(args) == 0 {
		c.printUsage()
		return nil
	}
	cmd := args[0]
	rest := args[1:]

	switch cmd {
	case "start":
		return c.cmdStart(ctx)
	case "load":
		return c.cmdLoad(ctx, rest)
	case "list":
		return c.cmdList(ctx)
	case "status":
		return c.cmdStatus(ctx, rest)
	case "get":
		return c.cmdGet(ctx, rest)
	case "find":
		return c.cmdFind(ctx, rest)
	case "chat":
		return c.cmdChat(ctx, rest)
	case "retry":
		return c.cmdRetry(ctx, rest)
	case "help", "-h", "--help":
		c.printUsage()
		return nil
	default:
		return fmt.Errorf("unknown command %q (use 'help')", cmd)
	}
}

func (c *CLI) printUsage() {
	fmt.Fprintln(c.stdout, `Умный помощник для конспектирования встреч — CLI

Команды:
  start                            регистрация пользователя
  load <path>                      загрузить файл и создать задачу обработки
  list                             список встреч пользователя
  status <id>                      статус обработки встречи
  get <id>                         получить транскрипцию и выжимку
  find <keyword>                   поиск по встречам
  chat <id> <вопрос>               задать вопрос по материалам встречи
  retry <id>                       перезапустить обработку встречи со статусом failed

Пользователь задаётся через -user <id> или USER_ID.
Фоновая обработка запускается командой: assistant worker`)
}

// checkUser проверяет, что user_id задан.
func (c *CLI) checkUser() error {
	if strings.TrimSpace(c.userID) == "" {
		return errors.New("user_id is required: use -user <id> or set USER_ID")
	}
	return nil
}

func (c *CLI) cmdStart(ctx context.Context) error {
	if err := c.checkUser(); err != nil {
		return err
	}

	if _, err := c.svc.Register(ctx, c.userID); err != nil {
		return fmt.Errorf("register: %w", err)
	}
	fmt.Fprintf(c.stdout, "Пользователь %q зарегистрирован\n", c.userID)
	return nil
}

func (c *CLI) cmdLoad(ctx context.Context, args []string) error {
	if err := c.checkUser(); err != nil {
		return err
	}
	if len(args) < 1 {
		return errors.New("usage: load <path>")
	}
	path := args[0]
	m, task, err := c.svc.Create(ctx, service.CreateInput{
		UserExternalID: c.userID,
		FilePath:       path,
	})
	if err != nil {
		return fmt.Errorf("load: %w", err)
	}
	fmt.Fprintf(c.stdout, "Встреча создана: id=%d, задача=%d, статус=%s\n", m.ID, task.ID, m.Status)
	fmt.Fprintf(c.stdout, "Проверьте статус: assistant status %d\n", m.ID)
	return nil
}

func (c *CLI) cmdList(ctx context.Context) error {
	if err := c.checkUser(); err != nil {
		return err
	}
	meetings, err := c.svc.List(ctx, c.userID)
	if err != nil {
		return fmt.Errorf("list: %w", err)
	}
	if len(meetings) == 0 {
		fmt.Fprintln(c.stdout, "Встреч пока нет")
		return nil
	}
	tw := tabwriter.NewWriter(c.stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "ID\tСТАТУС\tСОЗДАНА\tЗАГОЛОВОК")
	for _, m := range meetings {
		fmt.Fprintf(tw, "%d\t%s\t%s\t%s\n", m.ID, m.Status, m.CreatedAt.Format("2006-01-02 15:04"), m.Title)
	}
	return tw.Flush()
}

func (c *CLI) cmdStatus(ctx context.Context, args []string) error {
	if err := c.checkUser(); err != nil {
		return err
	}
	if len(args) < 1 {
		return errors.New("usage: status <id>")
	}
	id, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return fmt.Errorf("invalid id: %w", err)
	}
	m, err := c.svc.Status(ctx, c.userID, id)
	if err != nil {
		return fmt.Errorf("status: %w", err)
	}
	fmt.Fprintf(c.stdout, "Встреча %d\n", m.ID)
	fmt.Fprintf(c.stdout, "  статус:     %s\n", m.Status)
	fmt.Fprintf(c.stdout, "  создана:    %s\n", m.CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Fprintf(c.stdout, "  обновлена:  %s\n", m.UpdatedAt.Format("2006-01-02 15:04:05"))
	if m.Error != "" {
		fmt.Fprintf(c.stdout, "  ошибка:     %s\n", m.Error)
	}
	return nil
}

func (c *CLI) cmdGet(ctx context.Context, args []string) error {
	if err := c.checkUser(); err != nil {
		return err
	}
	if len(args) < 1 {
		return errors.New("usage: get <id>")
	}
	id, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return fmt.Errorf("invalid id: %w", err)
	}
	full, err := c.svc.Get(ctx, c.userID, id)
	if err != nil {
		return fmt.Errorf("get: %w", err)
	}
	fmt.Fprintf(c.stdout, "Встреча %d — %s\n", full.Meeting.ID, full.Meeting.Status)
	if full.Meeting.Error != "" {
		fmt.Fprintf(c.stdout, "Ошибка обработки: %s\n", full.Meeting.Error)
	}
	fmt.Fprintln(c.stdout, "\n--- Транскрипция ---")
	if full.Transcript == "" {
		fmt.Fprintln(c.stdout, "(нет)")
	} else {
		fmt.Fprintln(c.stdout, full.Transcript)
	}
	fmt.Fprintln(c.stdout, "\n--- Краткая выжимка ---")
	if full.Summary == "" {
		fmt.Fprintln(c.stdout, "(нет)")
	} else {
		fmt.Fprintln(c.stdout, full.Summary)
	}
	return nil
}

func (c *CLI) cmdFind(ctx context.Context, args []string) error {
	if err := c.checkUser(); err != nil {
		return err
	}
	if len(args) < 1 {
		return errors.New("usage: find <keyword>")
	}
	keyword := strings.Join(args, " ")
	meetings, err := c.svc.Find(ctx, c.userID, keyword)
	if err != nil {
		return fmt.Errorf("find: %w", err)
	}
	if len(meetings) == 0 {
		fmt.Fprintf(c.stdout, "Ничего не найдено по запросу %q\n", keyword)
		return nil
	}
	tw := tabwriter.NewWriter(c.stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "ID\tСТАТУС\tСОЗДАНА\tЗАГОЛОВОК")
	for _, m := range meetings {
		fmt.Fprintf(tw, "%d\t%s\t%s\t%s\n", m.ID, m.Status, m.CreatedAt.Format("2006-01-02 15:04"), m.Title)
	}
	return tw.Flush()
}

func (c *CLI) cmdChat(ctx context.Context, args []string) error {
	if err := c.checkUser(); err != nil {
		return err
	}
	if len(args) < 2 {
		return errors.New("usage: chat <id> <вопрос>")
	}
	id, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return fmt.Errorf("invalid id: %w", err)
	}
	question := strings.Join(args[1:], " ")
	answer, err := c.svc.Chat(ctx, c.userID, id, question)
	if err != nil {
		return fmt.Errorf("chat: %w", err)
	}
	fmt.Fprintf(c.stdout, "Вопрос: %s\n\nОтвет: %s\n", question, answer)
	return nil
}

func (c *CLI) cmdRetry(ctx context.Context, args []string) error {
	if err := c.checkUser(); err != nil {
		return err
	}
	if len(args) < 1 {
		return errors.New("usage: retry <id>")
	}
	id, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return fmt.Errorf("invalid id: %w", err)
	}
	task, err := c.svc.Retry(ctx, c.userID, id)
	if err != nil {
		if errors.Is(err, service.ErrRetryNotAllowed) {
			return fmt.Errorf("retry: встречу нельзя перезапустить (не в failed или попытки исчерпаны)")
		}
		return fmt.Errorf("retry: %w", err)
	}
	fmt.Fprintf(c.stdout, "Задача %d перезапущена (попытка %d)\n", task.ID, task.Attempts)
	return nil
}

// CheckUserExists — экспорт для main (проверка перед запуском).
func (c *CLI) CheckUserExists(ctx context.Context) (*domain.User, error) {
	return nil, nil // заглушка, не используется
}
