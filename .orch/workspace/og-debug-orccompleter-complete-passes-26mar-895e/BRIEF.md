# Brief: orch-go-o0nqu

## Frame

The daemon's auto-complete path had a flag that would fail the moment it was actually called. `OrcCompleter.Complete()` passed `--force` to `orch complete`, but `--force` requires a `--reason` argument (minimum 10 characters) that was never provided. No one noticed because no auto-tier agents currently complete through this path — it was a landmine waiting for traffic.

## Resolution

This was the same class of bug fixed a few hours earlier in orch-go-uiv9d, where `CompleteLight` had the same problem. The fix is identical: replace `--force` with `--headless`, which skips interactive prompts without requiring a reason string. I extracted a `BuildCompleteCommand()` function so the args construction is unit-testable — the test explicitly asserts `--headless` is present and `--force` is absent, preventing regression.

## Tension

All three `OrcCompleter` methods now use `--headless`, which raises a question: is `--force` still serving any purpose on `orch complete`, or is it dead code that should be deprecated? It exists but every automated caller has migrated away from it.
