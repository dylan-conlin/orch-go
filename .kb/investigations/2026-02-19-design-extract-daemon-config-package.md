# Design: Extract Daemon Config into Dedicated Package

**Date:** 2026-02-19
**Phase:** Complete
**Status:** Complete
**Type:** Architect
**Issue:** orch-go-1092
**Trigger:** Adding a single boolean (`reflect_open`) caused agent ek0b to spiral at 526K tokens because config surface area spans 10-12 files.

---

## Design Question

How should we extract daemon config into a focused package so that adding a new config key requires fewer file touches?

## Problem Framing

### Success Criteria

- Adding a new daemon config boolean touches **6 or fewer** code locations (down from 10-12)
- No duplicate Config structs in the codebase
- No duplicate plist templates
- Incremental migration possible (can be done in 2-3 PRs, not a big-bang rewrite)
- Zero behavioral changes to existing config

### Constraints

- Must preserve `~/.orch/config.yaml` format (no user-facing config migration)
- Must preserve `daemon.Config` API used by `pkg/daemon/daemon.go` consumer code
- Must not break existing tests
- Accretion boundary: files >1500 lines require extraction before feature additions

### Scope

**In:** Struct consolidation, plist generation consolidation, migration path
**Out:** Config-as-data/codegen (future optimization), new config keys, behavioral changes

---

## Current Surface Area (The Problem)

When adding a single boolean to daemon config, you must touch:

| # | File | What changes | Purpose |
|---|------|-------------|---------|
| 1 | `pkg/daemon/daemon.go` | `Config` struct field + `DefaultConfig()` | Runtime config |
| 2 | `pkg/daemonconfig/config.go` | IDENTICAL `Config` struct + `DefaultConfig()` | **Exact duplicate** of #1 |
| 3 | `pkg/userconfig/userconfig.go` | `DaemonConfig` struct (YAML) + accessor method | Persistence layer |
| 4 | `cmd/orch/daemon.go` | Flag variable + `init()` registration + `daemonConfigFromFlags()` | CLI flags |
| 5 | `cmd/orch/config_cmd.go` | `PlistData` struct + `plistTemplate` + `buildPlistData()` | Plist generation |
| 6 | `cmd/orch/serve_system.go` | `PlistDataAPI` + `plistTemplateAPI` + `buildPlistDataForAPI()` | **Copy-paste of #5** |
| 7 | `cmd/orch/serve_system.go` | `DaemonConfigAPIResponse` + `DaemonConfigUpdateRequest` + handlers | API layer |
| 8 | `cmd/orch/doctor.go` | `parsePlistValues()` + `checkPlistDrift()` comparison table | Drift detection |
| 9 | `web/src/lib/stores/daemonConfig.ts` | TypeScript interfaces + store | Frontend types |
| 10 | `web/src/lib/components/daemon-config-panel/` | Svelte form bindings | Frontend UI |
| 11 | `pkg/userconfig/userconfig_test.go` | Accessor tests | Tests |
| 12 | `cmd/orch/doctor_test.go` | Plist parsing tests | Tests |

**Root causes of duplication:**

1. **Three identical Config structs:** `pkg/daemon/daemon.go:Config`, `pkg/daemonconfig/config.go:Config`, and the runtime-equivalent fields in `pkg/userconfig/userconfig.go:DaemonConfig`
2. **Two copy-pasted plist templates:** `config_cmd.go:plistTemplate` and `serve_system.go:plistTemplateAPI` (with comments acknowledging the copy)
3. **Two copy-pasted PlistData structs:** `config_cmd.go:PlistData` and `serve_system.go:PlistDataAPI`

---

## Exploration: Decision Forks

### Fork 1: Where does the single Config struct live?

**Options:**
- A: `pkg/daemonconfig/config.go` (already exists, partially populated)
- B: `pkg/daemon/config.go` (alongside daemon.go)
- C: New `pkg/daemon/config/` sub-package

**Substrate says:**
- Code Extraction Patterns guide: "shared utilities FIRST, then domain files"
- The existing `pkg/daemonconfig/` was started by a prior agent — it already has the Config struct

**RECOMMENDATION:** Option A — `pkg/daemonconfig/config.go`

**Reasoning:** It already exists and has the correct Config struct. `pkg/daemon/daemon.go` imports it. Having config in a separate package from `pkg/daemon/` is actually good — it prevents import cycles (config_cmd.go can import daemonconfig without importing all of daemon).

