## Summary (D.E.K.N.)

**Delta:** The spawn_mode config value 'opencode' was being ignored because only 'claude' had an else-if branch.

**Evidence:** Code at line 1188 checked `projCfg.SpawnMode == "claude"` but had no branch for `"opencode"`, so spawnBackend stayed at default "claude".

**Knowledge:** When adding new config options, all valid values must be explicitly handled - implicit defaults don't catch new values.

**Promote to Decision:** recommend-no (tactical fix, not architectural)

---

# Investigation: Fix Spawn Mode Config Bug

**Question:** Why isn't spawn_mode: opencode in .orch/config.yaml being respected?

**Started:** 2026-01-20
**Updated:** 2026-01-20
**Owner:** Worker agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Config has spawn_mode: opencode but agents spawn with claude backend

**Evidence:** `.orch/config.yaml` contains:
```yaml
spawn_mode: opencode
opencode:
    model: deepseek
```
But agents were being spawned with claude backend.

**Source:** `/Users/dylanconlin/Documents/personal/orch-go/.orch/config.yaml`

**Significance:** This is the root cause - the config value is correct but not being respected by spawn_cmd.go.

---

### Finding 2: else-if branch only handles 'claude', not 'opencode'

**Evidence:** At line 1188 in spawn_cmd.go:
```go
} else if projCfg != nil && projCfg.SpawnMode == "claude" {
    // Config default: respect project spawn_mode setting
    spawnBackend = "claude"
}
```
No handling for `SpawnMode == "opencode"`.

**Source:** `cmd/orch/spawn_cmd.go:1188`

**Significance:** spawnBackend defaults to "claude" at line 1143, so without an explicit branch for "opencode", the config value is ignored.

---

### Finding 3: Fix is a simple else-if addition

**Evidence:** Added:
```go
} else if projCfg != nil && projCfg.SpawnMode == "opencode" {
    // Config default: respect project spawn_mode setting (for DeepSeek/pay-per-token)
    spawnBackend = "opencode"
}
```

**Source:** `cmd/orch/spawn_cmd.go:1191-1193`

**Significance:** Minimal change, follows existing pattern, enables DeepSeek as default worker backend as intended.

---

## Synthesis

**Key Insights:**

1. **Missing branch for new config value** - When the spawn_mode config was extended to support "opencode" (for DeepSeek), the code only added the "claude" branch but not the "opencode" branch.

2. **Default masking the bug** - Since spawnBackend defaults to "claude", the missing branch wasn't immediately obvious - agents still spawned, just with the wrong backend.

**Answer to Investigation Question:**

The spawn_mode config value was being read correctly but not acted upon because the else-if chain at line 1188 only checked for "claude" and didn't have a branch for "opencode". Since spawnBackend defaults to "claude", the "opencode" config value was effectively ignored.

---

## Structured Uncertainty

**What's tested:**

- ✅ Code compiles after fix (verified: `go build ./cmd/orch`)
- ✅ Spawn-related tests pass (verified: `go test ./cmd/orch/... -run Spawn`)
- ✅ All project tests pass except pre-existing failures in pkg/model (verified: `go test ./...`)

**What's untested:**

- ⚠️ End-to-end spawn with opencode backend (not tested - would require spawning actual agent)

**What would change this:**

- Finding would be wrong if spawnBackend had multiple code paths that could override the config setting

---

## Implementation Recommendations

### Recommended Approach ⭐

**Add else-if branch for opencode** - One-line fix following existing pattern.

**Why this approach:**
- Minimal change
- Follows existing code pattern
- Directly addresses the bug

**Implementation sequence:**
1. Add else-if branch (done)
2. Verify build passes (done)
3. Verify tests pass (done)

---

## References

**Files Examined:**
- `cmd/orch/spawn_cmd.go` - spawn backend selection logic
- `.orch/config.yaml` - project config showing spawn_mode: opencode

**Commands Run:**
```bash
# Build verification
go build ./cmd/orch

# Test verification
go test ./cmd/orch/... -run Spawn
go test ./...
```
