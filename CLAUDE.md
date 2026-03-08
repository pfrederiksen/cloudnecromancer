# CloudNecromancer

## Project Overview
CLI tool that reconstructs point-in-time AWS infrastructure snapshots by replaying CloudTrail events. Written in Go 1.22+.

## Build & Test
```bash
make build    # builds to bin/cloudnecromancer
make test     # go test ./...
make lint     # golangci-lint run
```

## Architecture
- `cmd/` — Cobra CLI commands (fetch, resurrect, diff, export, info)
- `internal/aws/` — AWS SDK client interface + CloudTrail fetcher
- `internal/parser/` — Event parser interface, registry, per-service parsers
- `internal/engine/` — Resurrection replay engine, snapshot model, diff logic
- `internal/store/` — DuckDB event cache
- `internal/export/` — Exporters (JSON, Terraform/HCL, CloudFormation, CDK, Pulumi, OCSF, CSV)
- `testdata/` — CloudTrail event JSON fixtures for parser tests

## Key Patterns
- All AWS calls go through the `CloudTrailAPI` interface (in `internal/aws/client.go`) for testability
- Parsers self-register via `init()` → `parser.Register()`
- `ResourceDelta` is the shared contract between parser and engine
- Exporters implement `export.Exporter` interface
- Table-driven tests with `t.Run()` subtests throughout
- Errors wrapped with `fmt.Errorf("context: %w", err)`

## Dependencies
- CLI: `github.com/spf13/cobra`
- AWS: `github.com/aws/aws-sdk-go-v2`
- DB: `github.com/marcboeker/go-duckdb` (CGO required)
- Output: `github.com/charmbracelet/lipgloss`, `github.com/schollz/progressbar/v3`
- Concurrency: `golang.org/x/sync/errgroup`
- Testing: `github.com/stretchr/testify`

## Export Formats
`GetExporter(format)` in `internal/export/exporter.go` supports:
- `json` — indented JSON snapshot
- `terraform` / `hcl` / `tf` — Terraform HCL with import blocks
- `cloudformation` / `cfn` — CloudFormation JSON template
- `cdk` — CDK TypeScript stack
- `pulumi` — Pulumi TypeScript program
- `ocsf` — OCSF Inventory Info (NDJSON)
- `csv` — Splunk lookup table

## Code Quality
- No `panic` in library code
- `go vet` and `staticcheck` clean
- 80%+ test coverage target on `internal/` packages
- No real AWS calls in tests — use mock implementing `CloudTrailAPI`
