<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Beads should remain as external dependency with current abstraction approach (Phase 3 plan) - alternatives are higher cost with lower benefit.

**Evidence:** 19 exec.Command calls use only 7 bd subcommands (comment, comments, create, list, ready, show, stats); 1,192 existing issues with 36h average lead time; existing decision record (2025-12-21) established clean-slate relationship with upstream.

**Knowledge:** The beads interface is already narrow (7 commands); full replacement would sacrifice dep graph, convergence tracking, and multi-repo sync; GitHub Issues lacks dependency-first design which is core to orch workflow.

**Next:** Proceed with Phase 3 abstraction layer (pkg/beads/client.go) per ecosystem audit - this addresses the risk while preserving value.

**Confidence:** High (85%) - clear feature gap analysis, but uncertainty in future beads API stability.

---

# Investigation: Beads Dependency Strategy Follow-Up

**Question:** Should beads remain as external dependency or should we pursue an alternative approach (fork, replace, or migrate to GitHub Issues)?

**Started:** 2025-12-24
**Updated:** 2025-12-24
**Owner:** Design Session Agent
**Phase:** Complete
**Next Step:** None - recommend proceeding with Phase 3 abstraction
**Status:** Complete
**Confidence:** High (85%)

**Extracted-From:** `.kb/investigations/2025-12-24-inv-full-ecosystem-audit-scope-simplify.md` (Phase 3 follow-up)

---

## Context

The ecosystem audit (orch-go-uhsi) identified beads as the only external dependency in the ecosystem:
- **Owner:** stevey (not Dylan)
- **Size:** 31MB binary, 347MB repo
- **CLAUDE.md:** 28k characters (complex internal architecture)
- **Usage:** 307 skill references, core workflow tool

The audit recommended Phase 3: "Create pkg/beads/client.go interface, reduce direct bd CLI calls, enable mock testing." This investigation evaluates whether that's the right path vs alternatives.

---

## Findings

### Finding 1: Narrow Interface Surface

**Evidence:** orch-go uses only 7 of beads' 30+ commands:

| Command | Usage Location | Purpose |
|---------|---------------|---------|
| `bd comment` | opencode/service.go:161 | Add progress comments |
| `bd comments` | verify/check.go:42, opencode/service.go:147 | Get comments |
| `bd create` | main.go:1502 | Create issues |
| `bd list` | verify/check.go:539, swarm.go:232, handoff.go:388,420,473 | Query issues |
| `bd ready` | daemon.go:313, focus.go:391 | Get ready queue |
| `bd show` | verify/check.go:485,513, spawn/skill_requires.go:249,265 | Get issue details |
| `bd stats` | serve.go:858 | Dashboard stats |

**Source:** `grep -r "exec.Command.*bd"` across orch-go (19 total call sites)

**Significance:** The dependency surface is already narrow. An abstraction layer over these 7 commands is feasible and low-risk.

---

### Finding 2: Current Beads Relationship Already Resolved

**Evidence:** Decision record exists: `.kb/decisions/2025-12-21-beads-oss-relationship-clean-slate.md`

Key decisions already made:
- Drop all local features and use upstream beads as-is
- Don't fork, don't maintain local patches
- One skill reference to `--discovered-from` updated

**Source:** Decision record from 3 days ago

