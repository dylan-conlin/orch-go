## Summary (D.E.K.N.)

**Delta:** serve_agents_status.go hotspot acceleration is a false positive — 83.5% of additions (264/316) are birth churn from extracting a 1713-line monolith.

**Evidence:** `git log --numstat` shows 264 lines from file creation (a7b6b38df, Feb 18 extraction of serve_agents.go) and one bug fix (+52/-19, feed9593c). File is 297 lines — well under any threshold.

**Knowledge:** The hotspot detector flags files created by extraction the same as organic growth. Files born from extraction are the opposite of a hotspot — they are the *fix* for hotspots.

**Next:** Close as false positive. No action needed.

**Authority:** implementation - Tactical classification of false positive, no architectural impact.

---

# Investigation: Hotspot Acceleration — serve_agents_status.go

**Question:** Is the +316 lines/30d acceleration in serve_agents_status.go a genuine hotspot risk or a false positive?

**Started:** 2026-03-17
**Updated:** 2026-03-17
**Owner:** orch-go-iytbo
**Phase:** Complete
**Next Step:** None
**Status:** Complete

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| N/A - novel investigation (same pattern as prior birth-churn FPs) | - | - | - |

---

## Findings

### Finding 1: File was created via extraction from 1713-line monolith

**Evidence:** Commit a7b6b38df (2026-02-18) "feat: extract serve agents files (orch-go-1064)" deleted `serve_agents.go` (1713 lines) and created 7 smaller files:
- `serve_agents_handlers.go` (732 lines)
- `serve_agents_discovery.go` (324 lines)
- `serve_agents_status.go` (264 lines) ← this file
- `serve_agents_activity.go` (224 lines)
- `serve_agents_gap.go` (120 lines)
- `serve_agents_types.go` (59 lines)
- `serve_agents_cache_handler.go` (36 lines)

**Source:** `git show a7b6b38df --numstat | grep serve_agents`

**Significance:** The file's entire initial content (264/316 = 83.5% of total additions) is birth churn from an extraction — the exact workflow the hotspot system recommends. This is not organic growth.

---

### Finding 2: Only one post-birth commit, a small bug fix

**Evidence:** Commit feed9593c (2026-03-04) "fix: single canonical status derivation — eliminate Class 5 defects (orch-go-p6wg)" added 52 lines, removed 19 lines (net +33). This is the only post-birth modification.

**Source:** `git log --format="%h %ad %s" --date=short --numstat -- cmd/orch/serve_agents_status.go`

**Significance:** Post-birth growth is only 33 net lines over 30 days. At this rate the file would need years to approach the 1500-line extraction threshold.

---

### Finding 3: Current file size is well within safe bounds

**Evidence:** File is 297 lines containing 4 well-scoped functions:
- `getProjectAPIPort()` (18 lines) — port lookup
- `checkWorkspaceSynthesis()` (12 lines) — synthesis file check
- `determineAgentStatus()` (57 lines) — canonical status mapping
- `extractLastActivityFromMessages()` (73 lines) — message parsing
- `emitCompletionBacklogMetrics()` (68 lines) — metrics emission
Plus package-level vars and imports.

**Source:** `wc -l cmd/orch/serve_agents_status.go` → 297 lines

**Significance:** 297 lines is 20% of the 1500-line threshold. No extraction concern.

---

## Synthesis

**Key Insights:**

1. **Birth churn dominates** — 83.5% of detected additions are from file creation during extraction, not organic growth.

2. **Extraction artifacts are anti-hotspots** — This file exists *because* a hotspot was fixed. Flagging it as a new hotspot is a detector deficiency.

3. **Post-birth trajectory is healthy** — Only +33 net lines in one commit over 30 days. Well-scoped functions, clear single responsibility.

**Answer to Investigation Question:**

This is a false positive. The file was created by extracting a genuine 1713-line hotspot into 7 well-scoped modules. The detected "+316 lines/30d" is 83.5% birth churn. Post-birth growth is minimal (+33 lines, one bug fix). No action needed.

---

## Structured Uncertainty

**What's tested:**

- ✅ File creation was via extraction (verified: git log --follow --diff-filter=A shows a7b6b38df)
- ✅ 264/316 additions are birth churn (verified: git log --numstat shows 264+0 at creation)
- ✅ Only one post-birth commit exists (verified: git log --since="30 days ago" shows 2 commits total)

**What's untested:**

- ⚠️ Whether the hotspot detector could be modified to exclude birth churn (not in scope)

**What would change this:**

- Finding would be wrong if there were additional untracked commits modifying this file (unlikely given git history is authoritative)

---

## References

**Files Examined:**
- `cmd/orch/serve_agents_status.go` — The flagged file, 297 lines
- `cmd/orch/serve_agents.go` — The original 1713-line monolith (deleted in extraction)

**Commands Run:**
```bash
# File creation commit
git log --oneline --follow --diff-filter=A -- cmd/orch/serve_agents_status.go

# Commit history with line counts
git log --format="%h %ad %s" --date=short --follow --numstat -- cmd/orch/serve_agents_status.go

# Extraction commit details
git show a7b6b38df --numstat | grep serve_agents
```
