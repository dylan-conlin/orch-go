## Summary (D.E.K.N.)

**Delta:** Synthesized 16 OpenCode investigations into comprehensive guide at `.kb/guides/opencode.md`.

**Evidence:** Read all 16 investigations (2025-12-19 to 2025-12-26), ran `kb chronicle "opencode"` (282 entries), identified 5 common problems, 6 settled decisions, and documented API patterns.

**Knowledge:** OpenCode is runtime infrastructure (not external tool like beads) with 3,600+ LoC integration; guides provide single authoritative reference vs scattered investigations.

**Next:** Close - guide created, ready for use by future agents.

**Promote to Decision:** no (tactical synthesis, follows existing decision that 10+ investigations should become guides)

---

# Investigation: Synthesize OpenCode Investigations (16)

**Question:** Can 16 scattered OpenCode investigations be consolidated into a single authoritative reference guide?

**Started:** 2026-01-06
**Updated:** 2026-01-06
**Owner:** og-feat-synthesize-opencode-investigations-06jan-0739
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Investigations span 7 days with clear evolution arc

**Evidence:** 16 investigations from 2025-12-19 (POC) to 2025-12-26 (themes) covering:
- POC and client package (Dec 19)
- Integration tradeoffs and spawn modes (Dec 20)
- Disk session cleanup (Dec 21)
- API redirect issues, plugin crashes (Dec 23)
- Ecosystem audit, cleanup fixes (Dec 24)
- Crashes, token refresh, session leaks, themes (Dec 25-26)

**Source:** kb chronicle "opencode" output (282 entries)

**Significance:** Investigations show natural maturation from exploratory (POC) to operational (bugs, cleanup, configuration).

---

### Finding 2: Five recurring problems documented across investigations

**Evidence:** Same issues appear in multiple investigations:
1. `/sessions` vs `/session` endpoint confusion (3 investigations)
2. Session accumulation / cleanup needed (2 investigations)
3. Plugin dependency issues (1 investigation)
4. Token/auth confusion (2 investigations)
5. Active session detection gaps (1 investigation)

**Source:** Pattern analysis across investigation "Common Problems" and "Findings" sections

**Significance:** Guide consolidates these into single reference with consistent fix documentation.

---

### Finding 3: Six key architectural decisions already settled

**Evidence:** Decisions documented across investigations:
1. OpenCode is runtime infrastructure, not external tool
2. Standalone + API Discovery is recommended spawn approach
3. Sessions created via TUI ARE visible via API
4. x-opencode-directory header controls disk vs memory
5. OAuth auto-refresh handled by OpenCode plugin
6. pkg/opencode/ provides right abstraction level

**Source:** "Key Decisions" sections and DKEN summaries in investigations

**Significance:** Future agents should not re-investigate these settled questions.

---

## Synthesis

**Key Insights:**

1. **Investigations followed exploratory → operational arc** - Started with "can we do X?" questions, evolved to "why is X broken?" and "how do we configure X?" questions.

2. **Multiple investigations tackled same underlying issues** - API endpoint confusion, session lifecycle, cleanup procedures appeared repeatedly. Guide consolidates into single reference.

3. **Deep integration requires comprehensive documentation** - 3,600+ LoC across 8 files, 12+ HTTP endpoints, OAuth management - too complex for scattered investigations.

**Answer to Investigation Question:**

Yes, the 16 investigations were successfully consolidated into `.kb/guides/opencode.md` (400+ lines). The guide provides:
- Architecture overview with visual diagram
- API reference (working vs proxied endpoints)
- 5 common problems with causes and fixes
- 6 settled decisions that shouldn't be re-investigated
- Debugging checklist for future issues
- Configuration patterns (instructions, plugins, themes)

---

## Structured Uncertainty

**What's tested:**

- ✅ All 16 investigations read and analyzed (verified: full content review)
- ✅ Guide created with all major sections (verified: file written successfully)
- ✅ kb chronicle confirms topic scope (verified: 282 entries returned)

**What's untested:**

- ⚠️ Guide usefulness in practice (not yet used by another agent)
- ⚠️ Completeness of API reference (based on investigations, not OpenAPI spec)

**What would change this:**

- Future investigation discovers undocumented API endpoint or behavior
- OpenCode version update changes documented behavior

---

## Implementation Recommendations

### Recommended Approach ⭐

**Guide-first consultation** - Future agents should read `.kb/guides/opencode.md` before spawning OpenCode-related investigations.

**Why this approach:**
- Prevents duplicate investigations (6+ decisions already settled)
- Provides quick debugging checklist (6 steps before investigation)
- Documents working API endpoints vs proxied endpoints

**Trade-offs accepted:**
- Guide may become stale as OpenCode evolves
- Some edge cases not covered

### Alternative Approaches Considered

**Option B: Keep investigations scattered**
- **Pros:** No consolidation work needed
- **Cons:** 16 files to search, same issues documented multiple times
- **When to use instead:** If investigations had no overlap

---

## References

**Files Examined:**
- `.kb/investigations/2025-12-19-inv-client-opencode-session-management.md`
- `.kb/investigations/2025-12-19-simple-opencode-poc-spawn-session-via.md`
- `.kb/investigations/2025-12-20-design-explore-tradeoffs-orch-opencode-integration.md`
- `.kb/investigations/2025-12-20-inv-refactor-orch-tail-use-opencode.md`
- `.kb/investigations/2025-12-20-research-opencode-native-context-loading.md`
- `.kb/investigations/2025-12-21-inv-implement-verify-opencode-disk-session.md`
- `.kb/investigations/2025-12-23-debug-opencode-api-redirect-loop.md`
- `.kb/investigations/2025-12-23-inv-oc-command-opencode-dev-wrapper.md`
- `.kb/investigations/2025-12-24-inv-addendum-ecosystem-audit-opencode.md`
- `.kb/investigations/2025-12-24-inv-fix-orch-clean-verify-opencode.md`
- `.kb/investigations/2025-12-25-inv-opencode-crashes-no-user-message.md`
- `.kb/investigations/2025-12-26-inv-can-we-auto-refresh-opencode.md`
- `.kb/investigations/2025-12-26-inv-investigate-opencode-session-accumulation-causing.md`
- `.kb/investigations/2025-12-26-inv-opencode-health-endpoint-returns-redirected.md`
- `.kb/investigations/2025-12-26-inv-opencode-theme-selection-system.md`
- `.kb/investigations/2025-12-26-inv-port-full-opencode-theme-system.md`

**Commands Run:**
```bash
kb chronicle "opencode"
kb create guide "opencode"
```

**Related Artifacts:**
- **Guide:** `.kb/guides/opencode.md` - Created by this investigation
- **Workspace:** `.orch/workspace/og-feat-synthesize-opencode-investigations-06jan-0739/`

---

## Investigation History

**2026-01-06:** Investigation started
- Initial question: Can 16 OpenCode investigations be synthesized into single guide?
- Context: Spawned by orch-go-ucswq to consolidate accumulated knowledge

**2026-01-06:** All 16 investigations read and analyzed
- Identified 5 recurring problems, 6 settled decisions
- Pattern: POC → integration → operational issues arc

**2026-01-06:** Investigation completed
- Status: Complete
- Key outcome: Created `.kb/guides/opencode.md` (400+ lines) as single authoritative reference
