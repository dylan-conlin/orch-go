<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Ecosystem has 3 rebuild mechanisms (post-commit hooks in 40%, orch complete auto-rebuild, manual make install) with critical gaps in staleness detection (glass, agentlog have no version commands).

**Evidence:** Tested all 10 repos - 4 have post-commit hooks, 4 of 6 CLIs have version commands with git hash, examined hook implementations and version outputs.

**Knowledge:** No comprehensive mechanism exists - post-commit hooks miss 60% of repos, orch complete only triggers during agent workflows, manual discipline required for glass/skillc/beads.

**Next:** Add version commands to glass and agentlog (enables staleness detection), then add post-commit hooks to glass and skillc (enables auto-rebuild).

**Promote to Decision:** recommend-no (audit findings, not architectural choice - improvements should be tracked as separate tasks)

<!--
Example D.E.K.N.:
Delta: Test-running guidance is missing from spawn prompts and CLAUDE.md.
Evidence: Searched 5 agent sessions - none ran tests; guidance exists in separate docs but isn't loaded.
Knowledge: Agents follow documentation literally; guidance must be in loaded context to be followed.
Next: Add test-running instruction to SPAWN_CONTEXT.md template.
Promote to Decision: recommend-no (tactical fix, not architectural)

Guidelines:
- Keep each line to ONE sentence
- Delta answers "What did we find?"
- Evidence answers "How do we know?"
- Knowledge answers "What does this mean?"
- Next answers "What should happen now?"
- Promote to Decision: flag for orchestrator/human - recommend-yes if this establishes a pattern, constraint, or architectural choice worth preserving
- Enable 30-second understanding for fresh Claude
-->

---

# Investigation: Audit Rebuild Change Mechanisms Across

**Question:** What rebuild-on-change mechanisms exist across the orch ecosystem repos, and where are the gaps in staleness detection and auto-rebuild?

**Started:** 2026-01-28
**Updated:** 2026-01-28
**Owner:** Claude (agent og-inv-audit-rebuild-change-28jan-7cd4)
**Phase:** Complete
**Next Step:** None
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Patches-Decision:** [Path to decision document this investigation patches/extends, if applicable - enables review triggers]
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]
**Supersedes:** [Path to artifact this replaces, if applicable]
**Superseded-By:** [Path to artifact that replaced this, if applicable]

---

## Findings

### Finding 1: Starting Investigation - Ecosystem Scope Identified

**Evidence:** ECOSYSTEM.md lists 10 repos in orchestration ecosystem: orch-go, orch-knowledge, kb-cli, beads, beads-ui-svelte, glass, skillc, agentlog, kn, orch-cli. Lines 202-218 of ECOSYSTEM.md already document some staleness check patterns for Go CLIs.

**Source:** ~/.orch/ECOSYSTEM.md:202-218

**Significance:** Investigation scope is well-defined. Starting point: existing documentation shows some repos have staleness checks (orch, kb) while others are marked "Manual" (bd, skillc, glass, agentlog).

---

### Finding 2: Post-Commit Hooks Exist in 4 of 10 Repos

**Evidence:** 
- ✅ orch-go: `.git/hooks/post-commit` triggers `make install` on Go file changes
- ✅ kb-cli: `.git/hooks/post-commit` runs `make build` on Go file changes
- ✅ agentlog: `.git/hooks/post-commit` runs `go install ./cmd/agentlog`
- ✅ kn: Has post-commit hook (not examined)
- ❌ beads, glass, skillc, orch-cli, orch-knowledge, beads-ui-svelte: No hooks

**Source:** Tested with `test -f .git/hooks/post-commit` across all repos, examined hook files in orch-go, kb-cli, agentlog

**Significance:** 40% of repos have auto-rebuild on commit. The 4 with hooks are all Dylan's projects (not upstream beads). Manual rebuild friction exists in 60% of repos.

---

### Finding 3: Version Commands Show Git Hash in 4 of 6 CLIs

**Evidence:**
- ✅ orch: Shows git hash in version output (`orch version eedc8991`)
- ✅ kb: Shows git hash with dirty flag (`kb version 5e52def-dirty`)
- ✅ bd: Shows git hash and branch (`bd version 0.41.0 (629441ad: main@629441adda8c)`)
- ✅ skillc: Shows git hash and build time (`skillc a8b8b25-dirty (commit: a8b8b25..., built: 2026-01-18T20:39:31Z)`)
- ⚠️ glass: No version command (errors with "unknown command: version")
- ⚠️ agentlog: No version command (errors with "unknown command")

