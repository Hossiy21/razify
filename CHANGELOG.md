# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased] - Phase 1: Architecture Foundation & SARIF

### Added
- **Internal Package Architecture:** Decoupled monolithic `cmd/` package into modular `internal/` packages:
  - `internal/schema`: Environment variable parsing, AST/tag extraction (`@type`, `@enum`, `@range`, `@requires`, `@required`), and type validation.
  - `internal/scanner`: Secret regex rules, danger patterns, weak credential detection, and Shannon entropy analysis.
  - `internal/reporter`: Standardized diagnostic reporting, SARIF v2.1.0 generator, and unified exit code handling.
  - `internal/sandbox`: Foundation primitives for environment sanitization and filtered execution.
- **SARIF v2.1.0 Export:** Added `--sarif` flag to `check`, `scan`, and `validate` commands generating compliant OASIS SARIF v2.1.0 reports for GitHub Code Scanning and IDEs.
- **Golden-File Test Suite:** Added golden-file regression tests locking down the exact `razify check --json` output schema across clean, missing keys, secret leaks, and invalid type states.
- **Exit Code Contract & Override Flag:** Standardized exit codes across all commands:
  - `0`: Clean / all checks passed.
  - `1`: Violations found (schema errors or secret leaks detected).
  - `2`: Fatal runtime/usage errors (e.g., missing input files, invalid arguments).
  - Added global `--exit-code N` flag to override the violation exit code.
- **Benchmarking Suite:** Added Go benchmarks (`testing.B`) proving sub-10ms performance:
  - `BenchmarkValidate`: ~1.2 µs
  - `BenchmarkSarifRender`: ~9.8 µs
  - `BenchmarkScan`: ~11.7 µs
  - `BenchmarkCheck` (with disk I/O): ~5.2 ms
- **Linter Configuration:** Added `.golangci.yml` maintaining strict passing status with `golangci-lint` and `go vet`.

### Changed
- **Offline-First Enforcement:**
  - Removed automatic outbound GitHub release API checks from `audit`.
  - Made the `version` command update check strictly opt-in via `razify version --check`.
  - Core integrity checks and pre-commit guard remain 100% offline with zero cloud dependencies.
- Refactored `cmd/check.go`, `cmd/scan.go`, `cmd/validate.go`, `cmd/diff.go`, `cmd/fix.go`, `cmd/docs.go`, and `cmd/audit.go` to delegate logic to `internal/` packages while maintaining 100% backward compatibility for all CLI flags and JSON output formats.

### Fixed
- Cleaned up unused concurrency locks in `cmd/fix.go`.
- Fixed tagged switches and formatting issues in `cmd/check.go` and `cmd/docs.go`.

