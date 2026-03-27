# Brief: orch-go-5b9st

## Frame

The daemon has become a 312-issue hotspot. Four independent bugs were fixed in the last few sessions — SIGKILL crashes, stuck comprehension counters, duplicate spawns, frozen dashboard events — and they all had the same shape: subsystems that grew independently with no shared infrastructure. The question was whether the next fix should be another patch, or whether something structural needed to change.

## Resolution

I expected to find a sprawling mess that needed a rewrite. Instead, I found that the hard structural work was already done — the dedup pipeline was extracted into composable gates, the cycle cache was sharing expensive queries across periodic tasks, and PID validation was already handling stale status files. The daemon grew from 5 operations to 30+ subsystems in three months, and most of them are wired correctly.

The surprise was the double logging root cause. Every daemon log line appears twice because two independent write paths target the same file: DaemonLogger opens `~/.orch/daemon.log` directly AND launchd redirects stdout to the same path. That's a textbook Defect Class 5 — contradictory authority signals — and it's a 30-line fix. The SIGKILL crashes have a similar structural simplicity: there's no budget for shutdown work, just a 3-second timeout on reflection with no enforcement of the total. Any future defer that does unbounded work silently erodes the margin.

The dedup question ("should dedup move to issue-creation time?") turned out to be a layer confusion. Spawn-time dedup prevents executing duplicate work — that's infrastructure, and the 7-gate pipeline handles it. The upstream problem is architects creating follow-up issues for work already committed. That's a skill concern, not a daemon concern. Both layers are needed but at different abstraction levels.

## Tension

The CAS-based dedup redesign from the March 1st investigation remains the right long-term answer — atomic status transitions would eliminate the correlated fail-open risk where beads unavailability degrades 3 gates simultaneously. But it requires beads fork changes, and the current pipeline is a stable intermediate state. The question is whether "stable intermediate" becomes permanent through inertia, or whether the CAS work gets prioritized before the next compound failure scenario hits overnight.
