## Summary (D.E.K.N.)

**Delta:** orch-go's code implements a sophisticated forward path (models → spawn context → agent probe guidance) but has zero reverse path (probe verdicts → model updates); the `DefaultProbeTemplate` constant exists but is dead code, and completion/verification has no probe awareness.

**Evidence:** Traced all 6 Phase 1 findings through Go source code: `pkg/spawn/probes.go` (probe infrastructure), `pkg/spawn/kbcontext.go:728-792` (model injection), `cmd/orch/complete_cmd.go` (zero probe references), `pkg/daemon/skill_inference.go` (no probe routing), `pkg/verify/` (zero probe references).

**Knowledge:** The probe system is architecturally half-built: spawn injects model content and lists recent probes, but completion ignores probe verdicts entirely. The code's `DefaultProbeTemplate` constant at `probes.go:160` is never used - it's dead code from the entropy-spiral branch restoration.

**Next:** Three fixes needed: (1) wire `DefaultProbeTemplate` to `.orch/templates/PROBE.md` or use the constant directly, (2) add probe verdict parsing to `pkg/verify/synthesis_parser.go`, (3) reconcile skill loader path to prevent dual-version investigation skill.

**Authority:** architectural - Cross-component gap (spawn, verify, skills) requires orchestrator coordination.

---

# Investigation: Audit orch-go Implementation of Model/Probe/Investigation System (Phase 2)

**Question:** How does the orch-go CODE actually implement the model/probe/investigation routing documented in Phase 1, and where do gaps exist between documentation and implementation?

**Started:** 2026-02-13
**Updated:** 2026-02-13
**Owner:** Worker agent (spawned by orchestrator)
**Phase:** Complete
**Next Step:** None - findings ready for orchestrator review
**Status:** Complete

**Patches-Decision:** N/A
**Extracted-From:** N/A

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| .kb/investigations/2026-02-13-inv-audit-model-probe-investigation-claims.md | extends | yes | Phase 1 said PROBE.md template doesn't exist - partially wrong, `DefaultProbeTemplate` exists as dead code in probes.go |
| .kb/investigations/2026-02-13-inv-restore-models-probes-go-infrastructure.md | confirms | yes | Confirms probes.go was restored from entropy-spiral branch |

---

## Findings

### Finding 1: Forward Path (Models → Spawn Context) Is Fully Implemented

**Evidence:** The code builds a complete pipeline from model files to spawn context:

1. `pkg/spawn/kbcontext.go:107-148` — `RunKBContextCheck()` shells out to `kb context` CLI with tiered search (local first, global fallback)
2. `pkg/spawn/kbcontext.go:322-449` — `parseKBContextOutput()` parses kb output into typed matches including "model" type
3. `pkg/spawn/kbcontext.go:645-650` — Models rendered under `### Models (synthesized understanding)` header
4. `pkg/spawn/kbcontext.go:744-792` — `formatModelMatchForSpawn()` extracts Summary, Critical Invariants, Why This Fails sections from model files
5. `pkg/spawn/kbcontext.go:779` — Appends "Your findings should confirm, contradict, or extend the claims above."
6. `pkg/spawn/kbcontext.go:782-789` — Injects recent probes via `ListRecentProbes()` from `.kb/models/{model}/probes/`
7. `pkg/spawn/context.go:83-85` — Template inserts `{{.KBContext}}` into SPAWN_CONTEXT.md
8. `cmd/orch/spawn_cmd.go:1220` — KBContext assigned to Config struct

**Source:** Verified by reading all files and tracing data flow. The SPAWN_CONTEXT for THIS investigation (shown in my own context) demonstrates the working forward path — model summaries, invariants, failure modes, and probe references are all present.

**Significance:** Phase 1's Finding 5 said "orchestrator routing is sound but worker execution is broken." The code shows the forward path is solid. The orch-go code correctly injects model content that creates the detection markers (`- Summary:`, `- Critical Invariants:`, `- Why This Fails:`) that the investigation skill's probe mode relies on. The issue is elsewhere.

---

### Finding 2: DefaultProbeTemplate Exists as Dead Code

