<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Backend selection logic needs redesign into single-responsibility function with clear priority chain (flags > project config > global config > default opencode) and advisory-only infrastructure detection.

**Evidence:** Current logic has 7+ overlapping decision factors across 90 lines, misaligned priority chain, complex infrastructure override behavior with configSetBackend boolean tracking.

**Knowledge:** Infrastructure detection should warn but not override user intent; model selection should be separate concern; global config provides useful fallback defaults.

**Next:** Implement feat-051: resolveBackend() function with clean priority chain and warning-only infrastructure detection.

**Promote to Decision:** recommend-yes - establishes architectural pattern for config precedence and advisory safety mechanisms

<!--
Example D.E.K.N.:
Delta: Test-running guidance is missing from spawn prompts and CLAUDE.md.
Evidence: Searched 5 agent sessions - none ran tests; guidance exists in separate docs but isn't loaded.
Knowledge: Agents follow documentation literally; guidance must be in loaded context to be followed.
Next: Add test-running instruction to SPAWN_CONTEXT.md template.
Promote to Decision: recommend-no (tactical fix, not architectural)

Guidelines:
- Keep each line to ONE sentence
- Delta answers "What did we find?"
- Evidence answers "How do we know?"
- Knowledge answers "What does this mean?"
- Next answers "What should happen now?"
- Promote to Decision: flag for orchestrator/human - recommend-yes if this establishes a pattern, constraint, or architectural choice worth preserving
- Enable 30-second understanding for fresh Claude
-->

---

# Investigation: Backend Selection Logic Spawn Cmd

**Question:** How should backend selection logic in spawn_cmd.go be redesigned into a coherent, single-responsibility function to resolve overlapping decision factors and bugs?

**Started:** 2026-01-20
**Updated:** 2026-01-20
**Owner:** architect agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Patches-Decision:** [Path to decision document this investigation patches/extends, if applicable - enables review triggers]
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]
**Supersedes:** [Path to artifact this replaces, if applicable]
**Superseded-By:** [Path to artifact that replaced this, if applicable]

---

## Findings

### Finding 1: Current logic has 7+ overlapping decision factors across ~90 lines

**Evidence:** Code analysis shows decision factors include: --backend flag, --opus flag, project config spawn_mode, model auto-detect, infrastructure gate, orchestrator detection, hardcoded default. The logic spans lines 1148-1230 with complex conditional nesting.

**Source:** `/Users/dylanconlin/Documents/personal/orch-go/cmd/orch/spawn_cmd.go:1148-1230`

**Significance:** Multiple overlapping factors create bugs where intended default (opencode/DeepSeek) gets overridden unexpectedly. Each fix reveals another layer of complexity.

---

### Finding 2: Priority chain claimed in comments doesn't match code structure

**Evidence:** Comments claim priority: 1) --backend flag, 2) --opus flag, 3) config default, 4) model auto-detect, 5) default to claude. But code shows infrastructure detection can override config (lines 1184-1228), and model auto-detect logic (lines 1169-1180) doesn't actually switch to opencode.

**Source:** `/Users/dylanconlin/Documents/personal/orch-go/cmd/orch/spawn_cmd.go:1139-1147` (comments), lines 1169-1180 (model auto-detect), lines 1184-1228 (infrastructure detection)

**Significance:** Misalignment between documented behavior and actual implementation causes confusion and bugs.

---

### Finding 3: Infrastructure gate was 'safety override' but became 'override everything'

**Evidence:** `isCriticalInfrastructureWork()` function detects work on OpenCode server files. Originally a safety override, but now has complex logic with `configSetBackend` boolean tracking whether to warn vs override. This creates two different behaviors based on config state.

**Source:** `/Users/dylanconlin/Documents/personal/orch-go/cmd/orch/spawn_cmd.go:1184-1228` (infrastructure logic), lines 2323-2367 (isCriticalInfrastructureWork function)

**Significance:** Infrastructure detection should warn, not override user intent, but current implementation has become complex gatekeeper logic.

---

## Synthesis

**Key Insights:**

1. **Complexity from overlapping concerns** - The current logic mixes backend selection (claude vs opencode), model resolution, infrastructure safety, and config precedence in one ~90-line block. This violates single responsibility principle.

2. **Misalignment between documented and actual behavior** - Comments claim one priority chain, but code implements a different one with infrastructure detection overriding config in some cases.

3. **Infrastructure detection evolved beyond its original purpose** - Started as safety override for OpenCode server restarts, now has complex logic with `configSetBackend` boolean and different behaviors based on config state.

**Answer to Investigation Question:**

The backend selection logic needs a redesign that extracts decision factors into a single-responsibility function with clear priority chain: 1) explicit flags, 2) project config, 3) global config, 4) default (opencode for cost optimization). Infrastructure detection should warn but not override user intent. The function should return a clear decision with reasoning for logging/debugging.

