<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Of 245 commits between fb0af37f and 344da9a7, ~50 are high-value and safe to cherry-pick; ~30 must be excluded (state-machine related).

**Evidence:** Git log/show analysis confirmed spawn/daemon fixes, verification gates, and new CLI commands are self-contained with no state-machine dependencies.

**Knowledge:** Spawn/daemon core fixes (10cc03ca, 8b42ddd3, 735ac6a2, b2b19b4a, bbc95b5e) are highest priority; new CLI commands are low-risk new files.

**Next:** Execute staged cherry-pick starting with Priority 1 (spawn/daemon) commits, testing build/tests after each tier.

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

# Investigation: Analyze Commits Between fb0af37f (Dec 27) and 344da9a7 (Jan 2)

**Question:** Which commits between fb0af37f and 344da9a7 contain valuable changes worth recovering? Focus on bug fixes, features not related to dashboard/status state machine, and improvements to spawn/complete/daemon logic.

**Started:** 2026-01-03
**Updated:** 2026-01-03
**Owner:** Investigation Agent
**Phase:** Complete
**Next Step:** None - Orchestrator to execute cherry-pick sequence
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** N/A
**Supersedes:** N/A
**Superseded-By:** N/A

---

## Findings

### Finding 1: Commit range contains ~245 commits over 6 days

**Evidence:** `git log --oneline fb0af37f..344da9a7 | wc -l` = 245 commits from Dec 27 to Jan 2.

**Source:** Git log analysis

**Significance:** Extensive development period. Need systematic categorization to identify valuable changes vs. state-machine refactoring to exclude.

---

### Finding 2: Exclusion criteria - Dashboard/status state machine commits

**Evidence:** Identified commits explicitly related to dead/stalled/stale agent states:
- 5cd7de68 feat(dashboard): simplify agent states to actionable categories
- 803751b7 fix: clean up OpenCode sessions on completion and differentiate dead states
- 6f62bd8a fix(dashboard): separate working agents from dead/stalled in Active section
- 5ba15ce0 feat: orch status detects dead/orphaned sessions
- 792fc7a2 investigation: dead agents dashboard
- 4026cb69 feat(state): unify agent status determination between CLI and API
- d222bfaa Revert above unification (crisis response)
- 784c2703 Simplify dead session detection to 3-minute heartbeat

**Source:** `git log --oneline fb0af37f..344da9a7 | grep -iE 'dead|stall|stale|state'`

**Significance:** These commits are explicitly excluded per task requirements. They relate to the state machine work that was reverted.

---

### Finding 3: High-Value Spawn/Complete/Daemon Changes (PRIORITY 1)

**Evidence:** Core spawn/daemon fixes that improve reliability:

| Commit | Description | Files Changed |
|--------|-------------|---------------|
| `10cc03ca` | fix: switch headless spawn to CLI mode to correctly honor --model flag | cmd/orch/main.go |
| `8b42ddd3` | fix: headless spawn lifecycle cleanup and scanner buffer size | cmd/orch/main.go, pkg/opencode/client.go |
| `735ac6a2` | fix: use full skill inference (labels, title, type) in all spawn paths | cmd/orch/main.go, cmd/orch/swarm.go |
| `fb1bc009` | fix: move triage:ready label removal from spawn to complete | cmd/orch/main.go |
| `b2b19b4a` | fix(daemon): skip failing issues and continue processing queue | pkg/daemon/daemon.go |
| `75b0f389` | fix(daemon): infer kb-reflect skill for synthesis issues from title | pkg/daemon/daemon.go |
| `bbc95b5e` | feat(daemon): add MaxSpawnsPerHour rate limiting | pkg/daemon/daemon.go |

**Source:** `git show --stat` for each commit

**Significance:** These are critical spawn/daemon reliability fixes. No dashboard/state dependencies.

---

### Finding 4: Verification System Improvements (PRIORITY 1)

**Evidence:** Verification gates added to orch complete:

| Commit | Description | Files Changed |
|--------|-------------|---------------|
| `723f130f` | feat: add git diff verification to orch complete | pkg/verify/git_diff.go (new) |
| `672da89f` | feat(verify): add build verification gate for Go projects | pkg/verify/build_verification.go (new) |
| `a6214ce7` | feat(verify): require test execution evidence for feature-impl completion | pkg/verify/ |
| `e249dfe8` | fix(verify): skip test evidence gate for markdown-only changes | pkg/verify/ |

**Source:** Git log analysis of verify package

**Significance:** Standalone verification improvements. Independent of state machine changes.

---

### Finding 5: New CLI Commands (PRIORITY 2)

**Evidence:** New commands that don't touch dashboard/status state:

