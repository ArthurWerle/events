# AGENTS.md

Guidance for AI agents and contributors working in this repository.

## Overview

`events` is a PostgreSQL-backed event/job queue service written in Go. Producers
enqueue jobs (a `job_type`, a string `payload`, and a `callback_url`). A
background processor delivers each job by issuing an HTTP **GET** to its
`callback_url`, retrying with exponential backoff, and recording every attempt.

It is both a **standalone HTTP service** (`cmd/server`) and a set of **reusable
Go packages** (notably `client` for producers).

## Tech stack

- **Go 1.25** (module name: `events`, see `go.mod`)
- **chi/v5** — HTTP router (`github.com/go-chi/chi/v5`)
- **pgx/v5** — PostgreSQL driver / pool (`github.com/jackc/pgx/v5`)
- **golang-migrate/v4** — schema migrations, run on startup
- **godotenv** — loads `.env` automatically
- **PostgreSQL** — backing store (docker-compose provided for local dev)

## Project layout

```
.
├── api/rest/        # REST handlers; MountRoutes(r, queueService, executionRepo)
├── client/          # Go HTTP client for producers: New(baseURL), (*Client).Enqueue
├── cmd/server/      # Service entrypoint (main.go) — listens on :3000
├── db/              # Pool + migration runner (db_service.go)
│   └── migrations/  # golang-migrate SQL files (000001..000004)
├── model/           # Domain types: Event, EventExecution, Status
├── repository/      # Data access: EventRepository, ExecutionRepository
├── service/         # Business logic: QueueService, ProcessorService
├── types/           # Queue interface
├── templates/       # index.html dashboard
└── docker-compose.yml
```

## Architecture / data flow

1. **Enqueue** — `POST /api/events` → `service.QueueService.Enqueue` →
   `EventRepository.Create` inserts a row in `events` with status `pending`.
2. **Poll** — `service.ProcessorService.Consume()` loops forever
   (`cmd/server/main.go` runs it in a goroutine). Each iteration calls
   `EventRepository.GetProcessable` (row-locked with `FOR UPDATE SKIP LOCKED`)
   and marks the job `processing`.
3. **Deliver** — `callCallback` issues
   `GET <callback_url>?job_type=<type>&payload=<payload>`.
4. **Record** — every attempt is written to `event_executions` via
   `ExecutionRepository.Create` (status code, error, duration).
5. **Resolve** — `2xx` → `done`; all retries exhausted → `failed`.

## Key types & interfaces

- `types.Queue` — `Enqueue(jobType, payload, callbackURL) (model.Event, error)`,
  `Lookup(*model.Status) ([]model.Event, error)`.
- `service.NewQueueService(eventRepo) types.Queue`.
- `service.NewProcessorService(eventRepo, execRepo, ProcessorConfig) *ProcessorService`
  with `Consume()`. `ProcessorConfig{ MaxRetries, PollInterval }`; zero values
  fall back to defaults: `MaxRetries=3`, `PollInterval=5s`. Internal constants:
  `initialBackoff=2s`, `defaultHTTPTimeout=10s`.
- `repository.EventRepository` — `Create`, `FindByID`, `FindAll`,
  `GetProcessable`, `Update`.
- `repository.ExecutionRepository` — `Create`, `FindByEventID`.
- `client.New(baseURL) *client.Client`,
  `(*client.Client).Enqueue(jobType, payload, callbackURL) (client.Event, error)`.
- `model.Event` — `ID, CreatedAt, Payload, Status, JobType, CallbackURL`.
- `model.EventExecution` — `ID, EventID, AttemptedAt, StatusCode, Error, DurationMs`
  (`StatusCode`, `Error`, `DurationMs` are pointers / nullable).
- `model.Status` — `pending`, `processing`, `done`, `failed`; `Status.IsValid()`.

## Environment / config

- `DATABASE_URL` — Postgres connection string. **Required** by
  `cmd/server/main.go`; also read by `db.Initialize()`.
- docker-compose reads `POSTGRES_USER`, `POSTGRES_PASSWORD`, `POSTGRES_DB`,
  `POSTGRES_CONTAINER_NAME`.
- `.env` in the repo root is loaded automatically via `godotenv.Load()`.

## Run & build commands

```bash
docker compose up -d     # start local Postgres
go run ./cmd/server      # run the service (from repo root); serves :3000
go build ./...           # build everything
go vet ./...             # static checks
```

## Conventions & gotchas

- **Callbacks are HTTP GET**, and the `payload` travels in the **query string**
  (url-encoded). Keep payloads small; very large payloads do not belong in a URL.
- **Run from the repo root.** Migrations load from the relative path
  `file://db/migrations` (`db/db_service.go`), so the working directory matters.
- The processor handles **one job per poll cycle**, then sleeps `PollInterval`.
- A non-`2xx` response or any transport error triggers a retry; backoff doubles
  each attempt starting at 2s.
- **Module name is `events`** (not a fully-qualified path), so the Go packages
  are importable in-repo but not `go get`-able from other modules as-is. Prefer
  the REST API for cross-service use.
- **No tests, no linter config, and no CI** exist in the repo yet. If you add
  behavior, consider adding `*_test.go` coverage; there is currently no baseline.

## Database schema (summary)

- `events` — `id`, `payload`, `status` (enum: pending/processing/done/failed),
  `job_type`, `callback_url`, `created_at`.
- `event_executions` — `id`, `event_id` (FK → events), `attempted_at`,
  `status_code`, `error`, `duration_ms`.

Schema is defined by the migrations in `db/migrations/` (`000001`–`000004`).
