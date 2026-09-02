# TaskFlow

A small, production-style task-management microservice written in Go. It was
built to exercise the core skills of a backend engineering role: clean
microservice architecture, REST API design, Go concurrency (goroutines and
channels), graceful lifecycle management, tests, and containerization.

It uses only the Go standard library — no external dependencies — so it builds
anywhere Go 1.22+ is installed.

## Architecture

The service follows a layered, dependency-inverted design. Each layer depends
only on the layer beneath it through interfaces, which keeps business logic
isolated and testable.

```
cmd/taskflow            # entrypoint: wiring + graceful shutdown
  └── internal/
      ├── config        # env-based configuration (12-factor)
      ├── domain        # core entities + errors (no framework deps)
      ├── repository    # storage contract + in-memory implementation
      ├── service       # business logic: validation, IDs, events
      ├── httpapi       # HTTP transport: router, handlers, middleware
      └── worker        # async event processor (goroutine pool + channel)
```

Request flow: `httpapi` → `service` → `repository`. The service emits domain
events that the `worker` consumes asynchronously off a buffered channel, so the
request path never blocks on side effects.

## API

Base path: `/api/v1`

| Method | Path              | Description        | Success |
|--------|-------------------|--------------------|---------|
| GET    | `/healthz`        | Liveness check     | 200     |
| GET    | `/api/v1/tasks`   | List tasks         | 200     |
| POST   | `/api/v1/tasks`   | Create a task      | 201     |
| GET    | `/api/v1/tasks/{id}` | Get a task      | 200     |
| PATCH  | `/api/v1/tasks/{id}` | Partial update  | 200     |
| DELETE | `/api/v1/tasks/{id}` | Delete a task   | 204     |

A task has `id`, `title`, `description`, `status` (`todo` | `in_progress` |
`done`), `created_at`, and `updated_at`. Errors return `{ "error": "..." }`
with the appropriate status code (400 for validation, 404 when not found).

### Examples

```bash
# Create
curl -s -X POST localhost:8080/api/v1/tasks \
  -d '{"title":"Design the API","description":"end to end"}'

# List
curl -s localhost:8080/api/v1/tasks

# Move to in_progress (replace <id>)
curl -s -X PATCH localhost:8080/api/v1/tasks/<id> \
  -d '{"status":"in_progress"}'

# Delete
curl -s -X DELETE localhost:8080/api/v1/tasks/<id> -i
```

## Concurrency model

- Each incoming request runs in its own goroutine (courtesy of `net/http`).
- The in-memory store guards its map with a `sync.RWMutex`, allowing concurrent
  reads while serializing writes.
- The `worker.EventProcessor` runs a pool of goroutines that consume events from
  a bounded, buffered channel. `Publish` is non-blocking: if the queue is full
  it drops the event and records a counter rather than adding latency to the
  request path.
- On shutdown the HTTP server stops accepting requests first, then the worker
  drains its queue and exits. This ordering guarantees no handler publishes to a
  closed channel.

## Configuration

All settings come from environment variables with defaults:

| Variable                    | Default | Description                        |
|-----------------------------|---------|------------------------------------|
| `TASKFLOW_ADDR`             | `:8080` | HTTP listen address                |
| `TASKFLOW_SHUTDOWN_TIMEOUT` | `10s`   | Max wait for in-flight requests    |
| `TASKFLOW_EVENT_BUFFER`     | `256`   | Event queue capacity               |
| `TASKFLOW_EVENT_WORKERS`    | `4`     | Number of event worker goroutines  |

## Running

```bash
# Run locally
make run                 # or: go run ./cmd/taskflow

# Build a binary
make build               # -> bin/taskflow

# Tests (with race detector + coverage)
make test

# Vet / format
make vet
make fmt
```

### Docker

```bash
make docker              # build the image
docker compose up --build
```

The image is a static binary on a distroless non-root base for a small, secure
runtime footprint.

## Notes

Storage is in-memory, so data does not persist across restarts. The repository
is defined as an interface (`repository.TaskRepository`), so swapping in a
Postgres or DynamoDB implementation is a localized change that leaves the
service and transport layers untouched.
