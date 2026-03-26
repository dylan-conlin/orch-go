# Probe: Ownership-Based Harness Design — Does File Ownership Close the Verification Gap?

**Model:** Completion Verification Architecture
**Secondary Model:** Harness Engineering
**Date:** 2026-03-26
**Status:** Active
**Investigator:** architect session (orch-go-y012i)

---

## Question

The completion-verification model has 14 gates but **no gate that enforces file-to-issue ownership** — an agent can close an issue while leaving tracked dirty files behind, and another agent can inherit a dirty worktree full of unscoped changes. Does adding ownership-based enforcement (spawn-time manifests, commit-time scope gates, close-time reconciliation) close this gap? Or does it create a new instance of Class 5 (contradictory authority signals) by adding yet another authority source?

### Specific Claims Under Test

1. **Completion-verification model claim:** "Gates are structurally independent but functionally level-selective." Does ownership enforcement fit cleanly into the V0-V3 level hierarchy, or does it cross-cut all levels?

2. **Harness-engineering model claim:** "Gates work through signaling, not blocking — 100% bypass rate on blocking gates." Would an ownership gate repeat the accretion-gate failure (bypassed instantly) or is ownership different because the invariant is binary (owned vs unowned), not continuous (line count)?

3. **Harness-engineering model claim:** "Agent failure is a harness bug, not an agent bug." The split-commit defect (orch-go-r7avo: callers committed without implementations) was a build-breaking failure that slipped past completion. Is this a missing gate (harness bug) or a worker error that no gate can prevent?

4. **Defect-class taxonomy:** Does the dirty-worktree problem constitute a new defect class, or is it a composition of existing classes (Class 3 stale accumulation + Class 5 contradictory authority)?

---

## What I Tested

### Evidence Gathering

1. **Current dirty-worktree composition** — analyzed git status output:
   - 2 modified tracked files (source code): `.beads/issues.jsonl` (always dirty), `cmd/orch/serve_briefs.go` (open issue orch-go-c29fl)
   - 7,294 deleted entries: all `.orch/workspace/` files tracked before gitignore rule was added
   - 33+ dirty tracked source/docs files identified by orch-go-1selx probe

2. **Split-commit defect** — verified build failure:
   - Commit `da9b666b4` (closed issue orch-go-r7avo) committed callers in `pkg/daemon/ooda.go` referencing `RouteModel()` and 4-arg `RouteIssueForSpawn()` that only exist in the dirty tree
   - Build gate exists but wasn't run between commit and close (daemon path or --headless)

3. **Existing infrastructure** — catalogued what already exists for ownership:
   - `AGENT_MANIFEST.json`: WorkspaceName, BeadsID, GitBaseline, Skill
   - `git diff <baseline>..HEAD` in `pkg/verify/git_diff.go`: already computes files touched since spawn
   - SYNTHESIS.md Delta section: agents already claim files they modified
   - No forward-looking scope declaration (what files agent IS ALLOWED to modify)
   - No reconciliation between issue closure and working-tree state

4. **Hook bypass rates** — reviewed decisions:
   - Accretion gates: 100% bypass rate over 2 weeks (decision 2026-03-17)
   - Git-add-all hook: fires on regex, bypassed by quoting or alternative syntax
   - Governance file protection: effective because it's an Edit/Write block (not Bash regex)

5. **Contradictory skill text** — verified:
   - worker-base SKILL.md: "NEVER use `git add -A` or `git add .`"
   - feature-impl validation.md: "git add -A && git commit" (4+ occurrences)
   - systematic-debugging completion.md: "git add . && git commit"

---

## What I Observed

### Finding 1: The primary invariant must be ownership, not cleanliness

The current harness protects the wrong abstraction. Evidence:
- 7,294 of 7,296 dirty entries are historical artifacts — they make `git status` illegible but are NOT harmful
- The 2 actual source-code modifications are both legitimately dirty (one is intentionally-dirty beads state, one belongs to an open issue)
- The BUILD-BREAKING problem (split-commit) wasn't dirty-file noise — it was a closed issue leaving uncommitted implementations behind

**Conclusion:** "Clean worktree" is the wrong invariant. "Every tracked dirty file is owned by an open issue or an allowed artifact class" is the right one.

### Finding 2: Ownership enforcement avoids the bypass problem

Accretion gates failed because:
- The invariant was continuous (line count) — agents always have "one more line" arguments
- The gate could be bypassed by adding a flag or environment variable

