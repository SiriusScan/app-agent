package server

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	pb "github.com/SiriusScan/app-agent/proto/hello"
	goapistore "github.com/SiriusScan/go-api/sirius/store"
	"go.uber.org/zap"
)

type fakeScanStateStore struct {
	mu                      sync.Mutex
	values                  map[string]string
	beforeCAS               func(*fakeScanStateStore, string)
	beforeCreate            func(*fakeScanStateStore, string)
	readErr, writeErr       error
	reads, writes, attempts int
}

func (f *fakeScanStateStore) Get(_ context.Context, key string) (string, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.reads++
	value, ok := f.values[key]
	return value, ok, f.readErr
}
func (f *fakeScanStateStore) CompareAndSetValue(_ context.Context, key, expected, value string) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.attempts++
	if f.beforeCAS != nil {
		f.beforeCAS(f, key)
	}
	if f.writeErr != nil {
		return false, f.writeErr
	}
	if raw, ok := f.values[key]; !ok || raw != expected {
		return false, nil
	}
	f.values[key] = value
	f.writes++
	return true, nil
}
func (f *fakeScanStateStore) Create(_ context.Context, key, value string, _ time.Duration) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.attempts++
	if f.beforeCreate != nil {
		f.beforeCreate(f, key)
	}
	if f.writeErr != nil {
		return false, f.writeErr
	}
	if _, ok := f.values[key]; ok {
		return false, nil
	}
	f.values[key] = value
	f.writes++
	return true, nil
}

func testScan(t *testing.T, networkStatus string, agents ...string) string {
	t.Helper()
	meta, err := json.Marshal(AgentSubScanMetadata{DispatchedAgents: agents})
	if err != nil {
		t.Fatal(err)
	}
	scan := goapistore.ScanResult{
		ID: "scan-a", Status: "running", Targets: []string{"192.0.2.0/24"},
		SubScans: map[string]goapistore.SubScan{
			"network": {Type: "network", Enabled: true, Status: networkStatus},
			"agent":   {Type: "agent", Enabled: true, Status: "running", Metadata: meta},
		},
	}
	return encodeTestScan(t, scan)
}
func encodeTestScan(t *testing.T, scan goapistore.ScanResult) string {
	t.Helper()
	data, err := json.Marshal(scan)
	if err != nil {
		t.Fatal(err)
	}
	return base64Encode(data)
}
func decodeTestScan(t *testing.T, raw string) goapistore.ScanResult {
	t.Helper()
	data, err := base64Decode(raw)
	if err != nil {
		t.Fatal(err)
	}
	var scan goapistore.ScanResult
	if err := json.Unmarshal(data, &scan); err != nil {
		t.Fatal(err)
	}
	return scan
}
func testMerge(t *testing.T, f *fakeScanStateStore, agent string, exit int32) *agentScanMerge {
	t.Helper()
	s := &Server{logger: zap.NewNop()}
	merged, err := s.mergeAgentScanState(context.Background(), f, agent, "scan-a",
		&pb.CommandResult{ExitCode: exit, Error: "test failure"}, "", agent,
		[]goapistore.VulnerabilitySummary{{ID: "finding-" + agent, ScanSource: "agent", AgentID: agent}})
	if err != nil {
		t.Fatal(err)
	}
	return merged
}

