## Summary (D.E.K.N.)

**Delta:** Daemon auto-closed 22 agent issues despite zero git commits because no verification gate checks for committed (vs uncommitted) code; the git_diff gate uses `git diff --name-only` which shows working tree changes, and the GPT model bypass auto-passes Phase: Complete for OpenAI models.

**Evidence:** git reflog shows only `bd sync` commits in overnight window; `pkg/verify/git_diff.go:218-226` uses `git diff --name-only HEAD` or `git diff --name-only <baseline>` (both include uncommitted changes); `pkg/verify/check.go:642-647` bypasses Phase: Complete gate for GPT models; `pkg/daemon/completion_processing.go:475-508` closes issues after verification passes.

**Knowledge:** Three independent safety gaps compounded: (1) no "commit exists" gate, (2) git_diff gate conflates working tree with committed history, (3) GPT model bypass removes Phase: Complete as a meaningful signal. All 22 agents' uncommitted changes were visible to each other's verification because they share one working tree.

**Next:** Add a `GateCommitEvidence` gate that requires `git log --since=<spawn_time>` to show at least one commit. This is an **architectural** change (new gate in verification system affects daemon auto-complete, manual orch complete, and review paths).

**Authority:** architectural - New verification gate crosses daemon, complete pipeline, and verify package boundaries

---

# Investigation: Post-Mortem — Daemon Overnight Ghost Completions

**Question:** Why did the daemon auto-close 22 beads issues when zero git commits landed, and what safeguards are missing?

**Started:** 2026-02-09
**Updated:** 2026-02-09
**Owner:** investigation agent
**Phase:** Complete
**Next Step:** None — findings ready for orchestrator review
**Status:** Complete

**TLDR:** Daemon overnight run (08:33–09:10) spawned ~22 gpt-5.3-codex agents via OpenCode headless. All agents wrote code and reported Phase: Complete, but none ran `git commit`. The daemon's verification pipeline passed all 22 because: (a) no gate checks for actual commits, (b) the git_diff gate shows uncommitted working tree changes as valid, (c) the Phase: Complete gate is auto-bypassed for GPT models. Result: 22 issues closed with zero committed code, 81+ files interleaved in working tree.

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| `.kb/models/daemon-autonomous-operation.md` | extends | yes | Model documents completion polling but not verification gaps |
| `.kb/models/agent-lifecycle-state-model/probes/2026-02-08-commit-idle-auto-completion.md` | extends | pending | That probe explores commit+idle detection; this investigation shows the commit detection itself is broken |
| `.kb/investigations/2026-02-08-inv-probe-system-lifecycle-08feb.md` | extends | yes | Prior probe found "feedback loop closes in working tree but breaks at commit boundary" — confirmed here at scale |

---

## Findings

### Finding 1: No "commit exists" verification gate

**Evidence:** Examined all 14 gate constants in `pkg/verify/check.go:17-33`. The gates are: `phase_complete`, `synthesis`, `handoff_content`, `constraint`, `phase_gate`, `skill_output`, `visual_verification`, `test_evidence`, `model_connection`, `verification_spec`, `git_diff`, `build`, `decision_patch_limit`, `dashboard_health`. None verify that actual git commits exist. The `git_diff` gate (the closest candidate) checks that files claimed in SYNTHESIS.md appear in `git diff` output — but `git diff` shows working tree changes, not committed history.

**Source:** `pkg/verify/check.go:17-33` (gate constants), `pkg/verify/git_diff.go:366-457` (VerifyGitDiff implementation), `pkg/verify/git_diff.go:218-226` (GetGitDiffFiles)

**Significance:** This is the root cause. The entire verification pipeline assumes agents commit their work. When they don't, the system has no way to detect it. The git_diff gate was designed to catch agents who *claim* to modify files they didn't touch — not to catch agents who touched files but never committed.

---

### Finding 2: git_diff gate conflates working tree with committed history

