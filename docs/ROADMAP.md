# Razify Architecture & Evolution Roadmap

> **Vision:** Evolve Razify from a fast `.env` linter into a modern developer productivity platform anchored by a standout wedge: **"Agent-Safe Environments"** — enabling AI coding agents to inspect schemas, validate variables, and run workloads without ever viewing or exfiltrating real production secrets.

---

## 1. Executive Summary

Razify is a high-performance Go CLI designed to eliminate environment configuration drift and prevent secret leaks. Today, it operates primarily as an integrity checker, scanner, and git pre-commit hook.

To become the primary interface between developers, AI agents, and project environments, Razify is expanding into:
1. **Agent-Safe Environment Isolation:** Sandboxed execution (`razify run --sandbox`) and an MCP server (`razify mcp`) that completely shields raw `.env` contents from autonomous agents while giving them schema awareness.
2. **Developer Diagnostics:** An environment and toolchain health system (`razify doctor`).
3. **Automated Schema Inference:** Code-scanning template generator (`razify init --from-code`) supporting Go, TypeScript/JavaScript, and Python.
4. **Enterprise-Grade Standardization:** A decoupled `internal/` package layout, SARIF v2.1.0 export, structured JSON schemas, standardized exit codes, and sub-10ms benchmarks.

---

## 2. Current Architecture (v1.0.x)

```
                           main.go
                              │
                              ▼
                           cmd/root.go
                              │
     ┌──────────────┬─────────┴─────────┬──────────────┬─────────────┐
     ▼              ▼                   ▼              ▼             ▼
 cmd/check.go   cmd/scan.go       cmd/validate.go  cmd/diff.go   cmd/guard.go
     │              │                   │              │             │
     └──────┬───────┴─────────┬─────────┴──────────────┴─────────────┘
            ▼                 ▼
      cmd/helpers.go     cmd/fix.go / docs.go / audit.go / init.go / version.go
```

### 2.1 File & Responsibility Matrix

| File | Primary Responsibility | Key Functions / Structs |
|---|---|---|
| `main.go` | Entrypoint delegating to `cmd.Execute()`. | `main()` |
| `cmd/root.go` | Cobra root command setup and ASCII banner. | `rootCmd`, `PrintBanner()`, `Execute()` |
| `cmd/helpers.go` | Dotenv parsing, comment extraction, annotation tag parsing (`@type`, `@enum`, `@range`, `@requires`, `@required`). | `EnvVar`, `parseEnvFile()`, `ParseEnvWithMetadata()` |
| `cmd/validate.go` | Template comparison, placeholder detection, type checking (`int`, `bool`, `url`, `email`, `port`, `uuid`, `json`, `ip`), enum/range validation, cross-variable dependency check. | `validateCmd`, `RunValidate()`, `validateValue()` |
| `cmd/scan.go` | Secret detection via provider regex (AWS, Stripe, GitHub, etc.), weak passwords, and Shannon entropy analysis. | `scanCmd`, `RunScan()`, `ShannonEntropy()`, `maskValue()` |
| `cmd/check.go` | Flagship single-pass command combining `RunValidate` and `RunScan`. | `checkCmd`, `RunCheck()`, `CheckOutput` |
| `cmd/diff.go` | Key-by-key comparison of two `.env` files. | `diffCmd`, `RunDiff()`, `DiffResult` |
| `cmd/fix.go` | Interactive/atomic synchronization of missing keys from template with concurrency file locking. | `fixCmd`, `RunFix()`, `acquireFileLock()`, `atomicWriteFile()` |
| `cmd/docs.go` | Markdown documentation generator grouped by category. | `docsCmd`, `buildMarkdown()` |
| `cmd/audit.go` | Composite health score (0–100) combining scan, validate, and diff results. | `auditCmd`, `RunAudit()`, `AuditReport` |
| `cmd/guard.go` | Git pre-commit hook installation and lifecycle management. | `guardCmd`, `installGuard()`, `uninstallGuard()` |
| `cmd/init.go` | Interactive terminal prompt wizard for bootstrapping `.env.example`. | `initCmd` |
| `cmd/version.go` | Version string and network GitHub API update checker. | `versionCmd`, `CheckForUpdates()` |

