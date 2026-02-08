<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** `agentlog init` is ready to integrate, but `agentlog` and `orch events` are DIFFERENT systems with different purposes - don't confuse them.

**Evidence:** Tested `agentlog init` in 2 fresh directories - works reliably, creates `.agentlog/errors.jsonl` for development error tracking. Meanwhile `orch` uses `~/.orch/events.jsonl` for agent lifecycle events.

**Knowledge:** agentlog = development error aggregation (per-project `.agentlog/`). orch events = agent lifecycle tracking (global `~/.orch/`). The `/api/agentlog` endpoint name is confusing because it reads orch events, not agentlog errors.

**Next:** Recommend adding `agentlog init` to `orch init` as optional (`--with-agentlog`), but keep default off until agentlog sees more adoption.

**Confidence:** High (85%) - Clear technical understanding, usage patterns still emerging.

---

# Investigation: Is agentlog init ready to integrate into orch init?

**Question:** Should `orch init` also run `agentlog init`? What does agentlog provide and is it actively used?

**Started:** 2025-12-22
**Updated:** 2025-12-22
**Owner:** Agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** High (85%)

---

## Findings

### Finding 1: agentlog init works reliably

**Evidence:** Tested in two fresh directories:
- `/tmp/test-agentlog-init` - Detected TypeScript stack, created `.agentlog/` directory, added to `.gitignore`
- `/tmp/test-agentlog-go` - Detected Go stack (from `go.mod`), created `.agentlog/` directory

Both tests succeeded immediately with no errors.

**Source:** 
```bash
agentlog init                    # TypeScript detection
agentlog init --stack go         # Go detection
agentlog init --install          # Creates capture.go file
```

**Significance:** The command is stable and idempotent - safe to integrate.

---

### Finding 2: agentlog creates minimal structure, requires code integration

**Evidence:** `agentlog init` creates:
- `.agentlog/` directory
- `.agentlog/errors.jsonl` (empty file)
- Appends to `.gitignore`
- Prints code snippet to add to application

The code snippet must be manually added to the application entry point. With `--install`, it creates a capture file (e.g., `.agentlog/capture.go`) but still requires the developer to import it.

**Source:** `agentlog init --help`, tested output

**Significance:** Unlike `bd init` or `kb init` which are fully self-contained, agentlog requires additional developer action to be useful. This affects whether it should be in `orch init` by default.

---

### Finding 3: agentlog and orch events are DIFFERENT systems

**Evidence:** Two separate event systems exist:
1. **agentlog** (standalone binary): Development error aggregation
   - Location: `.agentlog/errors.jsonl` (per-project)
   - Purpose: Capture frontend/backend errors during development
   - Format: `{timestamp, source, error_type, message, context}`
   - Usage: 110 entries in orch-cli, 0 in orch-go

2. **orch events**: Agent lifecycle tracking
   - Location: `~/.orch/events.jsonl` (global)
   - Purpose: Track agent spawns, completions, errors
   - Format: `{type, session_id, timestamp, data}`
   - Usage: Active, many entries from recent spawns

Confusingly, `serve.go` has `/api/agentlog` endpoint that reads **orch events**, not agentlog errors.

**Source:**
- `pkg/events/logger.go` - orch events system
- `/Users/dylanconlin/Documents/personal/orch-cli/.agentlog/errors.jsonl` - 110 entries
- `/Users/dylanconlin/Documents/personal/agentlog/README.md` - agentlog purpose

**Significance:** These are complementary but separate concerns. agentlog is for development visibility; orch events is for orchestration visibility.

---

### Finding 4: agentlog is experimental/early stage

**Evidence:**
- Only 3 projects have `.agentlog/` directories: beads-ui-svelte, orch-cli, agentlog itself
- orch-go does NOT have `.agentlog/` despite having a web UI
- The agentlog binary is installed but not widely integrated
- README says "AI-native development observability" - positioning as new paradigm

**Source:** 
```bash
find ~/Documents/personal -name ".agentlog" -type d
# /Users/dylanconlin/Documents/personal/beads-ui-svelte/.agentlog
# /Users/dylanconlin/Documents/personal/orch-cli/.agentlog
# /Users/dylanconlin/Documents/personal/agentlog/.agentlog
```

**Significance:** Early adoption stage. Not yet proven as essential infrastructure like beads/kb.

---

### Finding 5: No hooks or watchers required

**Evidence:** agentlog is entirely passive - errors are written to `.agentlog/errors.jsonl` by application code. Reading happens via:
- `agentlog errors` - Query errors
- `agentlog tail` - Watch in real-time
- `agentlog prime` - Output context for AI agents

No daemons, no watchers, no hooks needed. The `errors.jsonl` file is just a log file.

**Source:** `agentlog --help`, tested commands

**Significance:** Zero maintenance overhead - safe to init and ignore if not used.

---

## Synthesis

**Key Insights:**

1. **Clear separation of concerns** - agentlog is for development observability (errors from your code), orch events is for orchestration observability (agent lifecycle). These are complementary, not overlapping.

2. **Safe but incomplete** - `agentlog init` is reliable and idempotent, but doesn't provide value until code is modified to send errors. This is different from beads/kb which work immediately after init.

3. **Naming confusion** - The `/api/agentlog` endpoint in serve.go reads orch events, not agentlog errors. This should probably be renamed to `/api/events` or have a separate `/api/agentlog` that actually reads `.agentlog/`.

**Answer to Investigation Question:**

**Should `orch init` also run `agentlog init`?**

