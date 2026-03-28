---
title: "Beads atom problem — work molecules need structural representation"
status: open
created: 2026-03-22
updated: 2026-03-22
resolved_to: ""
---

# Beads atom problem — work molecules need structural representation

## 2026-03-22

Consumer-last construction (10/10 open loops in orch-go) is a beads decomposition failure, not a code architecture failure. Architects design feedback loops (molecules) but beads decomposes them into independently completable atoms. Each atom passes all gates (build, test, review) but the molecule never closes. Threads and plans were the attempted fix — they capture intent that spans issues — but they have no structural coupling to the beads graph. A thread can say 'these are related' but nothing stops the daemon from completing the emitter atom and moving on forever. Dependencies (bd dep) model 'B waits for A' but the actual need is 'A is incomplete without B' — the emitter isn't done until the consumer exists and is wired. Two candidate structural fixes: (1) molecule primitive in beads — a unit of work that isn't done until all atoms compose correctly, (2) composition gate in completion pipeline — before closing architect implementation issues, verify the designed interaction works end-to-end. This connects to the coordination primitives framework: it's an Align failure at the work-decomposition level. Each agent's model of 'done' is locally correct but systemically wrong. The sensor gap is compositional — no sensor checks whether atoms compose into working molecules.
