---
linked_issues:
  - orch-go-1254
---

## Summary (D.E.K.N.)

**Delta:** Consolidated 5 prior probes into a single failure taxonomy. Identified 4 distinct failure modes causing agents to run with stale skills, ranked by frequency and severity. Confirmed the loader can't reach stale `src/` copies, making cleanup a hygiene issue not a correctness issue.

**Evidence:** `skillc deploy` exits 0 on partial failure (no exit code signal). 20 stale SKILL.md files persist in `~/.claude/skills/src/`, plus 22 in `~/.opencode/skill/`. Feature-impl `src/` copy has checksum `047ddb2689b3` (Jan 7) while canonical has `76a3920c0fe9` (Feb 25) — 7 weeks stale.

**Knowledge:** The silent deploy failure is not one bug but a pipeline with 4 independent failure points. Two require skillc code changes, one requires hook fixes, one is operational hygiene.

**Next:** Create 3 issues: (1) skillc exit code fix, (2) hook spawn detection fix, (3) stale copy cleanup.

---

# Investigation: skillc Deploy Silent Failures — Why Agents Run With Stale Skills

**Question:** Where in the skillc deploy pipeline can failures occur silently, causing agents to load stale skill content?

**Started:** 2026-02-25
**Owner:** orch-go-1254
**Phase:** Complete
**Status:** Complete

---

## Failure Taxonomy

Four independent failure modes cause agents to run with stale skills. Ordered by severity.

### Failure 1: Deploy Exits 0 on Partial Failure (CRITICAL)

**Problem:** `skillc deploy` loops over all `.skillc/` directories. When individual skills fail to compile or deploy, the error is printed to stderr and the loop `continue`s. The process exits 0 regardless.

**Evidence (from skillc source, `cmd/skillc/main.go`):**
```go
// Line ~1797: Target deploy failure
outputPath, err := compiler.CompileForDeploy(skillcDir, targetOutputDir, skillcDir)
if err != nil {
    fmt.Fprintf(os.Stderr, "✗ Failed to compile %s: %v\n", skillcDir, err)
    continue  // No exit code change
}

// Line ~1821: Agent deploy failure
if err := deployAgentFile(skillcDir, baseDir, agentDir); err != nil {
    fmt.Fprintf(os.Stderr, "✗ Failed to deploy agent file for %s: %v\n", skillcDir, err)
    // No return, no os.Exit(1)
}
```

**Impact:** Automation calling `skillc deploy` (CI, scripts, manual workflow) sees exit 0 and assumes success. No way to gate on deployment correctness programmatically.

**Frequency:** Every deploy where any skill has a compile error.

---

### Failure 2: Plugin Init-Time Caching (HIGH)

**Problem:** The OpenCode `orchestrator-session.ts` plugin reads `~/.claude/skills/meta/orchestrator/SKILL.md` once at plugin initialization and caches in memory. `skillc deploy` updates the file on disk, but OpenCode server is not restarted. All subsequent sessions use the cached (stale) version.

**Evidence (from prior probe `2026-02-17`):**
- Plugin uses `experimental.chat.system.transform` with cached content
- No file watch, no mtime check, no periodic reload
- Server restart is the only way to pick up new skill content

**Impact:** OpenCode interactive sessions run with stale orchestrator skill until server restart. This is the primary cause of "sessions behaving like generic assistants."

**Frequency:** Every `skillc deploy` that isn't followed by `orch-dashboard restart`.

---

### Failure 3: Cross-Project Injection Blocked (HIGH)

**Problem:** `load-orchestration-context.py` conflates `CLAUDE_CONTEXT=orchestrator` (set by interactive `cc()` launcher) with "was spawned by orch" and exits early, preventing skill injection in non-orch-go projects.

**Evidence (from prior probe `2026-02-25`):**
```python
# is_spawned_agent() checks CLAUDE_CONTEXT, not ORCH_SPAWNED
def is_spawned_agent():
    ctx = os.environ.get('CLAUDE_CONTEXT', '')
    return ctx in ('worker', 'orchestrator', 'meta-orchestrator')
# cc personal sets CLAUDE_CONTEXT=orchestrator → hook exits without injecting
```

**Impact:** Interactive orchestrator sessions in any project except orch-go receive NO skill content.

**Frequency:** Every `cc personal` session outside orch-go.

---

### Failure 4: Stale Copy Accumulation (LOW — hygiene)

**Problem:** `skillc deploy` writes to new canonical paths but never cleans old deployment locations. The orch-go skill loader (`FindSkillPath`) only searches one level deep under `~/.claude/skills/`, so the stale `src/` copies (two levels deep) are NOT loaded. But they create confusion for humans and are discoverable by OpenCode's multi-root skill scanner.

