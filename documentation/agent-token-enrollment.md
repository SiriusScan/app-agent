# Agent token enrollment

`ConnectStream` authenticates before registering or welcoming an agent. A missing
authentication store, failed read, or failed token write rejects the connection.
Only a typed `store.ErrKeyNotFound` means an agent ID is available for enrollment.
Any existing value, even malformed metadata, reserves that ID.

New token records are created with atomic `SET NX`. If concurrent connections
observe the same missing ID, exactly one can create its token; losing connections
must reconnect with the established token. There is no unconditional-write
fallback for custom stores without `store.ValueCreator`.

Returning connections validate their token and receive no replacement token.
The optional last-seen refresh uses compare-and-set against the exact record read,
so it cannot restore a deleted token or overwrite a concurrent replacement.
Custom stores without compare-and-set skip that metadata refresh.

A failed or uncertain enrollment acknowledgement is not rolled back: the write
may have succeeded. A later tokenless connection must not overwrite or recover
an existing token. Recovery of a lost enrollment token requires a separate,
authorized administrative workflow, not automatic re-enrollment.

## SDK and verification

These changes require the matching local `go-api` changes, including typed missing
keys, atomic create, and the compare-and-set capability. Use a development Go
workspace containing both repositories. A release must publish the SDK changes
and update the app-agent SDK dependency before building outside that workspace.
No dependency version has been guessed or published by this change.

Focused checks (from the parent workspace with `GOWORK` set):

```sh
env -u SIRIUS_TEST_VALKEY_ADDR -u SIRIUS_AGGREGATION_TEST_DSN \
  go test -race -count=10 ./minor-projects/go-api/sirius/store ./minor-projects/app-agent/internal/server
go test -run '^$' ./minor-projects/go-api/... ./minor-projects/app-agent/...
```

The enrollment tests use only fake credentials, fake streams, and in-memory
stores. The concrete Valkey adapter test uses `net.Pipe` to verify `SET NX`, nil
replies, and error handling without a listener or a live storage service.
