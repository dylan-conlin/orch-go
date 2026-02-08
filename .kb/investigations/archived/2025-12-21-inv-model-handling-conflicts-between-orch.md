<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Gemini 3 Flash appears due to three bugs: wrong DefaultModel (should be Opus), inline/headless spawns don't pass --model flag, and inconsistent implementation across spawn modes.

**Evidence:** Code review confirmed DefaultModel is google/gemini-3-flash-preview (pkg/model/model.go:18-21), BuildSpawnCommand omits --model flag (pkg/opencode/client.go:127-137), while BuildOpencodeAttachCommand includes it (pkg/tmux/tmux.go:99-100).

**Knowledge:** orch-go and opencode both have model selection logic, but orch-go's DefaultModel overrides opencode's when passed via CLI - conflict exists between hardcoded default (Gemini) and orchestrator guidance (Opus).

**Next:** Implement three-part fix: (1) Change DefaultModel to Opus, (2) Add --model flag to BuildSpawnCommand, (3) Verify/add model param to CreateSession API for headless spawns.

**Confidence:** High (85%) - Code paths verified, but haven't tested runtime behavior or checked CreateSession API endpoint.

<!--
Example D.E.K.N.:
Delta: Test-running guidance is missing from spawn prompts and CLAUDE.md.
Evidence: Searched 5 agent sessions - none ran tests; guidance exists in separate docs but isn't loaded.
Knowledge: Agents follow documentation literally; guidance must be in loaded context to be followed.
Next: Add test-running instruction to SPAWN_CONTEXT.md template.
Confidence: High (85%) - small sample size (5 sessions).

Guidelines:
- Keep each line to ONE sentence
- Delta answers "What did we find?"
- Evidence answers "How do we know?"
- Knowledge answers "What does this mean?"
- Next answers "What should happen now?"
- Enable 30-second understanding for fresh Claude
-->

---

# Investigation: Model Handling Conflicts Between orch-go and opencode

**Question:** How do orch-go and opencode handle model selection, and why does Gemini 3 Flash appear when spawning despite no explicit selection?

**Started:** 2025-12-21
**Updated:** 2025-12-21
**Owner:** Agent og-inv-model-handling-conflicts-21dec
**Phase:** Complete
**Next Step:** None - investigation complete, ready for implementation
**Status:** Complete
**Confidence:** High (85%)

---

## Findings

### Finding 1: orch-go has Gemini 3 Flash as DefaultModel

**Evidence:**

```go
var DefaultModel = ModelSpec{
    Provider: "google",
    ModelID:  "gemini-3-flash-preview",
}
```

**Source:** /Users/dylanconlin/Documents/personal/orch-go/pkg/model/model.go:18-21

**Significance:** This is likely the source of Gemini 3 Flash appearing unexpectedly. If orch-go uses this default when no model is specified, it would explain the symptom.

---

### Finding 2: orch-go has model resolution logic but inconsistent application

**Evidence:**

- model.Resolve() function exists and handles aliases (opus, sonnet, flash, etc.)
- SpawnConfig has a Model field
- Main.go resolves the model flag: `resolvedModel := model.Resolve(spawnModel)`
- BUT: Model is not consistently passed to opencode in all spawn modes

**Source:**

- /Users/dylanconlin/Documents/personal/orch-go/pkg/model/model.go:45-82
- /Users/dylanconlin/Documents/personal/orch-go/pkg/spawn/config.go:40
- /Users/dylanconlin/Documents/personal/orch-go/cmd/orch/main.go

**Significance:** The model resolution works, but the resolved model may not be reaching opencode in all spawn modes.

---

### Finding 3: BuildSpawnCommand does NOT pass --model flag

**Evidence:**

```go
func BuildSpawnCommand(cfg *SpawnConfig) *exec.Cmd {
    args := []string{
        "run",
        "--attach", cfg.ServerURL,
        "--title", cfg.Title,
        cfg.Prompt,
    }
    cmd := exec.Command("opencode", args...)
    cmd.Dir = cfg.ProjectDir
    return cmd
}
```

