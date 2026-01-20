<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Project config now provides backend-specific model defaults when --model flag not provided.

**Evidence:** Tests show spawn uses config's opencode.model: sonnet with --backend opencode, and claude.model: opus with --backend claude, while --model flag overrides config.

**Knowledge:** Config wiring requires model resolution after backend determination; helper function cleanly separates logic while maintaining existing behavior.

**Next:** Implementation complete and tested; ready for orchestrator review and completion.

**Promote to Decision:** recommend-no (implementation complete, no architectural decision needed)

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

# Investigation: Wire Up Project Config Orch

**Question:** How to wire up project config (.orch/config.yaml) to spawn command for default model selection?

**Started:** 2026-01-19
**Updated:** 2026-01-19
**Owner:** feature-impl agent
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

### Finding 1: Current model resolution ignores project config

**Evidence:** In spawn_cmd.go:1025, `model.Resolve(spawnModel)` is called with the flag value. If `spawnModel` is empty, it returns `DefaultModel` (Opus) from model.go:71. Project config is loaded on line 1126 but only used for `SpawnMode` check.

**Source:** spawn_cmd.go:1025, model.go:70-71, spawn_cmd.go:1126

**Significance:** The project config has `opencode.model` and `claude.model` fields but they're ignored when no `--model` flag is provided.

### Finding 2: Config structure supports backend-specific models

**Evidence:** Config struct has `Claude.Model` and `OpenCode.Model` fields. Example config shows `opencode.model: flash` and `claude.model: opus`.

**Source:** config.go:30, config.go:36, .orch/config.yaml:1-11

**Significance:** The infrastructure already exists to store backend-specific model preferences, just not wired up to spawn command.

### Finding 3: Backend determination happens before model resolution

**Evidence:** Backend is determined between lines 1136-1184, model is resolved on line 1025. Config is loaded on line 1126.

**Source:** spawn_cmd.go:1025, spawn_cmd.go:1126, spawn_cmd.go:1136-1184

**Significance:** Need to load config earlier or pass it to model resolution logic to use backend-specific defaults.

---

### Finding 2: [Brief, descriptive title]

**Evidence:** [Concrete observations, data, examples]

**Source:** [File paths with line numbers, commands run, specific artifacts examined]

**Significance:** [Why this matters, what it tells us, implications for the investigation question]

---

### Finding 3: [Brief, descriptive title]

**Evidence:** [Concrete observations, data, examples]

**Source:** [File paths with line numbers, commands run, specific artifacts examined]

**Significance:** [Why this matters, what it tells us, implications for the investigation question]

---

## Synthesis

**Key Insights:**

1. **Config loading exists but model wiring missing** - Project config is already loaded in spawn command (line 1126) but only used for `SpawnMode` determination, not model selection.

2. **Backend-specific defaults available** - Config structure supports separate model defaults for `claude` and `opencode` backends, matching the actual use case.

3. **Simple wiring needed** - Need to check config after backend determination but before model resolution, using `projCfg.Claude.Model` or `projCfg.OpenCode.Model` based on `spawnBackend`.

**Answer to Investigation Question:**

To wire up project config to spawn command: modify spawn_cmd.go to check project config for model defaults when `--model` flag is empty. Use `projCfg.Claude.Model` for claude backend or `projCfg.OpenCode.Model` for opencode backend, falling back to current `DefaultModel` behavior if config fields are empty.

---

## Structured Uncertainty

**What's tested:**

- ✅ Config-based model selection works for opencode backend - `opencode.model: sonnet` in config used when `--backend opencode` with no `--model` flag
- ✅ Config-based model selection works for claude backend - `claude.model: opus` in config used when `--backend claude` with no `--model` flag  
- ✅ Explicit `--model` flag overrides config - `--model opus` used even when config has `opencode.model: sonnet`
- ✅ Flash model validation triggers correctly - when config has `opencode.model: flash`, error message shown

**What's untested:**

- ⚠️ Config loading errors - config.Load errors are ignored with `_`
- ⚠️ Missing config fields - behavior when `claude.model` or `opencode.model` fields are empty
- ⚠️ Invalid model in config - what happens if config has invalid model string

**What would change this:**

