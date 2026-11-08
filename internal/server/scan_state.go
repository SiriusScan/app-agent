package server

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	pb "github.com/SiriusScan/app-agent/proto/hello"
	goapistore "github.com/SiriusScan/go-api/sirius/store"
	valkey "github.com/valkey-io/valkey-go"
)

const maxAgentScanUpdateAttempts = 8

// Keep the exact-value comparison and TTL semantics in sync with go-api's
// ValueCompareAndSetter. Using the existing client also supports older SDKs.
var agentScanCompareAndSet = valkey.NewLuaScript(`
if redis.call('GET', KEYS[1]) ~= ARGV[1] then
    return 0
end
redis.call('SET', KEYS[1], ARGV[2], 'KEEPTTL')
return 1
`)

type scanStateStore interface {
	Get(context.Context, string) (string, bool, error)
	CompareAndSetValue(context.Context, string, string, string) (bool, error)
	Create(context.Context, string, string, time.Duration) (bool, error)
}

type valkeyScanStateStore struct{ client valkey.Client }

func (s valkeyScanStateStore) Get(ctx context.Context, key string) (string, bool, error) {
	value, err := s.client.Do(ctx, s.client.B().Get().Key(key).Build()).ToString()
	if valkey.IsValkeyNil(err) {
		return "", false, nil
	}
	return value, err == nil, err
}

func (s valkeyScanStateStore) CompareAndSetValue(ctx context.Context, key, expected, value string) (bool, error) {
	result, err := agentScanCompareAndSet.Exec(ctx, s.client, []string{key}, []string{expected, value}).ToInt64()
	return result == 1 && err == nil, err
}

func (s valkeyScanStateStore) Create(ctx context.Context, key, value string, ttl time.Duration) (bool, error) {
	err := s.client.Do(ctx, s.client.B().Set().Key(key).Value(value).Nx().Ex(ttl).Build()).Error()
	if valkey.IsValkeyNil(err) {
		return false, nil
	}
	return err == nil, err
}

// The mutation can run repeatedly after contention. It must not publish side
// effects, and must revalidate identity and lifecycle against each fresh value.
func updateScanState(ctx context.Context, kv scanStateStore, key string, allowCreate bool,
	mutate func(string, bool) (string, bool, error),
) (bool, error) {
	for attempt := 0; attempt < maxAgentScanUpdateAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return false, err
		}
		raw, exists, err := kv.Get(ctx, key)
		if err != nil {
			return false, fmt.Errorf("read %s: %w", key, err)
		}
		if !exists && !allowCreate {
			return false, nil
		}
		next, changed, err := mutate(raw, exists)
		if err != nil || !changed {
			return false, err
		}
		var swapped bool
		if exists {
			swapped, err = kv.CompareAndSetValue(ctx, key, raw, next)
		} else {
			swapped, err = kv.Create(ctx, key, next, time.Hour)
		}
		if err != nil {
			return false, fmt.Errorf("update %s: %w", key, err)
		}
		if swapped {
			return true, nil
		}
	}
	return false, fmt.Errorf("%s changed too often during update", key)
}

func terminalScanStatus(status string) bool {
	return status == "completed" || status == "failed" || status == "cancelled"
}

// Completion is not success: failure takes precedence over cancellation.
func aggregateScanStatuses(statuses []string) (string, int, int) {
	terminal, failed, cancelled := 0, 0, 0
	for _, status := range statuses {
		if terminalScanStatus(status) {
			terminal++
		}
		if status == "failed" {
			failed++
		}
		if status == "cancelled" {
			cancelled++
		}
	}
	if len(statuses) == 0 || terminal != len(statuses) {
		return "running", terminal, failed
	}
	if failed > 0 {
		return "failed", terminal, failed
	}
	if cancelled > 0 {
		return "cancelled", terminal, failed
	}
	return "completed", terminal, failed
}

type agentScanMerge struct {
	metadata    AgentSubScanMetadata
	hostIP      string
	vulnsAdded  int
	dispatching bool
}