**Trade-off accepted:** Slightly longer import path (`daemonconfig.Config` vs `daemon.Config`). But `daemon.Config` can be a type alias for the transition.

### Fork 2: Where does plist generation live?

**Options:**
- A: `pkg/daemonconfig/plist.go` — alongside the config it describes
- B: `pkg/plist/` — dedicated package for plist generation
- C: Keep in `cmd/orch/` but consolidate to one file

**Substrate says:**
- "No local agent state" constraint (CLAUDE.md) — but plist gen is not state, it's config transformation
- Accretion boundaries principle — config_cmd.go and serve_system.go are both large

**RECOMMENDATION:** Option A — `pkg/daemonconfig/plist.go`

**Reasoning:** The plist is a _serialization of daemon config_. It belongs with the config, not with CLI commands. Both `config_cmd.go` and `serve_system.go` import from here, eliminating the copy.

**Trade-off accepted:** Plist generation moves from cmd/ to pkg/, but this is appropriate since it's pure data transformation.

### Fork 3: How does userconfig.DaemonConfig relate to daemonconfig.Config?

**Options:**
- A: **Keep both** — userconfig.DaemonConfig stays as YAML layer, daemonconfig.Config stays as runtime layer, with explicit conversion function
- B: **Merge** — Use daemonconfig.Config with YAML tags directly
- C: **Generate** — Code-generate one from the other

**Substrate says:**
- The two structs serve genuinely different purposes: YAML uses `*bool` pointer types for "was this explicitly set?" semantics. Runtime uses flat `bool` with defaults applied.

**RECOMMENDATION:** Option A — Keep both with explicit conversion

**Reasoning:** The YAML layer (`*bool` pointer types, `yaml:"..."` tags) and runtime layer (`bool`/`time.Duration`, no tags) serve different concerns. Merging them would lose the opt-in/explicit-set tracking. But we add a conversion function `ToRuntimeConfig()` in daemonconfig that maps userconfig → daemonconfig.Config, centralizing the default-application logic.

**Trade-off accepted:** Two config struct representations remain, but with a clear purpose distinction and a single conversion point (instead of scattered accessor methods).

### Fork 4: How to consolidate CLI flag → Config mapping?

**Options:**
- A: Move flag definition + mapping into `pkg/daemonconfig/flags.go`
- B: Keep flags in `cmd/orch/daemon.go` but simplify mapping
- C: Use cobra flag binding directly to struct

**RECOMMENDATION:** Option B — Keep flags in cmd/orch, but conversion uses `daemonconfig.FromUserConfig()` with CLI overrides

**Reasoning:** CLI flag registration is inherently a cmd/ concern (it depends on cobra). But the _mapping_ from flags to Config can be simplified: `daemonConfigFromFlags()` calls `daemonconfig.FromUserConfig()` first, then applies only the CLI overrides.

---

## Synthesis: Recommended Architecture

### After Extraction

```
pkg/daemonconfig/           # Single source of truth for daemon config
├── config.go               # Config struct + DefaultConfig()
├── plist.go                # PlistData + template + BuildPlistData()
└── convert.go              # FromUserConfig() - userconfig.DaemonConfig → Config

pkg/userconfig/
└── userconfig.go           # DaemonConfig (YAML) unchanged, but Daemon* accessor methods removed
                            # (replaced by daemonconfig.FromUserConfig)

pkg/daemon/
└── daemon.go               # Uses daemonconfig.Config (delete local Config + DefaultConfig)

cmd/orch/
├── daemon.go               # CLI flags stay here, daemonConfigFromFlags() uses daemonconfig
├── config_cmd.go           # Uses daemonconfig.BuildPlistData() (delete local PlistData/template)
├── serve_system.go         # Uses daemonconfig.BuildPlistData() (delete PlistDataAPI/templateAPI)
└── doctor.go               # Drift detection uses daemonconfig for comparisons
```

### What Adding a New Boolean Looks Like After

