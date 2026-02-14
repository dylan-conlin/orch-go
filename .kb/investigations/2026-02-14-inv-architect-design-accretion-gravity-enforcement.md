<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Accretion Gravity has detection infrastructure (hotspot analysis) but zero prevention/enforcement - violates "Gate Over Remind" principle.

**Evidence:** spawn_cmd.go (2,332 lines), session.go (2,166 lines), doctor.go (1,912 lines) all flagged as CRITICAL hotspots yet agents still freely add to them; hotspot check at spawn is warning-only (line 834-850); no completion gates block modifications to bloated files.

**Knowledge:** Enforcement requires four layers: (1) Spawn-time gates that block work in hotspot areas without extraction plan, (2) Real-time coaching detection when agents attempt to modify bloated files, (3) Completion verification that rejects PRs adding >50 lines to files >800 lines, (4) Explicit CLAUDE.md boundaries declaring "DO NOT MODIFY" files.

**Next:** Implement four-layer enforcement starting with spawn-time gates (highest ROI - prevents problem before it starts), then completion gates (catches violations before merge), then coaching plugin (real-time correction), finally CLAUDE.md boundaries (declarative prevention).

**Authority:** architectural - Cross-component design spanning spawn, completion, coaching, and documentation systems; requires synthesis across existing infrastructure.

<!--
Example D.E.K.N.:
Delta: Test-running guidance is missing from spawn prompts and CLAUDE.md.
Evidence: Searched 5 agent sessions - none ran tests; guidance exists in separate docs but isn't loaded.
Knowledge: Agents follow documentation literally; guidance must be in loaded context to be followed.
Next: Add test-running instruction to SPAWN_CONTEXT.md template.
Authority: implementation - Tactical fix within existing patterns, no architectural impact

Guidelines:
- Keep each line to ONE sentence
- Delta answers "What did we find?"
- Evidence answers "How do we know?"
- Knowledge answers "What does this mean?"
- Next answers "What should happen now?"
- Authority: Classify by who decides - implementation (worker within scope), architectural (orchestrator across boundaries), strategic (Dylan for irreversible/value choices)
- Enable 30-second understanding for fresh Claude
-->

---

# Investigation: Architect Design Accretion Gravity Enforcement

**Question:** How do we enforce Accretion Gravity as a gate rather than a reminder - preventing agents from growing files to 2,000+ lines through repeated tactical additions?

**Started:** 2026-02-14
**Updated:** 2026-02-14
**Owner:** Architect Agent (og-arch-design-accretion-gravity-14feb-e5c9)
**Phase:** Complete
**Next Step:** Implement Layer 1 (spawn-time gates) and Layer 4 (CLAUDE.md boundaries) as first phase
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Patches-Decision:** [Path to decision document this investigation patches/extends, if applicable - enables review triggers]
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| `.kb/models/completion-verification/probes/2026-02-13-friction-gate-inventory-all-subsystems.md` | extends | Yes - verified gate inventory, bypass ratios, systemic issues | None - prior probe confirms skill-class blindness (31.7% of bypasses), blanket bypass (16.7%), and noise patterns that inform accretion gate design |
| `.kb/guides/code-extraction-patterns.md` | extends | Yes - verified 13 extraction benchmarks, workflow exists | None - guide documents HOW but confirms no WHEN trigger exists |

**Relationship types:** extends, confirms, contradicts, deepens
**Verified:** Did you check claims against primary sources?
**Conflicts:** What contradictions did you find?

---

## Findings

### Finding 1: Hotspot Detection Exists But Is Warning-Only

**Evidence:**
- `orch hotspot` command successfully detects 115 hotspots including spawn_cmd.go (2,332 lines), session.go (2,166 lines), doctor.go (1,912 lines)
- Hotspot check runs at spawn time (spawn_cmd.go:834-850) but only prints warning: "⚠️ Proceeding with tactical approach in hotspot area"
- `RunHotspotCheckForSpawn()` returns `SpawnHotspotResult` with `HasHotspots` and `Warning` fields but spawn always proceeds
- Bloat threshold defaults to 800 lines; CRITICAL flag at 1,500+ lines

**Source:**
- cmd/orch/hotspot.go:36-155 (hotspot command implementation)
- cmd/orch/spawn_cmd.go:830-853 (hotspot check integration)
- `orch hotspot --threshold 3 --days 90` output (115 hotspots detected)

**Significance:** Detection infrastructure exists and works correctly, but lacks enforcement teeth. This is the classic "Gate Over Remind" violation - we warn but don't block, so agents ignore warnings and continue accretion.

