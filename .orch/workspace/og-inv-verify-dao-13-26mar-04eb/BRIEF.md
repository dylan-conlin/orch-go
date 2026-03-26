# Brief: orch-go-ff21f

## Frame

We were carrying around a pretty strong sentence in DAO-13: `SPAWN_CONTEXT.md` files are `63-76 KB` and eat `40-50K` GPT tokens. That matters because the whole GPT-5.4 routing question can feel blocked by prompt size before we even test the model. If the number is stale, we're making today's decisions with yesterday's bottleneck.

## Resolution

The turn was simple: measure the files that actually exist now instead of reasoning from the old claim. Once I did that, the picture changed fast. Today's active March 26 spawn contexts are about `38-42 KB`, and the repo's own estimator (`chars / 4`) puts them at roughly `9.5-10.5K` tokens. Across all active workspaces, the median is only about `11.8K` tokens.

That does not erase the older GPT-5.2 stall story. It does narrow what that story can honestly mean. DAO-13 still works as a historical reliability warning, but its prompt-size sentence is carrying more certainty than the current system earns. The present bottleneck looks less like "the prompt is already huge" and more like "can the model follow the protocol once spawned?"

## Tension

There is still one unresolved split in the evidence: the repo estimates tokens with a generic `chars / 4` rule, while DAO-13's older wording implies a much denser GPT tokenizer count. So the next judgment call is whether it's enough to relabel DAO-13 as historical, or whether you want an actual GPT-5.4 tokenizer measurement before letting that claim shape model-routing decisions.
