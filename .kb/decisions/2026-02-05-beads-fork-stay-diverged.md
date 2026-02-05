# Decision: Beads Fork - Stay Diverged from Upstream

**Date:** 2026-02-05
**Status:** Active
**Context:** Beads DB health audit and cleanup

---

## Decision

Maintain Dylan's beads fork (`~/Documents/personal/beads`) as a separate diverged codebase from upstream (`steveyegge/beads`). Do not attempt to merge upstream changes or contribute fixes back.

## Context

The fork diverged significantly:
- **Local:** 35 commits with sandbox/corruption fixes
- **Upstream:** 976 commits ahead, focused on sync/worktrees/Dolt

The fork addresses a specific problem: **agent spawns in Claude Code sandbox cause SQLite WAL corruption** due to rapid daemon restart loops. Upstream users likely run `bd` on host systems without this issue.

## Rationale

### Why stay diverged

1. **Different problem domains** - Fork solves sandbox corruption; upstream focuses on sync, worktrees, Dolt backend
2. **Maintenance burden** - PRs would require rebasing against 976 commits, testing against their codebase
3. **Working solution** - Fork's fixes have prevented corruption since Jan 21, 2026
4. **JSONL-only simplification** - Fork's default removes need for most upstream features

### Why not contribute upstream

1. **Niche use case** - Sandbox corruption affects AI agent workflows, not typical CLI users
2. **Architectural divergence** - Fork made JSONL-only the default, upstream still SQLite-first
3. **Testing complexity** - Would need to verify fixes work in their broader context

## Consequences

### Accepted trade-offs

- Miss upstream improvements (sync, Dolt, new features)
- Must maintain fork independently
- May diverge further over time

### Mitigations

- Document fork's purpose and key commits
- If upstream reports similar issues, point them to our fixes
- Periodically review upstream for critical security fixes

## Key Commits in Fork

| Commit | Fix |
|--------|-----|
| `9953b9cb` | Prevent daemon auto-start in sandbox |
| `4da15127` | Detect Claude Code and Docker environments |
| `98e5c750` | Use JSONL-only mode when sandbox detected |
| `2198ad78` | Prevent rapid restart loops |
| `041af3fa` | Pre-flight fingerprint validation |
| `629441ad` | Make JSONL-only the default storage mode |

## References

- `.kb/models/beads-database-corruption.md` - Root cause analysis
- `.kb/investigations/2026-01-21-inv-investigate-beads-sqlite-database-corruption.md` - Original investigation
- Fork location: `~/Documents/personal/beads`
- Binary symlink: `~/bin/bd` → `~/Documents/personal/beads/build/bd`