---

### Finding 2: Completion Gates Are Post-Facto Only

**Evidence:**
- 11 completion gates verify work AFTER it's done: Phase Complete, SYNTHESIS.md, test evidence, build verification, visual verification
- Zero gates check "did you add >50 lines to a file already >800 lines?"
- Git diff verification (pkg/verify/git_diff.go) checks claimed files match actual changes but doesn't analyze change size or target file size
- Skill-aware gating exists (feature-impl triggers test evidence, visual verification) but no skill triggers accretion prevention

**Source:**
- .kb/guides/completion-gates.md:1-430 (complete gate inventory)
- pkg/verify/check.go:528-557 (VerifyCompletionFull implementation)
- pkg/verify/git_diff.go (delta verification)

**Significance:** By the time completion runs, accretion has already happened. Agent spent hours adding 200 lines to a 2,000-line file. Rejecting at completion wastes the agent's work. We need PREVENTION (block before work starts) not just DETECTION (catch after work done).

---

### Finding 3: Coaching Plugin Has Real-Time Detection Infrastructure

**Evidence:**
- Coaching plugin tracks tool usage in real-time via tool.execute.after hook (coaching.ts:1-100)
- Semantic grouping for bash commands, behavioral variation detection, circular pattern detection
- Writes metrics to ~/.orch/coaching-metrics.jsonl for dashboard display
- Could detect "agent just edited spawn_cmd.go which is 2,332 lines" BEFORE commit

**Source:**
- plugins/coaching.ts:1-1830 (plugin implementation)
- .opencode/plugin/coaching.ts:1-1830 (same file, symlinked)
- coaching.ts:70-101 (tool category definitions and hooks)

**Significance:** Real-time detection infrastructure already exists for behavioral patterns. Extending it to detect accretion attempts (editing files >800 lines) would provide in-flight correction: "⚠️ spawn_cmd.go is 2,332 lines (CRITICAL hotspot). Extract logic to new file before adding features."

---

### Finding 4: Code Extraction Guide Documents HOW But Not WHEN

**Evidence:**
- `.kb/guides/code-extraction-patterns.md` provides comprehensive extraction workflow (shared utilities first, then domain files)
- Documents 13 successful extractions with line reduction benchmarks (serve_agents.go: -1106 lines)
- Phase-by-phase instructions for Go, Svelte, TypeScript extraction patterns
- NO enforcement trigger - agents must already know they should extract

**Source:**
- .kb/guides/code-extraction-patterns.md:1-339
- Line reduction benchmarks (lines 292-308): serve_agents.go (-1106), status_cmd.go (-1058), clean_cmd.go (-670)

**Significance:** Educational content without enforcement is insufficient. Agents working in hotspot areas need MANDATORY extraction before feature addition, not just a guide they might read if they think to look for it.

---

### Finding 5: CLAUDE.md Has No Explicit Accretion Boundaries

**Evidence:**
- CLAUDE.md (lines 1-300) documents architecture, spawn modes, key packages, commands
- No "DO NOT MODIFY" list for critical hotspot files
- No file-level rules like "if adding >10 lines to spawn_cmd.go, create new package"
- No spawn context injection of accretion constraints

**Source:**
- CLAUDE.md:1-299 (project context document)
- SPAWN_CONTEXT.md template (embedded in spawn.go) - no accretion warnings

**Significance:** Agents have no declarative guidance about accretion boundaries. Unlike test evidence requirements (explicitly stated in completion gates), accretion prevention is implicit tribal knowledge.

---

## Synthesis

**Key Insights:**

1. **We Have Detection Without Prevention** - Hotspot analysis (Finding 1) correctly identifies bloated files, but only warns at spawn time. Agents proceed anyway because warnings have no teeth. This is the definition of "reminder not gate" - we notice the problem but don't block it.

2. **Post-Facto Gates Waste Agent Work** - Completion gates (Finding 2) verify deliverables after hours of work. If we reject a PR because the agent added 200 lines to a 2,000-line file, we've wasted the agent's time AND still have the bloat problem. Prevention must happen BEFORE work starts, not after it's done.

3. **Real-Time Infrastructure Exists But Is Underutilized** - The coaching plugin (Finding 3) already has hooks for real-time tool detection. We could extend it to warn agents mid-session: "You're editing a CRITICAL hotspot file - extract first." This bridges the gap between spawn-time prevention (too early - agents might not touch hotspot files) and completion verification (too late - work already done).