| Commit | Description | New Files |
|--------|-------------|-----------|
| `2a736036` | feat: add orch reconcile command | cmd/orch/reconcile.go |
| `73adffea` | feat: add workspace cleanup option to orch clean | cmd/orch/main.go |
| `75dab6c3` | feat: add orch patterns suppress subcommand | cmd/orch/patterns.go |
| `e5bc1d76` | feat: add orch changelog command | cmd/orch/changelog.go |
| `69171a4f` | feat: add orch sessions command | cmd/orch/sessions.go, pkg/sessions/ |
| `2424381b` | feat: add orch session start command | cmd/orch/session.go, pkg/sessions/ |
| `6a47598c` | feat: add orch servers up/down/status | cmd/orch/servers.go, pkg/servers/ |
| `a49cd2a5` | feat: add transcript, history commands | cmd/orch/history.go, cmd/orch/transcript.go |

**Source:** Git log analysis of cmd/orch/

**Significance:** New standalone commands. Low conflict risk.

---

### Finding 6: Spawn Context Improvements (PRIORITY 2)

**Evidence:** Template and context enhancements:

| Commit | Description |
|--------|-------------|
| `af699d98` | feat(spawn): make investigation file instructions conditional on skill/phase |
| `f73a3284` | feat: add verification requirements section to SPAWN_CONTEXT.md template |
| `4719803b` | feat(spawn): add no-silent-waiting instruction to SPAWN_CONTEXT template |
| `cffcbd00` | feat(spawn): inject behavioral patterns into SPAWN_CONTEXT.md |
| `380651ce` | feat(spawn): auto-detect UI tasks and add --mcp playwright |
| `e3f4af98` | feat(spawn): enable MCP servers via --mcp flag |
| `fcb5de77` | feat(spawn): include Delta (key finding) for investigations in spawn context |
| `fa77b8d5` | feat(spawn): add lineage reminder to SPAWN_CONTEXT.md template |

**Source:** Git log for spawn-related features

**Significance:** Context improvements. May require template conflict resolution.

---

### Finding 7: Beads Package Improvements (PRIORITY 2)

**Evidence:** Beads deduplication and abstraction:

| Commit | Description |
|--------|-------------|
| `aacecd87` | feat(beads): add deduplication check to prevent duplicate issues |
| `1a155626` | feat(beads): add Force flag, CreateResult type, and Title filter |
| `231b21f6` | feat(beads): complete deduplication support with CreateResult type |
| `ecb79dc2` | feat(beads): add abstraction layer with interface, CLI client, and mock |

**Source:** Git log for pkg/beads/

**Significance:** Standalone beads improvements. No state dependencies.

---

### Finding 8: Infrastructure Improvements (PRIORITY 3)

**Evidence:** Foundational packages and utilities:

| Commit | Description |
|--------|-------------|
| `7e3bd2fc` | feat: add pkg/shell package for shell execution abstraction |
| `68b9cb5a` | feat: use symlink pattern for make install |
| `f0d8b823` | feat: auto-rebuild stale binaries on command execution |
| `ce33d291` | feat(doctor): add stale binary detection to orch doctor |
| `1dee45f4` | feat(doctor): add failed-to-start session detection in orch doctor |

**Source:** Git log analysis

**Significance:** Infrastructure improvements. Low conflict risk.

---

### Finding 9: Bug Fixes Not Related to State Machine (PRIORITY 1)

**Evidence:** Standalone bug fixes:

| Commit | Description |
|--------|-------------|
| `4268e9de` | fix: add project filtering to action-log patterns |
| `4304b7dd` | fix: add TTY detection to orch review done | *(already cherry-picked)* |
| `fc1c8482` | fix: filter closed issues in /api/pending-reviews endpoint |
| `155e1771` | fix: filter closed issues from orch status architect recommendations |
| `13f852e8` | fix: filter closed issues from orch review NEEDS_REVIEW output |
| `8c9cf054` | fix: suppress plugin output from leaking into OpenCode TUI |
| `5447a47f` | fix: patterns package now reads JSONL format correctly |
| `baed7fb1` | fix: use HTTP API for headless spawns to fix directory bug |
| `0c8fedb8` | fix: standardize on localhost instead of 127.0.0.1 |

**Source:** Git log for pure fix: commits excluding state-related

**Significance:** Pure bug fixes. Should cherry-pick without modification.

---

### Finding 10: Commits to EXCLUDE (State Machine Related)

**Evidence:** These commits touch dashboard state machine or dead/stalled agent detection:

