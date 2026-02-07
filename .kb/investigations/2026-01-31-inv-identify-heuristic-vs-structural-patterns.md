<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
-->

## Summary (D.E.K.N.)

**Delta:** The ecosystem has extensive documented rules but most enforcement remains heuristic; only 5 of 15+ documented rules have active structural enforcement.

**Evidence:** Examined opencode.json (3 active permissions), spawn_validation.go (4 gates), 12 plugins (only 3 active), principles.md (27 principles), and 6 key guides/decisions.

**Knowledge:** The system explicitly documents the need for structural enforcement (Gate Over Remind, Infrastructure Over Instruction) but hasn't achieved it - most rules rely on agent compliance with documented guidance.

**Next:** Architectural decision needed on which heuristic patterns to structurally enforce vs accept as guidance.

**Authority:** strategic - Involves resource allocation (plugin development) and irreversible system design choices about enforcement philosophy.

---

# Investigation: Identify Heuristic Vs Structural Patterns

**Question:** What patterns across the ecosystem are structural (enforced via code/infrastructure) vs heuristic (documented guidance that relies on agent compliance)?

**Started:** 2026-01-31
**Updated:** 2026-01-31
**Owner:** Worker agent (orch-go-21135)
**Phase:** Complete
**Next Step:** None
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Patches-Decision:** N/A
**Extracted-From:** N/A
**Supersedes:** N/A
**Superseded-By:** N/A

---

## Findings

### Finding 1: Active Structural Enforcement Mechanisms (5 total)

**Evidence:**

1. **opencode.json permissions** (ACTIVE)
   - `permission.task: deny` - Disables Task tool for this project
   - `gpt_orchestrator.permission.*` - Constrains GPT orchestrator's tool access

2. **Triage Bypass Gate** (ACTIVE)
   - Manual spawns require `--bypass-triage` flag
   - Logged to `~/.orch/events.jsonl` for tracking bypass patterns

3. **Gap Gate** (ACTIVE)
   - Context quality below threshold blocks spawn
   - `orch spawn --skip-gap-gate` to override

4. **Workspace Exists Check** (ACTIVE)
   - Prevents overwriting existing session artifacts
   - Requires `--force` to override

5. **Beads on_close Hook** (ACTIVE)
   - Emits `agent.completed` event when issues closed via `bd close`
   - Closes tracking gap for work bypassing `orch complete`

**Source:**
- `.opencode/opencode.json:5` - Task tool deny
- `cmd/orch/spawn_validation.go:195-217` - Triage bypass gate
- `cmd/orch/spawn_validation.go:119-155` - Gap gating
- `cmd/orch/spawn_validation.go:299-326` - Workspace exists check
- `.beads/hooks/on_close` - Event emission hook

**Significance:** These are true gates - spawn fails or behavior is blocked without explicit override flag. Override patterns are logged, creating accountability.

---

### Finding 2: Disabled or Superseded Enforcement (4 total)

**Evidence:**

1. **Decision Gate** (SUPERSEDED)
   - Pre-spawn check for decision conflicts with `blocks` frontmatter
   - Status changed to superseded Jan 30, 2026 due to "high false positive rate"
   - Code still exists in `spawn_validation.go:399-459`

2. **bd-close-gate Plugin** (DISABLED)
   - Was intended to prevent workers from running `bd close`
   - File: `~/.config/opencode/plugin/` shows symlink to `coaching.ts` (disabled)

3. **Coaching Plugin** (DISABLED)
   - Behavioral loop/thrashing detection
   - File: `orch-go/plugins/coaching.ts.disabled` (67KB)

4. **Task Tool Gate Plugin** (DISABLED)
   - Alternative to opencode.json permission
   - File: `orch-go/plugins/task-tool-gate.ts.disabled`

**Source:**
- `.kb/decisions/2026-01-28-decision-gate.md:1-4` - Superseded status
- `~/.config/opencode/plugin/` directory listing (ls -la output)
- `orch-go/plugins/` directory listing

**Significance:** Multiple attempts at structural enforcement have been disabled. Pattern suggests enforcement is harder to get right than documentation - false positives create friction that leads to disabling gates entirely.

