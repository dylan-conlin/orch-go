# Session Synthesis

**Agent:** og-debug-investigate-headless-codex-18feb-0eb3
**Issue:** orch-go-1051
**Duration:** 2026-02-18 → 2026-02-18
**Outcome:** partial

---

## TLDR

Traced the headless spawn path and found two primary “silent death” vectors: prompt_async errors are only logged server-side (no error returned to orch-go), and session listing is scoped by the directory header so curl/dashboard without `x-opencode-directory` won’t show the new session. CreateSession ignores TTL/model in the body, so TTL cleanup is not deleting these sessions early.

---

## Delta (What Changed)

### Files Created
- `.orch/workspace/og-debug-investigate-headless-codex-18feb-0eb3/SYNTHESIS.md` - Investigation synthesis

---

## Evidence (What Was Observed)

- Headless spawn calls CreateSession + SendMessageInDirectory without any post-check for session existence or prompt success: `pkg/orch/extraction.go:1152`, `pkg/orch/extraction.go:1242`.
- OpenCode resolves project scope from `x-opencode-directory` (or query) and defaults to server cwd, so listing sessions without the header will show a different project: `packages/opencode/src/server/server.ts:197`.
- POST `/session` only passes title/permission/metadata; directory/time_ttl are not part of the create schema and are ignored in the body: `packages/opencode/src/server/routes/session.ts:203`, `packages/opencode/src/session/index.ts:204`.
- `prompt_async` returns immediately and only logs errors server-side; caller never receives failure: `packages/opencode/src/server/routes/session.ts:750`.
- Prompt errors publish `session.error` events, but those are only visible via SSE or server logs, not returned to orch-go: `packages/opencode/src/session/prompt.ts:336`, `packages/opencode/src/session/processor.ts:376`.
- TTL cleanup only applies if `time.ttl` is set; `Session.create` does not set TTL, so worker TTL in orch-go is currently ignored: `packages/opencode/src/session/cleanup.ts:17`, `packages/opencode/src/session/index.ts:204`.

### Tests Run
```bash
# Not run (investigation only)
```

---

## Knowledge (What Was Learned)

### Constraints Discovered
- `prompt_async` is fail-quiet: errors are only logged and emitted as `session.error`, so headless spawns can look “successful” even when the provider/model/auth fails.
- Session visibility is project-scoped; missing `x-opencode-directory` causes false “session missing” symptoms for headless spawns.
- TTL passed from orch-go is ignored because OpenCode `Session.create` doesn’t accept/set TTL.

---

## Next (What Should Happen)

**Recommendation:** spawn-follow-up

### If Spawn Follow-up
**Issue:** “Headless spawn should verify session + surface prompt_async errors”
**Skill:** feature-impl (or systematic-debugging if reproducing)
**Context:**
```
Headless spawn creates session and sends prompt_async, but prompt failures are only logged server-side and session listing without x-opencode-directory hides the session. Add verification after CreateSession (GET /session/:id with directory header), and optionally subscribe to session.error or add a synchronous prompt endpoint to surface errors. Also update dashboard/curl guidance to pass x-opencode-directory.
```

### If Escalate
**Question:** Where should error propagation be fixed: orch-go post-checks vs OpenCode prompt_async response semantics?
**Options:**
1. Add orch-go post-check + SSE error probe (no OpenCode API change).
2. Extend OpenCode prompt_async to return errors (API change, clearer behavior).
**Recommendation:** Start with option 1 for immediate visibility, then option 2 if we want consistent API semantics.

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Does the dashboard pass `x-opencode-directory` or query `directory`, and if not, should it?
- Are codex auth failures emitting clear `session.error` payloads in server logs?

---

## Session Metadata

**Skill:** systematic-debugging
**Model:** gpt-5.2-codex
**Workspace:** `.orch/workspace/og-debug-investigate-headless-codex-18feb-0eb3/`
**Beads:** `bd show orch-go-1051`
