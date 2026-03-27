# Brief: orch-go-7iw5a

## Frame

The question looked simple at first: GPT-5.4 died once with zero tokens, then worked on the second run, so should orch just auto-retry that pattern? The danger is that "zero tokens" feels precise but is actually a weak story unless you know whether the agent really did nothing, partially worked, or just disappeared from the wrong part of the system.

## Resolution

The turn was realizing the bug is not lack of retries but lack of classification. Orch already retries transport failures, and it already knows how to respawn dead agents, but this failure sits between those layers. OpenCode says the session existed, status can say it went idle, beads never got Phase: Complete, and the current operator view collapses the token evidence into a dash. That means the system neither trusts the failure enough to recover it nor exposes it clearly enough for a human to judge.

So the recommendation is narrower than "retry zero tokens." Orch should classify an OpenCode empty execution: prompt accepted, session actually ran, no assistant output, no landed artifacts, zero tokens, then idle. If that fingerprint matches, retry once. If it happens again, stop and escalate. I turned that into a plan plus four concrete implementation issues so the work can land without losing the architectural guardrails.

## Tension

The unresolved judgment call is how much evidence is enough before orch can act without human review. One successful rerun justifies one bounded retry, but not broad trust yet; the next step should tell us whether this is a small transient class or the front edge of a noisier GPT-5.4 reasoning problem.