---

### Finding 3: Active Context Injection Plugins (Soft Enforcement)

**Evidence:**

1. **guarded-files.ts** (ACTIVE)
   - Injects protocol warning when editing protected files
   - Does NOT block - agent can proceed but sees warning

2. **session-compaction.ts** (ACTIVE)
   - Preserves critical context during OpenCode compaction
   - Injects tier, beads ID, constraints, completion protocol

3. **evidence-hierarchy.ts** (ACTIVE in orch-go/plugins/)
   - Warns when editing without prior search/read
   - Does NOT block - warning only

**Source:**
- `~/.config/opencode/plugin/guarded-files.ts:42-95`
- `~/.config/opencode/plugin/session-compaction.ts:273-299`
- `orch-go/plugins/evidence-hierarchy.ts:202-303`

**Significance:** These implement "surfacing" not "gating" - they make agents aware but don't prevent action. This is intermediate between pure heuristic (documentation) and pure structural (hard gate).

---

### Finding 4: Extensive Heuristic-Only Rules

**Evidence:**

| Documented Rule | Enforcement | Gap |
|-----------------|-------------|-----|
| Workers can't add dependencies | Decision frontmatter only | `bd dep add` works |
| Workers can't close issues | Disabled plugin | `bd close` works |
| Workers must create SYNTHESIS.md | Spawn instruction only | No verification |
| Constitutional hard limits | Skill documentation | No gate |
| Phase reporting via bd comment | Instruction only | Not checked |
| Investigation requires testing | Self-review checklist | Not enforced |
| Bloat control (800 lines) | Decision exists | Not implemented |
| Leave it Better requirement | Skill instruction | Not verified |

**Source:**
- `.kb/decisions/2026-01-19-worker-authority-boundaries.md:39-43` - Worker constraints
- `.kb/decisions/2026-01-22-skill-constitutional-constraints.md` - Hard limits
- `.kb/decisions/2026-01-30-bloat-control-enforcement-patterns.md:14-18` - Bloat gate planned
- Worker-base skill instructions (loaded in SPAWN_CONTEXT)

**Significance:** The majority of documented rules rely entirely on agent compliance. The system has detailed documentation about what agents *should* do but minimal infrastructure ensuring they *do* it.

---

### Finding 5: Principles Acknowledge the Gap

**Evidence:**

The principles.md explicitly addresses this enforcement gap:

1. **Gate Over Remind** (lines 162-189): "Reminders fail under cognitive load. Gates make capture unavoidable."

2. **Infrastructure Over Instruction** (lines 293-308): "Policy is Code: Critical protocols must be implemented as gates, surfacing mechanisms, or manifests, not just documented in skills."

3. **Track Actions, Not Just State** (lines 231-258): "Knowledge of correct behavior doesn't prevent incorrect behavior."

4. **Pain as Signal** (lines 265-283): "Friction that is only observed by the human is wasted signal for the agent."

**Source:** `~/.kb/principles.md:162-308`

**Significance:** The system is self-aware about this gap. These principles explicitly call for structural enforcement but the implementation hasn't caught up. This is a known problem, not a blind spot.

---

## Synthesis

**Key Insights:**

1. **Enforcement Pyramid** - The ecosystem has three enforcement tiers: (1) Hard gates (5 active - spawn fails without override), (2) Soft surfacing (3 active plugins - warnings without blocking), (3) Pure documentation (15+ rules - agent compliance only). Most rules sit at the bottom tier.

2. **Self-Aware Gap** - The principles explicitly document the need for structural enforcement (Gate Over Remind, Infrastructure Over Instruction) but implementation hasn't followed. This isn't a blind spot - it's a known backlog.

3. **Enforcement Fragility** - Multiple enforcement attempts have been disabled (decision gate, coaching plugin, bd-close gate, task-tool gate). False positives create friction that leads to disabling gates entirely. This suggests enforcement is harder to calibrate than documentation.

4. **Override as Signal** - The active gates use override flags (`--bypass-triage`, `--skip-gap-gate`, `--force`, `--acknowledge-decision`) with logging. This creates accountability without brittleness - gates can be overridden but the override is visible.