func (s *Server) mergeAgentScanState(ctx context.Context, kv scanStateStore, agentID, scanID string,
	result *pb.CommandResult, hostname, primaryIP string, vulns []goapistore.VulnerabilitySummary,
) (*agentScanMerge, error) {
	if result == nil || agentID == "" || scanID == "" {
		return nil, fmt.Errorf("result, agent ID, and scan ID are required")
	}
	var merged *agentScanMerge
	changed, err := updateScanState(ctx, kv, "currentScan", false, func(raw string, _ bool) (string, bool, error) {
		merged = nil
		decoded, err := base64Decode(raw)
		if err != nil {
			return "", false, fmt.Errorf("decode currentScan: %w", err)
		}
		var scan *goapistore.ScanResult
		if err := json.Unmarshal(decoded, &scan); err != nil {
			return "", false, err
		}
		if scan == nil {
			return "", false, fmt.Errorf("currentScan is null")
		}
		if scan.ID != scanID || terminalScanStatus(scan.Status) || scan.Status == "cancelling" {
			return "", false, nil
		}
		agent, exists := scan.SubScans["agent"]
		if exists && (!agent.Enabled || terminalScanStatus(agent.Status) || agent.Status == "cancelling") {
			return "", false, nil
		}
		if !exists {
			agent = goapistore.SubScan{Type: "agent", Enabled: true,
				Progress: goapistore.SubScanProgress{Label: "agents"}}
		}
		var meta AgentSubScanMetadata
		if len(agent.Metadata) > 0 {
			if err := json.Unmarshal(agent.Metadata, &meta); err != nil || string(agent.Metadata) == "null" {
				return "", false, fmt.Errorf("invalid agent metadata")
			}
		}
		for _, status := range meta.AgentStatuses {
			if status.AgentID == agentID && (terminalScanStatus(status.Status) || status.Status == "cancelling") {
				return "", false, nil // Duplicate or late callback: first terminal result wins.
			}
		}
		hostIP := s.resolveAgentHostIP(agentID, hostname, primaryIP, scan)
		now := time.Now().UTC().Format(time.RFC3339)
		meta.DispatchedAgents = appendUniqueString(meta.DispatchedAgents, agentID)
		entry := AgentStatusEntry{AgentID: agentID, CompletedAt: now, Status: "completed"}
		if result.ExitCode != 0 {
			entry.Status, entry.Error = "failed", result.Error
		} else if hostIP != "" {
			entry.HostsFound, entry.VulnerabilitiesFound = 1, len(vulns)
		}
		found := false
		for i, status := range meta.AgentStatuses {
			if status.AgentID == agentID {
				entry.StartedAt = status.StartedAt
				meta.AgentStatuses[i], found = entry, true
				break
			}
		}
		if !found {
			meta.AgentStatuses = append(meta.AgentStatuses, entry)
		}
		// Count only unique dispatched agents, not duplicate/unrelated status rows.
		statuses := make([]string, 0, len(meta.DispatchedAgents))
		seen := make(map[string]bool)
		for _, id := range meta.DispatchedAgents {
			if id == "" || seen[id] {
				continue
			}
			seen[id] = true
			status := "pending"
			for _, entry := range meta.AgentStatuses {
				if entry.AgentID == id {
					status = entry.Status
					break
				}
			}
			statuses = append(statuses, status)
		}
		dispatching := agent.Status == "dispatching"
		agent.Status, agent.Progress.Completed, _ = aggregateScanStatuses(statuses)
		// The UI registers the complete roster only after dispatch returns. Early
		// callbacks must not finish an agent sub-scan while that roster is open.
		if dispatching {
			agent.Status = "dispatching"
		}
		agent.Progress.Total = len(statuses)
		metadata, err := json.Marshal(meta)
		if err != nil {
			return "", false, err
		}
		// Preserve agent metadata extensions from other writers.
		if len(agent.Metadata) > 0 {
			metadata, err = overlayJSONFields(agent.Metadata, metadata)
			if err != nil {
				return "", false, err
			}
		}
		agent.Metadata = metadata
		if scan.SubScans == nil {
			scan.SubScans = make(map[string]goapistore.SubScan)
		}
		scan.SubScans["agent"] = agent
		enabledStatuses := make([]string, 0, len(scan.SubScans))
		for _, sub := range scan.SubScans {
			if sub.Enabled {
				enabledStatuses = append(enabledStatuses, sub.Status)
			}
		}
		if status, _, _ := aggregateScanStatuses(enabledStatuses); terminalScanStatus(status) {
			scan.Status = status
			if scan.EndTime == "" {
				scan.EndTime = now
			}
		}
		added := 0
		if hostIP != "" {
			hostIdx := -1
			for i, host := range scan.Hosts {
				if host.IP == hostIP {
					hostIdx = i
					break
				}
			}
			if hostIdx < 0 {
				scan.Hosts = append(scan.Hosts, goapistore.HostEntry{ID: hostIP, IP: hostIP, Hostname: hostname, Sources: []string{"agent"}})
			} else {
				host := &scan.Hosts[hostIdx]
				if hostname != "" {
					if host.Hostname == "" {
						host.Hostname = hostname
					} else if host.Hostname != hostname {
						host.Aliases = appendUniqueString(host.Aliases, hostname)
					}
				}
				host.Sources = appendUniqueString(host.Sources, "agent")
			}
			scan.Vulnerabilities, added = appendAgentVulnerabilitiesDeduped(scan.Vulnerabilities, vulns, hostIP, agentID)
		}
		updated, err := json.Marshal(scan)
		if err != nil {
			return "", false, err
		}
		updated, err = overlayJSONFields(decoded, updated)
		if err != nil {
			return "", false, err
		}
		merged = &agentScanMerge{metadata: meta, hostIP: hostIP, vulnsAdded: added, dispatching: dispatching}
		return base64Encode(updated), true, nil
	})
	if err != nil || !changed {
		return nil, err
	}
	return merged, nil
}