---

## Structured Uncertainty

**What's tested:**

- ✅ Current logic has 7+ overlapping decision factors (verified: code analysis lines 1148-1230)
- ✅ Priority chain claimed in comments doesn't match code (verified: compare comments 1139-1147 with actual logic)
- ✅ Infrastructure detection has complex behavior (verified: `configSetBackend` boolean tracking)

**What's untested:**

- ⚠️ Actual behavior of proposed redesign (hypothesis: cleaner priority chain will reduce bugs)
- ⚠️ Impact of changing default from claude to opencode (hypothesis: cost optimization benefit)
- ⚠️ User response to infrastructure warnings vs overrides (hypothesis: warnings sufficient)

**What would change this:**

- If cost analysis shows claude backend is actually cheaper for typical workloads
- If infrastructure warnings prove insufficient and agents die frequently
- If global config introduces unexpected conflicts with project config

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**Clean function extraction with clear priority chain** - Extract current backend selection logic into `resolveBackend()` function with signature: `resolveBackend(backendFlag, opusFlag, projCfg, globalCfg, task, beadsID) (backend string, warnings []string)`

**Why this approach:**
- **Single responsibility** - Extracts 90 lines of mixed logic into focused function
- **Testable** - Clear inputs/outputs enable unit testing of priority chain
- **Clear priority** - Implements desired chain: flags > project config > global config > default opencode
- **Advisory infrastructure detection** - Returns warnings instead of overriding

**Trade-offs accepted:**
- **Breaking change** - Default changes from claude to opencode (aligns with cost optimization goal)
- **Behavior change** - Infrastructure detection warns instead of overrides (users must heed warnings)
- **Added complexity** - Global config fallback adds another layer (but provides flexibility)

**Implementation sequence:**
1. **Create `resolveBackend()` function** - Extract logic from spawn_cmd.go with new priority chain
2. **Add global config loading** - Load `~/.orch/config.yaml` for fallback defaults
3. **Update infrastructure detection** - Change from override to warning-only behavior
4. **Update tests** - Add unit tests for priority chain and edge cases

### Alternative Approaches Considered

**Option B: Refactor current logic as-is**
- **Pros:** Minimal behavior change, less risk
- **Cons:** Doesn't solve overlapping concerns, keeps `configSetBackend` boolean smell
- **When to use instead:** If time constraints prevent full redesign

**Option C: Config-driven backend selection only**
- **Pros:** Simplest, no flags except `--backend` override
- **Cons:** Removes useful `--opus` shortcut, less flexible
- **When to use instead:** If we want to force config-based defaults

**Rationale for recommendation:** Option A addresses all findings (overlapping concerns, misaligned priority, complex infrastructure logic) while providing testable, maintainable solution. The breaking changes align with stated goals (cost optimization, respecting user intent).

---

### Implementation Details

**What to implement first:**
- **Function signature design** - Define clear inputs/outputs with error handling
- **Priority chain implementation** - Code the 4-level priority with early returns
- **Global config integration** - Load user config with proper fallback logic

**Things to watch out for:**
- ⚠️ **Backward compatibility** - Ensure `--opus` flag still forces claude backend
- ⚠️ **Config precedence** - Project config must override global config
- ⚠️ **Warning display** - Infrastructure warnings must be visible but not blocking

**Areas needing further investigation:**
- **Cost impact analysis** - Measure actual cost difference claude vs opencode backends
- **User warning response** - Do users heed infrastructure warnings or ignore them?
- **Model-backend compatibility** - Should we validate combinations (e.g., opus + opencode)?

**Success criteria:**
- ✅ **Tests pass** - All existing spawn tests pass with new implementation
- ✅ **Priority chain verified** - Unit tests confirm flags > project > global > default
- ✅ **Infrastructure warnings** - Critical work triggers warnings but doesn't override
- ✅ **Default is opencode** - When no flags/config, backend = "opencode"

---

## References

**Files Examined:**
- [File path] - [What you looked at and why]
- [File path] - [What you looked at and why]

**Commands Run:**
```bash
# [Command description]
[command]

# [Command description]
[command]
```

**External Documentation:**
- [Link or reference] - [What it is and relevance]

**Related Artifacts:**
- **Decision:** [Path to related decision document] - [How it relates]
- **Investigation:** [Path to related investigation] - [How it relates]
- **Workspace:** [Path to related workspace] - [How it relates]

---

## Investigation History

**[YYYY-MM-DD HH:MM]:** Investigation started
- Initial question: [Original question as posed]
- Context: [Why this investigation was initiated]

**[YYYY-MM-DD HH:MM]:** [Milestone or significant finding]
- [Description of what happened or was discovered]

**[YYYY-MM-DD HH:MM]:** Investigation completed
- Status: [Complete/Paused with reason]
- Key outcome: [One sentence summary of result]
