// Main JavaScript for SiriusScan Agent Manager Dashboard

// State management
const appState = {
  websocket: null,
  connected: false,
  agents: [],
  commands: [],
  activities: [],
  lastUpdate: null,
  selectedAgent: "all",
  selectedCommandType: "custom",
  refreshInterval: null,
  statusCheckInterval: null,
};

// DOM Elements
const elements = {
  statusIndicator: document.querySelector(".status-indicator"),
  statusDot: document.querySelector(".status-dot"),
  statusText: document.querySelector(".status-text"),
  lastUpdateTime: document.querySelector(".last-update-time"),
  refreshBtn: document.querySelector(".refresh-btn"),
  serverStatus: document.getElementById("server-status"),
  uptime: document.getElementById("uptime"),
  serverVersion: document.getElementById("server-version"),
  totalAgents: document.getElementById("total-agents"),
  activeAgents: document.getElementById("active-agents"),
  totalCommands: document.getElementById("total-commands"),
  completedCommands: document.getElementById("completed-commands"),
  activityFeed: document.querySelector(".activity-feed"),
  commandHistory: document.querySelector(".command-history"),
  commandForm: document.getElementById("command-form"),
  agentSelect: document.getElementById("agent-select"),
  commandTypeSelect: document.getElementById("command-type"),
  customCommandInput: document.getElementById("custom-command"),
  predefinedCommandSelect: document.getElementById("predefined-command"),
  commandArguments: document.getElementById("command-arguments"),
  submitBtn: document.getElementById("submit-command"),
};

// Initialize the application
function initApp() {
  // Set up event listeners
  setupEventListeners();

  // Connect to WebSocket
  connectWebSocket();

  // Start intervals
  startIntervals();

  // Initial data fetch
  fetchInitialData();
}

// Setup event listeners for user interactions
function setupEventListeners() {
  // Refresh button
  elements.refreshBtn.addEventListener("click", refreshDashboard);

  // Command form
  elements.commandForm.addEventListener("submit", handleCommandSubmit);

  // Command type selection
  elements.commandTypeSelect.addEventListener("change", toggleCommandInputs);

  // Agent selection change
  elements.agentSelect.addEventListener("change", function (e) {
    appState.selectedAgent = e.target.value;
  });
}

// Connect to the WebSocket server
function connectWebSocket() {
  updateConnectionStatus("connecting");

  const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
  const wsUrl = `${protocol}//${window.location.host}/api/ws`;

  appState.websocket = new WebSocket(wsUrl);

  // WebSocket event handlers
  appState.websocket.onopen = handleWebSocketOpen;
  appState.websocket.onmessage = handleWebSocketMessage;
  appState.websocket.onclose = handleWebSocketClose;
  appState.websocket.onerror = handleWebSocketError;
}

// Handle WebSocket connection open
function handleWebSocketOpen() {
  console.log("WebSocket connection established");
  updateConnectionStatus("connected");
  appState.connected = true;

  // Request initial state
  sendWebSocketMessage({
    type: "subscribe",
    data: { topics: ["agents", "commands", "activity"] },
  });
}

// Handle WebSocket messages
function handleWebSocketMessage(event) {
  try {
    const message = JSON.parse(event.data);
    console.log("WebSocket message received:", message);

    switch (message.type) {
      case "agents":
        updateAgents(message.data);
        break;
      case "commands":
        updateCommands(message.data);
        break;
      case "activity":
        updateActivity(message.data);
        break;
      case "system":
        updateSystemStatus(message.data);
        break;
      case "error":
        showError(message.data);
        break;
      default:
        console.warn("Unknown message type:", message.type);
    }

    updateLastUpdateTime();
  } catch (error) {
    console.error("Error processing WebSocket message:", error);
  }
}

// Handle WebSocket connection close
function handleWebSocketClose(event) {
  console.log("WebSocket connection closed:", event);
  updateConnectionStatus("disconnected");
  appState.connected = false;

  // Attempt to reconnect after delay
  setTimeout(connectWebSocket, 5000);
}

