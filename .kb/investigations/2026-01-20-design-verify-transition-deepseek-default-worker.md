<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** The `spawn_mode: opencode` config is NOT functional - backend selection logic has a bug that ignores the opencode setting, causing all spawns to default to Claude mode.

**Evidence:** Live testing showed spawns use Claude mode even when config has `spawn_mode: opencode`. Code analysis shows line 1188 in spawn_cmd.go only checks `projCfg.SpawnMode == "claude"` - there's no branch that sets opencode backend from config.

**Knowledge:** The transition to DeepSeek as default worker backend is incomplete. Config was updated but code doesn't implement it. DeepSeek V3 function calling DOES work when explicitly spawned with `--backend opencode`.

**Next:** Fix the bug in spawn_cmd.go to respect `spawn_mode: opencode` config, then re-verify daemon spawns use DeepSeek.

**Promote to Decision:** recommend-no (bug fix, not architectural - the decision to use DeepSeek was already made)

---

# Investigation: Verify Transition DeepSeek Default Worker

**Question:** Is the transition to DeepSeek as the default worker backend complete and operational in headless mode?

**Started:** 2026-01-20
**Updated:** 2026-01-20
**Owner:** Architect agent (spawned)
**Phase:** Complete
**Next Step:** Bug fix required in spawn_cmd.go
**Status:** Complete

---

## Findings

### Finding 1: Config correctly specifies DeepSeek as default

**Evidence:** `.orch/config.yaml` contains:
```yaml
spawn_mode: opencode
opencode:
    model: deepseek
    server: http://localhost:4096
```

**Source:** `/Users/dylanconlin/Documents/personal/orch-go/.orch/config.yaml`

**Significance:** The configuration intent is correct - DeepSeek is meant to be the default for opencode mode.

---

### Finding 2: Backend selection logic has critical bug

**Evidence:** In `spawn_cmd.go` lines 1143-1191:
- Line 1143: Default is `spawnBackend := "claude"` (hardcoded)
- Line 1188: Only checks `projCfg.SpawnMode == "claude"`, never sets opencode
- Result: `spawn_mode: opencode` in config has no effect

Missing code path:
```go
// This branch does NOT exist:
} else if projCfg != nil && projCfg.SpawnMode == "opencode" {
    spawnBackend = "opencode"
}
```

**Source:** `cmd/orch/spawn_cmd.go:1143-1191`

**Significance:** This is a critical bug - the opencode config value is simply ignored, causing ALL spawns to default to Claude mode regardless of config.

---

### Finding 3: DeepSeek works when explicitly specified

**Evidence:** Test spawn with `--backend opencode` flag:
```
$ orch spawn --bypass-triage --no-track --backend opencode investigation "test"
Spawned agent (headless):
  Model:      deepseek/deepseek-chat
```

Without explicit flag (bug behavior):
```
$ orch spawn --bypass-triage --no-track investigation "test"
Spawned agent in Claude mode (tmux):
  Window:     workers-orch-go:3
```

**Source:** Live spawn tests performed 2026-01-20

**Significance:** DeepSeek headless mode IS functional - the only issue is the config isn't respected.

---

### Finding 4: DeepSeek V3 function calling verified working

**Evidence:** Previous investigations confirm DeepSeek V3 tool use:
- `.orch/workspace/og-inv-test-deepseek-tool-18jan-1bb9/SYNTHESIS.md`: Read, Grep, Bash tools all verified working
- `.orch/workspace/og-inv-test-deepseek-v3-19jan-25d3/SYNTHESIS.md`: Additional verification of function calling

**Source:** Prior investigation synthesis documents

**Significance:** DeepSeek V3 is operationally ready for worker tasks - the tooling works, only the config plumbing is broken.

---

### Finding 5: Escape hatch detection works as designed

**Evidence:** The `isInfrastructureWork()` function at line 2270+ correctly detects infrastructure keywords and applies escape hatch (claude + tmux). This is intentional behavior for self-referential work.

Keywords checked: "opencode", "orch-go", "spawn", "daemon", etc.

**Source:** `cmd/orch/spawn_cmd.go:2270-2333`

**Significance:** The escape hatch is NOT causing the issue - it's working as designed for infrastructure work. The bug is in the baseline config handling.

---

## Synthesis

**Key Insights:**

1. **Config-Code Mismatch** - The config was updated to use DeepSeek but the code doesn't implement reading `spawn_mode: opencode`. This is a classic "config without implementation" bug.

2. **Default Hardcoding** - The hardcoded default `spawnBackend := "claude"` at line 1143 means any missing config branch falls through to Claude mode.

