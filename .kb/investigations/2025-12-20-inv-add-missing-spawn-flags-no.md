**TLDR:** Question: How to add three missing spawn flags (--no-track, --mcp, --skip-artifact-check) to orch-go? Answer: Added flags following existing patterns in cmd/orch/main.go and pkg/spawn/config.go, with proper event logging and user output. High confidence (95%) - implementation complete, tests passing.

---

# Investigation: Add Missing Spawn Flags

**Question:** How to implement --no-track, --mcp, and --skip-artifact-check flags for the orch-go spawn command?

**Started:** 2025-12-20
**Updated:** 2025-12-20
**Owner:** Worker agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** Very High (95%)

---

## Findings

### Finding 1: Existing flag patterns in spawn command

**Evidence:** The spawn command already has several flags defined using Cobra's flag system:

- `--issue` (string) - Beads issue ID
- `--phases` (string) - Feature-impl phases
- `--mode` (string) - Implementation mode
- `--validation` (string) - Validation level
- `--inline` (bool) - Run inline instead of tmux
- `--model` (string) - Model selection

**Source:** cmd/orch/main.go:117-127 (init function for spawn flags)

**Significance:** Following these patterns ensures consistency. Bool flags use `BoolVar`, string flags use `StringVar`, all with short descriptions.

---

### Finding 2: SpawnConfig struct needs extension

**Evidence:** The spawn.Config struct in pkg/spawn/config.go holds all spawn configuration. New flags need corresponding fields added here.

**Source:** pkg/spawn/config.go:13-41

**Significance:** Adding fields to Config allows the configuration to be passed through the spawn pipeline and logged in events.

---

### Finding 3: Beads tracking logic needs conditional

**Evidence:** The current code in runSpawnWithSkill always creates a beads issue when spawnIssue is empty. The --no-track flag should skip this.

**Source:** cmd/orch/main.go:518-526 (beads issue creation logic)

**Significance:** Implemented conditional: if --no-track, skip beads issue creation and generate a local-only ID instead.

---

## Synthesis

**Key Insights:**

1. **Flag addition is straightforward** - Cobra's flag system makes adding new flags simple, just follow existing patterns in the init function.

2. **Config propagation is clean** - Adding fields to SpawnConfig and passing them through ensures the values are available for logging and future use.

3. **Minimal changes required** - The --mcp and --skip-artifact-check flags are mostly informational at this stage (logged in events, shown in output) while --no-track has active behavior (skipping beads tracking).

**Answer to Investigation Question:**

Implementation required changes to three files:

1. cmd/orch/main.go - Flag declarations, init registration, beads tracking logic, event logging, output
2. pkg/spawn/config.go - New fields in Config struct

---

## Confidence Assessment

**Current Confidence:** Very High (95%)

**Why this level?**

Implementation is complete, all tests pass, and help output confirms flags are registered correctly.

**What's certain:**

- ✅ All three flags are registered and appear in help output
- ✅ --no-track skips beads issue creation
- ✅ All flag values are logged in spawn events
- ✅ Build and tests pass

**What's uncertain:**

- ⚠️ --mcp flag is logged but not yet used to configure MCP servers (future work)
- ⚠️ --skip-artifact-check is logged but pre-spawn checks are orchestrator-side (Python)

---

## References

**Files Modified:**

- cmd/orch/main.go - Flag declarations, usage, event logging
- pkg/spawn/config.go - Config struct fields

**Commands Run:**

```bash
# Verify build
go build -o /dev/null ./cmd/orch

# Run tests
go test ./...

# Verify help output
go run ./cmd/orch spawn --help
```

---

## Investigation History

**2025-12-20:** Investigation started

- Initial question: Add three missing spawn flags
- Context: Flags needed for orchestrator compatibility

**2025-12-20:** Implementation complete

- Final confidence: Very High (95%)
- Status: Complete
- Key outcome: Three flags added (--no-track, --mcp, --skip-artifact-check) following existing patterns
