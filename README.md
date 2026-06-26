# events

A lightweight, PostgreSQL-backed **event / job queue** with HTTP delivery.

Other services enqueue jobs (a `job_type` + an arbitrary `payload` + a
`callback_url`). A background processor picks up each job and delivers it by
calling the job's `callback_url` over HTTP, retrying with exponential backoff
and recording every delivery attempt.

## How it works

1. A producer **enqueues** a job via the REST API (or the Go client).
2. The job is stored in Postgres with status `pending`.
3. A background **processor** polls for pending jobs and marks them `processing`.
4. It delivers the job by issuing an HTTP **GET** to the job's `callback_url`
   with `job_type` and `payload` as query parameters:
   `GET <callback_url>?job_type=<type>&payload=<payload>`
5. A `2xx` response marks the job `done`. Otherwise it retries with exponential
   backoff (defaults: 3 attempts, 2s initial backoff, 10s HTTP timeout). If all
   attempts fail, the job is marked `failed`.
6. Every attempt — status code, error, and duration — is recorded so you can
   inspect delivery history.

Status lifecycle: `pending → processing → done` (or `failed`).

## Tech stack

Go 1.25 · [chi](https://github.com/go-chi/chi) router ·
[pgx/v5](https://github.com/jackc/pgx) · PostgreSQL ·
[golang-migrate](https://github.com/golang-migrate/migrate) (migrations run
automatically on startup).

## Running the service

You need Go 1.25+ and a PostgreSQL database.

```bash
# 1. Start Postgres (a docker-compose.yml is provided for local dev)
docker compose up -d

# 2. Configure the database connection
export DATABASE_URL="postgres://events:events@localhost:5432/events?sslmode=disable"

# 3. Run the service (run from the repo root — migrations load from ./db/migrations)
go run ./cmd/server
```

The service listens on **`:3000`**, runs the processor in the background, and
serves a small HTML dashboard at `/`. Migrations are applied automatically at
startup.

### Environment variables

| Variable                  | Used by        | Description                                  |
| ------------------------- | -------------- | -------------------------------------------- |
| `DATABASE_URL`            | the service    | Postgres connection string (**required**)    |
| `POSTGRES_USER`           | docker-compose | Postgres username                            |
| `POSTGRES_PASSWORD`       | docker-compose | Postgres password                            |
| `POSTGRES_DB`             | docker-compose | Postgres database name                       |
| `POSTGRES_CONTAINER_NAME` | docker-compose | Container name for the Postgres service      |

A local `.env` is loaded automatically. Example:

```dotenv
DATABASE_URL=postgres://events:events@localhost:5432/events?sslmode=disable
POSTGRES_USER=events
POSTGRES_PASSWORD=events
POSTGRES_DB=events
POSTGRES_CONTAINER_NAME=events-postgres
```

## Using it from another service

### From any language (REST)

Enqueue a job by POSTing to `/api/events`:

```bash
curl -X POST http://localhost:3000/api/events \
  -H "Content-Type: application/json" \
  -d '{
    "job_type": "email-send",
    "payload": "{\"to\":\"user@example.com\",\"template\":\"welcome\"}",
    "callback_url": "http://my-service:8080/jobs/email"
  }'
```

Response (`201 Created`):

```json
{
  "id": 42,
  "job_type": "email-send",
  "payload": "{\"to\":\"user@example.com\",\"template\":\"welcome\"}",
  "callback_url": "http://my-service:8080/jobs/email",
  "status": "pending",
  "created_at": "2026-06-26T10:30:00Z"
}
```

**The callback contract:** when the job runs, the events service calls your
`callback_url` with an HTTP **GET**, passing `job_type` and `payload` as query
parameters. Return a `2xx` status to mark the job done; any other status (or a
network error) triggers a retry. A minimal handler in any framework — here in
Go — looks like:

```go
// Receives: GET /jobs/email?job_type=email-send&payload=%7B...%7D
http.HandleFunc("/jobs/email", func(w http.ResponseWriter, r *http.Request) {
    jobType := r.URL.Query().Get("job_type")
    payload := r.URL.Query().Get("payload") // your JSON string, url-decoded

    if err := doWork(jobType, payload); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError) // -> retried
        return
    }
    w.WriteHeader(http.StatusOK) // 2xx -> job marked "done"
})
```

### From Go (client package)

The repo ships a small Go client in the `client` package:

```go
package main

import (
    "log"

    "events/client"
)

func main() {
    c := client.New("http://events-service:3000")

    event, err := c.Enqueue(
        "email-send", // job_type
        `{"to":"user@example.com","template":"welcome"}`, // payload (any string)
        "http://my-service:8080/jobs/email",              // callback_url
    )
    if err != nil {
        log.Fatalf("enqueue failed: %v", err)
    }

    log.Printf("enqueued job %d with status %s", event.ID, event.Status)
}
```

> **Importing the Go client from another module:** this module is currently
> named `events` (see `go.mod`), so `import "events/client"` resolves only
> within this repository. For cross-service integration, the **REST API above is
> the primary, language-agnostic path**. To `go get` the client from another
> module, the module path would need to match its repository
> (e.g. `github.com/ArthurWerle/events`).

## Querying status

Look up jobs by status (defaults to `pending`):

```bash
curl "http://localhost:3000/api/events?status=done"
```

Valid statuses: `pending`, `processing`, `done`, `failed`.

Inspect the delivery attempts for a specific job:

```bash
curl "http://localhost:3000/api/executions?event_id=42"
```

```json
[
  {
    "id": 1,
    "event_id": 42,
    "attempted_at": "2026-06-26T10:31:00Z",
    "status_code": 200,
    "error": null,
    "duration_ms": 245
  }
]
```

## API reference

| Method | Path                            | Description                                        |
| ------ | ------------------------------- | -------------------------------------------------- |
| `POST` | `/api/events`                   | Enqueue a job (`job_type`, `payload`, `callback_url`). Returns `201` + the created event. |
| `GET`  | `/api/events?status=<status>`   | List events by status (defaults to `pending`).     |
| `GET`  | `/api/executions?event_id=<id>` | List delivery attempts for an event.               |
| `GET`  | `/`                             | HTML dashboard.                                    |
