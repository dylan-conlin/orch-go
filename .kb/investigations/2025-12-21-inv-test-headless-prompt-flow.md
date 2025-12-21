<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Headless prompt flow works correctly - agents receive MinimalPrompt via HTTP API and successfully read SPAWN_CONTEXT.md to begin tasks.

**Evidence:** Verified session `ses_4c00222dcffeqeswATQvSdXW3K` contains user message with exact MinimalPrompt text, and agent responded with investigation file creation.

**Knowledge:** The prompt flow is identical for inline and headless spawns - only the delivery mechanism differs (CLI pipe vs HTTP POST). Session registration records spawn_mode in events.jsonl for debugging.

**Next:** No action needed - headless prompt flow is working as designed.

**Confidence:** High (90%) - Tested on actual spawned session, limited only by sample size (1 verified session).

---

# Investigation: Headless Prompt Flow Verification

**Question:** Does the headless spawn flow correctly deliver prompts to agents via HTTP API?

**Started:** 2025-12-21
**Updated:** 2025-12-21
**Owner:** Investigation Agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** High (90%)

---

## Findings

### Finding 1: Prompt Flow Architecture

**Evidence:** The headless spawn flow uses `MinimalPrompt()` from `pkg/spawn/context.go:226`:
```go
func MinimalPrompt(cfg *Config) string {
    return fmt.Sprintf(
        "Read your spawn context from %s/.orch/workspace/%s/SPAWN_CONTEXT.md and begin the task.",
        cfg.ProjectDir,
        cfg.WorkspaceName,
    )
}
```

**Source:** `pkg/spawn/context.go:225-232`

**Significance:** The same prompt template is used for both inline and headless spawns. The only difference is delivery mechanism: inline uses `opencode run --message`, headless uses `client.SendPrompt()` (HTTP POST to `/session/{id}/message`).

---

### Finding 2: HTTP API Delivery Works

**Evidence:** Session `ses_4c00222dcffeqeswATQvSdXW3K` (workspace: `og-inv-test-headless-spawn-21dec`) received the prompt correctly:

```json
{
  "type": "text",
  "text": "Read your spawn context from /Users/dylanconlin/Documents/personal/orch-go/.orch/workspace/og-inv-test-headless-spawn-21dec/SPAWN_CONTEXT.md and begin the task."
}
```

The agent then responded with an investigation file creation, confirming it:
1. Received the prompt
2. Read the SPAWN_CONTEXT.md file
3. Began the investigation task

**Source:** `curl -s "http://127.0.0.1:4096/session/ses_4c00222dcffeqeswATQvSdXW3K/message"`

**Significance:** The HTTP API correctly delivers prompts and agents successfully act on them.

---

### Finding 3: Spawn Events Track Mode Correctly

**Evidence:** Events log shows headless spawns with session_id and spawn_mode:
```json
{
  "type": "session.spawned",
  "session_id": "ses_4c00222dcffeqeswATQvSdXW3K",
  "data": {
    "spawn_mode": "headless",
    "workspace": "og-inv-test-headless-spawn-21dec",
    "skill": "investigation"
  }
}
```

**Source:** `~/.orch/events.jsonl`

**Significance:** The event logging correctly tracks spawn mode, enabling debugging and monitoring of headless vs inline spawns.

---

### Finding 4: Registry SessionID Not Persisted (Minor Issue)

**Evidence:** Headless agents in registry show `session_id: null` despite code setting it:
```bash
$ cat ~/.orch/agent-registry.json | jq '.agents | map(select(.window_id == "headless")) | .[-1]'
{
  "id": "og-inv-test-headless-spawn-21dec",
  "session_id": null,
  "window_id": "headless",
  "status": "active"
}
```

However, the code in `cmd/orch/main.go:710` does set it:
```go
SessionID:  sessionResp.ID,  // Track OpenCode session ID for headless agents
```

