# Uncategorized Investigations Audit - Quick Reference

**Date:** 2026-02-15  
**Full Investigation:** `.kb/investigations/2026-02-15-audit-uncategorized-investigations-855-archive-vs-cluster.md`

---

## Summary

**855 uncategorized investigations** analyzed with the following recommendations:

| Action | Count | Percentage |
|--------|-------|------------|
| CLUSTER | ~655-705 | 77-82% |
| ARCHIVE | ~150-200 | 17-23% |
| KEEP-UNCATEGORIZED | ~20-30 | 2-3% |

---

## Key Finding

**87% of investigations (741) are from the entropy spiral period (Dec 21, 2025 - Feb 12, 2026), BUT most document features that SURVIVED the rollback.**

Do NOT bulk-archive by date. Classification requires topic-based analysis.

---

## Recommended Clusters (15)

1. **spawn-system** (79) - Spawn modes, backends, headless
2. **cli-commands** (70) - orch complete, spawn, status, review, wait, etc.
3. **agent-lifecycle** (60) - Registry, status, completion, abandonment
4. **dashboard-ui** (50) - Web UI, agent visualization
5. **synthesis-artifacts** (49) - Synthesis protocol, SYNTHESIS.md
6. **verification-gates** (48) - Completion verification, gates
7. **skills-development** (37) - Skill creation, skillc
8. **knowledge-system** (35) - KB, models, probes
9. **daemon-mode** (31) - Autonomous daemon, skill inference
10. **artifact-system** (30) - Artifact types, templates, citation
11. **opencode-integration** (28) - OpenCode client, sessions
12. **system-audits** (25) - Comprehensive audits
13. **beads-integration** (23) - Beads CLI integration
14. **template-system** (20) - CLAUDE.md templates
15. **design-explorations** (50-70) - Architecture designs (MIXED: some to archive)

---

## Archive Categories (~150-200 total)

1. **Debug sessions** (20-30) - Transient debugging from entropy era
2. **Rolled-back features** (30-50) - Features confirmed deleted
3. **One-time validation** (20-30) - "Final sanity check" tasks
4. **Superseded designs** (40-60) - Different approach chosen
5. **Duplicates** (20-30) - Multiple investigations, one canonical
6. **Obsolete infrastructure** (20-30) - Python migration artifacts

---

## Implementation Commands

### Create Cluster Directories

```bash
cd /Users/dylanconlin/Documents/personal/orch-go

mkdir -p .kb/investigations/synthesized/{spawn-system,cli-commands,dashboard-ui,verification-gates,synthesis-artifacts,agent-lifecycle,daemon-mode,opencode-integration,skills-development,beads-integration,knowledge-system,design-explorations,system-audits,artifact-system,template-system}
```

### Create Archive Directories

```bash
mkdir -p .kb/archive/entropy-spiral-2025-2026/{debug-sessions,rolled-back-features,one-time-tasks,superseded-designs,duplicates,obsolete-infrastructure}
```

### Topic Analysis File (for scripting)

Classification data available in:
```
/tmp/inv_analysis.txt
```

Format: `date|topic|filename`

---

## Implementation Approaches

### Option A: Manual (Precise but Slow)
Review each investigation, manually move to cluster or archive

### Option B: Script-Assisted (Fast but Needs Review)
Use `/tmp/inv_analysis.txt` to batch-move by topic, review edge cases

### Option C: Incremental (Safest)
Process one cluster at a time:
1. spawn-system (79 files)
2. cli-commands (70 files)
3. agent-lifecycle (60 files)
...

**Recommended:** Script-assisted with human review of edge cases (design-* and debug-* files)

---

## Files Generated

- `/tmp/uncategorized_investigations.txt` - Full list of 855 files
- `/tmp/inv_analysis.txt` - Classification data (date|topic|filename)
- `.kb/investigations/2026-02-15-audit-uncategorized-investigations-855-archive-vs-cluster.md` - Full investigation

---

## Next Action

Orchestrator decides:
- Implement now vs defer
- Implementation approach (manual/script/incremental)
- Review archive candidates before moving

**No files have been moved - this is recommendations only per scope.**