3. **Operational Readiness** - DeepSeek V3 function calling is verified working. Once the bug is fixed, the transition will be complete.

**Answer to Investigation Question:**

**No, the transition is NOT complete.** The config change was made (`spawn_mode: opencode`, `opencode.model: deepseek`) but the code doesn't respect the `spawn_mode: opencode` value. All spawns currently default to Claude mode unless `--backend opencode` is explicitly specified.

**Fix required:** Add a code path in spawn_cmd.go to set `spawnBackend = "opencode"` when `projCfg.SpawnMode == "opencode"`.

---

## Structured Uncertainty

**What's tested:**

- ✅ Config file correctly specifies DeepSeek (verified: read .orch/config.yaml)
- ✅ DeepSeek spawns work with `--backend opencode` (verified: live spawn test)
- ✅ DeepSeek V3 function calling works (verified: prior investigations with tool use)
- ✅ Code analysis shows missing opencode config branch (verified: read spawn_cmd.go)

**What's untested:**

- ⚠️ Daemon behavior after fix (not tested - requires code change first)
- ⚠️ Long-running DeepSeek agent stability (prior tests were short)
- ⚠️ Cost comparison in production workloads (theoretical only)

**What would change this:**

- If there's another code path setting opencode backend that was missed
- If the config loading fails silently (but test showed config is loaded correctly)

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation.

### Recommended Approach ⭐

**Fix spawn_cmd.go to respect spawn_mode: opencode** - Add missing else-if branch to set opencode backend from config.

**Why this approach:**
- Minimal change (2-3 lines of code)
- Fixes the actual bug without changing architecture
- Aligns code behavior with documented config options

**Trade-offs accepted:**
- None - this is a straightforward bug fix

**Implementation sequence:**
1. Add `else if projCfg.SpawnMode == "opencode"` branch after line 1188
2. Set `spawnBackend = "opencode"` in that branch
3. Test daemon spawns use DeepSeek by default

### Code Fix

```go
// After line 1188-1191 (the claude check), add:
} else if projCfg != nil && projCfg.SpawnMode == "opencode" {
    // Config default: respect project spawn_mode setting
    spawnBackend = "opencode"
}
```

---

### Implementation Details

**What to implement first:**
- The 3-line code fix in spawn_cmd.go
- Unit test for config-driven backend selection

**Things to watch out for:**
- ⚠️ Ensure the fix doesn't interfere with escape hatch detection
- ⚠️ Verify daemon spawns still work after fix

**Success criteria:**
- ✅ Spawn without `--backend` flag uses opencode when config says opencode
- ✅ Daemon-spawned agents use DeepSeek model
- ✅ Escape hatch still works for infrastructure keywords

---

## References

**Files Examined:**
- `.orch/config.yaml` - Project config with spawn_mode and model settings
- `cmd/orch/spawn_cmd.go:1130-1200` - Backend selection logic
- `cmd/orch/spawn_cmd.go:2270-2333` - Infrastructure work detection
- `pkg/config/config.go` - Config struct definitions

**Commands Run:**
```bash
# Test with explicit backend
orch spawn --bypass-triage --no-track --backend opencode investigation "test"
# Result: Uses DeepSeek headless

# Test without backend (should use config)
orch spawn --bypass-triage --no-track investigation "test"
# Result: Uses Claude mode (BUG!)

# Test with model flag
orch spawn --bypass-triage --no-track --model deepseek investigation "test"
# Result: Uses Claude mode (BUG - model flag doesn't set backend)
```

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2026-01-18-research-compare-deepseek-models-anthropic-models.md` - DeepSeek capability analysis
- **Investigation:** `.kb/investigations/2026-01-19-inv-test-deepseek-v3-function-calling.md` - Function calling verification
- **Workspace:** `.orch/workspace/og-inv-test-deepseek-v3-19jan-25d3/` - DeepSeek test evidence

---

## Investigation History

**2026-01-20 09:00:** Investigation started
- Initial question: Verify DeepSeek transition is complete
- Context: Spawned to confirm operational readiness

**2026-01-20 09:15:** Found config correctly set
- Config has spawn_mode: opencode and opencode.model: deepseek

**2026-01-20 09:30:** Discovered backend selection bug
- Code only checks for SpawnMode == "claude"
- No branch sets opencode from config

**2026-01-20 09:45:** Verified DeepSeek works with explicit flag
- `--backend opencode` successfully spawns DeepSeek agents
- Bug is isolated to config handling

**2026-01-20 10:00:** Investigation completed
- Status: Complete
- Key outcome: Transition NOT complete - config bug prevents DeepSeek from being used by default
