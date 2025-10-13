# Developer Handoff - Template System MVP

## Welcome to the Team! 🚀

This document contains everything you need to begin implementing the Template System MVP. All planning, architecture, and decision-making is complete. You're ready to start coding.

---

## 📚 Documentation Overview

All documentation is located in the `project/` directory. Here's what each document is for:

### **For Getting Started:**

#### `PROJECT-INTRO.md` - **READ THIS FIRST**

**Purpose**: Quick onboarding guide  
**Use When**: You're new to the project  
**Contains**:

- What this project is (5-minute overview)
- Quick start (get running fast)
- Architecture overview (high-level)
- Common tasks (everyday operations)
- FAQ (frequently asked questions)

**Start here** to understand what you're building and why.

---

#### `PLAN.agent-template-system-implementation.md` - **YOUR ROADMAP**

**Purpose**: Detailed implementation plan  
**Use When**: You need to know what to build and how  
**Contains**:

- 12 implementation phases (week-by-week)
- Task-by-task breakdown with test strategies
- 6 major milestones
- Risk management
- Success criteria

**Use this** to understand the implementation sequence and dependencies.

---

#### `tasks/template-system-mvp.json` - **YOUR DAILY TASK LIST**

**Purpose**: Exhaustive task list in JSON format  
**Use When**: Starting each work session  
**Contains**:

- All tasks with IDs, descriptions, details
- Task status (pending/in_progress/done/blocked)
- Dependencies between tasks
- Test strategies for each task
- Priority levels

**Follow this** for day-to-day work. Update task status as you complete work.

---

### **For Understanding Decisions:**

#### `BRAINSTORM.template-system-notes.md` - **WHY WE CHOSE THIS**

**Purpose**: Design decisions and rationale  
**Use When**: You wonder "why did we do it this way?"  
**Contains**:

- 7 major architectural decisions
- Research summaries
- Alternative approaches considered
- Rationale for choices made

**Reference this** when you need to understand the reasoning behind design choices.

---

#### `documentation/agent_template_system_PRD.md` - **THE REQUIREMENTS**

**Purpose**: Product requirements document  
**Use When**: You need to understand project scope and goals  
**Contains**:

- Project objectives
- Success metrics
- Core features
- Non-goals
- Risks and mitigations

**Check this** when you need to verify if something is in scope or not.

---

### **For Cleanup & Deprecation:**

#### `PROJECT-CLEANUP-ANALYSIS.md` - **WHAT TO DELETE**

**Purpose**: Project directory cleanup plan  
**Use When**: Starting Phase 0 (cleanup)  
**Contains**:

- Files to keep (green)
- Files to refactor (yellow)
- Files to delete (red)
- Cleanup strategy (3 phases)
- Post-cleanup structure

**Use this** during Phase 0 to clean up the project safely.

---

#### `CODE-DEPRECATION-ANALYSIS.md` - **CODE TO REMOVE**

**Purpose**: Identifies obsolete code to remove during implementation  
**Use When**: You're working on Phases 9-11  
**Contains**:

- ~3,000 lines of code to remove
- When to remove each piece
- Safe deprecation checklist
- Risk mitigation

**Reference this** when you're ready to remove old POC code (don't remove too early!).

---

### **For Risk Avoidance:**

#### `CRITICAL-CONSIDERATIONS.md` - **DON'T FALL INTO THESE TRAPS**

**Purpose**: Pitfalls and risk mitigation  
**Use When**: Before starting implementation, and regularly during development  
**Contains**:

- 17 critical technical issues
- Architectural pitfalls
- Security considerations
- Performance pitfalls
- Pre-implementation checklist

**Read this** before starting any major component to avoid known pitfalls.

---

### **For Completion:**

#### `BRAINSTORM-COMPLETE-SUMMARY.md` - **PLANNING SUMMARY**

**Purpose**: Summary of entire brainstorming phase  
**Use When**: You want a high-level overview of all decisions  
**Contains**:

- All documents created (this list)
- All major decisions
- Scope summary
- Success criteria

**Use this** as a quick reference to understand the full planning context.

---

## 🎯 Your First Steps

### Step 1: Read & Understand (Day 1 Morning)

