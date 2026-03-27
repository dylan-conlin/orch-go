# Brief: orch-go-5e02e

## Frame

The daemon has two gates that both ask the same question: "is Dylan keeping up with what agents produce?" One counts completions in RAM. The other counts beads labels. They have different thresholds, different scopes, and different state lifetimes. After completing 8 agents in an orchestrator session, the daemon was still paused because the RAM counter didn't know about the completions. Yesterday's fix (ResyncWithBacklog) was a patch. Today's question: should these be one gate?

## Resolution

They should, but the interesting finding isn't that they overlap — it's that the comprehension gate was accidentally broken for the most common work type. When an agent finishes and gets labeled ready-for-review, the daemon immediately fires a headless `orch complete` to pre-generate the brief. That headless completion, as a side effect, removes the `comprehension:unread` label. By the next poll cycle, the comprehension gate can't see the item. So for all non-trivial work, only the verification tracker (the one with stale in-memory state) was actually throttling.

The fix is two things that happen to look like one thing. First: stop headless completions from clearing the `comprehension:unread` label — only interactive `orch complete` should clear it. This makes the label mean "no human has reviewed this," which is what the verification tracker was trying to enforce with fragile signal files and in-memory counters. Second: remove the verification tracker entirely. One gate, backed by durable beads labels, with a compliance-derived threshold. The 22-file verification tracker surface disappears. The ResyncWithBacklog patch becomes unnecessary because beads labels don't go stale.

## Tension

Auto-completed agents will now count toward the gate. Previously they were invisible to verification (by design — they're already closed) and invisible to comprehension (by accident — the race clears the label). After this change, every agent the daemon completes adds to the review backlog count. The orchestrator will see higher numbers and need to actively drain them (via `orch complete` or `orch daemon resume`). This is arguably correct — "you're not keeping up" should include all output — but it changes the operational feel of the daemon from "handles simple stuff silently" to "everything demands acknowledgment." Is that the right pressure?
