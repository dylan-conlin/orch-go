# Brief: orch-go-c5ha1

## Frame

Today you read 33 briefs together and found 4 patterns none of them named. Eight briefs independently discovered the same identity gap. Five shared the same epistemic dishonesty. The system produced every atom of that understanding — and had no mechanism to compose them. The insight lived in the relationships between briefs, and nothing structural picked it up. You did it manually, in conversation, and it took the whole session.

## Resolution

The composition layer turns out to be a specific instance of something the system already models: Stage 3 (clustering) of the signal-to-design loop. Briefs are the signal. `.kb/briefs/` is the accumulation. What's missing is the clustering — and the design resolves to: orchestrator-session composition, triggered when 5+ unprocessed briefs pile up at session start, producing a "digest" that surfaces clusters with draft thread proposals.

The surprising finding was about what clusters well. Frame sections — the narrative of what happened — are too diverse to cluster reliably. But Tension sections — the unresolved questions each brief ends with — converge naturally when briefs share an underlying gap. Eight briefs asking different versions of "will changing the framing actually change behavior?" look unrelated by their stories but identical by their open questions. Composition should cluster on what's unresolved, not on what was done.

The hardest fork was the comprehension queue. The temptation to have composition drain the queue — "these briefs have been clustered, mark them processed" — was strong because it would unblock spawning. But that would be the 6th instance of the system treating "I processed this" as "I understood this." The queue stays honest. Composition is a navigation aid, not a comprehension act. The throttle pressure is doing its job: the system is producing faster than you can read, and pretending otherwise by relabeling clustered briefs as comprehended is exactly the epistemic dishonesty five of those briefs just identified.

## Tension

The design puts composition in the orchestrator session, which means it only fires when you're present. Between sessions, briefs accumulate without being composed, and the queue backs up, and spawning throttles. That's the design working as intended — but it means the system's productivity is gated on your session frequency. The question is whether that gate is the right size. If you have a week where sessions are sparse, the system essentially pauses itself. Is that back-pressure a feature (the system shouldn't run ahead of understanding), or is there a version of between-session composition that doesn't create a closed loop?
