# Investigation: Dead Orch Commands

**Date:** 2026-02-05
**Status:** Active
**Beads:** orch-go-21306

## Problem

The orch CLI has 54 total commands, cluttering the help output. Need to identify which are actually used vs dead.

## Methodology

1. Listed all commands registered to rootCmd (54 total)
2. Checked git commit messages (last 3 months)
3. Checked usage in project docs/scripts
4. Checked usage in skills and kb files

## Evidence

### All Commands (54 total)

Commands registered via init() functions across multiple files:

**From main.go:**
spawn, send, monitor, status, complete, work, daemon, tail, claim, question, abandon, clean, registry, account, wait, focus, drift, next, review, version, port, init, retries, frontier, usage

**From other files:**
serve, stats, kb, deploy, doctor, config, handoff, automation, test-report, patterns, hotspot, model, logs, docs, guarded, emit, reconcile, resume, attach, history, changelog, transcript, sessions, servers, swarm, mode, learn, tokens, fetchmd

### Usage Analysis

**Git commits (last 3 months) - top mentions:**

- status: 40
- complete: 28
- review: 15
- clean: 14
- serve: 12
- doctor: 11
- spawn: 8
- learn: 8
- send: 7
- init: 7
- servers: 6

**Project docs/scripts - top mentions:**

- complete: 9879
- spawn: 8756
- servers: 5836
- status: 5683
- review: 1423
- serve: 1396
- clean: 917
- send: 865
- init: 659
- learn: 626
- daemon: 578
- tail: 547
- doctor: 513

**Skills/kb files - top mentions:**

- spawn: 264
- servers: 87
- complete: 81
- account: 29
- wait: 25
- kb: 24
- serve: 23
- daemon: 22
- doctor: 21
- monitor: 19

## Findings

### Commands with NO or VERY LOW usage (candidates for hiding):

**Zero or near-zero mentions:**

1. **claim** - not in docs/skills
2. **registry** - 1 mention in git only
3. **retries** - 0 mentions
4. **deploy** - 2 git mentions only
5. **config** - 0 mentions
6. **test-report** - 0 mentions
7. **model** - 0 mentions
8. **logs** - 4 git mentions only
9. **docs** - 0 mentions
10. **guarded** - 1 git mention only
11. **emit** - 1 git mention only
12. **history** - 1 skill mention only
13. **changelog** - 2 git mentions only
14. **transcript** - 0 mentions
15. **swarm** - 2 git mentions only
16. **tokens** - 0 mentions
17. **fetchmd** - 0 mentions

**Total dead commands: 17**

This matches the task estimate of "17+ unused commands".

### Commands to KEEP visible (actively used):

Core workflow: spawn, send, status, complete, review, work, abandon, clean, wait
Server management: serve, servers, daemon, doctor
Learning/monitoring: learn, monitor, tail, usage, stats
Project management: init, account, frontier, focus, drift, next
Integration: kb, attach, resume, sessions
Specialized but used: port, question, patterns, hotspot, handoff, automation, reconcile, mode

## Recommendation

Hide the 17 dead commands using Cobra's `Hidden` field:

- Keep them functional (don't remove)
- Hide from default help output
- Still accessible if users know the command name

## Implementation Plan

1. Add `Hidden: true` to each of the 17 dead command definitions
2. Verify help output is cleaner (54 → 37 commands visible)
3. Test that hidden commands still work when invoked directly

## Related

- Task: Hide dead orch commands (orch-go-21306)
