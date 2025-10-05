# Performance Budgets

| Scenario | Target | Measurement Method | Latest Result | Notes |
|----------|--------|--------------------|---------------|-------|
| Installer dry-run | < 10 minutes | `scripts/capture-dry-run.sh` (collects CLI timing) | Pending | Requires sudo and kind cluster |
| Helm apply benchmark | < 1 ms/op | `go test -bench BenchmarkHelmInstall -run ^$ -benchmem ./pkg/helm` | See `artifacts/performance/upgrade_baseline.json` | Run on build agent |
| Bootstrap benchmark | < 150 ns/op (mocked) | `go test -bench BenchmarkBootstrap -run ^$ -benchmem ./pkg/bootstrap` | See `artifacts/performance/install_baseline.json` | Mocked runner, validates overhead |
| Memory footprint | < 512 MB RSS | `GODEBUG=madvdontneed=1` with `chainctl cluster install` dry-run | Pending | Requires sudo/k3s cluster |
| Goroutine ceiling | < 200 goroutines | `GODEBUG=scheddetail=1` + pprof capture | Pending | Collect via `go tool pprof` during e2e |

## Validation Procedure

1. Set `GOCACHE=$(pwd)/.gocache` to avoid permission issues.
2. Run benchmarks:
   ```bash
   go test -bench BenchmarkBootstrap -run ^$ -benchmem ./pkg/bootstrap
   go test -bench BenchmarkHelmInstall -run ^$ -benchmem ./pkg/helm
   ```
3. Capture dry-run metrics:
   ```bash
   sudo scripts/capture-dry-run.sh
   ```
4. For memory and goroutine profiling:
   ```bash
   GODEBUG=madvdontneed=1,schedtrace=1000 ./chainctl cluster install --dry-run ...
   go tool pprof -top -http=:0 cpu.prof
   ```
5. Update the table above with observed values and commit updated artifacts under `artifacts/performance/`.
