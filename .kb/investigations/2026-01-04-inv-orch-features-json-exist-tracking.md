<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** `.orch/features.json` is an orphaned architect artifact - no code reads it, and all its fields can be expressed in beads.

**Evidence:** No Go/TS/Svelte code references features.json; beads has skill:* labels, dependencies, parent-child, description, priority that map to all features.json fields; cross-repo beads databases exist.

**Knowledge:** features.json emerged from architect sessions as a design output but was never integrated into tooling - it's a human-readable backlog that drifted from the active beads-based workflow.

**Next:** Recommend deprecation - migrate remaining 29 todo features to beads issues, then delete features.json.

---

# Investigation: Orch Features Json Exist Tracking

**Question:** Why does `.orch/features.json` exist? What is it tracking that beads doesn't already track? Should it be deprecated in favor of beads?

**Started:** 2026-01-04
**Updated:** 2026-01-04
**Owner:** Investigation agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** N/A
**Supersedes:** N/A
**Superseded-By:** N/A

---

## Findings

### Finding 1: features.json is not read by any code

**Evidence:** Searched all Go, TypeScript, Svelte, and shell files for "features.json" - zero matches in actual codebase (only Playwright node_modules has unrelated reference). No struct definitions parse this file. No CLI commands load it.

**Source:** `grep -r "features\.json" . --include="*.go" --include="*.ts" --include="*.svelte" --include="*.sh"` - only match was `./web/node_modules/playwright-core/types/protocol.d.ts` (unrelated)

**Significance:** This is an orphaned artifact - no tooling integration means it serves purely as human-readable documentation. It has drifted from the actual work tracking system (beads).

---

### Finding 2: All features.json fields map to beads capabilities

**Evidence:** Field-by-field comparison:

| features.json field | beads equivalent |
|---------------------|------------------|
| id | id (e.g., orch-go-7rgz) |
| title | title |
| description | description |
| status (todo/done) | status (open/closed) |
| skill | label skill:feature-impl (confirmed: 87 issues have this) |
| priority | priority (1-4) |
| created | created_at |
| source | Can be in description or label |
| notes | Can be in description |
| depends_on | dependencies with dependency_type |
| repo | Separate .beads/ per repo exists (kb-cli, glass both have .beads/) |
| completed | closed_at / close_reason |

**Source:** `bd help create`, `bd label list-all` (shows 22 unique labels including skill:*), `bd show orch-go-f884 --json` (shows dependencies structure)

**Significance:** There is no field in features.json that cannot be expressed in beads. The only potential difference is features.json being a single cross-repo file, but separate beads databases exist per repo.

---

### Finding 3: features.json is an architect artifact

**Evidence:** Git history shows features.json was created/modified by architect sessions, not by automated tooling:
- First commit: `25ca65ef architect: add feature list with servers/API separation recommendations`
- Recent commits: All `architect:` prefix commits
- 25 total commits since Dec 1, 2025

**Source:** `git log --oneline .orch/features.json | head -20`

**Significance:** features.json emerged organically as architect agents produced design recommendations. It was never intended as a tracking system but evolved into one. The artifact format is stable but disconnected from the actual work dispatch system (beads + daemon).

---

### Finding 4: features.json has stale/duplicate data

**Evidence:** 
- features.json shows 29 todo, 2 done (31 total)
- beads has 15 open issues, 1062 closed (1077 total)
- Many features.json entries likely already exist as beads issues (e.g., feat-005 "Dashboard progressive disclosure" is marked done but may or may not have corresponding beads issue)
- No synchronization between systems

**Source:** `.orch/features.json` meta section, `bd stats`

**Significance:** Maintaining two tracking systems creates confusion about source of truth. The daemon and orchestrator use beads exclusively - features.json is orphaned from the actual workflow.

---

## Synthesis

**Key Insights:**

1. **Orphaned artifact pattern** - features.json is a design-time output that was never integrated into runtime tooling. Architect agents write recommendations here, but no tooling reads them.

2. **Beads is the source of truth** - The daemon spawns from beads issues, orch complete closes beads issues, bd stats shows work status. features.json has no integration.

3. **Cross-repo isn't a unique value** - While features.json is cross-repo (tracks glass, kb-cli work), each of those repos has its own .beads/ database that can track local work. The orchestrator skill already documents using `--workdir` for cross-repo spawning.

**Answer to Investigation Question:**

**Why does features.json exist?** It emerged from architect sessions as a structured output format for design recommendations. When architect agents analyze codebases, they produce feature lists. This artifact accumulated over time.

