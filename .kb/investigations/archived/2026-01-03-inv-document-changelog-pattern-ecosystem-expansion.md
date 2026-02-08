<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Documented the changelog system architecture, semantic parsing taxonomy, ecosystem expansion process, and integration points.

**Evidence:** Created docs/changelog-system.md with comprehensive documentation; updated README.md and ECOSYSTEM.md with quick references.

**Knowledge:** The changelog system has three integration points: CLI, API, and orch complete; repo expansion requires editing ExpandedOrchEcosystemRepos map.

**Next:** Close - documentation complete and committed.

---

# Investigation: Document Changelog Pattern Ecosystem Expansion

**Question:** How does the changelog system work and how can new repos be added to monitoring?

**Started:** 2026-01-03
**Updated:** 2026-01-03
**Owner:** Worker agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Changelog System Architecture

**Evidence:** The changelog system has three entry points:
1. CLI command (`orch changelog`) with `--days`, `--project`, and `--json` flags
2. API endpoint (`GET /api/changelog`) with `?days` and `?project` query params
3. orch complete integration (`detectNotableChangelogEntries`) for surfacing notable changes

**Source:** 
- `cmd/orch/changelog.go:25-45` - CLI command definition
- `cmd/orch/serve.go:252-253` - API endpoint registration
- `cmd/orch/main.go:3508-3572` - orch complete integration

**Significance:** Three distinct integration points serve different use cases: CLI for ad-hoc queries, API for dashboard integration, orch complete for workflow integration.

---

### Finding 2: Ecosystem Repo Registry

**Evidence:** Repos are defined in a map in `pkg/spawn/ecosystem.go`:
```go
var ExpandedOrchEcosystemRepos = map[string]bool{
    "orch-go": true,
    "orch-cli": true,
    // ... more repos
}
```

The system searches common paths (`~/Documents/personal/`, `~/`, `~/projects/`, `~/code/`) to find repos.

**Source:** `pkg/spawn/ecosystem.go:16-29` and `cmd/orch/changelog.go:126-151`

**Significance:** Adding new repos requires editing the registry map and rebuilding; no runtime configuration needed.

---

### Finding 3: Semantic Parsing Taxonomy

**Evidence:** The system uses three dimensions for semantic analysis:
1. **ChangeType:** documentation, behavioral, structural, unknown (based on conventional commit prefix or file types)
2. **BlastRadius:** local, cross-skill, infrastructure (based on which files/packages are modified)
3. **Category:** skills, kb, cmd, pkg, web, docs, config, other (based on file paths)

**Source:** `cmd/orch/changelog.go:56-72` (types), `cmd/orch/changelog_test.go:239-550` (test cases documenting behavior)

**Significance:** Semantic parsing enables filtering for "notable" changes in orch complete and provides useful categorization in output.

---

## Synthesis

**Key Insights:**

1. **Three integration points, one core function** - `GetChangelog(days, project)` is the shared core; CLI formats output, API returns JSON, orch complete filters for notable entries.

2. **Ecosystem is statically defined** - No dynamic discovery; adding repos requires code change + rebuild. This is intentional for reliability.

3. **Semantic parsing is conservative** - Falls back to file-based inference when conventional commit format isn't used; never fails, just returns "unknown".

**Answer to Investigation Question:**

The changelog system aggregates git commits from repos defined in `ExpandedOrchEcosystemRepos`. To add a new repo:
1. Edit `pkg/spawn/ecosystem.go` to add the repo name to the map
2. Run `make install` to rebuild
3. Ensure the repo exists at a discoverable path

Documentation now exists at:
- `docs/changelog-system.md` - comprehensive system documentation
- `README.md` - quick reference for CLI usage
- `~/.orch/ECOSYSTEM.md` - ecosystem expansion instructions

---

## References

**Files Examined:**
- `cmd/orch/changelog.go` - Core changelog implementation
- `cmd/orch/changelog_test.go` - Test cases documenting behavior
- `cmd/orch/serve.go` - API endpoint registration
- `cmd/orch/main.go:3508-3600` - orch complete integration
- `pkg/spawn/ecosystem.go` - Repo registry

**Files Created/Modified:**
- `docs/changelog-system.md` - Created comprehensive documentation
- `README.md` - Added changelog command section
- `~/.orch/ECOSYSTEM.md` - Added ecosystem expansion instructions

---

## Investigation History

**2026-01-03:** Investigation started
- Initial question: How to document changelog pattern for ecosystem expansion
- Context: Orchestrator requested documentation of existing changelog implementation

**2026-01-03:** Investigation completed
- Status: Complete
- Key outcome: Created comprehensive documentation across three files covering CLI, API, semantic parsing, and ecosystem expansion