// Handle WebSocket connection error
function handleWebSocketError(error) {
  console.error("WebSocket error:", error);
  updateConnectionStatus("error");
}

// Send a message through the WebSocket
function sendWebSocketMessage(message) {
  if (appState.websocket && appState.websocket.readyState === WebSocket.OPEN) {
    appState.websocket.send(JSON.stringify(message));
    return true;
  } else {
    console.warn("WebSocket not connected, cannot send message");
    return false;
  }
}

// Update the connection status in the UI
function updateConnectionStatus(status) {
  elements.statusIndicator.className = "status-indicator " + status;

  switch (status) {
    case "connected":
      elements.statusText.textContent = "Connected";
      elements.statusDot.style.backgroundColor = "var(--success-color)";
      break;
    case "connecting":
      elements.statusText.textContent = "Connecting...";
      elements.statusDot.style.backgroundColor = "var(--warning-color)";
      break;
    case "disconnected":
    case "error":
      elements.statusText.textContent = "Disconnected";
      elements.statusDot.style.backgroundColor = "var(--danger-color)";
      break;
  }
}

// Start periodic intervals for data refresh and status checking
function startIntervals() {
  // Clear any existing intervals
  if (appState.refreshInterval) clearInterval(appState.refreshInterval);
  if (appState.statusCheckInterval) clearInterval(appState.statusCheckInterval);

  // Refresh dashboard data every 30 seconds
  appState.refreshInterval = setInterval(refreshDashboard, 30000);

  // Check WebSocket status every 10 seconds
  appState.statusCheckInterval = setInterval(checkWebSocketStatus, 10000);
}

// Check WebSocket status and reconnect if needed
function checkWebSocketStatus() {
  if (!appState.websocket || appState.websocket.readyState !== WebSocket.OPEN) {
    console.log("WebSocket not connected, attempting to reconnect...");
    connectWebSocket();
  }
}

// Fetch initial dashboard data from REST API as fallback
function fetchInitialData() {
  fetchAgents();
  fetchCommands();
  fetchActivity();
  fetchSystemStatus();
}

// Refresh all dashboard data
function refreshDashboard() {
  if (appState.connected) {
    sendWebSocketMessage({ type: "refresh" });
  } else {
    fetchInitialData();
  }
}

// Fetch agents data from REST API
function fetchAgents() {
  showLoading(elements.totalAgents.parentElement);

  fetch("/api/agents")
    .then((response) => response.json())
    .then((data) => {
      updateAgents(data);
      hideLoading(elements.totalAgents.parentElement);
    })
    .catch((error) => {
      console.error("Error fetching agents:", error);
      hideLoading(elements.totalAgents.parentElement);
    });
}

// Fetch commands data from REST API
function fetchCommands() {
  showLoading(elements.totalCommands.parentElement);

  fetch("/api/commands")
    .then((response) => response.json())
    .then((data) => {
      updateCommands(data);
      hideLoading(elements.totalCommands.parentElement);
    })
    .catch((error) => {
      console.error("Error fetching commands:", error);
      hideLoading(elements.totalCommands.parentElement);
    });
}

// Fetch activity data from REST API
function fetchActivity() {
  showLoading(elements.activityFeed);

  fetch("/api/activity")
    .then((response) => response.json())
    .then((data) => {
      updateActivity(data);
      hideLoading(elements.activityFeed);
    })
    .catch((error) => {
      console.error("Error fetching activity:", error);
      hideLoading(elements.activityFeed);
    });
}

// Fetch system status from REST API
function fetchSystemStatus() {
  fetch("/api/status")
    .then((response) => response.json())
    .then((data) => {
      updateSystemStatus(data);
    })
    .catch((error) => {
      console.error("Error fetching system status:", error);
    });
}

// Update agents data in the UI
function updateAgents(agents) {
  appState.agents = agents;

  // Update statistics
  elements.totalAgents.textContent = agents.length;
  elements.activeAgents.textContent = agents.filter(
    (a) => a.status === "online"
  ).length;

  // Update agent select dropdown
  updateAgentSelect();
}

