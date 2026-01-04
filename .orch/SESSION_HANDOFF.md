# Session Handoff - Jan 4, 2026 (Evening)

## What Happened This Session

**Focus:** Discovered and addressed a systemic failure mode - individual agents making locally-correct patches that accumulate into globally-incoherent systems.

### Key Accomplishments

1. **Hotspot Detection System (Epic yz3d - CLOSED)**
   - `orch hotspot` command - surfaces files with high fix-commit density and investigation clusters
   - Pre-spawn warnings - alerts when spawning to hotspot areas
   - Daemon preview integration - flags hotspot issues before auto-spawning
   - Dashboard UI indicator - fire emoji on agents working in hotspot areas
   - Added `--exclude` flag with defaults for data files (*.jsonl, *.json, *.lock, go.sum)

2. **Priority Cascade Model for Dashboard Status**
   - Replaced 10+ scattered conditions with single `determineAgentStatus()` function
   - Priority order: Beads Closed > Phase Complete > SYNTHESIS.md > Session Activity
   - Fixed idle/untracked agents incorrectly showing as "active"

3. **Artifact Taxonomy Clarification**
   - Confirmed 4-type model: Investigation, Decision, Guide, Quick
   - RESEARCH.md and KNOWLEDGE.md deprecated (zero usage)
   - Guides ARE the synthesis output for investigation clusters
   - Updated kb reflect to suggest "Action: kb create guide" instead of "kb chronicle"

4. **Constraint Captured**
   - `kn-9865a3`: "High patch density signals missing coherent model - spawn architect before more patches"

### Files Changed

```
# Hotspot detection
cmd/orch/hotspot.go (new)
cmd/orch/hotspot_test.go (new)
cmd/orch/daemon.go (hotspot integration)
cmd/orch/serve.go (API endpoint)
web/src/lib/stores/hotspot.ts (new)
web/src/routes/+page.svelte (fire emoji indicator)

# Priority Cascade
cmd/orch/serve_agents.go (determineAgentStatus function)

# kb-cli (separate repo)
~/.kb/templates/RESEARCH.md (deprecation notice)
~/.kb/templates/KNOWLEDGE.md (deprecation notice)
cmd/kb/reflect.go (synthesis suggestion text)
```

## Ready Work

### Epic: Split cmd/orch/main.go (orch-go-uf4u) - P1

main.go has 49 fix commits in 28 days. Architect designed 4-phase plan:

| Phase | Issue | Task | Lines |
|-------|-------|------|-------|
| 1 | orch-go-uf4u.6 | Extract complete_cmd.go | ~400 |
| 2 | orch-go-uf4u.7 | Extract clean_cmd.go | ~350 |
| 3 | orch-go-uf4u.8 | Extract account_cmd.go + port_cmd.go | ~430 |
| 4 | orch-go-uf4u.9 | Extract small commands | ~510 |

Design: `.kb/investigations/2026-01-04-inv-cmd-orch-main-go-49.md`

### Other Open Issues

- `kb-cli-xrm`: kb reflect should check for existing guides before flagging synthesis opportunities

## Stale Agents (Ignore)

4 stale entries showing in `orch status` - sessions died but entries persist. Will clear eventually.

## Start Next Session With

```bash
orch status
bd show orch-go-uf4u  # Review main.go split epic
orch spawn feature-impl "Extract complete_cmd.go from main.go" --issue orch-go-uf4u.6 --phases implementation,validation
```

## Key Insight

The hotspot detection system exists because we kept patching dashboard status logic for weeks without stepping back to design a coherent model. Now the system can detect this pattern (5+ fix commits in 28 days) and recommend architect before complexity compounds. This is the meta-improvement that prevents the class of problems we were experiencing.
