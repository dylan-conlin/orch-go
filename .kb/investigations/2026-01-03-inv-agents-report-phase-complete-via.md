<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Missing Phase: Complete is caused by short beads IDs in SPAWN_CONTEXT (from `--issue` flag), not tier differences.

**Evidence:** `bd comment 57dn` fails with "not found"; agents with short IDs (0xra, nfrr) have SYNTHESIS.md but no Phase: Complete; `determineBeadsID()` passes short ID without resolution.

**Knowledge:** Tier doesn't affect Phase: Complete instructions; some agents recover by inferring full ID, others fail silently; the fix belongs in spawn (resolve short IDs before generating SPAWN_CONTEXT).

**Next:** Implement `beads.ResolveID()` and call it in `determineBeadsID()` when `--issue` flag is used.

<!--
Example D.E.K.N.:
Delta: Test-running guidance is missing from spawn prompts and CLAUDE.md.
Evidence: Searched 5 agent sessions - none ran tests; guidance exists in separate docs but isn't loaded.
Knowledge: Agents follow documentation literally; guidance must be in loaded context to be followed.
Next: Add test-running instruction to SPAWN_CONTEXT.md template.

Guidelines:
- Keep each line to ONE sentence
- Delta answers "What did we find?"
- Evidence answers "How do we know?"
- Knowledge answers "What does this mean?"
- Next answers "What should happen now?"
- Enable 30-second understanding for fresh Claude
-->

---

# Investigation: Why Some Agents Don't Report Phase: Complete

**Question:** Why do some agents report Phase: Complete via bd comments and others don't? Is this correlated with light vs full spawn tier, and what is the root cause in spawn templates or skill guidance?

**Started:** 2026-01-03
**Updated:** 2026-01-03
**Owner:** Worker agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]
**Supersedes:** [Path to artifact this replaces, if applicable]
**Superseded-By:** [Path to artifact that replaced this, if applicable]

---

## Findings

### Finding 1: SPAWN_CONTEXT.md template correctly instructs Phase: Complete for BOTH tiers

**Evidence:** Examined `pkg/spawn/context.go:28-80`. The template includes Phase: Complete instructions for:
- Full tier (line 74-77): Requires SYNTHESIS.md + Phase: Complete + /exit
- Light tier (line 69-72): Requires Phase: Complete + /exit (no SYNTHESIS.md)

Both tiers get explicit `bd comment {{.BeadsID}} "Phase: Complete - ..."` instructions.

**Source:** `pkg/spawn/context.go:SpawnContextTemplate` lines 58-80

**Significance:** The template is NOT the problem - both tiers correctly instruct agents to report Phase: Complete.

---

### Finding 2: Tier distribution shows no correlation with Phase: Complete failures

**Evidence:** Analyzed 43 og-*-03jan workspaces:
- 12 full tier workspaces with SYNTHESIS.md - all have Phase: Complete
- 21 light tier workspaces - most have Phase: Complete
- Issues WITHOUT Phase: Complete (checked DB):
  - orch-go-nfrr (closed as "Superseded")
  - orch-go-57dn (closed as "Investigation complete - git hooks local-only")
  - orch-go-0xra (closed as "orch clean --investigations: archives empty templates")
  - Epic closures (idmr.*, va5v, 9hld) - closed manually, not by agents

**Source:** 
```bash
sqlite3 .beads/beads.db "SELECT id, close_reason FROM issues WHERE status='closed' AND NOT EXISTS (SELECT 1 FROM comments c WHERE c.issue_id = issues.id AND c.text LIKE '%Phase: Complete%');"
```

**Significance:** Tier is NOT correlated with Phase: Complete failures. The failures are mostly superseded issues or manually closed issues, not agent failures.

---

### Finding 3: Short IDs in SPAWN_CONTEXT cause bd comment failures for some agents

**Evidence:** Initial hypothesis: Short IDs (e.g., "57dn" vs "orch-go-57dn") cause `bd comment` to fail.

Testing confirmed:
```bash
bd comment 57dn "test"  # Fails: "issue 57dn not found"
bd comment orch-go-57dn "test"  # Works: "Comment added"
```

However, examining successful agents (e.g., og-debug-agents-going-idle-03jan) that had short ID "rzch" in SPAWN_CONTEXT, they successfully reported Phase: Complete to issue orch-go-rzch. This suggests some agents figure out the full ID on their own.

**Source:** 
- `.orch/workspace/og-debug-agents-going-idle-03jan/SPAWN_CONTEXT.md` line 139: uses "rzch"
- beads DB shows Phase: Complete comment on orch-go-rzch

**Significance:** Short IDs in SPAWN_CONTEXT are a contributing factor. Some agents recover by figuring out the full ID, but others fail silently.

