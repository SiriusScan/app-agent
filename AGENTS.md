# Repository Guidelines

## Project Structure & Module Organization
Entry points live in `cmd/agent` and `cmd/server`, each with their own README for runtime cues. Shared logic resides in `internal/` packages, with `internal/agent` and `internal/server` orchestrating workflows and complementary packages such as `internal/commands`, `internal/modules`, `internal/fingerprint`, and `internal/config` supporting command execution, registry management, and telemetry. gRPC contracts stay in `proto/hello/hello.proto`; regenerate code whenever the proto changes. Build artifacts land in `bin/`, helper automation in `scripts/`, and containerized test harnesses plus fixtures in `testing/`.

## Build, Test, and Development Commands
`./scripts/generate_proto.sh` regenerates Go stubs from `proto/hello/hello.proto`. Build binaries with `go build -o bin/server cmd/server/main.go` and `go build -o bin/agent cmd/agent/main.go`; `./scripts/build.sh` cross-compiles for Linux, Windows, macOS, and the host. Use `go run cmd/server/main.go` and `go run cmd/agent/main.go` for quick local smoke checks, or invoke `docker-compose -f testing/docker-compose.dev.yaml run --rm agent-test` after `make shell -C testing` to exercise the container harness.

## Coding Style & Naming Conventions
Format Go sources with `gofmt` or `go fmt ./...` before committing; imports should be organized via `goimports`. Follow idiomatic Go naming—packages are short and lowercase, exported symbols use CamelCase when they represent shared APIs, and internal helpers remain unexported. Keep configuration files in `internal/config` and environment keys uppercase with underscores. Prefer structured logging through `zap` as shown in `internal/common`.

## Testing Guidelines
Unit and integration tests live alongside code in `*_test.go` files under `internal/`. Run `go test ./...` before opening a PR and ensure new behavior has direct test coverage. Integration exercises that rely on templates should be run via `make quick -C testing` or targeted with `make test-template TEMPLATE=testing/test-templates/<name>.yaml`. When adding templates or commands, include representative fixtures under `testing/test-data`.

## Commit & Pull Request Guidelines
Use Conventional Commit prefixes (`feat`, `chore`, `fix`, etc.) as seen in recent history (`feat: implement template type system`). Each PR should summarize behavioral changes, reference the related issue or task ID, note any config or dependency updates, and document manual test steps or screenshots. Request at least one reviewer familiar with the touched module and confirm cross-platform builds if `scripts/build.sh` is affected.

## Agent-Specific Tips
The module depends on `github.com/SiriusScan/go-api`; the `go.mod` currently `replace`s it with `../go-api`, so ensure the mirror repo is present when running locally. Server and agent both default to `localhost:50051`; override via `SERVER_ADDRESS`, and set `AGENT_ID` when testing multiple instances.
