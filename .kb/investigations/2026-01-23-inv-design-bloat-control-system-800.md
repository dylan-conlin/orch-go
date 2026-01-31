<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Bloat control should use CI gate + spawn-time context injection, not beads hooks or pre-commit hooks.

**Evidence:** Beads corruption investigation shows hooks that interact with beads in complex ways cause database corruption; 42 files currently over 800 lines with 12 critical files over 1500 lines; `orch hotspot` already provides detection but lacks enforcement gates.

**Knowledge:** Gate Over Remind principle requires enforcement gates (not warnings); beads corruption lesson requires minimal hook complexity; spawn-time context injection surfaces issues without blocking; CI gate provides true enforcement outside sandbox.

**Next:** Implement spawn-time bloat context injection in pkg/spawn/context.go, then add GitHub Actions bloat check workflow.

**Promote to Decision:** Issue created: orch-go-21084 (bloat control decision)

---

# Investigation: Design Bloat Control System for 800-Line Gate Enforcement

**Question:** How should we enforce the 800-line bloat gate across the agent workflow, applying Gate Over Remind principle while minimizing beads corruption risk?

**Started:** 2026-01-23
**Updated:** 2026-01-23
**Owner:** Architect agent
**Phase:** Complete
**Next Step:** Implement spawn-time bloat injection + CI gate
**Status:** Complete

**Patches-Decision:** N/A
**Extracted-From:** N/A
**Supersedes:** N/A
**Superseded-By:** N/A

---

## Context: Current State

**Bloat inventory (from `find | wc -l`):**
- 42 files over 800 lines
- 12 critical files over 1500 lines (spawn_cmd.go: 2630, daemon_test.go: 2696, doctor.go: 2165, etc.)

**Existing infrastructure:**
- `orch hotspot --bloat-threshold 800` detects bloated files with severity-based recommendations
- `.kb/models/extract-patterns.md` defines 800-line gate rationale (Context Noise threshold)
- `.kb/guides/code-extraction-patterns.md` provides extraction workflow

**Gap:** Detection exists, but no enforcement gates to prevent bloat accumulation.

---

## Findings

### Finding 1: Gate Over Remind principle requires enforcement, not warnings

**Evidence:** From `~/.kb/principles.md:162-189`:
- "Reminders fail under cognitive load. When deep in a complex problem, 'remember to update kn' gets crowded out. Gates make capture unavoidable."
- "Gates must be passable by the gated party" - a valid gate is one the agent can satisfy by their own work

**Source:** `~/.kb/principles.md:162-189`

**Significance:** Current bloat detection (`orch hotspot`) is a reminder system - agents can ignore it. The principle says we need a gate that blocks progress. However, the gate must be passable - agents must be able to fix the bloat themselves.

---

### Finding 2: Beads corruption was caused by complex hooks in sandbox environment

**Evidence:** From `.kb/investigations/2026-01-21-inv-investigate-beads-sqlite-database-corruption.md`:
- 57 daemon start attempts on corruption day, each opening/closing database
- Root cause: "Sandbox environment cannot run beads daemon (chmod fails), but auto-start keeps trying"
- Recommendation: "Minimize hook complexity - hooks that interact with beads in complex ways are risky"

**Source:** `.kb/investigations/2026-01-21-inv-investigate-beads-sqlite-database-corruption.md`

**Significance:** Adding beads-based hooks for bloat control would risk the same corruption pattern. The lesson is clear: hooks should be simple, should not interact with beads in complex ways, and should fail gracefully without blocking operations.

---

### Finding 3: CI gates run outside sandbox, avoiding corruption risk

**Evidence:**
- CI workflows (GitHub Actions) run on GitHub infrastructure, not in Claude Code sandbox
- CI can access the full codebase for analysis
- CI gates cannot be bypassed (unlike `--no-verify` for pre-commit hooks)
- CI is the standard enforcement point for code quality gates in software projects

**Source:** Industry standard practice; `.kb/guides/resilient-infrastructure-patterns.md` documents escape hatches and secondary paths

**Significance:** CI is the appropriate layer for true enforcement gates. It's outside the sandbox, runs reliably, and can't be bypassed by agents or developers.

---

### Finding 4: Spawn-time context injection surfaces issues at the right moment

