# Two-Tier Sensing Pattern

**Pattern Type:** Architecture
**Created:** 2026-01-10
**Context:** Discovered while designing orchestrator coaching plugin (orch-go-tjn1r)

---

## Summary

A design pattern for systems that need constant monitoring combined with selective, expensive reasoning. Separates cheap pattern detection (tier 1) from expensive contextual investigation (tier 2).

**Core principle:** Pattern matching is cheap but dumb. Reasoning is smart but expensive. Use pattern matching as a filter that reduces the problem space for reasoning.

---

## The Pattern

```
Tier 1: Cheap Detection (constant monitoring)
├─ Fast, deterministic heuristics (regex, counters, thresholds)
├─ Runs on every event/transaction
├─ Low false negative rate (catch everything suspicious)
└─ Emits alert when pattern matches
         ↓
Tier 2: Expensive Investigation (selective reasoning)
├─ LLM reasoning, tool use, context gathering
├─ Triggered ONLY when tier 1 alerts
├─ Validates: Real issue or false positive?
└─ Decides action (intervene, escalate, or dismiss)
```

---

## When To Use

**Use this pattern when:**
- System needs constant monitoring (can't miss events)
- Reasoning is expensive (tokens, latency, human time)
- False positives are costly (investigation fatigue)
- Events have high volume but low signal (most are benign)

**Don't use this pattern when:**
- Events are rare (just investigate everything)
- Detection rules are perfect (no false positives)
- Investigation is cheap (no need to filter)
- Reasoning can run on every event (no cost constraint)

---

## Economics

The pattern works because of **cost asymmetry**:

| Tier | Cost | Frequency | Total Cost |
|------|------|-----------|------------|
| Detection | $0.0001 | Every event (1000/day) | $0.10/day |
| Investigation | $0.10 | Only alerts (10/day) | $1.00/day |

If investigation ran on every event: $100/day (100x more expensive)

**Key insight:** A 10% false positive rate in tier 1 is acceptable if it reduces tier 2 load by 90%.

---

## Implementation Strategies

### Strategy 1: Rule-Based Detection + LLM Investigation

**Tier 1:** Hand-written rules (regex, thresholds, state machines)
**Tier 2:** LLM with tools (Read context, analyze, decide)

**Pros:**
- Predictable, fast detection
- Explainable (know why it triggered)

**Cons:**
- Rules brittle, need tuning
- Can't adapt to new patterns

**Example:** Orchestrator coaching plugin (this codebase)

### Strategy 2: ML Classifier + LLM Validator

**Tier 1:** Trained classifier (logistic regression, small neural net)
**Tier 2:** LLM validates + explains decision

**Pros:**
- Can learn from data
- Better generalization than rules

**Cons:**
- Needs training data
- Less explainable detection

**Example:** Content moderation (flag inappropriate content → LLM validates context)

### Strategy 3: Statistical Anomaly + LLM Root Cause

**Tier 1:** Statistical thresholds (mean + 3σ, percentiles)
**Tier 2:** LLM investigates why anomaly occurred

**Pros:**
- No manual rule writing
- Self-adjusting baselines

**Cons:**
- Sensitive to distribution shifts
- Can miss expected-but-bad patterns

**Example:** APM systems (detect latency spike → LLM analyzes recent deploys)

---

## Design Considerations

### 1. Alert Fatigue vs. Missed Signals

**Trade-off:** Tier 1 sensitivity

- High sensitivity → fewer missed issues, more false positives → alert fatigue
- Low sensitivity → fewer false positives, more missed issues → silent failures

**Solution:** Tune based on cost of missed signal vs. cost of investigation

### 2. Context Preservation

**Challenge:** Tier 2 needs context from when tier 1 triggered

**Solutions:**
- Include context in alert (recent events, state snapshot)
- Provide pointers for tier 2 to gather context (investigation IDs, timestamps)
- Hybrid: Include summary + pointers for deep dive

### 3. Feedback Loops

**Pattern drift:** What triggers tier 1 today may not matter tomorrow

**Solutions:**
- Log tier 2 verdicts (real issue vs. false positive)
- Periodic review of tier 1 rules (are they still relevant?)
- A/B test rule changes (measure impact on tier 2 load)

### 4. Graceful Degradation

**What if tier 2 is unavailable?** (LLM rate limit, downtime)

**Options:**
- Queue alerts for later investigation
- Fall back to human review
- Auto-escalate if tier 2 can't keep up

---

## Real-World Examples

### 1. Intrusion Detection (Snort, Suricata)

**Tier 1:** Packet inspection with signature matching
**Tier 2:** Security analyst investigates alert

**Pattern:** Network traffic (millions/sec) → Suspicious packets (10/min) → Real threats (1/hour)

### 2. Application Performance Monitoring (Datadog)

**Tier 1:** Metrics collection (latency, error rate, throughput)
**Tier 2:** On-call engineer investigates anomaly

**Pattern:** Metrics (1000/sec) → Threshold violations (5/min) → Incidents (1/day)

### 3. Medical Monitoring Systems

**Tier 1:** Vital signs sensors (heart rate, blood pressure)
**Tier 2:** Doctor investigates abnormal reading

**Pattern:** Sensor readings (continuous) → Alerts (few/day) → Medical intervention (rare)

### 4. Fraud Detection (Stripe, PayPal)

**Tier 1:** Rule-based alerts (transaction amount, velocity, location)
**Tier 2:** Risk analyst reviews transaction + user history

**Pattern:** Transactions (1M/day) → Flagged (1000/day) → Confirmed fraud (10/day)

### 5. Orchestrator Coaching Plugin (This Codebase)

**Tier 1:** Pattern matching in coaching.ts plugin
- Behavioral variation: 3+ bash commands in same semantic group
- Circular pattern: Keyword contradiction vs. prior investigations

**Tier 2:** Coach session (Claude)
- Receives metric event
- Reads investigations to validate
- Decides whether to intervene

**Pattern:** Bash commands (100/session) → Pattern triggers (5/session) → Coach intervention (1/session)

---

## Potential Applications (Orch Ecosystem)

### Agent Health Monitoring

**Tier 1:** Pattern detection
- Agent idle >30min with no phase progress
- Token usage >90% of context limit
- Same error message repeated 3+ times

**Tier 2:** Health investigator (LLM)
- Reads agent workspace, recent messages
- Diagnoses: Stuck? Waiting for input? Context exhausted?
- Recommends: Resume with guidance, abandon, or wait

### Beads Issue Quality

**Tier 1:** Validation rules
- Issue missing type or priority
- Description <50 characters
- No dependencies but marked as "blocked"

**Tier 2:** Issue validator (LLM)
- Reads issue context, related issues
- Validates if actually under-specified or clear from context
- Auto-fills metadata or requests clarification

### Session Focus Drift

**Tier 1:** Focus tracking
- 3+ spawns outside session focus area
- Time in non-focus work >30min

**Tier 2:** Focus coach (LLM)
- Reviews spawn history, session goal
- Validates: Legitimate pivot? Discovery? Distraction?
- Intervenes: Remind of focus or suggest updating goal

### Code Review Intelligence

**Tier 1:** Static analysis
- Complexity metrics (cyclomatic, cognitive)
- Duplication detection
- Security patterns (SQL injection, XSS)

**Tier 2:** Code reviewer (LLM)
- Validates if pattern is actual problem
- Suggests specific fix
- Explains trade-offs

---

## Anti-Patterns

### 1. Over-Engineering Tier 1

**Symptom:** Complex ML models in tier 1, high latency
**Problem:** Defeats purpose of cheap detection
**Fix:** Use simpler rules, move complexity to tier 2

### 2. Under-Resourced Tier 2

**Symptom:** Alert backlog growing, investigations rushed
**Problem:** Tier 1 generates more alerts than tier 2 can handle
**Fix:** Increase tier 2 capacity OR tune tier 1 to be more selective

### 3. No Feedback Loop

**Symptom:** False positive rate unknown, rules never updated
**Problem:** Pattern drift - tier 1 becomes irrelevant over time
**Fix:** Log tier 2 verdicts, periodic rule review

### 4. Skipping Tier 1

**Symptom:** "Let's just have LLM watch everything"
**Problem:** Token costs explode, latency increases
**Fix:** Add tier 1 filtering, even if simple (time-based sampling, keyword matching)

### 5. Tier 2 Blindness

**Symptom:** Tier 2 makes decisions without context from tier 1
**Problem:** Can't validate why alert triggered
**Fix:** Include detection context in alert (matched pattern, threshold values, recent history)

---

## Testing Strategy

### Tier 1 Testing (Detection)

**Metrics:**
- False negative rate (missed real issues)
- False positive rate (triggered on benign events)
- Latency (detection speed)

**Approach:**
- Unit tests for each pattern
- Historical replay (run tier 1 on past data, compare to known outcomes)
- Canary deployment (A/B test rule changes)

### Tier 2 Testing (Investigation)

**Metrics:**
- Investigation accuracy (% correct verdicts)
- Investigation latency (time to decision)
- Token usage (cost per investigation)

**Approach:**
- Manual review of sample investigations
- Feedback collection (was intervention helpful?)
- Cost tracking (tokens per alert type)

### End-to-End Testing

**Metrics:**
- System precision (% of interventions that were helpful)
- System recall (% of real issues caught)
- Alert fatigue (% of alerts dismissed)

**Approach:**
- Week-long pilot with real usage
- Log all alerts + tier 2 verdicts + user reactions
- Tune tier 1 thresholds based on outcomes

---

## Iteration Strategy

Start simple, measure, evolve:

**Phase 1: MVP (Week 1)**
- Implement simplest tier 1 rules
- Manual tier 2 (human investigates alerts)
- Measure: Alert volume, false positive rate

**Phase 2: Automation (Week 2)**
- Add LLM tier 2
- Log all verdicts
- Measure: Investigation accuracy, token cost

**Phase 3: Tuning (Week 3-4)**
- Adjust tier 1 thresholds based on tier 2 feedback
- Add/remove patterns based on signal quality
- Measure: End-to-end precision/recall

**Phase 4: Scale (Month 2+)**
- Optimize tier 2 prompts (reduce tokens)
- Add new patterns as failure modes discovered
- Measure: ROI (value of caught issues vs. cost of system)

---

## Success Criteria

A well-tuned two-tier system has:

- **High recall** (catches >90% of real issues)
- **Low tier 2 load** (tier 1 filters >90% of benign events)
- **Low alert fatigue** (tier 2 false positive rate <20%)
- **Sustainable economics** (tier 2 cost << value of caught issues)

---

## References

- **Origin:** Orchestrator coaching plugin design (orch-go-tjn1r)
- **Related patterns:**
  - Lambda architecture (batch + streaming)
  - SIEM (Security Information and Event Management)
  - Observability triad (logs, metrics, traces)
- **See also:**
  - `.kb/investigations/2026-01-10-inv-probe-pattern-retrospective-validate-detection.md` (pattern validation)
  - `.kb/investigations/2026-01-10-inv-probe-1b-session-session-streaming.md` (tier 2 streaming)
  - `.orch/workspace/og-feat-phase-behavioral-variation-10jan-1eed/SYNTHESIS.md` (tier 1 implementation)
  - `.orch/workspace/og-feat-phase-cross-document-10jan-8ebb/SYNTHESIS.md` (tier 1 circular detection)

---

## Provenance

This pattern was **discovered**, not invented. While designing the orchestrator coaching plugin, we realized the architecture (pattern matching → LLM investigation) matches a broader category of sensing systems.

The pattern exists in nature (immune system), security (IDS), operations (APM), and healthcare (monitoring). This guide documents it for future reuse in software systems.
