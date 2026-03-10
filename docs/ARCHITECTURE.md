# mgc Architecture Design

## Overview

`mgc` (Microsoft Graph CLI) is a cross-platform command-line tool for the Microsoft Graph API. It follows a layered architecture with clear separation of concerns.

## Design Principles

1. **Agent-friendly first**: All output is machine-parseable. No interactive prompts unless in an interactive terminal or `--interactive` flag is used.
2. **Cross-platform**: Single binary for Linux, macOS, Windows. No OS-specific dependencies in the CLI layer.
3. **Composable**: Commands follow Unix conventions — pipe-friendly, predictable exit codes, stderr for errors.
4. **Extensible**: New API resources can be added by creating a new `cmd/` package.
5. **Transparent**: `--debug` flag shows all HTTP requests/responses.

## Layer Diagram

```
┌─────────────────────────────────────────────────────────────┐
│                     User / Agent / Script                    │
└─────────────────────────────┬───────────────────────────────┘
                              │ CLI invocation
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                     CLI Layer (cmd/)                         │
│                                                              │
│  ┌──────────┐  ┌──────────┐  ┌────────┐  ┌──────────────┐  │
│  │  users/  │  │  groups/ │  │  mail/ │  │  calendar/   │  │
│  └──────────┘  └──────────┘  └────────┘  └──────────────┘  │
│  ┌──────────┐  ┌──────────┐  ┌────────┐                     │
│  │  files/  │  │  auth/   │  │  root  │                     │
│  └──────────┘  └──────────┘  └────────┘                     │
│                                                              │
│  Framework: github.com/spf13/cobra                          │
│  Config:    github.com/spf13/viper                          │
└─────────────────────────────┬───────────────────────────────┘
                              │ calls
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                   Internal Packages (internal/)              │
│                                                              │
│  ┌─────────────────────────────────────┐                    │
│  │  internal/auth/                     │                    │
│  │  - AuthManager                      │                    │
│  │  - Device flow OAuth2               │                    │
│  │  - Token caching (encrypted JSON)   │                    │
│  └─────────────────────────────────────┘                    │
│                                                              │
│  ┌─────────────────────────────────────┐                    │
│  │  internal/client/                   │                    │
│  │  - GraphClient                      │                    │
│  │  - HTTP with retry (429, 503)       │                    │
│  │  - OData pagination (@nextLink)     │                    │
│  │  - Typed error responses            │                    │
│  └─────────────────────────────────────┘                    │
│                                                              │
│  ┌─────────────────────────────────────┐                    │
│  │  internal/output/                   │                    │
│  │  - Printer (json/table/yaml/tsv)    │                    │
│  │  - JMESPath filtering              │                    │
│  │  - Color support (--no-color)       │                    │
│  └─────────────────────────────────────┘                    │
│                                                              │
│  ┌─────────────────────────────────────┐                    │
│  │  internal/testutil/                 │                    │
│  │  - Mock HTTP servers               │                    │
│  │  - Test fixtures                    │                    │
│  └─────────────────────────────────────┘                    │
└─────────────────────────────┬───────────────────────────────┘
                              │ HTTPS
                              ▼
┌─────────────────────────────────────────────────────────────┐
│            Microsoft Graph API v1.0                          │
│         https://graph.microsoft.com/v1.0                    │
│                                                              │
│  /users  /groups  /me/messages  /me/events  /me/drive       │
└─────────────────────────────────────────────────────────────┘
```

## Data Flow

### Command execution flow:
```
mgc users list --filter "..." --output json
     │
     ▼ cobra parses args/flags
cmd/users/list.go
     │
     ▼ calls
internal/auth → GetToken() → returns cached/refreshed Bearer token
     │
     ▼ calls
internal/client → GET /users?$filter=... → sends HTTP request
     │                                     retries on 429/503
     │                                     follows @odata.nextLink if --all
     ▼ returns []json.RawMessage
internal/output → Print(data, "json") → applies JMESPath --query
     │                                   formats as json/table/yaml/tsv
     ▼
stdout (machine-readable)
```

## Directory Structure

