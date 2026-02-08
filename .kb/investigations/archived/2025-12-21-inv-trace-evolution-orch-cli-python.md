## Summary (D.E.K.N.)

**Delta:** orch evolved from a Python CLI for Claude Code (Nov 29) through OpenCode integration to a Go rewrite (Dec 19-21), driven by three concerns: scalability (tmux → HTTP API), distribution (pip → single binary), and architecture clarity (five-concern separation).

**Evidence:** Traced 575 commits in orch-cli (Python) and 218 commits in orch-go over 22 days. Examined 5 architectural decisions and 200+ investigations across both repos.

**Knowledge:** orch's identity: "kubectl for AI agents" - spawn, monitor, coordinate, complete. The Go rewrite isn't abandonment but architectural evolution toward the same goal with better primitives (OpenCode API, single binary, goroutines).

**Next:** Continue orch-go development using OpenCode as backend; orch-cli (Python) becomes fallback/reference. Focus on porting agent management (clean, abandon, wait) next.

**Confidence:** High (85%) - Based on comprehensive git history analysis and code examination; some uncertainty about which Python features are actively used.

---

# Investigation: Trace Evolution from orch-cli (Python) to orch-go

**Question:** What was orch trying to be, how did it evolve, and what should it become?

**Started:** 2025-12-21
**Updated:** 2025-12-21
**Owner:** Worker Agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** High (85%)

---

## Findings

### Finding 1: The Origin Story (Nov 29, 2025)

**Evidence:** orch-cli started on November 29, 2025 with a clear vision expressed in the first commit (`db2d500`): "orch CLI for AI agent orchestration."

The README captured the essence: **"kubectl for AI agents"** - a command-line tool for managing AI coding agents, providing:
- Spawning: Launch agents with structured context
- Monitoring: Track progress in real-time
- Coordination: Manage multiple agents working together
- Completion: Verify agent work and clean up

The analogy is precise: just as kubectl manages container lifecycle across Kubernetes clusters, orch manages agent lifecycle across AI sessions.

**Source:** `orch-cli/README.md`, `git log --reverse | head -30`

**Significance:** The identity was clear from day one. Everything that followed was implementation and refinement, not redefinition.

---

### Finding 2: Rapid Feature Velocity in Python (Nov 29 - Dec 19)

**Evidence:** 575 commits over ~20 days in orch-cli Python. Key milestones:

| Date | Feature | Significance |
|------|---------|--------------|
| Nov 29 | Initial commit | Core CLI skeleton |
| Nov 30 | beads integration | Task tracking via `bd` CLI |
| Nov 30 | SessionStart hook | Automatic context loading |
| Dec 1 | OpenCode backend | Alternative to Claude Code |
| Dec 5 | Five concerns architecture | Architectural clarity (orch, beads, kb, skills, tmux) |
| Dec 6 | Eliminate WORKSPACE.md | beads becomes sole state tracker |
| Dec 8 | Daemon commands | Autonomous agent processing |
| Dec 12 | OpenCode as default backend | OpenCode displaces Claude Code |
| Dec 14 | Playwright MCP | Browser automation for agents |
| Dec 18 | Go + OpenCode decision | Formal decision to rewrite in Go |

The Python version grew from ~1,000 lines to ~27,000 lines (67 .py files) with ~30 commands covering:
- Agent lifecycle (spawn, status, complete, clean, abandon)
- Monitoring (tail, question, wait, check)
- Meta-orchestration (focus, drift, next, daemon)
- Analysis (friction, synthesis, lint)

**Source:** `git log --format="%as %s" | grep -E "(feat|fix):" | head -40`, `wc -l src/orch/*.py`

**Significance:** Python enabled rapid prototyping and feature discovery. The 27k lines represent learned requirements, not just code.

---

### Finding 3: Architectural Decision - Five Concerns (Dec 1)

