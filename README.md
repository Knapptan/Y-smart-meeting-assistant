Система представляет собой backend-сервис для загрузки, обработки, хранения и поиска материалов встреч.

Пользователь может взаимодействовать с системой через один из доступных интерфейсов:
CLI-утилиту;

Cтруктура

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
