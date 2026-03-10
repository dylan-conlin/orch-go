---
title: "Harness engineering — structural enforcement for multi-agent systems"
status: resolved
created: 2026-03-07
updated: 2026-03-10
resolved_to: "https://dylanconlin.com/blog/harness-engineering"
---

# Harness engineering — structural enforcement for multi-agent systems

## 2026-03-07

Accretion structural analysis revealed daemon.go grew +892 lines (667→1559) in 60 days from 30 individually-correct commits. 6 cross-cutting concerns independently reimplemented across 4-9 files (~2,100 lines of duplicated infrastructure). Extraction without routing is a pump — daemon.go regrew past its pre-extraction baseline. But spawn_cmd.go *shrank* -840 lines after pkg/spawn/backends/ was created — proving structural attractors break the re-accretion cycle.

Connected this to the OpenAI "Harness Engineering" paper (Codex team, ~1M lines, zero manual code) and Fowler/Bockeler's analysis. Same insight from opposite directions: OpenAI designed gates before code (greenfield); we discovered the need through pain (3 entropy spirals, 1,625 lost commits).

Unified five existing models (architectural-enforcement, entropy-spiral, skill-content-transfer, extract-patterns, completion-verification) into a single framework: hard harness (deterministic, mechanically enforced) vs soft harness (probabilistic, context-dependent, driftable). The three-type vocabulary from skill content transfer maps directly: knowledge→context, stance→attractors, behavioral→constraints/gates.

Core governance insight: in multi-agent systems, codebase architecture IS governance. Package structure is a routing table for agentic contributions. Every convention without a gate will eventually be violated.

## 2026-03-08

Reconstructed after ~/deletion incident. Model text survived as untracked file. Missing artifacts: the accretion investigation (.kb/investigations/2026-03-07-inv-analyze-accretion-pattern-orch-go.md) and any Layer 1-3 implementation work from March 7-8 (structural test expansion, entropy agent). Model itself serves as the reconstruction roadmap — implementation layers 0-4 are clearly defined with status.
