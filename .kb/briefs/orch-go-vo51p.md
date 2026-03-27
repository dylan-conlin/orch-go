# Brief: orch-go-vo51p

## Frame

The dashboard was born as an execution monitoring tool and later grew comprehension surfaces on top. Now that the product decision says comprehension is the center, the question is: which surfaces actually belong, and which are kept around only because they already exist?

## Resolution

The surprising finding was that the comprehension layer is already in good shape. Threads are above the fold. Nav reads Threads | Briefs | Knowledge | Work | Harness. The review queue and open questions are prominent. The thread-first home surface design that was proposed a few hours ago? Already partially implemented.

The actual problem is below the fold. Scroll past the threads and briefs, and you hit a full execution monitoring suite — active agent grids, coaching health indicators, services panels, swarm maps, event streams. Two entire dashboard modes (operational and historical) exist solely to organize this execution content. It's roughly 3-4x the comprehension layer by code weight, and it sends the message "this is fundamentally an agent monitoring tool that also has some thinking features."

The classification came out cleanly: 7 core comprehension surfaces (keep as-is), 5 bridge surfaces (condense to summaries), 11 execution residue surfaces (move to the Work route or delete). One route — /thinking — is completely dead code with no backend implementation, a prototype whose concept was absorbed by threads and briefs months ago.

The recommended next move is subtraction, not addition. Move the execution sections from the home page to the Work route. Delete the dead /thinking route. Replace the below-fold operational sprawl with a single summary line: "3 agents active, 5 ready, 2 need review — View Work." The comprehension layer doesn't need to be built. It needs room to breathe.

## Tension

The classification assumes the below-fold execution content on the home page isn't part of Dylan's daily workflow. If it turns out he actually starts mornings by scrolling past threads to check the active agents grid or coaching health, several "residue" items should be reclassified as bridge. The demotion plan should be validated by a few days of conscious observation before committing to the subtraction.
