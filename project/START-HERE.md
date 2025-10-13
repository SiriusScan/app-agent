# START HERE - Template System MVP

## 🎯 Quick Navigation

**New Developer?** → Read `DEVELOPER-HANDOFF.md` first  
**Need Overview?** → Read `PROJECT-INTRO.md` next  
**Ready to Code?** → Open `tasks/template-system-mvp.json` and begin with Task 0.1

---

## 📚 Complete Documentation Index

### Essential (Read in Order):

1. **DEVELOPER-HANDOFF.md** - Your complete guide to getting started
2. **PROJECT-INTRO.md** - What you're building and why
3. **tasks/template-system-mvp.json** - Your daily task list (JSON format)
4. **PLAN.agent-template-system-implementation.md** - Complete implementation plan

### Reference (Use As Needed):

5. **BRAINSTORM.template-system-notes.md** - Why we made each decision
6. **PROJECT-CLEANUP-ANALYSIS.md** - What to delete during Phase 0
7. **CODE-DEPRECATION-ANALYSIS.md** - What code to remove during implementation
8. **CRITICAL-CONSIDERATIONS.md** - Pitfalls to avoid
9. **BRAINSTORM-COMPLETE-SUMMARY.md** - Summary of entire planning phase
10. **documentation/agent_template_system_PRD.md** - Product requirements

---

## 🚀 Project Status

### ✅ COMPLETE:

- [x] Product requirements defined (PRD)
- [x] Architecture designed (all major decisions made)
- [x] Implementation plan created (12 phases, 6 weeks)
- [x] Task list created (93 detailed tasks)
- [x] Risk analysis complete (17 critical issues identified & mitigated)
- [x] Code deprecation plan (what to remove and when)
- [x] Developer documentation (onboarding, guides, references)

### 🔄 NEXT STEPS:

- [x] ~~Phase 0, Task 0.1: Project Cleanup~~ ✅ **COMPLETE** (see `CLEANUP-COMPLETED.md`)
- [ ] Begin Phase 0, Task 0.2: Directory Structure Setup
- [ ] Follow task list from `tasks/template-system-mvp.json`
- [ ] Update task status as work progresses

---

## 📋 Task List Overview

**Total Tasks**: 93 (across 12 phases)  
**Format**: JSON (follows project standards)  
**Location**: `tasks/template-system-mvp.json`

### Phase Breakdown:

| Phase | Name                         | Tasks | Duration | Status  |
| ----- | ---------------------------- | ----- | -------- | ------- |
| 0     | Project Cleanup & Foundation | 3     | 2 days   | pending |
| 1     | Core Architecture            | 4     | 3 days   | pending |
| 2     | Template Parser              | 3     | 3 days   | pending |
| 3     | FileHash Module              | 3     | 3 days   | pending |
| 4     | Template Executor            | 3     | 2 days   | pending |
| 5     | FileContent Module           | 2     | 2 days   | pending |
| 6     | CLI with Cobra               | 3     | 3 days   | pending |
| 7     | Worker Pool                  | 2     | 2 days   | pending |
| 8     | Result Output Format         | 2     | 1 day    | pending |
| 9     | CommandVersion Module        | 2     | 2 days   | pending |
| 10    | Documentation & Examples     | 4     | 3 days   | pending |
| 11    | Integration & Polish         | 4     | 3 days   | pending |
| 12    | Final Testing & Release      | 4     | 2 days   | pending |

**Total Estimated Duration**: 4-6 weeks

---

## 🎯 MVP Scope

### ✅ INCLUDED (MVP):

**Core Functionality:**

- Standalone mode (agent runs templates locally)
- YAML template parsing and validation
- Three detection modules (FileHash, FileContent, CommandVersion)
- Worker pool for parallel template execution
- AND/OR detection logic
- JSON/JSONL output formats
- Cobra-based CLI with subcommands

**Development:**

- Container-based Linux testing (Ubuntu 22.04)
- Fast iteration workflow (<10 sec)
- Integration test suite
- Example templates

**Documentation:**

- Template writing guide
- Module development guide
- API documentation
- Complete examples

### ❌ DEFERRED (Post-MVP):

- Script module (will use embedded Lua, not Bash/PowerShell/Python)
- Server mode integration (template sync)
- Template repository sync from GitHub/server
- macOS support
- Windows support
- Complex logic expressions `(step1 AND step2) OR step3`
- Template variables `${OS}`, `${ARCH}`
- Result caching
- Template signing/verification

---

## 🏗️ Architecture Highlights

### Module Registry Pattern:

- Struct-based registration (idiomatic Go)
- Thread-safe with `sync.RWMutex`
- Minimal Module interface (Execute only)
- Rich metadata via Descriptor struct

### CLI Structure:

