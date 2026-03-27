# Brief: orch-go-2mlhl

## Frame

The question looked simple at first: if Opus is rate-limited, what model comes next? But the real problem was that "fallback" can mean two different things. Sometimes Opus itself is exhausted while Claude is still fine; other times the whole Anthropic path is unhealthy. If we collapse those into one branch, the system will either downgrade too aggressively or pretend the models are more interchangeable than they are.

## Resolution

The turn in the work was realizing that orch-go already has most of the policy shape, just not the missing signal. Reasoning-heavy skills are still explicitly pinned to Opus, while implementation work already falls through to Sonnet. Account routing already knows how to stay inside Anthropic by picking the healthiest account. The gap is that the capacity layer receives Anthropic's `seven_day_opus` field and then throws it away, so the system cannot tell "use Sonnet" from "leave Anthropic entirely."

That leads to a cleaner cascade than the old Gemini-era wording. First try to preserve Opus by switching to the healthiest Anthropic account. If Opus is what's exhausted but Claude still has room, drop to Sonnet. Only if the Anthropic path itself is unavailable should the system cross providers, and even then GPT-5.4 should be automatic only for `feature-impl`. The evidence for reasoning-heavy GPT fallback is not good enough yet, so the honest thing is to stop there instead of silently degrading.

## Tension

The open judgment call is whether the alternate-account branch still deserves first place now that the system is effectively provisioned around one Anthropic subscription path plus one OpenAI path. The other live tension is strategic: how quickly do we want to benchmark GPT-5.4 on architect/investigation/debugging so this policy can evolve from a guarded stop into a broader overflow lane?
