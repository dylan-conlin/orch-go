# Brief: orch-go-s1r5m

## Frame

You keep reaching for tmux to check if agents are alive, despite a dashboard that's supposed to make tmux unnecessary. The dashboard says "3 agents active" and you believe it — sort of — but believing isn't the same as seeing. Tmux shows text scrolling by. That scroll IS the proof. The dashboard gives you a number that asks for trust.

## Resolution

I traced the data flow end to end and found something I didn't expect: the dashboard *already has* the pulse data. The SSE connection on the home page receives every tool call, every file read, every reasoning block from every OpenCode agent. The agent store updates in real-time with activity type and text. The agent card component on the work-graph page even has functions — `getExpressiveStatus()`, `formatActivityAge()` — that convert raw events into readable strings like "Running Bash... 3s ago" and "Hatching... (thought for 12s)."

But the home page throws all of it away. It renders `$trulyActiveAgents.length` — a bare count. Three agents active. Period. No activity, no timestamp, no proof.

The reason is architectural: when the comprehension-first redesign subtracted execution content from the home page, it treated all agent activity data as "execution." But there's a distinction the model missed. Execution monitoring is "this agent is reading serve_agents.go line 352" — it demands attention, it's what the work-graph is for. Liveness assurance is "something is happening, here's proof" — it's a typing indicator, peripheral vision, a ticking timestamp. The subtraction removed both because the model only had two categories. There should be three: comprehension (prose you read), execution (detail you scan), and presence (proof you glance at).

## Tension

The fix looks simple — render activity text in the condensed summary line, data's already there. But the two-tier problem complicates it: OpenCode agents have real-time SSE pulse, Claude-backend agents have only periodic phase comments. As more agents move to OpenCode (GPT-5.4 already has no tmux), the dashboard becomes the *only* monitoring surface for a growing population — and "3 agents active" isn't going to cut it. The question isn't whether to add pulse rendering. It's whether adding it to the home page actually kills the tmux habit, or whether the real problem is that you don't trust the dashboard's data enough yet for pulse to matter.
