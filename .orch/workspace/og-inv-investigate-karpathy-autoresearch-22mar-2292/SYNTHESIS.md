# Session Synthesis

**Agent:** og-inv-investigate-karpathy-autoresearch-22mar-2292
**Issue:** orch-go-h40x4
**Duration:** 2026-03-22
**Outcome:** success

---

## Plain-Language Summary

Karpathy's autoresearch (48k stars in 16 days) is a 1,225-line repo where a single AI agent autonomously runs ML experiments while you sleep. It works by constraining the problem so tightly — one file to edit (train.py), one metric to optimize (val_bpb), fixed 5-minute runs, keep-or-discard decisions via git — that no orchestration framework is needed. The "architecture" is a 114-line markdown prompt (program.md) that tells Claude/Codex to loop forever, editing hyperparameters, running experiments, and advancing a git branch when results improve.

This contrasts sharply with orch-go, which handles open-ended, multi-agent, governance-heavy workflows. The key lessons: (1) constraint design can eliminate orchestration machinery for narrow problems, (2) narrative packaging ("research while you sleep") drives adoption more than technical sophistication, (3) orch-go's "no local agent state" / git-as-truth principles are validated, and (4) the skill system (SKILL.md) is autoresearch's program.md concept with more power.

## TLDR

autoresearch succeeds through constraint design (1 file, 1 metric, 5-min runs) not orchestration machinery. orch-go solves a fundamentally harder problem (open-ended multi-agent work), but should steal the constraint-first principle and narrative packaging.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-03-22-inv-investigate-karpathy-autoresearch-48k-stars.md` — Full investigation with 6 findings, synthesis, and recommendations

### Commits
- Investigation file with complete analysis

---

## Evidence (What Was Observed)

- autoresearch repo is 1,225 lines total: train.py (630), prepare.py (389), program.md (114), README.md (92)
- 60 git commits in the repo history — many are actual experiment results (hyperparameter tweaks)
- No agent framework code exists anywhere — the "orchestration" is entirely in program.md's 114-line prompt
- The experiment loop uses git branch position as state: keep = advance, discard = reset
- train.py contains serious ML engineering: MuonAdamW optimizer with torch.compile, Flash Attention 3, value embeddings, sliding window attention
- program.md explicitly instructs: "NEVER STOP" — agent runs until manually interrupted

### Tests Run
```bash
# Read all source files
wc -l train.py prepare.py program.md README.md  # 1225 total

# Analyzed git history
git log --oneline --all | wc -l  # 60 commits
git log --oneline --all | head -30  # recent: mostly experiment results and docs
git log --oneline --all | tail -20  # earliest: initial commit + rapid iteration
```

---

## Architectural Choices

No architectural choices — this was a pure investigation session.

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-03-22-inv-investigate-karpathy-autoresearch-48k-stars.md` — Complete analysis of autoresearch architecture, popularity drivers, and orch-go comparison

### Constraints Discovered
- autoresearch proves that tight constraint surfaces (1 file, 1 metric, fixed budget) eliminate the need for orchestration machinery
- Single-agent hill-climbing with git rollback is sufficient for scalar optimization problems
- The viral success formula was: trusted brand + "while you sleep" narrative + tangible results + extreme accessibility

---

## Verification Contract

See `VERIFICATION_SPEC.yaml` for verification criteria. Key outcomes:
- Investigation file complete with 6 findings, synthesis, structured uncertainty, and recommendations
- All source files read and analyzed (not speculated about)
- Comparison to orch-go grounded in actual architecture (CLAUDE.md, skill system)

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (investigation file with full analysis)
- [x] Investigation file has Phase: Complete
- [x] Ready for `orch complete orch-go-h40x4`

---

## Unexplored Questions

- Could orch-go benefit from a "tight loop" optimization mode (orch optimize) for tasks with clear scalar metrics?
- How would autoresearch perform with multi-agent parallelism (e.g., 4 agents exploring different directions)?
- Is the "research org code" framing useful for Dylan's blog/career narrative about agent orchestration?
- What's the ceiling on single-agent hill-climbing before you need structured decomposition?

---

## Friction

Friction: none

---

## Session Metadata

**Skill:** investigation
**Model:** claude-opus-4-5
**Workspace:** `.orch/workspace/og-inv-investigate-karpathy-autoresearch-22mar-2292/`
**Investigation:** `.kb/investigations/2026-03-22-inv-investigate-karpathy-autoresearch-48k-stars.md`
**Beads:** `bd show orch-go-h40x4`
