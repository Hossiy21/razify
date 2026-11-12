# Razify Performance Benchmarks

Razify is engineered from the ground up for sub-10ms execution, zero cloud dependency, and offline-first performance across developer machines, CI/CD pipelines, and agent sandboxes.

## Benchmark Results (Go 1.25+ / Linux amd64)

The benchmark suite is executed inside the isolated Docker sandbox via `go test -run=^# -bench=. -benchmem ./...`.

| Component / Operation | Benchmark | Latency / op | Memory / op | Allocations / op |
|---|---|---|---|---|
| **Schema Validation** | `BenchmarkValidate` | **1.22 µs** (1,223 ns) | 418 B | 5 allocs |
| **SARIF Serialization** | `BenchmarkSarifRender` | **9.80 µs** (9,803 ns) | 8,327 B | 10 allocs |
| **Secret Scanning & Entropy** | `BenchmarkScan` | **11.72 µs** (11,719 ns) | 714 B | 9 allocs |
| **Doctor Fast Mode (In-Memory)** | `BenchmarkDoctor_Fast` | **28.03 µs** (28,030 ns) | 25,741 B | 123 allocs |
| **Doctor Full Probing (Mocked)** | `BenchmarkDoctor_FullMocked` | **32.08 µs** (32,082 ns) | 27,368 B | 160 allocs |
| **Sandbox Environment Processing** | `BenchmarkProcessEnvForSandbox` | **9.91 µs** (9,912 ns) | 4,296 B | 65 allocs |
| **Sandbox Docker Argv Builder** | `BenchmarkBuildDockerArgs` | **21.18 µs** (21,175 ns) | 3,332 B | 36 allocs |
| **Sandbox Stream Masker** | `BenchmarkStreamMasker` | **11.45 µs** (11,445 ns) | 7,328 B | 65 allocs |
| **MCP Tool Listing** | `BenchmarkMCP_ToolsList` | **31.02 µs** (31,021 ns) | 73,518 B | 44 allocs |
| **MCP Schema Inspection** | `BenchmarkMCP_SchemaInspect` | **150.05 µs** (150,057 ns) | 77,285 B | 59 allocs |
| **MCP Candidate Config Validation** | `BenchmarkMCP_EnvValidateConfig` | **153.82 µs** (153,822 ns) | 75,655 B | 59 allocs |
| **Go AST Code Extraction** | `BenchmarkExtract_Go` | **13.18 µs** (13,175 ns) | 11,224 B | 195 allocs |
| **TypeScript/JavaScript Extraction** | `BenchmarkExtract_JS` | **14.10 µs** (14,104 ns) | 5,517 B | 24 allocs |
| **Python Code Extraction** | `BenchmarkExtract_Python` | **13.69 µs** (13,685 ns) | 5,441 B | 21 allocs |
| **Schema Template Generation** | `BenchmarkGenerateEnvExample` | **9.61 µs** (9,614 ns) | 7,611 B | 64 allocs |
| **Full Single-Pass Check (with Disk I/O)** | `BenchmarkCheck` | **5.25 ms** (5,247,518 ns) | 17,787 B | 78 allocs |

### Performance SLA Verification
- ✅ **Core in-memory validation:** ~1.2 µs (well under 100 µs target).
- ✅ **Secret pattern & entropy scan:** ~11.7 µs per environment file.
- ✅ **Sandbox argv generation & stream masking:** ~10–21 µs overhead.
- ✅ **MCP JSON-RPC 2.0 tool execution:** ~31–153 µs latency per request with zero secret disclosure.
- ✅ **Multi-language AST Extraction:** ~9–14 µs per source file analysis.
- ✅ **End-to-End `razify check` with disk I/O:** ~5.2 ms (surpassing the sub-10ms requirement).
- ✅ **Zero Network Overhead:** Offline-first architecture ensures 0ms network latency.

## Running Benchmarks

To reproduce the benchmark suite inside the Docker sandbox:

```bash
docker run --rm -v "${PWD}:/workspace" -w /workspace golang:1.27-alpine \
  sh -c "go test -run=^# -bench=. -benchmem ./..."
```

