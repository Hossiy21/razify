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
| 🩺 **Workspace Doctor** | `razify doctor` | Health checks environment integrity, Go/Node/Docker toolchains, port collisions & DB reachability with deterministic 0-100 scoring. |
| 🔒 **Secret Leak Scanner** | `razify scan` | High-entropy string detection catching API keys, JWTs, and passwords. |
| 📋 **Type Schema Engine** | In `.env.example` | Enforces `@type(...)`, `@enum(...)`, `@range(...)`, and `@requires(...)`. |
| 🎨 **VS Code Extension** | `vscode-razify` | Real-time red/yellow squigglies, hover tooltips & auto-completions. |
| 🛡️ **Git Pre-Commit Guard** | `razify guard` | Sub-10ms pre-commit git hook blocking broken or insecure commits. |
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

`razify doctor` executes an offline-first diagnostic health check of your workspace, evaluating configuration security, developer toolchains, local TCP port conflicts, and database connectivity.

```bash
# Run full diagnostic suite
razify doctor

# Fast mode: sub-10ms run skipping network/port probes (ideal for pre-commit hooks)
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
| **Git / Files (35%)** | Gitignore Leak Guard | **15** | `.gitignore` shields `.env` from accidental commits |
| | `.env` Present | **10** | Target environment file exists |
| | `.env.example` Present | **10** | Schema template exists for team drift alignment |
| **Schema & Secrets (35%)** | Secret Leak Scan | **15** | Zero high-confidence credentials or secrets in `.env` |
| | Required Variables | **12** | All `@required` variables defined in `.env` |
| | Schema Constraints | **8** | Values satisfy `@type`, `@enum`, `@range` constraints |
| **Toolchains (15%)** | Go Compiler | **5** | Go 1.25+ detected in system PATH |
| | Node.js Runtime | **5** | Node.js runtime detected in system PATH |
| | Docker Engine | **5** | Docker CLI and daemon responsive |
| **Network & Services (15%)** | Port Conflict Check | **8** | All declared application ports are free to bind |
| | Database Probes | **7** | Declared databases/services respond to TCP connect |

Every warning or failure provides a single copy-pasteable **Remediation Command** (e.g. `echo ".env" >> .gitignore`, `lsof -ti :3000 | xargs kill -9`, `docker compose up -d`).

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