4. **Education Without Enforcement Fails** - Code extraction patterns guide (Finding 4) documents HOW to extract, but agents only read it if they already know they should. We need MANDATORY extraction gates, not optional reading material. Compare to test evidence gates: agents don't optionally run tests; completion REQUIRES test output.

5. **Declarative Boundaries Are Missing** - CLAUDE.md (Finding 5) documents architecture but doesn't declare "spawn_cmd.go is off-limits for feature additions." Agents need explicit constraints in loaded context, not implicit tribal knowledge. Pattern: test evidence is explicit in gates; accretion prevention should be explicit too.

**Answer to Investigation Question:**

To enforce Accretion Gravity as a gate rather than a reminder, we need **four enforcement layers** working in concert:

1. **Spawn-Time Gates (Prevention)** - Block spawning feature-impl tasks targeting CRITICAL hotspot files (>1,500 lines) without explicit extraction plan. Force architects to scope extraction BEFORE feature work begins. This is the highest ROI layer - prevents the problem before any work happens.

2. **Completion Verification Gates (Rejection)** - Add new gate: "If git diff shows +50 lines to any file already >800 lines, require extraction evidence." Similar to test evidence gate - won't let you complete without proof you extracted, not accreted.

3. **Real-Time Coaching Detection (Correction)** - Extend coaching plugin to detect when agent attempts to edit files >800 lines. Inject warning into tool result: "⚠️ spawn_cmd.go is 2,332 lines (CRITICAL). Extract logic to pkg/ before adding features. See .kb/guides/code-extraction-patterns.md"

4. **CLAUDE.md Boundaries (Declaration)** - Document explicit accretion constraints in project CLAUDE.md: "Files >1,500 lines are CRITICAL hotspots. Feature additions require extraction first. See orch hotspot for current list."

These layers are complementary, not redundant:
- Spawn gates catch planned accretion (task description says "add feature to spawn_cmd.go")
- Completion gates catch unplanned accretion (agent modified hotspot file during implementation)
- Coaching plugin provides real-time correction (agent learns during session, not after rejection)
- CLAUDE.md boundaries make constraints explicit (agents know rules without hitting gates)

---

## Decision Forks

### Fork 1: Completion gate severity — warning vs error?

**Options:**
- A: Hard error at 800 lines (block completion for any accretion to bloated files)
- B: Tiered — warning at 800, error at 1,500 lines
- C: Warning-only at all thresholds

**Substrate says:**
- Principle (Gate Over Remind): Must be a gate, not a reminder — warnings alone proven ineffective
- Probe (friction-gate-inventory): Gates with high bypass ratios (>5:1) become noise. test_evidence at 5.5:1 is noisy because skill-class blindness creates false positives
- Principle (Capture at Context): Gate must fire at meaningful threshold, not arbitrary one

**RECOMMENDATION:** Option B — Tiered thresholds. Warning at 800 lines gives agents actionable feedback. Error at 1,500 lines creates a hard gate for CRITICAL files. This avoids the noise problem (hard error at 800 would fire too often on legitimately large but manageable files) while still enforcing on truly dangerous files.

**Trade-off accepted:** Agents can still accrete to files between 800-1,500 lines without being blocked. Acceptable because files in this range are growing but not yet dangerous — the warning provides learning signal, and Coherence Over Patches (hotspot analysis) catches files trending toward 1,500.

**When this would change:** If files consistently grow past 1,500 lines despite warnings at 800, lower the error threshold to 1,000.

---

### Fork 2: Which skills are exempt from spawn gates?

**Options:**
- A: No exemptions — all skills blocked from CRITICAL hotspots
- B: Exempt knowledge-producing skills (architect, investigation, capture-knowledge, audit)
- C: Exempt based on `--force-hotspot` flag only

**Substrate says:**
- Principle (Gate Over Remind caveat): "Gates must be passable by the gated party." An architect analyzing a 2,000-line file to design its decomposition cannot be blocked from that file
- Model (Spawn Architecture): Tier system already distinguishes light/full; skill-class awareness exists
- Probe (friction-gate-inventory): Skill-class blindness causes 31.7% of completion bypass events — gates that don't distinguish skills become noise

**RECOMMENDATION:** Option B — Exempt knowledge-producing skills. architect, investigation, capture-knowledge, and codebase-audit need to read and analyze hotspot files to do their job. feature-impl and systematic-debugging are the skills that accrete.