// Update agent select dropdown options
function updateAgentSelect() {
  // Save current selection
  const currentSelection = elements.agentSelect.value;

  // Clear existing options except the "All Agents" option
  while (elements.agentSelect.options.length > 1) {
    elements.agentSelect.remove(1);
  }

  // Add agent options
  appState.agents.forEach((agent) => {
    const option = document.createElement("option");
    option.value = agent.id;
    option.textContent = `${agent.name} (${agent.status})`;
    if (agent.status !== "online") {
      option.disabled = true;
    }
    elements.agentSelect.appendChild(option);
  });

  // Restore selection if possible
  if (
    currentSelection &&
    Array.from(elements.agentSelect.options).some(
      (opt) => opt.value === currentSelection
    )
  ) {
    elements.agentSelect.value = currentSelection;
  }
}

// Update commands data in the UI
function updateCommands(commands) {
  appState.commands = commands;

  // Update statistics
  elements.totalCommands.textContent = commands.length;
  elements.completedCommands.textContent = commands.filter(
    (c) => c.status === "completed"
  ).length;

  // Update command history
  updateCommandHistory();
}

// Update command history in the UI
function updateCommandHistory() {
  // Clear existing content
  elements.commandHistory.innerHTML = "";

  if (appState.commands.length === 0) {
    elements.commandHistory.innerHTML =
      '<div class="loading-indicator">No command history available</div>';
    return;
  }

  // Sort commands by timestamp (most recent first)
  const recentCommands = [...appState.commands]
    .sort((a, b) => new Date(b.timestamp) - new Date(a.timestamp))
    .slice(0, 10);

  // Add each command to history
  recentCommands.forEach((command) => {
    const historyItem = document.createElement("div");
    historyItem.className = `history-item ${command.status}`;

    const historyHeader = document.createElement("div");
    historyHeader.className = "history-header";

    const commandText = document.createElement("div");
    commandText.className = "history-command";
    commandText.textContent = command.command;

    const statusBadge = document.createElement("span");
    statusBadge.className = `history-status ${command.status}`;
    statusBadge.textContent = capitalizeFirstLetter(command.status);

    historyHeader.appendChild(commandText);
    historyHeader.appendChild(statusBadge);

    const agentInfo = document.createElement("div");
    agentInfo.className = "history-agent";
    agentInfo.textContent = `Agent: ${command.agent_name || command.agent_id}`;

    const timeInfo = document.createElement("div");
    timeInfo.className = "history-time";
    timeInfo.textContent = formatTimestamp(command.timestamp);

    historyItem.appendChild(historyHeader);
    historyItem.appendChild(agentInfo);
    historyItem.appendChild(timeInfo);

    elements.commandHistory.appendChild(historyItem);
  });
}

// Update activity feed in the UI
function updateActivity(activities) {
  appState.activities = activities;

  // Clear existing content
  elements.activityFeed.innerHTML = "";

  if (activities.length === 0) {
    elements.activityFeed.innerHTML =
      '<div class="loading-indicator">No activity data available</div>';
    return;
  }

  // Sort activities by timestamp (most recent first)
  const recentActivities = [...activities]
    .sort((a, b) => new Date(b.timestamp) - new Date(a.timestamp))
    .slice(0, 10);

  // Add each activity to feed
  recentActivities.forEach((activity) => {
    const activityItem = document.createElement("div");
    activityItem.className = "activity-item";

    const iconType = getActivityIconType(activity.type);

    const activityIcon = document.createElement("div");
    activityIcon.className = `activity-icon ${iconType}`;
    activityIcon.innerHTML = `<i class="${getActivityIcon(
      activity.type
    )}"></i>`;

    const activityContent = document.createElement("div");
    activityContent.className = "activity-content";

    const activityHeader = document.createElement("div");
    activityHeader.className = "activity-header";

    const activityTitle = document.createElement("div");
    activityTitle.className = "activity-title";
    activityTitle.textContent = activity.title || getActivityTitle(activity);

    const activityTime = document.createElement("div");
    activityTime.className = "activity-time";
    activityTime.textContent = formatTimestamp(activity.timestamp);

    activityHeader.appendChild(activityTitle);
    activityHeader.appendChild(activityTime);

    const activityDescription = document.createElement("div");
    activityDescription.className = "activity-description";
    activityDescription.textContent = activity.description;

    const activityMeta = document.createElement("div");
    activityMeta.className = "activity-meta";
    activityMeta.textContent = getActivityMeta(activity);

    activityContent.appendChild(activityHeader);
    activityContent.appendChild(activityDescription);
    activityContent.appendChild(activityMeta);

    activityItem.appendChild(activityIcon);
    activityItem.appendChild(activityContent);

    elements.activityFeed.appendChild(activityItem);
  });
}