```
mgc/
├── main.go                    # Entry point, Version constant
├── go.mod
├── go.sum
├── Makefile
├── .golangci.yml
├── .gitignore
│
├── cmd/
│   ├── root.go                # Root command, global flags
│   ├── version.go             # mgc version
│   ├── config.go              # mgc config list/set
│   ├── init.go                # mgc init (setup wizard)
│   ├── completion.go          # mgc completion
│   │
│   ├── auth/
│   │   ├── auth.go            # mgc auth command group
│   │   ├── login.go           # mgc auth login
│   │   ├── logout.go          # mgc auth logout
│   │   └── status.go          # mgc auth status
│   │
│   ├── users/
│   │   ├── users.go           # mgc users command group
│   │   ├── list.go            # mgc users list
│   │   ├── get.go             # mgc users get <id>
│   │   ├── create.go          # mgc users create
│   │   ├── update.go          # mgc users update <id>
│   │   └── delete.go          # mgc users delete <id>
│   │
│   ├── groups/
│   │   ├── groups.go
│   │   ├── list.go
│   │   ├── get.go
│   │   ├── create.go
│   │   ├── members.go         # mgc groups members list/add/remove
│   │   └── owners.go          # mgc groups owners list
│   │
│   ├── mail/
│   │   ├── mail.go
│   │   ├── list.go
│   │   ├── read.go
│   │   ├── send.go
│   │   └── folders.go
│   │
│   ├── calendar/
│   │   ├── calendar.go
│   │   ├── events.go          # mgc calendar events (sub-group)
│   │   ├── list.go
│   │   ├── create.go
│   │   ├── delete.go
│   │   └── calendars.go       # mgc calendar calendars list
│   │
│   └── files/
│       ├── files.go
│       ├── list.go
│       ├── download.go
│       ├── upload.go
│       └── delete.go
│
└── internal/
    ├── auth/
    │   ├── auth.go            # AuthManager interface + implementation
    │   ├── token.go           # Token storage and encryption
    │   ├── device_flow.go     # OAuth2 device flow
    │   └── auth_test.go
    │
    ├── client/
    │   ├── graph.go           # GraphClient
    │   ├── response.go        # Response types
    │   ├── errors.go          # GraphError types
    │   └── graph_test.go
    │
    ├── output/
    │   ├── format.go          # Printer, format logic
    │   └── format_test.go
    │
    └── testutil/
        └── testutil.go        # Shared test utilities
```

## Configuration

Config file: `~/.config/mgc/config.json` (Linux/macOS)  
Config file: `%APPDATA%\mgc\config.json` (Windows)

```json
{
  "default_tenant": "contoso.onmicrosoft.com",
  "client_id": "14d82eec-204b-4c2f-b7e8-296a70dab67e",
  "default_output": "table",
  "default_select": ""
}
```

Credentials file: `~/.config/mgc/credentials.json` (encrypted, XOR + base64)

```json
{
  "tenants": {
    "contoso.onmicrosoft.com": {
      "access_token": "<encrypted>",
      "refresh_token": "<encrypted>",
      "expires_at": "2026-03-10T23:00:00Z",
      "user_id": "user@contoso.com"
    }
  }
}
```

## Error Handling Strategy

1. **User errors** (wrong flags, missing required args): Show usage + specific error message. Exit code 1.
2. **Auth errors** (401, no credentials): Show `Run 'mgc auth login' to authenticate`. Exit code 2.
3. **Not found** (404): Show `Resource not found: <id>`. Exit code 3.
4. **Rate limit** (429): Retry automatically with exponential backoff. User only sees error after 3 retries.
5. **Server errors** (5xx): Show Graph API error message. Exit code 4.

All errors go to stderr. Use `2>/dev/null` to suppress in scripts.

## Output Format Details

### table (default)
Uses `tablewriter` for aligned columns. Only shows key columns unless `--select` is used.

### json
Pretty-printed JSON array (or object for single items). Suitable for `jq` piping.

### yaml
YAML format. Suitable for configuration-like output.

### tsv
Tab-separated values with header row. Suitable for `awk`/`cut` processing.

## Authentication Design (Device Flow)

Why device flow?
- Works in headless environments (CI, agent scripts, SSH)
- No redirect URI needed
- Works with MFA
- Single browser step, then token is cached

Flow:
1. `mgc auth login` → POST to `/oauth2/v2.0/devicecode`
2. Print user code and URL to stderr
3. Poll `/oauth2/v2.0/token` every 5 seconds
4. On success, store access_token + refresh_token
5. Token refresh happens transparently when expired

Client ID: `14d82eec-204b-4c2f-b7e8-296a70dab67e` (Microsoft Graph Explorer - public)

## Testing Strategy

- **Unit tests**: All internal packages. Use `httptest.NewServer` for HTTP mocking.
- **Integration tests** (cmd/): Test cobra commands with mock HTTP server.
- **No real network calls** in automated tests.
- Coverage targets: ≥70% for `internal/`, ≥60% for `cmd/`.

Run: `cd mgc && go test ./... -race`
