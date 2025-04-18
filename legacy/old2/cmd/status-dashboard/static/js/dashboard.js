document.addEventListener("DOMContentLoaded", function () {
  // Initialize the dashboard
  initializeDashboard();

  // Command type selection handler
  initializeCommandForm();

  // Set up activity refresh button
  const refreshBtn = document.querySelector(".refresh-btn");
  if (refreshBtn) {
    refreshBtn.addEventListener("click", function () {
      fetchRecentActivity();
    });
  }

  // Setup auto-refresh
  setInterval(function () {
    updateDashboardData();
  }, 30000); // Update every 30 seconds
});

function initializeDashboard() {
  // Fetch initial data
  updateDashboardData();

  // Set current year in footer if needed
  const yearElement = document.querySelector(".footer .year");
  if (yearElement) {
    yearElement.textContent = new Date().getFullYear();
  }
}

function updateDashboardData() {
  // Update dashboard statistics
  fetchDashboardStats();

  // Update activity feed
  fetchRecentActivity();

  // Update last updated time
  document.getElementById("last-update-value").textContent = formatDateTime(
    new Date()
  );
}

function fetchDashboardStats() {
  // In a real implementation, this would fetch data from the API
  // For now, we'll use mock data

  // Simulate API request
  setTimeout(() => {
    const mockData = {
      agents: {
        total: 12,
        active: 8,
        inactive: 4,
      },
      commands: {
        total: 156,
        completed: 143,
        pending: 8,
        failed: 5,
      },
      system: {
        uptime: "3d 14h 27m 18s",
        startTime: new Date(Date.now() - 3 * 24 * 60 * 60 * 1000).toISOString(),
      },
    };

    updateStats(mockData);
  }, 500);
}

function updateStats(data) {
  // Update agent stats
  document.getElementById("total-agents").textContent = data.agents.total;
  document.getElementById("active-agents").textContent = data.agents.active;
  document.getElementById("inactive-agents").textContent = data.agents.inactive;

  // Update command stats
  document.getElementById("total-commands").textContent = data.commands.total;
  document.getElementById("completed-commands").textContent =
    data.commands.completed;
  document.getElementById("pending-commands").textContent =
    data.commands.pending;
  document.getElementById("failed-commands").textContent = data.commands.failed;

  // Update system info
  document.getElementById("uptime-value").textContent = data.system.uptime;
  document.getElementById("start-time-value").textContent = formatDateTime(
    new Date(data.system.startTime)
  );
}

function fetchRecentActivity() {
  const activityFeed = document.querySelector(".activity-feed");

  // Show loading indicator
  activityFeed.innerHTML = `
        <div class="loading-indicator">
            <i class="fas fa-spinner fa-spin"></i>
            <span>Loading activity...</span>
        </div>
    `;

  // Simulate API request
  setTimeout(() => {
    // Mock activity data
    const activities = [
      {
        type: "success",
        title: "Agent Connected",
        message: "Agent srv-db-01 connected to the system",
        time: new Date(Date.now() - 5 * 60000),
        icon: "fa-server",
      },
      {
        type: "info",
        title: "Command Executed",
        message: "Vulnerability scan completed on srv-web-03",
        time: new Date(Date.now() - 15 * 60000),
        icon: "fa-terminal",
      },
      {
        type: "danger",
        title: "Command Failed",
        message: "Command execution failed on srv-app-02: Permission denied",
        time: new Date(Date.now() - 30 * 60000),
        icon: "fa-exclamation-triangle",
      },
      {
        type: "info",
        title: "Configuration Updated",
        message: "Agent srv-db-01 configuration updated successfully",
        time: new Date(Date.now() - 45 * 60000),
        icon: "fa-cog",
      },
      {
        type: "warning",
        title: "Agent Disconnected",
        message: "Agent srv-cache-01 disconnected unexpectedly",
        time: new Date(Date.now() - 60 * 60000),
        icon: "fa-exclamation-circle",
      },
    ];

    displayActivityFeed(activities);
  }, 1000);
}

function displayActivityFeed(activities) {
  const activityFeed = document.querySelector(".activity-feed");
  activityFeed.innerHTML = "";

  if (activities.length === 0) {
    activityFeed.innerHTML =
      '<div class="no-activity">No recent activity</div>';
    return;
  }

  activities.forEach((activity) => {
    const activityElement = document.createElement("div");
    activityElement.className = "activity-item fadeIn";

    activityElement.innerHTML = `
            <div class="activity-icon ${activity.type}">
                <i class="fas ${activity.icon}"></i>
            </div>
            <div class="activity-content">
                <div class="activity-title">${activity.title}</div>
                <div class="activity-details">
                    <span>${activity.message}</span>
                    <span class="activity-time">${formatTimeAgo(
                      activity.time
                    )}</span>
                </div>
            </div>
        `;

    activityFeed.appendChild(activityElement);
  });
}

