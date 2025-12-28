# Session Synthesis

**Agent:** og-work-solve-stale-binary-28dec
**Issue:** orch-go-3m23
**Duration:** 2025-12-28 ~13:00 → ~14:30
**Outcome:** success

---

## TLDR

Investigated the stale binary problem for human-used Go CLIs and produced a decision recommending symlinks from `~/bin/` to build output. Created 3 follow-up issues for implementation phases.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2025-12-28-inv-solve-stale-binary-problem-human.md` - Full investigation with findings and recommendations
- `.kb/decisions/2025-12-28-stale-binary-solution.md` - Decision artifact with chosen approach and trade-offs
- `.orch/workspace/og-work-solve-stale-binary-28dec/SYNTHESIS.md` - This file

### Files Modified
- None (design session - artifacts only)

### Commits
- None yet - artifacts need to be committed

### Issues Created
- `orch-go-4bi3` - Implement symlink-based install in orch-go Makefile (Phase 1)
- `orch-go-niyj` - Cleanup stale binaries and fix PATH order (Phase 2)
- `orch-go-jj4i` - Propagate symlink install pattern to other Go CLIs (Phase 3)

---

## Evidence (What Was Observed)

- **Dual locations:** Binaries exist in both `~/bin/` and `~/go/bin/` with different timestamps
  - `~/bin/orch` - Dec 28 12:56 (newer)
  - `~/go/bin/orch` - Dec 23 21:51 (5 days older)
- **Python orch shadow:** `/opt/homebrew/bin/orch` is Python orch (v0.2.0), appears before `~/bin` in PATH
- **PATH order:** Dylan's `.zshrc` adds `~/bin` at line 792 (near end), after `/opt/homebrew/bin`
- **Daemon uses correct PATH:** launchd plist explicitly sets `/Users/dylanconlin/bin` first
- **Inconsistent install patterns:** orch-go uses `make install` → copy, beads uses `go install`

### Commands Run
```bash
# Verify dual locations
ls -la ~/bin/ ~/go/bin/ | grep -E "(orch|kb|bd)"

# Check Python orch
cat /opt/homebrew/bin/orch  # Shows Python shim

# Check PATH order
echo $PATH | tr ":" "\n" | head -20

# Compare versions
~/bin/orch version && ~/go/bin/orch version
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2025-12-28-inv-solve-stale-binary-problem-human.md` - Full root cause analysis
- `.kb/decisions/2025-12-28-stale-binary-solution.md` - Chosen solution with trade-offs

### Decisions Made
- **Symlinks over copies**: Use `ln -sf build/binary ~/bin/binary` so `make build` automatically updates human CLI
- **~/bin as canonical location**: Already used by orch-go, not tied to GOPATH
- **Three-phase implementation**: 1) orch-go, 2) cleanup, 3) propagate to other CLIs

### Constraints Discovered
- **Glass doesn't have this problem** - because only Claude uses it (confirmed initial task context)
- **Daemon is fine** - explicit PATH in launchd plist works correctly
- **Problem is human terminals** - PATH order and duplicate binaries cause confusion

### Externalized via `kn`
- Will recommend orchestrator run:
  ```bash
  kn decide "Symlinks from ~/bin to build output for human-used Go CLIs" --reason "Make build automatically updates human CLI, no separate install step needed"
  ```

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (investigation + decision + follow-up issues)
- [x] Investigation file has `**Phase:** Complete`
- [x] Decision artifact created with clear recommendation
- [x] Follow-up issues created for implementation phases
- [x] Ready for `orch complete orch-go-3m23`

### Follow-up Work Created
The design session produced 3 actionable issues:
1. **orch-go-4bi3** - Implement symlink pattern in orch-go (validate approach)
2. **orch-go-niyj** - Cleanup stale binaries + PATH fix (environment prep)
3. **orch-go-jj4i** - Propagate to kb-cli, beads, kn, skillc (cross-project)

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- How did Python orch get installed? (pip install? Homebrew? Should be uninstalled cleanly)
- Are there other CLIs in Dylan's ecosystem with similar patterns?
- Should there be a `doctor` command that checks for stale binaries across all CLIs?

**Areas worth exploring further:**
- Automated staleness detection at shell startup (SessionStart hook for terminals?)
- Version consistency checking across the ecosystem

**What remains unclear:**
- Whether the symlink pattern will cause issues when building from a different machine
- Impact on Dylan's muscle memory (may need to run `make build` vs `make install`)

---

## Session Metadata

**Skill:** design-session
**Model:** claude-opus-4
**Workspace:** `.orch/workspace/og-work-solve-stale-binary-28dec/`
**Investigation:** `.kb/investigations/2025-12-28-inv-solve-stale-binary-problem-human.md`
**Decision:** `.kb/decisions/2025-12-28-stale-binary-solution.md`
**Beads:** `bd show orch-go-3m23`
