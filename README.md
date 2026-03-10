# mgc — Microsoft Graph CLI

[![Build & Test](https://github.com/chenxizhang/testagent/actions/workflows/build-test.yml/badge.svg)](https://github.com/chenxizhang/testagent/actions/workflows/build-test.yml)
[![Latest Release](https://img.shields.io/github/v/release/chenxizhang/testagent?label=mgc)](https://github.com/chenxizhang/testagent/releases)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/Go-1.22+-blue.svg)](https://golang.org)

A cross-platform, **agent-friendly** command-line tool for the Microsoft Graph API.  
Inspired by [`gh`](https://cli.github.com/) and [`az`](https://docs.microsoft.com/en-us/cli/azure/) — built for humans and AI agents alike.

---

## 🤖 How This Project Is Built

This project is an experiment in **fully automated multi-agent software development** on GitHub.

Three AI coding agents work together in a continuous loop, orchestrated entirely by GitHub Actions:

| Agent | Role | Responsibilities |
|-------|------|-----------------|
| **Copilot** | Developer | Feature implementation, CLI commands, API integration |
| **Claude** | Architect / Tester | Architecture design, code reviews, test writing, documentation |
| **Codex** | Developer | Parallel feature development, build tooling, CI/CD |

The **PM Orchestrator** workflow (`pm-orchestrator.yml`) runs every hour and:
1. Checks which project phase is active (by looking at GitHub issue labels)
2. If a phase is complete, creates the next batch of issues and assigns them to agents
3. Pings agents on stalled issues (inactive > 24h)
4. Reports project status in the workflow logs

See [`docs/AGENTS.md`](docs/AGENTS.md) for the full agent coordination design.

---

## 📦 Installation

### macOS / Linux (install script)
```bash
curl -fsSL https://raw.githubusercontent.com/chenxizhang/testagent/main/scripts/install.sh | bash
```

### Download binary
Download the latest binary for your platform from [Releases](https://github.com/chenxizhang/testagent/releases):

| Platform | Binary |
|----------|--------|
| Linux (amd64) | `mgc-linux-amd64` |
| Linux (arm64) | `mgc-linux-arm64` |
| macOS (Intel) | `mgc-darwin-amd64` |
| macOS (Apple Silicon) | `mgc-darwin-arm64` |
| Windows (amd64) | `mgc-windows-amd64.exe` |

### Build from source
```bash
git clone https://github.com/chenxizhang/testagent
cd testagent/mgc
go build -o mgc .
```

---

## 🚀 Quick Start

```bash
# 1. Authenticate
mgc auth login --tenant your-tenant-id

# 2. List your users
mgc users list

# 3. Get a specific user
mgc users get user@contoso.com
```

---

## 🔐 Authentication

`mgc` uses **OAuth 2.0 Device Flow** — no browser popup needed, perfect for headless/agent use:

```bash
mgc auth login --tenant contoso.onmicrosoft.com
# Output:
# To sign in, use a web browser to open https://microsoft.com/devicelogin
# and enter the code: ABCD-1234
```

After login, credentials are cached in `~/.config/mgc/credentials.json` (encrypted).

```bash
mgc auth status    # Show current authenticated user
mgc auth logout    # Clear cached credentials
```

---

## 📖 Commands Reference

### Users

```bash
mgc users list                                          # List all users
mgc users list --filter "startsWith(displayName,'A')"  # Filter users
mgc users list --all --output json                      # All users as JSON
mgc users get user@contoso.com                          # Get specific user
mgc users create --display-name "John Doe" --upn john@contoso.com --password P@ss1
mgc users update user@contoso.com --job-title "Engineer"
mgc users delete user@contoso.com --yes
```

### Groups

```bash
mgc groups list                                         # List all groups
mgc groups get <group-id>                               # Get group details
mgc groups create --display-name "DevTeam" --mail-nickname devteam --security-enabled
mgc groups members list <group-id>                      # List group members
mgc groups members add <group-id> --member <user-id>   # Add member
mgc groups owners list <group-id>                       # List owners
```

### Mail

```bash
mgc mail list                                           # List inbox
mgc mail list --folder sentitems --top 20               # Sent items
mgc mail read <message-id>                              # Read email
mgc mail send --to user@contoso.com --subject "Hello" --body "World"
mgc mail folders list                                   # List mail folders
```

### Calendar

```bash
mgc calendar events list                                # List upcoming events
mgc calendar events list --start 2026-03-10T00:00:00Z --end 2026-03-17T00:00:00Z
mgc calendar events create --subject "Meeting" --start 2026-03-15T10:00:00 --end 2026-03-15T11:00:00 --timezone UTC
mgc calendar events delete <event-id> --yes
mgc calendar calendars list                             # List calendars
```

### Files (OneDrive)

```bash
mgc files list                                          # List root files
mgc files list /Documents                               # List folder
mgc files download /Reports/Q1.xlsx --output-file ./Q1.xlsx
mgc files upload ./report.pdf --dest /Reports/report.pdf
mgc files delete /OldFile.txt --yes
```

---

## 🎨 Output Formats

All commands support `--output` (`-o`) flag:

```bash
mgc users list -o table    # Default: formatted table
mgc users list -o json     # Pretty JSON
mgc users list -o yaml     # YAML format
mgc users list -o tsv      # Tab-separated values (for scripts)
```

---

## 🔍 Query Filtering (JMESPath)

Use `--query` (`-q`) to filter and transform output using [JMESPath](https://jmespath.org/):

```bash
# Get just display names
mgc users list --query '[].displayName'

# Filter and reshape
mgc users list --query "[?department=='Engineering'].{name: displayName, email: mail}"

# Get first result
mgc calendar events list --query '[0].subject'

# Count items
mgc groups list -o json --query 'length(@)'
```

---

## 🤖 Agent-Friendly Usage

`mgc` is designed for use in scripts and AI agents:

```bash
# Use in scripts — exit code 0 on success, non-zero on error
if mgc users get user@contoso.com -o json > /dev/null 2>&1; then
  echo "User exists"
fi

# Pipe to jq for complex transformations
mgc users list -o json | jq '.[] | select(.jobTitle == "Engineer") | .mail'

# TSV output for easy parsing
mgc users list -o tsv | awk -F'\t' '{print $3}'  # Print UPN column

# Set output format via environment variable
export MGC_OUTPUT=json
mgc users list | jq '.[0].id'
```

---

## ⚙️ Configuration

Config is stored in `~/.config/mgc/config.json`:

```json
{
  "default_tenant": "contoso.onmicrosoft.com",
  "client_id": "14d82eec-204b-4c2f-b7e8-296a70dab67e",
  "default_output": "table"
}
```

```bash
mgc config list                        # Show all config
mgc config set default_output json     # Set a value
mgc config set default_tenant contoso.onmicrosoft.com
```

---

## 🔧 Shell Completion

```bash
mgc completion bash   >> ~/.bashrc      # Bash
mgc completion zsh    >> ~/.zshrc       # Zsh
mgc completion fish   > ~/.config/fish/completions/mgc.fish  # Fish
mgc completion powershell >> $PROFILE   # PowerShell
```

---

## 🏗️ Architecture

See [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) for the full design.

```
User / Agent
     │
     ▼
┌──────────┐
│ mgc CLI  │  (cobra commands)
│ cmd/     │
└────┬─────┘
     │
     ▼
┌──────────────┐
│ internal/    │
│ auth/        │  OAuth2 device flow, token caching
│ client/      │  HTTP client, retry, pagination
│ output/      │  json, table, yaml, tsv, JMESPath
└──────────────┘
     │
     ▼
Microsoft Graph API
https://graph.microsoft.com/v1.0
```

---

## 🤝 Contributing

This project is primarily developed by AI agents, but human contributions are welcome!

1. Fork the repository
2. Create a feature branch
3. Make your changes with tests
4. Submit a PR — automated review will run

See [`docs/AGENTS.md`](docs/AGENTS.md) for how the agent pipeline works.

---

## 📄 License

MIT — see [LICENSE](LICENSE)
