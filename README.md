<div align="center">

<img src="https://img.shields.io/badge/envy-env%20management-6C63FF?style=for-the-badge" alt="Envy" />

# Envy

**The missing CLI tool for `.env` file management.**

Diff, scan, validate, document, and audit your environment variables.  
Offline. No cloud account. One binary. Every language.

[![Go Version](https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat&logo=go)](https://go.dev)
[![License](https://img.shields.io/badge/license-MIT-brightgreen?style=flat)](LICENSE)
[![Built with Cobra](https://img.shields.io/badge/Built%20with-Cobra-blue?style=flat)](https://github.com/spf13/cobra)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen?style=flat)](https://github.com/hossiy21/envy/pulls)

</div>

---

## The Problem

Every development team has lost hours to `.env` issues:

- **"It works on my machine"** — environment inconsistencies across team members
- **"Which variables do I need?"** — no documentation, no standard
- **"Did someone commit a secret?"** — API keys and passwords leaked to version control
- **"What does this variable do?"** — no one remembers, original author left the team

Envy solves all four problems with a single binary.

---

## Features

| Command | What it does |
|---|---|
| `envy diff` | Compare two `.env` files and show exactly what changed |
| `envy scan` | Detect secret leaks, weak passwords, and exposed credentials |
| `envy validate` | Ensure all required variables are present before deploying |
| `envy docs` | Auto-generate markdown documentation from `.env.example` |
| `envy audit` | Full health report with a score out of 100 |
| `envy guard` | Block git commits that contain exposed secrets |

---

## Installation

```bash
go install github.com/hossiy21/envy@latest
```

Verify:
```bash
envy --help
```

---

## Usage

### `envy diff` — Compare environments

```bash
envy diff .env .env.staging
```

```
Comparing .env → .env.staging

  ✘  MISSING in .env.staging: API_KEY
  ✔  ADDED in .env.staging:   NEW_FEATURE
  ~  CHANGED: DB_HOST
      .env: localhost
      .env.staging: staging.server.com

7 difference(s) found.
```

---

### `envy scan` — Secret leak detection

```bash
envy scan .env
envy scan .env --json
```

```
Scanning .env...

  ✘  [CRITICAL] Line 6: DB_PASSWORD
     Value : ch****me
     Reason: Weak or default value detected

  ⚠  [HIGH]     Line 5: AWS_ACCESS_KEY
     Value : AK****************LE
     Reason: Cloud provider credential

Summary: 1 CRITICAL  4 HIGH  1 MEDIUM

  ✘  ACTION REQUIRED: Never commit this file to git!
```

---

### `envy validate` — Pre-deploy validation

```bash
envy validate .env .env.example
envy validate .env .env.example --json
```

```
Validating .env against .env.example...

  ✘  [MISSING]     STRIPE_KEY
      Required key not found in .env

  ~  [PLACEHOLDER] DB_HOST
      Value looks like it was never changed from example

  ✔  [OK]          API_KEY
  ✔  [OK]          JWT_TOKEN

Summary: 6 OK   1 MISSING   2 EMPTY/PLACEHOLDER

  ✘  ACTION REQUIRED: Add missing keys before deploying!
```

---

### `envy docs` — Auto-generate documentation

```bash
envy docs .env.example
envy docs .env.example -o ENV_DOCS.md
```

```
| Variable       | Required  | Default     | Description                    |
|----------------|-----------|-------------|--------------------------------|
| `DB_HOST`      | No        | `localhost` | Primary database host          |
| `API_KEY`      | **Yes**   | —           | Main API key for external use  |
| `STRIPE_KEY`   | **Yes**   | —           | Stripe payment processing key  |
```

---

### `envy audit` — Full health report

```bash
envy audit .env .env.example
```

```
  ┌─────────────────────────────┐
  │      Envy Audit Report      │
  └─────────────────────────────┘

  ▸ Running scan...
  ▸ Running validate...
  ▸ Running diff...

  ┌─────────────────────────────┐
  │          Results            │
  └─────────────────────────────┘

  Scan        1 CRITICAL  4 HIGH  1 MEDIUM
  Validate    1 MISSING  2 PLACEHOLDER  6 OK
  Diff        7 difference(s) from .env.example

  ┌─────────────────────────────┐
  │        Health Score         │
  └─────────────────────────────┘

  5/100  Critical — needs immediate attention

  Recommendations:
  ✘  Rotate exposed credentials immediately
  ⚠  Add missing required variables before deploying
  ~  Replace placeholder values with real ones
```

---

### `envy guard` — Git commit protection

```bash
envy guard install
envy guard status
envy guard uninstall
```

```
  ✔  Envy Guard installed successfully!
     Every git commit in this repo will now be scanned.
     Commits with exposed secrets will be blocked automatically.
```

---

## CI/CD Integration

```yaml
jobs:
  validate-env:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Install Envy
        run: go install github.com/hossiy21/envy@latest

      - name: Scan for secrets
        run: envy scan .env --json

      - name: Validate environment
        run: envy validate .env .env.example --json
```

---

## JSON Output

Every command supports `--json` for scripting and AI agent integration:

```bash
envy scan .env --json
```

```json
{
  "file": ".env",
  "results": [
    {
      "line": 3,
      "key": "API_KEY",
      "value": "se*****23",
      "reason": "Looks like an API key",
      "risk": "HIGH"
    }
  ],
  "summary": {
    "critical": 1,
    "high": 4,
    "medium": 1,
    "total": 6
  }
}
```

---

## Compatibility

Works with any project that uses `.env` files.

| Framework | Compatible |
|---|---|
| React / Next.js | ✅ |
| Node.js | ✅ |
| Python / Django / FastAPI | ✅ |
| Go | ✅ |
| Laravel (PHP) | ✅ |
| Ruby on Rails | ✅ |

---

## Roadmap

- [x] `envy diff` — Compare env files
- [x] `envy scan` — Secret leak detection
- [x] `envy validate` — Required variable enforcement
- [x] `envy docs` — Auto-generate documentation
- [x] `envy audit` — Full health report
- [x] `envy guard` — Git commit protection
- [x] `--json` flag — AI agent and script support
- [ ] `envy init` — Interactive setup wizard
- [ ] VS Code extension
- [ ] Web dashboard

---

## Contributing

```bash
git clone https://github.com/hossiy21/envy.git
cd envy
go build .
```

---

## License

[MIT](LICENSE) — free to use, modify, and distribute.

---

<div align="center">

Made by [hossiy21](https://github.com/hossiy21)

</div>