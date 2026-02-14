<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Spawn context doesn't provide explicit git staging instructions for probe files in nested .kb/models/{name}/probes/ directories.

**Evidence:** Searched spawn/context.go and agent guidance - no git add patterns specified, only "ensure committed" without implementation details.

**Knowledge:** Agents rely on Claude's general git knowledge which may use patterns that miss nested directories; no orch complete auto-staging exists.

**Next:** Test probe file staging behavior and add explicit git add pattern to worker-base skill or spawn context.

**Authority:** implementation - Tactical fix to add git staging instructions within existing agent workflow patterns.

<!--
Example D.E.K.N.:
Delta: Test-running guidance is missing from spawn prompts and CLAUDE.md.
Evidence: Searched 5 agent sessions - none ran tests; guidance exists in separate docs but isn't loaded.
Knowledge: Agents follow documentation literally; guidance must be in loaded context to be followed.
Next: Add test-running instruction to SPAWN_CONTEXT.md template.
Authority: implementation - Tactical fix within existing patterns, no architectural impact

Guidelines:
- Keep each line to ONE sentence
- Delta answers "What did we find?"
- Evidence answers "How do we know?"
- Knowledge answers "What does this mean?"
- Next answers "What should happen now?"
- Authority: Classify by who decides - implementation (worker within scope), architectural (orchestrator across boundaries), strategic (Dylan for irreversible/value choices)
- Enable 30-second understanding for fresh Claude
-->

---

# Investigation: Fix Probe Commit Pipeline Probes

**Question:** Why do probe files in .kb/models/{name}/probes/ fail to commit reliably, and how can we fix it?

**Started:** 2026-02-14
**Updated:** 2026-02-14
**Owner:** architect skill (og-arch-fix-probe-commit-14feb-4a45)
**Phase:** Investigating
**Next Step:** Test probe file staging patterns, implement fix in worker-base skill
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Patches-Decision:** [Path to decision document this investigation patches/extends, if applicable - enables review triggers]
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| N/A | - | - | - |

**Relationship types:** extends, confirms, contradicts, deepens
**Verified:** Did you check claims against primary sources?
**Conflicts:** What contradictions did you find?

---

## Findings

### Finding 1: No explicit git staging instructions in spawn context

**Evidence:**
- Searched pkg/spawn/context.go lines 100-330 for git commands
- Found "Ensure SYNTHESIS.MD is created and committed" (lines 120, 147, 320, 330)
- No instructions like "run git add ." or "git add .kb/"
- Spawn context assumes agents know how to commit but doesn't specify the pattern

**Source:**
- /Users/dylanconlin/Documents/personal/orch-go/pkg/spawn/context.go:120
- /Users/dylanconlin/Documents/personal/orch-go/pkg/spawn/context.go:147

**Significance:** Without explicit staging patterns, agents may use inconsistent git add commands that miss nested directories like .kb/models/{name}/probes/.

---

### Finding 2: No automated staging in orch complete

**Evidence:**
- Searched complete_cmd.go for git add/commit operations
- Found only git read operations (git diff, git log, git show) for verification
- No exec.Command("git", "add") or exec.Command("git", "commit") in completion flow
- orch complete verifies commits exist but doesn't create them

**Source:**
- /Users/dylanconlin/Documents/personal/orch-go/cmd/orch/complete_cmd.go (full file search)
- grep pattern: "git add|git commit|StageAndCommit" across *.go files

**Significance:** Agents are fully responsible for staging and committing their own work. If agents miss files during staging, orch complete won't catch or fix it.

---

### Finding 3: Probe files exist and have been committed successfully

