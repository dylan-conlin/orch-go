# Design: Ownership-Based Harness to Prevent Dirty-Worktree Recurrence

**Date:** 2026-03-26
**Status:** Complete
**Model:** completion-verification, harness-engineering, defect-class-taxonomy
**Issue:** orch-go-y012i
**Next:** Implementation slices (see Decomposition section)

---

## Design Question

How should the harness be redesigned so the enforced invariant shifts from "clean working tree" to "every tracked dirty file is owned by an open issue or an allowed artifact class"?

---

## Problem Framing

### Success Criteria

1. A closed issue cannot leave behind tracked dirty files that are unowned
2. `git status` is legible — noise from historical tracked-but-gitignored artifacts is eliminated
3. Worker skill text and hook enforcement express the same policy (no contradictory authority)
4. Hook failures degrade visibly — crashes never masquerade as policy denials
5. The design doesn't repeat the accretion-gate mistake (100% bypass rate on blocking gates)

### Constraints

- Preserve ability to keep a deliberately dirty tree when the dirt is legible
- No fragile shell regexes as the primary control plane
- Generated artifacts must not dominate the same status surface as source edits
- Hooks must degrade visibly when they fail
- Must work for both claude and opencode spawn backends
- Governance-protected files (pkg/spawn/gates/, pkg/verify/accretion.go, .orch/hooks/) require orchestrator sessions to modify

### Scope Boundaries

**IN:** Invariant definition, artifact classification, enforcement layer design, hook architecture, skill text alignment, migration path
**OUT:** Implementation code, CI/CD pipeline changes, cross-project enforcement

---

## Exploration: 6 Decision Forks

### Fork 1: What Is the Primary Invariant?

**Option A:** Clean working tree (current implicit invariant)
- Evidence against: 7,294 of 7,296 dirty entries are harmless historical artifacts. Cleanliness is noise-dominated. Enforcing it would require untracking 99.7% of current dirt before enforcement is meaningful.

**Option B:** Every tracked dirty file must be owned by an open issue or belong to an allowed artifact class
- Evidence for: The 2 actual dirty source files are both legitimately owned (one by .beads/ local-state class, one by open issue orch-go-c29fl). The BUILD-BREAKING failure (split-commit on orch-go-r7avo) was an ownership failure, not a cleanliness failure — a closed issue left uncommitted implementations behind.

**Recommendation: Option B** — ownership, not cleanliness.

**Substrate:** Harness-engineering model: "Agent failure is a harness bug." The current harness treats dirt as the failure. But dirt with clear ownership is not a failure — it's work in progress. Unowned dirt is the failure, and the harness doesn't check for it.

---

### Fork 2: What Artifact Classes Should Exist?

| Class | Examples | Tracking Policy | Ownership Rule |
|-------|----------|----------------|----------------|
| **source** | .go, .ts, .svelte files | Tracked | Must be owned by open issue when dirty |
| **docs** | .md in .kb/, docs/, skills/ | Tracked | Must be owned by open issue when dirty |
| **generated-workspace** | .orch/workspace/\* | **Untracked** (gitignored, but historical entries still in index) | No ownership needed — ephemeral per-session |
| **experiment-results** | experiments/\*/results/ | **Untracked** (gitignored, but historical entries still in index) | No ownership needed — archived separately |
| **local-state** | .beads/issues.jsonl, .claude/ | **Untracked** (already gitignored) | Always dirty, no ownership needed |
| **knowledge-backlog** | .kb/ files not tied to specific issues | Tracked | Allowed-residue class — committed in batches |

**Recommendation:** Formally define these 6 classes. Untrack generated-workspace and experiment-results via `git rm --cached -r .orch/workspace/ experiments/*/results/`. This eliminates 99.7% of current dirty entries.

**Substrate:** Defect-class taxonomy, Class 3 (Stale Artifact Accumulation): "Dead state never cleaned up, interferes later." The 7,294 historical workspace entries are textbook Class 3.

---

### Fork 3: Where Should Enforcement Happen?

