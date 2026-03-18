---
title: "Orchestrator skill reframe — policy to coordination model"
status: resolved
created: 2026-03-11
updated: 2026-03-17
resolved_to: "Incrementally resolved through skill pruning and hard harness buildout. Skill went from 1,251→512 lines (59% reduction). Content shifted from behavioral policy to coordination model: 28 model/reasoning terms vs 7 directive terms. Hard harness (gates, hooks, spawn infra) now handles enforcement; skill focuses on judgment routing. Metadata still says skill-type: policy — cosmetic, could be updated but low priority."
---

# Orchestrator skill reframe — policy to coordination model

## 2026-03-11

The orchestrator skill (1,251 lines) is the system's primary soft harness for agent coordination — but the harness engineering model's own findings say soft harness degrades past 10 constraints, and this skill has far more. It's a behavioral policy (do X in situation Y) that tries to enforce coordination through prose instruction. The reframe: the skill should become a coordination model spec — why agents degrade (knowledge accretion observations), what hard harness enforces (gates, hooks, spawn infrastructure), and what genuine judgment calls remain. Shorter, more mental model than rulebook. Blocked on: measurement audit (orch-go-ca0k0) should include 'is the skill actually influencing behavior?' before redesigning. But the direction — harden what matters, shed the behavioral policy weight — is right regardless.

## 2026-03-17 — Resolution

**Blocker cleared:** orch-go-ca0k0 (measurement audit) closed 2026-03-11.

**The reframe happened incrementally**, not as a single redesign:

- **Size:** 1,251 → 512 lines (59% reduction). Well below the ~10-constraint soft harness degradation threshold.
- **Language shift:** Content now reads as coordination model, not behavioral policy. 28 model/reasoning terms (why, because, pattern, mental model, judgment) vs 7 directive terms (must, never, always). The skill explains *why agents degrade* and *what judgment calls remain*, rather than listing situational rules.
- **Hard harness took over enforcement:** Gates (`pkg/spawn/gates/`), hooks (`.claude/settings.json`), and spawn infrastructure now enforce what the skill previously tried to enforce through prose. The skill no longer competes with infrastructure.
- **5 skill commits since thread opened** (docs sync, completion gate removal, KB guidance consolidation, config drift remediation, investigation standards) — each one pruned policy weight.

**Remaining cosmetic:** `skill-type: policy` in metadata could become `skill-type: coordination-model` or similar, but the content has already shifted. Low priority.
