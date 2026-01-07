## Summary (D.E.K.N.)

**Delta:** Synthesized 16 CLI investigations (Dec 19 - Jan 4) into single authoritative guide at `.kb/guides/cli.md`.

**Evidence:** Read all 16 investigations, identified 7 distinct categories, found 2 duplicates, created consolidated reference covering identity, commands, patterns, debugging.

**Knowledge:** CLI evolution is stable - "kubectl for AI agents" identity from day one. Most investigations are historical implementation records, not evolving knowledge. Key active knowledge: binary staleness causes silent SIGKILL failures.

**Next:** Close - guide created, investigations categorized, duplicates identified.

---

# Investigation: Synthesize CLI Investigations (16)

**Question:** Can 16 CLI investigations be consolidated into a single authoritative reference?

**Started:** 2026-01-06
**Updated:** 2026-01-06
**Owner:** Feature Implementation Agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Investigations fall into 7 distinct categories

**Evidence:** Categorizing the 16 investigations:

| Category | Count | Investigations |
|----------|-------|----------------|
| Initial implementation | 4 | spawn, status, complete commands + scaffolding |
| Feature addition | 2 | README updates, focus/drift/next commands |
| Evolution/comparison | 2 | Python vs Go, trace evolution |
| Bug fixes | 2 | Stale binary issues (DUPLICATE) |
| Auto-detection | 2 | CLI command detection (partial DUPLICATE) |
| Integration | 2 | snap CLI, glass CLI |
| Recent enhancements | 2 | Command recovery, hotspot |

**Source:** All 16 investigation files in `.kb/investigations/`

**Significance:** Most investigations are point-in-time implementation records. Only a few contain evolving knowledge (binary staleness, tool selection patterns).

---

### Finding 2: Two sets of duplicate/superseding investigations found

**Evidence:**

**Duplicate 1 - Stale binary issue:**
- `2025-12-23-inv-cli-output-not-appearing-orch.md` 
- `2025-12-23-inv-cli-output-not-appearing.md`
Both investigate same SIGKILL exit 137 issue with stale binaries. Same root cause, same fix.

**Duplicate 2 - Auto-detect feature:**
- `2025-12-26-inv-auto-detect-cli-commands-needing.md` - Found feature already exists
- `2025-12-26-inv-auto-detect-new-cli-commands.md` - Implementation details
Second supersedes first (feature was already implemented when first investigation started).

**Source:** Reading both investigation pairs

**Significance:** These can be merged or one marked as superseded. The guide consolidates the essential knowledge from both.

---

### Finding 3: Core CLI identity has been stable since inception

**Evidence:** From `2025-12-21-inv-trace-evolution-orch-cli-python.md`:

> "The README captured the essence: **'kubectl for AI agents'** - a command-line tool for managing AI coding agents"

This identity was established Nov 29, 2025 and remained stable through:
- 575 Python commits
- 218 Go commits (in first 3 days)
- 793+ total commits

**Source:** Evolution investigation, git history analysis

**Significance:** CLI investigations don't represent architectural drift - the core model is stable. Guide can confidently state the identity without hedging.

---

### Finding 4: Binary staleness is the most actionable recurring issue

**Evidence:** Two investigations (Dec 23) found same pattern:
- Stale `./orch` binary vs updated `~/bin/orch`
- macOS kills stale binary with SIGKILL (exit 137) - NO ERROR OUTPUT
- Symptoms: missing commands, silent failures
- Fix: rebuild and reinstall

This is operational knowledge that future agents/users will need repeatedly.

**Source:** Both Dec 23 bug fix investigations

**Significance:** This is the primary "active" knowledge worth preserving - unlike implementation investigations which are historical.

---

### Finding 5: Existing guides cover spawn and lifecycle deeply