**Option A:** Spawn-time scope restriction (manifest declaring allowed-write paths)
- Problem: Agents discover scope during work. An agent spawned to "fix the daemon" might touch 8 files found during debugging. Declaring scope at spawn time has a high false-positive rate.
- Accretion-gate lesson: Blocking gates that restrict legitimate work are bypassed 100% of the time.
- Verdict: **Advisory only** — useful for spawn context ("here's what we expect you'll touch") but not enforcement.

**Option B:** Commit-time staged-file gate
- Problem: Duplicates close-time check with less information. At commit time we don't know if the issue is closing.
- The git-add-all hook already operates here (regex-based) and has documented false-positive issues.
- Verdict: **Replace git-add-all regex with advisory ownership check** — warn if staged files are outside expected scope, never block.

**Option C:** Close-time reconciliation gate
- This is the only point where we have full information: which files were modified (git diff from baseline), which are committed, which are still dirty, and whether the issue is closing.
- Cannot be bypassed by agents — issue closure is orchestrator-controlled via `orch complete`.
- Binary invariant (owned/unowned) avoids the continuous-invariant bypass problem.
- Verdict: **Primary enforcement layer.**

**Option D:** Pre-push sanity check
- Useful as defense-in-depth but not the primary layer.
- Verdict: **Advisory only** — warn if pushing with unowned dirty tracked files.

**Recommendation: Option C as primary, Options A/B/D as advisory signals.**

**Substrate:** Harness-engineering model: "Gates work through signaling, not blocking." Close-time reconciliation is different from accretion gates because (1) the invariant is binary, and (2) the enforcement point is orchestrator-controlled. But Options A and B should follow the advisory pattern that worked for accretion.

---

### Fork 4: What Should Replace Command-Regex Policing?

**Current state:** `gate-git-add-all.py` uses regex to block `git add -A` and `git add .` in Bash commands. Documented problems:
- False positives on quoted strings (orch-go-gzrzl)
- CWD resolution failures bricking entire sessions (orch-go-w5ais)
- 2 implementations (project-level + global worker-level) with a governance lock preventing worker from fixing the global one
- Polices syntax (how files are staged) rather than semantics (what files are staged)

**Recommendation:** Phase out `gate-git-add-all.py` in favor of:

1. **Close-time reconciliation** (new Gate 15) — enforces what matters: dirty tracked files must be owned
2. **Skill text alignment** — remove all `git add -A` from skills so agents don't try it in the first place (soft harness)
3. **Advisory commit-time check** (optional, future) — if the hook infrastructure gets logging/self-test capability, an advisory check that warns "you're staging files outside your expected scope" is helpful but not blocking