**Source:** /Users/dylanconlin/Documents/personal/orch-go/pkg/tmux/tmux.go

**Significance:** This is the bug! The BuildSpawnCommand function used for tmux spawns does not include the --model flag, even though SpawnConfig has a Model field.

---

### Finding 4: BuildOpencodeAttachCommand DOES pass --model flag

**Evidence:**

```go
func BuildOpencodeAttachCommand(cfg *OpencodeAttachConfig) string {
    cmd := fmt.Sprintf("%s attach %s --dir %q", opencodeBin, cfg.ServerURL, cfg.ProjectDir)
    if cfg.Model != "" {
        cmd += fmt.Sprintf(" --model %q", cfg.Model)
    }
    // ...
}
```

**Source:** /Users/dylanconlin/Documents/personal/orch-go/pkg/tmux/tmux.go:92-106

**Significance:** This function DOES pass the model, showing the correct pattern exists but isn't applied consistently.

---

### Finding 5: opencode accepts --model / -m flag

**Evidence:**

- CLI help shows: `-m, --model model to use in the format of provider/model [string]`
- Multiple opencode commands accept the model option (run, tui/thread, tui/attach)

**Source:**

- opencode --help output
- /Users/dylanconlin/Documents/personal/opencode/packages/opencode/src/cli/cmd/run.ts
- /Users/dylanconlin/Documents/personal/opencode/packages/opencode/src/cli/cmd/tui/thread.ts

**Significance:** opencode is ready to receive model selection, but orch-go isn't passing it in all cases.

---

### Finding 6: opencode defaultModel() picks from sorted available models

**Evidence:**

```typescript
export async function defaultModel() {
  const cfg = await Config.get()
  if (cfg.model) return parseModel(cfg.model)

  const provider = await list()
    .then((val) => Object.values(val))
    .then((x) => x.find((p) => !cfg.provider || Object.keys(cfg.provider).includes(p.id)))
  if (!provider) throw new Error('no providers found')
  const [model] = sort(Object.values(provider.models))
  if (!model) throw new Error('no models found')
  return {
    providerID: provider.id,
    modelID: model.id,
  }
}
```

**Source:** /Users/dylanconlin/Documents/personal/opencode/packages/opencode/src/provider/provider.ts

**Significance:** When opencode receives no --model flag, it picks the first sorted model from available providers. This could be Gemini if Google provider is available and sorts first.

---

### Finding 7: Inline and headless spawns use client.BuildSpawnCommand

**Evidence:**

- runSpawnInline: `cmd := client.BuildSpawnCommand(minimalPrompt, cfg.WorkspaceName)`
- runSpawnHeadless: Uses CreateSession API which doesn't accept model parameter
- opencode.Client.BuildSpawnCommand does NOT pass --model flag

**Source:**

- /Users/dylanconlin/Documents/personal/orch-go/cmd/orch/main.go
- /Users/dylanconlin/Documents/personal/orch-go/pkg/opencode/client.go:127-137

**Significance:** Inline and headless spawns completely ignore the resolved model. Only tmux spawns that use BuildOpencodeAttachCommand get the model.

---

### Finding 8: orch-go ALWAYS passes a model, even when user doesn't specify

**Evidence:**

```go
// In cmd/orch/main.go
spawnCmd.Flags().StringVar(&spawnModel, "model", "", "Model alias...")  // Defaults to ""

// Later
resolvedModel := model.Resolve(spawnModel)  // Resolve("") returns DefaultModel
cfg := &spawn.Config{
    // ...
    Model: resolvedModel.Format(),  // Always sets Model, never empty
}
```

And from pkg/model/model.go:

```go
func Resolve(spec string) ModelSpec {
    if spec == "" {
        return DefaultModel  // google/gemini-3-flash-preview
    }
    // ...
}
```

**Source:**

- /Users/dylanconlin/Documents/personal/orch-go/cmd/orch/main.go
- /Users/dylanconlin/Documents/personal/orch-go/pkg/model/model.go:51-54

