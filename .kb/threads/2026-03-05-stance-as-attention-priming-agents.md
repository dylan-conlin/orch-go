---
title: "Stance as attention priming — what agents don't see"
status: open
created: 2026-03-05
updated: 2026-03-06
resolved_to: ""
---

# Stance as attention priming — what agents don't see

## 2026-03-05

Stance operates on input processing, not output constraint. Behavioral constraints tell agents what to produce/not produce (competes with priors, dilutes at 5+). Stance tells agents what to attend to (shifts what gets noticed before generation begins). Evidence: scenario 09 at N=6 — agents with full knowledge but no stance scored 17% on implicit contradictions. With stance, 83%. The information was identical. Only attention shifted.

Connection to defect classes: 4 of 7 named defect classes in orch-go (scope expansion, filter amnesia, contradictory authority, premature destruction) are attention failures. Agents had the information and didn't look. Current defenses are infrastructure gates (reactive — catch after mistake). Stance would be preventive — orient agents to notice relationships between components, absence as evidence, information freshness, and the question behind the question.

Proposed attention stances mapped to failure modes: (1) 'Every data path has implicit consumers — trace who reads the result' (Class 0), (2) 'Absence is evidence — check what existing consumers do that you don't' (Class 1), (3) 'Information decays — verify state is current before acting' (Class 7). These are not behavioral constraints ('NEVER do X') but attention primers ('LOOK FOR X').

Broader blind spots beyond defect classes: trajectory vs state (agents see what is, not what's changing), the question behind the question (intent behind content), what's not being said (absence in prompts/codebases). These are structurally hard for LLMs — trained on pattern matching over present tokens, so absence/trajectory/relationships require explicit orientation.

Testable: each attention type maps to a contrastive scenario. Same infrastructure as scenario 09. Unvalidated — thread becomes model when experiments confirm stances shift attention on these specific blind spots.

## 2026-03-06

Generalization confirmed: stance is cross-source reasoning primer, not generic amplifier. Lifts S12 (+5 relationship tracing) and S13 (+4 information freshness) but not S11 (-2 single-source absence). Detection-to-action gap discovered: agents see problems but still approve completion — stance improves input processing, behavioral constraints needed for output decisions. 72 total trials across 5 scenarios now.

Investigation stance experiment (54 trials): 'test before concluding' produces ZERO lift. The distinction is attention primer vs action directive. Attention primers change perception and transfer in --print mode (+4-7 on orchestrator). Action directives tell agents what to do and have no leverage without tool execution. Each worker skill stance must be classified: systematic-debugging and architect stances look like attention primers ('understand before fixing', 'decide what should exist'). Investigation needs reframing from action to attention.
