# Probe: Automated Adoption Rate Measurement — Can We Detect Signal Drift Before It Goes Silent?

**Model:** compositional-accretion
**Date:** 2026-03-27
**Status:** Complete
**claim:** CA-02
**verdict:** extends

---

## Question

CA-02 says "adding a composition signal reduces pile-up." The prior probe (artifact-type-audit) extended this with "signal adoption rate mediates the effect — only >80% adoption enables measurable composition." If adoption rate is the binding constraint, can we build automated measurement that detects drift before a signal goes effectively dead? The hotspot harness was disabled for 5 weeks without detection — same risk applies to adoption rates.

---

## What I Tested

Built `orch harness adoption` command that replicates the probe's grep/wc measurements in Go:
- Investigation model link rate (grep `**Model:**` in .kb/investigations/)
- Brief tension rate (grep `## Tension` in .kb/briefs/)
- Probe claim/verdict rate (grep `claim:` and `verdict:` in .kb/models/*/probes/)
- Thread resolved_to rate (grep `resolved_to:` in .kb/threads/)
- Beads enrichment label rate (parse .beads/issues.jsonl for routing labels)
- Decision Extends rate (grep `Extends:` in .kb/decisions/)

Wired into `orch orient` output so drift surfaces every session start.

```bash
# Build and run
make build && orch harness adoption

# Verify orient integration
orch orient 2>&1 | grep -A 10 "Adoption"
```

---

## What I Observed

### `orch harness adoption` output (live corpus, 2026-03-27):

```
═══ ADOPTION RATES ═══

  SIGNAL                        TOTAL  ADOPT    RATE  TARGET  STATUS
  ────────────────────────────────────────────────────────────────────────
  Investigation model link        369     59     16%     80%  CRITICAL
  Brief tension                    83     83    100%    100%  ok
  Probe claim                     292     35     12%     80%  CRITICAL
  Probe verdict                   292     35     12%     80%  CRITICAL
  Thread resolved_to               60     27     45%     80%  DRIFT
  Beads enrichment               2165    409     19%     80%  CRITICAL
  Decision Extends                 44      7     16%     50%  CRITICAL
```

### `orch orient` adoption drift section:

```
Adoption drift:
   [!!!] Investigation model link: 16% (target 80%)
   [!!!] Probe claim: 12% (target 80%)
   [!!!] Probe verdict: 12% (target 80%)
   [!] Thread resolved_to: 45% (target 80%)
   [!!!] Beads enrichment: 19% (target 80%)
   [!!!] Decision Extends: 16% (target 50%)
   Run: orch harness adoption
```

### Key observations:

1. **Only 1 of 7 signals meets its target** — Brief tension at 100% (the only opt-out/required signal). All opt-in signals are below target.

2. **The frontmatter format matters.** Original probe grep `grep -rl "claim:" .kb/models/*/probes/*.md` found 57 matches (including body text). The frontmatter-restricted measurement found 35. The difference (22) are false positives where "claim:" appears in model analysis text, not as a structured field. Automated measurement needs tighter matching than ad-hoc grep.

3. **Three severity tiers emerged naturally:** ok (>=target), drift (>=target/2 and <target), critical (<target/2). This maps to the model's insight: below ~40% adoption, the signal is "effectively dead" — structurally identical to no signal.

4. **Wiring into orient guarantees visibility.** Orient runs at every session start. The adoption drift section is sandwiched between health summary and daemon health — all three are structural health signals. If adoption degrades further, it will be visible within one session.

### Tests: 20 tests passing (17 adoption + 3 orient integration)

---

## Model Impact

- [x] **Extends** CA-02: The model says "adding a composition signal reduces pile-up." The automation confirms this is measurable and monitorable. But it also shows the current state is worse than the prior manual audit suggested — only 1 of 7 signals meets its target. The model should note that **automated measurement is a prerequisite for the adoption-rate claim** because manual audits happen too infrequently to detect drift (the hotspot harness was disabled for 5 weeks without detection).

- [x] **Extends** the model with a new invariant: **Measurement precision matters.** Ad-hoc grep counts and structured frontmatter counts diverge by ~40% for probe claim/verdict rates (57 vs 35). The model's claims about adoption rates should specify which measurement methodology was used, because the numbers change depending on whether you count body text mentions or frontmatter fields.

---

## Notes

The meta-measurement question — "does this check actually run?" — is answered by the orient integration. Orient runs at every session start, so adoption drift is visible in every session. This is the same pattern that makes daemon health visible (orient § Daemon health).

The current adoption rates are alarming (5 of 7 critical), but this is the baseline — the measurement didn't create the problem, it made it visible. The prior probe (artifact-type-audit) recommended making signals opt-out rather than opt-in. This measurement will track whether those interventions improve adoption rates over time.