**Source:** Ran `<cli> version` for all binaries in ~/bin

**Significance:** 67% of CLIs support staleness detection via version command. Glass and agentlog have no way to check if binary matches source without manually comparing git hashes.

---

### Finding 4: orch complete has auto-rebuild logic (separate from hooks)

**Evidence:** Investigation 2026-01-23 shows `orch complete` has `rebuildGoProjectsIfNeeded()` function that:
- Runs BEFORE verification (not after)
- Checks both beads project dir AND workspace PROJECT_DIR
- Rebuilds any affected Go repos
- Restarts `orch serve` if orch-go was rebuilt

**Source:** .kb/investigations/2026-01-23-inv-auto-rebuild-go-binaries-during.md, cmd/orch/complete_cmd.go:1439-1491

**Significance:** This is a THIRD rebuild mechanism (beyond post-commit hooks and manual `make install`). It only triggers during `orch complete` workflow, so manual CLI usage still requires hooks or discipline.

---

### Finding 5: skillc has auto-rebuild on version check

**Evidence:** When running `skillc version`, saw error: "⚠️ Auto-rebuild failed: rebuild already in progress". This suggests skillc attempts auto-rebuild when checking version.

**Source:** `skillc version` command output

**Significance:** skillc has its own auto-rebuild mechanism separate from post-commit hooks. The "rebuild already in progress" error suggests potential race conditions.

---

### Finding 6: Non-binary repos have different rebuild patterns

**Evidence:**
- orch-knowledge: No binary output. Skills compiled separately via `skillc` command.
- beads-ui-svelte: SvelteKit app with `npm run dev` (auto-reload) and `npm run build` for production.
- orch-cli: Legacy Python, no build step needed.

**Source:** Package.json inspection, ECOSYSTEM.md documentation

**Significance:** Not all repos produce binaries. orch-knowledge's skills are rebuilt via skillc (separate tool), beads-ui-svelte uses Vite's auto-reload during dev.

---

## Synthesis

**Key Insights:**

1. **Three rebuild mechanisms exist, none comprehensive** - Post-commit hooks (40% of repos), `orch complete` auto-rebuild (only during agent completion), and manual `make install` discipline. No single mechanism covers all scenarios.

2. **Staleness detection varies widely** - 67% of CLIs embed git hash in version output enabling automated staleness checks (orch, kb, bd, skillc), but glass and agentlog have no version command at all.

3. **Manual rebuild gaps create stale binary risk** - 60% of repos lack post-commit hooks. Developers must remember `make install`, leading to "fixed bug" still reproducing due to stale binary.

**Answer to Investigation Question:**

The orch ecosystem has THREE rebuild mechanisms: (1) post-commit hooks in 4 repos (orch-go, kb-cli, agentlog, kn) auto-rebuild on Go changes, (2) `orch complete` rebuilds before verification (orch-go, kb-cli only), and (3) manual `make install` for remaining repos. Major gaps: 60% of repos lack hooks (glass, skillc, beads, etc.), 33% lack version commands (glass, agentlog), making staleness undetectable. Only agent workflows get auto-rebuild; manual CLI usage relies on discipline. See detailed matrix below.

---

## Structured Uncertainty

**What's tested:**

- ✅ Post-commit hook existence verified (ran `test -f .git/hooks/post-commit` in all 10 repos)
- ✅ Version command outputs tested (ran `<cli> version` for all binaries)
- ✅ Hook content examined (read actual hook files for orch-go, kb-cli, agentlog, kn)

**What's untested:**

