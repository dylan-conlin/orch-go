# Investigation: Claude Code --worktree Flag for Agent Isolation

**Date:** 2026-02-27
**Status:** Complete
**Beads:** orch-go-3wga
**Type:** Investigation

---

## Problem Statement

Agents committing directly to master and leaving unstaged changes in the working tree causes cascading problems:
- `bd sync` can't pull (chicken-and-egg with unstaged beads export)
- Broken builds from partial implementations
- File contention between concurrent agents
- This session alone required 3 commits + 2 stashes just to get `bd sync` working

Claude Code now has a `--worktree` flag (`-w`) that gives each agent an isolated git worktree. Is this the right solution for orch-go?

---

## Prior Art: We Already Tried This (Feb 2026)

**Critical finding:** orch-go implemented worktree-per-agent isolation in the Feb 2026 entropy spiral period. It was **reverted as P0** on Feb 12, 2026 (`og-debug-p0-disable-worktree-12feb-1c88`).

**Why it failed:**
1. `orch complete` couldn't find artifacts on worktree branches — agents committed to their worktree branches but the complete pipeline looked at master
2. Agents got stuck as zombies — the worktree lifecycle was fragile
3. The entire pipeline was frozen until worktrees were disabled
4. **20+ commits** were needed just to make worktrees work (copy artifacts to worktree, fix bd fallbacks, fix phase_complete gate, etc.)

**Commits from that era:**
- `088126c6d` fix: check worktree path for agent artifacts in orch complete
- `6fc11e27d` fix: copy spawn artifacts to worktree so agents find SPAWN_CONTEXT.md
- `dd6a851a1` fix: set beads.DefaultDir before FallbackAddComment for worktree support
- `810e374b6` fix: SPAWN_CONTEXT.md PROJECT_DIR now uses worktree path for headless agents
- `2e77cd0b9` fix: disable worktree-assuming code in orch complete
- `bd21b3538` fix: bd command falls back to nearest .beads project root when worktree cwd is deleted

**From the handoff doc** (`2026-02-13-entropy-spiral-recovery.md`): "Stale worktrees: All removed. No more worktree isolation."

---

## How Claude Code --worktree Works (Feb 2026)

### Mechanics

| Property | Behavior |
|----------|----------|
| **Flag** | `claude --worktree [name]` or `claude -w [name]` |
| **Location** | `<repo>/.claude/worktrees/<name>/` |
| **Branch** | `worktree-<name>` (branched from default remote branch) |
| **Auto-name** | Random if name omitted (e.g., "bright-running-fox") |
| **Cleanup (no changes)** | Worktree + branch automatically deleted on exit |
| **Cleanup (with changes)** | User prompted: keep or remove |
| **Auto-merge** | **NO** — manual merge required |
| **Session resume** | Sessions visible across worktrees in same repo |

### CLI Integration

```bash
# Start in worktree
claude --worktree feature-auth

# With tmux (Claude's --tmux REQUIRES --worktree)
claude --worktree feature-auth --tmux

# Subagent isolation (in Claude API / Task tool)
{ "isolation": "worktree" }
```

### Key Distinction: Claude's --tmux vs orch's --tmux

**Claude CLI's `--tmux`** creates a tmux session *for the worktree* — it requires `--worktree`. This is different from orch's `--tmux` flag which creates a tmux window for process management.

When orch spawns with `--tmux`, it creates a tmux window and runs `cat SPAWN_CONTEXT.md | claude --dangerously-skip-permissions` inside it. Adding `--worktree` to that Claude command would mean the Claude process inside the tmux window operates in an isolated worktree.

---

## Analysis: What Would Change in orch spawn

### Current Spawn Flow (Claude backend)

```
orch spawn → BuildClaudeLaunchCommand() → tmux window → cat CONTEXT.md | claude --dangerously-skip-permissions
```

Agent works directly in the project directory. Commits go to whatever branch is checked out (usually master).

### Proposed Spawn Flow with --worktree

```
orch spawn → BuildClaudeLaunchCommand() → tmux window → cat CONTEXT.md | claude --dangerously-skip-permissions -w <workspace-name>
```

