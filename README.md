# tantrum
A small round-robin HTTP load balancer written in Go, with active health checks and automatic retry-on-failure. Like the name suggests: it takes a request and throws it at the right server(throwing a tantrum)

## Contents
- [Features](#features)
- [Architecture](#architecture)
- [Requirements](#requirements)
- [Installation](#installation)
- [Usage](#usage)
  - [Flags](#flags)
  - [Environment variables](#environment-variables)
  - [Examples](#examples)
- [How requests are handled](#how-requests-are-handled)
- [Testing](#testing)
- [Project structure](#project-structure)
- [Known limitations](#known-limitations)
- [Roadmap / ideas](#roadmap--ideas)

## Features
- **Round-robin routing** across a configurable set of backends
- **Active health checks** on a configurable interval, with an immediate check on startup so no backend is trusted blindly
- **Automatic failover** — a request that hits a dead or failing backend is silently retried against a different, live backend before the client ever sees an error
- **Idempotency-aware retries** — only methods safe to repeat (`GET`, `HEAD`, `OPTIONS`, `PUT`, `DELETE`) are retried; `POST` and similar are attempted once, to avoid duplicating side effects
- **Graceful shutdown** — on `SIGINT`/`SIGTERM`, tantrum stops accepting new connections but lets in-flight requests finish before exiting
- **Runtime configuration** via flags or environment variables — no rebuild needed to point it at different backends

## Architecture

```
                         ┌─────────────┐
        client ───────▶  │   tantrum   │
                         │ (port 8080) │
                         └──────┬──────┘
                                │
                   round robin, skips dead backends
                                │
              ┌─────────────────┼─────────────────┐
              ▼                 ▼                 ▼
        ┌───────────┐     ┌───────────┐     ┌───────────┐
        │ backend 1 │     │ backend 2 │     │ backend 3 │
        └───────────┘     └───────────┘     └───────────┘
              ▲                 ▲                 ▲
              └─────────────────┴─────────────────┘
                    periodic health checks
                    (independent goroutine)
```

Two things happen concurrently:
1. **The health checker** (`Balancer.HealthCheck`) runs in its own goroutine. It probes every backend once immediately at startup, then again on a fixed interval, marking each one alive or dead based on the response (or lack of one).
2. **The request handler** (`Balancer.ServeHTTP`) picks the next alive backend via an atomic round-robin counter, forwards the request through that backend's `httputil.ReverseProxy`, and — if the attempt fails — retries against a different alive backend before giving up. Each backend's proxy has a custom error handler that marks it dead the instant a live request fails against it, rather than waiting for the next scheduled health check.

Retries work by first buffering the request body (since it can only be read once) and then sending each attempt's response through an in-memory buffer instead of straight to the client. Only once an attempt succeeds (or every alive backend has been tried) does anything get written to the real connection — see [Known limitations](#known-limitations) for the trade-off this involves.

## Requirements
- Go 1.24 or later (cause it uses `strings.SplitSeq`)

## Installation
```bash
git clone https://github.com/thobbiz/tantrum.git
cd tantrum
go build .
```

## Usage
```bash
./tantrum [flags]
```
or, without building:
```bash
go run .
```

### Flags
| Flag | Description | Default |
|---|---|---|
| `-backends` | Comma-separated list of backend URLs | `http://localhost:8081,http://localhost:8082,http://localhost:8083` |
| `-health-interval` | Health check interval (e.g. `5s`, `1m`) | `10s` |

### Environment variables
Used as a fallback when the corresponding flag isn't set — handy for Docker/systemd where env vars are more natural than CLI args.

| Variable | Equivalent to |
|---|---|
| `BACKENDS` | `-backends` |
| `HEALTH_INTERVAL` | `-health-interval` |

Flags take priority over environment variables if both are set.

### Examples
```bash
# defaults, zero config
go run .

# custom backends and interval via flags
go run . -backends "http://localhost:9001,http://localhost:9002" -health-interval 5s

# via environment variables
BACKENDS="http://api-1:8080,http://api-2:8080" HEALTH_INTERVAL=15s go run .
```

tantrum listens on `:8080`.

## How requests are handled
1. A request comes in on `:8080`.
2. tantrum picks the next alive backend in round-robin order.
3. The request is forwarded to that backend. If it succeeds (any status under 500), the response is returned to the client as-is.
4. If it fails (connection error, or the backend returns 5xx), that backend is marked dead and — for idempotent methods only — tantrum retries against the next alive, untried backend.
5. If every alive backend is exhausted without success, the client gets a `503 Service Unavailable`.

Backends marked dead by a failed request, or by a routine health check, are automatically un-skipped the moment a later health check finds them responding again.

## Testing
```bash
go test ./balancer/... -v
```

Covers:
- Round-robin ordering, dead-backend skipping, and the all-dead / zero-backend edge cases (`balancer_test.go`)
- Health-check state transitions — alive on 200, dead on 5xx, dead on connection refused, and recovery from dead back to alive — against real `httptest.Server` instances (`healthcheck_test.go`)

The retry-on-failure behavior in `ServeHTTP` is currently verified manually rather than with an automated test — see [Roadmap](#roadmap--ideas).

## Project structure
```
.
├── main.go              # entry point: config parsing, server startup, graceful shutdown
├── balancer/
│   ├── backend.go        # Backend type: URL, reverse proxy, alive/dead state
│   ├── balancer.go        # Balancer type: round robin, health checks, retry logic
│   ├── balancer_test.go   # round-robin / skip-dead tests
│   └── healthcheck_test.go # health-check state transition tests
└── go.mod
```

## Current limitations
- **Responses are buffered in memory, not streamed.** To support retries, tantrum holds a backend's full response in memory before writing anything to the client. This makes it unsuitable, as-is, for large file downloads, chunked responses, or Server-Sent Events — a client would wait for the entire response before seeing any of it.
- **Health checks hit the backend's root path (`/`)**, not a dedicated health endpoint. A backend with no handler on `/` may be reported unhealthy even if it's otherwise fine.
- **No weighting or least-connections strategy** — every backend gets an equal share of traffic regardless of capacity.
- **No TLS support.**
- **Backend list is static for the life of the process** — changing it means restarting tantrum.

## Roadmap / ideas
- Configurable health-check path (e.g. `/healthz`) instead of `/`
- Automated test coverage for the retry path using `httptest.Server`
- Dockerfile / docker-compose for a one-command local demo
- Optional streaming mode for responses that don't need retry support
- Pluggable balancing strategies (least-connections, weighted round robin)

## Contributing
Contributions are welcome! Feel free to fork the repository, tweak it and submit a PR.
