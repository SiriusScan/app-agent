package fingerprint

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"go.uber.org/zap"
)

// UserCollectorImpl implements UserCollector interface
type UserCollectorImpl struct {
	logger  *zap.Logger
	timeout time.Duration
}

// NewUserCollector creates a new user collector
func NewUserCollector(logger *zap.Logger) UserCollector {
	return &UserCollectorImpl{
		logger:  logger,
		timeout: 30 * time.Second,
	}
}

// GetLocalUsers retrieves local user accounts
func (uc *UserCollectorImpl) GetLocalUsers(ctx context.Context) ([]*UserAccount, error) {
	uc.logger.Info("Starting user account enumeration", zap.String("platform", runtime.GOOS))

	timeoutCtx, cancel := context.WithTimeout(ctx, uc.timeout)
	defer cancel()

	var users []*UserAccount
	var err error

	switch runtime.GOOS {
	case "windows":
		users, err = uc.collectWindowsUsers(timeoutCtx)
	case "linux":
		users, err = uc.collectLinuxUsers(timeoutCtx)
	case "darwin":
		users, err = uc.collectMacOSUsers(timeoutCtx)
	default:
		return nil, fmt.Errorf("user collection not supported on platform: %s", runtime.GOOS)
	}

	if err != nil {
		uc.logger.Error("User collection failed", zap.Error(err))
		return nil, fmt.Errorf("failed to collect users: %w", err)
	}

	uc.logger.Info("User enumeration completed", zap.Int("total_users", len(users)))
	return users, nil
}

// GetLocalGroups retrieves local groups
func (uc *UserCollectorImpl) GetLocalGroups(ctx context.Context) ([]*UserGroup, error) {
	uc.logger.Info("Starting group enumeration", zap.String("platform", runtime.GOOS))

	timeoutCtx, cancel := context.WithTimeout(ctx, uc.timeout)
	defer cancel()

	var groups []*UserGroup
	var err error

	switch runtime.GOOS {
	case "windows":
		groups, err = uc.collectWindowsGroups(timeoutCtx)
	case "linux":
		groups, err = uc.collectLinuxGroups(timeoutCtx)
	case "darwin":
		groups, err = uc.collectMacOSGroups(timeoutCtx)
	default:
		return nil, fmt.Errorf("group collection not supported on platform: %s", runtime.GOOS)
	}

	if err != nil {
		uc.logger.Error("Group collection failed", zap.Error(err))
		return nil, fmt.Errorf("failed to collect groups: %w", err)
	}

	uc.logger.Info("Group enumeration completed", zap.Int("total_groups", len(groups)))
	return groups, nil
}

// GetCurrentUserPrivileges retrieves current user privileges
func (uc *UserCollectorImpl) GetCurrentUserPrivileges(ctx context.Context) (*UserPrivileges, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, uc.timeout)
	defer cancel()

	switch runtime.GOOS {
	case "windows":
		return uc.getWindowsCurrentUserPrivileges(timeoutCtx)
	case "linux":
		return uc.getLinuxCurrentUserPrivileges(timeoutCtx)
	case "darwin":
		return uc.getMacOSCurrentUserPrivileges(timeoutCtx)
	default:
		return nil, fmt.Errorf("privilege collection not supported on platform: %s", runtime.GOOS)
	}
}

// GetLoginSessions retrieves active login sessions
func (uc *UserCollectorImpl) GetLoginSessions(ctx context.Context) ([]*LoginSession, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, uc.timeout)
	defer cancel()

	switch runtime.GOOS {
	case "windows":
		return uc.getWindowsLoginSessions(timeoutCtx)
	case "linux":
		return uc.getLinuxLoginSessions(timeoutCtx)
	case "darwin":
		return uc.getMacOSLoginSessions(timeoutCtx)
	default:
		return nil, fmt.Errorf("session collection not supported on platform: %s", runtime.GOOS)
	}
}

// Windows implementations
func (uc *UserCollectorImpl) collectWindowsUsers(ctx context.Context) ([]*UserAccount, error) {
	var users []*UserAccount

	psScript := `Get-LocalUser | ConvertTo-Json`
	cmd := exec.CommandContext(ctx, "powershell", "-Command", psScript)
	output, err := cmd.Output()
	if err != nil {
		return uc.collectWindowsUsersFallback(ctx)
	}

	var userData []map[string]interface{}
	if err := json.Unmarshal(output, &userData); err != nil {
		return uc.collectWindowsUsersFallback(ctx)
	}

	for _, data := range userData {
		user := &UserAccount{
			Username:    getString(data, "Name"),
			FullName:    getString(data, "FullName"),
			UID:         getString(data, "SID"),
			Enabled:     getBool(data, "Enabled"),
			AccountType: "local",
		}
		users = append(users, user)
	}

	return users, nil
}

func (uc *UserCollectorImpl) collectWindowsUsersFallback(ctx context.Context) ([]*UserAccount, error) {
	var users []*UserAccount

	cmd := exec.CommandContext(ctx, "net", "user")
	output, err := cmd.Output()
	if err != nil {
		return users, err
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.Contains(line, "---") && !strings.Contains(line, "User accounts") {
			usernames := strings.Fields(line)
			for _, username := range usernames {
				if username != "" {
					users = append(users, &UserAccount{
						Username:    username,
						AccountType: "local",
						Enabled:     true,
					})
				}
			}
		}
	}

	return users, nil
}