**Answer to Investigation Question:**

The ecosystem has **~5 structural enforcement patterns** (hard gates that block without explicit override) vs **~15+ heuristic patterns** (documented rules relying on agent compliance). An additional **~3 patterns** occupy a middle ground (context injection plugins that surface information without blocking).

The ratio is approximately **1:3 (structural:heuristic)** - for every structurally enforced rule, there are roughly three documented-only rules.

Key structural patterns: opencode.json permissions, triage bypass gate, gap gate, workspace exists check, beads on_close hook.

Key heuristic patterns: worker authority boundaries, constitutional constraints, SYNTHESIS.md requirement, phase reporting, investigation testing requirement, bloat control.

---

## Structured Uncertainty

**What's tested:**

- ✅ opencode.json `permission.task: deny` is active (verified: file read shows this config)
- ✅ Triage bypass gate exists (verified: read spawn_validation.go:195-217)
- ✅ 3 plugins are active in ~/.config/opencode/plugin/ (verified: ls -la showed non-.disabled files)
- ✅ 4 plugins are disabled in orch-go/plugins/ (verified: .disabled extension present)
- ✅ Decision gate marked superseded (verified: frontmatter in decision file)
- ✅ Beads on_close hook exists (verified: read .beads/hooks/on_close)
- ✅ Worker authority boundaries are documentation-only (verified: no `bd dep add` gate in code)

**What's untested:**