**Evidence:** `.kb/decisions/2025-12-01-five-concerns-architecture.md` articulated the layered architecture:

| Tool | Layer | Storage | Purpose |
|------|-------|---------|---------|
| `bd` | Memory | `.beads/` | Task state, dependencies, execution log |
| `kb` | Knowledge | `.kb/` | Investigations, decisions, patterns |
| `skills` | Guidance | `~/.claude/skills/` | Agent behavioral procedures |
| `orch` | Lifecycle | (stateless) | Spawn, monitor, complete, verify |
| `tmux` | Session | (runtime) | Persistence, attach, output |

Key principle: **"Each tool owns one concern. Lifecycle layer (orch) has no state of its own - it orchestrates, but state lives in beads (tasks) and kb (knowledge)."**

**Source:** `orch-cli/.kb/decisions/2025-12-01-five-concerns-architecture.md`

**Significance:** This decision clarified orch's role: orchestration layer, not storage layer. State belongs elsewhere (beads, kb). Orch is the conductor, not the library.

---

### Finding 4: The Pivot Point - Go + OpenCode Decision (Dec 18)

**Evidence:** `.kb/decisions/2025-12-18-sdk-based-agent-management.md` documented the rewrite decision. Root causes for change:

1. **tmux-based architecture limitations:**
   - Manual completion discovery (cycling through windows)
   - No push notifications
   - Post-completion Q&A friction
   - No structured access for synthesis/handoff

2. **Python distribution friction:**
   - pip install, Python version issues
   - 100-300ms startup time
   - Different from other tools (bd, kn, kb are Go binaries)

3. **OpenCode provided better primitives:**
   - REST API for session management
   - SSE for real-time events
   - Native session persistence and Q&A
   - HTTP client simpler than subprocess management

**Source:** `orch-cli/.kb/decisions/2025-12-18-sdk-based-agent-management.md`

**Significance:** The rewrite wasn't failure but evolution. Python taught what was needed; Go provides better implementation primitives.

---

### Finding 5: The Go Rewrite - Rapid Convergence (Dec 19-21)

**Evidence:** 218 commits in 3 days in orch-go. The POC investigation (`2025-12-19-simple-opencode-poc-spawn-session-via.md`) proved viability immediately:

Day 1 (Dec 19):
- OpenCode HTTP client working
- spawn, send, monitor commands
- SSE event streaming
- Desktop notifications on completion

Day 2 (Dec 20):
- complete command
- daemon skeleton
- serve command (API for beads-ui)
- review command

Day 3 (Dec 21):
- tmux integration (optional)
- capacity management
- port allocation
- Four-layer reconciliation in clean

Current orch-go state:
- 17,364 lines of Go (main.go + pkg/)
- 23+ commands (spawn, status, complete, clean, abandon, wait, question, tail, review, serve, daemon, focus, drift, next, port, account, usage, etc.)
- Modular pkg structure: opencode, events, notify, skills, spawn, tmux, verify, registry, focus, daemon, capacity, port

**Source:** `git log --since="2025-12-19"`, `wc -l cmd/orch/main.go pkg/**/*.go`

**Significance:** Go version reached near feature parity in 3 days by leveraging learned requirements from Python. OpenCode API made implementation cleaner.

---

### Finding 6: What Was Learned - The Investigations

**Evidence:** The investigation counts tell the evolution story:

| Repo | Investigation Count | Time Period | Focus |
|------|---------------------|-------------|-------|
| orch-cli | 200+ | Nov 30 - Dec 19 | Feature discovery, debugging, architecture |
| orch-go | 164+ | Dec 19 - Dec 21 | Integration testing, porting, validation |

Key investigation patterns in orch-cli:
- Feature exploration: "how should X work?"
- Debugging: "why does X fail?"
- Architecture: "how should concerns be separated?"
- Integration: "how do tools interact?"

