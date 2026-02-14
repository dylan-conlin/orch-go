## Summary (D.E.K.N.)

**Delta:** The model/probe/investigation system has 6 critical inconsistencies across skill sources and CLAUDE.md files, most notably: a missing PROBE.md template, two conflicting deployed investigation skills, and contradictory model creation thresholds (3+ vs 15+).

**Evidence:** Read 25+ source files across orch-knowledge/skills/src/ and ~/.claude/skills/; verified .orch/templates/PROBE.md does not exist; confirmed two deployed investigation skill versions with different checksums and probe mode presence.

**Knowledge:** The system is split between orchestrator-side routing (clear and consistent) and worker-side execution (inconsistent due to deployment drift and missing template). The "probe" term is overloaded: model-scoped probes (investigation skill) vs decision-navigation probes (experiment to resolve unknown fork).

**Next:** Create PROBE.md template, reconcile deployed investigation skill versions, consolidate model creation threshold to single authoritative source.

**Authority:** architectural - Cross-skill, cross-repo consistency requires orchestrator synthesis across multiple components.

---

# Investigation: Audit Model/Probe/Investigation Claims Across Skills and CLAUDE.md

**Question:** What does each skill source and CLAUDE.md file claim about the model/probe/investigation system, and where are the inconsistencies?

**Started:** 2026-02-13
**Updated:** 2026-02-13
**Owner:** Worker agent (spawned by orchestrator)
**Phase:** Complete
**Next Step:** None - findings ready for orchestrator review
**Status:** Complete

**Patches-Decision:** .kb/decisions/2026-01-12-models-as-understanding-artifacts.md
**Extracted-From:** N/A

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| .kb/models/PHASE4_REVIEW.md | extends | yes | Phase 4 says "3+ investigations" trigger model creation (Question 4), contradicting README.md and lifecycle guide |
| .kb/models/PHASE3_REVIEW.md | extends | yes | Phase 3 says "~15 investigations" inflection, consistent with lifecycle guide |
| .kb/investigations/2026-02-13-inv-audit-skills-orch-knowledge-skills.md | related | pending | Parallel audit of skill documentation |

---

## Findings

### Finding 1: PROBE.md Template Does Not Exist

**Evidence:** `.orch/templates/PROBE.md` returns "File does not exist" when read. The templates directory contains only: `FAILURE_REPORT.md`, `SYNTHESIS.md`, `SESSION_HANDOFF.md`.

**Source:**
- `Read .orch/templates/PROBE.md` → error: file does not exist
- `Glob .orch/templates/*` → 3 files, no PROBE.md
- Referenced in orchestrator skill at lines 247, 268, 321, 468
- Referenced in investigation skill (.skillc/intro.md:18, .skillc/template.md:7, .skillc/completion.md:21)

**Significance:** Both the orchestrator skill and investigation skill instruct agents to use `.orch/templates/PROBE.md` as a template. Workers spawned in probe mode have no template to follow. This is a critical gap — the routing system points to a deliverable that doesn't exist, meaning probe creation relies entirely on inline skill instructions (4 required sections: Question, What I Tested, What I Observed, Model Impact).

---

### Finding 2: Two Conflicting Deployed Investigation Skills

**Evidence:** Two versions of the investigation skill exist at different paths with different content:

| Path | Checksum | Compiled | Has Probe Mode? | Description |
|------|----------|----------|-----------------|-------------|
| `~/.claude/skills/src/worker/investigation/SKILL.md` | 91b0e65cab3c | 2026-02-06 15:35:56 | **NO** | "Record what you tried and observed" |
| `~/.claude/skills/worker/investigation/SKILL.md` | 1cf402739ec0 | 2026-02-12 22:06:41 | **YES** | "default to model-scoped probes when injected model claims are present" |

The newer version (Feb 12) includes `Artifact Mode Selection (Probe Default)` with full mode detection logic. The older version (Feb 6) is purely investigation-focused with no probe routing.

