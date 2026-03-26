# Brief: orch-go-bptuy

## Frame

The first run of the mechanism experiment came back flat — 97% compliance at every constraint count from 1 to 20. This was supposed to show degradation. It showed nothing. The surprise wasn't that agents are good at following instructions; it's that we'd been measuring the wrong thing. "Add named returns" and "add a const block" are both requests an agent can satisfy by appending code — they don't compete for anything. It's like testing whether a chef can follow 20 recipes simultaneously by asking them to plate 20 garnishes. Of course they can. The question should have been: what happens when the recipes contradict each other?

## Resolution

I rebuilt the constraint pool around the idea of tension — pairs of constraints where following A makes B harder or impossible. "The function MUST return (string, error)" opposite "The function MUST return string only." "Use a switch for unit selection" opposite "Use a for loop, no switch." Twelve pairs, split into three tiers: HARD (logically contradictory — one side must lose), MEDIUM (both satisfiable but competing for implementation space), and EASY (standard Go patterns resolve both — the calibration control).

The three tiers are the real design. They turn the experiment into a mechanism discriminator. If EASY pairs degrade alongside HARD pairs as you add more, the mechanism is resource competition — the agent has a finite budget and everything suffers. If EASY pairs hold while HARD pairs degrade, the mechanism is interference — specific tensions cause specific failures. If everything flips at a critical N, that's threshold collapse. Phase 1 couldn't distinguish these because all its constraints lived in the "EASY" tier by accident. Scoring now measures which side the agent sacrifices per pair, not just how many constraints it followed.

## Tension

The obvious concern: agents might just refuse the contradictions. "These two constraints conflict, so I'll explain why and pick the most reasonable one." That's fine — that IS the data. But it means the "HARD" tier might show deterministic sacrifice (always pick side B) rather than degradation. If every HARD pair resolves identically regardless of N, we've learned something about agent decision-making under contradiction, but not about constraint competition. The real signal might live entirely in the MEDIUM tier, where tension is real but not impossible. Whether 4 MEDIUM pairs provide enough statistical power is the open question.
