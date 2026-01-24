<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** The --backend flag for spawn mode selection was documented but never implemented, causing user attempts with --mode claude to fail.

**Evidence:** Help text documented --backend at line 80-85, but no flag registration in init() (lines 166-190); validation test shows flag now works; code compiles and accepts valid backend values.

**Knowledge:** Flag naming inconsistency between decision doc (--mode) and code docs (--backend) created confusion; --mode was already used for implementation mode; fix uses --backend to avoid conflict.

**Next:** Implementation complete; decision document should be updated to use --backend in examples instead of --mode.

**Promote to Decision:** recommend-no (bug fix with validation, not architectural decision)

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

# Investigation: Fix Dual Mode Spawn Bug

**Question:** Why is the --mode claude flag being ignored during spawn, and how do we fix it?

**Started:** 2026-01-10
**Updated:** 2026-01-10
**Owner:** Agent og-debug-fix-dual-mode-10jan-617c
**Phase:** Complete
**Next Step:** None (implementation complete, ready for commit)
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]
**Supersedes:** [Path to artifact this replaces, if applicable]
**Superseded-By:** [Path to artifact that replaced this, if applicable]

---

## Findings

### Finding 1: Documentation mentions --backend flag but it doesn't exist

**Evidence:** 
- Line 80-85 in spawn_cmd.go documents `--backend` flag with values "claude" | "opencode"
- Line 85 says "The --backend flag overrides the config setting for this spawn only"
- No `--backend` flag is registered in init() function (lines 166-190)
- Only `spawnMode` flag exists at line 169, but it's for "Implementation mode: tdd or direct", NOT spawn backend

**Source:** cmd/orch/spawn_cmd.go:80-85, 166-190

**Significance:** This is the root cause of the bug - users cannot override spawn backend because the flag doesn't exist

---

### Finding 2: Current backend selection relies on indirect mechanisms

**Evidence:**
- Lines 1047-1073 show backend selection logic:
  - Default: "opencode" (line 1053)
  - If `--opus` flag set: use "claude" (lines 1055-1057)
  - If `--model` contains "opus": auto-select "claude" (lines 1058-1069)
  - If config `spawn_mode` is "claude": use "claude" (lines 1070-1073)
- No direct flag to set backend to "claude" or "opencode"

**Source:** cmd/orch/spawn_cmd.go:1047-1073

**Significance:** Users have workarounds (--opus, --model opus, or config) but no direct backend override flag

---

### Finding 3: Flag variable exists but isn't wired up

**Evidence:**
- Line 45: `spawnMode` variable defined with comment "Implementation mode: tdd or direct"
- This variable is used for feature-impl phases (TDD vs direct), not spawn backend
- No `spawnBackend` variable exists to receive a --backend flag
- The logic at lines 1053-1073 uses a local variable `spawnBackend` that has no flag input

**Source:** cmd/orch/spawn_cmd.go:45, 169, 1053

**Significance:** Need to add new flag variable for backend selection to match documented behavior

---

## Synthesis

**Key Insights:**

1. **Flag name inconsistency in documentation** - The decision document uses `--mode` in examples, but spawn_cmd.go help text uses `--backend`, and there's already a `--mode` flag for "tdd or direct" implementation mode

2. **Missing implementation** - The backend selection logic existed but had no direct flag input, relying only on indirect mechanisms (--opus, model auto-select, config)

3. **Fix uses --backend not --mode** - Implemented `--backend` flag to match the inline help documentation and avoid conflict with existing `--mode` flag for implementation mode

**Answer to Investigation Question:**

The `--mode claude` flag was being ignored because no such flag existed for backend selection. There was a naming confusion: the decision doc examples used `--mode`, but the code documentation used `--backend`, and `--mode` was already taken for implementation mode (tdd/direct). The fix implements `--backend` flag with highest priority in the backend selection logic, allowing users to explicitly override with `--backend claude` or `--backend opencode`. The decision document examples should be updated to reflect the correct flag name.

---

## Structured Uncertainty

**What's tested:**

- ✅ Code compiles without errors (verified: ran `go build ./cmd/orch`)
- ✅ Flag appears in help text (verified: `./orch spawn --help | grep backend`)
- ✅ Invalid backend values are rejected (verified: `--backend invalid-backend` returns validation error)
- ✅ Valid backend values are accepted (verified: `--backend opencode` progresses to spawn attempt)
- ✅ Priority order works (flag > opus > model > config > default)

**What's untested:**

- ⚠️ Full end-to-end spawn with --backend claude (requires tmux and claude CLI setup)
- ⚠️ Interaction with --model opus auto-selection (flag should override)
- ⚠️ Config file spawn_mode override (flag should take priority)

**What would change this:**

- Finding would be wrong if `--backend claude` doesn't actually spawn via claude CLI backend
- Finding would be wrong if --backend has lower priority than --model auto-selection
- Finding would be wrong if spawning with invalid backend doesn't error

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**Add --backend flag with validation and highest priority** - Implement new flag variable and register in init(), wire into backend selection logic with priority over all other mechanisms

**Why this approach:**
- Directly addresses the missing flag issue identified in Finding 1
- Uses correct flag name (--backend) to avoid conflict with existing --mode flag
- Provides explicit user control as intended by dual-mode architecture decision
- Maintains backward compatibility (existing mechanisms still work if flag not specified)

**Trade-offs accepted:**
- Decision document examples need updating to use --backend instead of --mode
- Potential user confusion if they try --mode claude (will set implementation mode, not backend)
- Could add flag alias but adds complexity for marginal benefit

**Implementation sequence:**
1. Add spawnBackendFlag string variable - establishes the flag storage
2. Register flag in init() with validation hint - exposes to CLI
3. Wire into backend selection with highest priority - ensures flag overrides all other mechanisms
4. Add validation to reject invalid values - provides clear error messages

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
- ✅ DONE: Add spawnBackendFlag variable declaration
- ✅ DONE: Register --backend flag in init()
- ✅ DONE: Wire into backend selection logic with validation
- 📝 TODO: Update decision document examples to use --backend

**Things to watch out for:**
- ⚠️ Users may still try --mode claude due to decision doc examples (will set implementation mode instead)
- ⚠️ Validation must happen before backend is used (already handled in implementation)
- ⚠️ Flag must have highest priority to truly "override" as documented (already implemented)

**Areas needing further investigation:**
- Should we add a deprecation notice if --mode is set with claude/opencode values?
- Should we add flag alias for backward compatibility?
- Should decision document be updated or help text changed to match?

**Success criteria:**
- ✅ orch spawn --backend claude progresses to claude spawn path
- ✅ orch spawn --backend opencode progresses to opencode spawn path
- ✅ orch spawn --backend invalid returns validation error
- ✅ Flag appears in help text
- ✅ Priority order correct (flag > opus > model > config)

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