**Source:**
- `Read ~/.claude/skills/src/worker/investigation/SKILL.md` → checksum 91b0e65cab3c, no probe mode
- `Read ~/.claude/skills/worker/investigation/SKILL.md` → checksum 1cf402739ec0, has probe mode
- My own spawn loaded the OLD version (91b0e65cab3c per SPAWN_CONTEXT header)

**Significance:** Workers may receive either version depending on which path gets loaded. The `src/worker/` path appears to be an older deployment location, while `worker/` is the current one. This means some spawned workers will be probe-aware and others won't, creating inconsistent behavior. The spawn that created THIS investigation loaded the old version, so I was investigation-only despite model markers being present in my SPAWN_CONTEXT.

---

### Finding 3: Model Creation Threshold Contradictions (3+ vs 15+)

**Evidence:** Three different thresholds for when to create a model:

| Source | Threshold | Quote |
|--------|-----------|-------|
| `.kb/models/README.md:12` | **3+** | "3+ investigations on same topic converge to understanding" |
| `.kb/guides/understanding-artifact-lifecycle.md:18,179` | **15+** | "15+ investigations on topic" and four-factor test (HOT, COMPLEX, OWNED, STRATEGIC_VALUE) |
| `.kb/models/PHASE3_REVIEW.md:274` | **~15** | "The inflection point appears to be ~15 investigations" |
| `.kb/models/PHASE4_REVIEW.md:213` | **3+** | "When 3+ investigations cluster on new topic" (in Hypothesis section) |
| `.kb/guides/understanding-artifact-lifecycle.md:400` | **10-15** | "Never below 10 investigations. Between 10-15, must pass four-factor test" |

**Source:**
- `.kb/models/README.md:12` — "Signals: 3+ investigations on same topic converge"
- `.kb/guides/understanding-artifact-lifecycle.md:179` — "Trigger: 15+ investigations on single topic"
- `.kb/guides/understanding-artifact-lifecycle.md:400` — "Never below 10"
- `.kb/models/PHASE3_REVIEW.md:264-274` — "inflection point ~15 investigations"
- `.kb/models/PHASE4_REVIEW.md:213` — "create models when 3+ investigations cluster"

**Significance:** A 5x range (3 vs 15) in the creation threshold is not a nuance — it's a fundamental disagreement. The README.md (which is the entry point agents read first) says 3+, but the detailed lifecycle guide says 15+ with a hard floor of 10. The Phase reviews are split. This means orchestrators and workers will have different thresholds depending on which document they consult first.

---

### Finding 4: "Probe" is Overloaded Across Two Distinct Meanings

**Evidence:** The term "probe" is used with two completely different meanings across the skill system:

**Meaning 1: Model-Scoped Probe (investigation/orchestrator skills)**
- File artifact in `.kb/models/{name}/probes/{date}-{slug}.md`
- Tests a specific model claim (confirms/contradicts/extends)
- Orchestrator routes to it when model exists for domain
- Required sections: Question, What I Tested, What I Observed, Model Impact
- Verdicts: `confirms` | `contradicts` | `extends`

**Meaning 2: Decision-Navigation Probe (decision-navigation/architect/design-session skills)**
- Ad-hoc experiment to resolve an unknown fork
- Not a file artifact — it's a pattern/technique
- "What's the smallest thing we could try to learn?"
- Types: Prototype, Sketch, Investigation, Ask
- Used in: decision-navigation:107-144, architect SKILL.md:175, design-session SKILL.md:211

**Source:**
- Orchestrator skill lines 143-149, 234-247 (model-scoped probes)
- Decision-navigation skill lines 107-144 (probing unknown forks)
- Architect skill line 175 (decision-navigation probes)
- Meta-orchestrator interface lines 203-205 (epic model probes — yet another variant)

**Significance:** A worker spawned with both decision-navigation and investigation skills receives conflicting definitions of "probe." The decision-navigation "probing protocol" describes lightweight experiments, while the investigation skill's "probe mode" describes a specific file artifact. The epic model's "probing" is yet another variant (tracking probes toward understanding). This semantic overload creates confusion about what "probe" means in any given context.