**Evidence:**
- `pkg/spawn/context.go` already generates SPAWN_CONTEXT.md with task context
- Spawned agents read SPAWN_CONTEXT.md as their primary context source
- The spawn context already includes skill guidance, beads issue context, and kb priming

**Source:** `pkg/spawn/context.go`, existing spawn context template structure

**Significance:** Adding bloat warnings to spawn context surfaces the issue when context exists (Capture at Context principle). The agent sees the warning as they start work, not after they've already written code. This is surfacing, not gating - but surfacing at the right moment is more effective than gating after the fact.

---

### Finding 5: Issue-level gates add complexity without enforcement

**Evidence:**
- Most beads issues are task-level ("implement X"), not file-level ("modify file Y")
- Auto-creating extraction issues would add noise to the issue tracker
- Warning on issue creation would require determining which files will be touched before work starts (often unknown)

**Source:** Observation of current beads usage patterns; `bd list` shows task-level issue descriptions

**Significance:** Issue-level gates don't map well to file-level concerns like bloat. They would add complexity to beads operations (contra corruption lesson) without providing effective enforcement.

---

## Synthesis

**Key Insights:**

1. **Two-layer enforcement** - Surfacing (spawn-time) + Gating (CI) provides the right combination. Surfacing at context ensures agents know about bloat before they exacerbate it. Gating at CI ensures bloat can't merge.

2. **Sandbox isolation** - Any enforcement that runs inside Claude Code sandbox should be minimal. Complex logic, beads interactions, and blocking operations belong outside the sandbox (CI layer).

3. **Pre-commit hooks are risky** - Pre-commit hooks run in sandbox, can be bypassed with `--no-verify`, and add complexity to the commit flow. They're also slow for large file analysis. CI is more reliable.

4. **Detection already exists** - `orch hotspot` provides the detection logic. The design challenge is enforcement, not detection. Don't duplicate detection infrastructure.

**Answer to Investigation Question:**

Bloat control should use a **two-layer approach**:

1. **Spawn-time context injection** (surfacing) - When spawning to work on a file over 800 lines, inject a bloat warning into SPAWN_CONTEXT.md with extraction recommendations. This surfaces the issue when context exists but doesn't block.

2. **CI gate** (enforcement) - GitHub Actions workflow that fails the build if any modified file exceeds 800 lines, or if total project bloat increases. This is the true gate that can't be bypassed.

This approach applies Gate Over Remind (CI gate provides enforcement), Capture at Context (spawn-time surfacing), and minimizes beads corruption risk (no complex hooks).

---

## Structured Uncertainty

**What's tested:**

- ✅ 42 files currently over 800 lines (verified: `find | wc -l | awk '$1 > 800'`)
- ✅ Beads corruption was caused by hook complexity in sandbox (verified: read corruption investigation)
- ✅ Gate Over Remind principle requires gates not reminders (verified: read principles.md)
- ✅ `orch hotspot` already detects bloated files (verified: hotspot.go source review)
- ✅ spawn context generation is in pkg/spawn/context.go (verified: source review)

**What's untested:**

- ⚠️ Performance impact of bloat check in CI (not benchmarked)
- ⚠️ Whether spawn-time warning actually influences agent behavior (would need before/after study)
- ⚠️ Edge cases for CI bloat detection (renamed files, deleted files, test files)

**What would change this:**

- If CI gates are too slow, might need to optimize or threshold differently
- If agents consistently ignore spawn-time warnings, might need blocking gate at spawn
- If beads hooks can be made safe (no database interaction), issue-level gates might be viable

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation.

### Recommended Approach ⭐

**Two-Layer Bloat Enforcement: Spawn-Time Context + CI Gate**

**Why this approach:**
- Applies Gate Over Remind: CI gate provides true enforcement that can't be bypassed
- Respects beads corruption lesson: No complex hooks or beads interactions in sandbox
- Applies Capture at Context: Spawn-time surfacing when agent context exists
- Builds on existing infrastructure: Uses `orch hotspot` detection, `pkg/spawn/context.go` for injection

**Trade-offs accepted:**
- Spawn-time is surfacing not gating (agent can still proceed to bloated file)
- CI gate catches after local development (some wasted work possible)
- Requires GitHub Actions (not enforced locally for non-CI workflows)

