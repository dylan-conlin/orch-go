# Probe: MCP Hot-Reload Production Failure

**Status:** Complete
**Date:** 2026-02-20
**Question:** Why does MCP hot-reload work in tests but not production despite passing tests?

## What I Tested

Testing the model claim that OpenCode fork session lifecycle includes proper MCP server initialization. The specific issue is orch-go-1126's hot-reload feature works in tests but not in production.

### 1. Code Path Analysis

Traced the initialization sequence and found a race condition.

**Bootstrap sequence (bootstrap.ts:17-28) — BEFORE FIX:**
```typescript
export async function InstanceBootstrap() {
  // ... other inits ...
  MCP.watchConfig()  // NOT AWAITED - returns before watcher ready!
}
```

**watchConfig (mcp/index.ts:1108-1110) — BEFORE FIX:**
```typescript
export function watchConfig() {
  configWatcherState()  // Returns Promise, NOT AWAITED
}
```

**configWatcherState init (mcp/index.ts:1033-1087):**
```typescript
const configWatcherState = Instance.state(
  async () => {
    const dir = Instance.directory
    // AWAIT happens here - BEFORE watcher setup!
    const initialConfig = await readProjectMcpConfig(dir)
    // ...
    // Watcher is set up AFTER the await
    let watcher: FSWatcher | undefined
    try {
      watcher = fsWatch(dir, (eventType, filename) => { ... })
    }
    // ...
  }
)
```

### 2. Test vs Production Difference

**Test sequence (config-watcher.test.ts) — BEFORE FIX:**
```typescript
MCP.watchConfig()  // Not awaited
const statusBefore = await MCP.status()  // This adds implicit delay!
// Write config file
await new Promise(resolve => setTimeout(resolve, 1500))  // Wait for watcher
```

The test worked by accident because `MCP.status()` is async and provides enough delay for the watcher Promise to resolve. Production doesn't have this implicit delay.

### 3. Race Condition Window

From when `watchConfig()` is called until `fsWatch()` completes:
- `readProjectMcpConfig(dir)` is awaited first (10-100ms file I/O)
- Only then is `fsWatch()` called
- Config changes during this window are missed

## What I Observed

**Root cause confirmed:** The async initialization is fire-and-forget at two levels:
1. `watchConfig()` didn't await `configWatcherState()` 
2. `InstanceBootstrap` didn't await `watchConfig()`

**Fix implemented:** Made `watchConfig()` async and awaited it in bootstrap.

## Model Impact

**Confirms** OpenCode Fork model claims about lazy state initialization.

**Extends** with new understanding:

### Understanding: Instance.state Async Init Returns Promise

When `Instance.state()` is called with an async init function:
1. Calling the returned function triggers init
2. The init function is called and its Promise is returned
3. If not awaited, initialization runs in background
4. Code after the call proceeds before state is ready

This is by design for lazy loading, but `MCP.watchConfig()` should have been async from the start since the watcher isn't ready until init completes.

## Verification

1. All 1116 OpenCode tests pass after fix
2. Added new test "watcher is ready immediately after watchConfig returns" that verifies the race condition fix
3. Test writes config immediately after `watchConfig()` returns and confirms it's detected

## Files Changed

- `packages/opencode/src/mcp/index.ts`: Made `watchConfig()` async, await `configWatcherState()`
- `packages/opencode/src/project/bootstrap.ts`: Await `MCP.watchConfig()`
- `packages/opencode/test/mcp/config-watcher.test.ts`: Updated tests to await `watchConfig()`, added race condition regression test
