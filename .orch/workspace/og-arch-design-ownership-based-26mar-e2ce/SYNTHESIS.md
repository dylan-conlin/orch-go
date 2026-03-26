# Session Synthesis

**Agent:** og-arch-design-ownership-based-26mar-e2ce
**Issue:** orch-go-y012i
**Outcome:** success

---

## Plain-Language Summary

The harness has been protecting the wrong thing. It polices git-add syntax with regex hooks while leaving the actual failure mode unchecked: agents close issues and leave uncommitted work behind. This design shifts the invariant from "clean worktree" to "every tracked dirty file must be owned by an open issue or belong to an allowed artifact class." The primary enforcement is a new completion gate (Gate 15: ownership_reconciliation) that runs at close time, where the orchestrator controls the enforcement point and agents can't bypass it. Supporting changes: untrack 7,294 historical workspace artifacts that dominate git status, fix contradictory skill text (`git add -A` in feature-impl vs NEVER in worker-base), and harden the build gate to verify committed state (not working tree) to catch split-commit defects.

## Verification Contract

See `VERIFICATION_SPEC.yaml` in this workspace. Key outcomes: design investigation produced with 6 forks navigated, probe created and merged into completion-verification and harness-engineering models, 5 implementation issues created for decomposed follow-up work.

---

## TLDR

Designed ownership-based harness to replace command-regex policing. The invariant shifts from "clean worktree" to "every tracked dirty file is owned." Close-time reconciliation gate (Gate 15) is the primary enforcement, supported by artifact class registry, skill text alignment, and build gate hardening. Five implementation issues created.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-03-26-design-ownership-based-harness-prevention.md` — Full design investigation with 6 forks, recommendations, migration path, decomposition
- `.kb/models/completion-verification/probes/2026-03-26-probe-ownership-based-harness-design-evaluation.md` — Probe evaluating ownership as verification gap closure
- `.orch/workspace/og-arch-design-ownership-based-26mar-e2ce/SYNTHESIS.md` — This file
- `.orch/workspace/og-arch-design-ownership-based-26mar-e2ce/BRIEF.md` — Dylan-facing brief
- `.orch/workspace/og-arch-design-ownership-based-26mar-e2ce/VERIFICATION_SPEC.yaml` — Verification spec

### Files Modified
- `.kb/models/completion-verification/model.md` — Added "Why This Fails" section 10 (no ownership reconciliation), Evolution Phase 12, probe entry
- `.kb/models/harness-engineering/model.md` — Added bypass-resistance finding (binary vs continuous invariants)

### Commits
- (pending — see completion protocol)

---

## Evidence (What Was Observed)

- `git status` shows 7,296 dirty entries; 7,294 are deleted `.orch/workspace/` historical artifacts, only 2 are actual source changes
- Build is broken on committed state: `pkg/daemon/ooda.go:237` calls `RouteModel()` which only exists in dirty tree (split-commit from closed issue orch-go-r7avo)
- feature-impl `validation.md` contains `git add -A` 4 times, contradicting worker-base which says "NEVER use `git add -A`"
- Accretion gates had 100% bypass rate over 2-week measurement (decision 2026-03-17)
- Governance file protection hook (Edit/Write block) is effective — fundamentally different enforcement type than Bash regex hooks
- AGENT_MANIFEST.json already stores GitBaseline — perfect anchor for ownership-based scoping

---

## Architectural Choices

### Close-time enforcement over spawn-time restriction
- **What I chose:** Gate 15 at issue close time (V2+ level)
- **What I rejected:** Spawn-time scope manifests that restrict which files agents can modify
- **Why:** Agents discover scope during work — restricting at spawn time would have high false-positive rate and repeat the accretion-gate bypass pattern. Close-time enforcement uses the orchestrator-controlled `orch complete` path, which agents cannot bypass.
- **Risk accepted:** Agent may leave dirty files that are only caught at close time, not prevented during work

### Binary invariant over continuous invariant
- **What I chose:** Ownership (binary: owned/unowned) over cleanliness (continuous: how many dirty files)
- **What I rejected:** Line-count-style thresholds for dirty-file tolerance
- **Why:** Accretion gates proved that continuous invariants are bypassed 100% of the time. Binary invariants leave no gradient to argue about.
- **Risk accepted:** Some legitimate "unowned" dirty files may need an explicit allowed-residue classification

### Advisory layers over blocking layers
- **What I chose:** Spawn-time scope context and commit-time warnings as advisory only
- **What I rejected:** Blocking hooks at commit time or spawn time
- **Why:** Harness-engineering model: blocking adds friction agents route around instantly. The only blocking layer is close-time (orchestrator-controlled).
- **Risk accepted:** Advisory signals may be ignored — but the close-time gate catches what they miss

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-03-26-design-ownership-based-harness-prevention.md` — Design investigation
- `.kb/models/completion-verification/probes/2026-03-26-probe-ownership-based-harness-design-evaluation.md` — Probe

### Decisions Made
- Primary invariant is ownership, not cleanliness (rationale: 99.7% of dirt is harmless artifacts)
- Close-time reconciliation is the only effective enforcement layer (rationale: orchestrator-controlled, binary invariant, no agent bypass)
- Skill text alignment is prerequisite — contradictory `git add -A` is Class 5 defect

### Constraints Discovered
- Build gate runs against working tree, not committed state — cannot detect split-commit defects
- Governance-protected files prevent workers from fixing hook false-positives in `~/.orch/hooks/`
- Skill text precedence unclear — worker-base should be canonical but feature-impl overrides in practice

---

## Next (What Should Happen)

**Recommendation:** close

### Implementation Issues Created (5 follow-ups)

| Phase | Issue | Description | Priority |
|-------|-------|-------------|----------|
| 1 | orch-go-0puaq | Untrack historical .orch/workspace/ and experiment artifacts | Immediate |
| 2 | orch-go-mq0si | Remove `git add -A` from feature-impl and systematic-debugging | Immediate |
| 3 | orch-go-1rb9d | Implement Gate 15: ownership_reconciliation | Implementation sprint |
| 4 | orch-go-o5r8j | Harden build gate (stash before go build) | Implementation sprint |
| 5 | orch-go-rppz0 | Hook invocation logging and self-test | Future |

---

## Unexplored Questions

- How exactly should the file-to-issue mapping work at close time? (Recommended: SYNTHESIS.md Delta section, already partially implemented)
- Should knowledge-backlog files (.kb/) have a time limit on being "allowed residue" before requiring batch commit?
- Can the ownership gate reuse `GetGitDiffFiles()` from `pkg/verify/git_diff.go` or does it need a separate implementation?

---

## Friction

Friction: none

---

## Session Metadata

**Skill:** architect
**Model:** claude-opus-4-5
**Workspace:** `.orch/workspace/og-arch-design-ownership-based-26mar-e2ce/`
**Investigation:** `.kb/investigations/2026-03-26-design-ownership-based-harness-prevention.md`
**Beads:** `bd show orch-go-y012i`