**Significance:** This investigation is specifically about structural relationship (external dep vs internal), not about fork vs upstream. The fork question is already answered (don't fork).

---

### Finding 3: Beads Feature Set vs Alternatives

**Evidence:** Feature comparison:

| Feature | Beads | GitHub Issues | Simple Internal |
|---------|-------|---------------|-----------------|
| Dependency graph (`bd dep`) | ✅ First-class | ⚠️ Manual links | ❌ Would need building |
| Ready queue (`bd ready`) | ✅ Built-in | ❌ No equivalent | ❌ Would need building |
| Multi-repo sync | ✅ Via daemon | ✅ Native | ❌ N/A |
| Epic structure | ✅ Parent/child | ⚠️ Via labels | ❌ Would need building |
| Comments | ✅ Native | ✅ Native | ⚠️ Simple |
| Phase tracking | ✅ Via comments | ⚠️ Via comments | ⚠️ Via comments |
| JSONL storage | ✅ Git-friendly | ❌ API only | ⚠️ Could implement |
| Offline capable | ✅ Full | ❌ Needs network | ✅ Full |

**Source:** Feature analysis, `bd --help`

**Significance:** Beads' dependency-first design is core to orch workflow. The `bd ready` command (issues ready to work with all blockers resolved) has no GitHub Issues equivalent. Replacing beads would mean rebuilding significant functionality.

---

### Finding 4: Migration Cost Analysis

**Evidence:** Current state:
- 1,192 issues in orch-go alone
- 36.7h average lead time (workflow is established)
- 19 call sites in orch-go code
- 307 skill references to `bd` commands

Migration to GitHub Issues would require:
1. Rewriting 19 call sites (moderate)
2. Migrating 1,192 issues (significant tooling)
3. Updating 307 skill references (significant)
4. Losing dependency tracking capability (breaking change)
5. Requiring network for all operations (workflow change)

Migration to internal tracker would require:
1. Building issue storage, querying, CRUD (high)
2. Building dependency graph and resolution (high)
3. Building ready queue logic (moderate)
4. All of the migration work above

**Source:** `bd stats --json`, skill reference counts from ecosystem audit

**Significance:** Both alternatives have high migration cost. Internal tracker has highest implementation cost. Neither provides clear benefit over current approach.

---

### Finding 5: Phase 3 Abstraction Is Already Planned

**Evidence:** Ecosystem audit Phase 3 recommendation:
> "Create pkg/beads/client.go interface, reduce direct bd CLI calls, enable mock testing"

This addresses the main risks without requiring replacement:
- **API stability risk:** Abstraction layer isolates changes
- **Testing difficulty:** Mock interface enables unit testing
- **Maintenance burden:** No additional burden (we're not modifying beads)

**Source:** `.kb/investigations/2025-12-24-inv-full-ecosystem-audit-scope-simplify.md`

**Significance:** The planned abstraction layer addresses the identified risks at much lower cost than replacement.

---

## Synthesis

**Key Insights:**

1. **Interface is already narrow** - Only 7 of 30+ commands used. Abstraction layer is low-cost.

2. **Unique value proposition** - Beads' dependency-first design (ready queue, dep graph, blockers) has no equivalent in GitHub Issues or simple internal trackers.

3. **Cost/benefit asymmetry** - Replacement costs (migration, rebuild, skill updates) vastly exceed abstraction costs.

4. **Decision already made** - The fork/upstream question was resolved 3 days ago. This is about external dep vs rebuild, not about maintaining patches.

**Answer to Investigation Question:**

Beads should **remain as external dependency** with the Phase 3 abstraction layer approach. The alternatives are:

| Option | Cost | Benefit | Verdict |
|--------|------|---------|---------|
| Keep with abstraction (Phase 3) | Low (interface layer) | Isolation, testability | ⭐ Recommended |
| Fork and maintain | Medium (fork overhead) | Control | ❌ Already rejected |
| Replace with GitHub Issues | High (migration, rebuild) | GitHub integration | ❌ Loses dependency graph |
| Replace with internal tracker | Very High (build from scratch) | Full control | ❌ Unjustified effort |

---

## Confidence Assessment

**Current Confidence:** High (85%)

**Why this level?**

Strong evidence from code analysis and feature comparison. Clear cost/benefit picture. Uncertainty only in external beads roadmap which is addressed by abstraction layer.

**What's certain:**

- ✅ Only 7 beads commands used (narrow interface)
- ✅ Dependency graph is core to workflow (can't easily replace)
- ✅ Abstraction layer addresses API stability risk
- ✅ Migration costs would be significant (1,192 issues, 307 skill refs)

**What's uncertain:**

- ⚠️ Future beads API changes (mitigated by abstraction)
- ⚠️ Long-term beads maintenance by stevey
- ⚠️ Whether beads features we don't use might become valuable

**What would increase confidence to Very High:**

- Commitment from stevey on API stability
- Longer track record of stable upstream releases
- Test coverage of abstraction layer

---

## Implementation Recommendations

### Recommended Approach ⭐

**Proceed with Phase 3 abstraction layer** - Create `pkg/beads/client.go` interface wrapping the 7 commands used.

**Why this approach:**
- Lowest cost path (just an interface layer)
- Addresses API stability risk (single point of change)
- Enables mock testing for daemon and verify packages
- Preserves all beads value (dependency graph, ready queue)

**Trade-offs accepted:**
- Still dependent on external project
- No control over beads roadmap

**Implementation sequence:**
1. Create `pkg/beads/interface.go` defining the 7-operation interface
2. Create `pkg/beads/client.go` implementing via `bd` CLI
3. Create `pkg/beads/mock.go` for testing
4. Update daemon, verify, serve to use interface
5. Add tests using mock

### Alternative Approaches Considered

**Option B: Migrate to GitHub Issues**
- **Pros:** Industry standard, GitHub integration, no external CLI
- **Cons:** Loses dependency graph, requires network, high migration cost
- **When to use instead:** If beads becomes unmaintained AND dependency tracking is dropped from workflow

**Option C: Build internal tracker**
- **Pros:** Full control, exact feature set needed
- **Cons:** Weeks of development, reinventing wheel, maintenance burden
- **When to use instead:** Never - cost/benefit doesn't justify

**Option D: Do nothing**
- **Pros:** Zero effort
- **Cons:** Direct CLI calls harder to test, API changes require scattered updates
- **When to use instead:** If testing and maintainability aren't priorities

---

### Implementation Details

**What to implement first:**
- Interface definition (establishes contract)
- Client implementation (proves interface works)
- One consumer migration (daemon) as validation

**Things to watch out for:**
- ⚠️ JSON parsing quirks (some commands return null for empty)
- ⚠️ Error message formats vary by command
- ⚠️ Working directory affects `.beads/` detection

**Success criteria:**
- ✅ All 19 call sites use interface
- ✅ daemon package has mock-based tests
- ✅ verify package has mock-based tests
- ✅ Zero direct `exec.Command("bd", ...)` calls outside pkg/beads

---

## References

**Files Examined:**
- `pkg/verify/check.go` - Comment and issue retrieval
- `pkg/daemon/daemon.go` - Ready queue polling
- `pkg/spawn/skill_requires.go` - Issue context loading
- `cmd/orch/serve.go` - Stats API
- `cmd/orch/swarm.go` - Batch spawning

**Commands Run:**
```bash
# Find beads CLI usage
grep -r "exec.Command.*bd" --include="*.go"

# Get beads stats
bd stats --json

# Check beads CLI commands
bd --help
```

**Related Artifacts:**
- **Decision:** `.kb/decisions/2025-12-21-beads-oss-relationship-clean-slate.md` - Fork/upstream decision
- **Investigation:** `.kb/investigations/2025-12-24-inv-full-ecosystem-audit-scope-simplify.md` - Parent audit
- **Workspace:** `.orch/workspace/og-work-follow-up-ecosystem-24dec/`

---

## Investigation History

**2025-12-24 17:30:** Investigation started
- Initial question: Should beads remain as external dependency vs alternatives?
- Context: Follow-up to ecosystem audit Phase 3 recommendation

**2025-12-24 17:45:** Context gathering complete
- Found 19 beads CLI call sites using 7 commands
- Found existing decision record on fork question
- Analyzed feature gaps in alternatives

**2025-12-24 18:00:** Investigation completed
- Final confidence: High (85%)
- Status: Complete
- Key outcome: Recommend proceeding with Phase 3 abstraction - alternatives have poor cost/benefit