Agent gets an isolated working copy at `.claude/worktrees/<name>/`. Commits go to branch `worktree-<name>`.

### Changes Needed in BuildClaudeLaunchCommand

**Minimal change** — add `-w <name>` to the claude command:

```go
// In BuildClaudeLaunchCommand, add worktree flag
worktreeFlag := ""
if worktreeName != "" {
    worktreeFlag = fmt.Sprintf(" -w %s", worktreeName)
}
return fmt.Sprintf("...cat %q | claude --dangerously-skip-permissions%s%s%s",
    contextPath, worktreeFlag, mcpFlag, disallowFlag)
```

**Name convention:** Use the workspace name (e.g., `og-feat-add-auth-27feb-abc1`) which is already unique per agent.

### Changes Needed in SPAWN_CONTEXT.md / Context

- `PROJECT_DIR` would point to the worktree path, not the original project dir
- `.beads/` directory is NOT in the worktree (it's in the main repo) — agents need `BEADS_DIR` env var or the `bd` CLI needs to find it
- `.orch/workspace/` is NOT in the worktree — agent manifest files need special handling

---

## Analysis: Completion/Verification Pipeline Impact

This is where the prior attempt broke down, and the hardest part of the integration.

### Problem 1: Agent commits are on a branch, not master

**Current:** `orch complete` checks `git diff HEAD~5..HEAD` on master to find agent's work.

**With worktrees:** Agent's commits are on `worktree-<name>` branch. The complete pipeline needs to:
1. Know which branch the agent worked on
2. Check that branch for Phase: Complete evidence
3. Merge/cherry-pick changes back to master
4. Clean up the worktree and branch

**Implication:** The `AgentManifest` struct needs a `WorktreeBranch` field. The complete pipeline needs a "merge phase" before archival.

### Problem 2: beads integration

**Current:** Agents run `bd comment <id> "Phase: Complete"` which writes to `.beads/issues.jsonl` in the working tree.

**With worktrees:** `.beads/` is NOT in the worktree (worktrees branch from remote HEAD, which may not have the latest beads state). Two options:
- **Option A:** Set `BEADS_DIR` to point to the main repo's `.beads/` (already done for cross-repo spawns)
- **Option B:** Let beads operate in the worktree and sync back

Option A is simpler and already has precedent in the codebase.

### Problem 3: Verification gates that use git

These gates all assume they're checking the main working tree:

| Gate | Current Behavior | Impact |
|------|-----------------|--------|
| `git_diff` | Checks HEAD~5..HEAD | Must check worktree branch |
| `build` | Runs `go build` in project dir | Must run in worktree or after merge |
| `test_evidence` | Parses beads comments | No change (beads is separate) |
| `accretion` | Checks file growth in git diff | Must check worktree branch diff |
| `hasGoChangesInRecentCommits` | Checks HEAD~5..HEAD | Must check worktree branch |

### Problem 4: Workspace artifacts

**Current:** Agent workspace at `.orch/workspace/<name>/` contains SPAWN_CONTEXT.md, AGENT_MANIFEST.json, VERIFICATION_SPEC.yaml.

**With worktrees:** The worktree is at `.claude/worktrees/<name>/`. The workspace artifacts are in the main repo at `.orch/workspace/<name>/`. The agent needs both paths to work correctly.

---

## Analysis: Dashboard/Status Visibility

### Current Dashboard

The dashboard (`serve_agents.go`) queries:
- OpenCode sessions (for headless agents)
- Beads issues (for tracking state)
- Tmux windows (for Claude CLI agents)

### With Worktrees

**No fundamental change to dashboard.** Worktrees are a git-level concept that's transparent to:
- Beads (uses `BEADS_DIR` or finds `.beads/` up the directory tree)
- Tmux monitoring (window still exists, agent still runs)
- Session status (reported via beads comments)

**Addition needed:** Display worktree branch name in agent status (useful for knowing which branch has the agent's work).

---

## Risk Assessment

### What's Different This Time vs Feb 2026 Attempt

| Factor | Feb 2026 (custom) | Now (Claude native) |
|--------|-------------------|---------------------|
| **Worktree creation** | orch-go created worktrees via git commands | Claude CLI handles worktree lifecycle |
| **Worktree location** | `.orch/worktrees/` | `.claude/worktrees/` |
| **Branch naming** | Custom | `worktree-<name>` (standardized by Claude) |
| **Cleanup** | orch had to manage | Claude prompts keep/remove on exit |
| **Headless support** | Had to replicate for OpenCode path | Claude CLI only (tmux path) |

**Key advantage:** Claude CLI now owns the worktree lifecycle (create, cleanup, branch). orch-go doesn't need to manage git worktrees directly — it just passes `-w <name>` to the claude command.

**Key remaining risk:** The merge-back problem is identical. Claude doesn't auto-merge. orch complete still needs to handle "agent's work is on a branch, move it to master."

### Mitigation: Squash-Merge on Complete

The simplest completion flow:

```bash
# In orch complete, after verifying Phase: Complete:
git merge --squash worktree-<name>
git commit -m "feat: <summary from beads> (<beads-id>)"
git worktree remove .claude/worktrees/<name>
git branch -d worktree-<name>
```

This collapses all agent commits into one clean commit on master. Benefits:
- Clean git history (no "fix typo" agent commits)
- Easy to revert if needed
- Matches existing `orch complete` flow (verify → close → archive)

---

## Recommendation

### Phase 1: Opt-in flag for Claude backend spawns

Add `--worktree` flag to `orch spawn` that passes `-w <workspace-name>` to the Claude CLI command. Requirements:
- Set `BEADS_DIR` to main repo's `.beads/` (already works for cross-repo)
- Store worktree branch in `AGENT_MANIFEST.json`
- Update `BuildClaudeLaunchCommand` to accept worktree name

**Scope:** ~50 lines of code change in `pkg/spawn/claude.go` and `cmd/orch/spawn_cmd.go`.

### Phase 2: Completion pipeline merge support

Modify `orch complete` to handle worktree agents:
- Detect worktree branch from manifest
- Run verification on the worktree branch
- Squash-merge to master
- Clean up worktree and branch

**Scope:** ~100-150 lines in `cmd/orch/complete_cmd.go` and `pkg/verify/`.

### Phase 3: Default behavior change

Once Phase 1-2 are validated, make `--worktree` the default for Claude backend spawns. All agents get isolation by default.

**Scope:** Config flag, minimal code change.

### NOT Recommended: Phase 0 (big-bang migration)

Do NOT attempt to make worktrees default immediately. The Feb 2026 failure was exactly this — too many moving parts changed at once. The phased approach lets each piece be validated independently.

---

## Open Questions for Orchestrator

1. **bd sync interaction:** If agents are on branches, does `bd sync` still work? It commits beads changes to the current branch — if we use `BEADS_DIR` pointing to main repo, bd should commit to master. Need to verify.

2. **Concurrent merge conflicts:** If 3 agents all modify `cmd/orch/main.go` and we squash-merge, the second merge will conflict. How do we handle this? Options: sequential completion, conflict detection before merge, or reject conflicting completes.

3. **Headless (OpenCode) path:** Worktrees only apply to Claude CLI backend. OpenCode headless agents still work directly on master. Is this acceptable asymmetry, or do we need worktree support for both paths?

4. **`.gitignore` for worktrees:** Claude docs recommend adding `.claude/worktrees/` to `.gitignore`. Should this be done during `orch init`?

---

## Evidence Sources

- Claude CLI help: `claude --help` (worktree flag confirmed)
- Prior failure: `og-debug-p0-disable-worktree-12feb-1c88` SPAWN_CONTEXT.md
- Recovery handoff: `.kb/handoffs/2026-02-13-entropy-spiral-recovery.md`
- Git history: 20+ worktree-related commits in the entropy spiral period
- Current spawn code: `pkg/spawn/claude.go:BuildClaudeLaunchCommand()`
- Current complete code: `cmd/orch/complete_cmd.go:runComplete()`
- Claude Code docs: worktree feature documentation (via web search)
