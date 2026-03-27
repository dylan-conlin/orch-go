# Brief: orch-go-xjdx9

## Frame

The benchmark engine could run suites, but writing "opus" in a config was a trust-fall — it passed through as a raw string and you wouldn't know until spawn time whether it resolved correctly. No way to check a config before committing to a live run, no way to override all scenarios to test a different model, and no way to discover what suite files existed in a directory.

## Resolution

Wired `model.Resolve()` into the config loading path so aliases resolve eagerly at parse time. The dry-run now shows the full chain — `opus (suite default) → anthropic/claude-opus-4-5-20251101` — so you can see exactly what the engine will spawn before anything runs. Added a `default_model` field at the suite level to reduce boilerplate when testing one model across many scenarios, and a `--model` CLI flag that overrides everything for quick cross-model comparisons.

The surprise was how little code this needed. The model resolution infrastructure was already complete; the bench system just wasn't using it. The bulk of the work was validation polish (duplicate scenario names, timeout format checking) and CLI ergonomics (`bench validate`, `bench list`). A reference suite config lives in `benchmarks/worker-reliability.yaml`.

## Tension

Right now the resolved model ID is used only for display in dry-run — actual spawn still passes the original alias string, because `orch spawn` has its own resolution. This means the same alias gets resolved twice (once for display, once for spawn), and if the two resolution paths ever diverge, the dry-run would lie. Should bench pass the fully-resolved `provider/model` format to spawn instead, making dry-run output authoritative?