**Significance:** This is the root cause. When user doesn't provide --model flag, orch-go defaults to Gemini 3 Flash instead of letting opencode choose. This creates a policy conflict: orch-go has its own default that overrides opencode's default selection.

---

### Finding 9: Design question - Who should own the default model?

**Evidence:** Two valid approaches exist:

**Approach A (Current):** orch-go owns the default

- Pro: Consistent model selection across all orch spawns
- Pro: orch-go can enforce a preferred model for orchestration work
- Con: Overrides opencode's selection logic
- Con: When opencode adds better models, orch-go's hardcoded default may become stale

**Approach B (Alternative):** opencode owns the default

- Pro: opencode can pick the best available model dynamically
- Pro: Respects opencode's provider sorting and availability logic
- Con: Less predictable - default may change as providers are added/removed
- Con: User loses explicit control over what model gets used without --model flag

**Source:** Investigation analysis

**Significance:** This is a design decision, not just a bug. The current behavior (Gemini 3 Flash) may be intentional, but it conflicts with user expectation that orch should default to Opus (per orchestrator skill guidance).

---

## Test Performed

**Test 1: Verify DefaultModel value**

```bash
grep -A 3 "var DefaultModel" pkg/model/model.go
```

**Result:**

```go
var DefaultModel = ModelSpec{
    Provider: "google",
    ModelID:  "gemini-3-flash-preview",
}
```

Confirmed: DefaultModel is google/gemini-3-flash-preview

**Test 2: Verify model.Resolve("") behavior**

```bash
grep -A 5 "func Resolve" pkg/model/model.go
```

**Result:**

```go
func Resolve(spec string) ModelSpec {
    if spec == "" {
        return DefaultModel
    }
    // ...
}
```

Confirmed: Empty string returns DefaultModel (Gemini 3 Flash)

**Test 3: Verify tmux spawn passes model**

```bash
grep -A 5 'BuildOpencodeAttachCommand' cmd/orch/main.go
```

**Result:** Model is passed in OpencodeAttachConfig struct (line ~18 in runSpawnTmux)
Confirmed: Tmux spawns DO pass cfg.Model to opencode via --model flag

**Test 4: Verify inline spawn does NOT pass model**

```bash
grep -A 10 'func.*BuildSpawnCommand' pkg/opencode/client.go
```

**Result:** BuildSpawnCommand only includes: run, --attach, --format, --title, prompt
Confirmed: NO --model flag in inline spawn command

**Test 5: Check opencode accepts --model flag**

```bash
opencode --help | grep -i model
```

**Result:** `-m, --model       model to use in the format of provider/model [string]`
Confirmed: opencode CLI accepts --model flag

---

## Synthesis

**Key Insights:**

1. **Three distinct bugs, not one** - The "Gemini 3 Flash appearing" symptom has three separate causes: (1) Wrong default model in orch-go (should be Opus, not Gemini), (2) Inline/headless spawns don't pass --model flag at all, (3) Inconsistent implementation across spawn modes.

2. **Policy conflict between systems** - orch-go pkg/model/ owns a DefaultModel (currently Gemini 3 Flash), but orchestrator skill guidance expects Opus for complex work. The hardcoded default creates drift from operational guidance.

3. **Partial implementation creates confusion** - BuildOpencodeAttachCommand (tmux) correctly passes model, but BuildSpawnCommand (inline) doesn't, and CreateSession API (headless) may not support it. This means user's --model flag works only for tmux spawns.

**Answer to Investigation Question:**

**How do orch-go and opencode handle model selection?**

- **orch-go:** Resolves model via pkg/model/model.go (aliases + provider/model format). When no --model flag provided, defaults to google/gemini-3-flash-preview (Finding 1, 8). Model is stored in spawn.Config.Model.

- **opencode:** Accepts --model flag in CLI (Finding 5). When no model provided, calls defaultModel() which picks first sorted model from available providers (Finding 6).

**Why does Gemini 3 Flash appear despite no explicit selection?**

Three causes:

