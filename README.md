<div align="center">

<img src="https://img.shields.io/badge/envy-env%20management-6C63FF?style=for-the-badge" alt="Envy" />

# Envy

**The missing CLI tool for `.env` file management.**

Diff, scan, validate, and document your environment variables.  
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

---

## Installation

**Via Go:**
```bash
go install github.com/hossiy21/envy@latest
```

**Verify:**
```bash
envy --help
```

---

## Usage

### `envy diff` — Compare environments

Catch configuration drift between environments before it causes incidents.

```bash
envy diff .env .env.staging
```

```
Comparing .env → .env.staging

  MISSING in .env.staging: API_KEY
  ADDED in .env.staging:   NEW_FEATURE
  CHANGED: DB_HOST
    .env: localhost
    .env.staging: staging.server.com

Done.
```

---

### `envy scan` — Secret leak detection

Scans for exposed credentials, weak passwords, and dangerous default values.  
Values are masked in output — safe to run in shared terminals.

```bash
envy scan .env
```

```
Scanning .env...

  !! [CRITICAL] Line 6: DB_PASSWORD
     Value : ch****me
     Reason: Weak or default value detected

  !  [HIGH] Line 5: AWS_ACCESS_KEY
     Value : AK****************LE
     Reason: Cloud provider credential

  !  [HIGH] Line 3: API_KEY
     Value : se*****23
     Reason: Looks like an API key

  ~  [MEDIUM] Line 7: SLACK_WEBHOOK
     Value : ht***********************************23
     Reason: Webhook URL — can be abused

Summary: 1 CRITICAL  2 HIGH  1 MEDIUM

  ACTION REQUIRED: Never commit this file to git!
```

---

### `envy validate` — Pre-deploy validation

Define required variables in `.env.example`. Envy enforces them.  
Exits with code `1` on failure — designed for CI/CD pipelines.

```bash
envy validate .env .env.example
```

```
Validating .env against .env.example...

  !! [MISSING]     STRIPE_KEY
      Required key not found in .env

  ~  [PLACEHOLDER] DB_HOST
      Value looks like it was never changed from example

  ok [OK]          API_KEY
  ok [OK]          JWT_TOKEN
  ok [OK]          APP_NAME

Summary: 3 OK   1 MISSING   1 EMPTY/PLACEHOLDER

  ACTION REQUIRED: Add missing keys before deploying!
```

---

### `envy docs` — Auto-generate documentation

Reads comments from `.env.example` and generates a full markdown reference.  
Paste it directly into your project README.

```bash
# Print to terminal
envy docs .env.example

# Save to file
envy docs .env.example -o ENV_DOCS.md
```

**Output:**

```
| Variable       | Required  | Default     | Description                    |
|----------------|-----------|-------------|--------------------------------|
| `DB_HOST`      | No        | `localhost` | Primary database host          |
| `DB_PORT`      | No        | `5432`      | Database port number           |
| `API_KEY`      | **Yes**   | —           | Main API key for external use  |
| `STRIPE_KEY`   | **Yes**   | —           | Stripe payment processing key  |
| `JWT_TOKEN`    | **Yes**   | —           | JWT signing token              |
```

---

## CI/CD Integration

Envy is designed to run inside pipelines. Add it to your GitHub Actions workflow:

```yaml
jobs:
  validate-env:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Install Envy
        run: go install github.com/hossiy21/envy@latest

      - name: Validate environment
        run: envy validate .env .env.example
```

If any required key is missing, the pipeline fails and the deploy is blocked.

---

## Compatibility

Envy works with any project that uses `.env` files — no language lock-in.

| Framework | Compatible |
|---|---|
| React / Next.js | ✅ |
| Node.js | ✅ |
| Python / Django / FastAPI | ✅ |
| Go | ✅ |
| Laravel (PHP) | ✅ |
| Ruby on Rails | ✅ |
| Any `.env`-based project | ✅ |

---

## Roadmap

- [x] `envy diff` — Compare env files
- [x] `envy scan` — Secret leak detection
- [x] `envy validate` — Required variable enforcement
- [x] `envy docs` — Auto-generate documentation
- [ ] `envy init` — Interactive setup wizard
- [ ] JSON and YAML output format flags
- [ ] VS Code extension
- [ ] Web dashboard

---

## Contributing

Contributions are welcome. Please open an issue first to discuss what you would like to change.

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