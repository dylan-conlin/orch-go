## Question

Does the OpenCode fork session metadata API cover create/update/status/TTL requirements for two-lane agent discovery?

## What I Tested

- `rg "metadata" /Users/dylanconlin/Documents/personal/opencode/packages/opencode/src/session/index.ts`
- `rg "metadata" /Users/dylanconlin/Documents/personal/opencode/packages/opencode/src/server/routes/session.ts`
- `rg "time_ttl|ttl" /Users/dylanconlin/Documents/personal/opencode/packages/opencode/src/session`
- Read `/Users/dylanconlin/Documents/personal/opencode/packages/opencode/src/session/session.sql.ts`
- Read `/Users/dylanconlin/Documents/personal/opencode/packages/opencode/src/session/status.ts`

## What I Observed

- Session schema stores `metadata` as `Record<string, string>` and exposes it on `Session.Info` (`/Users/dylanconlin/Documents/personal/opencode/packages/opencode/src/session/session.sql.ts`, `/Users/dylanconlin/Documents/personal/opencode/packages/opencode/src/session/index.ts`).
- Session create accepts optional `metadata` and forwards it to persistence (`/Users/dylanconlin/Documents/personal/opencode/packages/opencode/src/session/index.ts`).
- `PATCH /session/:sessionID` accepts `metadata` and calls `Session.setMetadata` (`/Users/dylanconlin/Documents/personal/opencode/packages/opencode/src/server/routes/session.ts`).
- Session status is tracked in-memory as `{ [sessionID]: { type } }` and does not include metadata (`/Users/dylanconlin/Documents/personal/opencode/packages/opencode/src/session/status.ts`).
- TTL cleanup skips busy sessions via `SessionPrompt.assertNotBusy` and only deletes expired, non-busy sessions (`/Users/dylanconlin/Documents/personal/opencode/packages/opencode/src/session/cleanup.ts`).

## Model Impact

- Confirms: session metadata exists and is persisted/returned via `Session.Info` in the fork.
- Confirms: TTL cleanup exists and guards against deleting busy sessions.
- Extends: `GET /session/status` returns only liveness status, not metadata.

## Status

- Complete
