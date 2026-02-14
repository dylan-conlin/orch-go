# Probe: Worker Detection Header Implementation

**Date:** 2026-02-14
**Status:** Active
**Model:** Coaching Plugin

## Question

Does re-implementing the `x-opencode-env-ORCH_WORKER` header reading in OpenCode's session.ts successfully restore worker detection for coaching metrics?

## What I Tested

1. Verified `orch spawn` sets header `x-opencode-env-ORCH_WORKER=1` in client.go:563
2. Verified coaching plugin checks `session?.metadata?.role === 'worker'` at coaching.ts:1323
3. Located session creation flow:
   - routes/session.ts POST "/" → Session.create() → createNext()
   - Header is sent but never read on server side

## What I Observed

**Root cause confirmed:** The server-side code in `routes/session.ts` does not read the `x-opencode-env-ORCH_WORKER` header and set `metadata.role = 'worker'` during session creation.

**Current flow:**
1. ✅ orch spawn sets header (pkg/opencode/client.go:563)
2. ❌ OpenCode server ignores header (routes/session.ts:204-208)
3. ❌ Session created without metadata.role
4. ✅ Coaching plugin checks metadata.role but never finds 'worker' (coaching.ts:1323)

**Expected fix:** Modify POST "/" handler in routes/session.ts to:
1. Read `x-opencode-env-ORCH_WORKER` header from request
2. Set `metadata.role = 'worker'` when header is present
3. Pass metadata through to Session.create()

## Model Impact

**Target Invariants:**
- Invariant 5: Worker detection caching is one-way - Only cache `true` results (confirmed worker), never cache `false`

**Expected Result:** Confirms that server-side header reading + metadata.role setting is the missing piece that prevents worker health tracking from firing.
