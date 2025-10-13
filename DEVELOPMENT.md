# Agent Enhancement Development Guide

This guide covers the development environment setup and workflow for the Sirius Agent Enhancement project.

## 🚀 Quick Start

The development environment is already configured with live code changes, debugging capabilities, and useful development tools.

### Prerequisites

- Docker and Docker Compose running
- The sirius-engine container is running with app-agent volume mounted

### Development Commands

```bash
# Run these commands inside the container:
docker exec -w /app-agent sirius-engine <command>

# Available commands via scripts/dev-commands.sh:
./scripts/dev-commands.sh test-scan      # Test scan command
./scripts/dev-commands.sh test-status    # Test status command
./scripts/dev-commands.sh debug-agent    # Start agent in debug mode
./scripts/dev-commands.sh debug-server   # Start server in debug mode
./scripts/dev-commands.sh live-server    # Start server with live reloading
./scripts/dev-commands.sh build-all      # Build all components
./scripts/dev-commands.sh clean          # Clean build artifacts
```

## 🔧 Development Environment

### Live Code Changes ✅

- Source code is volume mounted from `../minor-projects/app-agent` to `/app-agent` in the container
- Changes made locally are immediately reflected in the container
- No need to rebuild containers for code changes

### Debugging Setup ✅

- **Delve Debugger**: Installed and ready for Go debugging
- **VS Code Integration**: `.vscode/launch.json` configured with debug profiles
- **Debug Ports**:
  - Agent debug: `:2345`
  - Server debug: `:2346`

### Live Reloading ✅

- **Air**: Live reloading tool configured via `.air.toml`
- Automatically rebuilds and restarts on code changes
- Monitors `cmd/` and `internal/` directories

## 📁 Project Structure for Enhancement

```
app-agent/
├── cmd/                           # Main applications
│   ├── agent/                     # Agent entry point
│   └── server/                    # Server entry point
├── internal/
│   ├── commands/                  # Existing command system
│   │   ├── scan/                  # ✅ Fixed scan command
│   │   └── status/                # ✅ Status command
│   ├── detect/                    # 🆕 NEW: Vulnerability detection
│   │   ├── template/              # YAML template processor
│   │   ├── script/                # Custom script executor
│   │   └── hash/                  # File hash verification
│   ├── fingerprint/               # 🆕 NEW: System fingerprinting
│   └── repository/                # 🆕 NEW: Template/script management
├── templates/                     # 🆕 NEW: YAML vulnerability templates
│   ├── hash-based/
│   ├── registry-based/
│   └── config-based/
├── scripts/                       # 🆕 NEW: Custom detection scripts
│   ├── windows/                   # PowerShell scripts
│   ├── linux/                     # Bash scripts
│   └── cross-platform/            # Cross-platform scripts
└── .vscode/                       # VS Code debug configuration
```

## 🐛 Debugging

### VS Code Debugging

1. Open the project in VS Code
2. Press `F5` or go to Run & Debug panel
3. Choose from available configurations:
   - **Debug Agent**: Debug the agent component
   - **Debug Server**: Debug the server component
   - **Debug Scan Command**: Debug scan command specifically

### Command Line Debugging

```bash
# Debug agent on port 2345
./scripts/dev-commands.sh debug-agent

# Debug server on port 2346
./scripts/dev-commands.sh debug-server

# Connect with delve client
dlv connect localhost:2345
```

### Live Server Development

```bash
# Start server with automatic reloading
./scripts/dev-commands.sh live-server
```

## 🧪 Testing

### Manual Testing

```bash
# Test scan command directly
./scripts/dev-commands.sh test-scan

# Test status command
./scripts/dev-commands.sh test-status

# Test via terminal (from outside container)
# Use the sirius-ui terminal and run: scan
```

### Build Testing

```bash
# Build all components
./scripts/dev-commands.sh build-all

# Clean build artifacts
./scripts/dev-commands.sh clean
```

## 📋 Development Workflow

### Phase 0: Foundation ✅

- [x] Terminal scan command fixed
- [x] Development environment configured
- [ ] Database migration for SBOM schema
- [ ] Project structure established

### Phase 1: Enhanced SBOM and System Fingerprinting

- [ ] Extend scan command data structures
- [ ] Implement system hardware fingerprinting
- [ ] Implement network configuration detection
- [ ] Implement certificate store inventory
- [ ] Enhance package detection with metadata
- [ ] Integrate SBOM data with database
- [ ] Update API endpoints for enhanced data

### Phase 2: Template-Based Vulnerability Detection

- [ ] Design template data structures and interfaces
- [ ] Implement YAML template parser
- [ ] Implement file hash detection engine
- [ ] Implement Windows registry detection
- [ ] Implement configuration file pattern matching
- [ ] Create template execution engine
- [ ] Create initial template repository
- [ ] Integrate template detection with main scan command

### Phase 3: Script-Based Vulnerability Detection

- [ ] Design script execution framework
- [ ] Implement PowerShell script executor
- [ ] Implement Bash script executor
- [ ] Implement script security sandboxing
- [ ] Create script repository structure
- [ ] Implement script result standardization
- [ ] Integrate script detection with scan command
- [ ] Implement script audit logging

### Phase 4: Repository Management

- [ ] Design repository management architecture
- [ ] Implement repository manifest system
- [ ] Implement GPG signature verification
- [ ] Implement remote repository updates
- [ ] Implement atomic repository updates
- [ ] Create repository CLI commands
- [ ] Implement automatic update scheduling

### Phase 5: Integration and Testing

- [ ] Implement enhanced detect command
- [ ] Update database integration for custom vulnerabilities
- [ ] Comprehensive integration testing
- [ ] Performance optimization and tuning
- [ ] Security audit and hardening
- [ ] Update UI components for enhanced data
- [ ] Create comprehensive documentation
- [ ] Prepare production deployment

## 🔍 Current Status

✅ **COMPLETED**: Task 0.1 - Fix Terminal Scan Command Communication
✅ **COMPLETED**: Task 0.2 - Setup Development Environment

**Next**: Task 0.3 - Create Database Migration for SBOM Schema

## 📞 Support

For development questions or issues:

1. Check the logs: `docker exec sirius-engine tail -f /app-agent/server_commands.log`
2. Verify container status: `docker ps | grep sirius-engine`
3. Test basic functionality: `./scripts/dev-commands.sh test-scan`

---

**Happy Coding!** 🚀
