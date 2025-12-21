**TLDR:** Question: How to make headless mode the default spawn behavior? Answer: Modified spawn command logic to use headless as default, added --tmux as opt-in flag, deprecated --headless flag (now no-op). High confidence (95%) - all tests pass, help text verified.

---

# Investigation: Make Headless Mode Default for Spawn

**Question:** How to change orch-go spawn to use headless mode by default and make tmux window spawning opt-in?

**Started:** 2025-12-20
**Updated:** 2025-12-20
**Owner:** Agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** Very High (95%)

---

## Findings

### Finding 1: Current spawn mode logic had tmux as default

**Evidence:** In `cmd/orch/main.go:575-586`, the logic was:
```go
if headless {
    return runSpawnHeadless(...)
}
useTmux := !inline && tmux.IsAvailable()
if useTmux {
    return runSpawnInTmux(...)
}
return runSpawnInline(...)
```

**Source:** `cmd/orch/main.go:575-589`

**Significance:** The default path when no flags specified was tmux (if available), not headless. This needed to be inverted.

---

### Finding 2: Work command also needed updating

**Evidence:** The `work` command used `runWork(serverURL, beadsID, workInline)` with only inline flag. It needed a tmux flag for consistency.

**Source:** `cmd/orch/main.go:247-249, 478-503`

**Significance:** For consistent UX, both spawn and work commands should support the same mode flags (--inline, --tmux, default headless).

---

### Finding 3: Daemon calls work command via subprocess

**Evidence:** In `pkg/daemon/daemon.go:176-183`:
```go
func SpawnWork(beadsID string) error {
    cmd := exec.Command("orch-go", "work", beadsID)
    ...
}
```

**Source:** `pkg/daemon/daemon.go:176-183`

**Significance:** Daemon spawns via subprocess to work command. Since work now defaults to headless, daemon inherits headless behavior automatically.

---

## Synthesis

**Key Insights:**

1. **Spawn mode priority changed** - Logic now is: inline > tmux (opt-in) > headless (default)

2. **Backward compatibility preserved** - The --headless flag is marked deprecated but still accepted (no-op since headless is now default)

3. **Consistent UX across commands** - Both spawn and work commands now support --inline, --tmux, with headless as default

**Answer to Investigation Question:**

Changed spawn mode logic by:
1. Adding `--tmux` flag to spawn and work commands (opt-in for tmux window spawning)
2. Inverting priority: headless is now default, tmux is opt-in
3. Marking `--headless` as deprecated (kept for backward compatibility, now a no-op)
4. Updated all help text and CLAUDE.md documentation

---

## Confidence Assessment

**Current Confidence:** Very High (95%)

**Why this level?**

All tests pass, build succeeds, help text verified correct. The change is purely flag/logic inversion with no new functionality.

**What's certain:**

- ✅ Spawn defaults to headless when no flags specified
- ✅ --tmux flag works to opt-in to tmux window spawning
- ✅ --headless flag is deprecated and accepted (no-op)
- ✅ Work command follows same pattern
- ✅ All tests pass

**What's uncertain:**

- ⚠️ Live testing with actual OpenCode server not performed (requires running server)

**What would increase confidence to Very High (100%):**

- Integration test with running OpenCode server

---

## Implementation Recommendations

### Recommended Approach ⭐

**Headless-first with tmux opt-in** - Already implemented

**Why this approach:**
- Headless is more resource-efficient (no tmux windows)
- Better for automation/daemon workflows
- TUI still available via --inline or --tmux when needed

**Implementation sequence:**
1. Changed spawn logic priority
2. Added --tmux flag
3. Deprecated --headless flag
4. Updated work command similarly
5. Updated documentation

---

## References

**Files Modified:**
- `cmd/orch/main.go` - Spawn and work command flags, logic
- `CLAUDE.md` - Updated spawn flow documentation

**Commands Run:**
```bash
# Build and test
go build ./...
go test ./...

# Verify help text
go run ./cmd/orch spawn --help
go run ./cmd/orch work --help
```

---

## Investigation History

**2025-12-20:** Investigation started
- Initial question: How to make headless mode default?
- Context: Headless mode is preferred for automation

**2025-12-20:** Investigation completed
- Final confidence: Very High (95%)
- Status: Complete
- Key outcome: Headless is now default, --tmux is opt-in, all tests pass
