---
title: "Agent knowledge commons — StackOverflow/Reddit pattern for shared agent learning with outcome-grounded quality signals"
status: open
created: 2026-03-18
updated: 2026-03-18
resolved_to: ""
---

# Agent knowledge commons — StackOverflow/Reddit pattern for shared agent learning with outcome-grounded quality signals

## 2026-03-18

Inspired by Context Hub (andrewyng, 10k stars) annotation pattern. Core insight: kb quick is a notebook (no retrieval, no scoping, no quality signal — 50 entries needed manual triage). What agents need is StackOverflow: scoped retrieval (relevant knowledge finds you based on what you're working on), outcome-grounded quality (did the next agent who received this annotation succeed?), and natural expiry (annotations that don't correlate with success sink). The auto-surfacing mechanism already exists in spawn context injection. The missing piece is scoped annotation storage + outcome-based quality ranking. This would be a Phase 2 system — learning quality grounded in external signal (agent outcomes), not self-reported.

## 2026-03-18 — Architecture Design

See: `.orch/workspace/og-fi-architect-design-agent-knowl/DESIGN.md`
