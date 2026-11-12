# Razify Performance Benchmarks

Razify is engineered from the ground up for sub-10ms execution, zero cloud dependency, and offline-first performance across developer machines, CI/CD pipelines, and agent sandboxes.

## Benchmark Results (Go 1.25+ / Linux amd64)

The benchmark suite is executed inside the isolated Docker sandbox via `go test -run=^# -bench=. -benchmem ./...`.

| Component / Operation | Benchmark | Latency / op | Memory / op | Allocations / op |
|---|---|---|---|---|
| **Schema Validation** | `BenchmarkValidate` | **1.22 µs** (1,223 ns) | 418 B | 5 allocs |
| **SARIF Serialization** | `BenchmarkSarifRender` | **9.80 µs** (9,803 ns) | 8,327 B | 10 allocs |
| **Secret Scanning & Entropy** | `BenchmarkScan` | **11.72 µs** (11,719 ns) | 714 B | 9 allocs |
| **Full Single-Pass Check (with Disk I/O)** | `BenchmarkCheck` | **5.25 ms** (5,247,518 ns) | 17,787 B | 78 allocs |

### Performance SLA Verification
- ✅ **Core in-memory validation:** ~1.2 µs (well under 100 µs target).
- ✅ **Secret pattern & entropy scan:** ~11.7 µs per environment file.
- ✅ **End-to-End `razify check` with disk I/O:** ~5.2 ms (surpassing the sub-10ms requirement).
- ✅ **Zero Network Overhead:** Offline-first architecture ensures 0ms network latency.

## Running Benchmarks

To reproduce the benchmark suite inside the Docker sandbox:

```bash
docker run --rm -v "${PWD}:/workspace" -w /workspace golang:1.27-alpine \
  sh -c "go test -run=^# -bench=. -benchmem ./..."
```

