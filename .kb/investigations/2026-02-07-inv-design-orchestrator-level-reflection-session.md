## Summary (D.E.K.N.)

**Delta:** Designed a hybrid, event-driven orchestrator reflection protocol with scoped sessions and explicit prioritization rules.

**Evidence:** `kb reflect` output shows high-volume low-coherence candidates (e.g., keyword clusters like `quick` and `document`) and validates overload risk in full-scope sessions.

**Knowledge:** Reflection creates value when orchestrator judgment selects a small number of high-amnesia-tax items and converts them into durable artifacts.

**Next:** Promote the attached decision record and implement minimal tooling support (session state + pre-scored candidates) without embedding heavy policy into skill prompt text.

**Authority:** strategic - This sets cross-session operating policy for orchestrator behavior and impacts long-term knowledge-system governance.

---

# Investigation: Design Orchestrator Level Reflection Session

**Question:** What orchestrator-level protocol should govern reflection sessions so they happen reliably, stay scoped, and produce measurable reductions in rediscovery work?

**Started:** 2026-02-07
**Updated:** 2026-02-07
**Owner:** OpenCode agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

**Patches-Decision:** `.kb/decisions/2026-02-07-orchestrator-reflection-session-protocol.md`
**Extracted-From:** N/A

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| `2025-12-21-inv-design-self-reflection-protocol-specification.md` | extends | yes | none |
| `2025-12-21-inv-reflection-checkpoint-pattern-agent-sessions.md` | extends | yes | none |
| `2025-12-21-inv-temporal-signals-autonomous-reflection.md` | deepens | yes | none |
| `2026-01-06-inv-automated-reflection-daemon-kb-reflect.md` | deepens | yes | none |

---

## Findings

### Finding 1: Full-scope reflection output creates triage overload

**Evidence:** Running `kb reflect` returns a long list across synthesis opportunities, stale decisions, open actions, principle candidates, and skill candidates in one pass.

**Source:** Command output from `kb reflect` on 2026-02-07 in `/Users/dylanconlin/Documents/personal/orch-go`.

**Significance:** A protocol must scope each session to avoid review fatigue and low-quality promotion decisions.

---

### Finding 2: Keyword-density clustering is useful for surfacing but weak for coherence

**Evidence:** High-ranked clusters include semantically broad tokens (`quick`, `document`) alongside coherent clusters (`rebuild`, `untracked`).

**Source:** `kb reflect` synthesis list (2026-02-07), including mixed-coherence candidate groups.

**Significance:** Prioritization cannot rely on investigation count alone; orchestrator-level coherence judgment is required.

---

### Finding 3: Existing principles and decisions support signal-triggered, system-level reflection

**Evidence:** `Reflection Before Action` and `Premise Before Solution` principles require process-level improvement before one-off fixes; existing knowledge decisions state reflection should be signal-triggered and orchestrator-driven.

**Source:** `~/.kb/principles.md:697`, `~/.kb/principles.md:796`, and reflection decisions in current kb context (`Temporal density and repeated constraints are highest value signals`, `Self-reflection is signal-triggered not time-scheduled`, `Reflection value comes from orchestrator review + follow-up`).

**Significance:** The protocol should be event-driven with explicit forks for what to automate versus what remains human synthesis work.

---

## Synthesis

**Key Insights:**

1. **Cadence must be hybrid, not purely calendar-based** - Weekly-only aspirations fail without event triggers tied to work volume and recurrence.

2. **Session scope should be lane-based** - One primary lane per session preserves decision quality while still allowing lightweight hygiene sweeps.

3. **Success should be measured by rediscovery reduction, not output volume** - More promoted artifacts are not useful unless repeat investigations and stale drift decline.

**Answer to Investigation Question:**

Use a hybrid reflection protocol: trigger sessions from concrete events (investigation volume, recurrence spikes, milestone completions) with a light time floor for safety; run one primary lane per session; prioritize candidates by an amnesia-tax rubric instead of raw counts; and evaluate effectiveness by reduced rediscovery and improved citation/follow-through. Automation should surface and pre-score candidates, while orchestrator judgment decides coherence, intervention type, and promotion.

---

## Structured Uncertainty

**What's tested:**

- ✅ `kb reflect` currently emits broad, high-volume mixed-signal outputs (verified by direct command run)
- ✅ Reflection principles in substrate explicitly favor process-level responses (verified in `~/.kb/principles.md`)
- ✅ Existing kb context already encodes signal-triggered reflection behavior (verified through spawn context evidence)

**What's untested:**

- ⚠️ Quantitative metric thresholds (e.g., exact trigger counts) are not backtested on historical data yet
- ⚠️ Estimated session duration targets (30-45 minutes) are not yet validated across multiple weeks
- ⚠️ Proposed amnesia-tax scoring reliability has not been benchmarked for inter-rater consistency

