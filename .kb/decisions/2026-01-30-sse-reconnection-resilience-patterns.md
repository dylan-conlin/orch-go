# Decision: SSE Reconnection Resilience Patterns

**Date:** 2026-01-30
**Status:** Accepted
**Context:** Synthesized from investigations 2026-01-28-inv-sse-reconnection-opencode-client-survive.md and 2026-01-29-inv-server-restarts-strand-workers-4096.md

## Summary

SSE reconnection handled via conditional break fix in OpenCode SDK (serverSentEvents.gen.ts:220-221). Fix must survive SDK regeneration via post-processing step in build.ts. OpenCode server (:4096) restarts historically stranded workers; orch serve (:3348) restarts proven harmless. SSE reconnection (3-30s exponential backoff) makes auto-resume fallback rather than primary mechanism.

## The Problem

Before fix:
- Agent sessions died when OpenCode server crashed/restarts
- SSE stream broke, `for await` loop terminated, client process exited
- Unconditional `break` at serverSentEvents.gen.ts:220 prevented retry loop from executing
- Retry logic only ran if `reader.read()` threw error, not when stream completed normally (done=true)

Root cause: When server drops connection, `reader.read()` returns `{done: true}` (graceful close), breaks inner loop, finally block runs, then unconditional break exits outer retry loop. Catch block with retry logic never reached.

## The Decision

### Fix: Conditional Break in SSE Client

Change serverSentEvents.gen.ts:220 from:
```typescript
break  // Unconditional - exits retry loop
```

To:
```typescript
if (signal.aborted) break  // Only exit on abort, otherwise reconnect
```

**Effect:** Enables automatic reconnection when SSE stream completes normally. Client retries with exponential backoff (3-30s) instead of dying.

**Test confirmed:** Client PID 78918 survived server kill/restart, reconnected automatically, completed successfully with exit code 0.

### Make Fix Permanent: Post-Process in build.ts

SDK generation process uses @hey-api/openapi-ts which auto-generates serverSentEvents.gen.ts. Fix will be overwritten on regeneration.

**Solution:** Add post-processing step in `packages/sdk/js/script/build.ts`:

```typescript
// After line 41 (after prettier, before tsc)
// Fix SSE reconnection: change unconditional break to conditional
// See: .kb/decisions/2026-01-30-sse-reconnection-resilience-patterns.md
const sseFile = path.join(dir, "src/v2/gen/core/serverSentEvents.gen.ts")
const content = await Bun.file(sseFile).text()
const fixed = content.replace(
  /(\s+)break\s*$/m,  // Match unconditional break at end of line
  "$1if (signal.aborted) break  // Only exit on abort, otherwise reconnect"
)
await Bun.write(sseFile, fixed)
```

**Why post-processing (not upstream patch or template modification):**
- Survives SDK regeneration (runs every build)
- Simple 1-2 line addition to build.ts
- No external dependencies or upstream coordination
- Easy to review (change is visible in build script)
- Can be removed if @hey-api fixes upstream

### Server Architecture: OpenCode vs orch serve

**OpenCode server (:4096):**
- Provides SSE stream at `:4096/event`
- Agents connect directly for SSE events
- Restarts historically stranded workers (before fix)
- Now: SSE reconnection handles restarts (3-30s retry)

**orch serve (:3348):**
- Dashboard API server that PROXIES OpenCode SSE stream
- Does NOT originate SSE - only proxies
- Restarts have NO IMPACT on agent SSE connections
- Tested: Killed orch serve PID 78577, restarted as PID 15169, agent survived and executed commands

**Implication:** Auto-resume mechanism (from Jan 17 investigation) becomes safety net for edge cases, not primary recovery mechanism.

## Why This Design

### Principle: Fix Root Cause, Not Symptoms

Auto-resume (detecting stranded workers and restarting them) treats symptom. SSE reconnection prevents the problem - workers don't strand in first place.

### Constraint: SDK Regeneration Must Preserve Fix

Generated code changes are fragile. Post-processing in build.ts ensures fix persists across SDK updates.

### Architectural Fact: SSE Authority is OpenCode

Agents connect to OpenCode server for SSE stream, not orch serve. This separation means orch serve restarts are harmless, OpenCode restarts are critical.

### Lesson: Test Infrastructure Claims

Jan 26 investigation claimed SSE client already had retry logic. Examination revealed retry logic existed but was unreachable due to unconditional break. Testing confirmed the fix works.

## Trade-offs

**Accepted:**
- Fix is applied to generated code (not at source)
- Requires documentation so future maintainers understand why
- If @hey-api changes SSE generation significantly, regex may break
- Not testing OpenCode server kill with live agent (would risk losing work)

**Rejected:**
- Option A (Patch @hey-api/openapi-ts templates): Requires forking or modifying node_modules, fragile
- Option C (Contribute upstream): Cleanest long-term but uncertain timeline, use post-processing as bridge
- Pre-commit hooks for verification: Runs in sandbox, bypassable
- Spawn-time blocking: Prevents spawning if OpenCode down, breaks workflow

## Constraints

1. **SSE reconnection must survive SDK regeneration** - Post-processing in build.ts is mandatory
2. **Never rely on orch serve restarts as failure mode** - orch serve doesn't originate SSE
3. **3-30 second retry window is acceptable** - Exponential backoff with 30s max
4. **Last-Event-ID header enables event resumption** - Already implemented in SDK (line 110-112)

## Implementation Notes

**File to modify:**
- `packages/sdk/js/script/build.ts` - Add post-processing step after line 41

**Test procedure:**
```bash
# 1. Run SDK build
cd packages/sdk/js
bun run build

# 2. Verify fix persists
grep -A 1 "Only exit on abort" src/v2/gen/core/serverSentEvents.gen.ts

# 3. Test SSE reconnection
cd ../../../orch-go
bash test-sse-reconnect.sh
```

**Success criteria:**
- SDK regeneration preserves the SSE reconnection fix (verified by grep after build)
- Agents survive OpenCode server kill/restart (tested with test-sse-reconnect.sh)
- No new reports of stranded workers correlated to server restarts
- Dashboard or logs show successful SSE reconnections when they occur

**Monitoring needs:**
- Track SSE reconnection events (success/failure)
- Log OpenCode server restarts
- Alert if reconnection takes >30s (indicates problem)

## References

**Investigations:**
- `.kb/investigations/2026-01-28-inv-sse-reconnection-opencode-client-survive.md` - Root cause and fix
- `.kb/investigations/2026-01-29-inv-server-restarts-strand-workers-4096.md` - Server architecture
- `.kb/investigations/2026-01-26-inv-opencode-server-keeps-crashing-dying.md` - Original problem
- `.kb/investigations/2026-01-17-inv-design-auto-resume-mechanism-stalled.md` - Auto-resume fallback

**Files:**
- `opencode/packages/sdk/js/src/v2/gen/core/serverSentEvents.gen.ts:220-221` - SSE fix location
- `opencode/packages/sdk/js/script/build.ts:41` - Post-processing insertion point
- `pkg/opencode/client.go:20,836` - OpenCode client configuration
- `cmd/orch/serve.go:32,64` - orch serve as SSE proxy

**Test Script:**
- `test-sse-reconnect.sh` - Automated test for SSE reconnection behavior

**External:**
- `@hey-api/openapi-ts` v0.90.10 - OpenAPI TypeScript generator
- `@hey-api/client-fetch` - Plugin that generates serverSentEvents.gen.ts
