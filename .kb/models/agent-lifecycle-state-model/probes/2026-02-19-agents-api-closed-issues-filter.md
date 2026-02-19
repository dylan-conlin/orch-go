# Probe: /api/agents closed issues filter

**Model:** agent-lifecycle-state-model
**Date:** 2026-02-19
**Status:** Complete

---

## Question

Does the tracked agents query path exclude closed beads issues when falling back to CLI listing?

---

## What I Tested

Checked current beads issues with the orch:agent label and compared them to the /api/agents response.

```bash
bd list -l orch:agent
curl -sk "https://localhost:3348/api/agents?since=all"
go test ./cmd/orch -run TestListTrackedIssuesCLIFiltersClosed
```

---

## What I Observed

- `bd list -l orch:agent` includes closed issues (e.g., orch-go-1085).
- `/api/agents?since=all` returned an entry with `beads_id: "orch-go-1085"` and `status: "completed"`.
- `go test ./cmd/orch -run TestListTrackedIssuesCLIFiltersClosed` passed, confirming closed issues are filtered in CLI fallback path.

---

## Model Impact

- [ ] **Confirms** invariant:
- [ ] **Contradicts** invariant:
- [x] **Extends** model with: CLI fallback path can surface closed beads issues in tracked agents results, causing /api/agents to include completed work.

---

## Notes

Observation indicates the tracked lane may include closed issues when beads RPC is unavailable and CLI fallback is used without status filtering.