// Preserve fields that this SDK does not understand, including sub-scan
// extensions. Arrays are replaced because their membership can change.
func overlayJSONFields(original, updated []byte) ([]byte, error) {
	var oldFields, newFields map[string]json.RawMessage
	if err := json.Unmarshal(original, &oldFields); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(updated, &newFields); err != nil {
		return nil, err
	}
	if oldFields == nil {
		return nil, fmt.Errorf("expected a JSON object")
	}
	for key, value := range newFields {
		var oldObject, newObject map[string]json.RawMessage
		if json.Unmarshal(oldFields[key], &oldObject) == nil && oldObject != nil &&
			json.Unmarshal(value, &newObject) == nil && newObject != nil {
			merged, err := overlayJSONFields(oldFields[key], value)
			if err != nil {
				return nil, err
			}
			value = merged
		}
		oldFields[key] = value
	}
	return json.Marshal(oldFields)
}

// Fold an accepted authoritative snapshot into the legacy projection. Merging
// per-agent terminal outcomes instead of replacing the snapshot also handles
// callbacks that reach this key in a different order than currentScan.
func updateAgentScanProjection(ctx context.Context, kv scanStateStore, scanID string, meta AgentSubScanMetadata, dispatching bool) error {
	if scanID == "" {
		return fmt.Errorf("scan ID is required")
	}
	_, err := updateScanState(ctx, kv, "agent_scan:"+scanID, true, func(raw string, exists bool) (string, bool, error) {
		status := make(map[string]interface{})
		if exists {
			if err := json.Unmarshal([]byte(raw), &status); err != nil {
				return "", false, err
			}
			if status == nil {
				return "", false, fmt.Errorf("agent scan status is null")
			}
			if id, ok := status["scanId"].(string); ok && id != scanID {
				return "", false, fmt.Errorf("agent scan ID mismatch")
			}
			priorStatus, _ := status["status"].(string)
			if terminalScanStatus(priorStatus) || priorStatus == "cancelling" {
				return "", false, nil
			}
		}
		status["scanId"] = scanID
		entries := make([]interface{}, 0)
		indices := make(map[string]int)
		if existing, ok := status["agentStatuses"]; ok {
			rows, ok := existing.([]interface{})
			if !ok {
				return "", false, fmt.Errorf("invalid agentStatuses")
			}
			for _, row := range rows {
				entry, ok := row.(map[string]interface{})
				if !ok {
					return "", false, fmt.Errorf("invalid agent status entry")
				}
				id, ok := entry["agentId"].(string)
				if !ok || id == "" {
					return "", false, fmt.Errorf("missing agentId")
				}
				if _, duplicate := indices[id]; duplicate {
					return "", false, fmt.Errorf("duplicate agentId")
				}
				indices[id] = len(entries)
				entries = append(entries, entry)
			}
		}
		ensure := func(id string) map[string]interface{} {
			if i, ok := indices[id]; ok {
				return entries[i].(map[string]interface{})
			}
			entry := map[string]interface{}{"agentId": id, "status": "pending", "hostsFound": 0, "vulnerabilitiesFound": 0}
			indices[id] = len(entries)
			entries = append(entries, entry)
			return entry
		}
		for _, id := range meta.DispatchedAgents {
			if id != "" {
				ensure(id)
			}
		}
		for _, incoming := range meta.AgentStatuses {
			if incoming.AgentID == "" {
				continue
			}
			entry := ensure(incoming.AgentID)
			prior, _ := entry["status"].(string)
			if terminalScanStatus(prior) || prior == "cancelling" {
				continue
			}
			// Old snapshots must not turn a newer running status back into pending.
			if !terminalScanStatus(incoming.Status) {
				continue
			}
			entry["status"] = incoming.Status
			entry["hostsFound"], entry["vulnerabilitiesFound"] = incoming.HostsFound, incoming.VulnerabilitiesFound
			entry["completedAt"] = incoming.CompletedAt
			if incoming.Error != "" {
				entry["error"] = incoming.Error
			} else {
				delete(entry, "error")
			}
		}
		total := len(entries)
		if prior, ok := status["totalAgents"].(float64); ok && int(prior) > total {
			total = int(prior)
		}
		statuses := make([]string, len(entries))
		for i, entry := range entries {
			statuses[i], _ = entry.(map[string]interface{})["status"].(string)
		}
		aggregate, completed, failed := aggregateScanStatuses(statuses)
		if completed < total {
			aggregate = "running"
		}
		if dispatching && (!exists || status["status"] == "dispatching") {
			aggregate = "dispatching"
		}
		status["agentStatuses"], status["totalAgents"] = entries, total
		status["completedAgents"], status["failedAgents"] = completed, failed
		status["status"] = aggregate
		if terminalScanStatus(aggregate) {
			if status["completedAt"] == nil || status["completedAt"] == "" {
				status["completedAt"] = time.Now().UTC().Format(time.RFC3339)
			}
		} else {
			delete(status, "completedAt")
		}
		updated, err := json.Marshal(status)
		if err != nil {
			return "", false, err
		}
		return string(updated), string(updated) != raw, nil
	})
	return err
}
