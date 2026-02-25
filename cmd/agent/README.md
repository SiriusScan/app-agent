# Legacy Agent Entrypoint (`cmd/agent`)

This entrypoint is legacy and is retained only for backward compatibility and transition support.

## Status

- Deprecated for primary runtime usage.
- Not used by the release pipeline.
- Superseded by `cmd/sirius-agent`.

## Canonical Runtime

Use `cmd/sirius-agent` for all new development, operational docs, and release workflows.

```bash
# Build canonical binary
go build -o sirius-agent ./cmd/sirius-agent

# Start in default server mode
./sirius-agent
```

## Why This Exists

`cmd/agent` remains in the repository to avoid breaking older local workflows while migration to `cmd/sirius-agent` is completed.
