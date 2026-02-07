<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** All infrastructure for Phase 1 health checks exists except cache headers - can implement 4 of 5 features immediately using existing patterns.

**Evidence:** Tested overmind status parsing, found pkg/notify notification system, verified port 5188 not checked, confirmed no X-Cache-Time headers in API responses.

**Knowledge:** Architecture changed from launchd to overmind (Jan 9), so health checks should parse `overmind status` not launchd plists; notification infrastructure is ready; doctor.go has extensible ServiceStatus pattern.

**Next:** Implement incrementally: (1) port 5188 check, (2) overmind services check, (3) --watch mode, (4) cache headers in serve.go, (5) cache freshness check.

**Promote to Decision:** recommend-no (tactical implementation, follows existing patterns)

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

# Investigation: P0 Implement Orch Doctor Health

**Question:** What infrastructure exists for implementing Phase 1 health checks (overmind, ports, notifications, cache) and what patterns should be followed?

**Started:** 2026-01-10
**Updated:** 2026-01-10
**Owner:** Agent og-feat-p0-implement-orch-10jan-d614
**Phase:** Complete
**Next Step:** None
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]
**Supersedes:** [Path to artifact this replaces, if applicable]
**Superseded-By:** [Path to artifact that replaced this, if applicable]

---

## Findings

### Finding 1: Overmind status output is parseable and machine-readable

**Evidence:**
```bash
$ overmind status
PROCESS   PID       STATUS
api       66172     running
web       66173     running
opencode  66174     running
```
Output has consistent column format: PROCESS (name), PID (process ID), STATUS (running/stopped).

**Source:**
- Command: `overmind status`
- Procfile:1-3 defines services: api, web, opencode

**Significance:** Can parse this output to check if all required services (api, web, opencode) are running. Simple string parsing on columns provides reliable health check without complex process inspection.

---

### Finding 2: Notification infrastructure exists and is already integrated

**Evidence:**
- pkg/notify/notify.go:1-97 provides `Default()` factory method
- Supports `.ServiceCrashed()` method specifically for service failures (line 79-86)
- Respects `~/.orch/config.yaml` notifications.enabled setting
- Uses beeep library for cross-platform desktop notifications

**Source:** pkg/notify/notify.go:1-97

**Significance:** Don't need to build notification infrastructure from scratch. Can reuse existing `notify.Default().ServiceCrashed()` for health check failures in `--watch` mode.

---

### Finding 3: Port 5188 (web UI) not currently checked by doctor

**Evidence:**
- cmd/orch/doctor.go checks ports 4096 (OpenCode) and 3348 (orch serve)
- Procfile:2 shows `web: cd web && bun run dev` which runs on port 5188 (from CLAUDE.md)
- No checkWebUI() function exists in doctor.go

**Source:**
- cmd/orch/doctor.go:218-309 (checkOpenCode, checkOrchServe)
- Procfile:2
- CLAUDE.md service ports table

**Significance:** Need to add port 5188 health check to complete Phase 1 requirements. Should follow same pattern as existing port checks (TCP dial or HTTP GET).

---

### Finding 4: No cache headers currently set on API responses

**Evidence:**
```bash
$ curl -sk https://localhost:3348/health
{"status":"ok"}
```
No `X-Cache-Time` or `X-Orch-Version` headers in response. Grepping serve.go for cache-related headers returns no matches.

**Source:**
- Command: `curl -sI https://localhost:3348/health`
- cmd/orch/serve.go (grepped for X-Cache, timestamp)

**Significance:** Phase 1 requirement for cache freshness checks requires adding timestamp headers to API responses first. This is a prerequisite for the doctor command to validate cache freshness.

---

### Finding 5: Architecture change from launchd to overmind

**Evidence:**
- CLAUDE.md states: "Changed Jan 9, 2026: Replaced launchd with overmind for dashboard services"
- Decision document (.kb/decisions/2026-01-09-dashboard-reliability-architecture.md:77) references launchd services
- Procfile now defines services instead of launchd plists

**Source:**
- /Users/dylanconlin/Documents/personal/orch-go/CLAUDE.md (Dashboard Server Management section)
- .kb/decisions/2026-01-09-dashboard-reliability-architecture.md:76-81
- Procfile:1-3

