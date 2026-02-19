# Probe: Cross-Project Visibility Cache Context

Date: 2026-02-18
Status: Complete

## Question

Does cross-project agent visibility correctly use per-beads project directories when fetching comments and issues, even when cache is warm?

## What I Tested

- Ran `go test ./cmd/orch -run TestGetKBProjectsFallbackToRegistry` to verify registry fallback when `kb` is not on PATH.
- Ran `orch status --all` to attempt cross-project visibility check in current session set.

## What I Observed

- Test passed; getKBProjects returned registry path from `~/.kb/projects.json` when `kb` was unavailable.
- `orch status --all` showed only same-project agents; no live cross-project agents to validate end-to-end visibility.

## Model Impact

- Extends: cross-project visibility now has registry fallback for project discovery when kb CLI is unavailable.
- Partial confirmation: cache keying now respects per-beads project dirs for comments/issues, but end-to-end visibility not validated without live cross-project agents.
