# Brief: orch-go-y85zx

## Frame

The compositional accretion model said the right thing: outward-pointing atoms compose, inward-pointing atoms pile up. But saying it and proving it across every artifact type the system actually produces are different acts. The model was built from 7 examples in one conversation. This audit tested it against all 13 artifact types with real adoption measurements — and found the model was right about direction but wrong about the bottleneck.

## Resolution

I expected the audit to find some surfaces composing and others piling up, and it did: briefs compose (100% tension coverage), investigations pile up (81.9% orphan rate), beads issues pile up (82% unenriched). The model predicted all of that. But the interesting finding was why investigations pile up despite *having* a compositional signal. They have defect-class. They have model links. Both are outward-pointing. Both exist in the template. And both are filled less than 20% of the time.

The problem isn't signal design — it's signal adoption. The brief Tension section works because it's opt-out: the BRIEF.md template requires it, and agents can't complete without it. The investigation defect-class field is opt-in: agents can submit without it, and 80% do. The same pattern holds everywhere: opt-out signals (issue_type, Tension, Model Impact) hit 84-100% adoption. Opt-in signals (enrichment labels, claim/verdict, decision Extends) plateau at 15-25%.

The model now has a 4th design criterion question: "What is the adoption rate, and is the signal opt-in or opt-out?" And a new claim (CA-06): only opt-out signals achieve the adoption necessary for measurable composition.

## Tension

The obvious move is to make every signal opt-out — require the model link on investigations, require enrichment labels on issues, require resolved_to on threads. But the brief Tension section works because articulating what you don't know is natural to the writing act. Requiring agents to fill a model link field before they know what model their investigation relates to might produce garbage metadata to satisfy the gate — exactly the failure mode the model's own constraints warn about. The gap between "opt-out" and "quality opt-out" is the next question the model doesn't answer.