Key investigation patterns in orch-go:
- Porting: "implement X from Python"
- Validation: "test X works with OpenCode"
- Edge cases: "handle Y failure mode"
- Synthesis: "deep pattern analysis"

**Source:** `ls -la .kb/investigations/` in both repos, pattern analysis of filenames

**Significance:** Investigations are the learning record. 200+ investigations taught what orch needs to be. The Go rewrite applies those lessons.

---

## Synthesis

**Key Insights:**

1. **Identity was stable from day one:** "kubectl for AI agents" - spawn, monitor, coordinate, complete. The core purpose never changed; only implementation evolved.

2. **Python was the prototype phase:** 27k lines, 30 commands, 200+ investigations discovered what orch needs. The complexity wasn't waste - it was learning.

3. **Three forces drove the rewrite:**
   - **Scalability:** tmux visual access → OpenCode HTTP API
   - **Distribution:** pip install → single binary
   - **Architecture:** conflated concerns → five-layer separation

4. **OpenCode was the catalyst:** Providing REST API + SSE eliminated tmux subprocess management overhead. Go became natural choice once HTTP client was the interface.

5. **The rewrite preserved knowledge:** Investigations, decisions, and patterns from Python phase inform Go implementation. The `.kb/` directory is the institutional memory.

**Answer to Investigation Question:**

**What was orch trying to be?**
kubectl for AI agents - managing agent lifecycle (spawn, monitor, coordinate, complete) just as kubectl manages container lifecycle.

**How did it evolve?**
1. **Origin (Nov 29):** Python CLI for Claude Code, tmux-based
2. **Growth (Dec 1-17):** Feature discovery, OpenCode integration, architectural clarity
3. **Pivot (Dec 18):** Decision to rewrite in Go using OpenCode API
4. **Convergence (Dec 19-21):** Go rewrite reaching feature parity

**What should it become?**
orch-go is the answer: a single-binary CLI that orchestrates AI agents via OpenCode's HTTP API, with:
- Core lifecycle: spawn, monitor, send, complete, clean, abandon
- Meta-orchestration: focus, drift, next, daemon
- Analysis: review, synthesis (via agents)
- Integration: beads (tasks), kb (knowledge), skills (guidance)

The five-concern architecture remains the north star:
- orch = orchestration layer (stateless)
- beads = task memory
- kb = knowledge artifacts
- skills = agent guidance
- OpenCode = AI session management

---

## Confidence Assessment

**Current Confidence:** High (85%)

**Why this level?**

Comprehensive evidence from git history (793 commits total), architecture decisions (5 documented), and investigation artifacts (364+). Direct code examination of both repos.

**What's certain:**

- ✅ Identity was "kubectl for AI agents" from day one
- ✅ Python phase produced 27k lines and 30 commands
- ✅ Five-concern architecture was decided Dec 1
- ✅ Go rewrite started Dec 19 after OpenCode decision Dec 18
- ✅ Go version reached near parity in 3 days (218 commits)

**What's uncertain:**

- ⚠️ Which Python features are actively used vs legacy
- ⚠️ Whether all 30 Python commands need Go equivalents
- ⚠️ Long-term maintenance model (Go only? hybrid?)

**What would increase confidence to Very High (95%+):**

- Usage analytics on Python commands
- Multi-week operation with Go version
- User feedback on missing features

---

## Implementation Recommendations

**Purpose:** Guide orch development going forward.

### Recommended Approach ⭐

**orch-go as primary, orch-cli (Python) as reference/fallback:**

1. **Continue orch-go development** - Port remaining high-value commands
2. **Use Python for complex analysis** - friction, synthesis can shell out if needed
3. **Maintain architectural discipline** - orch orchestrates, doesn't store state
4. **Let OpenCode evolve** - orch-go benefits from OpenCode improvements

**Priority for next porting:**
1. Agent management: tail, question (nearly done), resume
2. Project setup: init (minimal, mostly done)
3. Meta-orchestration: daemon completion (polling, spawning)
4. Analysis: Consider spawning agents for friction/synthesis rather than coding in orch

