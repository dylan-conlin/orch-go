## Summary (D.E.K.N.)

**Delta:** ~70 kb quick entries about spawn backends synthesized into spawn.md and model-access-spawn-paths.md updates.

**Evidence:** Grepped entries.jsonl for spawn/backend/escape-hatch patterns; reviewed existing guides for coverage gaps.

**Knowledge:** spawn.md lacked triple backend architecture docs; model-access-spawn-paths.md lacked Docker container constraints (BEADS_NO_DAEMON, PATH, rate limit clarification).

**Next:** Close - guides now document triple spawn architecture comprehensively.

**Promote to Decision:** recommend-no (tactical documentation update, not architectural change)

---

# Investigation: Synthesize Spawn Backend KB Quick Entries

**Question:** What kb quick entries about spawn backends need to be synthesized into guide updates?

**Started:** 2026-01-21
**Updated:** 2026-01-21
**Owner:** Worker agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: spawn.md missing triple backend architecture

**Evidence:** spawn.md documented "Spawn Modes" (headless/tmux/inline) but not backend architecture (claude/opencode/docker). kb quick entries establish:
- Claude CLI is default backend (Jan 18 decision)
- Only two viable API paths: claude+opus or opencode+sonnet
- Docker provides fingerprint isolation for rate limit bypass
- Infrastructure detection is advisory-only

**Source:**
- `.kb/quick/entries.jsonl` - kb-318507, kb-a906ec, kb-d562c9, kb-e3e0a8
- `.kb/guides/spawn.md` - lines 55-70

**Significance:** Agents consulting spawn.md wouldn't know about backend selection priority, escape hatches, or when to use Docker.

---

### Finding 2: Docker container constraints undocumented

**Evidence:** Multiple kb quick entries document Docker-specific constraints not in model-access-spawn-paths.md:
- `BEADS_NO_DAEMON=1` required (Unix sockets fail over Docker mounts)
- Container PATH must include `/usr/local/go/bin` for auto-rebuild
- Weekly quota is account-level (Docker doesn't bypass it)
- Real configs mounted read-only to override `~/.claude-docker/` overlay

**Source:**
- kb-6d3282: "Docker containers must set BEADS_NO_DAEMON=1"
- kb-6aabaa: "Docker backend container PATH must include /usr/local/go/bin"
- kb-c3dbe7: "Claude Max usage quota is account-level, not device-level"
- kb-55f9d7: "Docker spawns mount real configs explicitly"

**Significance:** Docker spawn failures were documented in kb quick but not synthesized into the model document. Fresh agents hitting these issues would need to re-discover them.

---

### Finding 3: Key spawn decisions scattered across entries

**Evidence:** ~70 kb quick entries document spawn-related decisions including:
- Fire-and-forget tmux spawn (no session ID capture)
- Agents limited to 3 iterations without human review
- Abandon agents after service crashes
- Don't spawn multiple agents for same file
- needs:playwright label triggers MCP in daemon spawns

**Source:** entries.jsonl lines 1-193 (grep results file)

**Significance:** These constraints and decisions were not consolidated in spawn.md's "Key Decisions" section.

---

## Synthesis

**Key Insights:**

1. **Triple spawn architecture was under-documented** - spawn.md focused on UI modes (headless/tmux/inline) but not backend selection (claude/opencode/docker). The backend determines available models, cost, and crash resilience.

2. **Docker constraints learned from production** - BEADS_NO_DAEMON, PATH requirements, and rate limit clarification were discovered through usage but not synthesized into guides.

3. **Spawn safety constraints scattered** - Rules about iteration limits, service crash handling, and concurrent file editing existed in kb quick but weren't consolidated.

**Answer to Investigation Question:**

70+ kb quick entries needed synthesis into two documents:
1. **spawn.md**: Added triple backend architecture section, updated Key Decisions with backend/safety constraints
2. **model-access-spawn-paths.md**: Added Docker container constraints and rate limit clarification

---

## Structured Uncertainty

**What's tested:**

- spawn.md now documents backend selection priority cascade
- model-access-spawn-paths.md now includes Docker environment constraints
- Key decisions consolidated from kb quick entries into guides

**What's untested:**

- Whether agents will actually consult these updated sections
- Whether there are additional kb quick entries that should have been synthesized

**What would change this:**

- New Docker spawn failures revealing additional undocumented constraints
- Backend selection logic changing in code but not reflected in docs

---

## Implementation Recommendations

**Purpose:** Documentation synthesis complete. No code changes needed.

### Recommended Approach

**Guides Updated** - spawn.md and model-access-spawn-paths.md now reflect kb quick entries.

**Files modified:**
- `.kb/guides/spawn.md` - Added Backend Architecture section, updated Key Decisions
- `.kb/models/model-access-spawn-paths.md` - Added Docker constraints and rate limit clarification

---

## References

**Files Examined:**
- `.kb/quick/entries.jsonl` - All kb quick entries (~70 spawn-related)
- `.kb/guides/spawn.md` - Existing spawn documentation
- `.kb/models/model-access-spawn-paths.md` - Existing model/spawn path documentation
- `.kb/guides/dual-spawn-mode-implementation.md` - Implementation guide
- `.kb/guides/model-selection.md` - Model selection reference

**Commands Run:**
```bash
# Find spawn-related kb quick entries
grep -E "spawn|backend|escape.*hatch|tmux|headless|docker|claude.*mode|opencode.*mode" .kb/quick/entries.jsonl
```

**Related Artifacts:**
- **Guide:** `.kb/guides/spawn.md` - Updated with triple backend architecture
- **Model:** `.kb/models/model-access-spawn-paths.md` - Updated with Docker constraints

---

## Investigation History

**2026-01-21 18:55:** Investigation started
- Initial question: What kb quick entries need synthesis into guides?
- Context: Task to cluster ~50 spawn backend entries and update guides

**2026-01-21 19:15:** Investigation completed
- Status: Complete
- Key outcome: spawn.md and model-access-spawn-paths.md updated with ~70 kb quick entry findings