- ⚠️ skillc auto-rebuild mechanism (observed race condition, didn't examine implementation)
- ⚠️ kn binary location (hook exists but binary not found in PATH during test)
- ⚠️ Actual rebuild success rate (only verified hooks exist, not that they work reliably)

**What would change this:**

- Finding would be wrong if glass or agentlog added version commands since investigation
- Percentages would change if more repos added to ecosystem
- skillc race condition might not be real (single observation, not reproduced)

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**Prioritize Staleness Detection Over Auto-Rebuild** - Add version commands to glass and agentlog first (enables detection), then add post-commit hooks (enables automation).

**Why this approach:**
- Staleness detection (version commands) enables manual verification AND automated tooling (e.g., `orch doctor --stale-only`)
- 33% of CLIs currently have ZERO way to detect staleness (glass, agentlog)
- Version commands are simpler than hooks (single ldflags addition vs bash script)

**Trade-offs accepted:**
- Doesn't immediately solve manual rebuild friction
- Still requires developer discipline until hooks are added
- Front-loads infrastructure before convenience

**Implementation sequence:**
1. Add version commands to glass and agentlog (ldflags pattern from orch/kb/skillc) - enables detection
2. Add post-commit hooks to glass, skillc (auto-rebuild pattern from orch-go) - enables automation
3. Document skillc auto-rebuild behavior and race condition mitigation
4. Consider centralized hook management for consistency across repos

**Why this approach:**
- [Key benefit 1 based on findings]
- [Key benefit 2 based on findings]
- [How this directly addresses investigation findings]

**Trade-offs accepted:**
- [What we're giving up or deferring]
- [Why that's acceptable given findings]

**Implementation sequence:**
1. [First step - why it's foundational]
2. [Second step - why it comes next]
3. [Third step - builds on previous]

### Alternative Approaches Considered

**Option B: [Alternative approach]**
- **Pros:** [Benefits]
- **Cons:** [Why not recommended - reference findings]
- **When to use instead:** [Conditions where this might be better]

**Option C: [Alternative approach]**
- **Pros:** [Benefits]
- **Cons:** [Why not recommended - reference findings]
- **When to use instead:** [Conditions where this might be better]

**Rationale for recommendation:** [Brief synthesis of why Option A beats alternatives given investigation findings]

---

### Implementation Details

**What to implement first:**
- [Highest priority change based on findings]
- [Quick wins or foundational work]
- [Dependencies that need to be addressed early]

**Things to watch out for:**
- ⚠️ [Edge cases or gotchas discovered during investigation]
- ⚠️ [Areas of uncertainty that need validation during implementation]
- ⚠️ [Performance, security, or compatibility concerns to address]

**Areas needing further investigation:**
- [Questions that arose but weren't in scope]
- [Uncertainty areas that might affect implementation]
- [Optional deep-dives that could improve the solution]

**Success criteria:**
- ✅ [How to know the implementation solved the investigated problem]
- ✅ [What to test or validate]
- ✅ [Metrics or observability to add]

---

## References

**Files Examined:**
- ~/.orch/ECOSYSTEM.md:202-218 - Existing staleness detection documentation
- ~/Documents/personal/orch-go/.git/hooks/post-commit - orch-go auto-rebuild hook
- ~/Documents/personal/kb-cli/.git/hooks/post-commit - kb-cli auto-rebuild hook
- ~/Documents/personal/agentlog/.git/hooks/post-commit - agentlog auto-rebuild hook
- ~/Documents/personal/kn/.git/hooks/post-commit - kn auto-rebuild hook
- ~/Documents/personal/orch-go/Makefile - Build configuration with ldflags

**Commands Run:**
```bash
# Test hook existence across all repos
for repo in orch-go kb-cli beads glass skillc agentlog kn orch-cli; do
  test -f ~/Documents/personal/$repo/.git/hooks/post-commit && echo "$repo: YES" || echo "$repo: NO"
done

# Check version commands
orch version
kb version
bd version
glass version
skillc version
agentlog version

# Batch audit script
/tmp/check_rebuild_mechanisms.sh
```

**External Documentation:**
- ECOSYSTEM.md constraint kn-8d192d - All Go CLIs must embed git commit hash

**Related Artifacts:**
- **Investigation:** .kb/investigations/2026-01-23-inv-auto-rebuild-go-binaries-during.md - orch complete auto-rebuild implementation
- **Investigation:** .kb/investigations/2025-12-24-inv-auto-rebuild-after-go-changes.md - Original auto-rebuild investigation

---

## Investigation History

**2026-01-28 19:15:** Investigation started
- Initial question: What rebuild-on-change mechanisms exist across ecosystem?
- Context: Need comprehensive audit to identify gaps in staleness detection and auto-rebuild

**2026-01-28 19:30:** Batch audit completed
- Systematically tested all 10 repos for hooks, version commands, build systems
- Identified 3 distinct rebuild mechanisms and multiple gaps

**2026-01-28 19:50:** Matrix and recommendations completed
- Created detailed matrix of all repos with recommendations prioritized
- Status: Complete
- Key outcome: 40% have auto-rebuild, 33% lack staleness detection (glass, agentlog critical gaps)
# Rebuild Mechanism Matrix

## Overview Matrix

| Repo | Artifacts | Post-Commit Hook | Version w/ Git Hash | `orch complete` Auto-Rebuild | Manual Rebuild | Failure Mode |
|------|-----------|------------------|---------------------|------------------------------|----------------|--------------|
| **orch-go** | `~/bin/orch` binary | ✅ YES (detects cmd\|pkg/*.go, runs `make install`) | ✅ YES (`orch version eedc8991`) | ✅ YES (rebuilds before verification) | `make install` | Daemon uses stale binary until restart |
| **kb-cli** | `~/bin/kb` binary | ✅ YES (detects *.go, runs `make build`) | ✅ YES (`kb version 5e52def-dirty`) | ✅ YES (cross-project support) | `make install` | Agents use stale kb commands |
| **beads** | `~/bin/bd` binary | ❌ NO | ✅ YES (`bd version 0.41.0 (629441ad)`) | ⚠️ PARTIAL (not owned by Dylan) | `make install` | "Fixed" bugs still reproduce |
| **glass** | `~/bin/glass` binary | ❌ NO | ❌ NO (no version command) | ❌ NO | `make install` | No staleness detection at all |
| **skillc** | `~/bin/skillc` binary | ❌ NO | ✅ YES (`skillc a8b8b25-dirty`) | ❌ NO | `make install` | Skills compiled with stale skillc |
| **agentlog** | `~/bin/agentlog` binary | ✅ YES (runs `go install` unconditionally) | ❌ NO (no version command) | ⚠️ PARTIAL | `go install ./cmd/agentlog` | No staleness check possible |
| **kn** | `~/bin/kn` binary | ✅ YES (detects *.go, runs `go build`) | ⚠️ UNKNOWN (binary not found) | ❌ NO | `go build -o kn ./cmd/kn` | Unknown |
| **orch-cli** | Python (no binary) | ❌ NO | N/A (Python) | N/A | N/A | Python imports latest code |
| **orch-knowledge** | Skills (via skillc) | ⚠️ PRE-COMMIT (bd sync) | N/A (no binary) | N/A | `skillc deploy` | Stale skills if not recompiled |
| **beads-ui-svelte** | Vite dev server | ❌ NO | N/A (JS) | N/A | `npm run build` | Dev server auto-reloads |

## Detailed Findings by Repo

### orch-go
- **Post-commit hook**: `.git/hooks/post-commit` detects `cmd|pkg/*.go` changes, runs `make install`
- **Version command**: `orch version` shows git hash + build time + source dir
- **Auto-rebuild**: `orch complete` has `rebuildGoProjectsIfNeeded()` that rebuilds BEFORE verification
- **Staleness check**: `orch doctor --stale-only` compares binary git hash to repo HEAD
- **Failure mode**: Daemon continues using old binary until `launchctl kickstart` restart

### kb-cli
- **Post-commit hook**: `.git/hooks/post-commit` detects `*.go` changes, runs `make build`
- **Version command**: `kb version` shows git hash with dirty flag
- **Auto-rebuild**: Supported by `orch complete` cross-project logic
- **Manual**: `make install` or `make build`
- **Failure mode**: Agents use stale kb commands if binary not rebuilt

### beads
- **Post-commit hook**: ❌ None (upstream OSS, Dylan doesn't modify)
- **Version command**: ✅ `bd version 0.41.0 (629441ad: main@629441adda8c)`
- **Auto-rebuild**: Not applicable (upstream project)
- **Manual**: `make install` when building from source
- **Failure mode**: Rare (Dylan uses upstream releases)

### glass
- **Post-commit hook**: ❌ None
- **Version command**: ❌ No version command exists (errors with "unknown command")
- **Auto-rebuild**: ❌ None
- **Manual**: `make install`
- **Failure mode**: No way to detect if binary is stale without manual git comparison

### skillc
- **Post-commit hook**: ❌ None
- **Version command**: ✅ `skillc a8b8b25-dirty (commit: ..., built: 2026-01-18T20:39:31Z)`
- **Auto-rebuild**: Has its own mechanism that triggers on version check (race condition observed)
- **Manual**: `make install`
- **Failure mode**: Skills compiled with stale skillc binary produce incorrect output

### agentlog
- **Post-commit hook**: ✅ Runs `go install ./cmd/agentlog` unconditionally
- **Version command**: ❌ No version command (errors with "unknown command")
- **Auto-rebuild**: Hooks rebuild, but no staleness detection
- **Manual**: `go install ./cmd/agentlog`
- **Failure mode**: Can't verify if binary matches source

### kn
- **Post-commit hook**: ✅ Detects `*.go` changes, runs `go build -o kn ./cmd/kn`
- **Version command**: ⚠️ Unknown (binary not in PATH during testing)
- **Auto-rebuild**: Hook handles it
- **Manual**: `go build -o kn ./cmd/kn`

### Non-Binary Repos
- **orch-cli**: Python, no build step
- **orch-knowledge**: Skills compiled via `skillc deploy`, has pre-commit hook for `bd sync`
- **beads-ui-svelte**: Vite dev server auto-reloads, production build via `npm run build`

## Gap Analysis

### Critical Gaps (High Impact)
1. **glass**: No version command + no post-commit hook = zero staleness detection
2. **agentlog**: No version command = can't verify binary matches source
3. **skillc**: No post-commit hook = stale skillc compiles skills incorrectly

### Medium Impact Gaps
4. **beads**: No post-commit hook (but upstream OSS, lower priority)
5. **orch-knowledge skills**: Depend on manual `skillc deploy` after skill source changes

### Low Impact
6. **kn**: Has hook but binary location unclear
7. **beads-ui-svelte**: Dev workflow has auto-reload (production build is manual but infrequent)

## Recommendations

### Immediate Actions (High Priority)
1. **Add version command to glass**: Embed git hash via ldflags
2. **Add version command to agentlog**: Same pattern as orch/kb/skillc
3. **Add post-commit hook to skillc**: Auto-rebuild when Go files change

### Medium Priority
4. **Add post-commit hook to glass**: Auto-rebuild on Go file changes
5. **Document skillc auto-rebuild mechanism**: Investigate race condition, document behavior
6. **Consider automation for orch-knowledge skills**: Watch skill sources, auto-run `skillc deploy`

### Low Priority / Consider
7. **Standardize hook patterns**: All 4 repos with hooks have slightly different implementations
8. **Central hook management**: Consider shared hook scripts for consistency
9. **Automated staleness checks**: Expand `orch doctor` to check all ecosystem binaries

## Rebuild Mechanism Types

### Type 1: Post-Commit Hooks (40% coverage)
- **Repos**: orch-go, kb-cli, agentlog, kn
- **Pattern**: `.git/hooks/post-commit` detects file changes, triggers rebuild
- **Pros**: Automatic, no discipline required
- **Cons**: Per-repo setup, slight variation in implementation

### Type 2: orch complete Auto-Rebuild (Agent workflow only)
- **Repos**: orch-go, kb-cli (cross-project)
- **Pattern**: `rebuildGoProjectsIfNeeded()` in complete_cmd.go
- **Pros**: Ensures verification runs against fresh code
- **Cons**: Only triggers during agent completion, not manual CLI usage

### Type 3: Manual Discipline (60% of repos)
- **Repos**: beads, glass, skillc, orch-cli, orch-knowledge, beads-ui-svelte
- **Pattern**: Developer runs `make install` after committing
- **Pros**: Simple, no automation needed
- **Cons**: Easy to forget, causes "fixed bug still reproduces" friction

### Type 4: On-Demand (skillc special case)
- **Repos**: skillc
- **Pattern**: Auto-rebuild triggered on version check
- **Pros**: Lazy rebuild, only when needed
- **Cons**: Race conditions observed, non-standard pattern

## Failure Mode Analysis

| Failure Scenario | Affected Repos | Frequency | Impact | Current Mitigation |
|------------------|----------------|-----------|--------|-------------------|
| Code committed but binary not rebuilt | glass, skillc, beads | High | Medium | Developer discipline / manual check |
| Daemon using stale binary | orch-go | Medium | High | `make install-restart` reminder in Makefile |
| No staleness detection | glass, agentlog | Low | Medium | None (manual git hash comparison only) |
| Agent uses stale CLI | kb-cli (pre-hook) | Low (now) | High | Post-commit hook added 2026-01-23 |
| Skills compiled with stale skillc | skillc | Medium | Medium | Manual `make install` before `skillc deploy` |
