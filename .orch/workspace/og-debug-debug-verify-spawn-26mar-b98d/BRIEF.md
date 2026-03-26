# Brief: orch-go-x7pde

## Frame

The question looked small: if someone passes a new model name that is not in our alias table, does spawn still choose the right backend? That matters because model catalogs move faster than hardcoded aliases, and a resolver that only works for known nicknames quietly turns every new model into a routing bug.

## Resolution

The turn here was that the risky-looking part was not actually the routing logic. `pkg/spawn/resolve.go` already asks the model resolver for a normalized `provider/model` result and then picks a backend from the provider, which means alias membership is not the deciding factor once the model is explicit. The real gap was confidence: we had good behavior, but not a test proving that a future OpenAI or Anthropic model ID still takes the correct path.

So I left the implementation alone and added regression coverage instead. The new tests exercise unknown-but-explicit model strings like `openai/o4-mini-2026-01-15` and `anthropic/claude-sonnet-5-20260101`, and they show the behavior we want: OpenAI routes to `opencode`, Anthropic routes to `claude`, and Claude routing still implies `tmux`. I also hit an unrelated package test failure while validating, which is useful context because it means broad `pkg/spawn` test runs are currently noisier than they should be.

## Tension

The remaining judgment call is whether we are satisfied treating explicit `provider/model` strings as the long-term compatibility story, or whether we also want better inference for bare unknown model IDs that do not contain recognizable provider hints. The current behavior is safe for explicit specs, but bare future names may still default in surprising ways.