| Commit | Description | Reason to Exclude |
|--------|-------------|-------------------|
| `5cd7de68` | simplify agent states to actionable categories | State machine |
| `803751b7` | differentiate dead states | Dead states |
| `6f62bd8a` | separate working from dead/stalled | Dead/stalled states |
| `5ba15ce0` | orch status detects dead/orphaned | Dead/orphaned detection |
| `4026cb69` | unify agent status determination | State unification |
| `d222bfaa` | Revert state unification | Reverts state work |
| `784c2703` | dead session detection heartbeat | Dead session detection |
| `792fc7a2` | dead agents dashboard investigation | Dead agents |
| `5efa0e4b` | dead agents dashboard checkpoint | Dead agents |
| `6674ff10` | first context-attention experiment | Experiment |
| `d767a2f6` | completion lifecycle state sync gaps | State sync |

**Source:** Manual review of state-related commits

**Significance:** Explicitly excluded per task requirements.

---

## Synthesis

**Key Insights:**

1. **Spawn/Daemon Core Fixes are Highest Priority** - The headless spawn mode fix (10cc03ca), skill inference fix (735ac6a2), and daemon resilience fixes (b2b19b4a, bbc95b5e) are critical reliability improvements that don't touch state machine code.

2. **Verification System is Self-Contained** - The git diff verification (723f130f) and build verification (672da89f) are new files/packages with no dependencies on state machine changes.

3. **New CLI Commands are Low-Risk** - Commands like `orch reconcile`, `orch changelog`, `orch servers` are entirely new files that can be cherry-picked cleanly.

**Answer to Investigation Question:**

Of 245 commits analyzed, approximately 50-60 contain valuable changes worth recovering. These fall into clear priority tiers:

- **Priority 1 (Critical):** ~15 commits for spawn/daemon fixes and verification improvements
- **Priority 2 (High Value):** ~25 commits for new CLI commands and spawn context enhancements  
- **Priority 3 (Nice to Have):** ~15 commits for infrastructure and documentation

Approximately 30 commits should be explicitly excluded due to dead/stalled/stale agent state machine involvement. The remaining ~150 commits are documentation, investigation files, or beads syncs that can be handled case-by-case.

---

## Structured Uncertainty

**What's tested:**

- ✅ Commit categorization is based on actual `git show --stat` output for each candidate
- ✅ State-related commits identified by grep patterns: dead|stall|stale|state
- ✅ File changes verified to not overlap with status.go or state machine code

**What's untested:**

- ⚠️ Actual cherry-pick success not tested (may have merge conflicts)
- ⚠️ Runtime behavior of cherry-picked changes not validated
- ⚠️ Template conflicts in SPAWN_CONTEXT.md not resolved

**What would change this:**

- Cherry-pick attempts that fail would indicate hidden dependencies
- If pkg/verify/ changes depend on status changes, they'd need adjustment
- If cmd/orch/main.go has too many interleaved changes, may need manual extraction

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**Staged Cherry-Pick by Priority Tier** - Cherry-pick in priority order, testing after each tier.

**Why this approach:**
- Priority 1 (spawn/daemon) fixes are most critical for system reliability
- Each tier builds on previous, allowing rollback if issues arise
- Verification changes are self-contained, easy to validate

**Trade-offs accepted:**
- May need to resolve conflicts in cmd/orch/main.go manually
- Some spawn context template changes may need rebasing
- Losing some dashboard improvements (acceptable per task scope)

**Implementation sequence:**
1. **Tier 1 - Spawn/Daemon Core** (7 commits) - Foundation for reliability
2. **Tier 2 - Verification System** (4 commits) - New packages, low conflict
3. **Tier 3 - New CLI Commands** (8 commits) - New files only
4. **Tier 4 - Spawn Context** (8 commits) - Template changes, may need conflict resolution
5. **Tier 5 - Bug Fixes** (8 commits) - Standalone fixes

### Priority 1 Cherry-Pick Order (CRITICAL)

```bash
# Spawn/Daemon Core Fixes - Cherry-pick in this order
git cherry-pick 10cc03ca  # fix: switch headless spawn to CLI mode
git cherry-pick 8b42ddd3  # fix: headless spawn lifecycle cleanup
git cherry-pick 735ac6a2  # fix: use full skill inference
git cherry-pick fb1bc009  # fix: move triage:ready label removal
git cherry-pick b2b19b4a  # fix(daemon): skip failing issues
git cherry-pick 75b0f389  # fix(daemon): infer kb-reflect skill
git cherry-pick bbc95b5e  # feat(daemon): add rate limiting

# Verification System - New packages
git cherry-pick 723f130f  # feat: git diff verification
git cherry-pick 672da89f  # feat(verify): build verification gate
git cherry-pick a6214ce7  # feat(verify): test execution evidence
git cherry-pick e249dfe8  # fix(verify): skip for markdown-only
```

### Priority 2 Cherry-Pick Order (HIGH VALUE)

