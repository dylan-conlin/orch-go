# Session Synthesis

**Agent:** og-research-investigate-facebook-hyperagents-26mar-5ace
**Issue:** orch-go-2mfvw
**Duration:** 2026-03-26 → 2026-03-27
**Outcome:** success

---

## Plain-Language Summary

Facebook/Meta Research published HyperAgents (March 2026), a system where AI agents can modify their own improvement mechanisms. I read the full paper and codebase to understand the concrete mechanisms and compare them to orch-go's knowledge accretion patterns. The most striking finding: when given a minimal starting point (just "modify any part of the codebase"), the HyperAgents meta agent independently discovers persistent memory, performance tracking, evaluation analysis, and template systems — the same coordination infrastructure that orch-go built deliberately over three months. This is the first external validation of the knowledge accretion model's substrate independence claim from a research group that has never seen orch-go. The paper also reveals *why* coordination patterns transfer across domains: meta-level improvements (the coordination infrastructure itself) transfer while task-level improvements don't — providing a mechanism for the substrate independence that the model observed but couldn't explain.

---

## TLDR

Read the full HyperAgents paper and codebase. Extended the knowledge accretion model with first external validation (Meta Research independently discovers same attractor patterns), a mechanism for substrate independence (meta-level coordination transfers, task-level doesn't), and evolutionary selection as a new coordination category in the effectiveness hierarchy.

---

## Delta (What Changed)

### Files Created
- `.kb/models/knowledge-accretion/probes/2026-03-26-probe-hyperagents-external-validation-meta-level-accretion.md` — Full probe documenting HyperAgents analysis against knowledge accretion model

### Files Modified
- `.kb/models/knowledge-accretion/model.md` — Merged probe findings:
  - Added evolutionary agent populations as 6th confirmed substrate in generalization table
  - Extended effectiveness hierarchy with evolutionary selection as structural attractor
  - Updated KA-5 invariant with external validation and meta-level/task-level transfer mechanism
  - Added evolution entry documenting the probe

### Commits
- (pending) Knowledge artifacts from session

---

## Evidence (What Was Observed)

### Paper Analysis (arXiv:2603.19461)

**Architecture:** HyperAgents integrates a task agent (solves domain tasks) and meta agent (modifies both task agent and itself) into a single editable Python codebase. Self-modification is implemented as literal file editing inside Docker containers — the meta agent uses bash and editor tools to modify its own source code.

**Key experimental results:**
- Cross-domain transfer (Section 5.2): Meta agent trained on paper review + robotics transferred to Olympiad math grading achieves imp@50 = 0.630. Domain-specific (DGM) achieves 0.0 on same transfer. This proves meta-level improvements are domain-independent.
- Ablations: Without self-modification (fixed meta agent) → little improvement. Without open-ended exploration (no archive) → fails. Both mechanisms are necessary.
- Emergent capabilities (Section 4.3): PerformanceTracker, MemoryTool, evaluation analysis, compute-aware planning, bias detection, prompt templates — all discovered autonomously without engineering.

### Codebase Analysis (github.com/facebookresearch/HyperAgents)

- Meta agent core: ~15 lines. Instruction: "Modify any part of the codebase at '{repo_path}'."
- Selection: sigmoid-transformed performance score × novelty bonus (1/(1+children_count))
- Isolation: Docker container per generation, changes captured as git diffs
- Cost: 88.6M tokens per 100 iterations

### Mapping to orch-go

| HyperAgents | orch-go | Model Concept |
|-------------|---------|---------------|
| Meta agent | Skill system + daemon routing | Orchestration layer |
| Task agent | Spawned worker agents | Substrate layer |
| Archive (population) | .kb/ knowledge base | Accumulated cross-run state |
| Parent selection | Daemon triage + human review | Structural attractor (performance-weighted) |
| Compilation check | go build, model-stub precommit | Hard gate (unbypassable) |
| Emergent PerformanceTracker | Probe system | Attractor for evaluation data |
| Emergent MemoryTool | .kb/ investigations/threads | Attractor for persistent knowledge |

---

## Architectural Choices

No architectural choices — this was a research/analysis task within existing patterns.

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/models/knowledge-accretion/probes/2026-03-26-probe-hyperagents-external-validation-meta-level-accretion.md`

### Key Findings

1. **Meta-level/task-level separation explains substrate independence.** HyperAgents shows quantitatively that meta-level improvements (coordination infrastructure) transfer across domains while task-level improvements don't. This provides the *mechanism* for why the same attractor/gate/entropy patterns appear across orch-go's substrates.

2. **Evolutionary selection is a structural attractor.** Performance-weighted parent selection embeds coordination in system structure (evaluation function), not in runtime LLM decisions. Extends the effectiveness hierarchy with a new category.

3. **Emergent attractor convergence is real.** An independent system discovers the same coordination patterns (memory, tracking, evaluation, templates) without any domain engineering. Strongest external evidence for the artifact-attractors thread.

4. **Selection pressure as alternative to gates.** HyperAgents manages accretion through evolution (bad variants die) rather than gates (bad contributions blocked). This works because evaluation is automated. orch-go can't do this because evaluation of open-ended knowledge work requires human judgment — the verification bottleneck may be inherent to open-ended domains.

5. **Epistemic status tilts toward substrate-general.** Cross-domain transfer weakens training echo and architectural explanations, strengthens substrate-general. But training confound (LLM applying software engineering knowledge from training) prevents definitive resolution.

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete
- [x] Probe file created and model updated
- [x] Investigation findings merged into parent model
- [x] Ready for `orch complete orch-go-2mfvw`

---

## Unexplored Questions

**Questions that emerged during this session:**
- Could orch-go incorporate selection pressure for investigation synthesis? (When 3+ agents investigate same question, use automated evaluation to select best synthesis rather than expecting each to check prior work)
- Does meta agent coordination infrastructure degrade over generations? (Knowledge accretion within the meta agent itself — paper doesn't measure this)
- HyperAgents' self-modified parent selection "did not outperform the handcrafted mechanism" — one data point, but consistent with gate lifecycle arc

**What remains unclear:**
- Whether the training confound can ever be resolved (LLM applying training knowledge vs discovering substrate-general laws)
- Whether HyperAgents' approach (let accretion happen, apply selection pressure) could work for open-ended domains with well-defined sub-metrics

---

## Friction

Friction: none — smooth session. Paper was accessible, codebase was well-structured, model context was comprehensive.

---

## Session Metadata

**Skill:** research
**Model:** claude-opus-4-6
**Workspace:** `.orch/workspace/og-research-investigate-facebook-hyperagents-26mar-5ace/`
**Probe:** `.kb/models/knowledge-accretion/probes/2026-03-26-probe-hyperagents-external-validation-meta-level-accretion.md`
**Beads:** `bd show orch-go-2mfvw`