function initializeCommandForm() {
  const commandForm = document.getElementById("command-form");
  const commandTypeSelect = document.getElementById("command-type");
  const commandArgs = document.querySelectorAll(".command-args");

  if (commandTypeSelect) {
    // Handle command type change
    commandTypeSelect.addEventListener("change", function () {
      const selectedType = this.value;

      // Hide all command argument sections
      commandArgs.forEach((argSection) => {
        argSection.classList.add("hidden");
      });

      // Show the selected command argument section
      const selectedArgSection = document.getElementById(
        `${selectedType}-args`
      );
      if (selectedArgSection) {
        selectedArgSection.classList.remove("hidden");
      }
    });

    // Trigger change event to set initial state
    commandTypeSelect.dispatchEvent(new Event("change"));
  }

  if (commandForm) {
    commandForm.addEventListener("submit", function (e) {
      e.preventDefault();

      // Get form data
      const formData = new FormData(commandForm);
      const commandType = formData.get("command-type");
      const targetAgents = formData.get("target-agents");

      // Collect command data
      let commandData = {
        type: commandType,
        target_agents: targetAgents,
        priority: formData.get("priority") || "normal",
      };

      // Add command-specific arguments
      switch (commandType) {
        case "exec":
          commandData.command = formData.get("exec-command");
          commandData.args = formData.get("exec-args");
          commandData.timeout = formData.get("exec-timeout");
          break;
        case "scan":
          commandData.scan_type = formData.get("scan-type");
          commandData.scan_target = formData.get("scan-target");
          commandData.scan_options = formData.get("scan-options");
          break;
        case "status":
          commandData.status_type = formData.get("status-check-type");
          break;
        case "config":
          commandData.action = formData.get("config-action");
          commandData.key = formData.get("config-key");
          commandData.value = formData.get("config-value");
          break;
        case "health":
          // No additional parameters needed
          break;
      }

      console.log("Command data:", commandData);

      // Show success message (in a real implementation, this would send to the API)
      showCommandSuccess(commandData);

      // Reset form
      commandForm.reset();
      commandTypeSelect.dispatchEvent(new Event("change"));
    });
  }
}

function showCommandSuccess(commandData) {
  // Create a new activity item showing the command was sent
  const activityFeed = document.querySelector(".activity-feed");
  const activityElement = document.createElement("div");
  activityElement.className = "activity-item fadeIn";

  const commandId = `cmd-${Math.floor(Math.random() * 1000000)}`;
  const now = new Date();

  activityElement.innerHTML = `
        <div class="activity-icon info">
            <i class="fas fa-paper-plane"></i>
        </div>
        <div class="activity-content">
            <div class="activity-title">Command Sent</div>
            <div class="activity-details">
                <span>${getCommandDescription(
                  commandData
                )} (ID: ${commandId})</span>
                <span class="activity-time">just now</span>
            </div>
        </div>
    `;

  // Add to beginning of feed
  if (activityFeed.firstChild) {
    activityFeed.insertBefore(activityElement, activityFeed.firstChild);
  } else {
    activityFeed.appendChild(activityElement);
  }

  // If there's a loading indicator, remove it
  const loadingIndicator = activityFeed.querySelector(".loading-indicator");
  if (loadingIndicator) {
    loadingIndicator.remove();
  }

  // Remove excess activity items to keep the list manageable
  const activityItems = activityFeed.querySelectorAll(".activity-item");
  if (activityItems.length > 10) {
    for (let i = 10; i < activityItems.length; i++) {
      activityItems[i].remove();
    }
  }
}

function getCommandDescription(commandData) {
  switch (commandData.type) {
    case "exec":
      return `Execute "${commandData.command}" on ${commandData.target_agents}`;
    case "scan":
      return `Run ${commandData.scan_type} scan on ${commandData.target_agents}`;
    case "status":
      return `Check ${commandData.status_type} status on ${commandData.target_agents}`;
    case "config":
      return `${commandData.action} configuration on ${commandData.target_agents}`;
    case "health":
      return `Health check on ${commandData.target_agents}`;
    default:
      return `${commandData.type} command on ${commandData.target_agents}`;
  }
}

function formatDateTime(date) {
  return date.toLocaleString();
}

function formatTimeAgo(date) {
  const now = new Date();
  const diffMs = now - date;
  const diffSec = Math.floor(diffMs / 1000);
  const diffMin = Math.floor(diffSec / 60);
  const diffHr = Math.floor(diffMin / 60);
  const diffDays = Math.floor(diffHr / 24);

  if (diffDays > 0) {
    return `${diffDays}d ago`;
  } else if (diffHr > 0) {
    return `${diffHr}h ago`;
  } else if (diffMin > 0) {
    return `${diffMin}m ago`;
  } else {
    return "just now";
  }
}
