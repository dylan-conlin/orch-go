## Summary (D.E.K.N.)

**Delta:** Agent orch-go-ytdp was spawned with Opus (not Sonnet as reported); the model selection system is working correctly.

**Evidence:** events.jsonl line 3056 shows `"model":"anthropic/claude-opus-4-5-20251101"` for session ses_48e597de6ffeVTU5uxqM6B3RFT.

**Knowledge:** orch spawn correctly defaults to Opus via `model.DefaultModel`; daemon spawn path does not override model; the "used sonnet" claim was incorrect.

**Next:** Close investigation - no bug exists. Original report was mistaken about which model was used.

---

# Investigation: Why Did Agent orch-go-ytdp Use Sonnet Instead of Opus?

**Question:** Did agent orch-go-ytdp actually use claude-sonnet-4 model, and if so, how did this happen when Opus should be the default?

**Started:** 2025-12-30
**Updated:** 2025-12-30
**Owner:** Orchestrator
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Model Resolution Code Defaults to Opus

**Evidence:** In `cmd/orch/main.go` line 1422, the spawn command calls `model.Resolve(spawnModel)`. When `spawnModel` is empty string (default), this returns `model.DefaultModel` which is defined in `pkg/model/model.go` as `claude-opus-4-5-20251101`.

**Source:** 
- `cmd/orch/main.go:1422` - `resolvedModel := model.Resolve(spawnModel)`
- `pkg/model/model.go:14` - `DefaultModel = "claude-opus-4-5-20251101"`

**Significance:** The code path correctly defaults to Opus. No code path exists that would accidentally select Sonnet.

---

### Finding 2: Daemon Spawn Path Uses Same Model Default

**Evidence:** The daemon calls `SpawnWork(beadsID)` in `pkg/daemon/daemon.go` which executes `orch work <beadsID>`. The `orch work` command calls `runSpawnWithSkill()` without explicitly passing a model, relying on the same default resolution path.

**Source:**
- `pkg/daemon/daemon.go:603-609` - `SpawnWork()` function
- `cmd/orch/main.go:1076` - `orch work` command implementation

**Significance:** Both manual `orch spawn` and daemon-driven spawns use identical model resolution, both defaulting to Opus.

---

### Finding 3: Events Log Proves Agent Was Spawned With Opus

**Evidence:** The events.jsonl log entry for agent orch-go-ytdp explicitly shows:
```json
{
  "type": "session.spawned",
  "session_id": "ses_48e597de6ffeVTU5uxqM6B3RFT",
  "timestamp": 1767138296,
  "data": {
    "beads_id": "orch-go-ytdp",
    "model": "anthropic/claude-opus-4-5-20251101",
    ...
  }
}
```

**Source:** `~/.orch/events.jsonl` line 3056

**Significance:** This is definitive evidence that the agent WAS spawned with Opus, not Sonnet. The original report that "agent used sonnet" was incorrect.

---

### Finding 4: Daemon Spawn Confirmed

**Evidence:** Immediately following the session.spawned event, there's a daemon.spawn event:
```json
{
  "type": "daemon.spawn",
  "timestamp": 1767138296,
  "data": {
    "beads_id": "orch-go-ytdp",
    "count": 9,
    "skill": "systematic-debugging"
  }
}
```

**Source:** `~/.orch/events.jsonl` line 3057

**Significance:** Confirms this agent was spawned by the daemon (not manually), and the daemon correctly used Opus.

---

## Synthesis

**Key Insights:**

1. **Model selection system is working correctly** - Both manual and daemon spawn paths default to Opus as designed.

2. **Events log is the source of truth** - When there's ambiguity about what model was used, check events.jsonl which records the actual model at spawn time.

3. **Original report was mistaken** - The claim that agent orch-go-ytdp "used sonnet" was incorrect. The agent was provably spawned with Opus.

**Answer to Investigation Question:**

Agent orch-go-ytdp did NOT use Sonnet - it used Opus (`claude-opus-4-5-20251101`) as designed. The events.jsonl log definitively shows the model at spawn time. The model selection code in orch-go correctly defaults to Opus for both manual spawns and daemon-driven spawns. No bug exists.

---

## Structured Uncertainty

**What's tested:**

- ✅ Model resolution code path defaults to Opus (verified: traced code from spawn command through model.Resolve())
- ✅ Agent orch-go-ytdp was spawned with Opus (verified: events.jsonl session.spawned event shows model)
- ✅ Daemon spawn path uses same model default (verified: traced orch work → runSpawnWithSkill)

**What's untested:**

- ⚠️ Whether OpenCode can override model mid-session (not investigated - out of scope)
- ⚠️ How the original "used sonnet" claim originated (could be UI display issue, agent self-report error, etc.)

**What would change this:**

- Finding would be wrong if events.jsonl showed a different model than Opus
- Finding would be wrong if there was a code path that bypasses model.Resolve()

---

## Implementation Recommendations

**Recommended Approach: No Changes Needed**

The model selection system is working correctly. No implementation required.

**If false reports recur:**
1. Check events.jsonl for actual model used at spawn
2. Consider adding model display to `orch status` output for visibility
3. Investigate where the incorrect report originated (UI, agent self-report, etc.)

---

## References

**Files Examined:**
- `cmd/orch/main.go:1422` - Spawn command model resolution
- `pkg/model/model.go:14` - DefaultModel definition
- `pkg/daemon/daemon.go:603-609` - SpawnWork function
- `~/.orch/events.jsonl:3056-3057` - Spawn events for orch-go-ytdp

**Commands Run:**
```bash
# Search for model usage in spawn path
grep -n "spawnModel" cmd/orch/main.go

# Check events for specific agent
grep "orch-go-ytdp" ~/.orch/events.jsonl
```

---

## Investigation History

**2025-12-30:** Investigation started
- Initial question: Why did agent orch-go-ytdp use Sonnet instead of Opus?
- Context: Report that daemon-spawned agent used wrong model

**2025-12-30:** Traced model resolution code
- Found DefaultModel is Opus in pkg/model/model.go
- Found daemon spawn uses same path as manual spawn

**2025-12-30:** Found definitive evidence in events.jsonl
- Session ses_48e597de6ffeVTU5uxqM6B3RFT spawned with Opus
- Original report was incorrect

**2025-12-30:** Investigation completed
- Status: Complete
- Key outcome: No bug - agent was correctly spawned with Opus, original report was mistaken
