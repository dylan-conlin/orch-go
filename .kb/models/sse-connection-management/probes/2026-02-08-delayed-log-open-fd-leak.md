# Probe: Do follow-mode SSE handlers leak file descriptors when events.jsonl appears after connect?

**Model:** `.kb/models/sse-connection-management.md`
**Date:** 2026-02-08
**Status:** Complete

---

## Question

When `/api/agentlog?follow=true` or `/api/events/services?follow=true` starts while `~/.orch/events.jsonl` does not exist, does later file creation cause leaked open file descriptors after clients disconnect?

---

## What I Tested

**Command/Code:**
```bash
# Repro on pre-fix binary
HOME=/tmp/orch-sse-leak-home ./build/orch serve --port 4351
# 15 follow clients connect while log file absent, then file is created, clients exit
curl -skN --max-time 3 "https://localhost:4351/api/agentlog?follow=true"
curl -skN --max-time 3 "https://localhost:4352/api/events/services?follow=true"
# Measured leaked descriptors for ~/.orch/events.jsonl
lsof -nP -p <pid> | rg -F "/tmp/orch-sse-leak-home/.orch/events.jsonl" | wc -l

# Verify after fix
go build -o /tmp/orch-fixed ./cmd/orch
HOME=/tmp/orch-sse-leak-home-fixed /tmp/orch-fixed serve --port 4353
```

**Environment:**
- Repo: `/Users/dylanconlin/Documents/personal/orch-go`
- Isolated HOME dirs so `events.jsonl` starts missing
- 15 concurrent short-lived follow-mode SSE clients per endpoint

---

## What I Observed

**Output:**
```text
Pre-fix /api/agentlog?follow=true:
before_open_log_fds=0
after_clients_done_log_fds=15
gor=11 fd=27

Pre-fix /api/events/services?follow=true:
before_open_log_fds=0
after_clients_done_log_fds=15
gor=11 fd=27

Post-fix /api/agentlog?follow=true:
before_open_log_fds=0
after_clients_done_log_fds=0
gor=11 fd=12

Post-fix /api/events/services?follow=true:
before_open_log_fds=0
after_clients_done_log_fds=0
gor=11 fd=12
```

**Key observations:**
- Both follow-mode handlers leaked one `events.jsonl` FD per disconnected client when the file was opened after initial missing-file startup.
- Leak disappeared after adding unconditional deferred close that also covers late-opened file handles.

---

## Model Impact

**Verdict:** extends — connection-lifetime cleanup requirements for SSE paths

**Details:**
The model correctly describes SSE pool pressure and reconnection behavior, but this probe adds a cleanup invariant: SSE handlers that poll and lazily open resources must always close late-opened descriptors on handler exit. Without that, sustained reconnect churn leaks FDs even when goroutines return.

**Confidence:** High — reproduced twice pre-fix on two endpoints and verified zero leaked descriptors post-fix with the same workload.
