package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/SiriusScan/app-agent/internal/agent"
	"github.com/SiriusScan/app-agent/internal/store"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Constants for the dashboard
const (
	AgentStatusUpdate   = "agent_status"
	CommandStatusUpdate = "command_status"
	SystemStatusUpdate  = "system_status"
)

// AgentInfo represents agent information for display
type AgentInfo struct {
	ID             string            `json:"id"`
	Status         string            `json:"status"`
	LastSeen       string            `json:"last_seen"`
	LastSeenRaw    int64             `json:"last_seen_raw"`
	ActiveCommands int               `json:"active_commands"`
	CPU            float64           `json:"cpu"`
	Memory         float64           `json:"memory"`
	Metadata       map[string]string `json:"metadata"`
}

// CommandInfo represents command information for display
type CommandInfo struct {
	ID             string            `json:"id"`
	Type           string            `json:"type"`
	AgentID        string            `json:"agent_id"`
	Status         string            `json:"status"`
	CreatedAt      string            `json:"created_at"`
	CreatedAtRaw   int64             `json:"created_at_raw"`
	CompletedAt    string            `json:"completed_at"`
	CompletedAtRaw int64             `json:"completed_at_raw"`
	Args           map[string]string `json:"args"`
	Result         map[string]string `json:"result,omitempty"`
	Error          string            `json:"error,omitempty"`
}

// SystemStatus represents the overall system status
type SystemStatus struct {
	AgentCount         int       `json:"agent_count"`
	ActiveAgents       int       `json:"active_agents"`
	TotalCommands      int       `json:"total_commands"`
	CompletedCommands  int       `json:"completed_commands"`
	PendingCommands    int       `json:"pending_commands"`
	FailedCommands     int       `json:"failed_commands"`
	LastUpdateTime     time.Time `json:"last_update_time"`
	LastUpdateTimeStr  string    `json:"last_update_time_str"`
	SystemStartTime    time.Time `json:"system_start_time"`
	SystemStartTimeStr string    `json:"system_start_time_str"`
	Uptime             string    `json:"uptime"`
}

// DashboardServer represents the web dashboard server
type DashboardServer struct {
	store       store.KVStore
	logger      *zap.Logger
	addr        string
	upgrader    websocket.Upgrader
	clients     map[*websocket.Conn]bool
	clientsLock sync.Mutex
	startTime   time.Time
}

// NewDashboardServer creates a new dashboard server
func NewDashboardServer(store store.KVStore, logger *zap.Logger, addr string) *DashboardServer {
	return &DashboardServer{
		store:  store,
		logger: logger,
		addr:   addr,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins for demonstration
			},
		},
		clients:   make(map[*websocket.Conn]bool),
		startTime: time.Now(),
	}
}

// Start starts the dashboard server
func (s *DashboardServer) Start(ctx context.Context) error {
	// Set up routes
	mux := http.NewServeMux()

	// Serve static files
	fileServer := http.FileServer(http.Dir("cmd/status-dashboard/static"))
	mux.Handle("/static/", http.StripPrefix("/static/", fileServer))

	// Serve template files
	mux.HandleFunc("/", s.handleIndex)
	mux.HandleFunc("/agents", s.handleAgents)
	mux.HandleFunc("/commands", s.handleCommands)
	mux.HandleFunc("/ws", s.handleWebsocket)
	mux.HandleFunc("/api/agents", s.handleAPIAgents)
	mux.HandleFunc("/api/commands", s.handleAPICommands)
	mux.HandleFunc("/api/system", s.handleAPISystem)
	mux.HandleFunc("/api/command/send", s.handleAPISendCommand)

	// Start the server
	server := &http.Server{
		Addr:    s.addr,
		Handler: mux,
	}

	// Run the server in a goroutine
	go func() {
		s.logger.Info("Starting dashboard server", zap.String("addr", s.addr))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Error("Server error", zap.Error(err))
		}
	}()

	// Start the update broadcaster
	go s.broadcastUpdates(ctx)

	// Wait for context cancellation
	<-ctx.Done()
	s.logger.Info("Shutting down dashboard server")

	// Create a shutdown context with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Close all websocket connections
	s.clientsLock.Lock()
	for conn := range s.clients {
		conn.Close()
	}
	s.clientsLock.Unlock()

	// Shutdown the server
	return server.Shutdown(shutdownCtx)
}

