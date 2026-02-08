<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Cross-project agents have wrong `project_dir` because workspace cache parsing doesn't stop after finding the beads ID, allowing later template placeholders (`<beads-id>`) to overwrite the correct value.

**Evidence:** API returns `project_dir: "/Users/.../orch-go"` for `specs-platform-36` agent instead of correct `specs-platform` path. Test script confirmed parsing extracts `specs-platform-36` correctly but then continues iterating and overwrites with `<beads-id>` from template examples in SPAWN_CONTEXT.md.

**Knowledge:** SPAWN_CONTEXT.md files contain template examples with `<beads-id>` placeholder that match the `bd comment` parsing pattern. The "spawned from beads issue" line is authoritative and should stop parsing.

**Next:** Implement fix: add `break` after extracting beads ID from "spawned from beads issue" line.

**Promote to Decision:** recommend-no (bug fix, not architectural decision)

---

# Investigation: Dashboard Follow Mode Doesn't Show Cross-Project Agents

**Question:** Why don't daemon-spawned cross-project agents appear in dashboard when following their target project's context?

**Started:** 2026-01-29
**Updated:** 2026-01-29
**Owner:** Worker agent (orch-go-21024)
**Phase:** Synthesizing
**Next Step:** Implement fix
**Status:** In Progress

---

## Findings

### Finding 1: Cross-project agents have null or wrong project_dir

**Evidence:** API output for `specs-platform-36` agent:
```json
{
  "beads_id": "specs-platform-36",
  "project": "specs-platform",
  "project_dir": "/Users/dylanconlin/Documents/personal/orch-go",
  "status": "dead"
}
```

The `project` field is correct ("specs-platform") because it's extracted from the beads ID using `extractProjectFromBeadsID()`. But `project_dir` is wrong (shows orchestrator's cwd instead of target project).

**Source:** `curl -sk "https://localhost:3348/api/agents?since=1h" | jq`

**Significance:** The dashboard project filter uses `matchAgentProject(agent.Project, agent.ProjectDir, filters)`. When `project_dir` is wrong, cross-project agents don't match the filter for their target project.

---

### Finding 2: Workspace cache parsing doesn't stop after finding beads ID

**Evidence:** The workspace cache build code at `serve_agents_cache.go:463-488`:
```go
for _, line := range strings.Split(contentStr, "\n") {
    // Extract beads ID from "spawned from beads issue: **xxx**" or "bd comment xxx"
    if strings.Contains(strings.ToLower(line), "spawned from beads issue:") {
        // ... extract beads ID ...
        beadsID = rest[:endIdx]
        // NO BREAK HERE!
    } else if strings.HasPrefix(lineTrimmed, "bd comment ") {
        parts := strings.Fields(lineTrimmed)
        if len(parts) >= 3 {
            beadsID = parts[2]  // OVERWRITES correct beads ID!
        }
    }
    // ...
}
```

The loop continues after finding the "spawned from beads issue" line, allowing later `bd comment` lines to overwrite the beads ID.

**Source:** `cmd/orch/serve_agents_cache.go:463-488`

**Significance:** This is the root cause. Later template examples in SPAWN_CONTEXT.md match the `bd comment` pattern and overwrite the correct beads ID.

---

### Finding 3: SPAWN_CONTEXT.md contains template placeholders that match parsing pattern

**Evidence:** The SPAWN_CONTEXT.md file for `specs-platform-36` contains:
- Line 191: `You were spawned from beads issue: **specs-platform-36**` (correct)
- Lines 551+: `bd comment <beads-id> "Phase: X - details"` (template placeholder)

Test script output:
```
Final beadsID: '<beads-id>'
Final projectDir: '/Users/.../specs-platform'
```

The final cached beads ID is `<beads-id>` (template placeholder) instead of `specs-platform-36`.

**Source:** Manual analysis of `/Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/specs-platform/.orch/workspace/sp-feat-high-replace-deprecated-29jan-6fb6/SPAWN_CONTEXT.md`

**Significance:** The skill templates include example `bd comment` lines with `<beads-id>` placeholder. When the parsing doesn't stop early, these template lines corrupt the cached data.

---

### Finding 4: Completed workspaces don't set Project field

**Evidence:** The completed workspace loop at `serve_agents.go:691-800` extracts `BeadsID` and `Skill` but NOT `Project`:
```go
agent.BeadsID = extractBeadsIDFromWorkspace(workspacePath)
agent.Skill = extractSkillFromTitle(entry.Name())
// NOTE: We intentionally DON'T extract PROJECT_DIR or fetch beads data...
```

Missing line: `agent.Project = extractProjectFromBeadsID(agent.BeadsID)`

**Source:** `cmd/orch/serve_agents.go:785-796`

**Significance:** Secondary bug - completed workspaces have `project: null` in API response, preventing project filtering.

---

## Synthesis

**Key Insights:**

1. **Root cause is parsing order bug** - The workspace cache parsing finds the correct beads ID but doesn't stop, allowing template examples to overwrite it.

2. **SPAWN_CONTEXT.md structure creates the collision** - The "spawned from beads issue" line is near the middle (line 191) while skill templates with `<beads-id>` examples are near the end (lines 500+).

