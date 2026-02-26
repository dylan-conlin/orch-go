# Session Synthesis

**Agent:** og-inv-orientation-frame-knowledge-26feb-c755
**Issue:** orch-go-yx2c
**Duration:** 2026-02-26
**Outcome:** success

---

## Plain-Language Summary

Audited all 37 decisions flagged by `kb reflect` as having 0 citations and ages of 11-67 days. Each decision was read, its core claim validated against the current codebase, and categorized into one of four dispositions. **19 are valid and implemented, 6 are already correctly archived, 5 are stale/superseded and need action, and 7 are process principles that remain sound.** The most significant finding is that the orchestrator-session-lifecycle model references deleted infrastructure (session registry, sessions.json) and an outdated three-level hierarchy — it needs a model update.

## Verification Contract

See `VERIFICATION_SPEC.yaml` in this workspace for verification details.

---

## TLDR

Audited 37 uncited decisions against the live codebase. 19 are valid, 6 are already archived, 5 need action (superseded or never implemented), 7 are sound process principles. The orchestrator-session-lifecycle model is the most impacted — it references deleted infrastructure.

---

## Delta (What Changed)

### Files Created
- `.kb/models/orchestrator-session-lifecycle/probes/2026-02-26-probe-decision-staleness-audit-37-decisions.md` - Probe documenting model contradictions found during audit
- `.orch/workspace/og-inv-orientation-frame-knowledge-26feb-c755/SYNTHESIS.md` - This file
- `.orch/workspace/og-inv-orientation-frame-knowledge-26feb-c755/VERIFICATION_SPEC.yaml` - Verification spec

### Files Modified
- None (audit-only, no code changes)

---

## Evidence (What Was Observed)

### Full Decision Audit (37 Decisions)

#### Category A: Valid and Implemented (19 decisions)

