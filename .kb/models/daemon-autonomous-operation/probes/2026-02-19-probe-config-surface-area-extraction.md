# Probe: Config Surface Area Causes Agent Spiraling

**Model:** daemon-autonomous-operation
**Date:** 2026-02-19
**Status:** Complete

---

## Question

Does daemon config surface area (number of files touched per new field) cause agent spiraling, and can extraction into a dedicated package reduce it?

---

## What I Tested

Traced all code locations required when adding a single boolean (`reflect_open`) to daemon config. Counted files, identified duplicates, and mapped dependency chains.

---

## What I Observed

- Adding 1 boolean requires touching **10-12 files** across 5 layers (struct, YAML, CLI, plist, API, UI, tests).
- **Three identical Config structs** exist: `pkg/daemon/daemon.go:Config`, `pkg/daemonconfig/config.go:Config`, and runtime-equivalent fields in `pkg/userconfig/userconfig.go:DaemonConfig`.
- **Two copy-pasted plist templates**: `cmd/orch/config_cmd.go:plistTemplate` and `cmd/orch/serve_system.go:plistTemplateAPI` (with acknowledgment comments).
- **Two copy-pasted PlistData structs**: `config_cmd.go:PlistData` and `serve_system.go:PlistDataAPI`.
- Agent ek0b spiraled to 526K tokens because exploring this surface area required reading 10+ large files (daemon.go 1034 lines, serve_system.go 1263 lines, doctor.go 1826 lines) just to understand where a single field needs to go.
- `pkg/daemonconfig/config.go` already exists with the correct Config struct, started by a prior extraction attempt but never wired as the single source of truth.

---

## Model Impact

- [x] **Extends** model with: config surface area is a measurable daemon operability concern. When adding config fields requires 10+ file touches across files >1000 lines each, agents exhaust context budget on exploration before reaching implementation. Extraction to `pkg/daemonconfig/` (consolidating structs + plist generation) reduces to 5-7 locations representing genuinely different concerns.

---

## Notes

- Full design: `.kb/investigations/2026-02-19-design-extract-daemon-config-package.md`
- Three-phase incremental extraction recommended: (1) Config struct consolidation via type alias, (2) Plist generation consolidation, (3) FromUserConfig conversion function.
- The 5-7 remaining locations post-extraction represent genuinely different concerns (YAML persistence, runtime, CLI flags, plist serialization, API types, UI) that cannot be meaningfully merged without over-engineering (code generation).