**Evidence:** Found related guides already exist:
- `.kb/guides/spawn.md` - Detailed spawn workflow
- `.kb/guides/agent-lifecycle.md` - Spawn → work → complete flow
- `.kb/guides/daemon.md` - Daemon operations
- `docs/cli/orch-go_*.md` - Auto-generated command reference

**Source:** Guide directory listing

**Significance:** New CLI guide should reference these, not duplicate them. Focus on CLI-specific knowledge (binary management, command overview, debugging patterns).

---

## Synthesis

**Key Insights:**

1. **Historical vs Active knowledge** - Most CLI investigations are implementation records (historical reference). Only binary staleness and tool selection patterns are actively useful for future work.

2. **No architectural drift** - CLI identity ("kubectl for AI agents") has been stable from day one. Safe to create authoritative reference without tracking evolving positions.

3. **Guide-worthy knowledge is sparse** - After consolidation, the essential CLI knowledge fits in ~200 lines. The 16 investigations contain ~2500 lines total, but most is implementation detail.

**Answer to Investigation Question:**

Yes, the 16 CLI investigations consolidate well into a single guide. Created `.kb/guides/cli.md` covering:
- Identity and command categories
- Spawn modes and model selection  
- Binary management (the key operational knowledge)
- Command structure for adding new commands
- Common issues and debugging checklist
- Links to related detailed guides

The investigations are now historical reference. Future CLI questions should start with the guide, not spawn new investigations.

---

## Structured Uncertainty

**What's tested:**

- ✅ All 16 investigations read and categorized (verified: summarized each)
- ✅ Guide created with essential knowledge (verified: file written)
- ✅ No conflicts with existing guides (verified: cross-referenced spawn.md, agent-lifecycle.md)

**What's untested:**

- ⚠️ Whether guide is sufficient for all CLI debugging scenarios
- ⚠️ Whether any investigation contains knowledge not captured in guide

**What would change this:**

- Finding would be wrong if future CLI issue isn't addressable via guide
- Guide may need updates as CLI evolves

---

## Implementation Recommendations

### Recommended Approach ⭐

**Use guide as single reference, mark investigations as superseded** 

**Why this approach:**
- Single authoritative source reduces confusion
- Prevents agents from re-investigating solved problems
- Guide is maintained, investigations are static

**Trade-offs accepted:**
- Historical detail in investigations may be lost (acceptable - can always read originals)
- Guide needs maintenance as CLI evolves

**Implementation sequence:**
1. ✅ Created `.kb/guides/cli.md` with consolidated knowledge
2. Document supersession in this synthesis
3. Future: Consider archiving or marking old investigations

---

## References

**Files Examined:**
- All 16 CLI investigations (paths listed in spawn context)
- `.kb/guides/spawn.md` - Related spawn guide
- `.kb/guides/agent-lifecycle.md` - Related lifecycle guide
- `docs/cli/orch-go_*.md` - Auto-generated command docs

**Commands Run:**
```bash
# Create investigation
kb create investigation synthesize-cli-investigations-16-synthesis

# List existing guides
ls .kb/guides/
```

**Related Artifacts:**
- **Guide:** `.kb/guides/cli.md` - Created by this investigation
- **Guide:** `.kb/guides/spawn.md` - Related, detailed spawn workflow
- **Guide:** `.kb/guides/agent-lifecycle.md` - Related, lifecycle flow

---

## Investigation History

**2026-01-06 10:XX:** Investigation started
- Initial question: Can 16 CLI investigations be consolidated?
- Context: Accumulated investigations trigger synthesis threshold (10+)

**2026-01-06 11:XX:** Analysis complete
- Identified 7 categories
- Found 2 duplicate pairs
- Identified key active knowledge (binary staleness)

**2026-01-06 11:XX:** Guide created
- Created `.kb/guides/cli.md` (~200 lines)
- Consolidated essential knowledge from 16 investigations

**2026-01-06 11:XX:** Investigation completed
- Status: Complete
- Key outcome: Single CLI guide replaces 16 scattered investigations
