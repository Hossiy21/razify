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

## 📄 License

MIT License © 2026 [hossiy21](https://github.com/Hossiy21).