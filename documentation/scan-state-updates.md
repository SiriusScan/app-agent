# Agent scan state updates

Coordinated callbacks require a nonempty scan ID matching `currentScan`.
The agent server compares the exact stored base64 value before replacing it,
using the same Lua `GET` comparison and `SET ... KEEPTTL` contract as go-api.
A failed comparison triggers a fresh read, identity check, lifecycle check, and
merge, bounded to eight attempts. Missing current scans are never recreated.
Store errors and malformed state fail closed. No new wire fields are required.

Terminal or cancelling scans, disabled/terminal agent sub-scans, and already
terminal agent outcomes cannot be overwritten by late callbacks. Enabled
sub-scans must all finish before the overall scan finishes. Failure takes
precedence over cancellation; completion means every enabled sub-scan succeeded.
Callbacks arriving while the agent roster is `dispatching` retain their results
but cannot finish the sub-scan. The dispatch owner must seal the roster with a
conditional merge that preserves those early terminal results. The metadata
`dispatched_agents` roster retains every expected agent, including dispatch
failures represented by failed `agent_statuses` entries; the dispatch API may
still return successful dispatches separately. Unknown object
fields and other scanners' metadata are preserved.

After the authoritative update succeeds, `agent_scan:{scanID}` is updated with
its own compare-and-set. First creation uses `SET NX` and a one-hour TTL;
updates retain its existing TTL. The projection starts with all known dispatched
agents and combines terminal per-agent outcomes, so reversed callback order
cannot lose an agent or label a failure successful. Existing terminal or
cancelling projections are preserved. Other components writing these keys must
also use conditional updates/creation to avoid overwriting accepted results.

## Verification

- `go test -race ./internal/server`
- `go test -race ./...`
- `go vet ./...`
- Optional: `SIRIUS_TEST_VALKEY_ADDR=host:port go test -race ./internal/server -run TestAgentScanStateStoreIntegration`

The optional integration test uses only a fresh random expiring test key.
It never reads or changes `currentScan` or dispatches an agent command.