3. **Two related bugs** - (1) Workspace cache parsing overwrites beads ID, (2) Completed workspaces don't extract Project field.

**Answer to Investigation Question:**

Cross-project agents don't appear when following their target project because:

1. The workspace cache parsing extracts the correct beads ID (`specs-platform-36`) from "spawned from beads issue" line
2. But parsing continues through the entire file
3. Later `bd comment <beads-id>` template lines match the parsing pattern
4. The beads ID gets overwritten with `<beads-id>` (a template placeholder)
5. When looking up `wsCache.lookupProjectDir("specs-platform-36")`, no match is found because the cache key is `<beads-id>`
6. The agent's `project_dir` remains the orchestrator's cwd (set from OpenCode session)
7. Dashboard filter excludes the agent because `project_dir` doesn't match the target project

---

## Structured Uncertainty

**What's tested:**

- ✅ API returns wrong project_dir for cross-project agents (verified: curl command)
- ✅ Workspace has correct PROJECT_DIR in SPAWN_CONTEXT.md (verified: grep command)
- ✅ Parsing extracts correct beads ID but then overwrites it (verified: test script)
- ✅ Template placeholders `<beads-id>` cause the overwrite (verified: test script)

**What's untested:**

- ⚠️ Fix effectiveness (will implement and test)
- ⚠️ Performance impact of adding break statement (should be minimal - fewer iterations)

**What would change this:**

- If SPAWN_CONTEXT.md format changed to not include template examples, the bug wouldn't manifest
- If "spawned from beads issue" line moved to the end of the file, it would work (but fragile)

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation.

### Recommended Approach ⭐

**Add break after extracting beads ID from authoritative line** - Stop parsing after finding "spawned from beads issue" line since it's the authoritative source.

**Why this approach:**
- Minimal change with maximum impact
- "spawned from beads issue" IS the authoritative line - no need to continue
- Prevents any future template patterns from causing similar bugs

**Trade-offs accepted:**
- Won't extract beads ID from `bd comment` lines as fallback if "spawned from beads issue" is missing
- This is acceptable because all spawn contexts should have the authoritative line

**Implementation sequence:**
1. Add `break` after line 473 in serve_agents_cache.go
2. Also add `agent.Project = extractProjectFromBeadsID(agent.BeadsID)` in completed workspace loop
3. Add test case to verify parsing stops at correct line

### Alternative Approaches Considered

**Option B: Validate beads ID format before accepting**
- **Pros:** Would catch any invalid patterns
- **Cons:** More complex, regex needed, doesn't address root cause
- **When to use instead:** If templates must use valid-looking beads IDs

**Option C: Restructure SPAWN_CONTEXT.md to remove template examples**
- **Pros:** Would eliminate collision
- **Cons:** Changes skill templates, may impact agent guidance
- **When to use instead:** If templates are causing other issues

**Rationale for recommendation:** Option A is simplest and addresses root cause. The "spawned from beads issue" line is explicitly the authoritative source - the comment in the code even says so.

---

### Implementation Details

**What to implement first:**
- Add `break` after line 473 (beads ID extraction)
- Add Project extraction for completed workspaces

**Things to watch out for:**
- ⚠️ Ensure PROJECT_DIR is still extracted even with early break (it's parsed AFTER beads ID)
- ⚠️ Test with workspaces that have "spawned from beads issue" line at different positions

**Success criteria:**
- ✅ API returns correct `project_dir` for cross-project agents
- ✅ Dashboard shows cross-project agents when following their target project
- ✅ Completed workspaces have correct `project` field

---

## References

**Files Examined:**
- `cmd/orch/serve_agents_cache.go` - Workspace cache building with parsing bug
- `cmd/orch/serve_agents.go` - Agent API endpoint, completed workspace handling
- `cmd/orch/serve_filter.go` - Project matching logic
- `cmd/orch/serve_context.go` - Follow orchestrator context API
- `pkg/tmux/follower.go` - Multi-project config

**Commands Run:**
```bash
# Check API output for cross-project agents
curl -sk "https://localhost:3348/api/agents?since=1h" | jq

# Find workspace for specs-platform-36
grep -r "specs-platform-36" /path/to/specs-platform/.orch/workspace/*/SPAWN_CONTEXT.md

# Test parsing behavior
go run /tmp/test_bug.go
```

**Related Artifacts:**
- **Model:** `.kb/models/cross-project-visibility.md` - Documents cross-project agent visibility architecture
- **Model:** `.kb/models/follow-orchestrator-mechanism.md` - Documents dashboard follow mode

---

## Investigation History

**[2026-01-29 18:00]:** Investigation started
- Initial question: Why don't daemon-spawned cross-project agents appear in dashboard?
- Context: Bug report indicated agents only visible when following orch-go, not target project

**[2026-01-29 18:15]:** Found wrong project_dir in API output
- specs-platform-36 has project_dir = orch-go instead of specs-platform

**[2026-01-29 18:30]:** Root cause identified
- Workspace cache parsing doesn't stop after finding beads ID
- Template placeholders `<beads-id>` overwrite correct value

**[2026-01-29 18:45]:** Synthesis complete, ready for implementation