### 2.2 External Integrations Surface
- **VS Code Extension (`vscode-extension/src/extension.ts`):** Invokes `razify check --json` and relies directly on the JSON structure:
  - `status` ("PASSED" / "FAILED")
  - `validation_results` (`key`, `status` [MISSING, INVALID, EMPTY, PLACEHOLDER], `message`)
  - `secret_scan_results` (`line`, `key`, `value`, `reason`, `risk` [CRITICAL, HIGH, MEDIUM])
  - Also triggers `razify fix`, `razify init`, and `razify guard install`.
- **GitHub Action (`action.yml`):** Runs `go run main.go check "${{ inputs.env-file }}" "${{ inputs.example-file }}"` with exit code verification.
- **npm Wrapper (`npm/bin/razify.js`):** Downloads and runs release binaries from GitHub.
- **Documentation Website (`website/`):** Next.js marketing and documentation site.

---

## 3. Gap Analysis

| Area | Current State (v1.0.x) | Target State (v2.0) | Gap & Required Action |
|---|---|---|---|
| **Package Architecture** | Monolithic `cmd/` containing all CLI flag parsing, business logic, file I/O, and reporting. | Clean `internal/` packages: `schema/`, `scanner/`, `reporter/`, `sandbox/`. | Refactor core logic out of `cmd/` into modular packages. Keep `cmd/` thin (Cobra wiring only). |
| **Output Formats** | Custom terminal ANSI and ad-hoc JSON models (`CheckOutput`, `ValidateOutput`, `ScanOutput`). No SARIF. | Unified `Result` model supporting Human/TTY, `--json`, and `--sarif` (SARIF v2.1.0). | Add `internal/reporter` with standard SARIF and JSON serializers while preserving backward-compatible JSON flags. |
| **Exit Codes** | Ad-hoc `os.Exit(1)` on errors and violations alike. | Standardized POSIX exit codes: `0` (Success/Clean), `1` (System/Usage error), `2` (Violations found). | Implement centralized exit code handler. |
| **Diagnostics & Health** | Basic `razify audit` only checks `.env` vs `.env.example`. | `razify doctor` checks env integrity, Go/Node/Docker versions, port conflicts, and missing services. | Build system inspection engine detecting open ports, running runtime daemons, and calculating actionable health scores. |
| **Agent Safety & Isolation** | None. Agents reading workspace files have full access to plaintext secrets in `.env`. | `razify run --sandbox -- <cmd>` executes in a hardened Docker container with masked secrets and schema filtering. | Build Docker container runner with read-only rootfs, non-root user, dropped capabilities, no network, and schema-approved var injection. |
| **MCP Integration** | None. Agents must guess env vars or read files directly. | `razify mcp` JSON-RPC server with `agent-guard` mode blocking raw `.env` reads. | Implement MCP server exposing schema, diff, audit, and check tools with SHA-256 fingerprinting / redaction. |
| **Template Generation** | Manual terminal questionnaire (`razify init`). | Static source AST analyzer (`razify init --from-code`) for Go, TS/JS, and Python. | Implement multi-language AST scrapers that detect `os.Getenv`, `process.env`, `os.environ` and infer types/defaults. |
| **Offline-First Guarantee** | `version.go` and `audit.go` make outbound HTTP requests to `api.github.com`. | Zero external network calls unless explicitly requested. 100% offline-first. | Remove automatic GitHub release network checks; make update checks opt-in via flag or separate subcommand. |
| **Performance & Benchmarking** | Tests verify functionality, but zero benchmark tests exist. Sub-10ms performance claim is unmeasured. | Comprehensive `testing.B` benchmarks for parser, scanner, validator, and check commands. | Add Go benchmarks and automated performance gates vs `gitleaks` and `dotenv-linter`. |

---

## 4. Target Architecture & Package Layout

