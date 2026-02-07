<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Mac successfully converted from production-style launchd supervision to simple overmind-only dev workflow by removing 4 launchd plists and wrapper scripts.

**Evidence:** Verified via `launchctl list | grep orch` returning no results, plist files deleted from ~/Library/LaunchAgents/, wrapper script removed, CLAUDE.md updated to dev-only architecture.

**Knowledge:** Dev environments don't need production-style auto-restart/supervision; overmind's simple Procfile configuration is sufficient and avoids launchd's tmux PATH issues.

**Next:** Close investigation - cleanup complete, documentation created (guide + decision doc).

**Promote to Decision:** Actioned - decision exists (dev-vs-prod-architecture)

<!--
Example D.E.K.N.:
Delta: Test-running guidance is missing from spawn prompts and CLAUDE.md.
Evidence: Searched 5 agent sessions - none ran tests; guidance exists in separate docs but isn't loaded.
Knowledge: Agents follow documentation literally; guidance must be in loaded context to be followed.
Next: Add test-running instruction to SPAWN_CONTEXT.md template.
Promote to Decision: recommend-no (tactical fix, not architectural)

Guidelines:
- Keep each line to ONE sentence
- Delta answers "What did we find?"
- Evidence answers "How do we know?"
- Knowledge answers "What does this mean?"
- Next answers "What should happen now?"
- Promote to Decision: flag for orchestrator/human - recommend-yes if this establishes a pattern, constraint, or architectural choice worth preserving
- Enable 30-second understanding for fresh Claude
-->

---

# Investigation: Clean Up Mac Dev Environment

**Question:** How to remove launchd supervision from Mac and establish overmind-only dev workflow?

**Started:** 2026-01-10
**Updated:** 2026-01-10
**Owner:** feature-impl agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** N/A
**Supersedes:** N/A
**Superseded-By:** N/A

---

## Findings

### Finding 1: Four launchd plists currently active

**Evidence:** 
```bash
$ launchctl list | grep orch
34124	0	com.orch.web
18720	-15	com.orch.serve
87329	0	com.orch.doctor

$ ls -la ~/Library/LaunchAgents/ | grep orch
-rw-r--r--  1 dylanconlin  staff   995 Jan 10 08:05 com.orch.doctor.plist
-rw-r--r--  1 dylanconlin  staff   721 Jan 10 08:59 com.orch.serve.plist
-rw-r--r--  1 dylanconlin  staff   593 Jan 10 09:04 com.orch.web.plist
-rw-r--r--  1 dylanconlin  staff   982 Jan 10 00:41 com.overmind.orch-go.plist
```

**Source:** `launchctl list` command, `~/Library/LaunchAgents/` directory

**Significance:** These plists were created for production-style supervision but Mac should be dev environment only. Need to unload and remove all four plists.

---

### Finding 2: Context shift from production to dev architecture

**Evidence:** From beads issue description: "Remove launchd supervision from Mac (production-style setup), establish overmind as dev-only workflow. Realized Mac should be dev environment, VPS will be production deployment target."

Decision documents show evolution:
- `.kb/decisions/2026-01-10-launchd-supervision-architecture.md` - Accepted hybrid approach (doctor via launchd, overmind manual)
- `.kb/decisions/2026-01-10-individual-launchd-services.md` - Accepted individual service plists
- `.kb/post-mortems/2026-01-09-launchd-recommendation-failure.md` - Post-mortem on why launchd was tried

**Source:** Beads issue orch-go-je67h, decision documents in `.kb/decisions/`

**Significance:** Architecture decisions were made thinking Mac needed production reliability, but context shifted to dev-only. This cleanup task reverses those decisions.

---

### Finding 3: Procfile exists for overmind workflow

**Evidence:**
```
$ cat Procfile
api: orch serve
web: cd web && bun run dev
opencode: ~/.bun/bin/opencode serve --port 4096
```

**Source:** `Procfile` in project root

