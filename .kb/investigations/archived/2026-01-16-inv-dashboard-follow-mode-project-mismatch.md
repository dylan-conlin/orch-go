<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Dashboard follow mode didn't show price-watch agents because directory name ("price-watch") didn't match beads ID prefix ("pw").

**Evidence:** Tests pass: `GetIncludedProjects("price-watch", configs)` returns `["price-watch", "pw"]` after adding mapping.

**Knowledge:** When directory name differs from beads ID prefix, must add explicit mapping in `DefaultMultiProjectConfigs()`.

**Next:** Close - fix implemented and tested.

**Promote to Decision:** recommend-no (tactical config fix, not architectural)

---

# Investigation: Dashboard Follow Mode Project Mismatch

**Question:** Why doesn't dashboard follow mode show price-watch agents?

**Started:** 2026-01-16
**Updated:** 2026-01-16
**Owner:** og-debug-dashboard-follow-mode-16jan-a731
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Project Name Mismatch

**Evidence:**
- Context API (`/api/context`) returns `project: "price-watch"` from directory basename
- Agents spawned for price-watch have beads IDs like `pw-xxxx`
- `extractProjectFromBeadsID("pw-xxxx")` returns `"pw"` (strips hash suffix)
- Dashboard filter compares `"pw"` against `["price-watch"]` → no match

**Source:**
- `cmd/orch/serve_context.go:94` - `resp.Project = filepath.Base(projectDir)`
- `cmd/orch/shared.go:130-142` - `extractProjectFromBeadsID` function
- `pkg/tmux/follower.go:379-390` - `GetIncludedProjects` function

**Significance:** The issue is a simple mapping problem between directory name and beads ID prefix, not an architectural flaw.

---

### Finding 2: Existing Multi-Project Config Pattern

**Evidence:**
- `DefaultMultiProjectConfigs()` already handles this pattern for orch-go
- orch-go includes multiple related projects (orch-cli, beads, kb-cli, etc.)
- The pattern is to add additional project names to `IncludeProjects` list

**Source:** `pkg/tmux/follower.go:359-384`

**Significance:** The fix should use the existing pattern rather than creating a new mechanism.

---

## Synthesis

**Key Insights:**

1. **Beads ID prefix is project-specific** - Each beads project can have its own prefix (e.g., "pw" for price-watch), independent of directory name.

2. **Dashboard filter uses project name from context API** - When following orchestrator, the dashboard filters agents by project name derived from directory.

3. **Simple mapping solves the problem** - Adding "pw" to the included projects for "price-watch" directory enables the filter to match.

**Answer to Investigation Question:**

Dashboard follow mode didn't show price-watch agents because the project name "price-watch" (from directory) didn't include "pw" (from beads ID prefix) in its included projects list. Fixed by adding a `MultiProjectConfig` entry for "price-watch" that includes "pw".

---

## Structured Uncertainty

**What's tested:**

- ✅ `GetIncludedProjects("price-watch", configs)` returns `["price-watch", "pw"]` (verified: unit test passes)
- ✅ Build compiles without errors (verified: `make build` succeeds)
- ✅ All existing tmux tests still pass (verified: `go test ./pkg/tmux/... -v`)

**What's untested:**

- ⚠️ End-to-end dashboard filtering with actual price-watch agents (not tested live)
- ⚠️ Other projects with similar naming mismatches (not surveyed)

**What would change this:**

- Finding would be wrong if beads ID prefix is dynamic (it's configured per project)
- Finding would be wrong if dashboard filtering uses different logic than GetIncludedProjects

---

## Implementation Recommendations

### Recommended Approach ⭐

**Add price-watch/pw mapping to DefaultMultiProjectConfigs** - Simple config addition following existing pattern.

**Why this approach:**
- Uses existing mechanism, no new code needed
- Pattern already proven with orch-go multi-project config
- Minimal change, minimal risk

**Trade-offs accepted:**
- Manual maintenance required if more projects have this issue
- Config is hardcoded in Go, not user-configurable

**Implementation sequence:**
1. ✅ Add `MultiProjectConfig{Project: "price-watch", IncludeProjects: []string{"pw"}}` to follower.go
2. ✅ Also add "price-watch" and "pw" to orch-go's IncludeProjects (for visibility from orch-go)
3. ✅ Add unit test to verify mapping

### Alternative Approaches Considered

**Option B: Auto-derive beads prefix from project config**
- **Pros:** No manual mapping needed
- **Cons:** Requires reading .beads config, adds complexity
- **When to use instead:** If many projects have this issue

**Rationale for recommendation:** Only one project currently affected (price-watch), simple mapping is sufficient.

---

## References

**Files Examined:**
- `pkg/tmux/follower.go:359-390` - MultiProjectConfig and GetIncludedProjects
- `cmd/orch/serve_context.go` - Context API handler
- `cmd/orch/serve_agents.go:1007-1027` - Project filter application
- `cmd/orch/shared.go:130-142` - extractProjectFromBeadsID function

**Commands Run:**
```bash
# Build and test
make build
go test ./pkg/tmux/... -v
go test -v -run TestGetIncludedProjects ./pkg/tmux/
```

---

## Investigation History

**2026-01-16 18:05:** Investigation started
- Initial question: Why doesn't dashboard follow mode show price-watch agents?
- Context: Root cause already identified in spawn task - project name mismatch

**2026-01-16 18:10:** Investigation completed
- Status: Complete
- Key outcome: Added "pw" to included projects for "price-watch" and orch-go configs