**Evidence:** `pkg/spawn/probes.go:160-203` defines a `DefaultProbeTemplate` constant with the exact 4-section structure referenced by skills (Question, What I Tested, What I Observed, Model Impact). However:

- **Zero usages:** `grep -rn "DefaultProbeTemplate" pkg/ cmd/` only matches the definition and the probe restoration investigation. No other Go code references it.
- **Comment says:** `"The full template is expected to come from kb-cli (orch-go-xxa6e)."` — indicating this was a placeholder waiting for kb-cli integration.
- **`.orch/templates/PROBE.md` does not exist** — verified via `Glob .orch/templates/*` (3 files: FAILURE_REPORT.md, SYNTHESIS.md, SESSION_HANDOFF.md).

**Source:**
- `pkg/spawn/probes.go:160-161` — definition and comment
- `Grep "DefaultProbeTemplate"` across entire project — only 3 matches (definition, spawn context doc, investigation)

**Significance:** Phase 1 said the PROBE.md template is missing. The reality is more nuanced: a template EXISTS in code (`DefaultProbeTemplate` constant) but is never used. The file `.orch/templates/PROBE.md` doesn't exist either. The template is doubly stranded — it's neither a file nor wired into any function. This is dead code from the entropy-spiral branch restoration.

---

### Finding 3: Completion and Verification Have ZERO Probe Awareness

**Evidence:** Exhaustive search of completion and verification code:

- `Grep "probe|model|\.kb/models" cmd/orch/complete_cmd.go` — zero matches (only "model" in the word "telemetry" context, line 1096)
- `Grep "probe|model|\.kb/models" pkg/verify/` — zero matches across ALL verify package files
- `pkg/verify/synthesis_parser.go:44-92` — `ParseSynthesis()` extracts Agent, Issue, Duration, Outcome, D.E.K.N., NextActions — NO Model Impact section
- `pkg/verify/discovered_work.go:63-97` — `CollectDiscoveredWork()` extracts NextActions, AreasToExplore, Uncertainties — NO probe verdicts
- `pkg/verify/check.go:11-26` — 11 verification gates defined — NO probe processing gate
- `pkg/verify/synthesis_opportunities.go:70-169` — `DetectSynthesisOpportunities()` scans investigations for clustering — does NOT check probe results

**Source:** All files in `cmd/orch/complete*.go` and `pkg/verify/` read and searched.

**Significance:** This is the core architectural gap. The system creates a **one-way flow**: models are injected at spawn, agents produce probes, but `orch complete` never reads probe verdicts back. The guidance "Your findings should confirm, contradict, or extend the claims above" at `kbcontext.go:779` creates an expectation that's never fulfilled. Probe files accumulate in `.kb/models/{model}/probes/` but are only ever read forward (listing recent probes for next spawn), never backward (merging verdicts into models).

---

### Finding 4: Daemon Routing Has No Probe vs Investigation Distinction

**Evidence:** `pkg/daemon/skill_inference.go:27-42` — `InferSkill()` maps issue types directly to skills:

```
bug → "architect"
feature → "feature-impl"
task → "feature-impl"
investigation → "investigation"
```

Priority order at `skill_inference.go:96-113`: `skill:*` label > title pattern > issue type. No check for whether a model exists for the investigation domain. The daemon passes the issue to `orch spawn`, which runs kb context, which may inject model content, which the agent then uses for probe mode detection.

**Source:**
- `pkg/daemon/skill_inference.go:27-42` — `InferSkill()` function
- `pkg/daemon/skill_inference.go:96-113` — `InferSkillFromIssue()` master function
- `pkg/daemon/daemon.go:603` — where daemon calls `InferSkillFromIssue()`

**Significance:** The probe vs investigation decision is deliberately NOT made at the daemon or orch-go level. It's made by the spawned agent after reading SPAWN_CONTEXT markers. This is architecturally sound — the code defers the decision to the point where model context is available. But it means the daemon has no ability to prioritize probe work or track probe-vs-investigation ratios.

---

### Finding 5: Skill Loader Uses First-Match-Wins, Explaining Dual Versions