func TestAgentScanMergeRetriesConcurrentNetworkUpdate(t *testing.T) {
	f := &fakeScanStateStore{values: map[string]string{"currentScan": testScan(t, "running", "192.0.2.1")}}
	f.beforeCAS = func(f *fakeScanStateStore, key string) {
		f.beforeCAS = nil
		scan := decodeTestScan(t, f.values[key])
		scan.Hosts = append(scan.Hosts, goapistore.HostEntry{ID: "192.0.2.2", IP: "192.0.2.2", Sources: []string{"network"}})
		scan.HostsCompleted = 1
		scan.Vulnerabilities = append(scan.Vulnerabilities, goapistore.VulnerabilitySummary{ID: "network-finding", HostID: "192.0.2.2", ScanSource: "network"})
		network := scan.SubScans["network"]
		network.Status = "failed"
		scan.SubScans["network"] = network
		data, _ := base64Decode(encodeTestScan(t, scan))
		var fields map[string]interface{}
		if err := json.Unmarshal(data, &fields); err != nil {
			t.Fatal(err)
		}
		fields["future_writer_field"] = map[string]interface{}{"revision": 42}
		data, _ = json.Marshal(fields)
		f.values[key] = base64Encode(data)
	}
	if merged := testMerge(t, f, "192.0.2.1", 0); merged == nil {
		t.Fatal("merge skipped")
	}
	scan := decodeTestScan(t, f.values["currentScan"])
	if f.attempts != 2 || len(scan.Hosts) != 2 || len(scan.Vulnerabilities) != 2 || scan.HostsCompleted != 1 {
		t.Fatalf("concurrent network data lost: attempts=%d scan=%+v", f.attempts, scan)
	}
	if scan.Status != "failed" || scan.SubScans["network"].Status != "failed" || scan.EndTime == "" {
		t.Fatalf("failure lost: %+v", scan)
	}
	data, _ := base64Decode(f.values["currentScan"])
	var fields map[string]json.RawMessage
	_ = json.Unmarshal(data, &fields)
	if fields["future_writer_field"] == nil {
		t.Fatal("unknown field lost")
	}
}

func TestAgentScanMergeRevalidatesAfterContention(t *testing.T) {
	for _, next := range []string{"new-scan", "completed", "failed", "cancelled", "cancelling", "deleted"} {
		t.Run(next, func(t *testing.T) {
			f := &fakeScanStateStore{values: map[string]string{"currentScan": testScan(t, "running", "192.0.2.1")}}
			var replacement string
			f.beforeCAS = func(f *fakeScanStateStore, key string) {
				f.beforeCAS = nil
				if next == "deleted" {
					delete(f.values, key)
					return
				}
				scan := decodeTestScan(t, f.values[key])
				if next == "new-scan" {
					scan.ID = "scan-b"
				} else {
					scan.Status = next
				}
				replacement = encodeTestScan(t, scan)
				f.values[key] = replacement
			}
			if testMerge(t, f, "192.0.2.1", 0) != nil {
				t.Fatal("stale callback committed")
			}
			if f.writes != 0 || f.values["currentScan"] != replacement {
				t.Fatal("newer state overwritten or resurrected")
			}
		})
	}
}

func TestAgentScanMergeRejectsInvalidStateAndInput(t *testing.T) {
	for _, raw := range []string{"not-base64", base64Encode([]byte("null")), base64Encode([]byte("{")), base64Encode([]byte("[]"))} {
		f := &fakeScanStateStore{values: map[string]string{"currentScan": raw}}
		s := &Server{logger: zap.NewNop()}
		if _, err := s.mergeAgentScanState(context.Background(), f, "192.0.2.1", "scan-a", &pb.CommandResult{}, "", "192.0.2.1", nil); err == nil {
			t.Fatalf("invalid state accepted: %q", raw)
		}
		if f.writes != 0 {
			t.Fatal("invalid state overwritten")
		}
	}
	s := &Server{logger: zap.NewNop()}
	f := &fakeScanStateStore{values: map[string]string{"currentScan": testScan(t, "running", "192.0.2.1")}}
	if _, err := s.mergeAgentScanState(context.Background(), f, "192.0.2.1", "", &pb.CommandResult{}, "", "", nil); err == nil {
		t.Fatal("empty scan ID accepted")
	}
	if _, err := s.mergeAgentScanState(context.Background(), f, "192.0.2.1", "scan-a", nil, "", "", nil); err == nil {
		t.Fatal("nil result accepted")
	}
	if f.reads != 0 {
		t.Fatal("invalid input reached store")
	}
}