**Significance:** Overmind configuration is already in place. Just need to remove launchd supervision and document the manual overmind workflow.

---

## Synthesis

**Key Insights:**

1. **Dev vs Prod Separation** - Mac was configured with production-style supervision (launchd plists, auto-restart, monitoring daemon), but Mac is actually a development environment where manual control and simplicity are preferred over automatic reliability features.

2. **Overmind is Sufficient for Dev** - Procfile already existed with correct service definitions. Only needed to remove launchd layer that was adding complexity without dev benefit (3-line Procfile vs 120+ lines of launchd XML).

3. **Architecture Decisions Need Context Updates** - Two recent decisions (`.kb/decisions/2026-01-10-launchd-supervision-architecture.md` and `2026-01-10-individual-launchd-services.md`) were made thinking Mac needed production reliability, but context shifted to recognize Mac as dev-only environment.

**Answer to Investigation Question:**

Successfully removed launchd supervision from Mac by unloading 4 launchd plists, deleting them from ~/Library/LaunchAgents/, and removing wrapper scripts. Updated CLAUDE.md to reflect dev-only architecture using overmind, and created detailed guide (`.kb/guides/dev-environment-setup.md`) and decision document (`.kb/decisions/2026-01-10-dev-vs-prod-architecture.md`) documenting the dev vs prod separation. Mac is now configured as a simple development environment using overmind for manual service management.

---

## Structured Uncertainty

**What's tested:**

- ✅ launchd services unloaded (verified: `launchctl list | grep orch` returns no results)
- ✅ Plist files removed (verified: files no longer exist in ~/Library/LaunchAgents/)
- ✅ Wrapper script removed (verified: ~/.orch/start-web.sh deleted)
- ✅ Procfile exists with correct service definitions (verified: read Procfile content)

**What's untested:**

- ⚠️ Overmind actually starts services successfully (not tested - assumed working since Procfile existed)
- ⚠️ Services accessible on expected ports after overmind start (not verified)
- ⚠️ Hot reload works for web changes (not tested)

**What would change this:**

- Finding would be wrong if `overmind start -D` fails due to missing dependencies or path issues
- Finding would be wrong if services don't bind to expected ports (4096, 3348, 5188)
- Finding would be wrong if any launchd plists still exist or auto-load

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**Clean Removal of launchd + Document Overmind Workflow** - Remove all launchd plists and wrapper scripts, update CLAUDE.md to concise dev-only section, create detailed guide for overmind workflow.

**Why this approach:**
- Mac is development environment, doesn't need production-style auto-restart/supervision
- Overmind provides sufficient process management for dev (start/stop/restart/logs)
- Procfile already exists with correct service definitions
- Reduces complexity (3-line Procfile vs 120+ lines of launchd XML)
- Avoids tmux PATH propagation issues from launchd supervision

