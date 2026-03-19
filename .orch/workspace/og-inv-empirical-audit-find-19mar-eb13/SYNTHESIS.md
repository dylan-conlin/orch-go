# Session Synthesis

**Agent:** og-inv-empirical-audit-find-19mar-eb13
**Issue:** (ad-hoc, no beads tracking)
**Outcome:** success

---

## Plain-Language Summary

Audited the orch-go codebase for code that ended up in the wrong package because governance hooks blocked agents from writing to protected files. Found 2 confirmed instances (115 lines): a concurrency gate and a governance path checker that both belong in `pkg/spawn/gates/` but live in `pkg/orch/`. Found 1 probable instance (182 lines): an artifact validator in `pkg/completion/` that should be in `pkg/verify/`. The biggest finding isn't about displaced code — it's a documentation mismatch. The worker-base skill tells agents that ALL of `pkg/verify/*` is protected, but the actual hook only blocks 2 specific files (`precommit.go` and `accretion.go`). This phantom protection zone prevents workers from writing to 67 files they could safely modify.

## Verification Contract

See `VERIFICATION_SPEC.yaml` — key outcomes are the investigation file and displacement catalog.

---

## TLDR

2 confirmed + 1 probable code displacement instances (~297 lines total) from governance hook enforcement. But the bigger discovery is a documentation-enforcement mismatch: worker-base skill claims all of `pkg/verify/*` is protected when only 2 files actually are. This phantom zone is a stronger displacement force than the actual hook.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/simple/2026-03-19-empirical-audit-displaced-code-governance-hooks.md` — Full investigation with 9 findings, displacement catalog, and line counts
- `.orch/workspace/og-inv-empirical-audit-find-19mar-eb13/SYNTHESIS.md` — This file
- `.orch/workspace/og-inv-empirical-audit-find-19mar-eb13/VERIFICATION_SPEC.yaml` — Verification spec

### Files Modified
- None (investigation only — no code changes)

---

## Evidence (What Was Observed)

- `pkg/orch/spawn_preflight.go` lines 10-12 + 30-41: ConcurrencyCheck type and gate logic added by scs-sp-8dm (commit `61e12c298`) because `pkg/spawn/gates/concurrency.go` is protected. The commit message explicitly says "reinstates concurrency gate" and the file does not exist in gates/.
- `pkg/orch/governance.go` (87 lines): Spawn-time gate checking task against protected paths, created by worker orch-go-enwt2 (commit `7102b4722`). Follows exact gate pattern (check → result struct → log decision) used by `gates/{triage,hotspot,agreements,question}.go`.
- `pkg/completion/artifact.go` (182 lines): Completion validation gate created by worker orch-go-lqiel (commit `f45b40a12`). Could have been in `pkg/verify/` since the hook doesn't protect that path, but worker-base skill says it's protected.
- Worker-base skill line 56: `"Files in pkg/spawn/gates/* and pkg/verify/* cannot be modified by workers"` — overstates actual protection.
- Governance hook lines 53-55: Only 2 patterns for pkg/verify/: `precommit\.go$` and `accretion\.go$`.
- 5 sessions produced correct escalation (VERIFICATION_SPEC patches for orchestrator) without displacement: og-debug-populate-hotspot-bypass-11mar-9327, og-debug-stop-hook-escape-10mar-0966, og-debug-claude-print-output-11mar-be96, og-feat-accretion-gates-block-17mar-9f1a, og-debug-fix-concurrency-cap-27feb-8772.
- Prior agent og-debug-resolve-self-review-13mar-9a37 independently discovered the documentation mismatch (SYNTHESIS.md line 81).

### Tests Performed
```bash
# Verified governance hook protected patterns
cat ~/.orch/hooks/gate-governance-file-protection.py | grep 'pkg/verify'
# Result: only precommit.go and accretion.go patterns

# Verified worker-base skill claim
grep 'pkg/verify' skills/src/shared/worker-base/SKILL.md
# Result: "Files in pkg/spawn/gates/* and pkg/verify/* cannot be modified"

# Verified concurrency gate does not exist in canonical location
ls pkg/spawn/gates/concurrency* 2>/dev/null
# Result: no files found

# Verified governance.go was created by worker agent
git log --oneline --follow -- pkg/orch/governance.go
# Result: 7102b4722 feat: detect governance-protected files at spawn routing time (orch-go-enwt2)

# Verified artifact.go was created by worker agent
git log --oneline --follow -- pkg/completion/artifact.go
# Result: f45b40a12 feat: add completion artifact validator (orch-go-lqiel)

# Counted total files in pkg/verify/ that are NOT protected
ls pkg/verify/*.go | wc -l
# Result: 69 files total, only 2 protected
```

---

## Architectural Choices

No architectural choices — task was investigation only.

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/simple/2026-03-19-empirical-audit-displaced-code-governance-hooks.md` — Full displacement catalog

### Constraints Discovered
- CONSTRAINT: Worker-base skill overstates governance protection zone (`pkg/verify/*` vs actual 2-file protection). Fixing requires editing governance-protected skill file.
- CONSTRAINT: Migrating displaced code (concurrency gate, governance check) to canonical locations requires orchestrator direct session since target directories are protected.

---

## Next (What Should Happen)

**Recommendation:** spawn-follow-up (architect session)

### If Spawn Follow-up

**Three actions, in priority order:**

1. **Fix documentation mismatch** (highest impact, low effort): Change worker-base skill line 56 from `pkg/verify/*` to list the 2 actually-protected files (`pkg/verify/precommit.go`, `pkg/verify/accretion.go`). Requires orchestrator session since `skills/src/shared/worker-base/` is governance-protected.

2. **Migrate displaced code** (moderate impact, moderate effort):
   - Move concurrency gate from `pkg/orch/spawn_preflight.go:10-41` to `pkg/spawn/gates/concurrency.go`
   - Move governance check from `pkg/orch/governance.go` to `pkg/spawn/gates/governance.go`
   - Consider moving `pkg/completion/artifact.go` to `pkg/verify/completion_artifact.go`

3. **Add redirect hint to deny message** (moderate impact, low effort): Modify governance hook to say "Put code in a non-protected location and document intended destination in SYNTHESIS.md" (per parallel investigation recommendation).

---

## Unexplored Questions

- **How many other workers avoided pkg/verify/ due to the documentation mismatch?** The 1 confirmed case (artifact.go) may be the tip of the iceberg. A git log scan of `pkg/verify/` commits vs. worker agent commits could reveal whether workers systematically avoid it.
- **Should the governance hook protect ALL of pkg/verify/?** The current 2-file protection may be intentionally narrow. If so, the documentation should shrink. If it should be broader, the hook should grow.

---

## Friction

- `orch kb create investigation --orphan` flag doesn't exist — had to create investigation file manually. Minor friction.
- No other friction.

---

## Session Metadata

**Skill:** investigation
**Model:** claude-opus-4-5-20251101
**Workspace:** `.orch/workspace/og-inv-empirical-audit-find-19mar-eb13/`
**Investigation:** `.kb/investigations/simple/2026-03-19-empirical-audit-displaced-code-governance-hooks.md`