func (uc *UserCollectorImpl) collectWindowsGroups(ctx context.Context) ([]*UserGroup, error) {
	var groups []*UserGroup

	psScript := `Get-LocalGroup | ConvertTo-Json`
	cmd := exec.CommandContext(ctx, "powershell", "-Command", psScript)
	output, err := cmd.Output()
	if err != nil {
		return groups, nil
	}

	var groupData []map[string]interface{}
	if err := json.Unmarshal(output, &groupData); err != nil {
		return groups, nil
	}

	for _, data := range groupData {
		group := &UserGroup{
			Name:        getString(data, "Name"),
			GID:         getString(data, "SID"),
			Description: getString(data, "Description"),
			Type:        "local",
		}
		groups = append(groups, group)
	}

	return groups, nil
}

// Linux implementations
func (uc *UserCollectorImpl) collectLinuxUsers(ctx context.Context) ([]*UserAccount, error) {
	var users []*UserAccount

	cmd := exec.CommandContext(ctx, "cat", "/etc/passwd")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		fields := strings.Split(line, ":")
		if len(fields) >= 7 {
			user := &UserAccount{
				Username:      fields[0],
				FullName:      strings.Split(fields[4], ",")[0],
				UID:           fields[2],
				GID:           fields[3],
				HomeDirectory: fields[5],
				Shell:         fields[6],
				AccountType:   "local",
				Enabled:       !strings.Contains(fields[6], "nologin") && !strings.Contains(fields[6], "false"),
			}
			users = append(users, user)
		}
	}

	return users, nil
}

func (uc *UserCollectorImpl) collectLinuxGroups(ctx context.Context) ([]*UserGroup, error) {
	var groups []*UserGroup

	cmd := exec.CommandContext(ctx, "cat", "/etc/group")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		fields := strings.Split(line, ":")
		if len(fields) >= 4 {
			group := &UserGroup{
				Name: fields[0],
				GID:  fields[2],
				Type: "local",
			}

			if fields[3] != "" {
				group.Members = strings.Split(fields[3], ",")
			}

			groups = append(groups, group)
		}
	}

	return groups, nil
}

// macOS implementations
func (uc *UserCollectorImpl) collectMacOSUsers(ctx context.Context) ([]*UserAccount, error) {
	var users []*UserAccount

	cmd := exec.CommandContext(ctx, "dscl", ".", "list", "/Users")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		username := strings.TrimSpace(line)
		if username == "" || strings.HasPrefix(username, "_") {
			continue
		}

		user := &UserAccount{
			Username:    username,
			AccountType: "local",
			Enabled:     true,
		}
		users = append(users, user)
	}

	return users, nil
}

func (uc *UserCollectorImpl) collectMacOSGroups(ctx context.Context) ([]*UserGroup, error) {
	var groups []*UserGroup

	cmd := exec.CommandContext(ctx, "dscl", ".", "list", "/Groups")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		groupName := strings.TrimSpace(line)
		if groupName == "" {
			continue
		}

		group := &UserGroup{
			Name: groupName,
			Type: "local",
		}
		groups = append(groups, group)
	}

	return groups, nil
}

// Privilege methods
func (uc *UserCollectorImpl) getWindowsCurrentUserPrivileges(ctx context.Context) (*UserPrivileges, error) {
	privileges := &UserPrivileges{
		Username: "current",
		Groups:   []string{},
	}

	if output, err := exec.CommandContext(ctx, "whoami").Output(); err == nil {
		privileges.Username = strings.TrimSpace(string(output))
	}

	// Check admin by trying to access admin-only command
	if _, err := exec.CommandContext(ctx, "net", "session").Output(); err == nil {
		privileges.IsAdmin = true
	}

	return privileges, nil
}

func (uc *UserCollectorImpl) getLinuxCurrentUserPrivileges(ctx context.Context) (*UserPrivileges, error) {
	privileges := &UserPrivileges{
		Username: "current",
		Groups:   []string{},
	}

	if output, err := exec.CommandContext(ctx, "whoami").Output(); err == nil {
		privileges.Username = strings.TrimSpace(string(output))
	}

	// Check sudo access
	if _, err := exec.CommandContext(ctx, "sudo", "-n", "true").Output(); err == nil {
		privileges.SudoAccess = true
		privileges.IsAdmin = true
	}

	return privileges, nil
}

func (uc *UserCollectorImpl) getMacOSCurrentUserPrivileges(ctx context.Context) (*UserPrivileges, error) {
	return uc.getLinuxCurrentUserPrivileges(ctx)
}

// Session methods
func (uc *UserCollectorImpl) getWindowsLoginSessions(ctx context.Context) ([]*LoginSession, error) {
	return []*LoginSession{}, nil
}

func (uc *UserCollectorImpl) getLinuxLoginSessions(ctx context.Context) ([]*LoginSession, error) {
	var sessions []*LoginSession

	cmd := exec.CommandContext(ctx, "who")
	output, err := cmd.Output()
	if err != nil {
		return sessions, nil
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) >= 3 {
			session := &LoginSession{
				Username:    fields[0],
				SessionType: "console",
				Status:      "active",
			}
			sessions = append(sessions, session)
		}
	}

	return sessions, nil
}

func (uc *UserCollectorImpl) getMacOSLoginSessions(ctx context.Context) ([]*LoginSession, error) {
	return uc.getLinuxLoginSessions(ctx)
}

// Helper functions
func getBool(data map[string]interface{}, key string) bool {
	if val, ok := data[key]; ok {
		if b, ok := val.(bool); ok {
			return b
		}
	}
	return false
}