---

### Finding 5: Orchestrator Skill Has Clear Routing But Workers Can't Execute It

**Evidence:** The orchestrator skill provides an unambiguous routing table:

```
Spawn-Time Probe vs Investigation Routing:
- Model exists for domain → Probe in .kb/models/{model-name}/probes/
- No model exists → Investigation in .kb/investigations/
```

But execution depends on:
1. The worker receiving the probe-aware investigation skill (Finding 2 shows this is inconsistent)
2. The PROBE.md template existing (Finding 1 shows it doesn't)
3. The worker detecting model-claim markers in SPAWN_CONTEXT (which requires `kb context` to inject model content at spawn time)

The orchestrator skill correctly describes the routing decision at spawn time (lines 262-271, 315-324). But the worker-side skill loaded by `skillc deploy` may or may not include probe mode, and even when it does, the template file is missing.

**Source:**
- Orchestrator skill: lines 262-271, 315-324, 466-468
- Investigation skill (.skillc/intro.md): lines 5-27 (mode detection)
- `.orch/templates/PROBE.md` — does not exist

**Significance:** The orchestrator-to-worker handoff has a gap. Orchestrators are instructed to route to probes, workers are sometimes equipped to create them, but the template that both reference doesn't exist. This is a complete-chain failure: correct routing → incomplete tooling → broken execution.

---

### Finding 6: Global CLAUDE.md Decision Tree Contradicts Investigation Skill's Mode Detection

**Evidence:** The global CLAUDE.md provides a decision tree (lines 119-122):

```
- Model exists and the work is testing a specific claim? → Write a probe
- No model exists for the domain? → Write an investigation
- Model exists but claim is fundamentally wrong or novel/exploratory? → Start an investigation, then promote
```

The investigation skill's mode detection (newer version) uses a different mechanism:

```
1. Find ### Models (synthesized understanding) section
2. Check for markers: - Summary:, - Critical Invariants:, - Why This Fails:
3. If markers present → Probe Mode (default)
4. If markers absent → Investigation Mode
```

**Contradiction:** CLAUDE.md routes based on **intent** (testing a specific claim vs novel exploration). The investigation skill routes based on **SPAWN_CONTEXT markers** (are model claims injected?). These produce different results:
- A novel exploration in a domain WITH a model → CLAUDE.md says investigation, skill says probe
- A claim test in a domain WITHOUT a model → CLAUDE.md says probe (impossible), skill says investigation

The CLAUDE.md also adds a third case (model exists but claim is wrong → investigation, then promote) that the investigation skill doesn't handle.

**Source:**
- `~/.claude/CLAUDE.md:119-122` — Knowledge Placement artifact decision tree
- `~/.claude/skills/worker/investigation/SKILL.md` (newer version) lines 31-45

**Significance:** Workers see both CLAUDE.md and skill instructions. When they conflict, behavior depends on which the worker follows. The CLAUDE.md is loaded for ALL agents (not just investigation workers), so the routing logic is exposed to orchestrators, architects, and other skills who may interpret it differently.

---

### Finding 7: Coverage Analysis — Which Skills Mention the System

**Evidence:** Comprehensive search of all deployed skills:

| Skill | Mentions Models? | Mentions Probes? | Mentions Investigations? | Probe Meaning |
|-------|-----------------|-----------------|------------------------|---------------|
| **orchestrator** | Yes (stewardship, routing, merge) | Yes (model-scoped probes) | Yes (spawn routing) | Model-scoped |
| **investigation** (newer) | Yes (mode detection) | Yes (probe mode default) | Yes (primary artifact) | Model-scoped |
| **investigation** (older) | No | No | Yes (only artifact) | N/A |
| **architect** | No (via decision-navigation substrate) | Yes (fork probes) | Yes (primary output) | Decision-nav |
| **design-session** | No (via decision-navigation substrate) | Yes (fork probes) | Yes (output option) | Decision-nav |
| **decision-navigation** | Yes (substrate consultation) | Yes (probing protocol) | Yes (delegation option) | Decision-nav |
| **kb-reflect** | No | No | Yes (synthesis, closure) | N/A |
| **codebase-audit** | No | No | No | N/A |
| **feature-impl** | No | No | Yes (phase reference) | N/A |
| **systematic-debugging** | No | No | Yes (investigation file) | N/A |
| **reliability-testing** | No | No | No | N/A |
| **writing-skills** | No | No | No | N/A |
| **worker-base** | No | No | No | N/A |
| **issue-quality** | No | No | No | N/A |
| **meta-orchestrator** (interface) | No | Yes (epic probes) | Yes | Epic model probes |

**Skills that SHOULD mention the system but don't:**
- **capture-knowledge** — Doesn't exist as a deployed skill source, yet appears in the skills list
- **codebase-audit** — Produces findings that could feed into models but has no model/probe awareness
- **kb-reflect** — Handles synthesis (3+ investigations → consolidate) but has no model creation guidance, despite models being the primary synthesis artifact

**Source:**
- `Grep "probe|\.kb/models" ~/.claude/skills/` — found 9 files
- Individual skill reads (architect, design-session, decision-navigation, kb-reflect)

**Significance:** Only 3 skills (orchestrator, investigation-newer, decision-navigation) have substantive probe/model awareness. The rest either don't mention the system or use "probe" in a different sense. This means the model/probe system is primarily an orchestrator concern that most workers are unaware of.

---

### Finding 8: Model README vs Lifecycle Guide Have Different Creation Criteria

**Evidence:** Beyond the threshold number, the criteria themselves differ:

**README.md (entry point for models/):**
```
Signals:
- 3+ investigations on same topic converge to understanding
- You can explain "how X works" coherently
- Multiple downstream decisions/epics will reference this understanding
- Same confusion recurs (scattered understanding needs consolidation)

The test: Can you explain the mechanism in 1-2 paragraphs?
```

**understanding-artifact-lifecycle.md:**
```
When to create:
- Trigger: 15+ investigations on single topic (investigation cluster detected)
- Four-factor test (all required):
  1. HOT - Cluster exists (15+ investigations)
  2. COMPLEX - Has failure modes, constraints, state transitions
  3. OWNED - Our system internals (not external tools)
  4. STRATEGIC_VALUE - "Enable/constrain" answers save hours vs minutes
```

**Anti-patterns section in lifecycle guide:**
```
Creating Model Too Early: 3 investigations on topic → "let's create a model"
Why wrong: Models are synthesis artifacts. 3 investigations don't provide enough perspective.
Fix: Wait for cluster threshold (15+)
Never below 10 investigations.
```

The lifecycle guide EXPLICITLY calls the README.md's threshold an anti-pattern.

**Source:**
- `.kb/models/README.md:9-17`
- `.kb/guides/understanding-artifact-lifecycle.md:179-184,394-400`

**Significance:** The README.md is the first file agents read when encountering the models directory. It sets expectation at "3+" which the lifecycle guide explicitly labels as an anti-pattern. This direct contradiction means agents following the README will create models too early.

---

## Synthesis

**Key Insights:**

1. **The routing is sound but the execution is broken** — The orchestrator skill's probe-vs-investigation routing table is clear and consistent. The problem is downstream: the PROBE.md template doesn't exist, the deployed skill has two versions, and CLAUDE.md's routing mechanism (intent-based) conflicts with the skill's mechanism (marker-based).

2. **"Probe" has three distinct meanings** — Model-scoped probes (investigation skill), decision-navigation probes (experiment to resolve unknown fork), and epic model probes (tracking understanding). These serve different purposes but share a name, creating semantic confusion.

3. **The model creation threshold is unresolved** — README.md says 3+, lifecycle guide says 15+ (with 10 as hard floor), Phase reviews are split. The README explicitly triggers what the lifecycle guide calls an anti-pattern. This needs a single authoritative source.

4. **Most worker skills are model/probe-unaware** — Only the orchestrator and investigation skills participate in the model/probe system. Other understanding-producing skills (codebase-audit, kb-reflect, capture-knowledge) don't route to models or probes, meaning their output stays as investigations even when a model exists for the domain.

**Answer to Investigation Question:**

The model/probe/investigation system is conceptually well-designed at the orchestrator level but has 6 implementation gaps that prevent consistent execution:
1. Missing PROBE.md template (referenced 7+ times, doesn't exist)
2. Two deployed investigation skill versions (probe-aware vs probe-unaware)
3. Contradictory model creation thresholds (3+ vs 15+)
4. Overloaded "probe" terminology (3 distinct meanings)
5. Mismatched routing mechanisms (CLAUDE.md intent-based vs skill marker-based)
6. Low coverage (most worker skills don't participate in the system)

---

## Structured Uncertainty

**What's tested:**

- ✅ `.orch/templates/PROBE.md` does not exist (verified: Read tool returned "File does not exist")
- ✅ Two deployed investigation skills at different paths with different content (verified: Read both files, compared checksums and content)
- ✅ README.md says "3+" and lifecycle guide says "15+" (verified: Read both files, located exact lines)
- ✅ "probe" has different meanings in orchestrator vs decision-navigation skills (verified: Read all relevant skills, cataloged uses)
- ✅ This investigation was spawned with the OLD investigation skill without probe mode (verified: SPAWN_CONTEXT header shows checksum 91b0e65cab3c)

**What's untested:**

- ⚠️ Which deployed investigation skill path gets loaded by `skillc deploy` (depends on tooling configuration not examined)
- ⚠️ Whether the `.skillc/SKILL.md` in source (bc9a9bf69223) differs from both deployed versions (it does differ from both - different checksum - but root cause of three different versions untested)
- ⚠️ Whether any existing probes were created using a PROBE.md template that was later deleted

**What would change this:**

- Finding that PROBE.md template exists at a different path than referenced
- Finding that skill loading always uses the newer path (`worker/` not `src/worker/`), making the old version dead code
- Finding that the README.md "3+" threshold was intentionally kept as a "consider" signal vs the lifecycle guide's "create" trigger

---

## Canonical Decision Tree (As Emerged From All Sources)

This is the decision tree that emerges when synthesizing all sources:

```
UNDERSTAND something about the system?
│
├── Does a model exist for this domain? (check via kb context)
│   │
│   ├── YES (model exists)
│   │   │
│   │   ├── Testing a SPECIFIC CLAIM from the model?
│   │   │   └── Create PROBE in .kb/models/{model-name}/probes/
│   │   │       Template: .orch/templates/PROBE.md [⚠️ MISSING]
│   │   │       Sections: Question, What I Tested, What I Observed, Model Impact
│   │   │       Verdicts: confirms | contradicts | extends
│   │   │       WHO ROUTES: Orchestrator at spawn time
│   │   │       WHO CREATES: Worker (investigation skill, probe mode)
│   │   │       WHO MERGES: Orchestrator during orch complete
│   │   │
│   │   └── Novel exploration or fundamentally wrong claim?
│   │       └── Create INVESTIGATION in .kb/investigations/
│   │           (Then promote/update model after synthesis)
│   │           WHO ROUTES: Orchestrator at spawn time
│   │           WHO CREATES: Worker (investigation skill)
│   │           WHO SYNTHESIZES: Orchestrator
│   │
│   └── NO (no model exists)
│       │
│       ├── Create INVESTIGATION in .kb/investigations/
│       │   WHO CREATES: Worker
│       │
│       └── Should a MODEL be created?
│           ├── [Threshold unclear: 3+ per README vs 15+ per lifecycle guide]
│           ├── Four-factor test: HOT × COMPLEX × OWNED × STRATEGIC_VALUE
│           ├── WHO CREATES: Orchestrator (synthesis is not spawnable)
│           └── Template: .kb/models/TEMPLATE.md
│
MODEL exists and needs updating?
│
├── New failure mode discovered → Add to "Why This Fails"
├── Constraint changed → Update "Constraints"
├── System evolved → Add to "Evolution"
├── WHO UPDATES: Orchestrator (single writer for models)
│
PROBE completed, needs merging?
│
├── Read verdict: confirms | contradicts | extends
├── confirms → Keep model claim, strengthen confidence
├── extends → Merge new invariant/constraint into model
├── contradicts → Update model claim OR create blocking follow-up
├── WHO MERGES: Orchestrator during orch complete review
```

**Gaps in the canonical tree:**
1. No guidance for when to RETIRE a model
2. No guidance for probe FAILURE (what if probe test itself fails?)
3. No guidance for cross-project models (where do integration models live?)

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Create PROBE.md template | implementation | Missing file, template content already defined in skills |
| Remove old investigation skill at src/worker/ path | implementation | Dead code cleanup, newer version supersedes |
| Reconcile model creation threshold | architectural | Cross-document consistency, touches README + guide + reviews |
| Disambiguate "probe" terminology | architectural | Cross-skill semantic alignment |
| Update CLAUDE.md routing to match skill mechanism | architectural | Cross-document consistency |
| Add model/probe awareness to more worker skills | strategic | Scope expansion of knowledge system |

### Recommended Approach ⭐

**Fix the execution chain first** — Create PROBE.md template, remove old investigation skill version, then reconcile documentation.

**Why this approach:**
- The PROBE.md template is the highest-impact fix (unblocks probe creation for all workers)
- Removing the old skill version eliminates deployment ambiguity
- Documentation reconciliation can follow without blocking probe execution

**Trade-offs accepted:**
- Deferring terminology disambiguation (lower impact, harder to resolve)
- Deferring broader skill coverage (requires design discussion)

**Implementation sequence:**
1. Create `.orch/templates/PROBE.md` with 4 required sections from investigation skill — unblocks probe mode
2. Delete or redirect `~/.claude/skills/src/worker/investigation/SKILL.md` — eliminates old version
3. Update `.kb/models/README.md` threshold to match lifecycle guide (15+ with four-factor test, 3+ as "consider" signal)
4. Add note to CLAUDE.md clarifying that marker-based detection is the mechanism, intent is the rationale

### Alternative Approaches Considered

**Option B: Rewrite investigation skill to remove probe mode, keep probes as orchestrator-only concern**
- **Pros:** Simpler worker skill, clear separation of concerns
- **Cons:** Contradicts design intent (workers should be able to produce probes autonomously)
- **When to use instead:** If probe mode causes more confusion than it solves

**Option C: Full terminology refactoring (rename model-scoped "probes" to "model tests")**
- **Pros:** Eliminates semantic overload
- **Cons:** Requires updating 9+ files, existing probes in .kb/models/*/probes/ directories, all documentation
- **When to use instead:** If disambiguation is strategically important enough to warrant the churn

### Implementation Details

**What to implement first:**
- PROBE.md template (highest impact, simplest change)
- Old skill removal (prevents future deployment confusion)

**Things to watch out for:**
- ⚠️ The `src/worker/` vs `worker/` skill path difference may indicate a broader deployment pipeline issue (skillc deploy may write to both)
- ⚠️ Existing probes in `.kb/models/*/probes/` were created without the template — they may not follow the 4-section structure

**Success criteria:**
- ✅ `.orch/templates/PROBE.md` exists with Question, What I Tested, What I Observed, Model Impact sections
- ✅ Only one deployed investigation skill version (probe-aware)
- ✅ README.md and lifecycle guide agree on model creation threshold
- ✅ Next probe-mode investigation worker can find and use the template

---

## References

**Files Examined:**
- `~/.claude/CLAUDE.md:86-133` — Knowledge Placement table, artifact decision tree
- `~/Documents/personal/orch-go/CLAUDE.md` — Project instructions
- `.kb/models/README.md` — Model creation signals and lifecycle
- `.kb/models/TEMPLATE.md` — 6-section model structure
- `.kb/models/PHASE3_REVIEW.md` — N=5 model pattern analysis
- `.kb/models/PHASE4_REVIEW.md` — N=11 model pattern analysis
- `.kb/guides/understanding-artifact-lifecycle.md` — Epic Model → Understanding → Model promotion path
- `.kb/guides/two-tier-sensing-pattern.md` — General sensing pattern (not directly model/probe related)
- `.orch/templates/` — SYNTHESIS.md, SESSION_HANDOFF.md, FAILURE_REPORT.md (no PROBE.md)
- `~/orch-knowledge/skills/src/meta/orchestrator/.skillc/SKILL.md` — Orchestrator skill (models, probes, routing)
- `~/orch-knowledge/skills/src/meta/orchestrator/.skillc/reference/tools-and-commands.md` — Tool reference
- `~/orch-knowledge/skills/src/worker/investigation/.skillc/SKILL.md` — Compiled skill (with probe mode)
- `~/orch-knowledge/skills/src/worker/investigation/.skillc/intro.md` — Probe mode detection source
- `~/orch-knowledge/skills/src/worker/investigation/.skillc/template.md` — Probe/investigation template source
- `~/orch-knowledge/skills/src/worker/investigation/.skillc/completion.md` — Completion criteria source
- `~/.claude/skills/src/worker/investigation/SKILL.md` — Deployed OLD version (no probe mode)
- `~/.claude/skills/worker/investigation/SKILL.md` — Deployed NEW version (probe mode)
- `~/.claude/skills/worker/architect/SKILL.md` — Architect skill
- `~/.claude/skills/worker/design-session/SKILL.md` — Design session skill
- `~/.claude/skills/shared/decision-navigation/SKILL.md` — Decision navigation protocol
- `~/.claude/skills/meta/orchestrator/reference/meta-orchestrator-interface.md` — Meta-orchestrator interface
- `~/orch-knowledge/skills/src/worker/kb-reflect/.skillc/SKILL.md` — KB reflect skill
- `~/orch-knowledge/skills/src/shared/worker-base/.skillc/*.md` — Worker base components

**Commands Run:**
```bash
# Search for probe/model references across skill sources
Grep "probe|\.kb/models/" ~/orch-knowledge/skills/src/ (80 matches across 15+ files)

# Search for probe references in deployed skills
Grep "probe|PROBE" ~/.claude/skills/ (9 files found)

# Verify PROBE.md template existence
Glob .orch/templates/* (3 files, no PROBE.md)
Read .orch/templates/PROBE.md (File does not exist)
```

**Related Artifacts:**
- **Decision:** `.kb/decisions/2026-01-12-models-as-understanding-artifacts.md` — Why models exist
- **Guide:** `.kb/guides/understanding-artifact-lifecycle.md` — Lifecycle progression of understanding artifacts
- **Model:** `.kb/models/PHASE4_REVIEW.md` — Model pattern validation at N=11

---

## Investigation History

**2026-02-13:** Investigation started
- Initial question: What do all skill sources and CLAUDE.md files claim about model/probe/investigation system?
- Context: Orchestrator-initiated audit to catalog and reconcile claims across the documentation system

**2026-02-13:** All source files read (25+ files across 2 repos)
- Read orchestrator skill, investigation skill (3 versions), architect, design-session, decision-navigation, kb-reflect, worker-base, models README/TEMPLATE, lifecycle guide, PHASE3/4 reviews
- Verified PROBE.md template missing, confirmed two deployed investigation skill versions

**2026-02-13:** Investigation completed
- Status: Complete
- Key outcome: 6 critical inconsistencies found; PROBE.md template missing is the highest-impact gap; model creation threshold has 5x range (3 vs 15) across documents