**Why this approach:**
- Core lifecycle already works in Go
- Python remains available for complex features
- Focus energy on high-value, frequently-used commands

**Trade-offs accepted:**
- Some Python features may not be ported
- Two codebases temporarily (transition period)
- Complex analysis may require agent spawning vs in-CLI

### Alternative Approaches Considered

**Option B: Port everything to Go**
- **Pros:** Single codebase, consistent experience
- **Cons:** Significant effort, some features may be unused
- **When to use instead:** If Python causes operational issues

**Option C: Keep hybrid permanently**
- **Pros:** Each language for its strengths
- **Cons:** Two tools to maintain, user confusion
- **When to use instead:** If Go can't handle certain features

**Rationale for recommendation:** Go version is working and growing rapidly. Focus on completing core features rather than perfect parity. Python remains fallback.

---

## References

**Files Examined:**
- `orch-cli/README.md` - Original vision
- `orch-go/README.md` - Current Go state
- `orch-cli/.kb/decisions/2025-12-01-five-concerns-architecture.md` - Architecture decision
- `orch-cli/.kb/decisions/2025-12-18-sdk-based-agent-management.md` - Go rewrite decision
- `orch-cli/.kb/decisions/2025-12-06-eliminate-workspace-md.md` - State simplification
- `orch-go/.kb/investigations/2025-12-19-simple-opencode-poc-spawn-session-via.md` - Go POC
- `orch-go/.kb/investigations/2025-12-20-inv-compare-orch-cli-python-orch.md` - Feature comparison

**Commands Run:**
```bash
# Git history analysis
cd /Users/dylanconlin/Documents/personal/orch-cli && git log --oneline | wc -l  # 575
cd /Users/dylanconlin/Documents/personal/orch-go && git log --oneline | wc -l  # 218

# Code size
wc -l src/orch/*.py | tail -1  # 27345 Python
wc -l cmd/orch/main.go pkg/**/*.go | tail -1  # 17364 Go

# Investigation counts
ls .kb/investigations/*.md | wc -l  # 200+ (orch-cli), 164+ (orch-go)
```

**Related Artifacts:**
- **Decision:** `.kb/decisions/2025-12-01-five-concerns-architecture.md` - Separation of concerns
- **Decision:** `.kb/decisions/2025-12-18-sdk-based-agent-management.md` - Go rewrite
- **Investigation:** `2025-12-20-inv-compare-orch-cli-python-orch.md` - Feature comparison

---

## Investigation History

**2025-12-21 14:30:** Investigation started
- Initial question: What was orch trying to be, how did it evolve, what should it become?
- Context: Need narrative to guide future development

**2025-12-21 15:00:** Git history analysis complete
- Traced 575 Python commits, 218 Go commits
- Identified key milestones and architectural decisions

**2025-12-21 15:30:** Architecture decisions examined
- Read five-concerns decision, Go rewrite decision, workspace elimination
- Understood layered architecture evolution

**2025-12-21 16:00:** Investigation completed
- Final confidence: High (85%)
- Status: Complete
- Key outcome: orch's identity stable ("kubectl for AI agents"), Go rewrite is architectural evolution not pivot

---

## Self-Review

- [x] Real test performed (git history analysis, code examination)
- [x] Conclusion from evidence (not speculation)
- [x] Question answered
- [x] File complete
- [x] D.E.K.N. filled

**Self-Review Status:** PASSED

**Discovered Work Check:**
- No bugs discovered
- Enhancement idea: Update orch-go README to include evolution narrative for new contributors
- Documentation gap: No single "orch philosophy" document (this investigation could become one)

**Leave it Better:**
```bash
kn decide "orch-go is primary CLI, orch-cli (Python) is reference/fallback" --reason "Go provides better primitives (single binary, OpenCode HTTP client, goroutines); Python taught requirements"
```
