# Session Handoff: Entropy Spiral Recovery

**Date:** 2026-02-13
**Branch:** master (at `67141cce`)
**Baseline:** Jan 18 commit `0bca3dec` — reverted from 1163-commit entropy spiral

---

## What Happened

orch-go suffered a 26-day entropy spiral (Jan 18 — Feb 12): 1163 agent commits, 5.4M LOC churn, zero human commits. This was a repeat of the Dec 27-Jan 2 spiral with identical root causes and unimplemented mitigations. We reverted to the Jan 18 baseline, preserved evidence on branch `entropy-spiral-feb2026`, and began selective recovery of valuable work.

## Current State

### orch-go (master at `67141cce`)
- **Baseline:** Jan 18 (`0bca3dec`) — claude backend works, spawn works, all core commands functional
- **Added post-revert:**
  - `.beads/config.yaml` — added `issue-prefix: orch-go` (required for bd)
  - `.kb/models/` — recovered 4 new models + 9 validated probes from spiral
  - `pkg/spawn/probes.go` + `probes_test.go` — probe infrastructure restored, builds and tests pass
  - `pkg/spawn/kbcontext.go` — model content injection + probe formatting merged
  - `.kb/investigations/` — postmortem, skill audit, recovery audit
- **Beads:** Fresh JSONL (old 2170-entry JSONL archived as `.beads/issues.jsonl.bak`). bd works.
- **Build:** `go build ./cmd/orch/` passes. Binary installed at `~/bin/orch`.
- **Git remote:** Local is 8 commits ahead, 1161 behind origin (origin has the spiral). Need to force-push to make revert stick.
- **Stale worktrees:** All removed. No more worktree isolation.

### orch-knowledge (main at `52b2658`)
- **Skills fixed:** All `orch phase` → `bd comment` across worker-base and feature-impl (15 files)
- **Stale root SKILL.md:** Deleted (was old worker-base deployed to wrong path)
- **13 commits ahead of origin.** Ready to push.
- **Pre-commit hook:** References `orch lint --skills` which doesn't exist at Jan 18. Needs fix.

### Deployed Skills (`~/.claude/skills/`)
- 20 skills compiled and deployed via `skillc deploy`
- Zero `orch phase` references remain
- **Known stale references** (from orch-go-1 audit):
  - `orch frontier` in meta-orchestrator, orchestrator
  - `orch friction`, `orch health`, `orch stability`, `orch reap` in diagnostic
  - `orch lint` in orch-knowledge pre-commit hook
  - **diagnostic skill is non-functional** — depends on 5 missing commands

---

## Open Issues

| ID | Priority | Title | Status |
|----|----------|-------|--------|
| orch-go-1 | P2 | Audit skills against Jan 18 baseline for stale CLI references | Investigation done, fixes not applied |
| orch-go-2 | P1 | Audit entropy spiral for recoverable features by functional area | Investigation done, recovery not started |

---

## Recovery Priority List (from orch-go-2 audit)

| Priority | Feature | Files | Complexity | Status |
|----------|---------|-------|------------|--------|
| **P1** | Models/probes Go code | `pkg/spawn/probes.go`, `kbcontext.go` | LOW | **DONE** ✅ |
| **P1** | Spawn Backends abstraction | `pkg/spawn/backends/` (6 files) | LOW | Not started |
| **P1** | Verification Spec generation | `pkg/spawn/verification_spec.go` | LOW | Not started |
| **P2** | Attention System | `pkg/attention/` (pre-`3b004bef`) | HIGH | Not started — accidentally deleted by bd sync |
| **P2** | Skill Inference | `pkg/daemon/skill_inference.go` | LOW | Not started |
| **P3** | Dashboard/Web UI | `web/` directory | MEDIUM | Not started |
| **P3** | Daemon Rate Limiting | `pkg/daemon/daemon.go` | LOW | Not started |

**Recovery method:** Cherry-pick from `entropy-spiral-feb2026` branch using `git show` (do NOT checkout the branch).

---

## Known Issues / Gotchas

1. **Spawned agents checkout wrong branch.** When told to analyze `entropy-spiral-feb2026`, agents checked it out instead of using `git show`. Two of three agents committed to the wrong branch. **Fix:** Spawn context must explicitly say "stay on master, use `git show branch:path` to read."

2. **bd pre-commit hook hangs.** The beads pre-commit hook (`bd hooks run pre-commit`) frequently times out, especially after JSONL changes. The probes agent fought this for several attempts. **Workaround:** `--no-verify` when bd is stuck, or `rm .beads/jsonl.lock` first.

3. **OpenCode API returns HTML.** The OpenCode session API (`/session/{id}/message`) sometimes returns the SPA HTML instead of JSON. May need Accept header or different endpoint.

4. **orch-knowledge pre-commit hook broken.** References `orch lint --skills` which doesn't exist. Must use `--no-verify` for now.

5. **git remote diverged.** Master is 8 ahead, 1161 behind origin. Force-push needed to sync.

---

## Immediate Next Steps

1. **Fix stale skill references (orch-go-1):** Remove/replace `orch frontier`, `orch friction`, etc. from affected skills. Either stub the commands or update skills to use alternatives.

2. **Cherry-pick spawn backends (orch-go-2 P1):** `git show entropy-spiral-feb2026:pkg/spawn/backends/` — single clean commit `40d09539`, tests pass on spiral branch. Verify against Jan 18 `spawn_cmd.go`.

3. **Cherry-pick verification_spec (orch-go-2 P1):** `pkg/spawn/verification_spec.go` + test file. Self-contained.

4. **Force-push master to origin** (requires Dylan's approval).

5. **Push orch-knowledge** (13 commits ahead, skill fixes).

---

## Postmortem Reference

Full postmortem: `.kb/investigations/2026-02-12-inv-entropy-spiral-postmortem.md`

**Key finding:** This was a repeat of the Dec 27-Jan 2 spiral. Same root causes, same recommended mitigations, none implemented. Critical mitigation: human-in-the-loop gates that halt autonomous operation after N commits or anomaly detection.

**Evidence branch:** `entropy-spiral-feb2026` at `c5bb7bfc`
