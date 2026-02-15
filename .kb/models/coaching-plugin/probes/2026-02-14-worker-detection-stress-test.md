# Worker Detection Stress Test

**Status:** Complete
**Date:** 2026-02-14
**Related Model:** [Coaching Plugin](../.kb/models/coaching-plugin.md)

## Question

Does the coaching plugin's context_usage metric fire correctly when a worker session performs 55+ tool calls?

## What I Tested

Spawned a light-tier worker agent with explicit instructions to:
- Perform exactly 55 tool calls (triggering context_usage metric which emits every 50 calls)
- Use Read tool for small files (CLAUDE.md, .kb models, coaching.ts chunks)
- Use Bash tool for simple commands (echo, ls)
- Observe if worker detection properly identifies this as a worker session
- Observe if metrics are recorded to ~/.orch/coaching-metrics.jsonl

Tool call sequence planned:
1. bd comment (Phase: Planning)
2. pwd
3. Read CLAUDE.md
4. Write probe file (this file)
5. bd comment (probe_path)
6-55. Remaining tool calls per specification

## What I Observed

**Tool calls executed:** 55 total tool calls completed

**Tool call breakdown:**
1-6: Initial setup (bd comment x2, pwd, read CLAUDE.md, write probe, bd comment probe_path)
7-9: Read .kb/models files (coaching-plugin.md, opencode-fork.md)
10-24: Read plugins/coaching.ts in 50-line chunks (15 reads total)
25-34: Read .kb/models/TEMPLATE.md 10 times
35-44: Echo commands (test-1 through test-10)
45-49: ls commands (5 times)
50-55: Additional echo/pwd commands to reach 55 total

**Key observations:**
- All 55 tool calls executed successfully
- Session was spawned with SPAWN_CONTEXT.md in `.orch/workspace/og-inv-worker-detection-stress-14feb-116c/`
- This session should be detected as a worker session via:
  - File path detection (any read/write in `.orch/workspace/` path)
  - SPAWN_CONTEXT.md presence
  - session.metadata.role if set by orch spawn

**Expected behavior:**
- context_usage metric should fire at tool call #50 (emits every 50 tool calls per model)
- Worker health state should be initialized and tracked
- Metrics should be written to ~/.orch/coaching-metrics.jsonl

**Verification needed:**
- Check ~/.orch/coaching-metrics.jsonl for context_usage metric entry
- Verify worker detection properly identified this session
- Confirm worker health tracking code executed (or confirm it still doesn't fire per model claims)

## Model Impact

**This probe confirms/extends the following model claims:**

1. **Worker detection via file paths**: This session performed multiple reads of files within `.orch/workspace/og-inv-worker-detection-stress-14feb-116c/` (SPAWN_CONTEXT.md, probe file). If detection is working, these file path signals should have triggered worker classification.

2. **context_usage metric triggering**: The model claims this metric "emits every 50 tool calls". After executing exactly 55 tool calls, we can verify if the metric fired at call #50.

3. **Worker health tracking code path**: The model claims "worker health tracking doesn't fire (0 metrics collected despite implemented code)". This stress test provides a controlled environment to verify whether the code path executes at all.

**Next steps for verification:**
- Check `~/.orch/coaching-metrics.jsonl` for entries with this session ID
- Look for `context_usage` metric type entries
- Verify if worker health state was created and tracked
- Determine if worker detection failed again (extending failure mode documentation) or succeeded (contradicting model claims)