**Current stale inventory:**

| Location | Count | Risk |
|----------|-------|------|
| `~/.claude/skills/src/**` | 20 files | Low (orch-go can't reach) |
| `~/.opencode/skill/**` | 22 files | Medium (OpenCode may discover) |
| `~/.claude/skills/SKILL.md` (root orphan) | 1 file | Low |
| `~/.claude/skills/meta-orchestrator/` (stale duplicate) | 1 file | Medium (orch-go CAN reach) |

**The meta-orchestrator duplicate IS reachable:** `FindSkillPath("meta-orchestrator")` checks `~/.claude/skills/meta-orchestrator/SKILL.md` as a direct path — and it exists (Jan 27, stale). The canonical version is at `~/.claude/skills/meta/meta-orchestrator/SKILL.md` (Feb 25). ReadDir iteration order determines which wins, but since direct path is checked first, **the stale version wins**.

**Impact:** `meta-orchestrator` skill spawns load a 4-week-old version.

---

## Recommended Fixes (Priority Order)

### Fix 1: skillc deploy must exit non-zero on any failure

**Scope:** skillc repo (`~/Documents/personal/skillc`)
**Effort:** Small (track `hasErrors` bool, `os.Exit(1)` after loop)
**Impact:** Enables CI gating, script error handling

```go
// After the deploy loop:
if failCount > 0 {
    fmt.Fprintf(os.Stderr, "\n⚠ %d/%d skills failed to deploy\n", failCount, totalCount)
    os.Exit(1)
}
```

### Fix 2: Add `skillc deploy --verify` post-deploy validation

**Scope:** skillc repo
**Effort:** Medium (new flag, checksum comparison)
**Impact:** Catches stale-on-disk situations before agents load them

The verify step would:
1. For each successfully deployed skill, re-read the target file
2. Compare checksum with the just-compiled version
3. Report mismatches (file wasn't written, wrong file was updated, etc.)

### Fix 3: Fix cross-project injection (ORCH_SPAWNED env var)

**Scope:** orch-go + `~/.orch/hooks/load-orchestration-context.py`
**Effort:** Small (3 changes documented in probe `2026-02-25`)
**Impact:** Fixes interactive orchestrator sessions in all projects

Already fully specified in `2026-02-25-probe-orchestrator-skill-cross-project-injection-failure.md`.

### Fix 4: Delete stale copies (one-time cleanup)

**Scope:** Manual or script
**Effort:** Trivial
**Impact:** Removes confusion, fixes meta-orchestrator stale version

```bash
# Remove stale src/ mirror
rm -rf ~/.claude/skills/src/

# Remove root-level orphan
rm ~/.claude/skills/SKILL.md

# Remove stale meta-orchestrator duplicate (direct path that shadows canonical)
rm -rf ~/.claude/skills/meta-orchestrator/

# Remove legacy OpenCode skill copies (if no longer needed)
rm -rf ~/.opencode/skill/
```

### Fix 5: Add cleanup to `skillc deploy` to prevent re-accumulation

**Scope:** skillc repo
**Effort:** Medium
**Impact:** Prevents stale copy accumulation after future deploys

Before deploying, scan target directory for SKILL.md files with `<!-- AUTO-GENERATED by skillc -->` headers that don't correspond to any source `.skillc/` directory. Report and optionally remove them.

---

## Consolidated Prior Work

This investigation consolidates findings from 5 prior probes:

| Probe | Date | Finding Used |
|-------|------|-------------|
| `orchestrator-skill-injection-path-trace` | 2026-02-17 | 5 injection paths, plugin caching |
| `skillc-pipeline-audit` | 2026-02-18 | Deploy path resolution, load path |
| `orchestrator-skill-cli-staleness-audit` | 2026-02-18 | 13 stale CLI references |
| `orchestrator-skill-cross-project-injection-failure` | 2026-02-25 | CLAUDE_CONTEXT conflation |
| `orchestrator-skill-behavioral-compliance` | 2026-02-24 | Behavioral gap in sessions |

---

## Structured Uncertainty

**Tested:** skillc deploy exit code behavior (from source), orch-go loader path resolution (from code), stale copy reachability (from loader logic + filesystem state).

**Inferred:** OpenCode skill scanner behavior for `~/.opencode/skill/` (not tested, based on prior probe observations). Plugin caching behavior (confirmed in prior probe, not re-tested).

**Not tested:** Whether `orch-dashboard restart` actually triggers plugin reload (assumed from prior probe).
