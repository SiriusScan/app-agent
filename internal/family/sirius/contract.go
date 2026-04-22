package sirius

// Task IDs understood by the standalone Sirius agent runtime.
const (
	TaskTemplateScan       = "sirius.template.scan"
	TaskInventoryCollect   = "sirius.inventory.collect"
	TaskConnectorSync      = "sirius.connector.sync"
	TaskConnectorStatus    = "sirius.connector.status"
	TaskConnectorReconnect = "sirius.connector.reconnect"
)

// ConnectorConfig captures the family-local settings required for the
// standalone Sirius agent connector runtime.
type ConnectorConfig struct {
	AgentID              string `json:"agent_id,omitempty"`
	HostID               string `json:"host_id,omitempty"`
	ServerAddress        string `json:"server_address,omitempty"`
	APIBaseURL           string `json:"api_base_url,omitempty"`
	PowerShellPath       string `json:"powershell_path,omitempty"`
	EnableScripting      *bool  `json:"enable_scripting,omitempty"`
	LegacyInlineCommands bool   `json:"legacy_inline_commands,omitempty"`
}

// TemplateTask describes a template execution request for the
// sirius.scan.template role.
type TemplateTask struct {
	TemplatePath   string `json:"template_path,omitempty"`
	Directory      string `json:"directory,omitempty"`
	RunAll         bool   `json:"run_all,omitempty"`
	Workers        int    `json:"workers,omitempty"`
	TimeoutSeconds int    `json:"timeout_seconds,omitempty"`
	Format         string `json:"format,omitempty"`
	ScanID         string `json:"scan_id,omitempty"`
}

// InventoryTask describes an inventory collection request for the
// sirius.scan.inventory role.
type InventoryTask struct {
	Scripts []string `json:"scripts,omitempty"`
	Format  string   `json:"format,omitempty"`
}

// ConnectorTask describes a hosted control request for the resident
// sirius.connector role.
type ConnectorTask struct {
	ForceReconnect bool `json:"force_reconnect,omitempty"`
}

// TaskResult is the stable JSON envelope returned by Sirius family tasks.
type TaskResult struct {
	Output string `json:"output"`
	Format string `json:"format,omitempty"`
}