```
razify/
├── cmd/                          # Thin Cobra CLI commands
│   ├── root.go
│   ├── check.go
│   ├── scan.go
│   ├── validate.go
│   ├── diff.go
│   ├── fix.go
│   ├── docs.go
│   ├── audit.go
│   ├── guard.go
│   ├── doctor.go                 # Phase 2
│   ├── run.go                    # Phase 3
│   ├── mcp.go                    # Phase 4
│   └── init.go                   # Phase 5
├── internal/
│   ├── schema/                   # Env parsing, AST, tags (@type, @enum, etc.), type checks
│   │   ├── env.go
│   │   ├── parser.go
│   │   ├── validator.go
│   │   └── types.go
│   ├── scanner/                  # Secret rules, regex, Shannon entropy, masking
│   │   ├── rules.go
│   │   ├── entropy.go
│   │   ├── scanner.go
│   │   └── mask.go
│   ├── reporter/                 # Unified Result, SARIF v2.1.0, JSON, Terminal UI
│   │   ├── result.go
│   │   ├── sarif.go
│   │   ├── json.go
│   │   └── terminal.go
│   ├── doctor/                   # Toolchain, port conflict, service detection
│   │   ├── doctor.go
│   │   ├── ports.go
│   │   ├── runtimes.go
│   │   └── services.go
│   ├── sandbox/                  # Container hardening, env filter, process runner
│   │   ├── container.go
│   │   ├── filter.go
│   │   └── runner.go
│   ├── mcp/                      # Model Context Protocol server & agent guard
│   │   ├── server.go
│   │   ├── protocol.go
│   │   ├── tools.go
│   │   └── guard.go
│   └── codegen/                  # AST source code parsers for Go, TS/JS, Python
│       ├── extractor.go
│       ├── golang.go
│       ├── javascript.go
│       └── python.go
├── docs/                         # Architecture, benchmarks, specifications
│   └── ROADMAP.md
├── .github/workflows/            # CI workflows
└── main.go
```

---

## 5. Phased Implementation Roadmap

### Phase 1: Foundation (Internal Package Layout & Diagnostics Engine)
- **Objectives:**
  1. Refactor monolithic `cmd/` into `internal/schema`, `internal/scanner`, and `internal/reporter`.
  2. Implement unified `reporter.Result` and `reporter.Diagnostic` model.
  3. Implement `--sarif` (SARIF v2.1.0) and structured `--json` while preserving exact backward compatibility for `vscode-razify` and existing scripts.
  4. Standardize exit codes: `0` (Success/Pass), `1` (CLI/Runtime error), `2` (Validation or secret violations found).
  5. Remove automatic outbound network calls from `version.go` / `audit.go` to satisfy offline-first rule.
  6. Add Go benchmarks (`testing.B`) proving sub-10ms performance for `check`, `scan`, and `validate`.
- **Definition of Done:**
  - `go test -v ./...` passes inside Docker sandbox (`golang:1.27-alpine`).
  - `go vet ./...` clean.
  - Benchmarks recorded under `docs/BENCHMARKS.md`.
  - Backward compatibility verified with existing test suite and `vscode-extension` mock runs.
  - CHANGELOG entry added.

### Phase 2: `razify doctor` (Environment & Toolchain Diagnostics) [COMPLETED]
- **Objectives:**
  1. Implement `internal/doctor` to inspect project prerequisites.
  2. Integrity checks: validates `.env` against `.env.example`, verifies `.gitignore` guards `.env`.
  3. Toolchain inspection: validates Go (1.25+), Node.js, and Docker availability and version compliance.
  4. Port conflict detection: inspects declared ports via native `net.Listen` without parsing `netstat`/`ss`.
  5. Service detection: auth-less, read-only 300ms TCP-connect probes against declared databases (PostgreSQL, MySQL, Redis, MongoDB).
  6. Credential masking: zero secret leakage in diagnostics, messages, or SARIF output.
  7. Terminal output with deterministic Health Score (0–100) and actionable fix commands for every failure.
  8. `--fast` mode skipping network/service probes for sub-10ms runs.
  9. Unified `--json` and `--sarif` (OASIS v2.1.0 schema-validated) output.
- **Definition of Done:**
  - Table-driven unit tests for all diagnostic probes passing in Docker sandbox (`golang:1.25-alpine`).
  - `go vet ./...` and `golangci-lint run ./...` 100% clean (0 issues).
  - Benchmarks recorded under `docs/BENCHMARKS.md` (`BenchmarkDoctor_Fast`: ~28.0 µs).
  - Documentation and CHANGELOG updated.