- Cobra framework (industry standard)
- Subcommands: `template`, `module`
- Server mode backward compatible
- JSON output default (pipeable)

### Execution Model:

- Sequential steps within template (simple logic)
- Parallel templates via worker pool (scalable)
- Default workers: `runtime.NumCPU()`
- Context timeouts at every layer

### Testing Approach:

- Container as Linux execution environment
- Cross-compile on macOS host
- Integration tests (outcome-focused)
- No unit test coverage requirements

---

## ⚠️ Critical Reminders

### Security:

- **CommandVersion**: NO shell interpretation
- **File operations**: Validate paths (prevent traversal)
- **Regex**: Implement timeout (prevent ReDoS)

### Thread Safety:

- **Module registry**: Use `sync.RWMutex`
- **Worker pool**: Use channels (not shared maps)
- **Test with**: `go test -race ./...`

### Performance:

- **Timeouts**: Per-step (30s), per-template (5min)
- **Worker pool**: Default = CPU count
- **File I/O**: Limit concurrent operations

### Code Deprecation:

- **Don't delete old code** until replacement works
- **Remove in Phase 11** (after all modules functional)
- **Follow**: CODE-DEPRECATION-ANALYSIS.md

---

## 📊 Success Criteria

MVP is complete when:

- [ ] Agent runs in standalone mode
- [ ] Three modules work (FileHash, FileContent, CommandVersion)
- [ ] Templates use AND/OR logic
- [ ] Worker pool executes 1,000+ templates efficiently (<20 min)
- [ ] JSON/JSONL output correct
- [ ] Container testing workflow established (<10 sec iteration)
- [ ] Example templates demonstrate all features
- [ ] Documentation complete and reviewed
- [ ] Integration tests pass
- [ ] Code merged to main branch

---

## 🎓 How to Use This Documentation

### For First-Time Setup:

1. Read `DEVELOPER-HANDOFF.md` (complete guide)
2. Read `PROJECT-INTRO.md` (understand the system)
3. Skim `PLAN.agent-template-system-implementation.md` (see the roadmap)
4. Set up your environment (Go, Docker)
5. Create feature branch
6. Begin Task 0.1

### For Daily Work:

1. Open `tasks/template-system-mvp.json`
2. Find next available task (pending, dependencies met)
3. Mark as `in_progress`
4. Implement following task details
5. Run test strategy
6. Mark as `done`
7. Commit code + task file

### When You Need Help:

1. Check task details in `tasks/template-system-mvp.json`
2. Review relevant section in `PLAN.agent-template-system-implementation.md`
3. Read design rationale in `BRAINSTORM.template-system-notes.md`
4. Check for pitfalls in `CRITICAL-CONSIDERATIONS.md`
5. Ask the team

---

## 💡 Key Design Decisions

All major architectural decisions are documented in `BRAINSTORM.template-system-notes.md`:

1. **Module Registry** → Struct-based (matches Go stdlib patterns)
2. **CLI** → Cobra with subcommands (industry standard)
3. **Testing** → Container as execution environment (fast iteration)
4. **Output** → JSON/JSONL hybrid (streaming-friendly)
5. **Schema** → Simple AND/OR logic (MVP simplicity)
6. **Execution** → Sequential steps, parallel templates (scalable)
7. **MVP Scope** → Three modules (Script deferred to Lua)

---

## 🔗 Quick Links

### Planning Documents:

- [Developer Handoff](DEVELOPER-HANDOFF.md) - Complete getting started guide
- [Project Intro](PROJECT-INTRO.md) - System overview and onboarding
- [Implementation Plan](PLAN.agent-template-system-implementation.md) - Detailed roadmap
- [Task List](../tasks/template-system-mvp.json) - Daily task tracking

### Reference Documents:

- [Brainstorming Notes](BRAINSTORM.template-system-notes.md) - Design decisions
- [Cleanup Analysis](PROJECT-CLEANUP-ANALYSIS.md) - What to delete
- [Deprecation Analysis](CODE-DEPRECATION-ANALYSIS.md) - Code to remove
- [Critical Considerations](CRITICAL-CONSIDERATIONS.md) - Pitfalls to avoid
- [PRD](../documentation/agent_template_system_PRD.md) - Product requirements

---

## 🎉 Ready to Begin!

**Everything you need is documented.**  
**All decisions are made.**  
**The path is clear.**

### Your First Task:

**Task 0.1: Repository Cleanup**

1. Open `PROJECT-CLEANUP-ANALYSIS.md`
2. Open `tasks/template-system-mvp.json`
3. Mark task 0.1 as `in_progress`
4. Follow cleanup instructions
5. Test: `git status` and `go build ./...`
6. Mark task 0.1 as `done`
7. Commit

**Good luck, and welcome to the team!** 🚀

---

_For questions or clarifications, please review the documentation first. If still unclear, ask the team._