| # | Decision | Evidence |
|---|----------|----------|
| 1 | kb reflect command interface | `kb reflect --help` shows 9+ reflection types implemented |
| 3 | Replace confidence scores | No "confidence" in .orch/templates/ — fully removed |
| 4 | Template ownership model | Templates split: ~/.kb/templates/ (artifact) and .orch/templates/ (spawn) |
| 6 | Structured logging | Decision says hybrid (slog for daemon, fmt for CLI). slog not yet in daemon code but the principle (don't convert CLI output) is followed |
| 7 | Meta-orchestrator frame shift | Evolved to "strategic comprehender" — framing shift completed |
| 8 | Orchestrator session lifecycle | Three-level spawn hierarchy still exists in spawn patterns |
| 10 | Strategic orchestrator model | Orchestrator skill uses "strategic comprehender" framing |
| 11 | Synthesis is strategic orchestrator work | Daemon surfaces synthesis opportunities, doesn't auto-create. Correct delegation |
| 13 | Observation infrastructure principle | "If system can't observe it, system can't manage it" — principle active, several items implemented |
| 14 | Dashboard reliability architecture | `orch doctor` exists, health checks implemented, orch-dashboard script operational |
| 20 | Schema migration pattern | Delegated to OpenCode fork (correct boundary) |
| 22 | Verification bottleneck principle | Evolved to gate-based system with 13 named gates in complete_cmd.go |
| 27 | Registry contract spawn-cache only | Architecture lint test forbids registry packages. State derived from authoritative sources |
| 28 | Event-sourced monitoring | SSE implementation in pkg/opencode/sse.go, 27 files reference SSE |
| 29 | Three-tier workspace hierarchy | Active/archived/cleaned tiers visible in .orch/workspace/ |
| 32 | Investigation lineage enforcement | Prior-Work tables required in investigation skill |
| 33 | Orchestrator reflection session protocol | kb reflect fully implemented with 9+ modes |
| 36 | Knowledge maintenance automation loop | Implemented as distributed touchpoints (completion, spawn, daemon) via knowledge_maintenance.go |
| 37 | Daemon unified config construction | `daemonConfigFromFlags()` exists with 4 call sites in daemon.go |

#### Category B: Already Archived (6 decisions)

| # | Decision | Why Archived |
|---|----------|-------------|
| 15 | Abandon Claude Max OAuth, use Gemini | **Fully reversed** — Claude Sonnet is now default backend. Anthropic banned third-party OAuth, so Claude CLI became the primary path instead |
| 19 | Cancel second Claude Max subscription | **Not executed** — two accounts still exist in ~/.orch/accounts.yaml (personal + work). May have been intentionally kept for capacity-aware routing |
| 24 | Understanding lag pattern | Archived — observational insight about monitoring, not an actionable decision |
| 25 | Progressive session capture | Archived — evolved into session_handoff.go with HandoffSection validation |
| 26 | Models track architecture not implementation | Archived — subsumed into model template conventions |
| 30 | Five-tier completion escalation | Archived but **fully implemented** in pkg/verify/escalation.go with all 5 levels. The archival seems premature — code actively uses this |

#### Category C: Stale / Needs Action (5 decisions)

| # | Decision | Issue |
|---|----------|-------|
| 2 | Single-agent review (--preview flag) | **Never implemented.** No --preview flag in complete_cmd.go. Either implement or archive as superseded by the current 13-gate verification pipeline |
| 9 | Orchestrator lifecycle without beads | **Partially superseded.** Proposed ~/.orch/sessions.json which was built then **deleted and architecturally forbidden** via lint test. The "don't use beads for sessions" principle holds, but the proposed implementation (registry) was rejected |
| 12 | Load-bearing guidance data model | **Never implemented.** No `load_bearing` field found in any skill.yaml or skillc config. Decision exists but was not wired into skill system |
| 21 | Separate observation from intervention | **Not operationalized.** Coaching package (pkg/coaching/metrics.go) exists but has no observation/intervention separation. Decision remains aspirational |
| 35 | Probe as universal evidence primitive | **Partially implemented.** Probes exist and are the default in investigation skill, but 104+ investigations dated 2026-02 remain active. Investigations have NOT been retired as the decision proposed |

#### Category D: Process/Principle Decisions — Still Valid (7 decisions)

| # | Decision | Status |
|---|----------|--------|
| 5 | Orchestrator system resource visibility | Valid. Decision was "don't implement resource monitoring" — no monitoring code exists, correct |
| 16 | Dev vs prod architecture | Valid. Dev uses overmind (confirmed), prod planned for systemd (future VPS). orch-dashboard script uses overmind |
| 17 | Individual launchd services | Valid but **not deployed.** Infrastructure exists to generate launchd configs but no active .plist files in ~/Library/LaunchAgents/. Designed for future prod |
| 18 | Launchd supervision architecture | Valid but **not deployed.** Same as #17 — hybrid architecture designed but running overmind for now |
| 23 | Trust calibration assert knowledge | Valid. Fully reflected in global CLAUDE.md under "AI Deference Pattern" section |
| 31 | Questions as first-class entities | **Partially stale.** Question type exists conceptually and in pkg/question/ but NOT as first-class beads entities (no question type in .beads/config.yaml) |
| 34 | kb reflect cluster disposition | Valid but **point-in-time.** Specific guidance for agents/feature/quick clusters from Feb 2026 analysis. Still applicable to those clusters |

---

## Knowledge (What Was Learned)

### Key Findings

1. **The orchestrator-session-lifecycle model is the most stale.** It references deleted files (sessions.json, registry.go) and uses an outdated three-level hierarchy framing. Needs model update.

2. **Decision #15 is the most dramatically superseded.** It said "abandon Claude Max, use Gemini" but the exact opposite happened — Claude is now the default backend. Already archived but still in reflection audit.

3. **Decision #30 was archived prematurely.** The five-tier escalation is fully implemented in pkg/verify/escalation.go with all 5 levels and helper methods. The archival should be reviewed.

4. **5 decisions were never implemented** (#2, #12, #21) or only partially implemented (#9, #35). These represent design intent that never materialized — either archive them or create implementation issues.

5. **6 already-archived decisions** still appear in kb reflect because archiving doesn't remove them from citation tracking. This is a kb reflect limitation, not a decision problem.

### Constraints Discovered
- Architecture lint tests actively prevent re-introduction of session registry (forbidden file patterns)
- kb reflect doesn't exclude archived decisions from citation-count audits

### Externalized via `kb quick`
- See Leave it Better section below

---

## Next (What Should Happen)

**Recommendation:** close

### Recommended Follow-up Actions (for orchestrator)

1. **Update orchestrator-session-lifecycle model** — Remove session registry references, update hierarchy to strategic comprehender pattern. Priority: high (model actively misleads agents).

2. **Archive decisions #2, #12, #21** — Never implemented and no current plans to implement. Dead weight.

3. **Review decision #30 archival** — Five-tier escalation is fully implemented in code. Consider unarchiving or at minimum noting in archive that it's implemented.

4. **Review decision #19** — Two accounts still exist. Either the cancellation was intentionally abandoned (capacity routing) or this is still pending action.

5. **Consider kb reflect enhancement** — Exclude archived decisions from staleness audits, or give them a different weight.

### If Close
- [x] All deliverables complete (audit, probe, synthesis)
- [x] Probe file in .kb/models/ with all 4 sections
- [x] Ready for `orch complete orch-go-yx2c`

---

## Unexplored Questions

- **Are there decisions not in this audit that are also stale?** The directory has ~50 decision files total; only 37 were flagged. The remaining ~13 presumably have citations.
- **Should the knowledge maintenance automation loop handle decision staleness remediation?** Currently `kb reflect` detects but doesn't auto-remediate.
- **Is there a pattern to which decisions go stale?** The infrastructure/deployment decisions (#17, #18 - launchd) seem designed-but-not-deployed. Process decisions (#23, #5) tend to stay valid longer.

*(No stale artifacts encountered beyond what's documented in the audit itself.)*

---

## Session Metadata

**Skill:** investigation
**Model:** claude-opus-4-5
**Workspace:** `.orch/workspace/og-inv-orientation-frame-knowledge-26feb-c755/`
**Probe:** `.kb/models/orchestrator-session-lifecycle/probes/2026-02-26-probe-decision-staleness-audit-37-decisions.md`
**Beads:** `bd show orch-go-yx2c`