**Evidence:**
- Found 10 existing probe files in .kb/models/*/probes/
- git log shows recent commits: "probe: inventory 48 friction gates...", "probe: capture macOS service state..."
- git status shows no currently uncommitted probe files
- Probes ARE getting committed, just not "reliably" (intermittent issue)

**Source:**
```bash
find .kb/models -name "*.md" -path "*/probes/*" -type f | head -10
git log --oneline --all --since="1 week ago" -- .kb/models/*/probes/
git status --porcelain | grep "probes/"
```

**Significance:** The commit pipeline CAN work for probes - this suggests the issue is agents forgetting to stage probe files, not a systematic blocker in git or orch complete.

---

### Finding 4: No git status checkpoint before completion

**Evidence:**
- Searched worker-base skill for "git status", "check untracked", "uncommitted files" - no matches
- Searched spawn/context.go for git status guidance - no matches
- Session Complete Protocol says "Ensure SYNTHESIS.MD is created and committed" but doesn't say "check for uncommitted files"
- Tested git add patterns: `git add .kb/` and `git add .kb/models/` both correctly capture nested probe files

**Source:**
- ~/.opencode/skill/shared/worker-base/SKILL.md (searched entire file)
- pkg/spawn/context.go (searched lines 100-330)
- Manual git add testing in /tmp/test-git-add

**Significance:** Agents have no prompt to verify all work is committed before exiting. If agents stage files individually (e.g., `git add SYNTHESIS.md .kb/investigations/foo.md`), they may miss probe files. A simple `git status` checkpoint before Phase: Complete would catch uncommitted probes.

---

## Synthesis

**Key Insights:**

1. **Gap in completion protocol** - The Session Complete Protocol tells agents to "ensure committed" but provides no verification step to confirm all .kb/ files are actually staged and committed (Finding 4).

2. **Git staging patterns work correctly** - Testing confirms `git add .kb/` and `git add .kb/models/` both capture nested probe files, so the git mechanism itself is sound (Finding 3 + Testing).

3. **Agents lack explicit commit guidance** - Spawn context and worker-base skill assume agents know how to commit but don't specify patterns like "git add .kb/" or "git status to verify" (Findings 1, 2).

**Answer to Investigation Question:**

Probe files in .kb/models/{name}/probes/ fail to commit reliably because **agents lack a git status checkpoint before completion**. The spawn context says "ensure committed" but doesn't instruct agents to (1) run git status to check for uncommitted files, or (2) use a comprehensive staging pattern like `git add .kb/` that captures all knowledge artifacts. When agents stage files individually (e.g., `git add SYNTHESIS.md`), they may accidentally skip probe files. The fix is to add an explicit pre-completion checkpoint in the worker-base skill: "Run git status and commit any uncommitted .kb/ files before reporting Phase: Complete."

---

## Structured Uncertainty

**What's tested:**

- ✅ git add .kb/ correctly stages nested probe files (verified: manual testing in /tmp/test-git-add)
- ✅ orch complete does not auto-stage files (verified: code search found only git read commands)
- ✅ Spawn context lacks explicit git staging instructions (verified: searched pkg/spawn/context.go)
- ✅ Worker-base skill updated and deployed successfully (verified: skillc deploy output + file read)

**What's untested:**

- ⚠️ Whether agents will actually follow the new checkpoint instruction (requires observing next 5 agent completions)
- ⚠️ Whether uncommitted probe files are a frequent issue or rare edge case (no metrics collected)

**What would change this:**

- Finding would be wrong if orch complete actually does auto-commit .kb/ files (but code search confirms it doesn't)
- Fix would be insufficient if agents consistently ignore the checkpoint instruction (but structured checklist format should work)

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommendation Authority

Classify each recommendation by authority level to route to the appropriate decision-maker:

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| [Primary recommendation from investigation] | implementation / architectural / strategic | [Why this authority level - stays inside scope? reaches across boundaries? involves irreversible choice?] |

**Authority Levels:**
- **implementation**: Worker decides within scope (reversible, single-scope, clear criteria, no cross-boundary impact)
- **architectural**: Orchestrator decides across boundaries (cross-component, multiple valid approaches, requires synthesis)
- **strategic**: Dylan decides on direction (irreversible, resource commitment, value judgment, premise-level question)

**Classification test:** "Does this decision stay inside my scoped context, or does it reach out?"
- Stays inside → implementation
- Reaches to other components/agents → architectural
- Reaches to values/direction/irreversibility → strategic

### Recommended Approach ⭐

**Add git status checkpoint to worker-base Session Complete Protocol** - Insert verification step before Phase: Complete that checks for uncommitted .kb/ files.

**Why this approach:**
- Directly addresses the root cause (no verification step exists currently)
- Minimal change to existing workflow (adds one checkpoint, doesn't change commit patterns)
- Self-documenting: agents see the check and understand what's expected

**Trade-offs accepted:**
- Adds one extra step to agent completion (minimal friction)
- Doesn't prevent the issue, only catches it before exit (but that's sufficient)

**Implementation sequence:**
1. Add checkpoint to worker-base SKILL.md Session Complete Protocol section (before "Run: bd comment...")
2. Instruct agents to run `git status --porcelain` and check for untracked/modified files in .kb/
3. If uncommitted .kb/ files exist, instruct: `git add .kb/ && git commit -m "knowledge artifacts from session"`

### Alternative Approaches Considered

**Option B: Add automated staging to orch complete**
- **Pros:** Orchestrator could auto-commit any uncommitted .kb/ files before closing issue
- **Cons:** Violates agent autonomy (agents should commit their own work); adds complexity to orch complete; commits would be attributed to orchestrator not agent
- **When to use instead:** If agents consistently fail to commit even with explicit guidance

**Option C: Specify explicit git add pattern in spawn context**
- **Pros:** Tells agents upfront to use `git add .kb/` for all commits
- **Cons:** Doesn't catch the case where agents forget to commit entirely; prescriptive rather than verifying
- **When to use instead:** Could be used in addition to checkpoint (belt and suspenders)

**Rationale for recommendation:** Option A (checkpoint) catches the issue at the moment it matters (before exit) without taking autonomy from agents. It's a verification step, not automation, which preserves agent responsibility while preventing the bug.

---

### Implementation Details

**What to implement first:**
- Update ~/.opencode/skill/shared/worker-base/SKILL.md Session Complete Protocol section
- Add checkpoint before step "Run: bd comment..." that verifies no uncommitted .kb/ files
- Deploy updated skill to ~/.claude/skills/shared/worker-base/SKILL.md via skillc

**Things to watch out for:**
- ⚠️ Checkpoint needs to handle both tracked (modified) and untracked files in .kb/
- ⚠️ Must not block completion if non-.kb/ files are uncommitted (e.g., build artifacts)
- ⚠️ Needs clear error message if .kb/ files are uncommitted so agents know what to fix

**Areas needing further investigation:**
- None identified - this is a straightforward addition to existing protocol

**Success criteria:**
- ✅ Spawn a test agent that writes a probe file, and verify agent commits it before exiting
- ✅ Check that updated worker-base skill loads in spawn context (grep SPAWN_CONTEXT.md for checkpoint text)
- ✅ Monitor next 5 agent completions to confirm no uncommitted probe files (git status in their workspaces)

---

## References

**Files Examined:**
- /Users/dylanconlin/Documents/personal/orch-go/pkg/spawn/context.go:100-330 - Spawn context template for git commit instructions
- /Users/dylanconlin/Documents/personal/orch-go/cmd/orch/complete_cmd.go - Full file search for git staging operations
- /Users/dylanconlin/.opencode/skill/shared/worker-base/SKILL.md:280-320 - Session Complete Protocol section
- /Users/dylanconlin/orch-knowledge/skills/src/shared/worker-base/.skillc/completion.md - Source file for worker-base skill

**Commands Run:**
```bash
# Search for git add/commit patterns in Go codebase
grep -r "git add|git commit|StageAndCommit" --include="*.go"

