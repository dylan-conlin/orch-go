# Probe: Staleness Detection False Positives

**Model:** Orchestration Cost Economics
**Date:** 2026-02-28
**Trigger:** 11 stale spawn events after model was already updated (2026-02-27)

## Question

Why does the staleness detector keep firing for this model when the content is already current?

## Findings

### Finding 1: Tilde Path Expansion Bug (FIXED)

**Evidence:** `~/.local/share/opencode/auth.json` appears 42 times in staleness events as "deleted" despite existing.

**Root Cause:** `checkModelStaleness()` in `pkg/spawn/kbcontext.go` only checks `strings.HasPrefix(filePath, "/")` for absolute paths. Paths starting with `~` are treated as relative and prepended with `projectDir`, producing invalid paths like `~/Documents/personal/orch-go/~/.local/share/opencode/auth.json`.

**Fix Applied:** Added tilde expansion before the absolute path check. Now `~/` paths are expanded to the actual home directory via `os.UserHomeDir()`.

**Impact:** Eliminates the most common false positive (42 of 60 total events for this model).

### Finding 2: Same-Day Boundary Edge Case (NOT FIXED)

**Evidence:** After the model update commit at 20:33 on Feb 27, staleness events at 22:14 still show `changed: pkg/spawn/claude.go`.

**Root Cause:** `git log --since=2026-02-27` includes all commits from midnight Feb 27 onward. If a file was committed earlier the same day (before the model update), it still appears as "changed since". The `Last Updated` field is date-granular (YYYY-MM-DD), not timestamp-granular.

**Significance:** Minor — this only causes false "changed" on the same day as the model update, and resolves by the next day. Not worth fixing now.

### Finding 3: Historical References Are Not Deletions

**Evidence:** `pkg/spawn/backend.go` (35 events) — genuinely deleted file, but referenced in historical context ("Multi-file resolver replaces monolithic backend.go"), not as a current evidence file.

**Assessment:** The `extractCodeRefs()` function correctly scopes to the "Primary Evidence" section, so `backend.go` is NOT extracted by the current code. The 35 events are from before the model was restructured to scope extraction. No action needed.

## Model Impact

**Status:** Confirmed — No model content changes needed. Prior drift update (commit `4701d3097`) already brought the model current.

**Verification:**
- 39 aliases: matches `pkg/model/model.go` ✅
- 4 providers: matches code ✅
- `BuildClaudeLaunchCommand` with CLAUDE_CONFIG_DIR + BEADS_DIR: matches `pkg/spawn/claude.go` ✅
- `ShouldAutoSwitch` capacity checking: exists in `pkg/account/account.go:896` ✅
- `validateModel` flash gate: exists in `pkg/spawn/resolve.go:562` ✅
- All "Primary Evidence" files exist ✅
- No code commits to model-referenced files since model update ✅