**Evidence:** `pkg/skills/loader.go:51-88` — `FindSkillPath()` searches two patterns:
1. `~/.claude/skills/{skillName}/SKILL.md` (direct, line 61)
2. `~/.claude/skills/{category}/{skillName}/SKILL.md` (subdirectory, lines 66-85)

Returns on first match with no disambiguation. This means:
- `~/.claude/skills/investigation/SKILL.md` would match pattern 1 (if it exists)
- `~/.claude/skills/worker/investigation/SKILL.md` matches pattern 2
- `~/.claude/skills/src/worker/investigation/SKILL.md` matches pattern 2 (category=`src`)

The Phase 1 finding of two deployed versions (`src/worker/` at checksum 91b0e65cab3c vs `worker/` at checksum 1cf402739ec0) is explained by `skillc deploy` writing to both paths at different times, and the loader picking whichever it finds first.

**Source:**
- `pkg/skills/loader.go:51-88` — `FindSkillPath()` with search pattern
- `pkg/skills/loader.go:66-85` — subdirectory iteration

**Significance:** The loader has no version checking, no checksum validation, and no "prefer newer" logic. Whichever path the `os.ReadDir()` iteration encounters first wins. The Phase 1 finding of two conflicting deployed investigation skills is a deployment pipeline issue (skillc writes to multiple paths), not an orch-go loading issue.

---

### Finding 6: Spawn Context Correctly Creates Probe Mode Detection Markers

**Evidence:** The investigation skill's probe mode detection relies on finding these markers in SPAWN_CONTEXT:
- `### Models (synthesized understanding)` header
- `- Summary:` sub-item
- `- Critical Invariants:` sub-item
- `- Why This Fails:` sub-item

The orch-go code generates exactly these markers at:
- `kbcontext.go:646` — `### Models (synthesized understanding)` header
- `kbcontext.go:764` — `"  - Summary:\n"`
- `kbcontext.go:769` — `"  - Critical Invariants:\n"`
- `kbcontext.go:773` — `"  - Why This Fails:\n"`

Additionally, `kbcontext.go:728-742` — `hasInjectedModelContent()` checks whether these sections exist, tracked via `FormatContextForSpawnWithLimit()` result's `HasInjectedModels` field at line 508/603.

**Source:** Direct code reading of `pkg/spawn/kbcontext.go:645-792`

**Significance:** The code and the skill documentation are ALIGNED on marker format. This is one of the few areas where code and docs agree perfectly. The forward path marker injection works. Phase 1's Finding 6 (CLAUDE.md routing contradicts skill mechanism) is a documentation-level conflict, not a code-level one — the code only implements the marker-based mechanism.

---

## Synthesis

**Key Insights:**

1. **The code implements a half-duplex probe system** — Forward path (models → spawn context → agent guidance) is complete and well-tested (`probes.go`, `probes_test.go`, `kbcontext.go`). Reverse path (probe verdicts → model updates) is entirely absent from completion/verification code. This makes the probe system write-only from a model evolution perspective.

2. **`DefaultProbeTemplate` is dead code waiting for integration** — The constant exists in `probes.go:160-203` with a comment referencing a beads issue (`orch-go-xxa6e`) for kb-cli integration. The `.orch/templates/PROBE.md` file was never created. Both the skill's reference to the template file and the code's unused constant point to incomplete integration that was interrupted by the entropy-spiral recovery.

3. **The probe routing decision is correctly delegated to agents** — Neither the daemon nor `orch spawn` makes probe vs investigation decisions. Instead, the code injects model content markers, and the investigation skill detects them. This is architecturally sound — the decision is made where the context is available. But it means the system has no central probe tracking.

4. **Skill loader path ambiguity enables deployment drift** — The "first match wins" strategy at `loader.go:51-88` means duplicate skill files at different paths (`src/worker/` vs `worker/`) don't error — they silently pick one. This is the root cause of Phase 1's Finding 2 (two conflicting investigation skill versions).

**Answer to Investigation Question:**

The orch-go code implements the model/probe/investigation system as a **well-built forward pipeline with a missing reverse channel**:

| Phase 1 Finding | Code Reality |
|-----------------|-------------|
| 1. PROBE.md template missing | `DefaultProbeTemplate` exists as dead code in `probes.go:160`; file never created |
| 2. Two conflicting investigation skills | Skill loader (`loader.go:51-88`) uses first-match-wins; no duplicate detection |
| 3. Model creation threshold contradicts | Not in orch-go code — this is pure documentation |
| 4. Probe term overloaded | orch-go code only uses model-scoped probe meaning; no decision-navigation probes |
| 5. Orchestrator routing sound but worker broken | Forward path works perfectly; reverse path (completion) is missing |
| 6. Only 3/15+ skills model-aware | Correct — `kbcontext.go` injects models for ALL skills, but only investigation skill acts on markers |

---

## Structured Uncertainty

**What's tested:**

- ✅ `DefaultProbeTemplate` is defined at `probes.go:160` and never referenced elsewhere (verified: `Grep "DefaultProbeTemplate"` across project)
- ✅ Zero probe/model references in `complete_cmd.go` and `pkg/verify/` (verified: `Grep` of all files)
- ✅ `InferSkill()` maps issue types to skills without model-existence check (verified: read `skill_inference.go:27-42`)
- ✅ `formatModelMatchForSpawn()` creates exact markers investigation skill expects (verified: read `kbcontext.go:764-773`)
- ✅ `ListRecentProbes()` correctly discovers probes in `.kb/models/{model}/probes/` (verified: read `probes.go:84-131` and `probes_test.go`)
- ✅ Skill loader uses first-match-wins with no version disambiguation (verified: read `loader.go:51-88`)

**What's untested:**

- ⚠️ Whether the loader actually picks `src/worker/investigation` vs `worker/investigation` in practice (depends on `os.ReadDir` ordering, which is filesystem-dependent)
- ⚠️ Whether `HasInjectedModels` flag at `kbcontext.go:508` is used downstream for any behavior beyond tracking (searched but may have missed usage)
- ⚠️ Whether the kb-cli integration referenced in `probes.go:161` comment (`orch-go-xxa6e`) was ever completed

**What would change this:**

- Finding that `DefaultProbeTemplate` is used by kb-cli code (not in orch-go)
- Finding that completion processes probes via a hook or external script (not in Go code)
- Finding that the investigation skill at `src/worker/` path is intentionally kept for backwards compatibility

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Wire `DefaultProbeTemplate` to a usable location | implementation | Single-file change, no cross-boundary impact |
| Add probe verdict parsing to synthesis_parser.go | architectural | Cross-component (verify + spawn), affects completion behavior |
| Clean up duplicate investigation skill paths | implementation | Deployment hygiene, `skillc deploy` is the tool |
| Add probe processing to `orch complete` | architectural | New completion gate, affects all future probe completions |
| Reconcile model creation threshold in docs | implementation | Pure documentation, no code change |

### Recommended Approach: Wire Forward Path First, Then Build Reverse

**Why this approach:**
- The forward path is already working — agents are guided to create probes
- Probes are accumulating in `.kb/models/{model}/probes/` already
- The reverse path (reading verdicts back) can be incrementally added to `orch complete`
- Fixing dead code (`DefaultProbeTemplate`) and deployment drift (dual skill versions) are quick wins

**Trade-offs accepted:**
- Deferring full reverse path (probe → model merge) to a separate issue
- Manual model updates by orchestrators remain the only merge mechanism for now

**Implementation sequence:**
1. **Write `.orch/templates/PROBE.md`** from `DefaultProbeTemplate` content — unblocks template file reference in skills
2. **Clean duplicate investigation skill** — remove `src/worker/investigation/` or redirect to `worker/investigation/`
3. **Add `ModelImpact` parsing to `pkg/verify/synthesis_parser.go`** — extract Confirms/Contradicts/Extends from SYNTHESIS.md
4. **Surface probe verdicts in `orch complete` output** — display to orchestrator for manual merge
5. **Future: automated probe merge** — update model files based on probe verdicts (requires design)

### Alternative Approaches Considered