// Update system status in the UI
function updateSystemStatus(status) {
  elements.serverStatus.textContent = status.status || "Unknown";
  elements.serverStatus.className =
    status.status === "online" ? "status-value online" : "status-value offline";

  elements.uptime.textContent = formatDuration(status.uptime) || "Unknown";
  elements.serverVersion.textContent = status.version || "Unknown";
}

// Update the last update time in the UI
function updateLastUpdateTime() {
  appState.lastUpdate = new Date();
  elements.lastUpdateTime.textContent = formatTimestamp(appState.lastUpdate);
}

// Toggle between custom and predefined command inputs
function toggleCommandInputs() {
  const commandType = elements.commandTypeSelect.value;
  appState.selectedCommandType = commandType;

  const customCommandGroup = document.getElementById("custom-command-group");
  const predefinedCommandGroup = document.getElementById(
    "predefined-command-group"
  );
  const argumentsGroup = document.getElementById("arguments-group");

  if (commandType === "custom") {
    customCommandGroup.classList.remove("hidden");
    predefinedCommandGroup.classList.add("hidden");
    argumentsGroup.classList.add("hidden");
  } else {
    customCommandGroup.classList.add("hidden");
    predefinedCommandGroup.classList.remove("hidden");
    argumentsGroup.classList.remove("hidden");
  }
}

// Handle command form submission
function handleCommandSubmit(event) {
  event.preventDefault();

  const agentId = elements.agentSelect.value;
  const commandType = elements.commandTypeSelect.value;

  let command, args;

  if (commandType === "custom") {
    command = elements.customCommandInput.value.trim();
    args = "";
  } else {
    command = elements.predefinedCommandSelect.value;
    args = elements.commandArguments.value.trim();
  }

  if (!command) {
    showError("Please enter a command");
    return;
  }

  // Create command object
  const commandData = {
    agent_id: agentId === "all" ? null : agentId,
    command: command,
    args: args,
  };

  // Send command
  submitCommand(commandData);
}

// Submit a command to the server
function submitCommand(commandData) {
  const submitMethod = appState.connected
    ? submitCommandWebSocket
    : submitCommandREST;

  // Disable submit button
  elements.submitBtn.disabled = true;

  submitMethod(commandData)
    .then((response) => {
      console.log("Command submitted successfully:", response);

      // Clear form
      if (appState.selectedCommandType === "custom") {
        elements.customCommandInput.value = "";
      } else {
        elements.commandArguments.value = "";
      }

      // Show success message
      showMessage(`Command sent to ${commandData.agent_id || "all agents"}`);

      // Refresh command history after a short delay
      setTimeout(() => {
        if (appState.connected) {
          sendWebSocketMessage({
            type: "refresh",
            data: { topics: ["commands", "activity"] },
          });
        } else {
          fetchCommands();
          fetchActivity();
        }
      }, 1000);
    })
    .catch((error) => {
      console.error("Error submitting command:", error);
      showError(`Failed to send command: ${error.message || "Unknown error"}`);
    })
    .finally(() => {
      // Re-enable submit button
      elements.submitBtn.disabled = false;
    });
}