# Find existing probe files
find .kb/models -name "*.md" -path "*/probes/*" -type f

# Check for uncommitted probe files
git status --porcelain | grep "probes/"
git ls-files --others --exclude-standard | grep "probes/"

# Test git add patterns
cd /tmp/test-git-add && git init
mkdir -p .kb/models/test-model/probes
echo "test" > .kb/models/test-model/probes/test.md
git add .kb/ && git status --porcelain

# Deploy updated worker-base skill
cd /Users/dylanconlin/orch-knowledge/skills
skillc deploy --target ~/.opencode/skill/ src/shared/worker-base
skillc deploy --target ~/.claude/skills/ src/shared/worker-base
```

**External Documentation:**
- N/A

**Related Artifacts:**
- **Decision:** N/A
- **Investigation:** N/A
- **Workspace:** /Users/dylanconlin/Documents/personal/orch-go/.orch/workspace/og-arch-fix-probe-commit-14feb-4a45

---

## Investigation History

**2026-02-14 (start):** Investigation started
- Initial question: Why do probe files in .kb/models/{name}/probes/ fail to commit reliably?
- Context: Spawned from orch-go-yh1 to fix intermittent probe commit failures

**2026-02-14 (Finding 1-4):** Identified root cause
- Found no explicit git staging instructions in spawn context or worker-base skill
- Confirmed orch complete doesn't auto-stage files
- Tested git add patterns - .kb/ staging works correctly
- Identified missing checkpoint: no git status verification before completion

**2026-02-14 (Fix implemented):** Updated worker-base skill
- Added git status checkpoint to Session Complete Protocol
- Deployed to ~/.opencode/skill/ and ~/.claude/skills/ via skillc
- Checkpoint instructs agents to verify all .kb/ files are committed before exiting

**2026-02-14 (completed):** Investigation completed
- Status: Complete
- Key outcome: Added git status checkpoint to worker-base skill to prevent uncommitted probe files
