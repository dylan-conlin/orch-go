# Session Handoff - 22 Dec 2025

## TLDR

Fixed investigation failures by adding pre-spawn kb context check, but **critical bug: missing --global flag**. Also shipped phantom detection, liveness warnings, structured uncertainty. Created issues for knowledge extraction design.

---

## ⚠️ CRITICAL: Fix Before Spawning

**orch-go-cjl0** - Pre-spawn kb context check missing `--global` flag

The pre-spawn hook was implemented without `--global`, so cross-repo decisions won't be found. This was the exact problem we were fixing. One-line fix in `pkg/spawn/kbcontext.go:65`:

```go
// Current (broken):
cmd := exec.Command("kb", "context", query)

// Should be:
cmd := exec.Command("kb", "context", "--global", query)
```

**Fix this first** or the pre-spawn check is useless for cross-repo work.

---

## What Shipped This Session

| Feature | Description |
|---------|-------------|
| `pkg/state/reconcile.go` | `IsLive()` cross-referencing 4 state sources |
| `orch status` phantom detection | Shows "active" vs "phantom", accurate count |
| `orch complete` liveness warning | Prompts before closing if agent still running |
| Pre-spawn kb context check | Surfaces existing decisions (BUT missing --global) |
| Confidence → structured uncertainty | Templates updated in ~/.claude/skills/ |
| Beads db integrity fix | Removed 235 orphaned dependencies |
| Beads prefix parsing fix | Compound prefixes (orch-go-) now work |

---

## Blocking Issues

| Issue | Problem | Impact |
|-------|---------|--------|
| **orch-go-cjl0** | Pre-spawn missing --global | Cross-repo decisions invisible |
| **orch-go-abeu** | Source templates not updated | orch build skills will overwrite changes |

---

## Open Issues Created

| Issue | Description | Priority |
|-------|-------------|----------|
| orch-go-cjl0 | Pre-spawn missing --global flag | **P0 - fix first** |
| orch-go-abeu | Update orch-knowledge source templates | P1 |
| orch-go-jgc1 | Implement kb extract command | P2 |
| orch-go-p73c | Implement kb supersede command | P2 |
| orch-go-hkkh | Add lineage headers to templates | P2 |
| orch-go-jtat | Sync kb-cli hardcoded template | P2 |
| orch-go-oo1f | Retire stale orch-knowledge template | P2 |

---

## Key Findings

### Root Cause of Investigation Failures

Two investigations produced wrong conclusions because orchestrator didn't run `kb context --global` before spawning. Cross-repo decisions (skillc's stated vision) were invisible.

**Fix:** Pre-spawn hook that runs `kb context --global` and surfaces existing decisions. But the implementation missed `--global`.

### Template Fragmentation

Found 6 distinct template systems with significant divergence:
- kb-cli hardcoded (25 lines) vs ~/.kb/templates/ (234 lines)
- orch-knowledge embedded templates are stale
- Recommendation: Domain-based ownership (kb-cli for artifacts, orch-go for spawn-time)

### Knowledge Extraction Design

When extracting components to new repos, knowledge doesn't follow. Recommended:
- Inline lineage metadata (headers in artifacts)
- `kb extract` and `kb supersede` commands
- DEPRECATED.md pattern for project deprecation

---

## Decisions Made This Session

1. **Replace confidence scores with structured uncertainty** - `.kb/decisions/2025-12-22-replace-confidence-scores-with-structured-uncertainty.md`
2. **Inline lineage metadata over centralized registry** - Aligns with Session Amnesia principle

---

## Previous Epic to Revisit

**orch-go-4ztg** - Epic: Migrate skills to skillc

This epic was created based on a faulty investigation that said "skillc and orch build skills are complementary, no migration needed." The investigation didn't check skillc's stated vision (compile all AI artifacts including SKILL.md). After fixing the pre-spawn hook, may want to re-evaluate this epic.

---

## Account Status

- **personal:** ~30% used (resets in ~4d)
- **work:** Was at 90%, may have reset

---

## Stale Tmux Windows

Still have old tmux windows from completed agents. Run:
```bash
orch clean
```

---

## Next Session Priority

1. **Fix orch-go-cjl0** - Add --global to pre-spawn (5 min fix)
2. **Fix orch-go-abeu** - Update orch-knowledge source templates
3. **Re-evaluate orch-go-4ztg** - Should skillc replace orch build skills?