### Phase 3: `razify run --sandbox` (Hardened Container Runtime)
- **Objectives:**
  1. Implement `internal/sandbox` container runtime using Docker.
  2. Container hardening parameters:
     - Non-root execution (`--user 1000:1000`).
     - Read-only root filesystem (`--read-only`) with temporary tmpfs mounts if necessary.
     - Capability dropping (`--cap-drop=ALL`).
     - Network isolation: `--network=none` by default; optional `--allow-network` or host allowlist.
     - Resource constraints (`--cpus`, `--memory`).
  3. Schema-based injection: Only environment variables declared and validated in `.env.example` are passed to the container.
  4. Secret safety: Secret-typed variables are masked in output logs and never written to temporary image layers or host volumes.
  5. Preview mode: `--dry-run` flag printing exactly which variables and flags will be supplied to the container without launching.
- **Definition of Done:**
  - Integration tests in Docker sandbox verifying network isolation, capability drops, and secret masking.
  - Verification that `--dry-run` produces clean, redacted preview.
  - CHANGELOG and command reference documentation.

### Phase 4: `razify mcp` (Agent-Safe Environment Wedge)
- **Objectives:**
  1. Implement an MCP (Model Context Protocol) server over standard I/O (JSON-RPC 2.0).
  2. Expose safe tools to AI coding agents:
     - `schema_inspect`: Returns documented environment variables, types, and descriptions.
     - `env_diff`: Compares keys between files without exposing secret values.
     - `env_audit`: Returns configuration health score and missing required variables.
     - `env_check`: Returns validation status and warnings, redacting secret values (replacing them with SHA-256 fingerprints or masks).
  3. Implement `agent-guard` mode:
     - Pre-commit and filesystem access blocker preventing coding agents from directly reading `.env` files.
     - Directs agents to use MCP tools instead of raw file reads.
- **Definition of Done:**
  - Unit tests verifying JSON-RPC 2.0 compliance and secret redaction.
  - End-to-end test simulating an agent querying environment schema without seeing secrets.
  - Documentation detailing how to configure Razify MCP in Cursor, Claude Desktop, Antigravity, and Gemini CLI.

### Phase 5: `razify init --from-code` (AST-Based Template Inference)
- **Objectives:**
  1. Implement `internal/codegen` with multi-language static code analysis:
     - **Go:** Parser for `os.Getenv`, `os.LookupEnv`, `os.Setenv`.
     - **TypeScript/JavaScript:** Parser for `process.env[KEY]`, `process.env.KEY`, `import.meta.env.KEY`.
     - **Python:** Parser for `os.environ[...]`, `os.getenv(...)`, `os.environ.get(...)`.
  2. Infer variable names, categories, and types (`int`, `bool`, `url`, `port`) from contextual usage and default values.
  3. Generate a clean, annotated `.env.example` file formatted with `@type`, `@required`, and `@enum` tags.
- **Definition of Done:**
  - Table-driven tests using fixture repositories in Go, TypeScript, and Python.
  - Verification that generated `.env.example` passes `razify check`.
  - Documentation and CHANGELOG updated.

### Phase 6: Polish, Benchmarking & Release
- **Objectives:**
  1. Comprehensive README rewrite highlighting the "Agent-Safe Environments" story.
  2. Benchmark table comparing speed, memory, and detection accuracy against `gitleaks` and `dotenv-linter`.
  3. 60-second terminal demonstration tape/GIF.
  4. Documentation site update in `website/`.
  5. Full v2.0 release notes and migration guide.
- **Definition of Done:**
  - All CI workflows green.
  - Sub-10ms benchmark SLA verified and documented.
  - Zero compiler warnings, `go vet` clean, clean linter state.

---

## 6. Non-Negotiable Operational Guardrails

1. **Docker Sandbox Execution:** All builds, tests, benchmarks, and experiments must run inside Docker (`golang:1.27-alpine` or target container).
2. **Zero Real Secrets:** Test suites must strictly use fake mock credentials (e.g., `AKIA1234567890EXAMPLE`). Never read, print, or commit live credentials.
3. **Strict Backward Compatibility:** Existing CLI commands (`check`, `scan`, `guard`, `diff`, `fix`, `docs`, `audit`), GitHub Action inputs, and the VS Code extension JSON schema contracts must remain fully backward-compatible.
4. **Sub-10ms Performance:** The core path (`check`, `scan`, `guard`) must execute in under 10 milliseconds for standard repositories.
5. **Zero Cloud Dependencies:** All features must function completely offline without connecting to external cloud services or telemetry.