- ⚠️ Whether evidence-hierarchy plugin is actually loaded (symlink may not be configured)
- ⚠️ Gap gate effectiveness (don't know false positive/negative rate)
- ⚠️ Whether disabled plugins would work if re-enabled (may have bitrot)
- ⚠️ Cross-project patterns (only examined orch-go, not unified-kb or orch-knowledge)

**What would change this:**

- If opencode.json permissions were being bypassed somehow, structural count would decrease
- If disabled plugins were actually loaded via alternative path, soft enforcement count would increase
- If worker authority boundaries had shell-level enforcement I missed, heuristic count would decrease

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Decide which heuristic patterns to structurally enforce | **strategic** | Involves resource allocation (plugin development) and irreversible design choices |
| Categorize patterns into tiers | **architectural** | Cross-component impact, orchestrator synthesis required |
| Document current state in model file | **implementation** | Knowledge capture within existing patterns |

### Recommended Approach ⭐

**Tiered Enforcement Model** - Accept that not all rules need structural enforcement; explicitly categorize patterns into tiers with different enforcement strategies.

**Why this approach:**
- Acknowledges enforcement is expensive to calibrate (Finding 2: disabled gates due to false positives)
- Aligns with "override as signal" pattern that works (Finding 5: active gates use logged overrides)
- Avoids false binary of "gate everything" vs "document everything"

**Trade-offs accepted:**
- Some rules remain heuristic-only (accepting agent non-compliance risk)
- Enforcement effort focused on highest-impact patterns first
- Soft surfacing (context injection) used as intermediate tier

**Implementation sequence:**
1. **Document current state** - Create `.kb/models/enforcement-patterns.md` with current categorization
2. **Prioritize enforcement candidates** - Identify highest-impact heuristic patterns (e.g., bd close restriction)
3. **Prototype new gates with extensive testing** - Avoid false positive problem that killed decision gate

### Alternative Approaches Considered

**Option B: Gate Everything**
- **Pros:** Maximum enforcement, aligns with Gate Over Remind principle
- **Cons:** False positive problem (Finding 2); enforcement fragility means gates get disabled
- **When to use instead:** Never as blanket approach; selective gating is better

**Option C: Accept Documentation-Only**
- **Pros:** Minimal development effort, no false positive risk
- **Cons:** Ignores explicit principle guidance (Gate Over Remind, Infrastructure Over Instruction)
- **When to use instead:** For low-impact rules where enforcement cost exceeds non-compliance cost

**Rationale for recommendation:** Finding 5 shows the system already knows it needs structural enforcement but hasn't achieved it. Finding 2 shows why - enforcement is hard to calibrate. The tiered model provides a framework for selective enforcement based on impact/cost analysis.

---

### Implementation Details

**What to implement first:**
- Create enforcement patterns model documenting current state
- Identify 2-3 highest-impact heuristic patterns for enforcement prototype
- Re-examine disabled gates (coaching, bd-close) for revival with better calibration

**Things to watch out for:**
- ⚠️ False positive rate - decision gate was superseded due to this
- ⚠️ Plugin loading complexity - disabled plugins may have path/config issues
- ⚠️ Cross-project consistency - orch-go patterns may not match other projects

**Areas needing further investigation:**
- Why exactly was decision gate superseded? What false positives occurred?
- Could coaching plugin be revived with better thresholds?
- What enforcement patterns exist in unified-kb or orch-knowledge?

**Success criteria:**
- ✅ Model file documents all patterns with enforcement tier
- ✅ 2-3 high-impact gates prototyped without false positive problems
- ✅ Clear criteria for when to gate vs document vs inject

---

## References

**Files Examined:**
- `.opencode/opencode.json` - Permission configuration for this project
- `cmd/orch/spawn_validation.go` - All pre-spawn gates and validation logic
- `~/.config/opencode/plugin/` directory - Active vs disabled plugins
- `orch-go/plugins/` directory - Project-level plugins (mostly disabled)
- `.beads/hooks/on_close` - Beads event emission hook
- `~/.kb/principles.md` - Foundation principles including enforcement guidance
- `.kb/decisions/2026-01-28-decision-gate.md` - Superseded decision gate
- `.kb/decisions/2026-01-30-bloat-control-enforcement-patterns.md` - Planned enforcement
- `.kb/decisions/2026-01-19-worker-authority-boundaries.md` - Worker constraints (heuristic)
- `.kb/decisions/2026-01-22-skill-constitutional-constraints.md` - Constitutional limits (heuristic)
- `.kb/guides/opencode-plugins.md` - Plugin system architecture
- `.kb/guides/decision-authority.md` - Authority delegation rules (heuristic)
- `~/.config/opencode/plugin/guarded-files.ts` - Active context injection plugin
- `~/.config/opencode/plugin/session-compaction.ts` - Active compaction plugin
- `orch-go/plugins/evidence-hierarchy.ts` - Evidence hierarchy enforcement

**Commands Run:**
```bash
# List OpenCode plugins
ls -la ~/.config/opencode/plugin/

# List orch-go plugins
ls -la orch-go/plugins/

# Verify project directory
pwd
```

**External Documentation:**
- N/A

**Related Artifacts:**
- **Decision:** `.kb/decisions/2026-01-28-decision-gate.md` - Superseded enforcement attempt
- **Decision:** `.kb/decisions/2026-01-30-bloat-control-enforcement-patterns.md` - Planned but unimplemented enforcement
- **Guide:** `.kb/guides/opencode-plugins.md` - Plugin system for enforcement
- **Principles:** `~/.kb/principles.md` - Gate Over Remind, Infrastructure Over Instruction

---

## Investigation History

**2026-01-31 11:45:** Investigation started
- Initial question: What patterns are structural vs heuristic across the ecosystem?
- Context: Orchestrator task to identify enforcement gaps

**2026-01-31 11:50:** Found core structural enforcement mechanisms
- Identified opencode.json permissions, spawn_validation.go gates, beads hooks
- 5 active structural mechanisms total

**2026-01-31 11:55:** Discovered disabled enforcement attempts
- 4 plugins/gates disabled or superseded
- Pattern: false positives lead to disabling gates entirely

**2026-01-31 12:00:** Mapped heuristic-only rules
- 15+ documented rules with no structural enforcement
- Includes worker authority, constitutional constraints, SYNTHESIS.md requirement

**2026-01-31 12:05:** Connected to principles
- Gate Over Remind and Infrastructure Over Instruction explicitly call for enforcement
- System is self-aware about the gap

**2026-01-31 12:10:** Investigation completed
- Status: Complete
- Key outcome: ~1:3 ratio of structural:heuristic patterns; system knows it needs more enforcement but hasn't achieved it