**Evidence:** `GetGitDiffFiles` at `pkg/verify/git_diff.go:218-251` has three code paths:
1. With baseline: `git diff --name-only <baseline>` — shows ALL changes (staged+unstaged+committed) since baseline
2. With spawn time only: `git log --name-only --since=<time>` — shows only COMMITTED changes
3. Neither (zero time, no baseline): `git diff --name-only HEAD` — shows uncommitted changes only

Most overnight agents had a git baseline from `AGENT_MANIFEST.json` (set at spawn time), so path #1 was taken. This path includes all 81+ uncommitted files from all 22 agents in a single diff output. Each agent's claimed files appeared valid because the working tree contained them.

**Source:** `pkg/verify/git_diff.go:218-226`, `pkg/verify/git_diff.go:405-407` (baseline loaded from manifest)

**Significance:** With 22 agents sharing one working tree, Agent A's uncommitted changes are visible to Agent B's verification. The git_diff gate becomes meaningless for detecting individual agent work when agents don't commit. Path #2 (using `git log --since`) would have been more correct but only fires when baseline is empty.

---

### Finding 3: GPT model bypass auto-passes Phase: Complete

**Evidence:** `shouldBypassPhaseCompleteForModel()` at `pkg/verify/check.go:809-825` checks if the workspace model is from OpenAI (GPT provider) and auto-bypasses the Phase: Complete gate:

```go
func shouldBypassPhaseCompleteForModel(workspacePath string) bool {
    manifest, err := spawn.ReadAgentManifest(workspacePath)
    resolved := model.Resolve(strings.TrimSpace(manifest.Model))
    if strings.ToLower(resolved.Provider) != "openai" {
        return false
    }
    return strings.Contains(strings.ToLower(resolved.ModelID), "gpt")
}
```

This was added because "GPT models frequently miss Phase: Complete reporting." But in this incident, the agents DID report Phase: Complete via `bd comment`. The bypass is overly broad — it removes the Phase: Complete gate entirely for GPT models, rather than just making it a warning.

**Source:** `pkg/verify/check.go:639-647` (bypass logic), `pkg/verify/check.go:809-825` (model check)

**Significance:** While not the primary cause (agents did report Phase: Complete), this bypass removes a defense layer. If agents had NOT reported Phase: Complete, the bypass would have let them through anyway. This is a safety regression — the Phase: Complete gate is the daemon's primary signal for detecting completion.

---

### Finding 4: ProcessCompletion closes issues after verification passes with no commit check

**Evidence:** `ProcessCompletion` in `pkg/daemon/completion_processing.go:416-509` follows this flow:
1. Run `VerifyCompletionFull()` — all gates pass (no commit gate exists)
2. Determine escalation level — returns `EscalationNone` or `EscalationInfo` for clean completions
3. Check `escalation.ShouldAutoComplete()` — returns true for levels ≤ `EscalationReview`
4. Call `verify.CloseIssueForce()` — issue closed

The `CloseIssueForce` call at line 503 uses `force=true` which bypasses bd's own Phase: Complete check, passing `true` for the `skipPhaseComplete` parameter. This was intentional ("bypass bd's redundant Phase: Complete gate since we already verified it via ListCompletedAgents") but compounds the problem.

**Source:** `pkg/daemon/completion_processing.go:475-509`

**Significance:** The daemon's auto-close path trusts the verification pipeline completely. Since no gate checks for commits, the daemon happily closes issues with zero committed work.

---

### Finding 5: git reflog confirms zero agent commits overnight

**Evidence:** Running `git log --oneline --since="2026-02-08T20:00:00" --until="2026-02-09T10:00:00"` shows only:
- `bd sync` commits (automated beads state persistence)
- Manual Dylan commits (probe/model work from before the daemon run)
- One manual batch commit `0143b2c8` ("commit 9 uncommitted probes and 13 model updates") — this was Dylan rescuing agent output after noticing the problem

Zero commits from any of the 22 headless agents. The agents wrote code to the working tree, reported Phase: Complete, and exited without committing.

**Source:** `git reflog --all -30`, `git log --oneline --since="2026-02-08T20:00:00"`

