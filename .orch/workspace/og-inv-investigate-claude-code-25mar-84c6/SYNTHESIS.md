# Session Synthesis

**Agent:** og-inv-investigate-claude-code-25mar-84c6
**Issue:** orch-go-zxe2j
**Duration:** 2026-03-25 ~12:50PM → ~1:15PM
**Outcome:** success

---

## Plain-Language Summary

Claude Code has three working mechanisms for external process injection: (1) stream-JSON stdin mode where an external process holds a pipe and writes messages whenever it wants, (2) session resume where `claude -p --resume <id> "new prompt"` loads full conversation history into a new process and adds a turn, and (3) Channels (research preview) where an MCP server receives webhooks and forwards them as notifications. The first two are available today in v2.1.83. This means a daemon can wake up an orchestrator session without waiting for Dylan to start a conversation — either by running a persistent stream-JSON session that receives work on stdin, or by resuming the orchestrator's session on demand to inject completion events.

---

## Delta (What Changed)

### Files Created
- `.kb/models/claude-code-agent-configuration/probes/2026-03-25-probe-external-wakeup-mechanisms.md` — Probe documenting three external injection vectors with empirical evidence
- `.orch/workspace/og-inv-investigate-claude-code-25mar-84c6/VERIFICATION_SPEC.yaml` — Test evidence
- `.orch/workspace/og-inv-investigate-claude-code-25mar-84c6/SYNTHESIS.md` — This file
- `.orch/workspace/og-inv-investigate-claude-code-25mar-84c6/BRIEF.md` — Comprehension artifact

### Files Modified
- `.kb/models/claude-code-agent-configuration/model.md` — Added "External Injection Vectors" section, updated Evaluate Next table (HTTP hooks superseded by stream-json), added probe reference

---

## Evidence (What Was Observed)

- **Stream-JSON multi-turn**: Sent two sequential JSON messages via stdin pipe; both processed in same session. Output confirmed: `"text":"FIRST_MESSAGE"`, `"text":"SECOND_MESSAGE"`.
- **Session resume with context**: Created session with secret word "PINEAPPLE", resumed with `--resume`, model recalled word correctly. `cache_read_input_tokens: 11916` confirms full history loaded.
- **Resume + stream-json**: The combination `--resume + --input-format stream-json` is valid — hooks fire with SessionStart:resume subtype.
- **Background tasks are pull-only**: `run_in_background` writes to file; no push mechanism. Binary string search confirmed no inotify/fsnotify wakeup.
- **Claude Code is Mach-O arm64 binary**: Compiled with Bun runtime, not inspectable Node.js source.
- **`--input-format stream-json` undocumented**: GitHub issue #24594 tracks this. Schema discovered empirically.

### Tests Run
```bash
# Test 1: Stream-JSON bidirectional
(echo '{"type":"user","message":{"role":"user","content":"Say exactly: FIRST_MESSAGE"}}'; \
 sleep 5; \
 echo '{"type":"user","message":{"role":"user","content":"Say exactly: SECOND_MESSAGE"}}') \
| claude -p --input-format stream-json --output-format stream-json --verbose \
  --dangerously-skip-permissions --effort low 2>/dev/null | grep -o '"text":"[^"]*"'
# Result: Both messages processed

# Test 2: Session resume
SESSION_ID=$(echo "Remember: the secret word is PINEAPPLE" | claude -p --output-format json ...)
echo "What was the secret word?" | claude -p --resume "$SESSION_ID" --output-format stream-json --verbose ...
# Result: "PINEAPPLE" — full context recall
```

---

## Architectural Choices

No architectural choices — this was pure investigation mapping existing capabilities.

---

## Knowledge (What Was Learned)

### Constraints Discovered
- Stream-JSON requires `--verbose` when combined with `--output-format stream-json` (otherwise errors)
- Session resume fires SessionStart hooks with subtype "resume" — distinguishable from fresh sessions
- Background process notification is pull-only — no push mechanism exists in Claude Code today
- `--input-format stream-json` requires `-p` (print mode) — not available in interactive TUI mode

### Decisions Made
- None (investigation only — architectural decisions for wakeup integration deferred to architect phase)

---

## Next (What Should Happen)

**Recommendation:** close (investigation complete, findings merged to model)

### If Close
- [x] All deliverables complete
- [x] Probe file created with all 4 required sections
- [x] Model updated with new findings
- [x] VERIFICATION_SPEC.yaml with test evidence
- [x] Ready for `orch complete orch-go-zxe2j`

### Follow-up Work (for orchestrator to decide)
If Dylan wants to pursue async comprehension queue processing, the next step would be an architect spawn to design the integration:
- **Option A**: Persistent stream-JSON orchestrator session (daemon writes to stdin pipe)
- **Option B**: On-demand resume injection (daemon invokes `claude -p --resume` per completion)
- **Option C**: Channels MCP (when it exits research preview)

Each has different tradeoffs around session lifecycle, context window management, and error recovery.

---

## Unexplored Questions

- **Context window limits on resumed sessions**: If the orchestrator session accumulates many turns via resume injection, does context window management (compression) work correctly in print mode?
- **Concurrent resume safety**: What happens if two processes try to `--resume` the same session simultaneously?
- **Channels GA timeline**: When will Channels exit research preview? This is the designed solution.
- **Tmux TUI injection**: `tmux send-keys` should work for interactive sessions but was not tested to completion due to trust dialog interference. Worth verifying for the interactive orchestrator use case.
- **Stream-JSON message types beyond "user"**: What other message types does stream-json accept? Can control responses (permission approvals) be sent?

---

## Friction

- **tooling**: `strings` search on compiled Bun binary returned massive amounts of HTML/CSS noise embedded in the binary, making binary analysis impractical. Switched to empirical testing instead.
- **ceremony**: Trust dialog on test tmux session prevented quick tmux injection test. Had to abandon that test vector.

---

## Session Metadata

**Skill:** investigation
**Model:** claude-opus-4-6[1m]
**Workspace:** `.orch/workspace/og-inv-investigate-claude-code-25mar-84c6/`
**Probe:** `.kb/models/claude-code-agent-configuration/probes/2026-03-25-probe-external-wakeup-mechanisms.md`
**Beads:** `bd show orch-go-zxe2j`