**Substrate:** Harness-engineering model: "Hard harness doesn't need measurement — a build passes or fails." The ownership reconciliation is a binary check (owned/unowned). The regex was continuous (matches/doesn't-match with false positives). Binary checks are more robust.

---

### Fork 5: How Should Hook Governance Work?

**Current problems:**
- Zero invocation logging (hooks fire silently)
- Exit code 2 = deny decision (file-not-found masquerades as policy denial)
- 12 registrations across 2 settings.json, 1 duplicate
- Dead hook (pre-commit-knowledge-gate.py for dead `kn` CLI)
- Path resolution varies: `~/` (global), `$CLAUDE_PROJECT_DIR` (project), relative (broken)

**Recommendation: 5 requirements for hook runtime**

1. **Single settings source:** One canonical `orch harness hooks` command that generates both global and project-level settings.json from a declaration file. No hand-editing settings.json.

2. **Absolute paths only:** All hook registrations must use `$CLAUDE_PROJECT_DIR` (project-scope) or `$HOME` (global-scope). Relative paths are rejected at generation time.

3. **Logging:** Every hook invocation logs to `~/.orch/logs/hooks.jsonl`: timestamp, hook name, event, matcher, decision (allow/deny/error), duration_ms, exit_code. This makes "total agent blockage from broken hook" immediately diagnosable.

4. **Self-test:** `orch harness hooks test` runs each registered hook with synthetic inputs and verifies: (a) script exists at path, (b) script executes without error, (c) output is valid JSON, (d) response time < 500ms.

5. **Dedup:** Hook registration dedup at generation time. Currently `gate-worker-git-add-all.py` is registered twice.

**Substrate:** Harness-engineering model: "Every harness layer requires both an enforcement surface and a measurement surface." Current hooks have enforcement but zero measurement. Adding logging and self-test closes this gap.

---

### Fork 6: How Should Worker Skill Text Change?

**Current contradiction (Class 5 — Contradictory Authority Signals):**
- `worker-base`: "NEVER use `git add -A` or `git add .`"
- `feature-impl/validation.md`: "git add -A && git commit" (4 occurrences)
- `systematic-debugging/completion.md`: "git add . && git commit"

**Recommendation:**

1. **Remove all `git add -A` and `git add .` from feature-impl and systematic-debugging skills.** Replace with: "Stage only files you modified: `git add <specific-files>`"

2. **Worker-base is canonical for git staging behavior.** No other skill can override this. Add a comment in feature-impl validation: "Git staging: see worker-base (canonical)"

3. **No new hook enforcement for this.** The skill text fix + close-time reconciliation gate is sufficient. The regex hook was compensating for contradictory skill text — fix the text, the hook becomes unnecessary.

**Substrate:** Defect-class taxonomy, Class 5: "Multiple sources of truth disagree, fixes oscillate." Single canonical derivation: worker-base owns git staging behavior.

---

## Synthesis: The Ownership Harness Design

### Primary Invariant

**Dirty is acceptable. Unowned tracked work and ambiguous artifact classes are not.**

Enforcement: At issue close time (`orch complete`), the completion pipeline verifies that no tracked dirty files remain from this agent's work that are not:
- Committed to the repo, OR
- Owned by another open issue, OR
- Classified as allowed residue (knowledge-backlog, local-state)

### Architecture: 1 Enforcement Layer + 3 Advisory Layers

#### Layer 0 (Advisory): Spawn-Time Scope Context

**What:** SPAWN_CONTEXT.md includes "Expected file scope" section listing files/dirs the issue is likely to touch. Derived from issue description, skill type, and area detection.

**How:** Extend `GenerateContext()` in `pkg/spawn/context.go` to include a `## Expected Scope` section. No enforcement — agents can (and should) modify files outside this list when needed.

**Why advisory:** Premature scope restriction blocks legitimate work. But scope context helps agents self-organize and makes close-time reconciliation legible ("I touched files X, Y, Z — X and Y were expected, Z was discovered during debugging").

#### Layer 1 (Advisory): Commit-Time Ownership Signal

**What:** If a commit-time hook fires, it provides an advisory message: "These staged files are outside your expected scope: [list]. This is fine — just be aware during completion."

**When to build:** Only after hook runtime has logging and self-test capability (Fork 5). Without logging, a crash in this hook becomes silent agent blockage.

**Not yet — defer this layer.** The close-time gate handles the invariant; this layer adds UX without new invariants.

#### Layer 2 (Enforcement): Close-Time Reconciliation Gate — Gate 15

**What:** New completion verification gate: `GateOwnershipReconciliation = "ownership_reconciliation"`

**When it fires:** V2+ (same level as git_diff and build gates)

**Algorithm:**

```
1. Get git baseline from AGENT_MANIFEST.json
2. Compute files modified since baseline: git diff --name-only <baseline>
3. Compute tracked dirty files: git diff --name-only HEAD (uncommitted changes)
4. For each tracked dirty file:
   a. Is it in an allowed artifact class? (local-state, knowledge-backlog) → PASS
   b. Is it owned by another open issue? (check beads for file-to-issue mapping) → PASS
   c. Is it in the agent's own diff but uncommitted? → FAIL (agent left work behind)
   d. Is it a pre-existing dirty file (dirty before agent's baseline)? → PASS (not this agent's responsibility)
5. FAIL if any tracked dirty file is unowned and post-baseline
```

**Key design choice:** Pre-existing dirt is excluded. The gate only checks for NEW dirty files introduced since the agent's baseline. This avoids blocking completion on dirt created by other agents.

**Integration:** Add to `pkg/verify/check.go` alongside existing gates. Returns `GateResult` with pass/fail, owned files, unowned files, and recommended actions.

#### Layer 3 (Advisory): Pre-Push Sanity Check

**What:** `pre-push` git hook warns if pushing with unowned tracked dirty files.

**When to build:** Low priority. Close-time gate catches this at a better enforcement point.

### Supporting Infrastructure

#### 1. Artifact Class Registry

A simple map in `pkg/verify/artifact_classes.go`:

```go
var ArtifactClasses = map[string]ArtifactPolicy{
    "source":              {Tracked: true, RequiresOwnership: true},
    "docs":                {Tracked: true, RequiresOwnership: true},
    "generated-workspace": {Tracked: false, Pattern: ".orch/workspace/**"},
    "experiment-results":  {Tracked: false, Pattern: "experiments/*/results/**"},
    "local-state":         {Tracked: false, Pattern: ".beads/**,.claude/**"},
    "knowledge-backlog":   {Tracked: true, RequiresOwnership: false, AllowedResidue: true},
}
```

A file's class is determined by path matching against patterns. Unknown files default to `source` (requires ownership).

#### 2. Build Gate Hardening

The split-commit defect (orch-go-r7avo) happened because `go build` may have run against the working tree, not committed state. Fix:

```go
// In pkg/verify/build.go
// Before running go build, stash uncommitted changes
// Run go build against committed state only
// Restore stash after build
```

This is orthogonal to ownership but surfaced by the same investigation. Implementation: stash uncommitted changes, run `go build ./...`, restore stash. If build fails against committed state, the completion should flag it.

#### 3. Untrack Historical Artifacts

One-time migration to eliminate 99.7% of current dirty entries:

```bash
git rm --cached -r .orch/workspace/
git rm --cached -r experiments/coordination-demo/redesign/results/
git commit -m "chore: untrack historical workspace and experiment artifacts"
```

These paths are already in .gitignore. The `git rm --cached` removes them from the index without deleting local files.

#### 4. Skill Text Alignment

Remove contradictory `git add -A` / `git add .` from:
- `skills/src/worker/feature-impl/.skillc/phases/validation.md` (4 occurrences)
- `skills/src/worker/feature-impl/reference/phase-validation.md` (4 occurrences)
- `skills/src/worker/systematic-debugging/.skillc/completion.md` (1 occurrence)

Replace with: `git add <specific-files-you-modified>`

Then rebuild skills: `skillc build && skillc deploy`

#### 5. Hook Runtime Improvements (Future)

Add to hook infrastructure:
- **Logging:** `~/.orch/logs/hooks.jsonl` — every invocation recorded
- **Self-test:** `orch harness hooks test` — validates all registered hooks
- **Dedup:** Registration dedup at generation time
- **Path validation:** Reject relative paths at generation time

This enables retiring `gate-git-add-all.py` once the ownership gate is proven.

### Migration Path

**Phase 1: Clean the noise (immediate, no code changes)**
1. `git rm --cached -r .orch/workspace/` — untrack 7,294 historical entries
2. `git rm --cached -r experiments/coordination-demo/redesign/results/` — untrack experiment results
3. Commit: `chore: untrack historical workspace and experiment artifacts`
4. Verify: `git status` shows only actual source/docs changes

**Phase 2: Fix contradictory skill text (immediate, skill changes only)**
1. Remove `git add -A` from feature-impl validation and systematic-debugging
2. Rebuild and deploy skills
3. Retire need for `gate-git-add-all.py` (keep running but plan phase-out)

**Phase 3: Implement Gate 15 — Ownership Reconciliation (implementation sprint)**
1. Add `GateOwnershipReconciliation` constant to `pkg/verify/check.go`
2. Implement `VerifyOwnershipReconciliation()` in new file `pkg/verify/ownership.go`
3. Wire into `VerifyCompletionFull()` at V2+ level
4. Add artifact class registry
5. Test with existing dirty-worktree scenarios

**Phase 4: Build gate hardening (implementation sprint)**
1. Modify build gate to stash uncommitted changes before `go build`
2. Test against split-commit scenario

**Phase 5: Hook runtime improvements (future, lower priority)**
1. Add hook invocation logging
2. Add hook self-test
3. Phase out `gate-git-add-all.py` after ownership gate is proven in production

---

## Recommendations

### Recommendation 1: Shift the invariant from cleanliness to ownership

**Invariant:** "Every tracked dirty file must be owned by an open issue or belong to an allowed artifact class."

**Why:** The current implicit invariant (clean worktree) is violated by 7,296 entries, 99.7% of which are harmless. The harness spends energy on noise while missing the actual failure mode (closed issues leaving uncommitted work behind).

### Recommendation 2: Implement close-time reconciliation as the primary enforcement layer

**Gate 15:** `ownership_reconciliation` at V2+ level. Checks that no tracked dirty files remain from this agent's work that are unowned.

**Why:** This is the only enforcement point where we have full information AND the agent cannot bypass it (issue closure is orchestrator-controlled). Spawn-time and commit-time layers are advisory at most.

### Recommendation 3: Untrack historical artifacts to restore git status legibility

**Action:** `git rm --cached -r .orch/workspace/ experiments/*/results/`

**Why:** 7,294 deleted entries from historical tracked-but-gitignored files dominate git status, making real changes invisible. This is Class 3 (Stale Artifact Accumulation).

### Recommendation 4: Align skill text — single canonical authority for git staging

**Action:** Remove all `git add -A` / `git add .` from feature-impl and systematic-debugging. Worker-base is canonical.

**Why:** Contradictory skill text is Class 5 (Contradictory Authority Signals). The hook compensates for broken skill text. Fix the text; the hook becomes unnecessary.

### Recommendation 5: Harden build gate to run against committed state

**Action:** Stash uncommitted changes before `go build ./...` in the build gate.

**Why:** Split-commit defect (orch-go-r7avo) broke the build on committed state while the working tree was fine. Build gate may have verified working tree, not committed state.

### Recommendation 6: Plan hook runtime improvements before adding new hooks

**Action:** Add logging, self-test, and path validation to hook infrastructure before building any new commit-time hooks.

**Why:** Current hooks have zero observability. Adding more hooks without logging multiplies the silent-failure risk (orch-go-w5ais: broken hook path bricked entire agent session with zero diagnostics).

---

## Defect Class Exposure

| Design Component | Applicable Defect Classes | Mitigation |
|-----------------|--------------------------|------------|
| Ownership gate (Gate 15) | Class 1 (Filter Amnesia) — might miss files in new paths | Use git diff as canonical file list; don't maintain separate file tracker |
| Artifact class registry | Class 0 (Scope Expansion) — new file patterns not in registry | Default-to-ownership: unknown files require ownership (fail-safe) |
| Build gate stash | Class 7 (Premature Destruction) — stash could lose work on failure | Always `git stash pop` in defer; test stash/pop on dirty worktree |
| Hook logging | Class 3 (Stale Accumulation) — log files grow unbounded | TTL-based rotation (7 days) |

---

## Open Questions (Surfaced, Not Blocking)

1. **File-to-issue mapping:** How does the ownership gate know which open issue owns a dirty file? Options: (a) parse beads comments for file paths, (b) require agents to declare files in SYNTHESIS.md Delta section, (c) use git blame on recent commits to find the issue. Design recommends (b) since it's already partially implemented.

2. **Knowledge-backlog allowed-residue window:** How long can .kb/ files remain dirty without issue ownership before they're flagged? Design recommends: indefinitely (knowledge-backlog is an allowed-residue class), but with periodic batch-commit reminders.

3. **Cross-agent dirty-file inheritance:** When Agent B starts in a worktree dirtied by Agent A, should Agent B's ownership gate ignore pre-baseline dirt? Design says yes (pre-baseline exclusion), but this needs testing.

---

## References

- Probe: `.kb/models/completion-verification/probes/2026-03-26-probe-ownership-based-harness-design-evaluation.md`
- Decision (accretion gates advisory): `.kb/decisions/2026-03-17-accretion-gates-advisory-not-blocking.md`
- Decision (no code review gate): `.kb/decisions/2026-02-25-no-code-review-gate-expand-execution-verification.md`
- Reconciliation probe: `.kb/models/completion-verification/probes/2026-03-26-probe-dirty-worktree-closed-issue-reconciliation.md`
- Defect class taxonomy: `.kb/models/defect-class-taxonomy/model.md`
- Harness engineering model: `.kb/models/harness-engineering/model.md`
