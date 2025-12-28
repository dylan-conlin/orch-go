<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
-->

## Summary (D.E.K.N.)

**Delta:** Spawned agents now automatically receive local project ecosystem context from ~/.orch/ECOSYSTEM.md.

**Evidence:** Tests pass, ecosystem context (624 chars) is generated and included in SPAWN_CONTEXT.md.

**Knowledge:** The Quick Reference table from ECOSYSTEM.md provides a concise registry of local projects, preventing agents from searching GitHub for projects like glass, beads, kb-cli.

**Next:** Close - implementation complete. Supersedes constraint "When spawning research comparing external tools to local projects, explicitly describe what the local project is and where it lives".

---

# Investigation: Auto Inject Local Project Ecosystem

**Question:** How can we automatically inject local project ecosystem context into spawn prompts so agents know about Dylan's local projects (glass, beads, kb-cli, etc.) without GitHub searching?

**Started:** 2025-12-28
**Updated:** 2025-12-28
**Owner:** Agent og-feat-auto-inject-local-28dec
**Phase:** Complete
**Next Step:** None
**Status:** Complete

<!-- Lineage -->
**Supersedes:** Constraint "When spawning research comparing external tools to local projects, explicitly describe what the local project is and where it lives"

---

## Findings

### Finding 1: Ecosystem registry already exists at ~/.orch/ECOSYSTEM.md

**Evidence:** File exists with 299 lines, containing Quick Reference table and detailed repo information.

**Source:** `ls -la ~/.orch/ECOSYSTEM.md` - created 2025-12-22

**Significance:** No need to create new file - just need to read and inject existing content.

---

### Finding 2: SPAWN_CONTEXT.md is generated via Go template in pkg/spawn/context.go

**Evidence:** Template uses contextData struct with fields like KBContext, ServerContext.

**Source:** `pkg/spawn/context.go:17-247` - SpawnContextTemplate and contextData struct

**Significance:** Pattern is clear - add EcosystemContext field and template placeholder.

---

### Finding 3: Quick Reference table is the right abstraction level for spawn context

**Evidence:** Full ECOSYSTEM.md is 299 lines, but Quick Reference table is ~20 lines with key info.

**Source:** `~/.orch/ECOSYSTEM.md` lines 9-21

**Significance:** Extract just the Quick Reference section to keep spawn context concise.

---

## Synthesis

**Key Insights:**

1. **Existing infrastructure** - ~/.orch/ECOSYSTEM.md and the spawn context template system made this straightforward.

2. **Extraction pattern** - extractQuickReference() parses the Quick Reference section, falling back to first 50 lines if not found.

3. **Auto-injection** - GenerateEcosystemContext() is called unconditionally in GenerateContext(), with graceful empty return if file doesn't exist.

**Answer to Investigation Question:**

Implemented auto-injection of ecosystem context by:
1. Adding `EcosystemContext` field to contextData struct
2. Adding `GenerateEcosystemContext()` function to read ~/.orch/ECOSYSTEM.md
3. Adding template placeholder that renders "## LOCAL PROJECT ECOSYSTEM" section
4. Extracting Quick Reference table for concise context (~624 chars)

---

## Structured Uncertainty

**What's tested:**

- ✅ Template parsing works (go build passes)
- ✅ extractQuickReference correctly extracts table (unit test passes)
- ✅ GenerateEcosystemContext returns 624 chars from real file (integration test passes)
- ✅ GenerateContext includes ecosystem section (unit test passes)

**What's untested:**

- ⚠️ Behavior on machines without ~/.orch/ECOSYSTEM.md (returns empty, not tested in CI)
- ⚠️ Agent behavior improvement (need real spawn to verify agents don't GitHub search)

**What would change this:**

- If ECOSYSTEM.md format changes, extractQuickReference may need updates
- If ecosystem content exceeds token budget, may need truncation logic

---

## References

**Files Modified:**
- `pkg/spawn/context.go` - Added EcosystemContext to template, contextData, and GenerateEcosystemContext function
- `pkg/spawn/context_test.go` - Added tests for ecosystem context generation

**Commands Run:**
```bash
# Build verification
go build ./pkg/spawn/...

# Test verification
go test ./pkg/spawn/... -v -run "Ecosystem"

# Install new binary
make install
```

**Related Artifacts:**
- **Ecosystem registry:** ~/.orch/ECOSYSTEM.md - Source of project context

---

## Investigation History

**2025-12-28 11:25:** Investigation started
- Initial question: How to auto-inject local project ecosystem into spawn context
- Context: Agents were searching GitHub for local projects like glass, beads

**2025-12-28 11:27:** Found ~/.orch/ECOSYSTEM.md already exists
- No need to create new file, just read and inject

**2025-12-28 11:35:** Implementation complete
- Added EcosystemContext to template and contextData
- Added GenerateEcosystemContext() and extractQuickReference()
- All tests pass

**2025-12-28 11:45:** Investigation completed
- Status: Complete
- Key outcome: Spawned agents now automatically receive local project ecosystem context