| # | File | What changes |
|---|------|-------------|
| 1 | `pkg/daemonconfig/config.go` | Struct field + default |
| 2 | `pkg/userconfig/userconfig.go` | YAML struct field (DaemonConfig) |
| 3 | `pkg/daemonconfig/convert.go` | Mapping in `FromUserConfig()` |
| 4 | `cmd/orch/daemon.go` | CLI flag + override in `daemonConfigFromFlags()` |
| 5 | `pkg/daemonconfig/plist.go` | PlistData field + template line (if in plist) |
| 6 | `cmd/orch/serve_system.go` | API types + handler (if in API) |
| 7 | Web frontend (if in dashboard) |

**Result: 5-7 locations (down from 10-12), zero duplicates**

The remaining locations represent genuinely different concerns (YAML, runtime, CLI, plist, API, UI) that cannot be meaningfully merged.

---

## Implementation Phases

### Phase 1: Consolidate Config struct (1 PR)

**Scope:** Eliminate `pkg/daemon/daemon.go`'s Config + DefaultConfig, use `pkg/daemonconfig/`

1. Verify `pkg/daemonconfig/config.go` Config matches `pkg/daemon/daemon.go` Config (it does — they're identical)
2. In `pkg/daemon/daemon.go`: change `Config` to `type Config = daemonconfig.Config` (type alias)
3. Change `DefaultConfig()` to delegate: `func DefaultConfig() Config { return daemonconfig.DefaultConfig() }`
4. Update imports across callers
5. Verify: `go build ./...` + `go test ./...`

**Risk:** Low. Type alias is backward-compatible.
**Reversibility:** Trivially reversible by removing the alias.

### Phase 2: Consolidate plist generation (1 PR)

**Scope:** Single `pkg/daemonconfig/plist.go`, delete duplicates in config_cmd.go and serve_system.go

1. Create `pkg/daemonconfig/plist.go` with:
   - `PlistData` struct
   - `PlistTemplate` const
   - `BuildPlistData(cfg *userconfig.Config) (*PlistData, error)`
   - `GeneratePlist(cfg *userconfig.Config) ([]byte, error)`
2. Update `config_cmd.go`: replace `buildPlistData` call with `daemonconfig.BuildPlistData`
3. Update `serve_system.go`: replace `buildPlistDataForAPI` call with `daemonconfig.BuildPlistData`
4. Delete: `PlistData`, `PlistDataAPI`, `plistTemplate`, `plistTemplateAPI`, `buildPlistData`, `buildPlistDataForAPI`, `findOrchPath`, `findOrchPathForAPI`
5. Verify: `go build ./...` + `go test ./...`

**Risk:** Low-medium. Function signatures change but behavior is identical.
**Reversibility:** Standard revert.

### Phase 3: Add FromUserConfig conversion (1 PR)

**Scope:** Centralize the userconfig → daemonconfig.Config mapping

1. Create `pkg/daemonconfig/convert.go`:
   ```go
   func FromUserConfig(cfg *userconfig.Config) Config {
       return Config{
           PollInterval: time.Duration(cfg.DaemonPollInterval()) * time.Second,
           MaxAgents:    cfg.DaemonMaxAgents(),
           // ... all fields
       }
   }
   ```
2. Update `daemon.go:daemonConfigFromFlags()` to use `daemonconfig.FromUserConfig()` as the base, then apply CLI overrides
3. Optionally: remove individual `Daemon*()` accessor methods from userconfig.go (they become dead code)
4. Verify tests

**Risk:** Medium. Changes the config resolution flow.
**Reversibility:** Standard revert.

---

## Recommendations

⭐ **RECOMMENDED:** Three-phase incremental extraction (Phases 1 → 2 → 3)

- **Why:** Eliminates all duplicates, reduces per-field cost from 10-12 to 5-7 locations
- **Trade-off:** Still 5-7 locations per field, but these represent genuinely different concerns
- **Expected outcome:** The next `reflect_open`-style boolean takes 30 minutes, not a 526K token spiral

**Alternative: Config-as-data with code generation**
- **Pros:** Could reduce to 1-2 locations (declare field once, generate everything)
- **Cons:** Go code generation is complex, adds build tooling dependency, over-engineering for current config size (~25 fields)
- **When to choose:** If daemon config grows past ~40 fields or new config subsystems emerge

---

## Decision Gate Guidance (if promoting to decision)

**Add `blocks:` frontmatter when:**
- This addresses a recurring config accretion problem documented in this investigation

**Suggested blocks keywords:**
- "daemon config", "add daemon setting", "new daemon boolean", "plist generation"
