# Session Synthesis

**Agent:** og-inv-investigate-maintain-opencode-24mar-9294
**Issue:** orch-go-3nm4r
**Duration:** 2026-03-24
**Outcome:** success

---

## Plain-Language Summary

The OpenCode fork was created in January 2026 when OpenCode was the only way to spawn agents. It added memory management (preventing 8.4GB crashes), session metadata (tracking which beads issue belongs to which session), session TTL (auto-cleanup), and worker detection headers. Since February 19, 2026, the default backend flipped to Claude Code CLI, which doesn't use OpenCode at all. The fork is now 975 commits behind upstream, with 32 custom commits to maintain. Only 2 of 9 API integrations actually need fork-specific features, and those only apply to the secondary "opencode" backend path. The fork has become maintenance cost for a secondary code path.

---

## Verification Contract

See `VERIFICATION_SPEC.yaml` in this workspace. Key outcomes:
- Fork necessity assessment complete with quantified dependency metrics
- Probe created and model updated with Fork Relevance Assessment section
- Strategic options documented for orchestrator decision

---

## TLDR

The OpenCode fork is no longer critical infrastructure — it's maintenance overhead for a secondary backend. The claude backend (primary) has zero OpenCode dependency. Recommend: decide whether multi-model (opencode backend) is actively used; if not, consider dropping the opencode backend entirely to eliminate ~3,600 LoC and the fork maintenance burden.

---

## Delta (What Changed)

### Files Created
- `.kb/models/opencode-fork/probes/2026-03-24-probe-fork-necessity-assessment.md` — Full necessity assessment probe

### Files Modified
- `.kb/models/opencode-fork/model.md` — Added "Fork Relevance Assessment" section, updated Merged Probes table, updated Last Updated date

### Commits
- (pending — will commit with session completion)

---

## Evidence (What Was Observed)

- Fork is **32 commits ahead** and **975 commits behind** upstream (verified via `git rev-list --count`)
- Last sync was **2026-02-18** (5+ weeks ago)
- Default backend switched to `claude` in resolve.go: "Default backend is now claude since the default model is Anthropic (sonnet)"
- `2026-01-09-dual-spawn-mode-architecture.md` explicitly documents: "Anthropic OAuth ban (Feb 19, 2026) inverted the primary/secondary roles"
- Upstream has NO session metadata, NO time_ttl, NO ORCH_WORKER header, NO LRU eviction (verified via `git log upstream/dev --grep`)
- `pkg/opencode/` has 12 files, ~3,600+ LoC — only `CreateSession` (metadata+TTL) and `SetSessionMetadata` require fork features
- Claude backend (`pkg/spawn/claude.go`) spawns via tmux + `claude` CLI — no OpenCode session, no session ID, no metadata
- This very investigation was spawned with `Backend: claude` — confirming claude is the primary path

### Tests Run
```bash
# No code changes requiring tests — investigation/probe only
# Verified fork state via git commands against live repos
```

---

## Architectural Choices

No architectural choices — this was a pure investigation session.

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/models/opencode-fork/probes/2026-03-24-probe-fork-necessity-assessment.md` — Fork necessity assessment

### Decisions Made
- No decisions — investigation surfaces options for orchestrator

### Constraints Discovered
- Fork rebase cost is escalating (32 custom commits + upstream path restructuring)
- The opencode backend is the sole consumer of fork-specific features
- Claude Code hooks (`.claude/hooks/`) partially overlap with OpenCode plugin functionality

---

## Next (What Should Happen)

**Recommendation:** close

The investigation is complete. The orchestrator should decide which strategic option to pursue based on actual multi-model usage patterns:

1. **If multi-model is actively used** → keep fork, schedule a sync
2. **If multi-model is rarely used** → freeze the fork (stop syncing), plan eventual opencode backend removal
3. **If multi-model is never used** → drop opencode backend, remove fork dependency, remove ~3,600 LoC

### If Close
- [x] All deliverables complete
- [x] Probe file created with all 4 required sections
- [x] Probe merged into parent model
- [x] SYNTHESIS.md created
- [x] Ready for `orch complete orch-go-3nm4r`

---

## Unexplored Questions

- **How often is the opencode backend actually used?** Event log analysis would quantify this definitively. The events.jsonl file was empty/unavailable during this investigation.
- **Could OpenCode plugins be replaced by Claude Code hooks?** Gates, context injection, and observation patterns exist in both systems — a feature parity analysis would determine migration feasibility.
- **Would upstream accept PRs for the fork features?** Session metadata and TTL are generic enough. Memory management (LRU eviction) is critical for any multi-agent deployment. Worth evaluating.

---

## Friction

No friction — smooth session

---

## Session Metadata

**Skill:** investigation
**Model:** claude-opus-4-5
**Workspace:** `.orch/workspace/og-inv-investigate-maintain-opencode-24mar-9294/`
**Probe:** `.kb/models/opencode-fork/probes/2026-03-24-probe-fork-necessity-assessment.md`
**Beads:** `bd show orch-go-3nm4r`
