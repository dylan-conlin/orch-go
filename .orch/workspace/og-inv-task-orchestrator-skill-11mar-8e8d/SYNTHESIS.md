# Session Synthesis

**Agent:** og-inv-task-orchestrator-skill-11mar-8e8d
**Issue:** N/A (ad-hoc --no-track)
**Duration:** 2026-03-11
**Outcome:** success

---

## Plain-Language Summary

The orchestrator skill has undergone a dramatic transformation from a 2,368-line constraint-heavy document (Mar 1) to a 486-line knowledge-transfer document (current). This matches the design intent from 6 investigations spanning Jan 18 - Mar 5, 2026. Of 25+ specific recommendations across investigations 2-5, approximately 22 are fully implemented, 2 are partially implemented, and 1 critical gate remains open: the v4 skill was deployed without passing the `skillc test` bare-parity behavioral regression check (blocked because `skillc test` can't run from spawned agents due to CLAUDECODE env var). The hook infrastructure that replaced prompt-level constraints is robust — 6 of 7 hooks are registered, tested, and working. The main risk is that the 82% token reduction is validated structurally but not behaviorally: we don't know if v4 actually performs better than bare Claude on the scenarios where the old skill was supposed to help.

## Verification Contract

See VERIFICATION_SPEC.yaml for verification details. Key outcomes: all investigations read, current skill and hooks verified against recommendations, 4-section structured audit complete.

---

## TLDR

Audited the orchestrator skill against 6 investigations. 22 of 25 recommendations implemented, 6/7 hooks working, token count reduced 82% (27,200→5,995). One critical gate remains open: behavioral validation via `skillc test` has never been completed due to spawned-agent environment limitations.

---

## Delta (What Changed)

### Files Created
- `.kb/models/orchestrator-session-lifecycle/probes/2026-03-11-probe-orchestrator-skill-current-state-audit.md` — Probe documenting audit findings
- `.orch/workspace/og-inv-task-orchestrator-skill-11mar-8e8d/SYNTHESIS.md` — This file

---

## Evidence (What Was Observed)

# 1. RECOMMENDATION TRACKING

## Investigation 1 (Jan 18): Frustration Trigger Protocol

| Recommendation | Status | Evidence |
|---|---|---|
| Add frustration trigger to Fast Path table | **implemented** | Deployed skill line 72: "Dylan voices frustration → STOP tactical fixes → enter Probing mode" |
| Add full Frustration Trigger Protocol section | **implemented** | Deployed skill line 352: "Frustration Protocol" section with diagnostic questions and mode shift |

## Investigation 2 (Feb 24): Behavioral Compliance — Two-Layer Fix

### Layer 1: Skill Restructuring (Prompt-Level)

| Recommendation | Status | Evidence | Notes |
|---|---|---|---|
| 1a. Action-Identity Fusion at top of skill | **implemented** | Template lines 7-18: "Role" section with "Three jobs: COMPREHEND, TRIAGE, SYNTHESIZE" and tool action space table. Deployed skill lines 40-56 with exhaustive "Your Tools" table and affordance replacements | Restructured from original "Identity: Strategic Comprehender" |
| 1b. Affordance replacement at decision points | **implemented** | Deployed skill lines 54-56: explicit "Spawn workers: NOT Task tool", "Close completed work: NOT bd close", "Understand a topic: NOT reading code files" | Placed at section 1 (0% depth) as recommended |
| 1c. Reduce skill length from 640→<450 lines | **implemented** | Template: 486 lines, 5,995 tokens (vs 640 lines, ~8K tokens at investigation time). Note: slightly over 450 target but within margin | 82% token reduction from the Mar 1 peak of 27,200 tokens |
| 1d. Strategic repetition at 3 decision points | **partially-implemented** | Constraint "use orch spawn not Task tool" appears at: (1) Role section tool table, (2) Spawning Essentials "Agent Tool Is NOT a Spawn Mechanism" section, (3) Fast Path "Daemon down" row. bd close constraint at: (1) Role affordance replacement, (2) completion section. But repetition is 2 locations for bd close, 3 for Task tool | Meets spirit of recommendation. bd close enforcement shifted to hook so prompt repetition less critical |

### Layer 2: Tool-Layer Enforcement (Infrastructure-Level)

| Recommendation | Status | Evidence | Notes |
|---|---|---|---|
| 2a. Claude Code hook for orchestrator sessions | **implemented** | 6 hooks registered in `~/.claude/settings.json` under PreToolUse. Gate hooks: bd-close, bash-write, git-remote, spawn-context-validation. Nudge hooks: investigation-drift, spawn-context | Implemented as PreToolUse hooks on Bash/Read/Edit|Write matchers |
| 2b. Orchestrator session detection | **implemented** | Hooks detect orchestrator sessions via skill identity (load-skill-identity.sh at SessionStart) | Not via ORCH_ORCHESTRATOR env var as originally proposed — uses skill identity detection instead |
| 2c. Graduated response (warning → stronger → block) | **not-implemented** | Hooks are binary: nudges inject coaching message, gates block entirely. No escalation between violations within a session | Investigation 2 recommended 3-tier graduated response. Current hooks are either "nudge" (always coaching) or "gate" (always block). The nudge/gate distinction provides 2 levels but not per-session escalation |

## Investigation 3 (Mar 1): Testing Infrastructure Design

| Recommendation | Status | Evidence | Notes |
|---|---|---|---|
| `orch skill lint` / `skillc lint` — Static analyzer with 5 rules | **implemented** | `skillc lint` exists (per skill.yaml metadata and tools-and-commands.md reference doc). 5 rules: MUST-density, cosmetic redundancy, section sprawl, signal imbalance, dead constraint | Implemented as `skillc lint` not `orch skill lint` |
| `orch skill test` / `skillc test` — Scenario runner | **implemented** | `skillc test` exists. 13+ scenario YAML files in `.skillc/tests/scenarios/` (01 through 13), plus 4 fabrication scenarios and contrastive variants | More scenarios than the 7 originally proposed |
| `orch skill compare` / `skillc compare` — Result differ | **implemented** | `skillc compare` exists (per tools-and-commands.md) | |
| 7 scenario definitions with behavioral indicators | **implemented** | 13 scenarios exist (7 original + 6 additions: synthesis-after-completions, contradiction-detection, red-herring, absence-as-evidence, downstream-consumer-contract, stale-deprecation-claim) | Exceeded recommendation |
| `--bare` mode as control group | **implemented** | `skillc test --bare` flag exists (per investigation 4's blocked gate description) | |
| Behavioral test runs completed | **not-implemented** | Investigation 4 documents: "skillc test requires nested claude --print calls. From spawned agent: CLAUDECODE env var blocks nested sessions. Returns 0/0 scores." | **BLOCKED**: Can't run from spawned agents. Must be run from terminal |
| Skill variant infrastructure | **implemented** | `skills/src/meta/orchestrator/.skillc/variants/` contains 5 variants: 1C-neutral.md, 2C-neutral.md, 5C-neutral.md, 10C-neutral.md, plus padded variants | Used for constraint-count testing per dilution probes |

## Investigation 4 (Mar 4): Simplify Orchestrator Skill (v4)

| Recommendation | Status | Evidence | Notes |
|---|---|---|---|
| Strip from 2,368→~450 lines | **implemented** | Template: 486 lines. Deployed: 512 lines (includes skillc headers). Close to 450 target | 81% line reduction, 82% token reduction |
| Keep ~47 knowledge items (routing tables, vocabulary, intent distinctions) | **implemented** | Current skill contains: Fast Path surface table (19 rows), skill decision tree (9 mappings), intent clarification table (4 rows), stall triage table (6 rows), model selection table (3 rows), label taxonomy table (3 rows), completion workflow table (7 rows), epic model phases (3 rows), signal prefixes (4 rows), mode declarations (3 rows), principles (8 rows), commands reference (10 categories), tool ecosystem (5 tools) | Exceeds 47 items |
| Keep ≤4 behavioral norms (knowledge framing, not prohibition) | **implemented** | "Behavioral Norms" section (lines 389-399) explicitly states "Four judgment norms" and lists: Delegation, Filter before presenting, Act by default, Answer the question asked | Exactly 4 norms as designed |
| Remove hook-enforced constraint text (~350 lines) | **implemented** | No "NEVER use Edit/Write" text, no "NEVER git push" text, no "NEVER bd close" text, no pre-spawn ceremony requirements, no delegation rule repetitions, no anti-pattern tables for hook-enforced behaviors | Skill says "Infrastructure hooks enforce tool boundaries" (line 40) — single reference replacing ~350 lines |
| Knowledge framing, not prohibition framing | **implemented** | Skill uses "You COMPREHEND, TRIAGE, SYNTHESIZE" framing instead of "You NEVER write code, NEVER investigate, NEVER edit". The word "never" appears only in: "never implement" (behavioral norm 1), "Never use the Claude Code Agent tool" (spawn mechanism), and "Never use worktree isolation" (Docker constraint) — 3 occurrences vs 20+ in old skill | Dramatic shift from prohibition to knowledge framing |
| Behavioral testing gate (skillc test bare-parity regression) | **not-implemented** | Investigation 4's conclusion: "Remaining gate: skillc test bare-parity regression check must run from a non-Claude-Code terminal session." No evidence this was ever run | **CRITICAL PENDING GATE** — v4 deployed without passing this validation |
| Claim 4 partial separation (grammar in reference, routing+legibility in core) | **implemented** | Grammar/commands: `reference/tools-and-commands.md` (511 lines). Routing tables: in core skill. Legibility protocol (Dylan interface): in core skill section 7 | Clean separation as designed |

## Investigation 5 (Mar 5): 72-Commit Infrastructure Delta

| Changeset | Status | Evidence |
|---|---|---|
| A: Replace `orch frontier` → `orch status` (4 edits) | **implemented** | All 4 occurrences replaced. Current skill has `orch status` in: "When dashboard fails" (line 331), Session End Protocol step 2 (line 382), Commands Quick Reference Lifecycle (line 466), Commands Quick Reference Monitoring (line 468) |
| B: Add plan artifacts (3 edits) | **implemented** | Fast Path row (line 74): "Multi-phase coordination → orch plan create". Session End Protocol step 3 (line 383): "orch plan status". Knowledge Capture table (line 413): "orch plan create" |
| C: Update spawn flags (2 edits) | **implemented** | `--issue` description updated (line 237): "auto-created for `--no-track`". `--dry-run` row added (line 245) |
| D: Add review tiers to Completion Workflow (1 edit) | **implemented** | Completion Workflow table (lines 284-292) has Review Tier column + Knowledge capture (auto) and Issue creation (auto) rows |
| E: Daemon behavioral context in reference doc (4 additions) | **implemented** | `reference/tools-and-commands.md` lines 487-494: concurrency cap 5, round-robin, self-check invariants, auto-complete, stuck detection |
| F: `orch plan show` in Commands Quick Reference | **implemented** | Line 476: Strategic section includes `orch plan show` |

---

# 2. HOOK COVERAGE MATRIX

| Hook | Matcher | Enforces | Replaced Skill Text | Registered? | Working? |
|---|---|---|---|---|---|
| `gate-bd-close.py` | Bash | Only `orch complete` closes issues, not `bd close` | ~20 lines of "NEVER bd close" warnings and anti-patterns | Yes (PreToolUse/Bash) | Yes (37 tests per inv 4) |
| `gate-orchestrator-bash-write.py` | Bash | No Edit/Write tools for orchestrators | ~50 lines of "NEVER use Edit/Write", action space "You CANNOT" table | Yes (PreToolUse/Bash) | Yes (308 tests per inv 4) |
| `gate-orchestrator-git-remote.py` | Bash | No git push for workers | ~30 lines of "NEVER git push" text | Yes (PreToolUse/Bash) | Yes (64 tests per inv 4) |
| `gate-spawn-context-validation.py` | Bash | --issue and --intent required on orch spawn | ~30 lines of spawn ceremony requirement text | Yes (PreToolUse/Bash) | Yes (68 tests per inv 4) |
| `nudge-orchestrator-investigation-drift.py` | Read | Coaching when orchestrator reads code files (investigation drift) | ~100 lines of delegation rule repetitions and anti-pattern tables | Yes (PreToolUse/Read) | Yes (38 tests per inv 4) |
| `nudge-orchestrator-spawn-context.py` | Bash | Run kb context before spawn | ~40 lines of pre-spawn kb context ceremony text | Yes (PreToolUse/Bash) | Yes (40 tests per inv 4) |
| code-access gate (from inv 4) | — | Block/coach on code reads | ~80 lines of detailed protocol checklists | **NOT REGISTERED** as separate hook | Partially covered by investigation-drift nudge |

### Additional Hooks (not from orchestrator investigations)

| Hook | Matcher | Purpose | Notes |
|---|---|---|---|
| `gate-governance-file-protection.py` | Edit\|Write | Protects governance files from modification | Not from orchestrator skill investigations |
| `gate-worker-bd-dep-add.py` | Bash | Worker-only: controls bd dep add | Worker hook, not orchestrator |
| `gate-worker-git-add-all.py` | Bash | Worker-only: prevents git add -A/. | Worker hook, not orchestrator |
| `enforce-phase-complete.py` | Stop | Enforces Phase: Complete before session end | Session lifecycle hook |
| `pre-commit-knowledge-gate.py` | Bash | Pre-commit knowledge validation | Knowledge quality hook |
| `load-skill-identity.sh` | SessionStart | Loads skill identity at session start | Used for orchestrator detection |
| `orient-hook.sh` | SessionStart | Session orientation | Session lifecycle |

---

# 3. DRIFT ANALYSIS (Feb 28 Snapshot → Current)

## Structural Changes

| Dimension | Feb 28 Snapshot | Current Deployed | Change |
|---|---|---|---|
| **Line count** | 493 lines | 512 lines (deployed), 486 lines (template) | +19 lines deployed (includes skillc headers) |
| **Token count** | 6,376 | 5,995 | -381 tokens (-6%) |
| **Section structure** | 7 numbered sections (1-7) | 16 named sections | Complete restructure — domain-organized |
| **Section naming** | "1. Identity & Action Space", "2. Pre-Response Checks & Fast Path", "3. At Spawn Time", "4. During Work", "5. At Completion", "6. Session Boundaries", "7. Hard Constraints & Reference" | "Role", "Context Detection", "Fast Path", "Skill Selection", "Work Pipeline", "Spawning Essentials", "Completion Lifecycle", "Dylan Interface", "Behavioral Norms", "Knowledge Capture", "Principles Quick Reference", "Tool Ecosystem", "Config & Artifact Locations", "Commands Quick Reference", "Decision Promotion", "Workspace & Tier Architecture" | Shifted from lifecycle ordering to domain ordering |

## Content Type Ratios

| Content Type | Feb 28 Snapshot | Current | Change |
|---|---|---|---|
| **Routing/decision tables** | Fast Path (13 rows), Skill Decision tree, Intent Clarification | Fast Path (19 rows), Skill Decision tree, Intent Clarification, Completion Workflow, Stall Triage, Model Selection, Label Taxonomy | +6 tables, expanded Fast Path |
| **Behavioral constraints** | 8-item Pre-Response Checks, "Inviolable Constraints" section, Tool Action Space "You CANNOT" table, multiple NEVER declarations | 4 Behavioral Norms (knowledge-framed), single line "Infrastructure hooks enforce tool boundaries" | Dramatic reduction — constraint text replaced by hooks |
| **Vocabulary/concept definitions** | Three jobs, tool ecosystem, work pipeline | Same + Context Detection, Beads Tracking, Spawn Modes, Infrastructure Auto-Detection | Expanded vocabulary transfer |
| **Dylan interface** | Signal prefixes, Mode Declarations, Frustration Protocol, Epic Model Phases, Priority Emergence | Same set, restructured into dedicated "Dylan Interface" section | Same content, better organized |
| **Reference pointers** | Implicit (content inline) | Explicit pointers: "Full reference: tools-and-commands.md", "Full reference: workspace-architecture.md", "Full reference: ~/.kb/principles.md" | Shift to reference-doc architecture |

## Token Trajectory (Full History from stats.json)

```
Dec 2025:  12K → 15K → 18K → 20K (steady accretion)
Jan 2026:  12K → 15K → 16K → 20K → 21K → 12K → 15K (restructure Jan 15, then regrowth)
Late Jan:  15K → 16K → 24K (rapid accretion Jan 29)
Feb 2026:  6.5K → 7K → 8K → 9K (major trim Feb 6, then slow growth)
Late Feb:  4.6K → 5.3K → 6.4K (another trim Feb 24, then regrowth)
Mar 1:     27.2K (PEAK — 2,368 lines, accretion crisis)
Mar 4:     4.8K (v4 simplification — 82% reduction)
Mar 5-11:  5.4K → 5.6K → 5.7K → 6.0K (slow regrowth from inv 5 additions)
```

**Pattern:** The skill shows a clear accretion → crisis → simplification cycle. Three major simplification events (Jan 15, Feb 6/24, Mar 4), each followed by gradual regrowth. Current trajectory suggests another cycle in ~2-3 months if additions continue at current rate (+1,165 tokens in 7 days = ~5K/month).

---

# 4. PENDING ITEMS

## Critical

### 4.1 Behavioral Validation Gate (from inv 3 + inv 4)
- **What:** Run `skillc test` bare-parity regression from non-spawned terminal
- **Why blocked:** CLAUDECODE env var blocks nested `claude --print` calls from spawned agents
- **Risk:** v4 deployed without behavioral validation — we know the skill is structurally sound (correct content, correct length) but don't know if it actually improves behavior vs bare Claude
- **How to unblock:** Run from a terminal session (not Claude Code):
  ```bash
  cd ~/Documents/personal/orch-go
  skillc test --scenarios skills/src/meta/orchestrator/.skillc/tests/scenarios/ --variant skills/src/meta/orchestrator/SKILL.md --model sonnet
  skillc test --scenarios skills/src/meta/orchestrator/.skillc/tests/scenarios/ --bare --model sonnet
  skillc compare <v4-result.json> <bare-result.json>
  ```

## Medium Priority

### 4.2 Graduated Hook Response (from inv 2)
- **What:** First violation → warning, second → stronger warning, third → block
- **Current state:** Hooks are binary — nudges always coach, gates always block
- **Impact:** Agents that accidentally trigger a nudge hook get the same coaching message every time. No escalation means no "pain as signal" progression.
- **Recommendation:** Low priority. The binary nudge/gate distinction provides adequate differentiation. Graduated response would add complexity to hook implementations for marginal gain.

### 4.3 Code-Access Gate Hook (from inv 4)
- **What:** Dedicated gate for orchestrator reading code files (.go, .ts, etc.)
- **Current state:** `nudge-orchestrator-investigation-drift.py` on Read matcher provides coaching but doesn't block
- **Impact:** Orchestrators can still read code files — they get coached but not prevented. This is by design (investigation-drift is a nudge, not a gate) but investigation 4 listed it as a desired gate.
- **Recommendation:** Keep as nudge. A hard gate on Read would prevent orchestrators from reading SYNTHESIS.md, CLAUDE.md, and other legitimate orchestration artifacts that happen to be files. The nudge provides appropriate friction.

### 4.4 Post-Simplification Regrowth Monitoring
- **What:** Token count growing 24% in 7 days (4,830 → 5,995)
- **Risk:** Investigation 5 added factual content (commands, flags, review tiers) that is genuinely useful, but the pattern from stats.json shows every trim is followed by regrowth. At current rate, skill reaches 10K tokens (~1,000 lines) by early April.
- **Recommendation:** Set a monitoring threshold. When skill exceeds 7,000 tokens, trigger a "is this still within dilution budget?" review. The behavioral grammars model (constraint dilution threshold probe) found ≤4 behavioral constraints and ≤50 knowledge items as the budget. Current skill has 4 norms and ~50+ knowledge items — already at the edge.

## Low Priority

### 4.5 A/B Compliance Measurement (from inv 2)
- **What:** Measure before/after compliance rates for orch spawn vs Task tool, orch complete vs bd close
- **Current state:** No compliance data exists. The infrastructure to measure it (skillc test) exists but behavioral tests haven't been run.
- **Recommendation:** Blocked by 4.1. Once behavioral tests are unblocked, measure compliance as part of the same test run.

---

## Architectural Choices

### Knowledge framing over prohibition framing
- **What I chose:** Current skill uses "You COMPREHEND, TRIAGE, SYNTHESIZE" framing
- **What I rejected:** Old skill's "NEVER write code, NEVER investigate, NEVER edit" approach
- **Why:** Investigation 2 showed prohibitions compete with system prompt at 17:1 disadvantage. Knowledge transfer (routing tables, vocabulary) has measurable lift; prohibitions are bare-parity.
- **Risk accepted:** Agents may not understand boundaries as sharply without explicit prohibitions — but hooks enforce the boundaries that matter.

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/models/orchestrator-session-lifecycle/probes/2026-03-11-probe-orchestrator-skill-current-state-audit.md` — Audit probe

### Constraints Discovered
- `skillc test` cannot run from spawned agents (CLAUDECODE env var blocks nested sessions) — this blocks the behavioral validation gate for all skill changes
- Post-simplification regrowth is predictable from token trajectory history: every trim is followed by ~5K tokens/month of additions

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (4-section structured audit)
- [x] Probe file created and filled
- [x] Investigation read and analyzed
- [x] Ready for orchestrator review

### Follow-up Recommendations
1. **HIGH:** Run `skillc test` bare-parity regression from terminal (not spawned agent) — item 4.1
2. **MEDIUM:** Set 7,000-token monitoring threshold on orchestrator skill to catch regrowth early — item 4.4
3. **LOW:** Consider graduated hook response if binary nudge/gate proves insufficient — item 4.2

---

## Unexplored Questions

- **Does the v4 skill actually outperform bare Claude on knowledge-transfer scenarios?** Investigation 4's Mar 1 baseline showed ~22/56 vs bare 17/56. The v4 simplification was designed to maintain or improve this, but no v4 behavioral data exists.
- **Is the 4-behavioral-norm budget saturated?** The dilution threshold probe found ≤4 as the budget. With exactly 4 norms, there's zero headroom for additions. If a 5th norm is needed, one must be removed.
- **Are the 13 test scenarios covering the right failure modes?** The original 7 were designed from specific failures (intent spiral, delegation speed). The 6 additions (scenarios 8-13) were added for comprehension testing. No data on whether they detect real regressions.

---

## Friction

No friction — smooth session. All files accessible, investigations well-documented, current state well-organized.

---

## Session Metadata

**Skill:** investigation
**Model:** claude-opus-4-6
**Workspace:** `.orch/workspace/og-inv-task-orchestrator-skill-11mar-8e8d/`
**Probe:** `.kb/models/orchestrator-session-lifecycle/probes/2026-03-11-probe-orchestrator-skill-current-state-audit.md`
