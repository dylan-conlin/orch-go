# Session Synthesis

**Agent:** og-work-mcp-vs-cli-28dec
**Issue:** orch-go-shao
**Duration:** 2025-12-28 ~12:00 → 2025-12-28 ~13:00
**Outcome:** success

---

## TLDR

Investigated whether bd/kb/orch ecosystem tools should use MCP instead of CLI for agent discoverability. Conclusion: CLI is optimal for stateless one-shot tools; MCP only warranted for stateful browser automation (glass). No changes needed - current architecture is correct.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2025-12-28-inv-mcp-vs-cli-orch-ecosystem.md` - Investigation with D.E.K.N. summary and 4 findings

### Files Modified
- None

### Commits
- Investigation file to be committed

---

## Evidence (What Was Observed)

- Skills contain 100+ CLI invocations (`bd comment`, `kb context`, `orch spawn`) - all working reliably
- Prior investigation (2025-12-26): CLI 58 lines vs MCP 694 lines (12x complexity difference)
- Glass has dual interface (MCP + CLI) with capability parity - demonstrates when each is appropriate
- Prior decisions already captured: "MCP for agent-internal use, CLI for orchestrator/scripts/humans"
- CLAUDE.md and skill files effectively surface CLI commands to agents

### Tests Run
```bash
# No runtime tests - this was an architecture investigation
# Evidence was code review and prior investigation analysis
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2025-12-28-inv-mcp-vs-cli-orch-ecosystem.md` - MCP vs CLI decision framework for orch ecosystem

### Decisions Made
- **CLI preferred for bd/kb/orch**: These are stateless one-shot operations. MCP's persistent connection model provides no benefit.
- **MCP only for glass**: Browser automation is stateful (tab focus, multi-step flows). MCP is appropriate here.
- **No unified ecosystem MCP**: Would violate "Compose Over Monolith" principle and add complexity without benefit.

### Constraints Discovered
- MCP adds ~12x complexity per tool exposed (schema definitions, handlers, transport setup)
- "Surfacing Over Browsing" is satisfied by documentation (skills, CLAUDE.md), not protocols

### Pattern Documented
| Tool Type | Interface | Rationale |
|-----------|-----------|-----------|
| Stateless one-shot (bd, kb, orch) | CLI via Bash | Simple, 12x less code, no state needed |
| Stateful interactive (glass) | MCP | Persistent connection, shared state, multi-step flows |
| Validation gates (glass assert) | CLI | Scripts need exit codes, not agent sessions |

### Externalized via `kn`
- Decision to externalize: "CLI for stateless tools, MCP for stateful browser automation"

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (investigation file with D.E.K.N.)
- [x] Tests passing (N/A - design investigation)
- [x] Investigation file has `**Phase:** Complete`
- [ ] Ready for `orch complete orch-go-shao`

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Token cost comparison between MCP tool schemas and CLI invocations (MCP consumes context for schemas)
- Whether future OpenCode versions might prefer MCP over Bash tool

**Areas worth exploring further:**
- None immediate - decision is clear

**What remains unclear:**
- Whether any future capability would require MCP for bd/kb/orch (unlikely given stateless nature)

---

## Session Metadata

**Skill:** design-session
**Model:** opus (Claude)
**Workspace:** `.orch/workspace/og-work-mcp-vs-cli-28dec/`
**Investigation:** `.kb/investigations/2025-12-28-inv-mcp-vs-cli-orch-ecosystem.md`
**Beads:** `bd show orch-go-shao`