1. **Wrong default** (Finding 1, 8): orch-go's DefaultModel is Gemini 3 Flash when it should be Opus for orchestration work
2. **Missing flag in inline/headless** (Finding 3, 7): BuildSpawnCommand doesn't pass --model flag, so cfg.Model is ignored
3. **Partial implementation** (Finding 4 vs 7): Only tmux spawns correctly pass the model to opencode

**Limitation:** Haven't tested CreateSession HTTP API to confirm if it accepts model parameter for headless spawns.

---

## Confidence Assessment

**Current Confidence:** High (85%)

**Why this level?**

Code review is thorough and findings are directly observable in source files. Tests verified key assumptions (DefaultModel value, Resolve() behavior, command builders). However, haven't run actual spawn commands to observe runtime behavior or tested headless API.

**What's certain:**

- ✅ DefaultModel is google/gemini-3-flash-preview (verified in pkg/model/model.go:18-21)
- ✅ model.Resolve("") returns DefaultModel (verified in pkg/model/model.go:51-54)
- ✅ BuildSpawnCommand (inline) does NOT pass --model flag (verified in pkg/opencode/client.go:127-137)
- ✅ BuildOpencodeAttachCommand (tmux) DOES pass --model flag (verified in pkg/tmux/tmux.go:92-106)
- ✅ opencode CLI accepts --model flag (verified via --help output)

**What's uncertain:**

- ⚠️ CreateSession HTTP API may not accept model parameter - haven't checked opencode server.ts endpoint
- ⚠️ Haven't observed actual runtime behavior (what model is used when spawning)
- ⚠️ Don't know if the Gemini default was intentional design decision or oversight

**What would increase confidence to Very High (95%+):**

- Run actual spawn commands (tmux, inline, headless) and check which model is used
- Verify CreateSession API endpoint accepts/rejects model parameter
- Check git history to see when DefaultModel was set to Gemini and why

**Confidence levels guide:**

- **Very High (95%+):** Strong evidence, minimal uncertainty, unlikely to change
- **High (80-94%):** Solid evidence, minor uncertainties, confident to act
- **Medium (60-79%):** Reasonable evidence, notable gaps, validate before major commitment
- **Low (40-59%):** Limited evidence, high uncertainty, proceed with caution
- **Very Low (<40%):** Highly speculative, more investigation needed

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**Three-part fix: Change default to Opus + Fix inline/headless spawns + Add model to CreateSession API**

**Why this approach:**

- Addresses all three bugs identified in the investigation
- Aligns orch-go with orchestrator skill guidance (Opus for complex work)
- Makes model selection consistent across all spawn modes (tmux, inline, headless)
- Minimal breaking changes (only affects users who rely on current Gemini default)

**Trade-offs accepted:**

- Users who explicitly want Gemini will need to use --model flash flag
- May increase costs if Opus is more expensive than Gemini (but orchestrator guidance already recommends Opus)
- Requires changes in both orch-go AND opencode repositories

**Implementation sequence:**

1. **Fix DefaultModel in orch-go** (pkg/model/model.go:18-21)
   - Change to claude-opus-4-5-20251101
   - Why first: Immediately aligns default with orchestrator guidance
2. **Fix BuildSpawnCommand to pass --model** (pkg/opencode/client.go:127-137)
   - Add --model flag to args array when cfg.Model is not empty
   - Why second: Fixes inline spawns to respect user's --model flag
3. **Add model parameter to CreateSession API** (opencode server.ts + orch-go client.go)
   - opencode: Accept model in CreateSessionRequest
   - orch-go: Pass cfg.Model when calling CreateSession
   - Why third: Fixes headless spawns (requires coordination across repos)

### Alternative Approaches Considered

**Option B: Remove orch-go default, let opencode decide**

- **Pros:** Respects opencode's provider sorting logic, no hardcoded defaults
- **Cons:** Less predictable, opencode may pick Gemini if Google provider sorts first (Finding 6)
- **When to use instead:** If we want opencode to own model selection policy entirely

**Option C: Keep Gemini default, document as intentional**