Ownership is different:
- The invariant is binary (file is owned or it isn't) — no gradient to argue about
- The enforcement point is issue closure (not commit-time) — the agent can't bypass it by adding flags because closure is orchestrator-controlled
- The "escape hatch" is explicit: transfer ownership to another issue or classify as allowed residue

**However:** Spawn-time scope manifests (restricting which files an agent CAN write) would repeat the blocking-gate mistake. Agents need to modify unexpected files all the time (discovery during implementation). Scope restriction would have a high false-positive rate.

### Finding 3: Three-layer design collapses to one effective layer

The proposed 3-layer design (spawn-time manifest, commit-time gate, close-time reconciliation) sounds complete but:

- **Layer 1 (spawn-time scope restriction):** Would block legitimate work. Agent spawned to "fix the daemon" might need to touch 8 files discovered during debugging. Declaring scope at spawn time is premature — the agent doesn't know what files it needs until it starts working.
- **Layer 2 (commit-time ownership gate):** Duplicates Layer 3 less effectively. If we verify ownership at close time, commit-time checks add friction without new information.
- **Layer 3 (close-time reconciliation):** This is the only layer that works. At close time, we know: (a) which files the agent modified (git diff from baseline), (b) whether those modifications are committed, (c) whether dirty tracked files exist that should have been committed.

**Effective design: Layer 3 only, with Layers 1 and 2 as optional advisory signals.**

### Finding 4: The split-commit defect is a build gate timing problem

The build gate (#10) is the only unfakeable gate. But it only runs at `orch complete` time. The split-commit defect (orch-go-r7avo) happened because:
1. Agent committed partial work (callers without implementations)
2. Agent reported Phase: Complete
3. Completion ran verification — but `go build` may have passed against the WORKING TREE (which had the implementations) not the COMMITTED STATE
4. Issue closed with a broken committed state

**Fix:** Build gate must run against committed state (`git stash && go build ./... && git stash pop`), not working tree. This is orthogonal to ownership design but surfaced by the same investigation.

### Finding 5: Artifact classification resolves the noise problem

The 7,294 "deleted" entries from `.orch/workspace/` are tracked-then-gitignored historical files. Three artifact classes need explicit policy:

| Class | Examples | Policy |
|-------|----------|--------|
| Source/docs | .go, .md, .kb/ | Must be owned by issue at all times |
| Generated workspace | .orch/workspace/ | Untrack via `git rm --cached` |
| Experiment results | experiments/ | Untrack or archive per experiment lifecycle |
| Local state | .beads/, .claude/ | Never track (gitignored) |

### Finding 6: Skill text contradiction IS a Class 5 defect instance

Two authority sources disagree about the same behavior (git staging):
- worker-base says NEVER blanket stage
- feature-impl validation says DO blanket stage

This is textbook Class 5 (contradictory authority signals). The fix is single canonical derivation: worker-base is authoritative for git staging behavior, feature-impl validation must not contradict it.

---

## Model Impact

### Completion-verification model

**EXTENDS:** The model documents 14 gates but has no "dirty-worktree reconciliation" gate. This probe recommends adding:
- **Gate 15: Ownership Reconciliation** — at completion, verify all tracked dirty files since baseline are either committed or owned by another issue. Type: Evidence. Level: V2+.
- **Build gate clarification:** Gate #10 should run against committed state, not working tree, to prevent split-commit defects.

**CONFIRMS:** "Gates are structurally independent but functionally level-selective" — ownership reconciliation fits cleanly at V2 level alongside git_diff and build gates. It does not cross-cut all levels.

### Harness-engineering model

**EXTENDS with new principle:** Not all gates are equal in bypass resistance. The bypass rate correlates with invariant type:
- **Continuous invariants** (line counts, timing thresholds): high bypass rate — agents always find edge cases
- **Binary invariants** (file owned or not, build passes or not): low bypass rate — no gradient to exploit
- **Orchestrator-controlled enforcement points** (issue closure): zero agent bypass — agent cannot close its own issue

**CONFIRMS:** "Agent failure is a harness bug" — the split-commit defect and the contradictory skill text are both harness failures, not agent errors. The agent followed its skill text (feature-impl said `git add -A`). The harness text was wrong.

### Defect-class taxonomy

**CONFIRMS:** Dirty-worktree is NOT a new defect class. It is a composition of:
- **Class 3 (Stale Artifact Accumulation):** .orch/workspace/ historical tracked files never cleaned up
- **Class 5 (Contradictory Authority Signals):** worker-base vs feature-impl staging guidance
- **Class 0 (Scope Expansion):** git status output expanded by stale artifacts, making real changes invisible

---

## Recommendation Summary

1. **Primary invariant:** Every tracked dirty file must be owned by an open issue or belong to an allowed artifact class. Clean worktree is not the goal.
2. **Single effective enforcement layer:** Close-time reconciliation gate (Gate 15). Spawn-time and commit-time layers are advisory at most.
3. **Artifact classification:** Untrack `.orch/workspace/` and `experiments/` results via `git rm --cached`. This eliminates 99.7% of current dirty entries.
4. **Skill text fix:** Remove all `git add -A` / `git add .` from feature-impl and systematic-debugging skills. Worker-base is canonical.
5. **Build gate hardening:** Run `go build` against committed state, not working tree, to catch split-commit defects.
6. **Hook simplification:** Replace command-regex policing (`gate-git-add-all.py`) with a commit-time ownership check. The regex prevents a symptom; ownership prevents the cause.

(Full design in investigation: `.kb/investigations/2026-03-26-design-ownership-based-harness-prevention.md`)
