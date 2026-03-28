# Brief: orch-go-tw4po

## Frame

The daemon trusted one signal — "issue is open" — to mean "work is needed." But today three spawns found their work already done. An architect created a follow-up issue for a fix that was already committed. Two separate issues described the same failing test with different words. The pipeline had five layers of dedup, but all five answered the same question: "has this specific issue been spawned before?" None asked: "has this work already been done?"

## Resolution

I added two new spawn gates. The first (CommitDedupGate, L6) extracts beads IDs referenced in the issue description and checks git log for commits containing those IDs. For orch-go-94bxz, this would find `orch-go-paatt` in the description, see commit 7e911222b, and reject the spawn — the work was already committed. The second (KeywordDedupGate, L7) computes keyword overlap between the candidate issue's title and recently spawned issues. "Fix failing spawn exploration judge flag test" and "Fix unrelated pkg/spawn explore judge model test failure" share {fix, spawn, judge, test} — a 57% overlap coefficient that triggers rejection.

The interesting moment was when hardcoding the git log check into `buildSpawnPipeline` broke a concurrency test. Ten goroutines racing to spawn the same issue — the fresh status gate normally serializes them so only one wins. But the ~50ms git subprocess call widened the race window enough for four goroutines to slip through. The fix: make the git check injectable on the Daemon struct (nil in tests, wired in production), following the pattern already used by SessionDedupGate and TitleDedupBeadsGate.

## Tension

The keyword overlap threshold (50% coefficient + 3 common keywords) is tuned to catch the known duplicates, but I haven't measured its false positive rate in production. Two issues like "Fix spawn test timeout" and "Fix spawn context test" would also trigger (60% overlap, 3 common keywords) — and those might be genuinely different work. The spawn-time check is a bandaid over the deeper problem: the architect skill created a follow-up issue without checking if the work was already committed. The orientation frame asked whether dedup belongs at issue-creation time or spawn-time — this fix addresses spawn-time only, and the question stands.