func TestAgentScanMergeTerminalAggregation(t *testing.T) {
	for _, tc := range []struct {
		network string
		exit    int32
		want    string
	}{
		{"completed", 0, "completed"}, {"failed", 0, "failed"}, {"cancelled", 0, "cancelled"},
		{"completed", 1, "failed"}, {"cancelled", 1, "failed"}, {"running", 1, "running"},
	} {
		t.Run(tc.network+tc.want, func(t *testing.T) {
			f := &fakeScanStateStore{values: map[string]string{"currentScan": testScan(t, tc.network, "192.0.2.1")}}
			testMerge(t, f, "192.0.2.1", tc.exit)
			scan := decodeTestScan(t, f.values["currentScan"])
			if scan.Status != tc.want {
				t.Fatalf("got %s want %s", scan.Status, tc.want)
			}
			if tc.exit != 0 && scan.SubScans["agent"].Status != "failed" {
				t.Fatal("agent failure presented as success")
			}
			if terminalScanStatus(tc.want) != (scan.EndTime != "") {
				t.Fatal("incorrect end time")
			}
		})
	}
}

func TestAgentScanMergeWaitsForEveryDispatchedAgent(t *testing.T) {
	f := &fakeScanStateStore{values: map[string]string{"currentScan": testScan(t, "completed", "192.0.2.1", "192.0.2.2", "192.0.2.2")}}
	testMerge(t, f, "192.0.2.1", 1)
	scan := decodeTestScan(t, f.values["currentScan"])
	if scan.Status != "running" || scan.SubScans["agent"].Progress.Completed != 1 || scan.SubScans["agent"].Progress.Total != 2 {
		t.Fatalf("early completion: %+v", scan)
	}
	testMerge(t, f, "192.0.2.2", 0)
	scan = decodeTestScan(t, f.values["currentScan"])
	if scan.Status != "failed" || len(scan.Hosts) != 2 || len(scan.Vulnerabilities) != 2 {
		t.Fatalf("bad final merge: %+v", scan)
	}
}

func TestAgentScanMergeDuplicateAndTerminalSubscanAreNoOps(t *testing.T) {
	f := &fakeScanStateStore{values: map[string]string{"currentScan": testScan(t, "running", "192.0.2.1", "192.0.2.2")}}
	testMerge(t, f, "192.0.2.1", 1)
	before := f.values["currentScan"]
	if testMerge(t, f, "192.0.2.1", 0) != nil || before != f.values["currentScan"] {
		t.Fatal("duplicate overwrote failed result")
	}
	for _, state := range []string{"completed", "failed", "cancelled", "cancelling", "disabled"} {
		scan := decodeTestScan(t, before)
		agent := scan.SubScans["agent"]
		if state == "disabled" {
			agent.Enabled = false
		} else {
			agent.Status = state
		}
		scan.SubScans["agent"] = agent
		f.values["currentScan"] = encodeTestScan(t, scan)
		before := f.values["currentScan"]
		if testMerge(t, f, "192.0.2.2", 0) != nil || before != f.values["currentScan"] {
			t.Fatalf("%s subscan overwritten", state)
		}
	}
}

func TestAgentScanConcurrentCallbacksKeepBothResults(t *testing.T) {
	f := &fakeScanStateStore{values: map[string]string{"currentScan": testScan(t, "completed", "192.0.2.1", "192.0.2.2")}}
	s := &Server{logger: zap.NewNop()}
	var wg sync.WaitGroup
	for _, agent := range []string{"192.0.2.1", "192.0.2.2"} {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			merged, err := s.mergeAgentScanState(context.Background(), f, id, "scan-a", &pb.CommandResult{}, "", id,
				[]goapistore.VulnerabilitySummary{{ID: id, ScanSource: "agent", AgentID: id}})
			if err != nil || merged == nil {
				t.Errorf("callback failed: %v", err)
			}
		}(agent)
	}
	wg.Wait()
	scan := decodeTestScan(t, f.values["currentScan"])
	if scan.Status != "completed" || len(scan.Hosts) != 2 || len(scan.Vulnerabilities) != 2 || scan.SubScans["agent"].Progress.Completed != 2 {
		t.Fatalf("agent result lost: %+v", scan)
	}
}