**Significance:** Confirms the hypothesis. This is not a race condition or timing issue — agents genuinely never committed. Likely a gpt-5.3-codex behavioral gap (model doesn't know to `git commit` in OpenCode headless mode).

---

### Finding 6: Issue orch-go-21509 is open (not a ghost completion)

**Evidence:** `bd show orch-go-21509` shows status: `open`, type: `feature`, created 2026-02-09 09:06. Events log shows a `daemon.dedup_blocked` event indicating the daemon tried to spawn it but it was blocked by the ProcessedCache. No auto-completion event exists for this issue.

**Source:** `bd show orch-go-21509`, `events.jsonl` grep for 21509

**Significance:** The reported ghost completion for 21509 appears to be a misattribution. The issue is currently open and was never closed. It may have been confused with another issue during the chaotic overnight run.

---

## Synthesis

**Key Insights:**

1. **Missing gate is the root cause** — The verification system has 14 gates but none check for actual git commits. This is a design gap, not a regression — the system never had this gate. It was masked by Claude models that reliably commit their work.

2. **Shared working tree makes per-agent verification meaningless** — With 22 agents writing to one directory, the git_diff gate sees ALL 81+ files in every agent's verification. This means an agent that wrote zero files would still pass the git_diff gate because other agents' uncommitted changes are visible.

3. **GPT model behavioral gap exposed system assumptions** — The system was implicitly designed around Claude model behavior (which commits reliably). The gpt-5.3-codex model exposed that the system assumed but never verified the commit step.

4. **Defense in depth failure** — Three independent assumptions broke simultaneously: (a) agents commit, (b) git_diff checks committed state, (c) Phase: Complete implies committed work. No single gate would have caught this; the system needed at least one explicit commit check.

**Answer to Investigation Question:**

The daemon auto-closed 22 issues because the verification pipeline has no gate that checks for actual git commits. The git_diff gate checks working tree state (which includes uncommitted changes from all 22 agents), and the Phase: Complete gate was either reported by agents or auto-bypassed for GPT models. The daemon's `ProcessCompletion` trusts the verification pipeline and closes issues when verification passes. Since all gates passed, all 22 issues were closed despite zero committed code.

---

## Structured Uncertainty

**What's tested:**

- ✅ git reflog confirms zero agent commits in overnight window (verified: `git log --since/--until` command)
- ✅ `GetGitDiffFiles` uses `git diff --name-only` which includes uncommitted changes (verified: read source code at git_diff.go:218-226)
- ✅ `shouldBypassPhaseCompleteForModel` auto-bypasses for GPT/OpenAI models (verified: read source code at check.go:809-825)
- ✅ `ProcessCompletion` calls `CloseIssueForce` with force=true after verification passes (verified: read source at completion_processing.go:503)
- ✅ orch-go-21509 is currently `open` status (verified: `bd show orch-go-21509`)

**What's untested:**

- ⚠️ Whether gpt-5.3-codex specifically lacks `git commit` behavior or if OpenCode headless suppresses it (would need to review OpenCode session transcripts)
- ⚠️ Whether all 81 files can be attributed to specific issues via session transcripts (not attempted)
- ⚠️ Whether the two concurrent daemon instances caused duplicate auto-completions (events.jsonl shows some duplicates but root cause unconfirmed)

**What would change this:**

- If gpt-5.3-codex DID commit but commits were lost (e.g., git reset), the root cause would shift from "no commit gate" to "commit destruction"
- If OpenCode headless mode blocks git operations in sandbox, the fix would need sandbox-level changes not just new gates

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Add `GateCommitEvidence` verification gate | architectural | New gate in verify package affects daemon, complete pipeline, and review paths |
| Fix git_diff gate to use committed-only diff | architectural | Changes verification semantics across all callers |
| Restrict GPT model bypass scope | implementation | Narrow existing bypass, no new patterns |
| Add agent commit instruction to spawn context | implementation | Template change within existing spawn system |

### Recommended Approach: Add GateCommitEvidence

**New verification gate** that checks `git log --since=<spawn_time> --oneline` returns at least one commit, or `git log <baseline>..HEAD --oneline` shows commits since baseline.

**Why this approach:**
- Directly addresses root cause (no commit check exists)
- Uses existing spawn metadata (spawn_time, git_baseline from AGENT_MANIFEST.json)
- Fits cleanly into existing gate architecture (new constant, new check function)
- Works for both daemon auto-complete and manual `orch complete`

**Trade-offs accepted:**
- Agents that legitimately produce zero code changes (pure investigation) would need a bypass
- Existing "light tier" spawns may need adjustment

**Implementation sequence:**
1. Add `GateCommitEvidence = "commit_evidence"` to `pkg/verify/check.go` gate constants
2. Implement `checkCommitEvidence()` in `pkg/verify/` using git log with baseline/spawn_time
3. Add to `verifyWorkerGates()` call chain in `check.go:381-406`
4. Make it a CoreGate (always run, even in batch mode)
5. Add `--skip-commit-evidence` flag for override (e.g., investigation-only agents)

### Alternative: Fix git_diff gate to distinguish committed vs uncommitted

Instead of a new gate, modify `GetGitDiffFiles` to use `git log --name-only --since=<time>` (committed only) instead of `git diff --name-only <baseline>` (includes uncommitted).

- **Pros:** No new gate, fixes existing semantics
- **Cons:** Changes git_diff gate purpose (currently "do claimed files exist in diff", would become "do claimed files exist in committed history"); may break agents that haven't committed yet at verification time
- **When to use:** If adding a new gate is deemed too complex

### Immediate Mitigation: Spawn context instructs commit

Add explicit `git commit` instruction to SPAWN_CONTEXT.md template for GPT models, or enforce it in the session complete protocol section of worker-base skill.

---

### Things to watch out for:
- ⚠️ Investigation-only agents (no code changes expected) would fail the commit evidence gate — need skill-aware bypass
- ⚠️ The GPT model bypass at check.go:642 should be narrowed from "auto-pass" to "warn-only" to preserve Phase: Complete as a meaningful signal
- ⚠️ Concurrent daemon instances can cause duplicate auto-completions — needs cross-instance dedup (separate issue)

**Success criteria:**
- ✅ Running daemon overnight with GPT models results in zero issues closed without commits
- ✅ `orch complete <id>` fails when agent workspace has zero commits since spawn
- ✅ Investigation-only agents can bypass with `--skip-commit-evidence`

---

## References

**Files Examined:**
- `pkg/verify/check.go:17-33, 261-274, 577-685, 809-825` — Gate constants, VerifyCompletionFull, Phase: Complete bypass
- `pkg/verify/git_diff.go:218-251, 366-457` — GetGitDiffFiles, VerifyGitDiff
- `pkg/daemon/completion_processing.go:28-54, 104-173, 416-509, 511-554` — CompletionConfig, ListCompletedAgents, ProcessCompletion, CompletionOnce
- `cmd/orch/daemon_loop.go:512-555` — processCompletions (daemon main loop)
- `cmd/orch/complete_gates.go:38-79, 110-217` — verifyCompletion, verifyRegularAgent
- `pkg/verify/escalation.go:57-60, 127-172` — ShouldAutoComplete, DetermineEscalation

**Commands Run:**
```bash
# Confirm zero agent commits overnight
git log --oneline --since="2026-02-08T20:00:00" --until="2026-02-09T10:00:00"

# Count uncommitted files
git status --short | wc -l

# Check reflog for all commits
git reflog --all -30

# Check 21509 issue status
bd show orch-go-21509

# Count auto-completion events
grep -c "session.auto_completed\|daemon.complete" ~/.orch/events.jsonl
```

**Related Artifacts:**
- **Model:** `.kb/models/daemon-autonomous-operation.md` — Daemon behavior model (extends: adds failure mode #4)
- **Model:** `.kb/models/agent-lifecycle-state-model.md` — Agent lifecycle (extends: commit boundary gap)
- **Probe:** `.kb/models/agent-lifecycle-state-model/probes/2026-02-08-commit-idle-auto-completion.md` — Related commit+idle detection work