**Option B: Remove probe mode from investigation skill entirely**
- **Pros:** Simplifies skill, removes the probe/investigation bifurcation
- **Cons:** Loses the lightweight testing pattern that probes enable
- **When to use instead:** If probes are determined to add more confusion than value

**Option C: Build full probe → model automation first**
- **Pros:** Completes the feedback loop immediately
- **Cons:** Complex, touches many files, high risk of breaking completion for all agents
- **When to use instead:** If probe accumulation is causing stale models to persist

---

## References

**Files Examined:**
- `pkg/spawn/probes.go` — Probe infrastructure (204 lines): path helpers, listing, formatting, dead template
- `pkg/spawn/probes_test.go` — Comprehensive probe tests (270 lines)
- `pkg/spawn/kbcontext.go` — KB context integration (930+ lines): query, parse, format, model injection, probe injection
- `pkg/spawn/context.go` — SPAWN_CONTEXT template (519 lines)
- `pkg/spawn/config.go` — Config struct with KBContext field
- `pkg/spawn/skill_requires.go` — Skill-driven context gathering (200 lines)
- `pkg/spawn/orchestrator_context.go` — Orchestrator spawn template
- `pkg/spawn/meta_orchestrator_context.go` — Meta-orchestrator spawn template
- `cmd/orch/spawn_cmd.go` — Spawn command (2000+ lines): context gathering, config assembly
- `cmd/orch/complete_cmd.go` — Complete command (1660 lines): no probe references
- `pkg/verify/synthesis_parser.go` — Synthesis parsing (224 lines): no Model Impact parsing
- `pkg/verify/discovered_work.go` — Discovered work extraction (195 lines): no probe verdicts
- `pkg/verify/synthesis_opportunities.go` — Synthesis detection (232 lines): no probe awareness
- `pkg/verify/check.go` — Verification gates (26 lines): no probe gate
- `pkg/daemon/skill_inference.go` — Skill inference (113 lines): no probe routing
- `pkg/daemon/daemon.go` — Daemon main (864 lines): calls InferSkillFromIssue
- `pkg/skills/loader.go` — Skill loader (190 lines): first-match-wins path resolution

**Commands Run:**
```bash
# Search for probe references in Go code
Grep "probe|PROBE" --type go  # Found in pkg/spawn/probes.go, kbcontext.go only

# Search for .kb/models references in Go code
Grep "\.kb/models" --type go  # Found in pkg/spawn/probes.go, probes_test.go only

# Search for probe/model in completion code
Grep "probe|model|\.kb/models" cmd/orch/complete*.go  # Zero matches

# Search for probe/model in verify package
Grep "probe|model|\.kb/models" pkg/verify/  # Zero matches

# Search for template references
Grep "DefaultProbeTemplate" --type go  # Only definition and references
Grep "PROBE\.md|probe.*template" --type go  # Only definition

# Search for skill inference
Grep "InferSkill" --type go  # Found in pkg/daemon/skill_inference.go
```

**Related Artifacts:**
- **Phase 1:** `.kb/investigations/2026-02-13-inv-audit-model-probe-investigation-claims.md` — Documentation audit this extends
- **Restoration:** `.kb/investigations/2026-02-13-inv-restore-models-probes-go-infrastructure.md` — How probes.go was restored
- **Model:** `.kb/models/spawn-architecture/model.md` — Spawn architecture model referenced by probe system

---

## Investigation History

**2026-02-13:** Investigation started
- Initial question: How does orch-go code implement the model/probe/investigation routing found in Phase 1?
- Context: Phase 2 of two-phase audit, extending Phase 1's documentation findings to code

**2026-02-13:** Parallel code exploration completed
- Launched 3 parallel agents to explore: (1) spawn context generation, (2) completion/verification, (3) daemon/skill loading
- Direct grep searches for probe, model, kb context references across all Go files
- Read 17 Go source files covering spawn, verify, daemon, and skills packages

**2026-02-13:** Investigation completed
- Status: Complete
- Key outcome: Forward path (models → spawn → agent) fully implemented; reverse path (probe verdicts → model updates) completely absent; `DefaultProbeTemplate` is dead code