func TestScanStateContentionAndStoreErrors(t *testing.T) {
	for _, mode := range []string{"read", "write", "contention", "cancelled-context"} {
		t.Run(mode, func(t *testing.T) {
			f := &fakeScanStateStore{values: map[string]string{"key": "original"}}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			switch mode {
			case "read":
				f.readErr = errors.New("read failed")
			case "write":
				f.writeErr = errors.New("write failed")
			case "contention":
				f.beforeCAS = func(f *fakeScanStateStore, key string) { f.values[key] += "!" }
			case "cancelled-context":
				cancel()
			}
			changed, err := updateScanState(ctx, f, "key", true, func(string, bool) (string, bool, error) { return "next", true, nil })
			if err == nil || changed || f.writes != 0 {
				t.Fatalf("unexpected success: %v", err)
			}
			if mode == "contention" && f.attempts != maxAgentScanUpdateAttempts {
				t.Fatal("contention was not bounded")
			}
		})
	}
}

func projectionMeta(agents ...string) AgentSubScanMetadata {
	meta := AgentSubScanMetadata{DispatchedAgents: []string{"192.0.2.1", "192.0.2.2"}}
	for _, id := range agents {
		meta.AgentStatuses = append(meta.AgentStatuses, AgentStatusEntry{AgentID: id, Status: "completed", CompletedAt: "2026-11-08T00:00:00Z", HostsFound: 1})
	}
	return meta
}
func readProjection(t *testing.T, f *fakeScanStateStore) map[string]interface{} {
	t.Helper()
	var status map[string]interface{}
	if err := json.Unmarshal([]byte(f.values["agent_scan:scan-a"]), &status); err != nil {
		t.Fatal(err)
	}
	return status
}
func TestAgentScanProjectionConcurrentCreateAndUpdate(t *testing.T) {
	for _, create := range []bool{true, false} {
		f := &fakeScanStateStore{values: make(map[string]string)}
		other := projectionMeta("192.0.2.2")
		other.AgentStatuses[0].Status, other.AgentStatuses[0].Error = "failed", "failed on agent"
		concurrent := &fakeScanStateStore{values: make(map[string]string)}
		if err := updateAgentScanProjection(context.Background(), concurrent, "scan-a", other, false); err != nil {
			t.Fatal(err)
		}
		inject := func(f *fakeScanStateStore, key string) {
			f.beforeCreate, f.beforeCAS = nil, nil
			f.values[key] = concurrent.values[key]
		}
		if create {
			f.beforeCreate = inject
		} else {
			if err := updateAgentScanProjection(context.Background(), f, "scan-a", projectionMeta(), false); err != nil {
				t.Fatal(err)
			}
			f.beforeCAS = inject
		}
		if err := updateAgentScanProjection(context.Background(), f, "scan-a", projectionMeta("192.0.2.1"), false); err != nil {
			t.Fatal(err)
		}
		status := readProjection(t, f)
		if status["status"] != "failed" || status["totalAgents"] != float64(2) || status["completedAgents"] != float64(2) || status["failedAgents"] != float64(1) {
			t.Fatalf("projection lost outcome: %+v", status)
		}
		// An older partial snapshot cannot regress either terminal agent.
		if err := updateAgentScanProjection(context.Background(), f, "scan-a", projectionMeta("192.0.2.1"), false); err != nil {
			t.Fatal(err)
		}
		if after := readProjection(t, f); after["status"] != "failed" || after["failedAgents"] != float64(1) {
			t.Fatal("late snapshot regressed failed result")
		}
	}
}
func TestAgentScanProjectionSeedsFullDispatchAndFailure(t *testing.T) {
	f := &fakeScanStateStore{values: make(map[string]string)}
	meta := projectionMeta("192.0.2.1")
	meta.AgentStatuses[0].Status = "failed"
	if err := updateAgentScanProjection(context.Background(), f, "scan-a", meta, false); err != nil {
		t.Fatal(err)
	}
	status := readProjection(t, f)
	if status["status"] != "running" || status["totalAgents"] != float64(2) || status["failedAgents"] != float64(1) || status["completedAt"] != nil {
		t.Fatalf("premature completion: %+v", status)
	}
}
func TestAgentScanProjectionFailsClosed(t *testing.T) {
	for _, raw := range []string{"null", "{", `{"agentStatuses":null}`, `{"scanId":"other"}`, `{"agentStatuses":[{}]}`} {
		f := &fakeScanStateStore{values: map[string]string{"agent_scan:scan-a": raw}}
		if err := updateAgentScanProjection(context.Background(), f, "scan-a", projectionMeta("192.0.2.1"), false); err == nil {
			t.Fatalf("invalid projection accepted: %s", raw)
		}
		if f.writes != 0 {
			t.Fatal("invalid projection overwritten")
		}
	}
	f := &fakeScanStateStore{values: map[string]string{}, readErr: errors.New("connection failed")}
	if err := updateAgentScanProjection(context.Background(), f, "scan-a", projectionMeta("192.0.2.1"), false); err == nil || f.attempts != 0 {
		t.Fatal("read error treated as missing key")
	}
	for _, state := range []string{"completed", "failed", "cancelled", "cancelling"} {
		raw := `{"scanId":"scan-a","status":"` + state + `"}`
		f := &fakeScanStateStore{values: map[string]string{"agent_scan:scan-a": raw}}
		if err := updateAgentScanProjection(context.Background(), f, "scan-a", projectionMeta("192.0.2.1"), false); err != nil || f.writes != 0 {
			t.Fatal("cancelled projection overwritten")
		}
	}
}

