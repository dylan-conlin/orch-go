# Probe: Communication Breakdown Post-Mortem — 3 Orchestrator Sessions

**Model:** orchestrator-session-lifecycle
**Date:** 2026-02-27
**Status:** Complete

---

## Question

The orchestrator-session-lifecycle model documents how orchestrators manage spawn context, skill injection, and behavioral compliance. But do the model's assumptions about orchestrator competence hold when orchestrators face **cross-project debugging under time pressure**? Specifically: do the Orchestrator Frame Guard, cross-repo knowledge boundaries, and CLI tooling prove adequate or become the failure modes themselves?

---

## What I Tested

Read 3 full session transcripts from 2026-02-27 where Dylan experienced total communication breakdown:

1. **Toolshed orchestrator** (feature-flag + shipping): `toolshed/2026-02-27-142555-ok-so-regarding-adminfeatures-feature-flag-vie.txt`
2. **Toolshed orchestrator** (shipping deep dive): `toolshed/2026-02-27-142543-ok-so-im-still-not-seeing-shipping-prices-for-os.txt`
3. **Price Watch orchestrator** (revenue-at-risk + triage + test fix): `price-watch/2026-02-27-142603-lets-take-a-look-at-e352.txt`

Additionally reviewed the kb agreements system implementation (5 built-in agreements + 5 custom cross-project agreements in kb-cli, non-blocking warning-only spawn gate in orch-go).

---

## What I Observed

See full investigation: `.kb/investigations/2026-02-27-postmortem-communication-breakdown-sessions.md`

Summary: Identified **7 distinct failure categories** across 20+ individual failures. The orchestrator model's behavioral compliance probe (2026-02-24) identified an "Identity vs Action compliance gap" — this postmortem confirms that gap is the dominant failure mode under real pressure, not an edge case.

---

## Model Impact

- [x] **Extends** model with: 7-category communication failure taxonomy observed across real orchestrator sessions under cross-project debugging pressure. The model documents spawn context and skill injection but lacks coverage of **in-session behavioral degradation** — orchestrators that comply with identity ("I'm an orchestrator") but fail at action (using Task instead of orch spawn, forgetting to create promised issues, contradicting own analysis). The frame guard, designed to prevent frame collapse, creates a new failure mode: **debugging paralysis** where the orchestrator can't trace data paths it needs to resolve the user's problem.

- [x] **Confirms** invariant: "Stale references are harmful" (from CLI staleness audit probe). Stale knowledge about OshCut collection method directly caused wrong diagnosis in 2 sessions.

- [x] **Contradicts** implicit assumption: The model assumes orchestrators have adequate tooling for their coordination role. In practice, CLI unfamiliarity (wrong orch flags, bd create arg requirements, orch complete gate cascade) consumed significant user patience across all 3 sessions.

---

## Notes

The 2024-02-24 behavioral compliance probe rated orchestrator action compliance at ~60%. This postmortem suggests that under cross-project debugging pressure, action compliance drops further — perhaps 40-50%. The frame guard is structurally sound for preventing frame collapse during routine orchestration, but becomes a liability during active debugging where the user needs the orchestrator to trace a data path across code boundaries.