**Implementation sequence:**
1. **Spawn-time bloat injection** - Modify `pkg/spawn/context.go` to check `orch hotspot` for file bloat and inject warning if over threshold
2. **CI workflow** - Create `.github/workflows/bloat-check.yml` that fails if modified files exceed 800 lines
3. **Documentation** - Update CLAUDE.md to reference bloat enforcement

### Alternative Approaches Considered

**Option B: Pre-commit hook for bloat**
- **Pros:** Catches bloat at commit time, before push
- **Cons:** Runs in sandbox (corruption risk), can be bypassed with `--no-verify`, slow for large files
- **When to use instead:** If CI is too slow or unavailable

**Option C: Issue-level gates (warn on creation, auto-create extraction issues)**
- **Pros:** Tracks bloat work as beads issues
- **Cons:** Most issues aren't file-specific, adds noise, complex beads interaction (corruption risk)
- **When to use instead:** Never - the mapping between issues and files is too loose

**Option D: Spawn-time blocking gate**
- **Pros:** Prevents spawning to work on bloated files
- **Cons:** Agents can't fix existing bloat if blocked from touching the file
- **When to use instead:** If spawn-time warnings are consistently ignored (escalation)

**Rationale for recommendation:** Option A (CI gate + spawn surfacing) provides true enforcement at CI layer (Finding 3), surfaces at the right moment (Finding 4), and avoids beads corruption risk (Finding 2). It's the only option that applies all three relevant principles correctly.

---

### Implementation Details

**What to implement first:**
- Spawn-time bloat context injection (lower risk, faster to implement)
- Then CI workflow (requires GitHub Actions setup)

**Things to watch out for:**
- ⚠️ Spawn-time injection should be fast - cache hotspot results or do quick line count
- ⚠️ CI workflow needs to handle test files specially (they're expected to be long)
- ⚠️ Need to distinguish "file was already bloated" from "this PR made it bloated"

**Areas needing further investigation:**
- Whether to allow PR-level exceptions for intentional large files
- How to handle vendored/generated code in bloat check
- Whether severity thresholds (800 warn, 1500 block) should differ between layers

**Success criteria:**
- ✅ Spawned agents see bloat warning when target file is over 800 lines
- ✅ PRs that increase file past 800 lines fail CI
- ✅ No beads corruption incidents after implementation
- ✅ Over 6-month period, total bloated file count decreases

---

## References

**Files Examined:**
- `~/.kb/principles.md` - Gate Over Remind, Capture at Context principles
- `.kb/investigations/2026-01-21-inv-investigate-beads-sqlite-database-corruption.md` - Beads corruption root cause
- `.kb/models/extract-patterns.md` - 800-line gate rationale
- `.kb/guides/code-extraction-patterns.md` - Extraction workflow
- `cmd/orch/hotspot.go` - Bloat detection implementation
- `pkg/spawn/context.go` - Spawn context generation

**Commands Run:**
```bash
# Count bloated files
find . -name "*.go" -o -name "*.svelte" -o -name "*.ts" | grep -v node_modules | xargs wc -l | sort -rn | awk '$1 > 800'

# Count critical files (>1500 lines)
find . -name "*.go" -o -name "*.svelte" -o -name "*.ts" | grep -v node_modules | xargs wc -l | sort -rn | awk '$1 > 1500'

# Check beads hooks
cat .beads/hooks/on_close
```

**External Documentation:**
- GitHub Actions workflow syntax - for CI gate implementation

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2026-01-17-inv-design-800-line-bloat-gate.md` - Prior design for `orch hotspot` bloat-size type
- **Investigation:** `.kb/investigations/2026-01-17-inv-implement-bloat-size-hotspot-type.md` - Implementation of bloat detection
- **Model:** `.kb/models/extract-patterns.md` - 800-line gate rationale and extraction patterns

---

## Investigation History

**2026-01-23 13:00:** Investigation started
- Initial question: How to enforce 800-line bloat gate with Gate Over Remind principle while respecting beads corruption constraints
- Context: 42 files over 800 lines, 12 critical >1500; need enforcement not just detection

**2026-01-23 13:30:** Findings complete
- Key insight: CI gate + spawn-time surfacing provides enforcement without beads corruption risk
- Rejected pre-commit hooks (sandbox risk, bypassable) and issue-level gates (bad mapping)

**2026-01-23 13:45:** Investigation completed
- Status: Complete
- Key outcome: Two-layer approach - spawn-time context injection for surfacing, CI gate for enforcement