**Trade-off accepted:** Knowledge-producing agents can still add code to hotspot files. Mitigated because these skills rarely produce implementation code (their deliverables are investigations and decisions, not features).

**When this would change:** If knowledge-producing skills start generating implementation code in hotspot files, add a secondary check: exempt from spawn gate but still subject to completion gate.

---

### Fork 3: How does the completion gate handle extraction tasks?

**Options:**
- A: Exempt extraction tasks entirely (skill-based exemption)
- B: Net-negative delta passes — if overall change reduces line count, gate passes
- C: Require explicit `--extraction` flag to bypass

**Substrate says:**
- Principle (Accretion Gravity): "The fix is structural constraints, not better agents" — extraction IS the structural fix
- Guide (code-extraction-patterns.md): Extraction benchmarks show net reductions (-1106, -1058, -670 lines) — the delta is always negative
- Principle (Gate Over Remind caveat): Gate must be passable — blocking extraction work defeats the purpose

**RECOMMENDATION:** Option B — Net-negative delta. If `git diff --numstat` shows the file's total line count decreased (or the net added lines across ALL files is negative), the gate passes. This naturally exempts extraction tasks without requiring special flags or skill-based exemptions.

**Trade-off accepted:** An agent could game this by deleting unrelated code to offset accretion. Unlikely in practice — agents don't spontaneously delete code, and the git diff still shows what was added vs removed.

**When this would change:** If agents start gaming the delta (deleting comments/blank lines to offset additions), add a check for net *meaningful* lines (exclude blank/comment lines from delta).

---

### Fork 4: Coaching plugin — block or warn?

**Options:**
- A: Blocking injection (prevent tool execution)
- B: Warning with escalating urgency (warn → strong warn → critical)
- C: Single non-escalating warning

**Substrate says:**
- Principle (Pain as Signal): "Agents must feel the friction in their active context" — warning must be in-context, not ignorable
- Model (coaching plugin architecture): Plugin uses `noReply: true` injection — cannot block tool execution, only inject messages between turns
- Existing pattern (frame collapse detection): Tiered injection at coaching.ts:1635-1679 — 1st edit warns, 3+ edits strong warning

**RECOMMENDATION:** Option B — Escalating urgency, matching existing frame collapse pattern. First edit to a >800 line file gets a warning. 3+ edits to the same bloated file gets a stronger message with extraction directive. This is architecturally consistent with how the coaching plugin already works (frame collapse uses same tiered pattern).

**Trade-off accepted:** Can't technically block execution. But Pain as Signal says the friction of repeated warnings changes behavior even without blocking — agents that see "you've edited this 2,332-line file 5 times, extract to pkg/ NOW" will self-correct. The coaching plugin's existing frame collapse pattern proves this approach works.

**When this would change:** If OpenCode adds tool-blocking capabilities to plugins, upgrade to blocking for files >1,500 lines.

---

### Fork 5: Threshold calibration

**Options:**
- A: Conservative — warn at 1,000, error at 2,000, delta at +100
- B: Moderate — warn at 800, error at 1,500, delta at +50
- C: Aggressive — warn at 500, error at 1,000, delta at +25

**Substrate says:**
- Existing infrastructure (hotspot.go:477-486): 800 lines = moderate bloat, 1,500 lines = CRITICAL — thresholds already calibrated
- Code extraction guide benchmarks: Target file size 300-800 lines (healthy range)
- Principle (Accretion Gravity): "25 agents each add one feature" — the +50 line delta captures a typical single-feature addition

**RECOMMENDATION:** Option B — Use existing thresholds. 800/1,500/+50 are already calibrated by hotspot analysis and match the extraction guide's target range. Introducing different thresholds would create confusion ("hotspot says 800, gate says 1,000").

**Trade-off accepted:** 800 lines is below some legitimate utility files. Mitigated by making 800 a warning (not error) and providing the net-negative delta escape for extraction work.

**When this would change:** After 2 weeks of operation, analyze bypass rate. If >50% of 800-line warnings are false positives, raise to 1,000. If files regularly hit 1,500 despite warnings, lower error threshold to 1,200.

---

## Structured Uncertainty

**What's tested:**

- ✅ Hotspot detection works correctly - ran `orch hotspot --threshold 3 --days 90`, detected 115 hotspots including spawn_cmd.go (2,332 lines)
- ✅ Hotspot check at spawn time is warning-only - verified spawn_cmd.go:834-850 shows warning but proceeds
- ✅ Completion gates don't check accretion - reviewed all 11 gates in .kb/guides/completion-gates.md, none check file size delta
- ✅ Coaching plugin has real-time tool hooks - verified coaching.ts:70-101 implements tool.execute.after pattern
- ✅ Code extraction guide exists - verified .kb/guides/code-extraction-patterns.md documents 13 extractions

