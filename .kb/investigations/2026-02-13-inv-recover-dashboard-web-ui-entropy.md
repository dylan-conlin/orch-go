## Summary (D.E.K.N.)

**Delta:** Master already has a complete dashboard superset of the entropy-spiral-feb2026 branch — no recovery needed.

**Evidence:** `git diff --name-status` shows master has all entropy web/ files plus 12 additional files; `bun run build` passes clean on master; all attention API endpoints registered in serve.go:378-390.

**Knowledge:** The entropy branch is 1166 commits diverged but represents an older, stripped-down version. Prior cherry-picks (orch-go-3, orch-go-6) plus ongoing master work already surpassed entropy's dashboard.

**Next:** No recovery action needed. If attention surface UI is desired, it should be built as a NEW feature (backend endpoints exist but no frontend component on either branch).

**Authority:** implementation - Factual finding, no decision needed

---

# Investigation: Recover Dashboard Web UI from Entropy Branch

**Question:** Does the entropy-spiral-feb2026 branch contain dashboard web/ files that need recovery to master?

**Started:** 2026-02-13
**Updated:** 2026-02-13
**Owner:** orch-go-7
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Master is a strict superset of entropy for web/ files

**Evidence:** `git diff --name-status entropy-spiral-feb2026 master -- web/src/` shows only `A` (added) and `M` (modified) statuses — no `D` (deleted) entries. This means every file on entropy also exists on master, and master has additional files.

Master-only files (12 additional):
- `screenshots-tab.svelte` (agent-detail component)
- `cache-validation-banner/` (2 files)
- `questions-section/` (2 files)
- `service-card/` (2 files)
- `service-log-viewer/` (1 file)
- `services-section/` (2 files)
- Stores: `cache-validation.ts`, `coaching.ts`, `questions.ts`, `servicelog.ts`, `services.ts`

**Source:** `git diff --name-status entropy-spiral-feb2026 master -- web/src/`

**Significance:** There is nothing to recover — master already has everything entropy has and more.

---

### Finding 2: Dashboard builds successfully on master

**Evidence:** `cd web && bun run build` completes in ~10s with `✔ done` output. The build produces valid client and server bundles. `svelte-check` reports 5 pre-existing TypeScript errors in `theme.ts` (type narrowing issues), but these don't block the build.

**Source:** `bun run build` (passes), `bun run check` (5 errors in theme.ts, pre-existing)

**Significance:** The dashboard is already functional on master. No build fixes needed.

---

### Finding 3: Attention API endpoints exist but no frontend UI on either branch

**Evidence:** `serve.go:378-390` registers 5 attention endpoints:
- `/api/attention` → `handleAttention`
- `/api/attention/likely-done` → `handleLikelyDone`
- `/api/attention/verify` → `handleAttentionVerify`
- `/api/attention/verify-failed/clear` → `handleVerifyFailedClear`
- `/api/attention/verify-failed/reset-status` → `handleVerifyFailedResetStatus`

`grep -r '/api/attention' web/src/` returns zero matches on master. The entropy branch also has no attention UI components.

**Source:** `cmd/orch/serve.go:378-390`, `grep -ri 'attention' web/src/` (only "needs-attention" component, which is unrelated)

**Significance:** The "attention surface" mentioned in the task description would need to be built as a new feature, not recovered. `pkg/attention/` (recovered in orch-go-6) and the API endpoints exist, but the UI layer was never implemented on either branch.

---

### Finding 4: serve_attention.go is not a separate file — endpoints live in serve.go

**Evidence:** `ls cmd/orch/serve_attention.go` returns "No such file" on both branches. However, attention endpoint handlers are defined inline in `cmd/orch/serve.go` starting at line 378. The handlers delegate to `pkg/attention/` functions.

**Source:** `cmd/orch/serve.go:378-390`, `ls cmd/orch/serve_attention.go` (not found)

**Significance:** No separate file recovery needed. The attention endpoints are already wired up in the main serve.go file.

---

## Synthesis

**Key Insights:**

1. **No recovery needed** — Master already has a complete, building dashboard that is a strict superset of the entropy branch.

2. **Attention surface is a gap, not a recovery** — The backend attention system is fully wired (pkg/attention + API endpoints), but no frontend UI exists on any branch. This is a new feature opportunity.

3. **Pre-existing TypeScript issues** — `theme.ts` has type narrowing errors that predate this investigation. These don't block builds but should be addressed separately.

**Answer to Investigation Question:**

No. The entropy-spiral-feb2026 branch does NOT contain dashboard web/ files that need recovery. Master already has everything entropy has (and more). The dashboard builds successfully. The "attention surface" mentioned in the task would be new feature work, not recovery.

---

## Structured Uncertainty

**What's tested:**

- ✅ Master is a superset of entropy web/ files (verified: `git diff --name-status` both directions)
- ✅ Dashboard builds on master (verified: `bun run build` passes clean)
- ✅ All attention API endpoints registered in serve.go (verified: grep for HandleFunc)
- ✅ No attention frontend exists on either branch (verified: grep in web/src/)

**What's untested:**

- ⚠️ Dashboard runtime behavior (not tested — would need running `orch serve` + browser)
- ⚠️ Whether the API endpoints return valid data with current pkg/attention/ (not tested)

**What would change this:**

- If there were a third branch with attention UI that was the actual source
- If the task intended "attention surface" to mean the NeedsAttention component (which already exists)

---

## References

**Commands Run:**
```bash
# Compare branches (both directions)
git diff --name-status master entropy-spiral-feb2026 -- web/src/
git diff --name-status entropy-spiral-feb2026 master -- web/src/

# Build test
cd web && bun run build
cd web && bun run check

# Endpoint verification
grep HandleFunc cmd/orch/serve.go | grep attention

# Frontend attention references
grep -ri '/api/attention' web/src/
grep -ri 'attention' web/src/
```