func TestAgentScanEarlyResultDoesNotFinishOpenDispatch(t *testing.T) {
	raw := testScan(t, "completed")
	scan := decodeTestScan(t, raw)
	agent := scan.SubScans["agent"]
	agent.Status = "dispatching"
	scan.SubScans["agent"] = agent
	f := &fakeScanStateStore{values: map[string]string{"currentScan": encodeTestScan(t, scan)}}
	merged := testMerge(t, f, "192.0.2.1", 0)
	if merged == nil || !merged.dispatching {
		t.Fatal("early result skipped")
	}
	scan = decodeTestScan(t, f.values["currentScan"])
	if scan.Status != "running" || scan.SubScans["agent"].Status != "dispatching" || scan.EndTime != "" {
		t.Fatalf("open dispatch finished early: %+v", scan)
	}
	if err := updateAgentScanProjection(context.Background(), f, "scan-a", merged.metadata, merged.dispatching); err != nil {
		t.Fatal(err)
	}
	if status := readProjection(t, f); status["status"] != "dispatching" || status["completedAt"] != nil {
		t.Fatalf("projection finished early: %+v", status)
	}
	// A second early callback is accepted rather than dropped as terminal.
	if testMerge(t, f, "192.0.2.2", 0) == nil {
		t.Fatal("second early result lost")
	}
	scan = decodeTestScan(t, f.values["currentScan"])
	if len(scan.Hosts) != 2 || scan.SubScans["agent"].Progress.Completed != 2 {
		t.Fatal("early results not retained")
	}
}

func TestAgentScanProjectionPreservesLargerExpectedRoster(t *testing.T) {
	// A total supplied by another writer must prevent early completion without
	// allocating an array of that untrusted size.
	f := &fakeScanStateStore{values: map[string]string{"agent_scan:scan-a": `{"scanId":"scan-a","status":"running","totalAgents":1000000000,"agentStatuses":[]}`}}
	if err := updateAgentScanProjection(context.Background(), f, "scan-a", projectionMeta("192.0.2.1", "192.0.2.2"), false); err != nil {
		t.Fatal(err)
	}
	status := readProjection(t, f)
	if status["status"] != "running" || status["completedAgents"] != float64(2) || status["totalAgents"] != float64(1000000000) {
		t.Fatalf("expected roster lost: %+v", status)
	}
}
