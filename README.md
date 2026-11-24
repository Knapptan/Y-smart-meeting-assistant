Система представляет собой backend-сервис для загрузки, обработки, хранения и поиска материалов встреч.

Пользователь может взаимодействовать с системой через один из доступных интерфейсов:
CLI-утилиту;

## Cтруктура

```tree
smart-meeting-assistant/
├── cmd/
│   └── assistant/
│       └── main.go              # точка входа, CLI
├── internal/
│   ├── cli/                     # CLI-адаптер: парсинг команд, вывод
│   │   ├── cli.go
│   │   └── commands.go
│   ├── app/                     # сборка зависимостей, запуск
│   │   └── app.go
│   ├── config/                  # флаги + env
│   │   └── config.go
│   ├── domain/                  # модели и интерфейсы (ядро)
│   │   ├── models.go            # User, Meeting, Task, Transcript, Summary
│   │   ├── status.go            # статусы
│   │   └── interfaces.go        # SpeechClient, LLMClient, Repository
│   ├── service/                 # бизнес-логика
│   │   ├── service.go
│   │   ├── meeting.go
│   │   └── processing.go        # воркер-пул
│   ├── repository/              # PostgreSQL
│   │   ├── postgres.go
│   │   ├── user_repo.go
│   │   ├── meeting_repo.go
│   │   └── task_repo.go
│   ├── speech/                  # speech-клиенты
│   │   └── mock.go
│   ├── llm/                     # LLM-клиенты
│   │   └── mock.go
│   └── logger/                  # структурированное логирование
│       └── logger.go
├── migrations/
│   ├── 000001_init.up.sql
│   └── 000001_init.down.sql
├── testdata/                    # тестовые транскрипции, аудио
│   ├── transcript_ru.txt
│   └── summary_ru.txt
├── go.mod
└── README.md
```

## Запуск end-to-end

Терминал 1: PostgreSQL

```bash
docker run --name sma-pg -e POSTGRES_PASSWORD=postgres -e POSTGRES_DB=sma -p 5432:5432 -d postgres
```

Терминал 2: воркер

```bash
go run ./cmd/assistant \
  -d 'postgres://postgres:postgres@localhost:5432/sma?sslmode=disable' \
  -user alice \
  worker
```

Логи:

```text
INFO migrations applied
INFO starting worker pool workers=3
INFO processing: starting worker pool workers=3
```

Терминал 3: CLI

```bash
export DB='postgres://postgres:postgres@localhost:5432/sma?sslmode=disable'
export USER_ID=alice
BIN='go run ./cmd/assistant'
```

# 1. Регистрация

go run ./cmd/assistant -d "$DB" -user alice start

# → Пользователь "alice" зарегистрирован

# 2. Загрузка

go run ./cmd/assistant -d "$DB" -user alice load testdata/transcript_ru.txt

# → Встреча создана: id=1, задача=1, статус=created

# → Проверьте статус: assistant status 1

# 3. Через секунду — статус (воркер в терминале 2 уже обработал)

go run ./cmd/assistant -d "$DB" -user alice status 1

# → статус: completed

# 4. Транскрипция + выжимка

go run ./cmd/assistant -d "$DB" -user alice get 1

# 5. Поиск

go run ./cmd/assistant -d "$DB" -user alice find бюджет

# 6. Вопрос по встрече

go run ./cmd/assistant -d "$DB" -user alice chat 1 "Какой бюджет на маркетинг?"

# → Вопрос: Какой бюджет на маркетинг?

# Ответ: Второй вопрос — бюджет на маркетинг. Утвердили сумму в один миллион рублей.

# 7. Проверка изоляции: другой пользователь не видит встречи alice

go run ./cmd/assistant -d "$DB" -user bob get 1

# → get: not found
