# Unique Clicks Counter Service

A lightweight HTTP service that counts unique user clicks on authors' publications per calendar day (server time). Designed as a self-contained in-memory solution for demonstration purposes.

## Problem Statement

In a content feed, each publication card shows how many unique users clicked on any publication of the same author during the previous day. The service receives:

- `POST /click` – records a click from a user on an author.
- `POST /click_stats` – returns the number of unique users who clicked on the given authors **yesterday**.

The service runs on a single machine, stores all data in memory, and handles concurrency safely.

## Features

- In-memory storage with per‑author unique user sets.
- Automatic daily rotation at midnight (server time) – yesterday's data is aggregated into counters and kept for queries.
- Thread‑safe using `sync.RWMutex`.
- Graceful shutdown.
- Configurable via environment variables (or `.env` file).
- Production‑ready code structure with separation of concerns.
- Unit tests for storage and HTTP handlers (with mocks).

## Architecture

The project follows a simple layered structure:

```
.
├── cmd/
│   └── main.go                 # entry point, server setup
├── internal/
│   ├── handler/                 # HTTP handlers
│   ├── models/                   # shared types (UserID, AuthorID)
│   └── storage/                  # data storage with rotation logic
├── pkg/
│   └── env/                       # helpers for env vars
├── go.mod
└── README.md
```

- **Storage**: maintains two maps: `today` (sets of users per author) and `yesterday` (aggregated counts). A background goroutine triggers rotation at midnight.
- **Handlers**: validate input, call storage methods, return JSON responses.
- **Env**: loads configuration from environment or `.env` file.

## Endpoints

### `POST /api/v1/click`

Record a click.

**Request body:**

```json
{
  "author_id": "author123",
  "user_id": "user456"
}
```

**Responses:**

- `202 Accepted` – click recorded.
- `400 Bad Request` – malformed JSON.
- `422 Unprocessable Entity` – missing `author_id` or `user_id`.
- `405 Method Not Allowed` – wrong HTTP method.

### `POST /api/v1/click_stats`

Retrieve yesterday's unique click counts for a list of authors.

**Request body:**

```json
{
  "author_ids": ["author1", "author2"]
}
```

**Response:**

```json
{
  "status": "OK",
  "stats": {
    "author1": 42,
    "author2": 0
  }
}
```

**Status codes:**

- `200 OK` – success.
- `400 Bad Request` – malformed JSON.
- `422 Unprocessable Entity` – empty author list.
- `405 Method Not Allowed` – wrong method.

## Configuration

Environment variables (can be placed in `.env` file):

| Variable               | Default     | Description             |
| ---------------------- | ----------- | ----------------------- |
| `SERVER_ADDRESS`       | `127.0.0.1` | Bind address            |
| `SERVER_PORT`          | `8080`      | Port                    |
| `SERVER_TIMEOUT_READ`  | `5`         | Read timeout (seconds)  |
| `SERVER_TIMEOUT_WRITE` | `10`        | Write timeout (seconds) |
| `SERVER_TIMEOUT_IDLE`  | `120`       | Idle timeout (seconds)  |

## Build and Run

### Prerequisites

- Go 1.24 or later

### Steps

1. Clone the repository:

   ```bash
   git clone https://github.com/yourname/lil_stats_service.git
   cd lil_stats_service
   ```

2. Build and run:

   ```bash
   go build -o server cmd/main.go
   ./server
   ```

   Or directly:

   ```bash
   go run cmd/main.go
   ```

3. (Optional) Create a `.env` file to override defaults.

### Testing

Run tests with coverage:

```bash
go test -cover ./...
```

## Implementation Notes

- **Concurrency**: All storage methods are protected by a read‑write mutex. The background rotator locks exclusively during rotation.
- **Rotation**: At midnight, the service computes counts from `today`, stores them in a new `yesterday` map, and resets `today`. This avoids copying large sets during queries.
- **Time dependency**: A `timeProvider` function is injected into the storage, making it easy to test rotation without waiting for real time.
- **Idempotency**: Repeated clicks from the same user on the same author within a day are ignored (unique count only).

## Limitations (by design)

- Single instance, in‑memory only – data is lost on restart.
- No persistence – suitable for demonstration, not for production with durability requirements.
- No external dependencies – only Go standard library.

## Future Improvements (if this were a real project)

- Add persistent storage (e.g., Redis, PostgreSQL).
- Implement sharding for horizontal scaling.
- Introduce metrics and structured logging.
- Add integration tests with real HTTP server.

---

**Author:** IvanDamNation  
**License:** MIT
