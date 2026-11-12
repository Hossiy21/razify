# Razify Performance Benchmarks & Ecosystem Comparison

Razify is engineered from the ground up for sub-10ms CLI responsiveness, minimal memory footprint, and offline-by-default execution across developer workstations, CI/CD pipelines, and autonomous agent sandboxes.

## 1. Multi-Tool Comparison Benchmark

### Methodology & Environment
- **Test Machine**: 11th Gen Intel(R) Core(TM) i5-1135G7 @ 2.40GHz (8 vCPUs), Linux 6.18 x86_64, Alpine Linux container.
- **Fixture**: Standard multi-variable project directory containing 100 environment variables across `.env` and typed `.env.example`.
- **Timing Command**: `hyperfine -N -i --warmup 20 -m 50` (executed directly via `execvp` without an intermediate `/bin/sh` shell process to eliminate shell spawn jitter).
- **Peak RSS Command**: GNU `/usr/bin/time -v` measuring `Maximum resident set size (kbytes)`.
- **Tool Versions**:
  - `razify`: v2.0.0 (Native Go 1.25)
  - `dotenv-linter`: 4.0.0 (Native Rust)
  - `gitleaks`: 8.30.1 (Native Go)

### Measured Results

| Metric / Attribute | **Razify v2.0** (`razify check`) | **dotenv-linter** (`dotenv-linter .env`) | **gitleaks** (`gitleaks dir --no-git .`) |
|---|---|---|---|
| **Mean Execution Time (± σ)** | **4.6 ms ± 0.6 ms** | **595.8 µs ± 79.1 µs** (~0.60 ms) | **269.2 ms ± 34.1 ms** |
| **Execution Time Range (min … max)** | 4.0 ms … 9.0 ms | 469.7 µs … 2.45 ms | 243.5 ms … 396.1 ms |
| **Peak Resident Set Size (RSS)** | **8,660 KB (~8.46 MB)** | **1,984 KB (~1.94 MB)** | **45,812 KB (~44.74 MB)** |
| **Analysis Scope** | Single-pass schema validation, types, ranges, enums, cross-variable dependencies, missing required keys, and high-entropy secret detection | Dotenv syntax checking, formatting, duplicate key detection, and alphabetical ordering | Filesystem and git commit secret scanning |
| **Detection Mechanics** | High-entropy analysis (Shannon entropy) + regular expression patterns + schema contract assertions | Static linter rules (quotes, spaces, ordering) | Regular expressions + Shannon entropy analysis + keyword pre-filters |
| **Docker Sandbox Execution** | **Yes** (`razify run --sandbox`) | ❌ *Unsupported* | ❌ *Unsupported* |
| **Model Context Protocol (MCP)** | **Yes** (`razify mcp` stdio server) | ❌ *Unsupported* | ❌ *Unsupported* |
| **Pre-Commit Hook Integration** | **Yes** (`razify guard install`) | **Yes** (via pre-commit config) | **Yes** (via pre-commit config) |
| **OASIS SARIF v2.1.0 Export** | **Yes** (`--sarif`) | ❌ *Unsupported* | **Yes** (`--report-format sarif`) |

### Benchmark Corrections & Clarifications
1. **Memory Metric Correction (8.46 MB vs. 17.7 MB):** Earlier drafts mistakenly transcribed the Go benchmark heap allocation metric (`17,787 B/op` from `BenchmarkCheck`) as "17.7 MB". Actual peak process memory measured via `/usr/bin/time -v` is **8,660 KB (~8.46 MB)**.
2. **dotenv-linter Timing Correction (596 µs vs. 24.5 µs):** The earlier 24.5 µs figure was an artifact of shell caching during warm-up. Directly measured via `hyperfine -N --warmup 20`, `dotenv-linter` executes in **595.8 µs ± 79.1 µs**.
3. **Gitleaks Detection Mechanics:** Gitleaks v8 combines regular expressions with Shannon entropy analysis (`entropy` rule attributes) and keyword pre-filtering, rather than regex matching alone.

---

## 2. In-Memory Microbenchmarks (`go test -bench`)

Microbenchmarks executed via `go test -run=^# -bench=. -benchmem ./...`:

| Component / Operation | Benchmark | Latency / op | Memory / op | Allocations / op |
|---|---|---|---|---|
| **Schema Validation** | `BenchmarkValidate` | **1.22 µs** | 418 B | 5 allocs |
| **SARIF Serialization** | `BenchmarkSarifRender` | **9.80 µs** | 8,327 B | 10 allocs |
| **Secret Scanning & Entropy** | `BenchmarkScan` | **11.72 µs** | 714 B | 9 allocs |
| **Doctor Fast Mode (In-Memory)** | `BenchmarkDoctor_Fast` | **28.03 µs** | 25,741 B | 123 allocs |
| **Doctor Full Probing (Mocked)** | `BenchmarkDoctor_FullMocked` | **32.08 µs** | 27,368 B | 160 allocs |
| **Sandbox Environment Processing** | `BenchmarkProcessEnvForSandbox` | **9.91 µs** | 4,296 B | 65 allocs |
| **Sandbox Docker Argv Builder** | `BenchmarkBuildDockerArgs` | **21.18 µs** | 3,332 B | 36 allocs |
| **Sandbox Stream Masker** | `BenchmarkStreamMasker` | **11.45 µs** | 7,328 B | 65 allocs |
| **MCP Tool Listing** | `BenchmarkMCP_ToolsList` | **31.02 µs** | 73,518 B | 44 allocs |
| **MCP Schema Inspection** | `BenchmarkMCP_SchemaInspect` | **150.05 µs** | 77,285 B | 59 allocs |
| **MCP Candidate Config Validation** | `BenchmarkMCP_EnvValidateConfig` | **153.82 µs** | 75,655 B | 59 allocs |
| **Go AST Code Extraction** | `BenchmarkExtract_Go` | **13.18 µs** | 11,224 B | 195 allocs |
| **TypeScript/JavaScript Extraction** | `BenchmarkExtract_JS` | **14.10 µs** | 5,517 B | 24 allocs |
| **Python Code Extraction** | `BenchmarkExtract_Python` | **13.69 µs** | 5,441 B | 21 allocs |
| **Schema Template Generation** | `BenchmarkGenerateEnvExample` | **9.61 µs** | 7,611 B | 64 allocs |
| **Full Single-Pass Check (with Disk I/O)** | `BenchmarkCheck` | **4.6–5.2 ms** | 17,787 B | 78 allocs |

---

## 3. Offline-by-Default Architecture

Razify operates offline-by-default with strictly zero background telemetry, cloud dependencies, or unprompted outbound network calls. Network communication is strictly limited to four explicit, user-initiated opt-in paths:

1. **Opt-In Update Checks (`razify version --check`)**: Performs an explicit GET request to the GitHub Releases API (`api.github.com`) to check for newer releases.
2. **npm Zero-Install Wrapper Download (`npx razify`)**: Downloads the precompiled native release binary from GitHub Releases over HTTPS and verifies its SHA-256 hash against `checksums.txt` before execution.
3. **Remote Doctor Probes (`razify doctor --probe-remote`)**: Probes remote database and cache addresses declared in environment variables (local loopback `127.0.0.1` checks run by default).
4. **Host Container Image Pulls**: The host Docker daemon downloads container images (e.g. `golang:1.25-alpine`) over the host network before starting `razify run --sandbox`. The container runtime itself executes strictly under `--network none` by default.