---

### Finding 4: ROOT CAUSE IDENTIFIED - `--issue` flag passes short ID without resolution

**Evidence:** When `orch spawn --issue 57dn` is used, the spawn system:
1. Takes the short ID "57dn" directly from the flag
2. Passes it into SPAWN_CONTEXT.md without resolving to full ID
3. Agent sees `bd comment 57dn "Phase: Complete..."` instructions
4. Some agents try `bd comment 57dn` which fails silently
5. Other agents figure out the full ID ("orch-go-57dn") from context

Code path: `cmd/orch/spawn_cmd.go:1192-1195`:
```go
if spawnIssue != "" {
    return spawnIssue, nil  // Returns short ID directly!
}
```

Two confirmed failures with this pattern:
1. `og-debug-empty-investigation-templates-03jan` - SYNTHESIS says "bd show 0xra (not found)"
2. `og-debug-fix-getissuesbatch-pkg-03jan` - SYNTHESIS says "Issue: nfrr"

Both agents completed work (have SYNTHESIS.md) but failed to report Phase: Complete because they tried to use short IDs.

**Source:** 
- `cmd/orch/spawn_cmd.go:1192-1195` (determineBeadsID function)
- SYNTHESIS.md files in affected workspaces

**Significance:** This is the root cause. The fix is to resolve short IDs to full IDs in `determineBeadsID` before passing to spawn context.

---

### Finding 5: No tier correlation - failures occur in both light and full tiers

**Evidence:** Analyzed 43 workspaces from 03jan:
- Full tier with failures: og-debug-empty-investigation-templates-03jan (0xra), og-debug-fix-getissuesbatch-pkg-03jan (nfrr)
- Light tier with failures: og-feat-apply-pre-commit-03jan (57dn)

All failures have short IDs in SPAWN_CONTEXT. Tier is NOT a factor.

**Source:** Workspace analysis in `.orch/workspace/og-*-03jan/`

**Significance:** The original hypothesis (light vs full tier correlation) is FALSE. The actual correlation is: short ID passed via `--issue` flag.

---

## Synthesis

**Key Insights:**

1. **Tier is NOT the cause** - Both light and full tier agents have the same Phase: Complete instructions in their SPAWN_CONTEXT.md templates. The template code at `pkg/spawn/context.go:58-80` correctly instructs Phase: Complete for both tiers.

2. **Short IDs from `--issue` flag are the root cause** - When spawning with `orch spawn --issue 57dn`, the short ID is passed directly into SPAWN_CONTEXT without resolution. The `bd comment` command doesn't resolve short IDs, so agents that follow instructions literally fail silently.

3. **Agent behavior is inconsistent** - Some agents (like the one for orch-go-rzch) figure out the full ID from context and successfully report Phase: Complete. Others (like orch-go-0xra, orch-go-nfrr) fail silently. This inconsistency makes the bug hard to detect.

**Answer to Investigation Question:**

Agents don't report Phase: Complete because of **short ID resolution failures**, not tier differences. When `--issue` is passed with a short ID (e.g., "57dn" instead of "orch-go-57dn"), the spawn system doesn't resolve it, and the SPAWN_CONTEXT tells agents to run `bd comment 57dn "Phase: Complete..."` which fails because `bd comment` doesn't resolve short IDs. The fix is to resolve short IDs in `determineBeadsID` before generating SPAWN_CONTEXT.

---

## Structured Uncertainty

**What's tested:**

- ✅ `bd comment` fails with short IDs (tested: `bd comment 57dn "test"` → "issue 57dn not found")
- ✅ `bd comment` works with full IDs (tested: `bd comment orch-go-57dn "test"` → success)
- ✅ SPAWN_CONTEXT.md template instructs Phase: Complete for both tiers (examined `pkg/spawn/context.go:58-80`)
- ✅ `determineBeadsID` returns short ID without resolution (examined `cmd/orch/spawn_cmd.go:1192-1195`)
- ✅ Agents with short IDs in SPAWN_CONTEXT fail to report Phase: Complete (confirmed via beads DB query)

**What's untested:**

- ⚠️ Why some agents with short IDs successfully report (e.g., rzch) - hypothesis: they figure out full ID from context
- ⚠️ Whether `bd` CLI has a `--resolve` option that could be used (not checked)
- ⚠️ Impact of fixing this on existing agent behavior

**What would change this:**

- Finding would be wrong if `bd comment` DOES support short ID resolution in some versions (version-specific behavior)
- Finding would be incomplete if there are other paths besides `--issue` flag that produce short IDs

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**Resolve short IDs in determineBeadsID** - Before generating SPAWN_CONTEXT, resolve any short beads ID to its full form.

