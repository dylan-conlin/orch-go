# Probe: kb-cli Public Release Readiness — Standalone Substrate Audit

**Model:** Knowledge Physics
**Date:** 2026-03-09
**Status:** Complete

---

## Question

The minimal-substrate probe identified kb-cli as a core component of the investigation/probe/model cycle's minimal substrate. This probe tests: **Is kb-cli actually standalone-ready for a solo developer with Claude Code + Git, or does orch-go coupling make the "minimal substrate" claim aspirational rather than actual?**

Specifically testing:
1. Model claim: "kb CLI is part of the minimal substrate" — does kb-cli actually function without orch, beads, skillc, or daemon?
2. Model claim: "The context injection gap — kb context returns paths but not extracted model sections — is the single tooling gap" — is --extract-models actually the only gap, or are there more?
3. Extending: What is the full inventory of orch-go coupling in kb-cli, and what must change for v0.1 public release?

---

## What I Tested

### Test 1: End-to-end standalone command verification

Tested each core command for orch dependencies:

| Command | Standalone? | Coupling | Evidence |
|---------|------------|----------|----------|
| `kb init` | YES | None | Creates .kb/ with investigations/, decisions/, guides/. No external deps. |
| `kb create investigation` | YES | None | Creates dated markdown files. --model and --orphan flags work. |
| `kb create decision` | YES | None | Creates decision records. |
| `kb list investigations` | YES | None | Lists .kb/investigations/*.md files. |
| `kb search "query"` | YES | None | Stemming-based search, relevance scoring, works on local .kb/. |
| `kb context "query"` | PARTIAL | `~/.orch/groups.yaml` for --siblings/--global | Local search works. Cross-project search reads ~/.orch/groups.yaml. Falls back gracefully (no error) when missing. |
| `kb reflect` | PARTIAL | beads (bd CLI) for --create-issue, dedup checks | Core reflection patterns work (synthesis, stale, drift, open). Issue creation shells out to `bd create`. Warns but continues if bd unavailable. |
| `kb ask "question"` | NO | opencode binary required | Shells out to `opencode run`. No fallback. Fails hard if opencode not installed. Error message tells user to install opencode-ai from npm. |
| `kb link` | NO | beads (bd CLI) required | Links artifacts to beads issue IDs. No standalone use case. |
| `kb learn` | PARTIAL | beads references in code | Some functionality references beads. |

### Test 2: Build & test health

```bash
cd ~/Documents/personal/kb-cli && go test ./... -count=1
```

**Results:**
- **Coverage:** 57.6% cmd/kb, 98.5% internal/search
- **3 failing tests:**
  1. `TestCreateGuide` — expects Lineage section in guide template, template doesn't include it
  2. `TestTruncateSummary` — sentence-boundary truncation algorithm differs from test expectations
  3. `TestFindOpenCandidatesFiltersByAge` — investigation age 98 days, expected ≤3 (date parsing issue)
- **No go vet errors**
- **Dead file:** `reflect_test.go.bak` exists in cmd/kb/

### Test 3: Dependency audit

```
go.mod dependencies (direct):
- github.com/kljensen/snowball v0.10.0  (stemming - lightweight, stable)
- github.com/spf13/cobra v1.10.1       (CLI framework - industry standard)
- gopkg.in/yaml.v3 v3.0.1              (YAML parsing - standard)
```

Excellent dependency hygiene. 3 direct deps, all stable, no security concerns.

### Test 4: Orch coupling inventory

Searched all .go files for orch-specific references:

| File | Coupling Type | What It Does | Severity |
|------|--------------|-------------|----------|
| `beads.go` | Hard | findBdPath() with hardcoded `~/Documents/personal/beads/build/bd` | HIGH — exposes dev path |
| `ask.go` | Hard | Shells out to `opencode` binary, error says "install opencode-ai from npm" | HIGH — wrong install advice for public |
| `groups.go` | Soft | DefaultGroupsPath() → `~/.orch/groups.yaml` | MEDIUM — orch-specific config path |
| `reflect.go` | Soft | runBdCommand() for --create-issue dedup, issue creation | MEDIUM — feature degrades gracefully |
| `context.go` | Soft | Uses groups.yaml via --siblings/--global flags | LOW — graceful fallback |
| `link.go` | Hard | Links artifacts to beads issues (requires bd) | MEDIUM — no standalone use case |
| `chronicle.go` | Soft | References beads issue types in output | LOW — cosmetic |
| `learn.go` | Soft | Some beads references | LOW |
| `migrate.go` | Soft | Migrates from .orch/ to .kb/ | LOW — helpful legacy support |

### Test 5: Missing pieces for v0.1

| Item | Status | Notes |
|------|--------|-------|
| LICENSE file | MISSING | README says MIT but no LICENSE file exists |
| README.md | INCOMPLETE | Doesn't mention context, reflect, ask, quick, models, guides, probes |
| CI/CD | MISSING | No .github/workflows/ directory |
| GoReleaser | MISSING | No .goreleaser.yml |
| CLAUDE.md | TEMPLATE ONLY | References .orch/CLAUDE.md, has no actual project context |
| Installation | PARTIAL | go install works but module path is github.com/dylanconlin/kb-cli |
| .gitignore | OK | Ignores /kb binary and .orch/ |

### Test 6: kb context --extract-models gap verification

Confirmed in orch-go's `pkg/spawn/kbcontext.go:1183`:
- `extractModelSectionsForSpawn()` reads model.md files and extracts:
  - Summary section
  - Critical Invariants section
  - Why This Fails section
- Truncates each to 2,500 chars
- Formats for injection into SPAWN_CONTEXT.md

`kb context` returns model file paths and keyword match lines but NOT section content. An agent must then Read each model file manually (2-4 extra tool calls per model).

### Test 7: Files without test coverage

| File | Lines | Risk |
|------|-------|------|
| `quick.go` | 1,027 | HIGH — core feature, no tests |
| `autorebuild.go` | 159 | LOW — internal dev tooling |
| `domain.go` | 71 | LOW — small helper |
| `actioned.go` | 25 | LOW — trivial |
| `main.go` | 45 | LOW — entry point |

---

## What I Observed

### The "Minimal Substrate" Claim Is Conditionally True

The model claims kb-cli is part of the minimal substrate for the investigation/probe/model cycle. This is **confirmed** for the core cycle (init → create → search → context → create probe → edit model), but **the CLI contains significant orch-go coupling** that would confuse or break things for standalone users.

### The Coupling Is Layered, Not Binary

Three tiers of coupling severity:

1. **Hard coupling (breaks for standalone users):**
   - `kb ask` requires opencode — and the error message says "install opencode-ai from npm" which is wrong for public users (that's Dylan's fork)
   - `beads.go` has hardcoded dev path: `~/Documents/personal/beads/build/bd`
   - `kb link` requires beads

2. **Soft coupling (degrades gracefully):**
   - `kb context --siblings/--global` needs `~/.orch/groups.yaml` — but falls back silently
   - `kb reflect --create-issue` needs `bd` — but warns and continues without it
   - Groups system references `~/.orch/` path

3. **No coupling (fully standalone):**
   - init, create, search, list, templates, archive, projects, quick, version, index
   - These are the core cycle commands and they work perfectly

### --extract-models Is NOT the Only Gap

The minimal-substrate probe identified `--extract-models` as "the single tooling gap." This audit finds **additional gaps**:

1. **`kb ask` is broken standalone** — requires opencode, not just a convenience gap
2. **No CLAUDE.md guidance** — standalone users won't know about the investigation/probe/model cycle
3. **No LICENSE file** — legally prevents public distribution
4. **README doesn't describe the core workflow** — context, reflect, models, probes all undocumented
5. **3 failing tests** — signals code isn't in release-ready state
6. **groups.yaml path is orch-specific** — should be `~/.kb/groups.yaml` for standalone

### Recommended v0.1 Scope

**Must fix (blocking release):**
1. Add MIT LICENSE file
2. Fix 3 failing tests
3. Remove hardcoded dev path from beads.go
4. Rewrite README with full command reference and workflow guide
5. Write real CLAUDE.md describing kb-cli for Claude Code users
6. Make `kb ask` gracefully fail or use a configurable LLM backend (not hardcoded opencode)
7. Remove/clean dead file (reflect_test.go.bak)

**Should fix (quality):**
8. Add `kb context --extract-models` flag (~100 lines)
9. Move groups.yaml path from `~/.orch/` to `~/.kb/` with fallback
10. Add tests for quick.go (1,027 untested lines)
11. Add GitHub Actions CI (go test, go vet, golangci-lint)
12. Add .goreleaser.yml for binary distribution

**Can defer:**
13. `kb ask` — could be removed entirely for v0.1 (LLM integration is optional)
14. `kb link` — beads-specific, could be behind feature flag
15. `kb reflect --create-issue` — works without beads, issue creation is bonus

---

## Model Impact

- [x] **Confirms** "kb CLI is part of the minimal substrate" — core cycle commands (init, create, search, context) work standalone
- [x] **Contradicts** "the context injection gap is the single tooling gap" — there are 6 additional gaps (ask broken, no LICENSE, README incomplete, CLAUDE.md empty, failing tests, orch-specific paths)
- [x] **Extends** model with: kb-cli coupling exists in three tiers (hard/soft/none), and the hard-coupled commands (ask, link) should be behind feature gates or removed for standalone use
- [x] **Extends** model with: The minimal substrate is achievable with ~7 focused changes. The core knowledge cycle (init → create → search → context) is already fully standalone.
- [x] **Extends** model with: groups.yaml at `~/.orch/groups.yaml` is an orch-specific path that should move to `~/.kb/groups.yaml` for the standalone substrate to have its own identity

---

## Notes

### Code Quality Assessment

The codebase is well-engineered for an internal tool:
- 3 direct dependencies (excellent hygiene)
- 57.6% test coverage with good test patterns
- Clean separation of testable functions from CLI wiring
- Intelligent search with stemming, relevance scoring, recency weighting
- 30k+ lines across 59 Go files — substantial but well-organized

The primary issue is that it was built as an internal tool for the orch ecosystem and has coupling assumptions baked in at the infrastructure level (bd CLI, opencode, ~/.orch/ paths).

### File Size Concerns

- `reflect.go` at 2,923 lines is approaching the 1,500-line extraction threshold from orch-go's accretion rules
- `create.go` at 1,724 lines already exceeds it
- For public release, these should be noted but don't block v0.1
