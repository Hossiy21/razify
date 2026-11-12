<div align="center">

<img src="assets/logo.png" alt="Razify Logo" width="180" />

# Razify

**The Configuration Integrity Engine for Environment Variables.**

Type-safe validation, schema compliance, secret leak protection, and instant IDE diagnostics.  
Sub-10ms performance. Offline-first. Zero cloud dependency. Every language.

[![npm version](https://img.shields.io/npm/v/razify.svg?color=6C63FF&style=flat-square)](https://www.npmjs.com/package/razify)
[![VS Code Extension](https://img.shields.io/badge/VS_Code-vscode--razify-blue?style=flat-square&logo=visualstudiocode)](https://github.com/Hossiy21/razify)
[![Go Version](https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat-square&logo=go)](https://go.dev)
[![License](https://img.shields.io/badge/license-MIT-brightgreen?style=flat-square)](LICENSE)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen?style=flat-square)](https://github.com/Hossiy21/razify/pulls)

<div align="center">
  <img src="razify-demo.gif" alt="Razify Demo" width="720" />
</div>

</div>

---

## ⚡ Quick Start (60 Seconds)

### Option 1: Zero-Install via `npx` (No Setup Required)
```bash
npx razify check
```

### Option 2: Installed CLI Binary
```bash
# Run single-pass integrity check
razify check

# Automatically sync missing template keys into .env
razify fix

# Protect repository with pre-commit git hook
razify guard install
```

---

## 💥 The Problem Razify Solves

Every engineering team loses hours to broken `.env` files:

- ❌ **"It works on my machine"** — Inconsistent environment keys across developer setups.
- ❌ **Runtime Type Crashes** — Application boots, but crashes when connecting to invalid URLs or ports.
- ❌ **Accidental Secret Leaks** — Staging API keys or passwords committed to Git.
- ❌ **Zero Documentation** — No one knows what an environment variable does after the author leaves.

**Razify solves all four problems in sub-10ms with a single tool.**

---

## 💎 Key Features

| Feature | Command | Description |
|---|---|---|
| ⚡ **Unified Integrity Check** | `razify check` | **Flagship**: Single-pass schema validation, type checking & secret leak scanning. |
| 🩺 **Workspace Doctor** | `razify doctor` | Health checks environment integrity, Go/Node/Docker toolchains, port collisions & DB reachability (~12ms full, ~sub-10ms with `--fast`). |
| 🔒 **Secret Leak Scanner** | `razify scan` | High-entropy string detection catching API keys, JWTs, and passwords. |
| 📋 **Type Schema Engine** | In `.env.example` | Enforces `@type(...)`, `@enum(...)`, `@range(...)`, and `@requires(...)`. |
| 🎨 **VS Code Extension** | `vscode-razify` | Real-time red/yellow squigglies, hover tooltips & auto-completions. |
| 🛡️ **Git Pre-Commit Guard** | `razify guard` | Sub-10ms pre-commit git hook blocking broken or insecure commits. |
| 🤖 **Agent MCP Server** | `razify mcp` | JSON-RPC 2.0 stdio server providing safe schema inspection for Cursor, Claude Desktop & Antigravity without secret leakage. |
| 🛡️ **Agent Policy Guard** | `razify guard --agent` | Shields `.env` files from AI coding assistants and enforces `AGENTS.md` policy directives. |
| 🤖 **GitHub Action** | `action.yml` | 1-step CI/CD Pull Request enforcement gate. |
| 🔄 **Environment Diffing** | `razify diff` | Compare `.env` files side-by-side. |
| 💡 **Auto-Fixer** | `razify fix` | Automatically sync missing template keys into `.env`. |
| 📖 **Doc Generator** | `razify docs` | Auto-generate Markdown documentation from `.env.example`. |
| 📊 **Health Auditor** | `razify audit` | Full configuration health report with a score out of 100. |

---

## 📝 Schema Annotations Guide

Annotate your `.env.example` comments to enforce strict validation rules across your team:

```ini
# @type(port) @range(1000-65535)
APP_PORT=3000

# @enum(dev,staging,prod)
NODE_ENV=dev

# @type(email) @required
ADMIN_EMAIL=admin@company.org

# @type(url)
DATABASE_URL=postgres://user:pass@localhost:5432/mydb

# @requires(DB_HOST)
DB_PASS=change_me

# Stripe API Key for payments
STRIPE_SECRET=sk_test_12345
```

---

## 📦 Installation Options

### 1. npm Zero-Install (`npx`)
Run instantly anywhere Node.js is installed without pre-installing binaries:
```bash
npx razify check
```

### 2. Homebrew (macOS / Linux)
```bash
brew tap Hossiy21/tap
brew install razify
```

### 3. Scoop (Windows)
```bash
scoop bucket add Hossiy21 https://github.com/Hossiy21/scoop-bucket
scoop install razify
```

### 4. Direct Go Install
```bash
go install github.com/Hossiy21/razify@latest
```

---

## 🎨 VS Code / IDE Extension

Install the official **`vscode-razify`** extension for Visual Studio Code / Antigravity IDE:

* 🔴 **Inline Diagnostics**: Real-time red & yellow error squigglies inside `.env` files.
* 💡 **Schema Hover Tooltips**: View `@type(...)` and `@enum(...)` schema rules on mouse hover.
* ⚡ **Auto-Completion**: Suggests missing `.env.example` keys while typing.
* 🛡️ **Check on Save**: Auto-runs background checks whenever a `.env` file is saved.

---

## 🩺 Workspace & Environment Doctor (`razify doctor`)

`razify doctor` executes an offline-first diagnostic health check of your workspace (~12ms full suite), evaluating configuration security, developer toolchains, local TCP port conflicts, and database connectivity.

```bash
# Run full diagnostic suite (~12ms)
razify doctor

# Fast mode: sub-10ms (~6.9ms) run using LookPath and skipping network/port probes (ideal for pre-commit hooks)
razify doctor --fast

# Output as structured JSON or SARIF v2.1.0
razify doctor --json
razify doctor --sarif

# Probe remote (non-localhost) database and cache endpoints
razify doctor --probe-remote
```

### Deterministic Health Score Weights (0–100)

| Category | Diagnostic Check | Max Pts | Passing Condition |
|---|---|---|---|
| **Git / Files (35%)** | Gitignore Leak Guard | **15** | `.gitignore` shields `.env` and file is not tracked in git index (`git ls-files`) |
| | `.env` Present | **10** | Target environment file exists |
| | `.env.example` Present | **10** | Schema template exists for team drift alignment |
| **Schema & Secrets (35%)** | Secret Leak Scan | **15** | Zero high-confidence credentials or secrets in `.env` |
| | Required Variables | **12** | All `@required` variables defined in `.env` |
| | Schema Constraints | **8** | Values satisfy `@type`, `@enum`, `@range` constraints |
| **Toolchains (15%)** | Relevant Project Toolchains | **15** | Go, Node.js, Python, or Docker detected in system PATH |
| **Network & Services (15%)** | Port Conflict Check | **8** | Declared application ports are free to bind on 127.0.0.1 (warnings only if occupied) |
| | Database Probes | **7** | Declared databases/services respond to TCP connect (localhost only by default) |

#### Dynamic Toolchain Point Redistribution
Toolchains are dynamically evaluated according to detected project manifests:
- **Go**: scored when `go.mod` exists.
- **Node.js**: scored when `package.json` exists.
- **Python**: scored when `requirements.txt`, `pyproject.toml`, or `Pipfile` exists.
- **Docker**: scored when `Dockerfile`, `compose.yaml`, or `docker-compose.yml` exists.

The 15 toolchain points are distributed equally across relevant manifests (e.g. 15 pts for a standalone Go project, 7/8 pts for a Go + Docker project). Irrelevant toolchains are marked `SKIPPED` with `0` max points and never penalize your score. If no manifests are detected, Go is evaluated by default.

#### Fast Mode & Score Thresholds
- `--fast`: Skips network and database probes entirely (0 max points assigned so health percentage is calculated purely on static checks). Replaces toolchain version subprocess invocations with `exec.LookPath`-only binary lookups, executing in sub-10ms (~6.9ms vs ~12ms for full doctor).
- `--min-score N`: Configurable threshold (default `90`). If `health_score < min-score`, doctor reports `DEGRADED` or `UNHEALTHY` and exits with code `1`.
- `--probe-remote`: Permits TCP connect probes against non-loopback endpoints. By default, only literal loopback hosts (`localhost`, `127.0.0.1`, `::1`) are probed without DNS lookups.

Every warning or failure provides a single copy-pasteable **Remediation Command** (e.g. `echo ".env" >> .gitignore`, `git rm --cached .env`, `docker compose up -d`).

---

## 🔒 Agent-Safe Sandboxed Execution (`razify run --sandbox`)

`razify run --sandbox` isolates command execution inside an unprivileged OCI container to reduce the risk of secret leakage, unwanted network egress, and unintended host filesystem modifications when executing untrusted or semi-trusted commands (such as those generated or driven by autonomous AI coding agents):

```bash
# Execute tests in an isolated sandbox with synthetic fake secrets and no network
razify run --sandbox -- go test ./...

# Run npm test using the Node project profile
razify run --sandbox -- npm test

# Preview the exact generated docker arguments without executing
razify run --sandbox --dry-run -- pytest

# Restrict injected variables to a specific subset
razify run --sandbox --only PORT,DB_URL -- node server.js
```

### Core Security Controls & Defense-in-Depth
- **Synthetic Secrets by Default (`--fake-secrets`)**: Generates deterministic synthetic values conforming to `.env.example` schema definitions (prefixed with `razify_fake_` so they never trip external scanners). Real credentials are never injected unless explicitly requested via `--inject-secrets`.
- **Zero Arguments in Process Table**: Sensitive values are passed via Docker CLI process environments (`-e KEY`), never as plaintext inline command arguments (`-e KEY=value`).
- **Network Isolation (`--network none`)**: Outbound TCP/UDP and DNS are disabled by default. Network egress requires explicit `--network bridge` (which warns loudly).
- **Read-Only Workspace with `.git` Protection**: Host workspace is mounted `:ro`. Even if `--read-write` is specified, `.git` remains strictly read-only (`:ro`) to mitigate tampering with `.git/hooks` or CI workflows.
- **`.env` File Masking**: Existing `.env` and `.env.*` files within the workspace are masked by bind-mounting an empty file over each one inside the container.
- **Compiler Usability**: An exec-capable temporary scratch filesystem (`--tmpfs /tmp:rw,exec,nosuid,size=512m`) and dedicated cache mappings (`GOCACHE`, `GOMODCACHE`, `npm_config_cache`, `PYTHONPYCACHEPREFIX`) ensure `go build`, `npm`, and `pytest` function smoothly.
- **Stream Masking**: Known secrets, along with their base64-encoded and URL-encoded variants, are automatically redacted from `stdout` and `stderr`.

For complete details on threat boundaries, guardrail rejections, and residual risks, see [docs/THREAT_MODEL.md](docs/THREAT_MODEL.md).

---

## 🤖 Agent Guard & Model Context Protocol (`razify mcp` / `razify guard --agent`)

Autonomous AI coding agents (such as Cursor, Claude Desktop, Antigravity, and Gemini CLI) often attempt to read `.env` files directly when exploring repositories, inadvertently pulling live production secrets into their prompt context windows.

Razify provides a two-pronged solution: an **Agent Policy Guard** that instructs agents never to read raw `.env` files, and a high-performance **Model Context Protocol (MCP)** server that exposes safe environment metadata tools.

### 🛡️ 1. Agent Guard Subsystem (`razify guard --agent`)

```bash
# Check repository protection status and view IDE MCP snippets
razify guard status --agent

# Automatically configure .gitignore and AGENTS.md policy directives
razify guard install --agent

# Remove agent guard policy directives from AGENTS.md
razify guard uninstall --agent
```

When installed, Razify Agent Guard ensures:
- `.env` and `.env.*` patterns are strictly ignored in `.gitignore`.
- An `AGENTS.md` policy block is installed directing coding agents to use Razify MCP tools rather than reading `.env` directly.
- Git pre-commit hooks verify that no secret files are tracked in the git index (`git ls-files`).

### 🔌 2. Model Context Protocol Server (`razify mcp`)

`razify mcp` starts a lightweight JSON-RPC 2.0 stdio server implementing the official MCP specification (2024-11-05). It executes sub-millisecond queries (~31–153 µs) with zero external network overhead:

- 🔍 **`schema_inspect`**: Inspects required keys, types (`@type`), enumeration sets (`@enum`), ranges (`@range`), and documentation from `.env.example`. Never opens or exposes raw `.env` secrets.
- 🧪 **`env_validate_config`**: Hermetically validates candidate key-value maps against schema rules in memory without touching disk, automatically masking secret-typed values in diagnostics.
- 🔄 **`env_diff_keys`**: Compares key names between `.env` and `.env.example` to identify missing or untracked keys with strictly zero value exposure.
- 🩺 **`env_health_audit`**: Computes project environment configuration health scores (0-100) and schema violations, automatically fingerprinting (SHA-256) or masking secret values.

### ⚙️ Connecting Razify MCP to Your AI Tools

#### Cursor (`.cursor/mcp.json`)
Create or edit `.cursor/mcp.json` in your workspace:
```json
{
  "mcpServers": {
    "razify": {
      "command": "razify",
      "args": ["mcp"]
    }
  }
}
```

#### Claude Desktop (`claude_desktop_config.json`)
Add to `claude_desktop_config.json`:
```json
{
  "mcpServers": {
    "razify": {
      "command": "razify",
      "args": ["mcp"]
    }
  }
}
```

#### Antigravity / Gemini CLI (`mcp_config.json`)
Add to your project or global `mcp_config.json`:
```json
{
  "mcpServers": {
    "razify": {
      "command": "razify",
      "args": ["mcp"]
    }
  }
}
```

---

## 🛡️ CI/CD & Team Enforcement

### Pre-Commit Git Hook
Protect local developer commits from leaking secrets or missing required keys:
```bash
razify guard install
```

### GitHub Actions Pipeline
Enforce environment configuration integrity in Pull Requests:

```yaml
name: Configuration Integrity

on: [push, pull_request]

jobs:
  check-env:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Razify Check
        uses: Hossiy21/razify@v1
        with:
          env-file: '.env'
          example-file: '.env.example'
```

---

## 🚦 Exit Code Contract

Razify maintains a strict, unified exit code contract across all terminal, `--json`, and `--sarif` output modes:

| Exit Code | Meaning | Description |
|---|---|---|
| `0` | **Clean / Passed** | Configuration is compliant, valid, and zero secrets or errors were detected. |
| `1` | **Violations Detected** | Schema violations (missing keys, invalid types) or secret leaks detected. |
| `2` | **Fatal Error** | Operational or system failure (target file missing, unparseable arguments, I/O failure). |

### Customizing Violation Exit Code
You can override the exit code returned when violations are found using the global `--exit-code` flag (useful for CI/CD gates that treat warnings with custom status codes):
```bash
razify check --exit-code 42
```

---

## 📊 Output Formats (JSON & SARIF v2.1.0)

Razify natively supports machine-readable output for IDEs, CI/CD pipelines, and GitHub Code Scanning:

```bash
# Output structured JSON
razify check --json

# Output OASIS standard SARIF v2.1.0 for GitHub Code Scanning
razify check --sarif
```

---

## 📄 License

MIT License © 2026 [hossiy21](https://github.com/Hossiy21).