# Decision: Use launchd for Dev Server Management

**Date:** 2025-12-28
**Status:** Accepted
**Deciders:** Dylan, Orchestrator

## Context

Need infrastructure for managing dev servers across projects. Evaluated 4 options:
- launchd plists per-project
- Docker Compose + restart policies
- Nix/devbox
- Hybrid (mix approaches)

## Decision

**Use launchd plists per-project** with Docker Compose integration for containerized services.

## Rationale

Dylan's context is unique: AI does all implementation work. This shifts the evaluation criteria:

| Criterion | Weight | Winner |
|-----------|--------|--------|
| Operational invisibility | High | launchd - no Docker Desktop to think about |
| AI debugging speed | Medium | Tie - both have good tooling |
| Reboot resilience | High | launchd - native macOS capability |
| Conceptual simplicity | High | launchd - "launchd runs my servers" (one concept) |

**Key insight:** "Implementation complexity" is NOT a cost when AI handles it. Only Dylan's cognitive load matters.

## Implementation

- `orch servers init <project>` - scans project, generates `.orch/servers.yaml` + launchd plists
- `orch servers up/down/status` - lifecycle management via launchd
- Docker projects: launchd plist starts `docker compose up -d`
- Plists inherit shell PATH (fix: 2025-12-28)
- Detection analyzes Go file content for actual server patterns (fix: 2025-12-28)

## Consequences

**Positive:**
- Dylan never thinks about dev servers after initial setup
- Mac reboot → servers just start
- Proven pattern (orch daemon already uses launchd)

**Negative:**
- macOS-only (acceptable: Dylan only uses macOS)
- Docker projects still need Docker Desktop (launchd handles "is Docker running?")

## References

- Investigation: `.kb/investigations/2025-12-27-design-dev-servers-infrastructure-evaluate-options.md`
- Implementation: `pkg/servers/`, `cmd/orch/servers.go`
