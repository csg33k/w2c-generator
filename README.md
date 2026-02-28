# W-2c EFW2C Generator — Tax Year 2021

A Go web application for generating **W-2c correction files** in SSA's
**EFW2C fixed-width format** (Publication 42-007, Tax Year 2021).

## Stack

| Layer | Technology |
|---|---|
| Language | Go 1.22+ |
| Templating | `html/template` (drop-in for `templ` — see migration note) |
| Interactivity | HTMX 1.9 |
| Styling | Tailwind CSS (CDN) |
| Database | SQLite via `go-sqlite3` |
| Build | Mage (magefile) |
| Architecture | Ports & Adapters (Hexagonal) |

## Project Layout

```
w2c-generator/
├── cmd/
│   └── server/          # Entrypoint
│       └── main.go
├── internal/
│   ├── domain/          # Core models (no framework deps)
│   │   └── models.go
│   ├── ports/           # Interfaces (repository + generator)
│   │   └── ports.go
│   ├── adapters/
│   │   ├── sqlite/      # SQLite repository implementation
│   │   │   └── repository.go
│   │   └── efw2c/       # EFW2C fixed-width file generator
│   │       └── generator.go
│   └── handlers/        # HTTP layer (HTMX + templates)
│       ├── handler.go
│       └── templates.go
├── magefiles/
│   └── magefile.go      # Mage build tasks
├── go.mod
└── README.md
```

## Prerequisites

```bash
# Go 1.22+
go version

# CGO is required for go-sqlite3
# On Ubuntu/Debian:
apt-get install gcc

# Install Mage globally
go install github.com/magefile/mage@latest
```

## Quick Start

```bash
# 1. Clone / enter the project
cd w2c-generator

# 2. Download dependencies
mage tidy

# 3. Build
mage build

# 4. Run (default port 8080, DB file: w2c.db)
mage dev
# OR
mage run

# 5. Open browser
open http://localhost:8080
```

## Environment Variables

| Variable | Default | Description |
|---|---|---|
| `PORT` | `8080` | HTTP listen port |
| `DB_PATH` | `w2c.db` | Path to SQLite database file |

## Mage Tasks

```bash
mage build      # Compile to ./bin/w2c-server
mage dev        # go run ./cmd/server (development)
mage run        # Build then run binary
mage test       # go test ./...
mage tidy       # go mod tidy
mage clean      # Remove bin/ and w2c.db
mage generate   # Run templ generate (if using .templ files)
mage lint       # Run golangci-lint (if installed)
mage install    # Install binary to $GOPATH/bin
```

## EFW2C Records Generated

The tool produces a complete EFW2C submission file with:

| Record | Description |
|---|---|
| `RCA` | Submitter record |
| `RCE` | Employer record |
| `RCW` | Employee correction record (one per employee) |
| `RCT` | Total record |
| `RCF` | Final record |

**Supported W-2c boxes:**
- Box 1 — Wages, tips, other compensation
- Box 2 — Federal income tax withheld
- Box 3 — Social security wages
- Box 4 — Social security tax withheld
- Box 5 — Medicare wages and tips
- Box 6 — Medicare tax withheld

## Migrating to `templ`

The templates in `internal/handlers/templates.go` use `html/template` for
zero-build-step portability. To migrate to proper `templ` files:

```bash
# 1. Install templ CLI
go install github.com/a-h/templ/cmd/templ@latest

# 2. Create .templ files in internal/templates/
# 3. Run: mage generate
# 4. Import generated *_templ.go files in handlers
```

## Architecture Notes

The app follows **Ports & Adapters** (Hexagonal Architecture):

- **Domain** (`internal/domain`) — pure Go structs, no dependencies
- **Ports** (`internal/ports`) — interfaces that the domain depends on
- **Adapters** — concrete implementations:
    - `adapters/sqlite` — satisfies `ports.SubmissionRepository`
    - `adapters/efw2c` — satisfies `ports.EFW2CGenerator`
- **Handlers** — HTTP layer, depends only on ports (not adapters directly)

Swapping SQLite for PostgreSQL means writing a new adapter that implements
`ports.SubmissionRepository` — zero domain code changes required.

## Disclaimer

This tool is provided for internal productivity use. Always validate your
EFW2C submission against the official SSA AccuWage software before filing.