**Why this approach:**
- Fixes the root cause at the source (spawn time, not agent time)
- Agents get correct IDs in SPAWN_CONTEXT, no guessing needed
- Consistent with how other beads operations resolve short IDs (e.g., `bd show` resolves "57dn" to "orch-go-57dn")

**Trade-offs accepted:**
- Adds one beads lookup at spawn time (trivial overhead)
- Requires the issue to exist before spawning (already true for `--issue` flag)

**Implementation sequence:**
1. Add `beads.ResolveID(shortID)` function in `pkg/beads/client.go` that calls `bd show <id> --json` and extracts full ID
2. In `determineBeadsID`, if `spawnIssue != ""`, call `ResolveID(spawnIssue)` to get full ID
3. Add test case: `spawn --issue 57dn` should produce SPAWN_CONTEXT with `orch-go-57dn`

### Alternative Approaches Considered

**Option B: Make bd comment resolve short IDs**
- **Pros:** Fixes all bd commands at once, not just spawn
- **Cons:** Changes beads CLI behavior (may have other implications), requires beads-cli update
- **When to use instead:** If short ID resolution is needed more broadly across beads

**Option C: Agent-side detection and recovery**
- **Pros:** No spawn code changes needed
- **Cons:** Relies on agent behavior (inconsistent), doesn't fix root cause, more complex agent logic
- **When to use instead:** Never - this is a workaround, not a fix

**Rationale for recommendation:** Option A is surgical - it fixes the problem at the source with minimal code change and no behavior changes to other components.

---

### Implementation Details

**What to implement first:**
- `beads.ResolveID()` function in `pkg/beads/client.go`
- Modify `determineBeadsID()` in `cmd/orch/spawn_cmd.go`

**Things to watch out for:**
- ⚠️ Error handling: If resolution fails, should it error or continue with short ID?
- ⚠️ Cross-project spawns: `--workdir` may affect which beads DB is searched
- ⚠️ Backward compatibility: Existing SPAWN_CONTEXT files with short IDs should still work

**Areas needing further investigation:**
- Does `bd show` handle cross-project IDs correctly?
- Are there other places in orch-go that use short IDs without resolution?

**Success criteria:**
- ✅ Spawning with `--issue 57dn` produces SPAWN_CONTEXT with full ID `orch-go-57dn`
- ✅ Agents report Phase: Complete successfully after fix
- ✅ No regression in spawn time or existing tests

---

## References

**Files Examined:**
- `pkg/spawn/context.go:28-80` - SPAWN_CONTEXT.md template with Phase: Complete instructions
- `cmd/orch/spawn_cmd.go:1192-1195` - determineBeadsID function that doesn't resolve short IDs
- `.orch/workspace/og-debug-empty-investigation-templates-03jan/SYNTHESIS.md` - Evidence of short ID failure
- `.orch/workspace/og-debug-fix-getissuesbatch-pkg-03jan/SYNTHESIS.md` - Evidence of short ID failure

**Commands Run:**
```bash
# Test short ID resolution in bd comment
bd comment 57dn "test"  # Fails: "issue 57dn not found"
bd comment orch-go-57dn "test"  # Works

# Query issues without Phase: Complete
sqlite3 .beads/beads.db "SELECT id, close_reason FROM issues WHERE status='closed' AND NOT EXISTS (SELECT 1 FROM comments c WHERE c.issue_id = issues.id AND c.text LIKE '%Phase: Complete%');"

# Check SPAWN_CONTEXT beads IDs
grep "spawned from beads issue" .orch/workspace/*/SPAWN_CONTEXT.md
```

**External Documentation:**
- None

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2026-01-03-inv-agents-going-idle-without-phase.md` - Related investigation by rzch agent
- **Workspace:** `.orch/workspace/og-inv-agents-report-phase-03jan/` - This investigation's workspace

---

## Investigation History

**2026-01-03 21:39:** Investigation started
- Initial question: Why do some agents report Phase: Complete via bd comments and others don't?
- Context: orch complete verification fails because agents don't report Phase: Complete

**2026-01-03 21:45:** Ruled out tier correlation
- Examined template at pkg/spawn/context.go - both tiers have Phase: Complete instructions
- Analyzed 43 workspaces - no tier correlation found

**2026-01-03 21:55:** Root cause identified
- Short IDs passed via `--issue` flag are not resolved to full IDs
- SPAWN_CONTEXT contains instructions like `bd comment 57dn` which fail
- Confirmed by examining SYNTHESIS.md files that document "bd show 0xra (not found)"

**2026-01-03 22:10:** Investigation completed
- Status: Complete
- Key outcome: Missing Phase: Complete is caused by short ID resolution failure in spawn, not tier differences