**What's untested:**

- ⚠️ Would spawn-time gates reduce accretion? (Hypothesis: blocking feature-impl spawns in CRITICAL hotspots forces extraction, but not tested with real agents)
- ⚠️ Would completion gates catch unplanned accretion? (Hypothesis: rejecting +50 lines to >800 line files catches violations, but edge cases unclear - what if extraction IS the task?)
- ⚠️ Would real-time coaching change agent behavior? (Hypothesis: mid-session warnings prompt extraction, but agents might ignore warnings like they ignore spawn-time warnings)
- ⚠️ What's the right line threshold? (Using 800 lines from existing bloat detection, but optimal threshold untested - too low creates false positives, too high allows accretion)

**What would change this:**

- Finding would be wrong if spawn-time gates block legitimate extraction work (architect sessions that NEED to read bloated files to design extraction)
- Finding would be wrong if agents find workarounds (split PRs into <50 line chunks to bypass completion gates)
- Finding would be wrong if coaching warnings have no effect (agents ignore real-time warnings just like spawn-time warnings)
- Finding would be wrong if 80% of hotspot violations are legitimate (sometimes you DO need to add to large files - shared utilities, framework extensions)

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommendation Authority

Classify each recommendation by authority level to route to the appropriate decision-maker:

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Implement four-layer accretion enforcement (spawn gates, completion gates, coaching plugin, CLAUDE.md boundaries) | architectural | Cross-component design spanning spawn subsystem, completion verification, coaching plugin, and documentation; requires synthesis of how layers interact and what thresholds to use |

**Authority Levels:**
- **implementation**: Worker decides within scope (reversible, single-scope, clear criteria, no cross-boundary impact)
- **architectural**: Orchestrator decides across boundaries (cross-component, multiple valid approaches, requires synthesis)
- **strategic**: Dylan decides on direction (irreversible, resource commitment, value judgment, premise-level question)

**Classification test:** "Does this decision stay inside my scoped context, or does it reach out?"
- Stays inside → implementation
- Reaches to other components/agents → architectural
- Reaches to values/direction/irreversibility → strategic

### Recommended Approach ⭐

**Four-Layer Defense in Depth** - Enforce Accretion Gravity through complementary prevention (spawn gates), detection (coaching plugin), rejection (completion gates), and declaration (CLAUDE.md boundaries).

**Why this approach:**
- **Highest ROI first:** Spawn gates prevent wasted work by blocking accretion before it starts, unlike completion gates that reject finished work
- **Real-time correction:** Coaching plugin catches unplanned accretion during sessions, when agents can still change course
- **Comprehensive coverage:** Spawn gates catch planned accretion ("add feature to spawn_cmd.go"), completion gates catch unplanned accretion (agent modified hotspot during implementation), coaching provides in-flight learning
- **Explicit over implicit:** CLAUDE.md boundaries make constraints discoverable without hitting gates
- **Builds on existing infrastructure:** All four layers extend existing systems (hotspot analysis, completion verification, coaching plugin) rather than creating new infrastructure

**Trade-offs accepted:**
- **False positives on extraction work:** Spawn gates will block architect sessions that legitimately need to read bloated files for extraction design. Mitigate with `--force-hotspot` override flag and skill exemptions (architect, investigation skills bypass spawn gates).
- **Threshold tuning required:** Using 800 lines (existing bloat threshold) may be too aggressive or too lenient. Will need iteration based on real usage. Start conservative (1,500 lines for spawn blocking, 800 lines for completion warnings), tighten after validating effectiveness.
- **Agent workarounds possible:** Agents might split PRs into <50 line chunks to bypass completion gates. Monitor for this pattern; if it emerges, add cumulative delta tracking across related commits.

**Implementation sequence:**
1. **Spawn-time gates (highest ROI)** - Modify spawn_cmd.go:834-853 to BLOCK (not warn) when feature-impl spawns target CRITICAL hotspots (>1,500 lines) without extraction plan. Prevents problem before work starts.
2. **Completion gates (catch escapes)** - Add new gate to pkg/verify/check.go: check git diff for +50 lines to files >800 lines, require extraction evidence in beads comments. Catches unplanned accretion.
3. **Coaching plugin extension (real-time learning)** - Add file size check to coaching.ts tool hooks: when agent edits file >800 lines, inject warning with link to extraction guide. Provides correction during session.
4. **CLAUDE.md boundaries (declarative)** - Document accretion constraints in project CLAUDE.md: "Files >1,500 lines are CRITICAL hotspots. Run `orch hotspot` to see current list. Feature additions require extraction first." Makes rules explicit.