**What is it tracking that beads doesn't?** Nothing. Every field maps to beads capabilities. The `source` field (linking to investigation origin) could be stored in beads description. The cross-repo tracking is handled by separate per-repo beads databases.

**Should it be deprecated?** Yes. The recommendation is:
1. Migrate remaining 29 todo entries to beads issues (respecting per-repo boundaries)
2. Delete features.json
3. Update architect skill to output beads issues directly instead of features.json

---

## Structured Uncertainty

**What's tested:**

- No code reads features.json (verified: grep across all code files)
- Beads has skill:* labels (verified: `bd label list-all` shows 87 skill:feature-impl issues)
- Beads has dependency tracking (verified: `bd show orch-go-f884 --json` shows dependencies)
- Cross-repo beads exist (verified: ls ~/Documents/personal/{kb-cli,glass}/.beads/)

**What's untested:**

- Whether all 29 todo entries already have corresponding beads issues (would require manual comparison)
- Whether any external tools or humans rely on features.json format (soft dependency)

**What would change this:**

- If tooling was discovered that reads features.json, deprecation would need migration path
- If a use case emerged for cross-repo aggregation that beads doesn't support, features.json might have value

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach: Migrate to Beads + Delete

**Migrate remaining features.json entries to beads issues, then delete the file.**

**Why this approach:**
- Eliminates dual-tracking confusion
- Integrates design outputs into the operational workflow (daemon can spawn)
- Reduces maintenance burden (one system to update)

**Trade-offs accepted:**
- Loses the single-file cross-repo view (acceptable: orchestrator skill handles cross-repo)
- Migration effort for 29 entries (one-time cost)

**Implementation sequence:**
1. Create migration script: `bd create "title" -d "description" -l skill:feature-impl -l source:investigation` for each features.json entry
2. For cross-repo entries (repo: kb-cli, glass), create in those repos' beads
3. Verify all entries migrated, then delete features.json
4. Update architect skill template to output `bd create` commands instead of JSON

### Alternative Approaches Considered

**Option B: Keep features.json as read-only archive**
- **Pros:** No migration work, preserves history
- **Cons:** Continues dual-tracking, architect sessions keep adding to dead file
- **When to use instead:** If migration effort is blocked

**Option C: Integrate features.json into tooling**
- **Pros:** Would make it useful
- **Cons:** Duplicates beads functionality, significant development effort
- **When to use instead:** Never - beads already exists

**Rationale for recommendation:** Option A (migrate + delete) is the only option that eliminates the root problem (orphaned artifact). Options B and C both preserve the dual-tracking issue.

---

### Implementation Details

**What to implement first:**
- Create beads issues for features.json entries that don't already exist in beads
- Use labels to preserve metadata: `skill:*`, `source:investigation`

**Things to watch out for:**
- Check if features.json entries already exist in beads (avoid duplicates)
- Cross-repo entries (repo: kb-cli, glass) need to be created in those repos

**Areas needing further investigation:**
- Whether architect skill template needs updating (out of scope for this investigation)

**Success criteria:**
- All 29 todo entries exist as beads issues (or confirmed duplicate)
- features.json deleted from repo
- No code references to features.json

---

## References

**Files Examined:**
- `.orch/features.json` - The file under investigation (369 lines, 31 features)
- `pkg/verify/visual.go` - Checked for "features" references (found only skill name, not file reference)

**Commands Run:**
```bash
# Check for code references
grep -r "features\.json" . --include="*.go" --include="*.ts" --include="*.svelte" --include="*.sh"

# Check beads capabilities
bd stats
bd label list-all
bd show orch-go-f884 --json
bd help create

# Check git history
git log --oneline .orch/features.json | head -20
```

**External Documentation:**
- None

**Related Artifacts:**
- **Investigation:** N/A
- **Workspace:** N/A

---

## Self-Review

- [x] Real test performed (not code review) - ran grep, bd commands, git log
- [x] Conclusion from evidence (not speculation) - based on actual field comparison and code search
- [x] Question answered - yes: features.json is orphaned, should be deprecated
- [x] File complete - all sections filled

**Self-Review Status:** PASSED

---

## Investigation History

**2026-01-04 13:50:** Investigation started
- Initial question: Why does features.json exist and should it be deprecated?
- Context: Orchestrator observed potential duplication between features.json and beads

**2026-01-04 14:05:** Core finding confirmed
- No code reads features.json - it's an orphaned architect artifact
- All fields map to beads capabilities

**2026-01-04 14:15:** Investigation completed
- Status: Complete
- Key outcome: features.json should be deprecated - migrate 29 entries to beads, then delete