// Submit command via WebSocket
function submitCommandWebSocket(commandData) {
  return new Promise((resolve, reject) => {
    const success = sendWebSocketMessage({
      type: "command",
      data: commandData,
    });

    if (success) {
      resolve({ success: true });
    } else {
      reject(new Error("WebSocket not connected"));
    }
  });
}

// Submit command via REST API
function submitCommandREST(commandData) {
  return fetch("/api/commands", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify(commandData),
  }).then((response) => {
    if (!response.ok) {
      throw new Error(`HTTP error ${response.status}`);
    }
    return response.json();
  });
}

// Show loading indicator in a container
function showLoading(container) {
  // Save current content
  container.dataset.originalContent = container.innerHTML;

  // Show loading indicator
  container.innerHTML =
    '<div class="loading-indicator"><i class="fas fa-spinner fa-spin"></i> Loading...</div>';
}

// Hide loading indicator and restore content
function hideLoading(container) {
  if (container.dataset.originalContent) {
    container.innerHTML = container.dataset.originalContent;
    delete container.dataset.originalContent;
  }
}

// Show error message
function showError(message) {
  // In a real app, you would show a toast or notification
  console.error("Error:", message);
  alert(`Error: ${message}`);
}

// Show success or info message
function showMessage(message) {
  // In a real app, you would show a toast or notification
  console.log("Message:", message);
  // Implementation depends on your UI library/framework
}

// Helper functions
function formatTimestamp(timestamp) {
  if (!timestamp) return "Unknown";

  const date = new Date(timestamp);

  // Check if the date is valid
  if (isNaN(date.getTime())) return "Invalid date";

  // If today, show only time
  const now = new Date();
  if (date.toDateString() === now.toDateString()) {
    return date.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });
  }

  // Otherwise show date and time
  return date.toLocaleString([], {
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
}

function formatDuration(seconds) {
  if (!seconds || isNaN(seconds)) return "Unknown";

  const days = Math.floor(seconds / 86400);
  const hours = Math.floor((seconds % 86400) / 3600);
  const minutes = Math.floor((seconds % 3600) / 60);

  if (days > 0) {
    return `${days}d ${hours}h ${minutes}m`;
  } else if (hours > 0) {
    return `${hours}h ${minutes}m`;
  } else {
    return `${minutes}m`;
  }
}

function capitalizeFirstLetter(string) {
  if (!string) return "";
  return string.charAt(0).toUpperCase() + string.slice(1);
}

function getActivityIconType(type) {
  switch (type) {
    case "agent_connected":
    case "agent_disconnected":
    case "agent_registered":
      return "agent";
    case "command_sent":
    case "command_completed":
    case "command_failed":
      return "command";
    case "error":
      return "error";
    default:
      return "";
  }
}

function getActivityIcon(type) {
  switch (type) {
    case "agent_connected":
    case "agent_registered":
      return "fas fa-server";
    case "agent_disconnected":
      return "fas fa-unlink";
    case "command_sent":
      return "fas fa-terminal";
    case "command_completed":
      return "fas fa-check-circle";
    case "command_failed":
      return "fas fa-times-circle";
    case "error":
      return "fas fa-exclamation-triangle";
    default:
      return "fas fa-info-circle";
  }
}

function getActivityTitle(activity) {
  switch (activity.type) {
    case "agent_connected":
      return "Agent Connected";
    case "agent_disconnected":
      return "Agent Disconnected";
    case "agent_registered":
      return "New Agent Registered";
    case "command_sent":
      return "Command Sent";
    case "command_completed":
      return "Command Completed";
    case "command_failed":
      return "Command Failed";
    case "error":
      return "System Error";
    default:
      return capitalizeFirstLetter(activity.type.replace(/_/g, " "));
  }
}

function getActivityMeta(activity) {
  if (activity.agent_id) {
    const agent = appState.agents.find((a) => a.id === activity.agent_id);
    return `Agent: ${agent ? agent.name : activity.agent_id}`;
  }
  return "";
}

// Initialize the application when the DOM is fully loaded
document.addEventListener("DOMContentLoaded", initApp);
