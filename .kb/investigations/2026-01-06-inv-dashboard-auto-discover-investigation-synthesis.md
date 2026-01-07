<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Added auto-discovery for investigation files when agents don't report `investigation_path:` via beads comment - uses fallback chain searching .kb/investigations/ by workspace keywords and beads ID.

**Evidence:** Tests pass for keyword extraction, hex detection, and discovery logic. Build succeeds. Discovery correctly matches workspace names like "og-inv-skillc-deploy-structure-06jan-ed96" to investigation files like "2026-01-04-inv-skillc-deploy-structure.md".

**Knowledge:** Workspace names contain meaningful keywords (topic words) between the skill prefix and date suffix. These can be extracted and matched against .kb/investigations/ filenames to find related investigations without requiring agents to explicitly report paths.

**Next:** Close this issue - implementation is complete with tests.

**Promote to Decision:** recommend-no (tactical improvement to existing feature, not architectural)

---

# Investigation: Dashboard Auto Discover Investigation Synthesis

**Question:** How can we make the dashboard Investigation tab work even when agents don't report investigation_path via beads comment?

**Started:** 2026-01-06
**Updated:** 2026-01-06
**Owner:** Feature implementation agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Current implementation relies on beads comment reporting

**Evidence:** In `serve_agents.go:580-583`, investigation_path is extracted from beads comments using `verify.ParseInvestigationPathFromComments()`. If no comment exists with `investigation_path:` prefix, the field remains empty.

**Source:** cmd/orch/serve_agents.go:580-583, pkg/verify/beads_api.go:130-144

**Significance:** Many agents don't report investigation_path (as noted in issue), leaving dashboard Investigation tab empty even when investigation files exist.

---

### Finding 2: Workspace names contain topic keywords

**Evidence:** Workspace names follow pattern `{project}-{skill}-{topic}-{date}-{hash}`. Examples:
- `og-inv-skillc-deploy-structure-06jan-ed96` contains keywords: "skillc", "deploy", "structure"
- `og-feat-dashboard-auto-discover-06jan-dfc6` contains keywords: "dashboard", "auto", "discover"

**Source:** Analyzed workspace naming patterns in .orch/workspace/

**Significance:** These keywords can be matched against .kb/investigations/ filenames which follow pattern `YYYY-MM-DD-{type}-{topic}.md`.

---

### Finding 3: Investigation files follow predictable naming

**Evidence:** Investigation files in .kb/investigations/ follow pattern:
- `2026-01-06-inv-dashboard-auto-discover-investigation-synthesis.md`
- `2026-01-04-inv-skillc-deploy-structure.md`

**Source:** Examined .kb/investigations/ directory structure

**Significance:** The topic portion of filenames matches workspace keywords, enabling auto-discovery through substring matching.

---

## Synthesis

**Key Insights:**

1. **Keyword extraction is reliable** - By skipping common prefixes (og, inv, feat), date suffixes, and hex-like hashes, we can extract meaningful topic keywords from workspace names.

2. **Multiple fallback locations** - Investigation files may be in .kb/investigations/, .kb/investigations/simple/, or even in the workspace directory itself.

3. **Low false-positive risk** - Matching by topic keywords is unlikely to return wrong files since investigation topics are typically unique enough.

**Answer to Investigation Question:**

The dashboard can auto-discover investigation files by extracting keywords from workspace names and matching them against filenames in .kb/investigations/. The implementation uses a 3-level fallback chain:
1. Search .kb/investigations/ for files matching workspace keywords
2. Search for files matching beads ID
3. Check workspace directory for local .md files

---

## Structured Uncertainty

**What's tested:**

- extractWorkspaceKeywords() correctly extracts topic keywords (verified: TestExtractWorkspaceKeywords passed)
- isHexLike() correctly identifies hex-like strings (verified: TestIsHexLike passed)
- discoverInvestigationPath() finds investigation files by keywords (verified: TestDiscoverInvestigationPath passed)
- Full build succeeds (verified: go build ./cmd/orch/...)

**What's untested:**

- Performance impact on /api/agents with many investigation files (not benchmarked)
- Edge cases with very long workspace names (not tested)
- Behavior when multiple investigation files match same keywords (returns first match)

**What would change this:**

- If investigation file naming convention changes, keyword matching would break
- If workspace naming convention changes, keyword extraction would need updating

---

## Implementation Recommendations

### Recommended Approach: Fallback Chain with Keyword Matching

**Implemented in:** cmd/orch/serve_agents.go

**Why this approach:**
- Works transparently without requiring agents to change behavior
- Uses existing naming conventions - no new metadata needed
- Graceful degradation - if auto-discovery fails, falls back to empty state

**Trade-offs accepted:**
- May occasionally match wrong file if topics overlap (acceptable given rarity)
- Small performance cost from directory scanning (mitigated by caching)

**Implementation sequence:**
1. Extract keywords from workspace name (extractWorkspaceKeywords)
2. Search .kb/investigations/ for matching files
3. Search .kb/investigations/simple/ as fallback
4. Check workspace directory for local investigation files

---

## References

**Files Examined:**
- cmd/orch/serve_agents.go - API handler and agent enrichment logic
- cmd/orch/serve_agents_cache.go - Workspace cache implementation
- pkg/verify/beads_api.go - Investigation path parsing from comments
- web/src/lib/components/agent-detail/investigation-tab.svelte - Frontend display

**Commands Run:**
```bash
# Build verification
go build ./cmd/orch/...

# Test discovery functions
go test -v ./cmd/orch/... -run "TestExtractWorkspace|TestIsHexLike|TestDiscoverInvestigation"

# Install updated binary
make install
```

**Related Artifacts:**
- **Decision:** Dashboard uses SYNTHESIS.md as fallback for untracked agent completion detection
- **Constraint:** Dashboard must be usable at 666px width (no impact on this change)

---

## Investigation History

**2026-01-06 21:00:** Investigation started
- Initial question: How to auto-discover investigation files when agents don't report paths?
- Context: Issue orch-go-wrrks - many agents not reporting investigation_path via beads comment

**2026-01-06 21:30:** Implementation complete
- Added discoverInvestigationPath() with fallback chain
- Added extractWorkspaceKeywords() for keyword extraction
- Added comprehensive tests

**2026-01-06 21:45:** Investigation completed
- Status: Complete
- Key outcome: Auto-discovery implemented using workspace keyword matching against .kb/investigations/ files
