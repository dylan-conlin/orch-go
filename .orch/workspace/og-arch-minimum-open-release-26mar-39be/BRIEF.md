# Brief: orch-go-ehper

## Frame

You accepted two decisions yesterday — the product is the comprehension layer, and it should be open at the boundary, opinionated at the core — and a third agent inventoried 28 product surfaces into a three-wave release strategy. Wave 1 was "publish artifact formats now, they're self-documenting." But nobody had checked whether a new user can actually understand the artifacts without the system that produces them. That's the question I went to answer.

## Resolution

I read every artifact type as if I'd never seen the system before. The briefs are good — Frame/Resolution/Tension is clear, a 3.5 out of 5. Decisions are standard ADRs, a 4. Everything else is opaque. Threads have seven unexplained orch IDs in their frontmatter. Models reference claim IDs (KA-05) that are meaningless without the CLI. Probes use verdict values that are never documented. The KB README — the official "here's what this is" document — covers four of seven artifact types and leaves out threads and briefs, the two most important ones. Average across all seven types: 2.6 out of 5.

The surprise wasn't any individual artifact being confusing. It was that the composition model — how the artifacts relate to each other — isn't documented anywhere. Thread spawns investigation, investigation produces probe, probe tests model, model gets updated, brief synthesizes for the human. That cycle IS the method. But it lives entirely in the CLI tooling and the skill definitions. A new user reading the raw artifacts sees pieces of a puzzle with no picture on the box.

The minimum release isn't "artifact formats" (Wave 1 from the matrix). It's artifact formats + a composition guide that diagrams the cycle + thread commands as the first thing a user does + curated examples cleaned of orch-specific noise. And the init flow needs to stop telling people to "spawn an agent" as step 3 — that's the substrate talking, not the product.

## Tension

The composition guide is the one artifact I'm most certain needs to exist and least certain how to write. If it's a 2-page diagram with one paragraph per artifact type, it teaches. If it's a 10-page specification, it documents. The method is simple enough to diagram — the risk is that explaining it in writing makes it sound more complex than it is. There's also a deeper question: do threads feel valuable on day 1 before any agent has produced evidence? If a user creates a thread and nothing happens, they're just taking notes in a structured directory. The method only clicks when the cycle completes — question, evidence, synthesis, new question. But you can't run the full cycle without agents, and agents are the substrate you're trying to background.