**Significance:** Health checks should verify overmind services (via `overmind status`), NOT launchd services. The decision document is slightly outdated in this respect.

---

### Finding 6: Existing doctor.go has extensible structure for new checks

**Evidence:**
- ServiceStatus struct (lines 70-78) with Name, Running, Port, Details, CanFix, FixAction
- runDoctor() adds checks to report.Services slice (lines 148-171)
- printDoctorReport() formats output (referenced line 174)
- --fix flag infrastructure exists (lines 177-212)

**Source:** cmd/orch/doctor.go:70-215

**Significance:** Can add new health checks by following existing pattern: create checkXXX() function returning ServiceStatus, append to report.Services, implement startXXX() for --fix support.

---

## Synthesis

**Key Insights:**

1. **Infrastructure is 90% ready** - Notification system (Finding 2), extensible doctor pattern (Finding 6), and overmind status parsing (Finding 1) are all in place. Only need to add new health checks following existing patterns.

2. **Overmind replaces launchd** - Architecture changed Jan 9, so health checks should use `overmind status` instead of launchd service checks (Finding 5). Decision document is slightly out of date.

3. **Cache headers are prerequisite** - Can't implement cache freshness checks until API responses include timestamp headers (Finding 4). This is a two-step implementation: (1) add headers to serve.go, (2) check headers in doctor.go.

**Answer to Investigation Question:**

All infrastructure exists for Phase 1 implementation except cache headers. Can implement 4 of 5 health checks immediately:
1. Overmind services check (parse `overmind status`)
2. Port 5188 check (follow existing port check pattern)
3. Orphaned vite processes check (ps aux filtering)
4. --watch mode (notification loop using existing notify package)

Cache freshness check requires adding `X-Cache-Time` headers to serve.go API responses first, then validating those headers in doctor.go.

Limitations: Did not investigate orphaned process detection patterns yet (need to understand what constitutes "orphaned" for vite processes under overmind management).

---

## Structured Uncertainty

**What's tested:**

