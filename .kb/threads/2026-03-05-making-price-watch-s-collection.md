---
title: "Making Price Watch's collection system autonomous and trustworthy"
status: open
created: 2026-03-05
updated: 2026-03-05
resolved_to: ""
---

# Making Price Watch's collection system autonomous and trustworthy

## 2026-03-05

From manual tool to self-running system Jim trusts. Arc: scheduler activation → shadow validation → model confidence UI → calculated mode cutover → sparse collection. Gate: Jim sees models proving themselves per-material over weeks before any data source changes.

Session 2026-03-05 shipped: scheduler activation (dual-gate), SCS formula shadow comparison, model_validation_runs table + confidence API, divergence cause annotations, confidence badges in comparison view, detail panel + divergence alerts. First scheduled run: Monday March 9, 2am PST.

Key insight: the session started with scheduler activation but pivoted to a deeper question — how does Jim trust pricing models enough to let them replace live data collection? That produced the full confidence UI pipeline: shadow comparison (invisible validation) → per-material confidence badges Jim can see and drill into.

Related issues: pw-ity2, pw-0ea7, pw-2t96, pw-0h7e, scs-sp-x1o, toolshed-n5r4, toolshed-ozlp. Next: pw-uv4b (sparse collection).
