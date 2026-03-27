# Brief: orch-go-0vm6n

## Frame

A lifecycle validation sweep flagged `TestExploreNoJudgeModelOmitsFlag` as failing. The test exists to ensure that when no judge model is configured, the exploration context doesn't inject a `--model` flag into the judge spawn command.

## Resolution

The test was right about what it wanted to verify but wrong about how it checked. It asserted that `--model` appears nowhere in the entire generated context — but the worker template also emits `--model <model-name>` in a `kb create investigation` instruction that has nothing to do with judging. The judge spawn command itself was correctly conditional (guarded by `{{if .ExploreJudgeModel}}`). The template never regressed; the test was overly broad from the start. Fix: scan only lines containing `exploration-judge` for the `--model` flag.

## Tension

This was introduced as a regression but was actually a latent test defect — the test only started failing when the `kb create investigation --model` instruction was added to the worker template. That means there's no git bisect point where "the judge flag broke." The question is whether other assertion patterns in the explore test suite have similar whole-content matching that could false-positive as the template grows.
