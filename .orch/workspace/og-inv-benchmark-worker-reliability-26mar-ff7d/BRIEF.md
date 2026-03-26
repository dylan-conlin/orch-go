# Brief: orch-go-1dhv8

## Frame

We asked which worker backends are safe to use in production and set out to benchmark Claude Code against GPT-5.4 and a cheaper fallback. The question felt empirical — run tasks on different models, measure completion rates, build a routing table. It mattered because the system is 100% locked to Anthropic and we wanted data to justify (or reject) diversifying.

## Resolution

The turn came early: there's nothing to benchmark against. Every single post-protocol agent — all 130 of them — is Opus on Claude Code. No Sonnet. No GPT. No Gemini. The system doesn't have a comparison population; it has a monoculture.

The Claude Code baseline is genuinely good. In the last week: 97% Phase:Complete on investigations, 93% on architect work, 100% on feature-impl and debugging. These aren't cherry-picked — they're the full population. The one reliability gap is SYNTHESIS.md creation on feature-impl (4%), but that's a protocol weight problem, not a stall. Agents finish the work and skip the paperwork.

GPT-5.4's plumbing is fully wired — model aliases, routing logic, API key. A dry-run routes perfectly. But it's never run a real task. The last attempt (March 24) died on missing OpenAI auth. OpenCode server being down blocked any testing today. The gap between "infrastructure ready" and "empirically validated" is about 15 minutes of Dylan's time: start the server, spawn 5 tasks, check results.

## Tension

The recommendation matrix has clear thresholds (80% Phase:Complete for overflow viability, 90% for default routing), but the data to apply them doesn't exist yet. We're making a strategic bet on whether to invest 15 minutes in a test that could unlock multi-model routing — or accept the Anthropic monoculture because it works. The uncomfortable part: the monoculture works *well enough* that the urgency to test alternatives keeps getting deferred. Each week it doesn't get tested is another week of confirmed lock-in by inaction.