**Time**: 2-3 hours  
**Goal**: Understand what you're building

1. **Read** `PROJECT-INTRO.md` (30 minutes)

   - Understand the project at a high level
   - Get familiar with architecture
   - See common tasks

2. **Skim** `PLAN.agent-template-system-implementation.md` (30 minutes)

   - Understand the overall flow
   - See the 12 phases
   - Note the milestones

3. **Read** Phase 0 in detail from `PLAN.agent-template-system-implementation.md` (30 minutes)

   - Understand what cleanup needs to happen
   - Review `PROJECT-CLEANUP-ANALYSIS.md`

4. **Read** `CRITICAL-CONSIDERATIONS.md` - Pre-Implementation Checklist section (30 minutes)

   - Understand risks before starting
   - Note security considerations
   - Review thread-safety requirements

5. **Review** `tasks/template-system-mvp.json` - Phase 0 tasks (30 minutes)
   - See exact tasks you'll work on first
   - Understand dependencies
   - Note test strategies

---

### Step 2: Set Up Development Environment (Day 1 Afternoon)

**Time**: 2-3 hours  
**Goal**: Get your environment ready

1. **Clone the repository** (if you haven't already)

   ```bash
   cd /path/to/workspace
   # Repository should already be cloned
   cd app-agent
   ```

2. **Install dependencies**

   ```bash
   # Ensure Go 1.24+ is installed
   go version

   # Ensure Docker is installed
   docker --version

   # Verify dependencies
   go mod download
   ```

3. **Create feature branch**

   ```bash
   git checkout -b feature/template-system-mvp
   ```

4. **Snapshot current state** (before any changes)

   ```bash
   git add -A
   git commit -m "snapshot: starting template-system-mvp implementation"
   ```

5. **Open task file** in your editor
   ```bash
   # Open tasks/template-system-mvp.json
   # You'll be updating this as you work
   ```

---

### Step 3: Begin Phase 0 - Task 0.1 (Day 1 Late Afternoon / Day 2)

**Time**: 3-4 hours  
**Goal**: Clean up project directory

1. **Mark task 0.1 as in_progress**

   ```json
   {
     "id": "0.1",
     "status": "in_progress",  // ← Change this
     ...
   }
   ```

2. **Follow `PROJECT-CLEANUP-ANALYSIS.md`** - Phase 1: Safe Removal section

   - Delete test commands (`cmd/test-*`)
   - Delete old binaries (`bin/`, root-level binaries)
   - Delete POC files
   - Archive old documentation

3. **Commit after each major deletion**

   ```bash
   # Example:
   git add .
   git commit -m "chore: remove test commands from cmd/"

   git add .
   git commit -m "chore: remove old binaries and POC files"

   git add .
   git commit -m "chore: archive old documentation"
   ```

4. **Run test strategy** from task 0.1:

   ```bash
   git status  # Verify only intentional deletions
   go build ./...  # Ensure project still compiles
   ```

5. **Mark task 0.1 as done**

   ```json
   {
     "id": "0.1",
     "status": "done",  // ← Change this
     ...
   }
   ```

6. **Commit task file**
   ```bash
   git add tasks/template-system-mvp.json
   git commit -m "chore: mark task 0.1 complete"
   ```

---

### Step 4: Continue Through Phase 0 (Day 2-3)

**Time**: Full 2 days  
**Goal**: Complete all Phase 0 setup

1. **Task 0.2**: Directory Structure Setup

   - Create new directories
   - Add README stubs
   - Commit

2. **Task 0.3**: Container Development Environment

   - Create Dockerfile.linux
   - Create docker-compose.dev.yaml
   - Create Makefile
   - Test: `make build-linux && make shell`
   - Commit

3. **After Phase 0 Complete**:
   - All tasks in Phase 0 should be status: "done"
   - Project is clean and structured
   - Container workflow is fast (<10 sec iteration)
   - Ready to begin Phase 1

---

### Step 5: Begin Phase 1 (Week 1 Weekend / Week 2)

**Continue following** `tasks/template-system-mvp.json` task by task.

**Pattern for each task**:

1. Mark as `in_progress`
2. Read task details and test strategy
3. Implement the feature
4. Run the test strategy
5. Mark as `done`
6. Commit code + task file

---

## 🔄 Daily Workflow

### At Start of Each Work Session:

1. **Review task file**: `tasks/template-system-mvp.json`
2. **Find next available task**: `status: "pending"` with all dependencies `done`
3. **Update status**: Change to `in_progress`
4. **Read task details**: Understand what to implement
5. **Check related docs**: Review PLAN, BRAINSTORM notes if needed

### During Work:

1. **Follow test strategy**: Test as you build
2. **Commit frequently**: Small commits with clear messages
3. **Update task status**: Mark `done` when complete
4. **Ask questions**: If unclear, refer to docs or ask team

### At End of Work Session:

1. **Commit task file**: Update with progress
2. **Push branch**: Keep remote up to date (if desired)
3. **Note blockers**: If blocked, update task status and note issue

---

## 📋 Task Management Rules

Following `Sirius/documentation/dev/operations/README.tasks.md`:

### Task Status Values:

| Status        | Meaning                 | When to Use                     |
| ------------- | ----------------------- | ------------------------------- |
| `pending`     | Not started             | Default for all tasks initially |
| `in_progress` | Currently working on it | When you start a task           |
| `done`        | Completed successfully  | When test strategy passes       |
| `blocked`     | Can't proceed           | When waiting on external factor |

### Dependency Rules:

- **Only work on tasks** where all dependencies are `done`
- **Check dependencies** before starting any task
- **Update dependent tasks** when you complete a task

### Commit Pattern:

```bash
# After completing task 1.2:
git add .
git commit -m "feat: implement template type system

- Created Template, TemplateInfo, DetectionConfig structs
- Added JSON/YAML tags
- Defined severity levels
- Task 1.2 complete"
```

---

## 🛠️ Development Tools & Commands

### Building:

```bash
# Build for your OS (macOS development)
go build -o bin/agent cmd/agent/main.go

# Cross-compile for Linux (testing)
GOOS=linux GOARCH=amd64 go build -o bin/agent cmd/agent/main.go

# Or use Makefile (once created in Phase 0)
cd testing
make build-linux
```

### Testing:

```bash
# After Phase 0 (Makefile created):
cd testing
make quick               # Fast iteration
make test-template TEMPLATE=01-file-hash.yaml
make test-all            # All tests
make shell               # Interactive debugging

# Unit tests (when written):
go test ./...

# With race detection:
go test -race ./...
```

### Code Quality:

```bash
# Format code
gofmt -w .

# Clean dependencies
go mod tidy

# Check for issues
go vet ./...
```

---

## 📖 Reference Materials

### Architecture Patterns:

- **Module Registry**: See `BRAINSTORM.template-system-notes.md` Section 1
- **CLI Structure**: See `BRAINSTORM.template-system-notes.md` Section 2
- **Container Setup**: See `BRAINSTORM.template-system-notes.md` Section 3
- **Results Format**: See `BRAINSTORM.template-system-notes.md` Section 4
- **YAML Schema**: See `BRAINSTORM.template-system-notes.md` Section 5
- **Execution Model**: See `BRAINSTORM.template-system-notes.md` Section 6

### Code Examples:

- See `PLAN.agent-template-system-implementation.md` for pseudocode examples
- See `PROJECT-INTRO.md` for complete working examples
- Once implemented, reference existing modules for patterns

### When Stuck:

1. **Check task details** in `tasks/template-system-mvp.json`
2. **Read relevant section** in `PLAN.agent-template-system-implementation.md`
3. **Review design rationale** in `BRAINSTORM.template-system-notes.md`
4. **Check for pitfalls** in `CRITICAL-CONSIDERATIONS.md`
5. **Ask the team** if still unclear

---

## ⚠️ Critical Reminders

### Security:

- **CommandVersion module**: NO shell interpretation (`exec.Command` directly, not `sh -c`)
- **File operations**: Validate paths to prevent traversal attacks
- **Regex matching**: Implement timeout to prevent ReDoS
- **See**: `CRITICAL-CONSIDERATIONS.md` Section "Security Considerations"

### Thread Safety:

- **Module registry**: Use `sync.RWMutex`
- **Worker pool**: Use channels, not shared maps
- **Test with**: `go test -race ./...`
- **See**: `CRITICAL-CONSIDERATIONS.md` Section "Concurrent Map Access"

### Performance:

- **Context timeouts**: Implement at every layer (per-step, per-template)
- **Worker pool**: Default to `runtime.NumCPU()`
- **File I/O**: Limit concurrent operations
- **See**: `CRITICAL-CONSIDERATIONS.md` Section "Performance Pitfalls"

### Don't Delete Too Early:

- **Old code**: Keep until replacement is working (see `CODE-DEPRECATION-ANALYSIS.md`)
- **Remove in Phase 11**: After all new modules are functional
- **Follow**: Phase-by-phase removal plan

---

## 🎯 Success Criteria (Reminder)

You'll know MVP is complete when:

- [ ] Agent runs in standalone mode
- [ ] Three modules work (FileHash, FileContent, CommandVersion)
- [ ] Templates use AND/OR logic
- [ ] Worker pool executes 1,000+ templates efficiently
- [ ] JSON/JSONL output correct
- [ ] Container testing workflow established
- [ ] Example templates work
- [ ] Documentation complete
- [ ] Integration tests pass
- [ ] Merged to main

---

## 🚦 Milestones to Celebrate

### Milestone 1: Foundation Complete (End Week 1)

- [ ] Project cleaned up
- [ ] Module registry working
- [ ] Type system defined
- [ ] Container workflow fast

### Milestone 2: First Module Working (End Week 2)

- [ ] Template parser complete
- [ ] FileHash module functional
- [ ] Can execute single template
- [ ] JSON output working

### Milestone 3: CLI & Multiple Modules (End Week 3)

- [ ] Cobra CLI implemented
- [ ] FileContent module working
- [ ] Template executor complete
- [ ] Integration tests passing

### Milestone 4: Production Features (End Week 4)

- [ ] Worker pool functional
- [ ] CommandVersion module working
- [ ] All output formats working
- [ ] Performance acceptable

### Milestone 5: Complete & Documented (End Week 5)

- [ ] All modules documented
- [ ] Example templates created
- [ ] Comprehensive tests passing
- [ ] Ready for wider testing

### Milestone 6: Production Ready (End Week 6)

- [ ] Final testing complete
- [ ] Documentation reviewed
- [ ] Release prepared
- [ ] Merged to main
- [ ] **CELEBRATE!** 🎉

---

## 📞 Getting Help

### Documentation Issues:

- Check if question is answered in docs
- Reference specific document section
- If docs unclear, note for improvement

### Technical Issues:

- Review relevant planning documents
- Check `CRITICAL-CONSIDERATIONS.md` for known issues
- Test your assumptions
- Ask team for second opinion

### Scope Questions:

- Check `documentation/agent_template_system_PRD.md`
- Review "What's IN Scope" vs "What's OUT of Scope"
- When in doubt, defer to post-MVP

---

## ✅ Pre-Start Checklist

Before you write any code, verify:

- [ ] Read `PROJECT-INTRO.md` (you understand what you're building)
- [ ] Read Phase 0 of `PLAN.agent-template-system-implementation.md` (you know next steps)
- [ ] Read `CRITICAL-CONSIDERATIONS.md` Pre-Implementation Checklist (you know the risks)
- [ ] Reviewed `tasks/template-system-mvp.json` Phase 0 (you know your first tasks)
- [ ] Go 1.24+ installed
- [ ] Docker installed
- [ ] Feature branch created
- [ ] Current state snapshotted in git
- [ ] Task file open in editor

---

## 🎉 You're Ready!

**Everything you need is documented.**  
**All decisions are made.**  
**The path is clear.**

Your first task is **Task 0.1: Repository Cleanup** in Phase 0.

**Start by opening**:

1. `PROJECT-CLEANUP-ANALYSIS.md` (what to delete)
2. `tasks/template-system-mvp.json` (mark 0.1 as in_progress)

**Good luck, and happy coding!** 🚀

---

_If you have questions about this handoff document or need clarification on any aspect, please ask before starting. We want you to be successful._
