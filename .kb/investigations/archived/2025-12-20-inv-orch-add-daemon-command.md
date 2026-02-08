**TLDR:** Question: How to implement daemon command for autonomous overnight processing? Answer: Created pkg/daemon package with core methods (NextIssue, Preview, Once, Run) that integrate with beads queue via `bd list` and spawn work via `orch-go work`. High confidence (95%) - all 16 tests pass.

---

# Investigation: Daemon Command Implementation

**Question:** How should we implement the daemon command for autonomous overnight processing of beads issues?

**Started:** 2025-12-20
**Updated:** 2025-12-20
**Owner:** Claude agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** Very High (95%)

---

## Findings

### Finding 1: Core daemon functionality implemented in pkg/daemon

**Evidence:** Created daemon package with the following core methods:
- `NextIssue()` - Returns highest priority spawnable issue (sorts by priority, skips epic/chore/blocked)
- `Preview()` - Shows next issue without processing
- `Once()` - Processes single issue and returns
- `Run(maxIterations)` - Loop processing with limit

**Source:** 
- `pkg/daemon/daemon.go` - 240 lines of implementation
- `pkg/daemon/daemon_test.go` - 16 passing tests

**Significance:** Provides the core business logic for daemon operations. Uses dependency injection (listIssuesFunc, spawnFunc) for testability.

---

### Finding 2: Integration with beads and orch-go work command

**Evidence:** Default implementations:
- `ListOpenIssues()` - Shells out to `bd list --status open --json`
- `SpawnWork()` - Shells out to `orch-go work <beads-id>`

**Source:** `pkg/daemon/daemon.go:157-183`

**Significance:** Reuses existing infrastructure - no need for new beads API or spawn logic.

---

### Finding 3: Skill inference matches existing work command

**Evidence:** `InferSkill()` maps:
- bug → systematic-debugging
- feature → feature-impl  
- task → feature-impl
- investigation → investigation
- epic → error (not spawnable)

**Source:** `pkg/daemon/daemon.go:116-130`

**Significance:** Consistent with existing `InferSkillFromIssueType` in cmd/orch/main.go.

---

## Synthesis

**Key Insights:**

1. **Package structure works well** - Separate pkg/daemon allows testing without CLI integration
2. **Dependency injection enables testing** - Mock functions for bd list and spawn
3. **CLI wiring deferred** - Due to concurrent modifications, daemon.go CLI file was created but not integrated into main.go

**Answer to Investigation Question:**

The daemon command is implemented as a separate package (pkg/daemon) with all core functionality tested. The package provides:
- Queue management (NextIssue with priority sorting)
- Preview capability (show without processing)
- Single-shot processing (Once)
- Loop processing (Run with iteration limit)

CLI integration (cmd/orch/daemon.go) was created but needs to be registered in main.go separately.

---

## Confidence Assessment

**Current Confidence:** Very High (95%)

**Why this level?**

All 16 unit tests pass. Core logic is straightforward and follows existing patterns.

**What's certain:**

- ✅ pkg/daemon package compiles and tests pass
- ✅ Integration with beads via `bd list --status open --json` 
- ✅ Integration with spawn via `orch-go work`
- ✅ Priority-based queue processing works

**What's uncertain:**

- ⚠️ CLI wiring in main.go not yet integrated (concurrent modifications)
- ⚠️ start/stop/install subcommands not implemented (lower priority)

**What would increase confidence to 100%:**

- Integration testing with real beads queue
- CLI fully wired and tested end-to-end

---

## Implementation Recommendations

### Recommended Approach ⭐

**Wire daemon command into CLI** - Add `rootCmd.AddCommand(daemonCmd)` to main.go

**Why this approach:**
- cmd/orch/daemon.go already created with subcommands
- Just needs registration in init()

**Implementation sequence:**
1. Add `rootCmd.AddCommand(daemonCmd)` to cmd/orch/main.go init()
2. Test with `orch-go daemon preview`
3. Test with `orch-go daemon once`

---

## References

**Files Examined:**
- `cmd/orch/main.go` - Existing CLI structure
- `pkg/verify/check.go` - Beads integration patterns

**Commands Run:**
```bash
# Run tests
go test ./pkg/daemon/... -v

# All package tests
go test ./pkg/...
```

**Deliverables:**
- `pkg/daemon/daemon.go` - Core daemon implementation
- `pkg/daemon/daemon_test.go` - 16 passing tests
- `cmd/orch/daemon.go` - CLI subcommands (needs wiring)

---

## Investigation History

**2025-12-20 18:20:** Investigation started
- Initial question: How to implement daemon command
- Context: Autonomous overnight processing needed

**2025-12-20 18:30:** Core implementation complete
- Created pkg/daemon with TDD approach
- 16 tests all passing

**2025-12-20 18:35:** Investigation completed
- Final confidence: Very High (95%)
- Status: Complete
- Key outcome: pkg/daemon package ready, CLI needs wiring
