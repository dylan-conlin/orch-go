# Synthesis: Model Handling Conflicts Between orch-go and opencode

**Investigation:** /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-21-inv-model-handling-conflicts-between-orch.md

**Date:** 2025-12-21

**Beads Issue:** orch-go-yase

---

## Executive Summary

Investigated why Gemini 3 Flash appears when spawning agents despite no explicit model selection. Found **three distinct bugs**:

1. **Wrong default model:** orch-go defaults to google/gemini-3-flash-preview (should be claude-opus-4-5-20251101 per orchestrator guidance)
2. **Inline spawns broken:** BuildSpawnCommand doesn't pass --model flag, ignoring user's selection
3. **Inconsistent implementation:** Only tmux spawns correctly pass model parameter

All three bugs stem from incomplete implementation of model parameter pass-through across spawn modes.

---

## Key Findings

### Root Cause: DefaultModel is Gemini 3 Flash

**Location:** pkg/model/model.go:18-21

```go
var DefaultModel = ModelSpec{
    Provider: "google",
    ModelID:  "gemini-3-flash-preview",
}
```

**Issue:** Conflicts with orchestrator skill guidance that recommends Opus as default for complex orchestration work.

**Impact:** Every spawn without --model flag uses Gemini instead of Opus.

---

### Bug #1: Inline Spawns Ignore Model Parameter

**Location:** pkg/opencode/client.go:127-137

BuildSpawnCommand builds this command:

```
opencode run --attach {serverURL} --format json --title {title} {prompt}
```

**Missing:** --model flag

**Impact:** Users running `orch spawn --inline investigation "task" --model opus` get Gemini anyway.

---

### Bug #2: Headless Spawns May Not Support Model

**Location:** pkg/opencode/client.go CreateSession API call

**Uncertainty:** Haven't verified if POST /session endpoint accepts model parameter.

**Impact:** Unknown - may affect daemon-driven spawns.

---

### What Works: Tmux Spawns

**Location:** pkg/tmux/tmux.go:92-106

BuildOpencodeAttachCommand correctly includes:

```go
if cfg.Model != "" {
    cmd += fmt.Sprintf(" --model %q", cfg.Model)
}
```

**This is the pattern to replicate** in BuildSpawnCommand.

---

## Recommended Implementation

### Three-Part Fix (Priority Order)

**1. Change DefaultModel to Opus** ⭐ (Quick win)

- File: pkg/model/model.go:18-21
- Change: `ModelID: "claude-opus-4-5-20251101"`
- Rationale: Aligns with orchestrator guidance
- Risk: Low - users can still use --model flash if needed

**2. Fix BuildSpawnCommand** ⭐ (Fixes inline spawns)

- File: pkg/opencode/client.go:127-137
- Add --model flag when provided
- Pattern: Copy from BuildOpencodeAttachCommand
- Risk: Low - only affects inline spawns

**3. Verify/Fix CreateSession API** (Fixes headless spawns)

- Files: opencode server.ts + orch-go client.go
- Investigate if model parameter is supported
- If not, add it with backward compatibility
- Risk: Medium - requires coordination with opencode repo

---

## Discovered Issues (For Beads)

1. **og-bug-default-model-gemini** (triage:ready)
   - DefaultModel should be Opus, not Gemini
   - Quick fix: one-line change
2. **og-bug-inline-spawn-no-model** (triage:ready)
   - BuildSpawnCommand doesn't pass --model flag
   - Medium effort: refactor command builder
3. **og-inv-headless-spawn-model-support** (triage:review)
   - Need to verify CreateSession API accepts model
   - Investigation before implementation

---

## Testing Validation

**After fixes, verify:**

```bash
# Test 1: Default is Opus
orch spawn investigation "test"
# Expected: Uses claude-opus-4-5-20251101

# Test 2: Inline respects --model
orch spawn --inline investigation "test" --model flash
# Expected: Uses gemini-2.5-flash

# Test 3: Headless respects --model (if supported)
orch spawn --headless investigation "test" --model sonnet
# Expected: Uses claude-sonnet-4-5-20250929
```

---

## Next Actions

1. Review this synthesis with orchestrator
2. Create beads issues for discovered bugs (or proceed with fixes if approved)
3. Implement fixes in priority order (DefaultModel → BuildSpawnCommand → CreateSession)
4. Add tests to prevent regression

---

## Confidence

**High (85%)** - Code review is thorough and findings are directly observable. Haven't tested runtime behavior or verified CreateSession API, but core bugs are confirmed.
