# Brief: orch-go-0ocus

## Frame

A spawn test was reportedly blocking `pkg/spawn` test runs — `TestExploreNoJudgeModelOmitsFlag` would fail because it checked for `--model` anywhere in the generated context, catching a legitimate `kb create investigation --model <model-name>` line that has nothing to do with judge spawning. The issue was filed as a follow-up to a resolver validation sweep.

## Resolution

Turns out this was already fixed. Another agent (`orch-go-0vm6n`) diagnosed and repaired the identical problem: narrowed the test assertion to only flag `--model` on lines that also contain `exploration-judge`. That fix landed in commit `1dd94e3be`. I confirmed all 21 explore/judge tests pass cleanly. No additional code changes were needed — this issue is a duplicate.

## Tension

Two issues were filed for the same bug from the same validation sweep. The spawn lifecycle sweep that surfaced the bug may have been too broad, creating duplicate tickets when findings weren't deduped before issue creation. Worth asking: should daemon/sweep-originated issues check for existing open issues with overlapping scope before creating new ones?
