# Session Synthesis

**Agent:** og-arch-design-question-should-23dec
**Issue:** orch-go-lb61
**Duration:** 2025-12-23 10:30 → 2025-12-23 11:20
**Outcome:** success

---

## TLDR

Analyzed whether `orch servers status` should check port listening vs tmux session existence. Recommendation: Keep tmux abstraction (fast, correct for dev servers), but separate `orch serve` API into dedicated infrastructure command since it's persistent monitoring infrastructure, not ephemeral project server.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2025-12-23-inv-design-question-should-orch-servers.md` - Design investigation with analysis of 3 approaches (keep current, add to tmuxinator, check port listening) and recommendation to separate infrastructure

### Files Modified
- None (design phase - no implementation)

### Commits
- `f02949b` - architect: servers status abstraction - recommend separating infrastructure from project servers

---

## Evidence (What Was Observed)

- **Current implementation uses tmux session check** - Verified in `cmd/orch/servers.go:172-186`, no port listening checks performed
- **`orch serve` runs outside tmuxinator** - Confirmed via `lsof -i :3348` showing separate orch process (PID 23280), not in workers-orch-go tmux session
- **Tmuxinator config has comment, not command** - `~/.tmuxinator/workers-orch-go.yml` shows `# api server on port 3348` (comment only), actual command is `bun run dev --port 5188` (web server)
- **Two distinct operational patterns** - Dev servers restart with project sessions (ephemeral), API server runs continuously (persistent infrastructure)

### Tests Run
```bash
# Verify tmux sessions
tmux list-sessions | grep workers
# Result: workers-orch-go exists

# Check port 3348 (orch serve API)
lsof -i :3348
# Result: orch process (PID 23280) listening, NOT in tmux

# Check port 5188 (web dev server)
lsof -i :5188
# Result: node process listening (tmuxinator-managed)
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2025-12-23-inv-design-question-should-orch-servers.md` - Design question analysis with recommendation

### Decisions Made
- **Decision:** Tmux session existence is the correct abstraction for `orch servers status` - matches dev server lifecycle, fast, semantically accurate for project-scoped development infrastructure
- **Decision:** `orch serve` should be managed separately - it's orchestrator infrastructure (global, persistent), not project infrastructure (local, ephemeral)

### Constraints Discovered
- **Constraint:** Port listening checks scale linearly with project count - acceptable for single-purpose infrastructure check (`orch api status`), prohibitive for batch status check across projects
- **Constraint:** Tmuxinator manages per-project dev servers only - by design, not suitable for cross-project infrastructure like monitoring API

### Externalized via `kn`
- Not applicable (design phase - recommendations documented in investigation, no runtime learnings to externalize)

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (investigation file created, analysis documented)
- [x] Investigation file has `**Phase:** Complete`
- [x] Ready for `orch complete orch-go-lb61`

**Recommended implementation sequence (for future feature-impl spawn):**
1. Add `orch api status` command (check port 3348 listening via net.Dial)
2. Remove port 3348 allocation for orch-go project (`orch port release orch-go api`)
3. Update tmuxinator template to remove API server comment
4. Document distinction in CLAUDE.md and README

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**

- **Should `orch api` be part of `orch daemon`?** - The API serves monitoring UI. Conceptually related to daemon operations. Could be `orch daemon api status` instead of top-level `orch api status`. Worth discussing naming/grouping.

- **Should `orch serve` write a PID file?** - Port checking can give false positives (different process on that port). PID file would be more reliable for lifecycle management.

- **Lifecycle management for `orch serve`** - Should there be `orch api start` and `orch api stop`? Or keep manual (`orch serve &` in background)?

**Areas worth exploring further:**

- **Performance benchmarking** - Measure port listening check overhead for 10, 50, 100 projects to quantify the cost if we ever need it
- **User workflow observation** - Do users start/stop `orch serve` per session, or run it continuously? Validates architectural assumption.

**What remains unclear:**

- **User expectations** - Haven't validated whether users expect `orch servers` to show `orch serve` status. Recommendation assumes they primarily care about dev servers.
- **Future per-project API servers** - If architecture evolves to per-project monitoring APIs, recommendation would change. Unlikely but worth noting.

---

## Session Metadata

**Skill:** architect
**Model:** anthropic/claude-opus-4-5-20251101
**Workspace:** `.orch/workspace/og-arch-design-question-should-23dec/`
**Investigation:** `.kb/investigations/2025-12-23-inv-design-question-should-orch-servers.md`
**Beads:** `bd show orch-go-lb61`