### Alternative Approaches Considered

**Option B: Spawn Gates Only (Prevention Without Detection)**
- **Pros:** 
  - Simplest to implement (single change to spawn_cmd.go)
  - Highest ROI - stops problem at the source
  - No real-time overhead (coaching plugin CPU cost)
- **Cons:** 
  - Misses unplanned accretion (agent modifies hotspot file during implementation, not explicitly targeted in task)
  - No in-session learning (agents hit gate rejection, don't understand WHY until they research)
  - No completion backstop (if spawn gate is bypassed with --force, nothing catches violation)
- **When to use instead:** If coaching plugin proves too noisy (too many false positives) or real-time detection overhead is unacceptable

**Option C: Completion Gates Only (Detection Without Prevention)**
- **Pros:** 
  - Simple to implement (single new gate in pkg/verify/check.go)
  - Catches all accretion regardless of how it happened (planned or unplanned)
  - No spawn-time friction (agents can start work immediately)
- **Cons:** 
  - Wastes agent work (agent spends hours adding feature, then gets rejected at completion)
  - Misses in-session correction opportunity (agent could have extracted mid-work if warned)
  - Creates frustration (rejection after work is more demoralizing than prevention before work)
- **When to use instead:** If spawn gates prove too restrictive (too many false positives blocking legitimate work) or extraction planning overhead is too high

**Option D: CLAUDE.md Boundaries Only (Education Without Enforcement)**
- **Pros:**
  - Zero implementation cost (just documentation)
  - Agents learn constraints organically through loaded context
  - No friction for legitimate edge cases (agents make judgment calls)
- **Cons:**
  - Proven ineffective (Finding 4 shows education without enforcement fails - agents don't read guides unless forced)
  - No measurable compliance (can't track if agents are following boundaries)
  - Same pattern as current state (hotspot analysis warns but doesn't block - agents ignore warnings)
- **When to use instead:** Never as sole approach; only as complement to gates (declarative boundaries + enforcement = effective; boundaries alone = ignored)

**Rationale for recommendation:** 

Four-layer approach beats alternatives because **each layer addresses a different failure mode:**

- Spawn gates catch **planned accretion** (task explicitly targets hotspot file)
- Completion gates catch **unplanned accretion** (agent modified hotspot during implementation)
- Coaching plugin enables **in-session learning** (agent warned mid-work, can still change course)
- CLAUDE.md boundaries provide **discoverability** (agents know constraints without hitting gates)

Single-layer approaches (Options B, C, D) each leave gaps. Two-layer (spawn + completion) is pragmatic minimum, but coaching plugin adds real-time correction at low cost, and CLAUDE.md boundaries cost nothing to add.

---

### Implementation Details

**Phase 1 (implement first — highest ROI + zero cost):**

1. **CLAUDE.md Accretion Boundaries** (zero cost, immediate effect)
   - Add section to CLAUDE.md documenting CRITICAL hotspot files and the rule: "Files >1,500 lines require extraction before feature addition"
   - Include link to `orch hotspot` and `.kb/guides/code-extraction-patterns.md`
   - Agents that load CLAUDE.md see constraints before starting work

2. **Spawn-Time Gate** (modify `spawn_cmd.go:834-853`)
   - Change `RunHotspotCheckForSpawn()` handling from warning-only to blocking for feature-impl/systematic-debugging skills
   - Exempt skills: architect, investigation, capture-knowledge, codebase-audit
   - Exempt flags: `--force-hotspot` (explicit override with reason logged to events)
   - Exempt: daemon-driven spawns (triage already happened)
   - Block when: `hotspotResult.HasHotspots && maxBloatScore >= 1500 && !isExemptSkill`
   - Error message: `"CRITICAL hotspot: [file] is [N] lines. Spawn architect to design extraction first, or use --force-hotspot to override."`

**Phase 2 (catch escapes):**

3. **Completion Accretion Gate** (new gate in `pkg/verify/check.go`)
   - Add constant: `GateAccretion = "accretion"`
   - Add function: `VerifyAccretionForCompletion(workspacePath, projectDir string) *AccretionResult`
   - Implementation:
     a. Get list of changed files from `git diff --numstat` (reuse `GetGitDiffFiles()` pattern from git_diff.go)
     b. For each changed file, count current line count with `wc -l`
     c. If file >1,500 lines AND net added >50 lines → error (hard gate)
     d. If file >800 lines AND net added >50 lines → warning (soft signal)
     e. If net delta is negative (extraction) → pass regardless of file size
   - Insert in `VerifyCompletionFull()` after git_diff gate, before build gate (~line 420)
   - Skip for orchestrator tier (`isOrch`) — orchestrators don't write code
   - Emit `gate.accretion.triggered` event for metrics tracking

**Phase 3 (real-time correction):**

4. **Coaching Plugin Extension** (modify `coaching.ts`)
   - Add to `tool.execute.after` hook, after existing frame collapse detection (~line 1679)
   - On `edit` or `write` tool, extract file path from `input.args.file_path`
   - Count lines with `fs.readFileSync` + split (or shell out to `wc -l`)
   - If >800 lines: inject accretion warning via existing `injectCoachingMessage()` pattern
   - Track per-session: `state.accretion = { fileWarnings: Map<string, number> }`
   - Escalating urgency (match frame collapse pattern):
     - 1st edit: `"📏 File Size Warning: [file] is [N] lines. Consider extraction before adding."`
     - 3+ edits to same file: `"🚨 Accretion Alert: You've edited [file] ([N] lines) [X] times. Extract to pkg/ BEFORE adding features. See .kb/guides/code-extraction-patterns.md"`
   - Add new `patternType: "accretion_warning"` to `injectCoachingMessage()`
   - Write metrics to `~/.orch/coaching-metrics.jsonl` with `metric_type: "accretion_warning"`

**File targets:**

| File | Action | Lines Changed (est.) |
|------|--------|---------------------|
| `CLAUDE.md` | Add accretion boundaries section | +15 |
| `cmd/orch/spawn_cmd.go:834-853` | Change warning to conditional blocking | +20 |
| `pkg/verify/check.go` | Add `GateAccretion` constant, `VerifyAccretionForCompletion()` | +60 |
| `pkg/verify/accretion.go` (new) | Accretion verification logic | +80 |
| `coaching.ts:1679+` | Add accretion detection to tool.execute.after | +40 |

**Things to watch out for:**
- ⚠️ **Spawn gate must not block the agent doing the extraction.** If an orchestrator spawns `feature-impl "extract spawn_cmd.go into packages"`, the spawn gate must not block it. Use task description keyword matching: if task contains "extract", "decompose", "refactor", "split", exempt from spawn gate.
- ⚠️ **Line counting performance.** `wc -l` on every changed file at completion time is cheap (milliseconds). `fs.readFileSync` in coaching plugin on every edit is slightly more expensive. Cache line counts per session to avoid repeated reads.
- ⚠️ **Coaching plugin runs for orchestrators too.** The accretion warning should only fire for workers (use existing `isWorker` detection at coaching.ts:1543). Orchestrators don't write code; coaching plugin already filters on this.
- ⚠️ **Event noise from coaching.** The friction gate probe found daemon dedup generates 3,866 events (55% of all events). Accretion coaching warnings should be low-volume (most agents don't touch bloated files). Monitor for first 2 weeks.

**Areas needing further investigation:**
- How to handle cross-file accretion (agent creates 5 new files of 50 lines each instead of one coherent package — total complexity increases but no single file is large). Out of scope for this design — address if pattern emerges.
- Integration with `orch hotspot --json` for dynamic threshold loading. Currently thresholds are hardcoded; future enhancement could read from a config file updated by periodic hotspot analysis.
- Whether to add accretion metrics to the dashboard. The coaching plugin already writes to `coaching-metrics.jsonl` which the dashboard reads — adding an accretion panel is straightforward but out of scope.

**Success criteria:**
- ✅ `orch spawn feature-impl "add X to spawn_cmd.go"` is BLOCKED with clear error message pointing to architect skill
- ✅ `orch spawn architect "design extraction for spawn_cmd.go"` proceeds normally (exempt skill)
- ✅ `orch complete <id>` warns when agent added +50 lines to file >800 lines, errors when >1,500 lines
- ✅ `orch complete <id>` passes when agent's changes resulted in net-negative line delta (extraction)
- ✅ Coaching plugin injects warning when worker agent edits file >800 lines
- ✅ CLAUDE.md documents accretion boundaries and links to hotspot command
- ✅ Events: `gate.accretion.triggered` emitted when completion gate fires; `spawn.accretion.blocked` when spawn gate blocks

**Acceptance criteria (testable):**
- Unit test: `TestAccretionGate_BlocksLargeAdditions` — verify gate errors when +50 lines to 1,500+ line file
- Unit test: `TestAccretionGate_PassesNetNegativeDelta` — verify gate passes when extraction reduces file size
- Unit test: `TestAccretionGate_WarnsModerateAdditions` — verify warning (not error) for +50 lines to 800-1,500 line file
- Integration: spawn a feature-impl task targeting spawn_cmd.go, verify spawn is blocked
- Integration: spawn an architect task targeting spawn_cmd.go, verify spawn proceeds

---

## Decision Gate Guidance (if promoting to decision)

**Add blocks: frontmatter when:**
- This decision resolves the recurring accretion problem (spawn_cmd.go grew from ~200 to 2,332 lines across 25+ agent sessions)
- This decision establishes structural constraints future agents might violate

**Suggested blocks keywords:**
- "accretion", "hotspot", "file size", "large file", "bloat"
- "add feature to spawn", "modify spawn_cmd", "add to session.go"

---

## References

**Files Examined:**
- `cmd/orch/spawn_cmd.go:830-860` — Hotspot check integration at spawn time (warning-only, never blocks)
- `cmd/orch/hotspot.go:36-486` — Hotspot analysis implementation (bloat thresholds 800/1500, hotspot types, severity recommendations)
- `pkg/verify/check.go:238-437` — VerifyCompletionFull with all 12 gate implementations and gate constant definitions
- `pkg/verify/git_diff.go:17-477` — Git diff verification (already has file list, delta detection, project dir — extension point for accretion check)
- `plugins/coaching.ts:1543-1829` — tool.execute.after hook with frame collapse pattern (tiered injection model for accretion detection)
- `plugins/coaching.ts:666-761` — injectCoachingMessage function with noReply pattern
- `.kb/guides/code-extraction-patterns.md` — Extraction workflow, 13 benchmarks, target file sizes 300-800 lines
- `~/.kb/principles.md:162-193` — Gate Over Remind principle (enforce through gates, gates must be passable)
- `~/.kb/principles.md:636-667` — Accretion Gravity principle (structural constraints fix)
- `~/.kb/principles.md:409-438` — Coherence Over Patches principle (5+ fix commits = structural issue)
- `~/.kb/principles.md:968-1005` — Strategic-First Orchestration principle (hotspot areas require architect)
- `.kb/models/completion-verification/probes/2026-02-13-friction-gate-inventory-all-subsystems.md` — 48-gate inventory across 3 subsystems, bypass ratios, skill-class blindness finding

**Related Artifacts:**
- **Principle:** `~/.kb/principles.md` — Accretion Gravity, Gate Over Remind, Strategic-First Orchestration — foundational principles this design enforces
- **Probe:** `.kb/models/completion-verification/probes/2026-02-13-friction-gate-inventory-all-subsystems.md` — Informs threshold calibration and noise avoidance (skill-class blindness, bypass ratios)
- **Guide:** `.kb/guides/code-extraction-patterns.md` — The HOW that this design adds WHEN triggers for

---

## Investigation History

**2026-02-14 10:37:** Investigation started (agent og-feat-architect-design-accretion-14feb-7515)
- Initial question: How do we enforce Accretion Gravity as a gate rather than a reminder?
- Context: spawn_cmd.go grew to 2,332 lines; hotspot analysis detects but doesn't prevent; Gate Over Remind principle violated

**2026-02-14 10:38-10:42:** Findings documented (5 findings)
- Verified hotspot detection is warning-only, completion gates are post-facto, coaching plugin has unused capacity, extraction guide lacks triggers, CLAUDE.md lacks boundaries

**2026-02-14 10:42:** Session resumed by second architect agent (og-arch-design-accretion-gravity-14feb-e5c9)
- Verified all 5 findings against actual code
- Consulted full substrate stack (5 principles, 1 probe, 1 guide)
- Navigated 5 decision forks with substrate traces
- Completed implementation details with file targets, acceptance criteria, success metrics

**2026-02-14:** Investigation completed
- Status: Complete
- Key outcome: Four-layer accretion enforcement architecture designed with concrete implementation plan — spawn gates (prevention), completion gates (rejection), coaching plugin (real-time correction), CLAUDE.md boundaries (declaration)
