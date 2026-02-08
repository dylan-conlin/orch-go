<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Added --auto-init flag to spawn command that automatically initializes .beads and .kb when missing, with clear error messages for users who don't use it.

**Evidence:** Tested in temp directories - spawn now fails gracefully with actionable suggestions when .beads is missing, and --auto-init correctly initializes all scaffolding.

**Knowledge:** The .orch/workspace/ and .orch/templates/ directories are already auto-created by spawn.WriteContext(); only beads tracking required explicit pre-initialization.

**Next:** Close - feature is complete with tests passing.

**Confidence:** High (90%) - tested manually and with unit tests, covers main use cases.

---

# Investigation: Auto-init for orch spawn

**Question:** How should spawn handle missing .orch directories to reduce friction for new projects?

**Started:** 2025-12-22
**Updated:** 2025-12-22
**Owner:** og-feat-orch-spawn-auto-22dec
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** High (90%)

---

## Findings

### Finding 1: .orch directories already auto-created

**Evidence:** spawn.WriteContext() calls os.MkdirAll for workspace path and EnsureSynthesisTemplate creates .orch/templates/. These directories are created on first spawn without requiring orch init.

**Source:** pkg/spawn/context.go:226-239

**Significance:** The only missing piece for spawn to work is beads tracking - if .beads/ doesn't exist, bd create fails.

---

### Finding 2: Beads failure is the blocker

**Evidence:** Running spawn in a fresh project without .beads/ fails with `bd create failed: exit status 1`. The error message was not helpful in suggesting alternatives.

**Source:** cmd/orch/main.go:1283-1291 (createBeadsIssue function)

**Significance:** The --no-track flag already bypasses this, but users didn't have clear guidance. Adding --auto-init provides another option.

---

### Finding 3: Issue spec wanted opt-in behavior

**Evidence:** Original issue stated "Should be opt-in or prompting (not silent)".

**Source:** bd show orch-go-ipq9

**Significance:** Auto-init should not be the default - it should be explicit via flag. Users should be informed of their options.

---

## Synthesis

**Key Insights:**

1. **Minimal auto-init needed** - Only beads and kb need initialization; .orch directories are created automatically by spawn internals.

2. **Three valid paths** - Users can: (1) run orch init explicitly, (2) use --auto-init during spawn, or (3) use --no-track to skip beads entirely.

3. **Error messages matter** - The key improvement is telling users what to do when spawn fails, not just that it failed.

**Answer to Investigation Question:**

Spawn should check for .beads/ before attempting to create a beads issue. If missing and --auto-init is set, run minimal initialization (beads + kb, skip CLAUDE.md and tmuxinator). If not set, show a clear error with the three options.

---

## Confidence Assessment

**Current Confidence:** High (90%)

**Why this level?**

Implementation tested manually in temp directories and with unit tests. Main use cases covered.

**What's certain:**

- ✅ --auto-init flag correctly initializes scaffolding
- ✅ --no-track still works without requiring initialization
- ✅ Error messages clearly explain options
- ✅ All tests pass

**What's uncertain:**

- ⚠️ bd init prompts for git hooks - may need stdin handling for fully automated use
- ⚠️ Haven't tested in production projects

**What would increase confidence to Very High:**

- Test with real automation/CI workflows
- Add timeout handling for bd init prompts

---

## References

**Files Examined:**
- cmd/orch/main.go - spawn command and new ensureOrchScaffolding function
- cmd/orch/init.go - initProject for understanding what gets initialized
- pkg/spawn/context.go - WriteContext to understand existing auto-creation

**Commands Run:**
```bash
# Test spawn in fresh project
cd /tmp/test-spawn-auto-init && git init -q
orch spawn --skip-artifact-check investigation "test"  # Fails with helpful error

# Test auto-init
orch spawn --auto-init --skip-artifact-check investigation "test"  # Works

# Test no-track
orch spawn --no-track --skip-artifact-check investigation "test"  # Works
```

**Related Artifacts:**
- **Workspace:** .orch/workspace/og-feat-orch-spawn-auto-22dec/