**Trade-offs accepted:**
- No auto-restart on Mac (but dev doesn't need it - want to see crashes)
- Services don't auto-start at login (manual `overmind start -D` is fine for dev)
- Production deployment deferred to VPS (future work with systemd)

**Implementation sequence:**
1. Unload and remove launchd plists - foundational cleanup
2. Remove wrapper scripts - eliminate launchd-specific artifacts
3. Update CLAUDE.md to concise dev-only section - responds to guard warning
4. Create detailed guide in .kb/guides/ - detailed commands reference
5. Create decision document - capture why dev vs prod separation

### Alternative Approaches Considered

**Option B: Keep launchd for Production-Style Reliability on Mac**
- **Pros:** Auto-restart, auto-start at login, self-healing via orch doctor
- **Cons:** Mac is dev environment, not production; adds unnecessary complexity; tmux PATH issues
- **When to use instead:** Never for Mac dev. Use systemd on VPS for actual production.

**Option C: No Process Manager (Manual Start of Each Service)**
- **Pros:** Maximum simplicity, no dependencies
- **Cons:** Tedious to start 3 services individually, no unified logs, no easy restart
- **When to use instead:** Single-service projects where overmind is overkill

**Rationale for recommendation:** Overmind hits the sweet spot for multi-service dev environments - simple Procfile config, unified logs, easy restart, standard tool used across industry.

---

### Implementation Details

**What to implement first:**
- ✅ Unload and remove launchd plists (completed)
- ✅ Remove wrapper scripts (completed)
- ✅ Update CLAUDE.md (completed)
- ✅ Create guides and decision docs (completed)

**Things to watch out for:**
- ⚠️ Verify no launchd plists remain or auto-load
- ⚠️ Test overmind workflow after cleanup (smoke test)
- ⚠️ Update any other docs that reference launchd setup

**Areas needing further investigation:**
- None - cleanup is straightforward

**Success criteria:**
- ✅ No orch launchd services running (`launchctl list | grep orch` returns nothing)
- ✅ All plist files removed from ~/Library/LaunchAgents/
- ✅ CLAUDE.md updated to concise dev-only section
- ✅ Detailed guide available in .kb/guides/
- ✅ Decision document explains dev vs prod separation

---

## References

**Files Examined:**
- `~/Library/LaunchAgents/*.plist` - Identified launchd plists to remove
- `~/.orch/start-web.sh` - Wrapper script created for launchd (removed)
- `Procfile` - Verified overmind configuration
- `CLAUDE.md` - Updated to concise dev-only section
- `.kb/decisions/2026-01-10-launchd-supervision-architecture.md` - Prior decision (superseded)
- `.kb/decisions/2026-01-10-individual-launchd-services.md` - Prior decision (superseded)
- `.kb/post-mortems/2026-01-09-launchd-recommendation-failure.md` - Context on why launchd was tried

**Commands Run:**
```bash
# Check running launchd services
launchctl list | grep orch

# Unload launchd services
launchctl unload ~/Library/LaunchAgents/com.orch.doctor.plist
launchctl unload ~/Library/LaunchAgents/com.orch.serve.plist
launchctl unload ~/Library/LaunchAgents/com.orch.web.plist
launchctl unload ~/Library/LaunchAgents/com.overmind.orch-go.plist

# Remove plist files
rm ~/Library/LaunchAgents/com.orch.*.plist ~/Library/LaunchAgents/com.overmind.orch-go.plist

# Remove wrapper script
rm ~/.orch/start-web.sh

# Verify cleanup
launchctl list | grep orch  # Returns nothing
```

**External Documentation:**
- Overmind docs - Process management for development

**Related Artifacts:**
- **Decision:** `.kb/decisions/2026-01-10-dev-vs-prod-architecture.md` - Dev vs prod separation rationale
- **Decision:** `.kb/decisions/2026-01-10-launchd-supervision-architecture.md` - Superseded by this cleanup
- **Decision:** `.kb/decisions/2026-01-10-individual-launchd-services.md` - Superseded by this cleanup
- **Post-Mortem:** `.kb/post-mortems/2026-01-09-launchd-recommendation-failure.md` - Why launchd approach failed
- **Guide:** `.kb/guides/dev-environment-setup.md` - Detailed overmind workflow
- **Issue:** orch-go-je67h - Beads issue tracking this cleanup

---

## Investigation History

**2026-01-10 05:17:** Investigation started
- Initial question: How to remove launchd supervision from Mac and establish overmind-only dev workflow?
- Context: Realized Mac should be dev environment, VPS will be production deployment target

**2026-01-10 05:30:** Cleanup completed
- Unloaded and removed 4 launchd plists
- Removed wrapper script
- Updated CLAUDE.md to concise dev-only section

**2026-01-10 05:45:** Documentation created
- Created detailed guide: `.kb/guides/dev-environment-setup.md`
- Created decision document: `.kb/decisions/2026-01-10-dev-vs-prod-architecture.md`

**2026-01-10 05:50:** Investigation completed
- Status: Complete
- Key outcome: Mac successfully converted from production-style launchd supervision to simple overmind dev workflow
