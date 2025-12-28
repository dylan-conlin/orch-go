# Session Synthesis

**Agent:** og-work-scope-ecosystem-context-28dec
**Issue:** orch-go-tr0b
**Duration:** 2025-12-28 ~11:00 -> ~11:50
**Outcome:** success

---

## TLDR

Investigated ecosystem context injection scope for spawned agents. Found that ECOSYSTEM.md already exists at ~/.orch/ with comprehensive documentation, but is not being injected into spawn contexts. Recommended tiered approach: always inject core tooling (bd, kb, orch, kn), inject project registry only for ecosystem repos, and reference (don't embed) full ECOSYSTEM.md.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2025-12-28-scope-ecosystem-context-injection-spawned.md` - Full investigation with findings and recommendations
- `.orch/workspace/og-work-scope-ecosystem-context-28dec/SYNTHESIS.md` - This file

### Files Modified
- None (investigation only, no implementation)

### Commits
- (to be committed)

---

## Evidence (What Was Observed)

- `~/.orch/ECOSYSTEM.md` exists with ~150 lines of comprehensive ecosystem documentation (repos, CLIs, purposes)
- Orchestrator skill at `~/.claude/skills/meta/orchestrator/SKILL.md` references ECOSYSTEM.md but workers don't receive it
- `pkg/spawn/kbcontext.go:15-22` already defines `OrchEcosystemRepos` with 6 repos (missing glass, beads-ui-svelte, skillc, agentlog)
- `pkg/spawn/context.go` has "CONTEXT AVAILABLE" section but only mentions CLAUDE.md files, not ecosystem tooling
- SPAWN_CONTEXT.md template has no {{.EcosystemContext}} injection point

### Key Observations
- Dylan's ecosystem has 10+ related projects with CLIs (orch, bd, kb, kn, glass, skillc, etc.)
- Spawned agents don't know these exist, leading to GitHub searches instead of local commands
- The prior investigation on "pre-spawn context gathering" established 5-minute rule for orchestrator context work

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2025-12-28-scope-ecosystem-context-injection-spawned.md` - Scoping investigation

### Decisions Made
- **Tiered injection:** Core tooling always (~10 lines), project registry for ecosystem repos only (~25 lines)
- **Reference not embed:** Point to ECOSYSTEM.md rather than duplicating its 150+ lines
- **Static over config:** Keep ecosystem repos list in code (OrchEcosystemRepos) rather than config file

### Constraints Discovered
- Token budget constraint: ~80k chars reserved for kb context (per MaxKBContextChars in kbcontext.go)
- Must balance completeness vs context bloat

### Externalized via `kn`
- Not run (kn command not available in PATH in this environment)

---

## Next (What Should Happen)

**Recommendation:** spawn-follow-up

### If Spawn Follow-up
**Issue:** Implement ecosystem context injection in spawn
**Skill:** feature-impl
**Context:**
```
Investigation complete (orch-go-tr0b). Implement tiered ecosystem context injection:
1. Expand OrchEcosystemRepos in pkg/spawn/kbcontext.go (add glass, skillc, agentlog, beads-ui-svelte)
2. Add GenerateEcosystemContext() function in new pkg/spawn/ecosystem.go
3. Add EcosystemContext field to spawn.Config
4. Update SpawnContextTemplate in pkg/spawn/context.go with {{.EcosystemContext}} section
5. Wire up in GenerateContext() with IsEcosystemRepo check
See investigation: .kb/investigations/2025-12-28-scope-ecosystem-context-injection-spawned.md
```

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Should non-ecosystem repos have opt-in `--ecosystem` flag for forcing context injection?
- Should ecosystem context be cached/precomputed for performance?
- How does this interact with `--workdir` cross-project spawns (spawning in different repo)?

**Areas worth exploring further:**
- Automatic discovery of ecosystem repos from kb projects list instead of static map
- Token impact measurement of ecosystem context on different model limits

**What remains unclear:**
- Exact token cost of 35-line ecosystem context (estimated ~150-200 tokens)
- Whether some spawns in ecosystem repos shouldn't get context (e.g., simple typo fixes)

---

## Session Metadata

**Skill:** design-session
**Model:** opus
**Workspace:** `.orch/workspace/og-work-scope-ecosystem-context-28dec/`
**Investigation:** `.kb/investigations/2025-12-28-scope-ecosystem-context-injection-spawned.md`
**Beads:** `bd show orch-go-tr0b`