**What would change this:**

- If backtesting shows trigger thresholds fire too often or too rarely, cadence thresholds should be tuned.
- If lane-scoped sessions miss critical cross-lane signals, add an explicit quarterly full-spectrum review.
- If pre-scoring quality is poor, keep scoring fully manual until better features are available.

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Adopt hybrid event-driven cadence with time floor | strategic | Defines operating rhythm across orchestrator sessions and affects cross-project behavior |
| Use one-lane-per-session scope with maintenance sweep | architectural | Changes orchestrator workflow and review boundaries |
| Add amnesia-tax ranking rubric and telemetry | architectural | Requires coordinated changes in kb reflect output and orchestrator process |
| Keep coherence/promotion judgment manual | strategic | Preserves human strategic control over knowledge quality |

### Recommended Approach ⭐

**Hybrid Triggered Reflection Protocol** - Run reflection when signals indicate likely rediscovery tax, with a lightweight calendar safety floor.

**Why this approach:**
- It matches observed behavior: signal-triggered reflection is already the only reliable pattern.
- It reduces cognitive overload by constraining review scope.
- It preserves orchestrator strategic synthesis instead of delegating coherence to heuristics.

**Trade-offs accepted:**
- Slightly more operational complexity than a weekly cron model.
- Some high-value items may wait until their lane is active.

**Implementation sequence:**
1. Record the protocol in a decision artifact (policy baseline).
2. Add lightweight state tracking (`last_reflection_at`, lane, outcomes) and pre-scored candidates.
3. Run 3-4 reflection cycles and tune thresholds based on rediscovery metrics.

### Alternative Approaches Considered

**Option B: Fixed weekly full-spectrum reflection**
- **Pros:** Easy to remember, predictable schedule.
- **Cons:** Known to fail in practice and reproduces overload.
- **When to use instead:** Small repos with low investigation volume.

**Option C: Fully automated reflection promotion**
- **Pros:** Lowest orchestrator time burden.
- **Cons:** High false-positive risk and weak coherence judgment.
- **When to use instead:** Narrow, high-structure domains with strict taxonomy.

**Rationale for recommendation:** Hybrid triggers plus scoped lanes maximize signal quality while keeping strategic judgment where it belongs.

---

### Implementation Details

**What to implement first:**
- Decision record for protocol governance.
- Reflection run log schema (`.kb/reflection-log.jsonl` or equivalent).
- Candidate ranking fields in `kb reflect` output (`recurrence`, `impact proxy`, `coherence confidence`, `actionability`).

**Things to watch out for:**
- ⚠️ Metric gaming (promoting easy artifacts to improve throughput).
- ⚠️ Overfitting thresholds to one noisy week.
- ⚠️ Conflating stale-citation artifacts with low value.

**Areas needing further investigation:**
- Historical backtest of recurrence and amnesia-tax scoring.
- Correlation between citation lag and actual utility.
- Whether skill-update candidates need a separate cadence from guide/decision hygiene.

**Success criteria:**
- ✅ 30-day repeat-investigation rate drops for top recurring clusters.
- ✅ Median time-to-first-citation for new decisions/guides improves.
- ✅ Reflection sessions produce 1-3 durable actions each without backlog inflation.

---

## References

**Files Examined:**
- `~/.kb/principles.md` - principle constraints for reflection and premise validation
- `.kb/decisions/2026-01-07-synthesis-is-strategic-orchestrator-work.md` - role boundary precedent for strategic synthesis

**Commands Run:**
```bash
# Verify workspace path and report planning phase
pwd && orch phase orch-go-21461 Planning "Designing orchestrator-level reflection session protocol"

# Create investigation artifact
kb create investigation design-orchestrator-level-reflection-session

# Report investigation path for orchestrator verification
bd comment orch-go-21461 "investigation_path: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-02-07-inv-design-orchestrator-level-reflection-session.md"

# Collect current reflection output for evidence
kb reflect
```

**External Documentation:**
- N/A

**Related Artifacts:**
- **Decision:** `.kb/decisions/2026-02-07-orchestrator-reflection-session-protocol.md` - recommended protocol definition

---

## Investigation History

**2026-02-07 00:00:** Investigation started
- Initial question: Define orchestrator-level reflection protocol (cadence, scope, prioritization, metrics, automation boundary, artifact placement)
- Context: Existing `kb-reflect` worker skill handles mechanics; orchestrator operating protocol is missing

**2026-02-07 00:00:** Evidence collection completed
- Ran `kb reflect`, inspected principle constraints, and reviewed prior strategic decision precedent

**2026-02-07 00:00:** Investigation completed
- Status: Complete
- Key outcome: Hybrid event-driven reflection protocol defined and captured in decision record