- If `resolveModelWithConfig` doesn't check backend before accessing config fields
- If config loading happens after model resolution (timing issue)
- If flash validation happens before config-based resolution

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**Implemented: Add resolveModelWithConfig helper function** - Created helper function that checks project config for backend-specific defaults when no explicit --model flag provided.

**Why this approach:**
- Minimal changes to existing code flow
- Reuses existing config loading infrastructure
- Maintains backward compatibility (explicit --model flag overrides config)
- Handles both claude and opencode backends separately

**Trade-offs accepted:**
- Config loading errors ignored (existing behavior)
- No validation of model strings in config (fail at spawn time)
- Duplicate config loading (line 1133 loads config again, but harmless)

**Implementation sequence:**
1. Added `resolveModelWithConfig` helper function after `validateModeModelCombo`
2. Declared `projCfg` variable with other spawn variables
3. Moved model resolution and flash validation after backend determination
4. Updated config loading to use assignment not declaration (line 1133)
5. Tested with various config scenarios

### Alternative Approaches Considered

**Option B: [Alternative approach]**
- **Pros:** [Benefits]
- **Cons:** [Why not recommended - reference findings]
- **When to use instead:** [Conditions where this might be better]

**Option C: [Alternative approach]**
- **Pros:** [Benefits]
- **Cons:** [Why not recommended - reference findings]
- **When to use instead:** [Conditions where this might be better]

**Rationale for recommendation:** [Brief synthesis of why Option A beats alternatives given investigation findings]

---

### Implementation Details

**What to implement first:**
- [Highest priority change based on findings]
- [Quick wins or foundational work]
- [Dependencies that need to be addressed early]

**Things to watch out for:**
- ⚠️ [Edge cases or gotchas discovered during investigation]
- ⚠️ [Areas of uncertainty that need validation during implementation]
- ⚠️ [Performance, security, or compatibility concerns to address]

**Areas needing further investigation:**
- [Questions that arose but weren't in scope]
- [Uncertainty areas that might affect implementation]
- [Optional deep-dives that could improve the solution]

**Success criteria:**
- ✅ [How to know the implementation solved the investigated problem]
- ✅ [What to test or validate]
- ✅ [Metrics or observability to add]

---

## References

**Files Examined:**
- `/Users/dylanconlin/Documents/personal/orch-go/cmd/orch/spawn_cmd.go` - Main spawn command implementation
- `/Users/dylanconlin/Documents/personal/orch-go/pkg/config/config.go` - Config structure definition
- `/Users/dylanconlin/Documents/personal/orch-go/pkg/model/model.go` - Model resolution logic
- `/Users/dylanconlin/Documents/personal/orch-go/.orch/config.yaml` - Example project config

**Commands Run:**
```bash
# Test spawn with opencode backend (uses config's opencode.model: sonnet)
./orch spawn --bypass-triage --no-track --backend opencode --force investigation "test"

# Test spawn with claude backend (uses config's claude.model: opus)  
./orch spawn --bypass-triage --no-track --backend claude --force investigation "test"

# Test explicit model flag overrides config
./orch spawn --bypass-triage --no-track --backend opencode --model opus --force investigation "test"

# Test flash model validation from config
# (when config had opencode.model: flash)
```

**External Documentation:**
- SPAWN_CONTEXT.md task description - Specified desired behavior

**Related Artifacts:**
- **Workspace:** `/Users/dylanconlin/Documents/personal/orch-go/.orch/workspace/og-feat-wire-up-project-19jan-ae88/` - This implementation workspace

---

## Investigation History

**[2026-01-19]:** Investigation started
- Initial question: How to wire up project config (.orch/config.yaml) to spawn command for default model selection?
- Context: Task from SPAWN_CONTEXT.md to implement config-based model selection

**[2026-01-19]:** Code analysis completed
- Found current model resolution ignores project config
- Identified config structure supports backend-specific models
- Discovered backend determination happens before model resolution

**[2026-01-19]:** Implementation completed
- Added `resolveModelWithConfig` helper function
- Moved model resolution after backend determination
- Tested with various config scenarios
- Verified explicit --model flag overrides config

**[2026-01-19]:** Investigation completed
- Status: Complete
- Key outcome: Successfully implemented config-based model selection with tests passing
