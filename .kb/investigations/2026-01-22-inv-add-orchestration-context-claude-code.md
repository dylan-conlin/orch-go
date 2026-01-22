<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Created `~/.claude/statusline.sh` that displays orchestration context (beads ID, skill, tier, spawn mode) alongside standard Claude Code status info.

**Evidence:** Tested script with mock JSON input - outputs `Opus | $1.23 | 46% | orch-go-ntpke:feature-impl:lite:docker` for orchestrated sessions and `Sonnet | $0.50 | 22%` for non-orchestrated sessions.

**Knowledge:** Claude Code status line receives JSON via stdin with `workspace.project_dir` field that can be used to locate the agent's workspace and its `AGENT_MANIFEST.json`.

**Next:** Commit the statusline.sh script - no further action needed.

**Promote to Decision:** recommend-no - tactical implementation, not architectural decision

---

# Investigation: Add Orchestration Context Claude Code

**Question:** How can we display orchestration context (beads issue, skill, tier) in Claude Code's status line?

**Started:** 2026-01-22
**Updated:** 2026-01-22
**Owner:** worker agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Claude Code status line receives JSON via stdin

**Evidence:** The status line command receives JSON data including:
- `model.display_name`: Current model name
- `workspace.project_dir`: Project directory path
- `cost.total_cost_usd`: Session cost
- `context_window.used_percentage`: Context window usage

**Source:** ~/.claude/cache/changelog.md, agent exploration research

**Significance:** The `project_dir` field can be used to locate the orchestration workspace and extract context.

---

### Finding 2: Agent manifests contain orchestration metadata

**Evidence:** Each spawned agent creates `AGENT_MANIFEST.json` in their workspace with:
```json
{
  "workspace_name": "og-feat-add-orchestration-context-22jan-3cd5",
  "skill": "feature-impl",
  "beads_id": "orch-go-ntpke",
  "project_dir": "/Users/dylanconlin/Documents/personal/orch-go",
  "spawn_time": "2026-01-22T08:33:08-08:00",
  "tier": "light",
  "spawn_mode": "docker"
}
```

**Source:** `/Users/dylanconlin/Documents/personal/orch-go/.orch/workspace/og-feat-add-orchestration-context-22jan-3cd5/AGENT_MANIFEST.json`

**Significance:** All orchestration context needed for display is available in a structured JSON file.

---

### Finding 3: Status line settings already configured in settings.json

**Evidence:** `~/.claude/settings.json` already contains:
```json
{
  "statusLine": {
    "type": "command",
    "command": "~/.claude/statusline.sh",
    "padding": 0
  }
}
```

**Source:** `~/.claude/settings.json:268-272`

**Significance:** The infrastructure is already in place - only need to create the script.

---

## Synthesis

**Key Insights:**

1. **Matching via project_dir** - The status line's `workspace.project_dir` can be matched against manifests' `project_dir` to find the active workspace for the current project.

2. **Most recent workspace wins** - When multiple workspaces exist for a project, comparing `spawn_time` identifies the currently active one.

3. **Graceful fallback** - When no orchestration context exists (non-spawned sessions), the script simply shows basic model/cost/context info.

**Answer to Investigation Question:**

Created `~/.claude/statusline.sh` that:
1. Reads JSON from stdin
2. Extracts project_dir
3. Finds the most recently spawned workspace's AGENT_MANIFEST.json
4. Displays orchestration info (beads_id:skill:tier:spawn_mode) when available

---

## Structured Uncertainty

**What's tested:**

- ✅ Script parses JSON input correctly (verified: echo with mock data)
- ✅ Script finds orchestration context from current project (verified: tested with orch-go project)
- ✅ Script falls back gracefully when no orchestration context exists (verified: tested with /tmp as project_dir)

**What's untested:**

- ⚠️ Performance impact of scanning workspace directories (not benchmarked)
- ⚠️ Behavior with very large number of workspaces (not tested)

**What would change this:**

- If status line is called with very high frequency, jq parsing overhead might be noticeable
- If workspace directory contains hundreds of manifests, scan time might become visible

---

## References

**Files Examined:**
- `~/.claude/settings.json` - Status line configuration
- `~/.claude/cache/changelog.md` - Status line feature documentation
- `.orch/workspace/*/AGENT_MANIFEST.json` - Agent manifest structure

**Commands Run:**
```bash
# Test with orchestration context
echo '{"model":{"display_name":"Opus"},"workspace":{"project_dir":"/Users/dylanconlin/Documents/personal/orch-go"},"cost":{"total_cost_usd":1.23},"context_window":{"used_percentage":45.7}}' | ~/.claude/statusline.sh
# Output: Opus | $1.23 | 46% | orch-go-ntpke:feature-impl:lite:docker

# Test without orchestration context
echo '{"model":{"display_name":"Sonnet"},"workspace":{"project_dir":"/tmp"},"cost":{"total_cost_usd":0.50},"context_window":{"used_percentage":22.1}}' | ~/.claude/statusline.sh
# Output: Sonnet | $0.50 | 22%
```

---

## Investigation History

**2026-01-22:** Investigation started
- Initial question: How to add orchestration context to Claude Code status line
- Context: Spawned from beads issue orch-go-ntpke

**2026-01-22:** Research complete
- Found status line receives JSON via stdin with project_dir
- Found AGENT_MANIFEST.json structure contains all needed fields

**2026-01-22:** Implementation complete
- Created ~/.claude/statusline.sh
- Tested with mock data
- Key outcome: Status line now shows orchestration context when available
