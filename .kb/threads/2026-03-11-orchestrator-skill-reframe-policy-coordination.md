---
title: "Orchestrator skill reframe — policy to coordination model"
status: open
created: 2026-03-11
updated: 2026-03-11
resolved_to: ""
---

# Orchestrator skill reframe — policy to coordination model

## 2026-03-11

The orchestrator skill (1,251 lines) is the system's primary soft harness for agent coordination — but the harness engineering model's own findings say soft harness degrades past 10 constraints, and this skill has far more. It's a behavioral policy (do X in situation Y) that tries to enforce coordination through prose instruction. The reframe: the skill should become a coordination model spec — why agents degrade (knowledge accretion observations), what hard harness enforces (gates, hooks, spawn infrastructure), and what genuine judgment calls remain. Shorter, more mental model than rulebook. Blocked on: measurement audit (orch-go-ca0k0) should include 'is the skill actually influencing behavior?' before redesigning. But the direction — harden what matters, shed the behavioral policy weight — is right regardless.
