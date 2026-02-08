<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Removed fake interactive prompt from `orch spawn` that asked users to confirm kb context inclusion - context is now automatically included.

**Evidence:** Smoke test confirms no prompt appears and spawn completes successfully with message "Found N relevant context entries - including in spawn context."

**Knowledge:** The prompt was unnecessary because the orchestrator has already decided to spawn; the answer should always be "yes".

**Next:** Close - implementation complete, tests passing, smoke test verified.

**Confidence:** Very High (95%) - straightforward code removal with clear success criteria.

---

# Investigation: Orch Spawn Remove Fake Interactive

**Question:** How to remove the fake interactive prompt from `orch spawn` that asks for kb context inclusion confirmation?

**Started:** 2025-12-26
**Updated:** 2025-12-26
**Owner:** og-debug-orch-spawn-remove-26dec
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** Very High (95%)

---

## Findings

### Finding 1: DisplayContextAndPrompt function read from stdin

**Evidence:** Function at `pkg/spawn/kbcontext.go:410-439` used `bufio.NewReader(os.Stdin)` to read user input asking "Include this context in SPAWN_CONTEXT.md? [Y/n]:"

**Source:** `pkg/spawn/kbcontext.go:423-434`

**Significance:** This was the source of the interactive prompt that blocked non-interactive use cases.

---

### Finding 2: Only one call site existed

**Evidence:** `grep DisplayContextAndPrompt` found only one usage at `cmd/orch/main.go:4101`

**Source:** `cmd/orch/main.go:4101`

**Significance:** Safe to remove the entire function since it's not reused elsewhere.

---

### Finding 3: The prompt was unnecessary

**Evidence:** The orchestrator has already decided to spawn when reaching this point. The answer should always be "yes" since context inclusion is the intended behavior.

**Source:** Flow analysis of `runPreSpawnKBCheckFull` function

**Significance:** The prompt adds no value - it's pure friction that breaks automation/scripting.

---

## Synthesis

**Key Insights:**

1. **Unnecessary UX friction** - The prompt asked a question with only one sensible answer ("yes")

2. **Single call site** - Enabled complete removal of the function without impact

3. **Clean removal** - Also removed unused imports (`bufio`, `os`, `golang.org/x/term` from different file)

**Answer to Investigation Question:**

Remove the call to `DisplayContextAndPrompt` in `runPreSpawnKBCheckFull` and replace with a simple output message: "Found N relevant context entries - including in spawn context." Then remove the unused function and its imports.

---

## Confidence Assessment

**Current Confidence:** Very High (95%)

**Why this level?**

All tests pass, build succeeds, and smoke test confirms the fix works correctly.

**What's certain:**

- ✅ No more interactive prompt during orch spawn
- ✅ Context is automatically included
- ✅ All tests pass
- ✅ Build succeeds

**What's uncertain:**

- ⚠️ None - this is a straightforward removal

---

## Implementation Recommendations

**Recommended Approach:** Complete - already implemented

**Implementation sequence:**
1. Remove `DisplayContextAndPrompt` call from `runPreSpawnKBCheckFull` (main.go)
2. Replace with simple output message
3. Remove `DisplayContextAndPrompt` function (kbcontext.go)
4. Remove unused imports

**Success criteria:**
- ✅ `orch spawn` completes without waiting for user input
- ✅ All tests pass
- ✅ Context is still included in SPAWN_CONTEXT.md

---

## References

**Files Modified:**
- `cmd/orch/main.go:4095-4107` - Removed prompt call, simplified flow
- `pkg/spawn/kbcontext.go:410-439` - Removed function
- `pkg/spawn/kbcontext.go:4-13` - Removed unused imports

**Commands Run:**
```bash
# Find all usages
grep DisplayContextAndPrompt **/*.go

# Run tests
go test ./pkg/spawn/... -v
go test ./cmd/orch/... -v -run "KB|Context"

# Build
go build ./...

# Smoke test
orch spawn --no-track --max-agents 10 investigation "test orch spawn context"
```

---

## Investigation History

**2025-12-26:** Investigation started
- Initial question: How to remove fake interactive prompt from orch spawn?
- Context: Prompt was blocking non-interactive use cases

**2025-12-26:** Investigation completed
- Final confidence: Very High (95%)
- Status: Complete
- Key outcome: Removed unnecessary prompt, context now auto-included