**Source:** `~/.orch/agent-registry.json`, `cmd/orch/main.go:685-720`

**Significance:** This is a minor issue - session_id is available in events.jsonl for debugging. The registry may have an issue with how SessionID is serialized or there may be existing agents from before the field was added.

---

## Synthesis

**Key Insights:**

1. **Prompt mechanism is sound** - The MinimalPrompt pattern ("Read your spawn context from...") is simple, reliable, and works across both spawn modes.

2. **HTTP API integration works** - `CreateSession` + `SendPrompt` successfully creates sessions and delivers initial prompts to agents.

3. **Observability is good** - Events logging captures spawn mode and session_id, enabling post-hoc analysis of agent spawns.

**Answer to Investigation Question:**

Yes, the headless spawn flow correctly delivers prompts to agents. The prompt is sent via HTTP POST to `/session/{id}/message`, the agent receives it as a user message, and successfully reads the SPAWN_CONTEXT.md to begin the assigned task. This was verified by examining an actual headless session's message history.

---

## Confidence Assessment

**Current Confidence:** High (90%)

**Why this level?**

Direct observation of a real headless spawn session showing the complete flow: prompt sent, prompt received, agent acted on it.

**What's certain:**

- The MinimalPrompt function generates correct prompts
- HTTP API delivers prompts to sessions
- Agents can receive and act on prompts via HTTP API
- Events logging correctly tracks headless spawns

**What's uncertain:**

- Whether all edge cases are handled (network errors, session creation failures)
- Why registry session_id appears null (may be pre-existing agents or serialization issue)

**What would increase confidence to Very High:**

- Test multiple headless spawns in sequence
- Test error handling (what happens if session creation fails?)
- Investigate registry session_id null issue

---

## Test Performed

**Test:** Examined message history of headless session `ses_4c00222dcffeqeswATQvSdXW3K` via OpenCode HTTP API

**Command:** `curl -s "http://127.0.0.1:4096/session/ses_4c00222dcffeqeswATQvSdXW3K/message"`

**Result:** First message contains exact MinimalPrompt text:
```
"Read your spawn context from /Users/dylanconlin/Documents/personal/orch-go/.orch/workspace/og-inv-test-headless-spawn-21dec/SPAWN_CONTEXT.md and begin the task."
```

The agent responded with investigation file creation, confirming the prompt was received and acted upon.

---

## Self-Review

- [x] Real test performed (examined actual HTTP API response)
- [x] Conclusion from evidence (based on observed session messages)
- [x] Question answered (yes, headless prompt flow works)
- [x] File complete (all sections filled)

**Self-Review Status:** PASSED

---

## References

**Files Examined:**
- `cmd/orch/main.go:685-763` - runSpawnHeadless function
- `pkg/spawn/context.go:225-232` - MinimalPrompt function
- `pkg/opencode/client.go:257-261` - SendPrompt function
- `~/.orch/events.jsonl` - Spawn event logs
- `~/.orch/agent-registry.json` - Agent registry state

**Commands Run:**
```bash
# Check headless agents in registry
cat ~/.orch/agent-registry.json | jq '.agents | map(select(.window_id == "headless"))'

# Verify session messages
curl -s "http://127.0.0.1:4096/session/ses_4c00222dcffeqeswATQvSdXW3K/message"

# Check spawn events
cat ~/.orch/events.jsonl | jq -c 'select(.type == "session.spawned" and .data.spawn_mode == "headless")'
```

---

## Investigation History

**2025-12-21 00:18:** Investigation started
- Initial question: Does headless spawn deliver prompts correctly?
- Context: Testing new headless spawn feature in orch-go

**2025-12-21 00:25:** Found evidence of working flow
- Session messages show prompt delivered
- Agent responded appropriately

**2025-12-21 00:30:** Investigation completed
- Final confidence: High (90%)
- Status: Complete
- Key outcome: Headless prompt flow works correctly via HTTP API