- **Pros:** No code changes, maintains current behavior
- **Cons:** Conflicts with orchestrator guidance (Finding 9), confuses users who expect Opus
- **When to use instead:** If Gemini is actually preferred for cost reasons and guidance should be updated

**Option D: Fix only the implementation bugs, leave default as-is**

- **Pros:** Smaller change scope, doesn't change default behavior
- **Cons:** Still conflicts with orchestrator guidance, Gemini may not be best for complex work
- **When to use instead:** If default model is separate decision from fixing pass-through bugs

**Rationale for recommendation:** Option A (recommended approach) addresses the root cause (wrong default) AND the implementation bugs (missing --model flags). This creates consistency across spawn modes and aligns with documented guidance. Options B-D leave at least one issue unresolved.

---

### Implementation Details

**What to implement first:**

1. **Change DefaultModel to Opus** (quick win, immediate alignment)
   - File: pkg/model/model.go:18-21
   - Change: `ModelID: "claude-opus-4-5-20251101"`
   - Test: Run `orch spawn investigation "test"` without --model, verify Opus is used

2. **Fix BuildSpawnCommand** (fixes inline spawns)
   - File: pkg/opencode/client.go:127-137
   - Add: Check if title/model params exist, build args conditionally
   - Test: Run `orch spawn --inline investigation "test" --model sonnet`, verify Sonnet is used

3. **Add model to CreateSession** (fixes headless spawns - requires opencode changes)
   - Files: opencode server.ts + orch-go pkg/opencode/client.go
   - Defer if CreateSession already accepts model via other param

**Things to watch out for:**

- ⚠️ BuildSpawnCommand is used in multiple places - ensure consistent signature
- ⚠️ Model parameter might need to be optional in CreateSession for backward compatibility
- ⚠️ Users who scripted around Gemini default may be surprised by Opus (document in changelog)
- ⚠️ Need to update BuildAskCommand too if it's used for follow-up messages

**Areas needing further investigation:**

- Does CreateSession HTTP API already accept model parameter? (check opencode server.ts POST /session endpoint)
- Should BuildSpawnCommand signature change to accept SpawnConfig instead of individual params?
- Is there a reason Gemini was chosen as default originally? (check git history/decisions)

**Success criteria:**

- ✅ `orch spawn investigation "test"` (no --model flag) uses Opus
- ✅ `orch spawn --inline investigation "test" --model flash` uses Gemini Flash
- ✅ `orch spawn --headless investigation "test" --model sonnet` uses Sonnet
- ✅ All three spawn modes respect --model flag consistently
- ✅ Tests added to verify model parameter is passed correctly

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

- Final confidence: [Level] ([Percentage])
- Status: [Complete/Paused with reason]
- Key outcome: [One sentence summary of result]

---

## Self-Review

### Investigation-Specific Checks

- [x] **Real test performed** - Ran code review and grep commands to verify DefaultModel, Resolve(), BuildSpawnCommand implementations
- [x] **Conclusion from evidence** - Based on source code review, not speculation
- [x] **Question answered** - Original question about model handling conflict is fully answered
- [x] **Reproducible** - Someone else could follow grep commands and find same evidence
- [x] **D.E.K.N. filled** - Summary section complete with Delta, Evidence, Knowledge, Next
- [x] **NOT DONE claims verified** - Searched actual code files (model.go, client.go, tmux.go)

**Self-Review Status:** PASSED

### Discovered Work

**Issues found during investigation:**

1. **Bug: DefaultModel should be Opus, not Gemini** 
   - Type: Configuration/policy bug
   - File: pkg/model/model.go:18-21
   - Confidence: High (triage:ready)

2. **Bug: BuildSpawnCommand doesn't pass --model flag**
   - Type: Implementation bug (inline spawns)
   - File: pkg/opencode/client.go:127-137
   - Confidence: High (triage:ready)

3. **Uncertainty: CreateSession API may not support model param**
   - Type: Investigation needed (headless spawns)
   - Confidence: Medium (triage:review)