// handleIndex handles the index page
func (s *DashboardServer) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	tmpl, err := template.ParseFiles("cmd/status-dashboard/templates/index.html")
	if err != nil {
		s.logger.Error("Failed to parse template", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	systemStatus, err := s.getSystemStatus(r.Context())
	if err != nil {
		s.logger.Error("Failed to get system status", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if err := tmpl.Execute(w, systemStatus); err != nil {
		s.logger.Error("Failed to execute template", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

// handleAgents handles the agents page
func (s *DashboardServer) handleAgents(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("cmd/status-dashboard/templates/agents.html")
	if err != nil {
		s.logger.Error("Failed to parse template", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	agents, err := s.getAgents(r.Context())
	if err != nil {
		s.logger.Error("Failed to get agents", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	data := struct {
		Agents []AgentInfo
	}{
		Agents: agents,
	}

	if err := tmpl.Execute(w, data); err != nil {
		s.logger.Error("Failed to execute template", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

// handleCommands handles the commands page
func (s *DashboardServer) handleCommands(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("cmd/status-dashboard/templates/commands.html")
	if err != nil {
		s.logger.Error("Failed to parse template", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	commands, err := s.getCommands(r.Context())
	if err != nil {
		s.logger.Error("Failed to get commands", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	data := struct {
		Commands []CommandInfo
	}{
		Commands: commands,
	}

	if err := tmpl.Execute(w, data); err != nil {
		s.logger.Error("Failed to execute template", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

// handleWebsocket handles websocket connections
func (s *DashboardServer) handleWebsocket(w http.ResponseWriter, r *http.Request) {
	// Upgrade the HTTP connection to a WebSocket connection
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.logger.Error("Failed to upgrade to websocket", zap.Error(err))
		return
	}

	// Register the client
	s.clientsLock.Lock()
	s.clients[conn] = true
	s.clientsLock.Unlock()

	s.logger.Info("New websocket client connected")

	// Handle client disconnect
	go func() {
		defer func() {
			// Unregister the client when the connection is closed
			s.clientsLock.Lock()
			delete(s.clients, conn)
			s.clientsLock.Unlock()
			conn.Close()
			s.logger.Info("Websocket client disconnected")
		}()

		// Simple message processing loop
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					s.logger.Warn("Websocket error", zap.Error(err))
				}
				break
			}
		}
	}()
}

// handleAPIAgents handles API requests for agent information
func (s *DashboardServer) handleAPIAgents(w http.ResponseWriter, r *http.Request) {
	agents, err := s.getAgents(r.Context())
	if err != nil {
		s.logger.Error("Failed to get agents", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(agents)
}

// handleAPICommands handles API requests for command information
func (s *DashboardServer) handleAPICommands(w http.ResponseWriter, r *http.Request) {
	commands, err := s.getCommands(r.Context())
	if err != nil {
		s.logger.Error("Failed to get commands", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(commands)
}

// handleAPISystem handles API requests for system status
func (s *DashboardServer) handleAPISystem(w http.ResponseWriter, r *http.Request) {
	status, err := s.getSystemStatus(r.Context())
	if err != nil {
		s.logger.Error("Failed to get system status", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// handleAPISendCommand handles API requests to send commands
func (s *DashboardServer) handleAPISendCommand(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse the command request
	var commandReq struct {
		AgentID string            `json:"agent_id"`
		Type    string            `json:"type"`
		Args    map[string]string `json:"args"`
	}

	if err := json.NewDecoder(r.Body).Decode(&commandReq); err != nil {
		s.logger.Error("Failed to parse command request", zap.Error(err))
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Create a task message
	task := agent.TaskMessage{
		ID:        fmt.Sprintf("task-%s", time.Now().Format("20060102-150405")),
		Type:      commandReq.Type,
		AgentID:   commandReq.AgentID,
		Args:      commandReq.Args,
		Status:    agent.TaskStatusQueued,
		UpdatedAt: time.Now(),
	}

	// Store the task
	taskData, err := json.Marshal(task)
	if err != nil {
		s.logger.Error("Failed to marshal task", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if err := s.store.SetValue(r.Context(), "task:"+task.ID, string(taskData)); err != nil {
		s.logger.Error("Failed to store task", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"task_id": task.ID,
		"message": "Command queued successfully",
	})
}

// broadcastUpdates periodically broadcasts updates to connected clients
func (s *DashboardServer) broadcastUpdates(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Get latest data
			agents, err := s.getAgents(ctx)
			if err != nil {
				s.logger.Error("Failed to get agents for broadcast", zap.Error(err))
				continue
			}

			commands, err := s.getRecentCommands(ctx, 10)
			if err != nil {
				s.logger.Error("Failed to get commands for broadcast", zap.Error(err))
				continue
			}

			systemStatus, err := s.getSystemStatus(ctx)
			if err != nil {
				s.logger.Error("Failed to get system status for broadcast", zap.Error(err))
				continue
			}

			// Broadcast agent updates
			agentUpdate := struct {
				Type   string      `json:"type"`
				Agents []AgentInfo `json:"agents"`
			}{
				Type:   AgentStatusUpdate,
				Agents: agents,
			}

			s.broadcast(agentUpdate)

			// Broadcast command updates
			commandUpdate := struct {
				Type     string        `json:"type"`
				Commands []CommandInfo `json:"commands"`
			}{
				Type:     CommandStatusUpdate,
				Commands: commands,
			}

			s.broadcast(commandUpdate)

			// Broadcast system status update
			systemUpdate := struct {
				Type   string       `json:"type"`
				Status SystemStatus `json:"status"`
			}{
				Type:   SystemStatusUpdate,
				Status: systemStatus,
			}

			s.broadcast(systemUpdate)
		}
	}
}

// broadcast sends a message to all connected websocket clients
func (s *DashboardServer) broadcast(message interface{}) {
	// Marshal the message to JSON
	data, err := json.Marshal(message)
	if err != nil {
		s.logger.Error("Failed to marshal broadcast message", zap.Error(err))
		return
	}

	// Send to all clients
	s.clientsLock.Lock()
	defer s.clientsLock.Unlock()

	for client := range s.clients {
		if err := client.WriteMessage(websocket.TextMessage, data); err != nil {
			s.logger.Warn("Failed to send message to client", zap.Error(err))
			client.Close()
			delete(s.clients, client)
		}
	}
}

// getAgents retrieves agent information from the store
func (s *DashboardServer) getAgents(ctx context.Context) ([]AgentInfo, error) {
	// This would normally query the store for agent information
	// For now, we'll return mock data
	agents := []AgentInfo{
		{
			ID:             "test-agent-1",
			Status:         "active",
			LastSeen:       time.Now().Format("2006-01-02 15:04:05"),
			LastSeenRaw:    time.Now().Unix(),
			ActiveCommands: 2,
			CPU:            32.5,
			Memory:         1256.4,
			Metadata: map[string]string{
				"version": "1.0.0",
				"os":      "linux",
			},
		},
		{
			ID:             "test-agent-2",
			Status:         "active",
			LastSeen:       time.Now().Add(-30 * time.Second).Format("2006-01-02 15:04:05"),
			LastSeenRaw:    time.Now().Add(-30 * time.Second).Unix(),
			ActiveCommands: 1,
			CPU:            15.2,
			Memory:         845.7,
			Metadata: map[string]string{
				"version": "1.0.0",
				"os":      "windows",
			},
		},
		{
			ID:             "test-agent-3",
			Status:         "disconnected",
			LastSeen:       time.Now().Add(-10 * time.Minute).Format("2006-01-02 15:04:05"),
			LastSeenRaw:    time.Now().Add(-10 * time.Minute).Unix(),
			ActiveCommands: 0,
			CPU:            0.0,
			Memory:         0.0,
			Metadata: map[string]string{
				"version": "1.0.0",
				"os":      "darwin",
			},
		},
	}

	return agents, nil
}

// getCommands retrieves command information from the store
func (s *DashboardServer) getCommands(ctx context.Context) ([]CommandInfo, error) {
	commands, err := s.getRecentCommands(ctx, 100)
	if err != nil {
		return nil, err
	}

	return commands, nil
}

// getRecentCommands retrieves the most recent commands
func (s *DashboardServer) getRecentCommands(ctx context.Context, limit int) ([]CommandInfo, error) {
	// This would normally query the store for command information
	// For now, we'll return mock data
	now := time.Now()
	commands := []CommandInfo{
		{
			ID:             "task-20240601-123456-1",
			Type:           "exec",
			AgentID:        "test-agent-1",
			Status:         "completed",
			CreatedAt:      now.Add(-5 * time.Minute).Format("2006-01-02 15:04:05"),
			CreatedAtRaw:   now.Add(-5 * time.Minute).Unix(),
			CompletedAt:    now.Add(-4 * time.Minute).Format("2006-01-02 15:04:05"),
			CompletedAtRaw: now.Add(-4 * time.Minute).Unix(),
			Args: map[string]string{
				"command": "ls -la",
				"timeout": "10",
			},
			Result: map[string]string{
				"output":    "total 24\ndrwxr-xr-x  8 user group 4096 Jun 1 12:34 .",
				"exit_code": "0",
			},
		},
		{
			ID:             "task-20240601-123457-1",
			Type:           "scan",
			AgentID:        "test-agent-2",
			Status:         "in_progress",
			CreatedAt:      now.Add(-2 * time.Minute).Format("2006-01-02 15:04:05"),
			CreatedAtRaw:   now.Add(-2 * time.Minute).Unix(),
			CompletedAt:    "",
			CompletedAtRaw: 0,
			Args: map[string]string{
				"scan_type": "vulnerability",
				"target":    "localhost",
				"depth":     "3",
				"timeout":   "60",
			},
		},
		{
			ID:             "task-20240601-123458-1",
			Type:           "status",
			AgentID:        "test-agent-1",
			Status:         "completed",
			CreatedAt:      now.Add(-10 * time.Minute).Format("2006-01-02 15:04:05"),
			CreatedAtRaw:   now.Add(-10 * time.Minute).Unix(),
			CompletedAt:    now.Add(-10 * time.Minute).Format("2006-01-02 15:04:05"),
			CompletedAtRaw: now.Add(-10 * time.Minute).Unix(),
			Args: map[string]string{
				"detail_level": "full",
			},
			Result: map[string]string{
				"cpu":    "45.2",
				"memory": "1256MB",
				"disk":   "10.5GB free",
				"uptime": "3d 12h 30m",
			},
		},
		{
			ID:             "task-20240601-123459-1",
			Type:           "exec",
			AgentID:        "test-agent-3",
			Status:         "failed",
			CreatedAt:      now.Add(-15 * time.Minute).Format("2006-01-02 15:04:05"),
			CreatedAtRaw:   now.Add(-15 * time.Minute).Unix(),
			CompletedAt:    now.Add(-14 * time.Minute).Format("2006-01-02 15:04:05"),
			CompletedAtRaw: now.Add(-14 * time.Minute).Unix(),
			Args: map[string]string{
				"command": "cat /etc/passwd",
				"timeout": "5",
			},
			Error: "agent disconnected",
		},
	}

	// Sort by creation time (most recent first)
	sort.Slice(commands, func(i, j int) bool {
		return commands[i].CreatedAtRaw > commands[j].CreatedAtRaw
	})

	// Apply limit
	if len(commands) > limit {
		commands = commands[:limit]
	}

	return commands, nil
}

// getSystemStatus retrieves system status information
func (s *DashboardServer) getSystemStatus(ctx context.Context) (SystemStatus, error) {
	// Get agents and commands
	agents, err := s.getAgents(ctx)
	if err != nil {
		return SystemStatus{}, err
	}

	commands, err := s.getCommands(ctx)
	if err != nil {
		return SystemStatus{}, err
	}

	// Calculate system status
	activeAgents := 0
	for _, agent := range agents {
		if agent.Status == "active" {
			activeAgents++
		}
	}

	completedCommands := 0
	pendingCommands := 0
	failedCommands := 0
	for _, command := range commands {
		switch command.Status {
		case "completed":
			completedCommands++
		case "queued", "in_progress":
			pendingCommands++
		case "failed", "cancelled":
			failedCommands++
		}
	}

	// Calculate uptime
	uptime := time.Since(s.startTime)
	uptimeStr := formatDuration(uptime)

	// Create system status
	status := SystemStatus{
		AgentCount:         len(agents),
		ActiveAgents:       activeAgents,
		TotalCommands:      len(commands),
		CompletedCommands:  completedCommands,
		PendingCommands:    pendingCommands,
		FailedCommands:     failedCommands,
		LastUpdateTime:     time.Now(),
		LastUpdateTimeStr:  time.Now().Format("2006-01-02 15:04:05"),
		SystemStartTime:    s.startTime,
		SystemStartTimeStr: s.startTime.Format("2006-01-02 15:04:05"),
		Uptime:             uptimeStr,
	}

	return status, nil
}

// formatDuration formats a duration as a human-readable string
func formatDuration(d time.Duration) string {
	days := int(d.Hours() / 24)
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60

	parts := []string{}
	if days > 0 {
		parts = append(parts, fmt.Sprintf("%dd", days))
	}
	if hours > 0 || days > 0 {
		parts = append(parts, fmt.Sprintf("%dh", hours))
	}
	if minutes > 0 || hours > 0 || days > 0 {
		parts = append(parts, fmt.Sprintf("%dm", minutes))
	}
	parts = append(parts, fmt.Sprintf("%ds", seconds))

	return strings.Join(parts, " ")
}

func main() {
	// Parse flags
	addr := flag.String("addr", ":5174", "HTTP service address")
	flag.Parse()

	// Set up logger
	config := zap.NewDevelopmentConfig()
	config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	config.EncoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05")
	logger, _ := config.Build()
	defer logger.Sync()

	// Create a context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up signal handling for graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		logger.Info("Shutting down gracefully...")
		cancel()
	}()

	// Create mock store
	mockStore := store.NewMockStore()

	// Create the dashboard server
	server := NewDashboardServer(mockStore, logger, *addr)

	// Start the server
	if err := server.Start(ctx); err != nil {
		logger.Fatal("Failed to start server", zap.Error(err))
	}

	logger.Info("Server shutdown complete")
}