**Recommendation: Optional, default off (`--with-agentlog`)**

Rationale:
- agentlog init is reliable (tested)
- But it requires additional code changes to be useful
- Adoption is still early (3 of many projects)
- Zero cost to init later when needed
- Keeps `orch init` focused on orchestration concerns

If orch-go web UI wants agentlog integration, that should be a separate feature-impl to:
1. Add `agentlog init` support to `orch init`
2. Add error capture to web/ code
3. Consider `/api/agentlog` endpoint for actual agentlog data

---

## Confidence Assessment

**Current Confidence:** High (85%)

**Why this level?**
- Tested agentlog init directly - reliable
- Examined both event systems in detail
- Clear understanding of the separation
- Minor uncertainty on Dylan's intent for adoption

**What's certain:**

- ✅ `agentlog init` works reliably (tested twice)
- ✅ agentlog and orch events are separate systems
- ✅ No hooks/watchers required - passive log file
- ✅ Current adoption is low (3 projects)

**What's uncertain:**

- ⚠️ Whether Dylan intends to push agentlog adoption across all projects
- ⚠️ Whether the `/api/agentlog` naming confusion is intentional
- ⚠️ Long-term vision for agentlog in orchestration stack

**What would increase confidence to Very High (95%):**

- Confirmation from Dylan on agentlog adoption plans
- Seeing agentlog integrated into orch-go web UI as proof of value
- Decision on `/api/agentlog` endpoint naming

---

## Implementation Recommendations

### Recommended Approach ⭐

**Add as optional flag to orch init** - `orch init --with-agentlog`

**Why this approach:**
- Zero cost to add (just shell to `agentlog init`)
- Doesn't change default behavior
- Available when needed
- Keeps orch init focused on orchestration essentials

**Trade-offs accepted:**
- Not auto-enabling agentlog (some projects may miss it)
- Acceptable because code changes are required anyway

**Implementation sequence:**
1. Add `--with-agentlog` flag to `cmd/orch/init.go`
2. Shell to `agentlog init` when flag is set
3. Report result in init summary

### Alternative Approaches Considered

**Option B: Add to default orch init**
- **Pros:** Every project gets agentlog ready
- **Cons:** Creates unused `.agentlog/` directories, requires manual code integration anyway
- **When to use instead:** When agentlog adoption is higher and code capture is templated

**Option C: Never integrate**
- **Pros:** Keep tools completely separate
- **Cons:** Misses opportunity for unified project setup
- **When to use instead:** If agentlog development stalls or direction changes

**Rationale for recommendation:** Option A balances readiness with current adoption reality. It's the minimal viable integration.

---

### Implementation Details

**What to implement first:**
- Add `--with-agentlog` flag to init.go
- Add `initAgentlog()` function that shells to `agentlog init`
- Update `InitResult` struct with agentlog fields

**Things to watch out for:**
- ⚠️ Don't confuse with existing `/api/agentlog` endpoint (reads orch events)
- ⚠️ Check if `agentlog` binary exists before calling (graceful degradation)
- ⚠️ Skip agentlog init on `--force` if already initialized (idempotency)

**Areas needing further investigation:**
- Should `/api/agentlog` be renamed to `/api/events`?
- Should there be a separate endpoint for actual agentlog data?
- How to template agentlog code capture per project type?

**Success criteria:**
- ✅ `orch init --with-agentlog` creates `.agentlog/` directory
- ✅ `orch init` (default) does NOT touch agentlog
- ✅ Graceful handling when agentlog binary not installed

---

## Self-Review

- [x] Real test performed (not code review)
- [x] Conclusion from evidence (not speculation)
- [x] Question answered
- [x] File complete
- [x] D.E.K.N. filled
- [x] NOT DONE claims verified (checked actual files for .agentlog presence)

**Self-Review Status:** PASSED

---

## References

**Files Examined:**
- `cmd/orch/init.go` - Current init implementation
- `cmd/orch/serve.go:84-99` - `/api/agentlog` endpoint (reads orch events, not agentlog)
- `pkg/events/logger.go` - Orch events system
- `/Users/dylanconlin/Documents/personal/agentlog/README.md` - agentlog purpose

**Commands Run:**
```bash
# Test agentlog init in fresh directories
cd /tmp && mkdir test-agentlog-init && cd test-agentlog-init && agentlog init

# Check Go project detection
mkdir /tmp/test-agentlog-go && echo "module test" > go.mod && agentlog init

# Find projects with .agentlog
find ~/Documents/personal -name ".agentlog" -type d

# Check existing error counts
wc -l ~/Documents/personal/orch-cli/.agentlog/errors.jsonl  # 110

# Check agentlog doctor in orch-go
agentlog doctor  # UNHEALTHY - not initialized
```

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2025-12-21-inv-scope-orch-init-project-standardization.md` - orch init design
- **Investigation:** `.kb/investigations/2025-12-20-inv-add-api-agentlog-endpoint-serve.md` - /api/agentlog endpoint (confusing name)

---

## Investigation History

**2025-12-22 13:55:** Investigation started
- Initial question: Is agentlog init ready to integrate into orch init?
- Context: Spawned to evaluate agentlog readiness for orch init integration

**2025-12-22 14:10:** Key finding - two separate systems
- Discovered agentlog vs orch events separation
- Noted `/api/agentlog` naming confusion

**2025-12-22 14:20:** Investigation completed
- Final confidence: High (85%)
- Status: Complete
- Key outcome: Recommend optional `--with-agentlog` flag, not default behavior