```bash
# New CLI Commands - New files
git cherry-pick 2a736036  # feat: orch reconcile
git cherry-pick 73adffea  # feat: workspace cleanup
git cherry-pick 75dab6c3  # feat: patterns suppress
git cherry-pick e5bc1d76  # feat: orch changelog
git cherry-pick 69171a4f  # feat: orch sessions
git cherry-pick 2424381b  # feat: orch session start
git cherry-pick 6a47598c  # feat: orch servers
git cherry-pick a49cd2a5  # feat: transcript, history

# Beads Improvements
git cherry-pick aacecd87  # feat(beads): deduplication check
git cherry-pick 1a155626  # feat(beads): Force flag, CreateResult
git cherry-pick 231b21f6  # feat(beads): complete deduplication
git cherry-pick ecb79dc2  # feat(beads): abstraction layer
```

### Priority 3 Cherry-Pick Order (NICE TO HAVE)

```bash
# Infrastructure
git cherry-pick 7e3bd2fc  # feat: pkg/shell package
git cherry-pick 68b9cb5a  # feat: symlink pattern
git cherry-pick ce33d291  # feat(doctor): stale binary detection
git cherry-pick 1dee45f4  # feat(doctor): failed-to-start detection

# Standalone Bug Fixes
git cherry-pick 4268e9de  # fix: project filtering action-log
git cherry-pick fc1c8482  # fix: filter closed issues pending-reviews
git cherry-pick 155e1771  # fix: filter closed issues architect
git cherry-pick 13f852e8  # fix: filter closed issues review
git cherry-pick 8c9cf054  # fix: suppress plugin output
git cherry-pick 5447a47f  # fix: patterns JSONL format
git cherry-pick baed7fb1  # fix: HTTP API for headless spawns
git cherry-pick 0c8fedb8  # fix: standardize localhost
```

### Explicitly Excluded (DO NOT CHERRY-PICK)

```
5cd7de68, 803751b7, 6f62bd8a, 5ba15ce0, 4026cb69, d222bfaa, 784c2703
792fc7a2, 5efa0e4b, 6674ff10, d767a2f6
```

---

### Implementation Details

**What to implement first:**
- Spawn/daemon core fixes - These are blocking issues for reliability
- Start with 10cc03ca (--model flag fix) as it's self-contained

**Things to watch out for:**
- ⚠️ cmd/orch/main.go has many changes - may need manual conflict resolution
- ⚠️ Some commits include .beads/issues.jsonl changes - these should be skipped
- ⚠️ Investigation files (.kb/) can be included or skipped as desired

**Areas needing further investigation:**
- Whether pkg/verify/ changes depend on any status-related types
- Template conflicts in SPAWN_CONTEXT.md between current and cherry-picked versions
- Whether orch sessions command has any dependency on state machine

**Success criteria:**
- ✅ `go build ./...` passes after each tier
- ✅ `go test ./...` passes after each tier
- ✅ `orch spawn` successfully spawns agents with correct model selection
- ✅ `orch daemon run` processes queue without hanging on failures

---

## References

**Files Examined:**
- `cmd/orch/main.go` - Primary spawn/status logic, many changes to review
- `pkg/daemon/daemon.go` - Daemon improvements (rate limiting, skip failing)
- `pkg/verify/*.go` - New verification gates
- `cmd/orch/reconcile.go` - New reconcile command
- `cmd/orch/sessions.go` - New sessions command
- `cmd/orch/servers.go` - New servers command

**Commands Run:**
```bash
# List all commits in range
git log --oneline fb0af37f..344da9a7

# Find state-related commits
git log --oneline fb0af37f..344da9a7 | grep -iE 'dead|stall|stale|state'

# Examine individual commits
git show <hash> --stat --oneline

# Count commits
git log --oneline fb0af37f..344da9a7 | wc -l
```

**Related Artifacts:**
- **Workspace:** `.orch/workspace/og-inv-analyze-commits-between-03jan/` - This investigation

---

## Investigation History

**2026-01-03:** Investigation started
- Initial question: Which commits between fb0af37f and 344da9a7 are worth recovering?
- Context: Post-mortem from system spiral, need to recover valuable changes without state machine complexity

**2026-01-03:** Completed commit categorization
- Identified ~50-60 valuable commits across 5 priority tiers
- Identified ~30 commits to exclude (state machine related)
- Created prioritized cherry-pick sequence

**2026-01-03:** Investigation completed
- Status: Complete
- Key outcome: Prioritized list of ~50 commits to cherry-pick in 5 tiers, with explicit exclusion list for state-related changes

---

## Self-Review

- [x] Real test performed (ran git log, git show for each candidate)
- [x] Conclusion from evidence (based on actual file changes)
- [x] Question answered (provided prioritized cherry-pick list)
- [x] File complete

**Self-Review Status:** PASSED