- ✅ Overmind status output is parseable (verified: ran `overmind status`, got consistent column format)
- ✅ Notification infrastructure exists (verified: read pkg/notify/notify.go, found Default() and ServiceCrashed())
- ✅ Port 5188 not checked (verified: grep'd doctor.go for checkWebUI or 5188, found nothing)
- ✅ No cache headers in API (verified: ran `curl -sI https://localhost:3348/health`, no X-Cache-Time)
- ✅ Architecture changed to overmind (verified: read CLAUDE.md, found "Replaced launchd with overmind" statement)

**What's untested:**

- ⚠️ Orphaned vite process detection logic (haven't determined what "orphaned" means under overmind)
- ⚠️ --watch mode notification frequency (don't know optimal polling interval)
- ⚠️ Cache freshness threshold (decision doc says >60s, but not validated)
- ⚠️ Port 5188 response time (assuming HTTP GET will work, haven't tested)

**What would change this:**

- Finding would be wrong if overmind status output format changes across versions
- Finding would be wrong if vite processes under overmind are never orphaned (PPID always correct)
- Finding would be wrong if cache headers already exist but under different name

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**Incremental Feature Additions Following Existing Pattern** - Add new health checks one at a time using the proven ServiceStatus pattern already in doctor.go.

**Why this approach:**
- Existing code has well-defined extension points (Finding 6) - just add new checkXXX() functions
- Notification infrastructure ready to use (Finding 2) - can enable --watch immediately
- Overmind status is already parseable (Finding 1) - no complex integration needed
- Minimizes risk by following established patterns rather than refactoring

**Trade-offs accepted:**
- Not implementing cache freshness check initially (requires API changes first - Finding 4)
- Deferring orphaned process detection until we understand what "orphaned" means under overmind
- Accepting that decision document becomes slightly outdated (references launchd not overmind - Finding 5)

**Implementation sequence:**
1. **Add port 5188 check** - Simplest addition, follows exact pattern of checkOrchServe() (Finding 3)
2. **Add overmind services check** - Parse `overmind status`, check all 3 services running (Finding 1)
3. **Add --watch mode flag** - Poll health every 30s, use notify.ServiceCrashed() on failures (Finding 2)
4. **Add cache headers to serve.go** - Middleware to set X-Cache-Time on all responses (Finding 4)
5. **Add cache freshness check** - Validate X-Cache-Time header is <60s old (depends on step 4)

### Alternative Approaches Considered

**Option B: Refactor doctor.go before adding features**
- **Pros:** Could clean up code structure, add better abstractions
- **Cons:** High risk, breaks working code, delays feature delivery
- **When to use instead:** If we had 10+ new checks to add and current pattern was insufficient

**Option C: Use system monitoring tool (Prometheus, Grafana)**
- **Pros:** Industry-standard observability, rich dashboards
- **Cons:** Heavy dependency, overkill for local development tool, adds deployment complexity
- **When to use instead:** If this was production infrastructure, not local developer tooling

**Rationale for recommendation:** Findings show infrastructure is 90% ready. Adding features incrementally using existing patterns is lowest risk, fastest delivery path. Can always refactor later if pattern proves insufficient.

---

### Implementation Details

**What to implement first:**
1. Port 5188 check - Highest immediate value, simple HTTP GET to http://localhost:5188
2. Overmind services check - Core reliability requirement, parse `overmind status` output
3. --watch mode - Enables continuous monitoring instead of one-time checks

**Things to watch out for:**
- ⚠️ Overmind must be installed and running - check for `which overmind` before parsing status
- ⚠️ Port 5188 may not be HTTPS unlike port 3348 - use plain HTTP not HTTPS
- ⚠️ Watch mode polling interval - too frequent = CPU waste, too infrequent = slow detection (recommend 30s)
- ⚠️ Notification fatigue - don't spam on every poll failure, only notify on state transitions (healthy → unhealthy)

**Areas needing further investigation:**
- What defines an "orphaned" vite process when running under overmind? (PPID=1, or zombie state, or something else?)
- Should --watch mode also check for stalled sessions, or just service health?
- Should cache freshness be configurable, or hardcode 60s threshold?

**Success criteria:**
- ✅ `orch doctor` shows all 5 services (binary, OpenCode, orch serve, web UI, overmind status)
- ✅ `orch doctor --watch` runs continuously and sends desktop notification when service fails
- ✅ Port 5188 failure correctly detected when web server is stopped
- ✅ Overmind service failures correctly detected when overmind is stopped
- ✅ Cache freshness check works after API headers added

---

## References

**Files Examined:**
- cmd/orch/doctor.go:1-1179 - Current health check implementation, ServiceStatus pattern
- pkg/notify/notify.go:1-97 - Desktop notification infrastructure
- cmd/orch/serve.go:1-100 - API server structure, need to add cache headers
- Procfile:1-3 - Overmind service definitions (api, web, opencode)
- CLAUDE.md - Dashboard Server Management section, overmind documentation

**Commands Run:**
```bash
# Check overmind status output format
overmind status

# Check overmind version
which overmind && overmind --version

# Check for vite processes
ps aux | grep -i vite | grep -v grep

# Check API health endpoint
curl -sk https://localhost:3348/health

# Check API headers
curl -sI https://localhost:3348/health
```

**Related Artifacts:**
- **Decision:** .kb/decisions/2026-01-09-dashboard-reliability-architecture.md - Phase 1 requirements (slightly outdated re: launchd vs overmind)
- **Investigation:** .kb/investigations/2026-01-09-inv-p0-implement-orch-doctor-health.md - Prior investigation by different agent
- **Beads Issue:** orch-go-uyveu - Tracking issue for this implementation

---

## Investigation History

**2026-01-10 00:32:** Investigation started
- Initial question: What infrastructure exists for implementing Phase 1 health checks?
- Context: Spawned to implement P0 health check system after architecture change from launchd to overmind

**2026-01-10 00:45:** Core infrastructure findings completed
- Discovered overmind status parsing, notification package, existing doctor pattern
- Identified cache headers as prerequisite for freshness checks

**2026-01-10 00:50:** Investigation completed
- Status: Complete - Ready to move to implementation
- Key outcome: 90% of infrastructure ready, incremental implementation recommended
