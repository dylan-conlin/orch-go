## Summary (D.E.K.N.)

**Delta:** The daemon's thinking can become consumable output via a filesystem-backed digest system (`.orch/digest/`) served through a new dashboard "Thinking" tab, using a periodic daemon task to scan .kb/ artifact changes and package them as typed products.

**Evidence:** Explored 40+ dashboard API endpoints, 120 daemon files, 26 models, 24 threads, synthesis auto-create flow, reflect pipeline, and frontend architecture (SvelteKit + SSE + polling stores).

**Knowledge:** The key insight is that thinking products are *operational* artifacts (not knowledge artifacts) — they have a lifecycle (new→read→starred→archived) that doesn't fit git-committed .kb/ files. Storage belongs in `~/.orch/digest/` alongside events.jsonl and reflect-suggestions.json, following the existing pattern of filesystem-as-API-data-source.

**Next:** Implement in 3 phases: (1) daemon digest producer + API, (2) dashboard Thinking tab, (3) quality gate feedback loop. Create implementation issues.

**Authority:** architectural — Cross-component design (daemon + API + dashboard) with multiple valid approaches, requires synthesis across boundaries.

---

# Investigation: Synthesis-as-Product Architecture

**Question:** How should the daemon package its thinking (investigations, probes, models, threads) as consumable output for Dylan to read asynchronously?

**Started:** 2026-03-14
**Updated:** 2026-03-14
**Owner:** architect agent
**Phase:** Complete
**Next Step:** None — implementation issues created
**Status:** Complete

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| `synthesis_auto_create.go` | extends — this produces *issues* for synthesis; our design produces *reading products* from synthesis | Yes, read source | None |
| `reflect.go` | extends — this produces raw suggestions; our design packages suggestions as consumer-ready products | Yes, read source | None |
| `knowledge-accretion model` | deepens — 85.5% orphan rate means most investigations never surface to Dylan | Yes, read probes | None — our design addresses this gap |
| `dashboard-architecture model` | extends — existing SSE + polling patterns inform API design | Yes, read serve*.go | None |

---

## Findings

### Finding 1: The understanding gap is structural, not informational

**Evidence:** The daemon produces understanding through multiple mechanisms:
- `pkg/daemon/reflect.go` runs `kb reflect` periodically, saves to `~/.orch/reflect-suggestions.json` (synthesis clusters, promote candidates, stale decisions, model drift)
- `pkg/daemon/synthesis_auto_create.go` creates beads issues when investigation clusters exceed threshold (default 5+)
- `pkg/daemon/knowledge_health.go` detects stale KB artifacts
- `.kb/threads/` has 24 strategic threads with active exploration
- `.kb/models/` has 26 models with 140+ probes

But none of these produce a consumer-ready artifact for Dylan. The reflect-suggestions are raw JSON. Synthesis auto-create produces beads issues (work items, not reading material). Threads evolve silently. Model probes confirm/contradict claims with no notification.

**Source:** `pkg/daemon/reflect.go:1-392`, `pkg/daemon/synthesis_auto_create.go:1-264`, `.kb/threads/` (24 files), `.kb/models/` (26 directories)

**Significance:** The problem isn't that the daemon doesn't think — it's that thinking isn't packaged for consumption. This is a *product design* problem, not an infrastructure problem.

---

### Finding 2: Existing infrastructure provides the production pipeline

**Evidence:** The daemon already has:
- **Periodic task scheduler** (`pkg/daemon/periodic.go`) — runs tasks on intervals
- **Reflect pipeline** (`reflect.go`) — surfaces synthesis clusters, stale decisions, model drift, defect patterns
- **Knowledge health scanner** (`knowledge_health.go`) — detects stale artifacts
- **Filesystem-as-data-source pattern** — `reflect-suggestions.json`, `events.jsonl`, `focus.json` all serve dashboard via file reads
- **Dashboard API pattern** — `serve_learn.go:handleReflect()` reads JSON file, serves to dashboard

The production pipeline already exists in pieces. What's missing is the *product formatting* layer that converts internal signals into reading products.

**Source:** `cmd/orch/serve_learn.go:117-212`, `cmd/orch/serve.go:335-336`, `pkg/daemon/periodic.go`

**Significance:** Implementation can compose existing mechanisms rather than building from scratch. The daemon periodic task system + filesystem storage + dashboard API handler is a proven pattern.

---

### Finding 3: Dashboard architecture supports a new tab cleanly

**Evidence:** The dashboard has 4 top-level routes:
- `/` (Dashboard) — agent grid, beads stats, review queue
- `/work-graph` — DAG visualization
- `/knowledge-tree` — KB artifact browser
- `/harness` — pipeline gate metrics

Navigation is in `web/src/routes/+layout.svelte:60-81` — simple `<a>` links with active state detection. Adding a 5th route requires ~8 lines of navigation code.

Data patterns are well-established:
- SSE for high-frequency (agents) — not needed for digest
- Polling (60s) for stable data (beads, harness, verification) — appropriate for digest
- Stores in `web/src/lib/stores/` — new `digest.ts` store follows existing patterns

**Source:** `web/src/routes/+layout.svelte:60-81`, `web/src/lib/stores/beads.ts`, `web/src/lib/stores/harness.ts`

**Significance:** The frontend integration is the lowest-risk part of this design. Existing patterns handle everything needed.

---

### Finding 4: Product storage must be operational, not knowledge

**Evidence:** Principle: Session Amnesia says state must externalize to files. Constraint: No local agent state prohibits projection DBs.

Products have a lifecycle (new→read→starred→archived) that doesn't fit git-committed `.kb/` files:
- `.kb/` artifacts are knowledge (long-lived, version-controlled, referenced by agents)
- Digest products are operational (ephemeral, consumed-and-dismissed, never referenced by agents)

The right precedent is `~/.orch/events.jsonl` — operational data stored outside the repo, served via API, never git-committed.

**Source:** Principle: Session Amnesia, Constraint: No local agent state, `~/.orch/events.jsonl` pattern

**Significance:** Storage decision: `~/.orch/digest/` directory with individual JSON files per product. Files enable per-product state updates without rewriting a monolithic file. Filesystem listing provides the query mechanism (no DB needed).

---

### Finding 5: Quality gate needs artifact-level change detection

**Evidence:** Not everything the daemon touches is worth surfacing. Current rates:
- Model probes: 140+ across 26 models — most are confirmatory, not surprising
- Threads: 24 — but entries added ~daily, only some represent breakthroughs
- Investigations: 1,166 — 85.5% orphaned, most are narrow/tactical
- Reflect synthesis: clusters of 5+ investigations — these are already high-signal

Effective quality gate needs artifact-level change tracking:
- **Thread delta detection:** Compare current thread file hash/size to last scan. New content > 200 words triggers product.
- **Model update detection:** Compare model.md hash to last scan. Any change triggers product. New probes with "contradicts" finding trigger high-significance product.
- **Investigation completion:** Status: Complete + Recommendations section triggers decision brief product.
- **Pattern detection:** 3+ of same defect class in 7 days triggers alert product.

**Source:** `.kb/models/knowledge-accretion/model.md` (orphan rate data), `pkg/daemon/reflect.go` (synthesis cluster thresholds)

**Significance:** The scan-and-diff approach avoids expensive file parsing on every cycle. Store last-seen file hashes in `~/.orch/digest-state.json` (operational state, not agent state — it's daemon's own bookkeeping).

---

## Synthesis

**Key Insights:**

1. **Operational, not knowledge** — Digest products are ephemeral notifications about knowledge changes, not knowledge artifacts themselves. They live in `~/.orch/digest/` and expire, while the underlying `.kb/` artifacts they reference are permanent.

2. **Compose, don't build** — The production pipeline already exists: daemon periodic tasks + filesystem storage + dashboard API. The new work is a product formatting layer, not new infrastructure.

3. **Products are pointers, not duplicates** — Each product contains a summary + pointer to the source artifact (thread path, model path, investigation path). The dashboard renders the summary and offers a link to read the full artifact via the existing `/api/file` endpoint.

**Answer to Investigation Question:**

The daemon should run a periodic "digest producer" task (every 30 minutes) that:
1. Scans `.kb/threads/`, `.kb/models/`, `.kb/investigations/` for changes since last scan
2. Applies significance threshold per artifact type
3. Writes product files to `~/.orch/digest/{timestamp}-{type}-{slug}.json`
4. Dashboard serves these via `/api/digest` endpoint and renders in a "Thinking" tab
5. Dylan marks products read/starred/archived via PATCH endpoint
6. Products auto-archive after 7 days if unread

---

## Structured Uncertainty

**What's tested:**

- ✅ Dashboard navigation supports new tabs (verified: read layout.svelte, 4 existing routes)
- ✅ Filesystem-as-API pattern works (verified: reflect-suggestions.json → /api/reflect)
- ✅ Daemon periodic tasks are composable (verified: read periodic.go, reflect.go)
- ✅ Frontend store patterns are replicable (verified: read beads.ts, harness.ts stores)

**What's untested:**

- ⚠️ 30-minute scan interval is appropriate (may need tuning based on artifact change frequency)
- ⚠️ File hash comparison is sufficient for thread delta detection (content-aware diff might be needed)
- ⚠️ Star rate as quality feedback signal works (Dylan may not star consistently)
- ⚠️ Product volume per day — could be 2-3 or 20+ depending on daemon activity level

**What would change this:**

- If product volume exceeds ~10/day, significance thresholds need to be stricter or daily digest rollup is needed
- If Dylan primarily reads on phone, the dashboard UI needs responsive design priority (current dashboard is desktop-optimized)
- If the "no local agent state" constraint is interpreted to prohibit `~/.orch/digest-state.json`, the hash tracking approach needs redesign

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Filesystem-backed digest in `~/.orch/digest/` | architectural | Cross-component (daemon + API + dashboard), establishes new operational data pattern |
| Dashboard "Thinking" tab | implementation | Follows existing tab pattern, no cross-boundary impact |
| Periodic digest producer daemon task | implementation | Extends existing periodic.go pattern within daemon scope |
| Quality gate with feedback loop | strategic | Dylan's reading behavior shapes system behavior — value judgment about what matters |

### Recommended Approach ⭐

**Filesystem-Backed Digest with Dashboard UI** — The daemon runs a periodic digest producer that scans KB artifacts for changes, packages notable changes as product files in `~/.orch/digest/`, and the dashboard serves these through a new "Thinking" tab.

**Why this approach:**
- Composes existing infrastructure (periodic tasks + filesystem storage + API handlers) — minimal new code
- Respects "no local agent state" — digest files are daemon's own operational output, not a projection of external state
- Works async by design — products accumulate as files, independent of dashboard or orchestrator
- Phone-accessible — dashboard works in any browser

**Trade-offs accepted:**
- 30-minute latency between artifact change and product creation (acceptable for async consumption)
- File-per-product creates many small files (acceptable: cleanup via auto-archive; similar to events.jsonl growth)
- No real-time push notifications in v1 (add desktop notifications in v2)

### Data Model

#### Product File

`~/.orch/digest/{id}.json`

```json
{
  "id": "20260314T1030-thread-polycentric-governance",
  "type": "thread_progression",
  "title": "Thread: Polycentric Governance — new entry on Ostrom's commons principles",
  "summary": "The polycentric governance thread gained 300 words exploring how Ostrom's 8 principles for commons management map to agent coordination. Key finding: monitoring (principle 4) maps to the coaching plugin pattern.",
  "significance": "high",
  "source": {
    "artifact_type": "thread",
    "path": ".kb/threads/2026-03-13-polycentric-governance-commons-ostrom-as.md",
    "change_type": "content_added",
    "delta_words": 300
  },
  "state": "new",
  "created_at": "2026-03-14T10:30:00Z",
  "read_at": null,
  "starred_at": null,
  "archived_at": null,
  "tags": ["governance", "coordination", "ostrom"]
}
```

#### Product Types

| Type | Source | Trigger | Significance |
|------|--------|---------|-------------|
| `thread_progression` | `.kb/threads/*.md` | Content changed, delta > 200 words | medium (high if thread resolved) |
| `model_update` | `.kb/models/*/model.md` | Model.md content changed | medium (high if contradicts finding) |
| `model_probe` | `.kb/models/*/probes/*.md` | New probe file created | low (high if contradicts) |
| `decision_brief` | `.kb/investigations/*` | Status: Complete + has Recommendations | high |
| `pattern_alert` | Reflect data | 3+ same defect class in 7 days | high |
| `synthesis_cluster` | Reflect data | New cluster with 10+ investigations | medium |
| `weekly_digest` | All of above | Sunday midnight rollup | medium |

#### Digest State File

`~/.orch/digest-state.json` — Tracks what's been scanned to avoid duplicate products.

```json
{
  "last_scan": "2026-03-14T10:00:00Z",
  "file_hashes": {
    ".kb/threads/2026-03-13-polycentric-governance.md": "sha256:abc123...",
    ".kb/models/daemon-autonomous-operation/model.md": "sha256:def456..."
  },
  "stats": {
    "total_produced": 47,
    "total_read": 32,
    "total_starred": 8,
    "read_rate_by_type": {
      "thread_progression": 0.85,
      "model_update": 0.60,
      "decision_brief": 0.95,
      "pattern_alert": 0.70
    }
  }
}
```

### API Design

```
GET  /api/digest?state=new&type=thread_progression&limit=20
     → { products: [...], unread_count: 5, total: 47 }

GET  /api/digest/stats
     → { unread: 5, read: 32, starred: 8, by_type: {...}, read_rate: 0.68 }

PATCH /api/digest/:id
     body: { state: "read" | "starred" | "archived" }
     → { ok: true }

POST /api/digest/archive-read?older_than=7d
     → { archived: 12 }
```

### Dashboard Wireframe

**New navigation tab:** `Thinking` (between Harness and nothing — 5th tab)

**Page layout:**

```
┌──────────────────────────────────────────────────────────┐
│ 🐝 Swarm  Dashboard  Work Graph  KB  Harness  Thinking  │
│                                              ●3 unread   │
├──────────────────────────────────────────────────────────┤
│  [All] [Threads] [Models] [Decisions] [Alerts]           │
│  Significance: [All ▾]    Sort: [Newest ▾]               │
├──────────────────────────────────────────────────────────┤
│                                                          │
│  ● Thread: Polycentric Governance — new entry     HIGH   │
│    Explores Ostrom's 8 principles mapped to agent        │
│    coordination. Monitoring = coaching plugin.            │
│    .kb/threads/2026-03-13-polycentric-governance.md      │
│    2h ago                            [★ Star] [Archive]  │
│  ─────────────────────────────────────────────────────── │
│  ○ Model: daemon-autonomous-operation updated     MED    │
│    Probe contradicted claim about spawn dedup L3         │
│    behavior under high concurrency. Model corrected.     │
│    .kb/models/daemon-autonomous-operation/model.md       │
│    5h ago                            [★ Star] [Archive]  │
│  ─────────────────────────────────────────────────────── │
│  ○ Decision Brief: Auth middleware redesign       HIGH   │
│    Investigation complete with recommendation:           │
│    session-based → JWT migration in 3 phases.            │
│    .kb/investigations/2026-03-14-design-auth.md          │
│    Yesterday                         [★ Star] [Archive]  │
│                                                          │
│  ── Read ──────────────────────────────────────────────  │
│  ✓ Pattern Alert: Defect Class 2 recurring    3d ago     │
│  ✓ Thread: Harness Engineering resolved       4d ago     │
│                                                          │
└──────────────────────────────────────────────────────────┘
```

**Interaction model:**
- ● = unread, ○ = read, ✓ = archived
- Clicking product summary expands to show full source content (fetched via `/api/file`)
- Auto-marks as "read" on expand
- Star button saves to starred state (never auto-archives)
- Archive button dismisses immediately
- Unread count shown in navigation badge

### Implementation Phases

#### Phase 1: Digest Producer + API (daemon + backend)

**Scope:** Daemon periodic task + filesystem storage + API endpoints

**Files to create/modify:**
- `pkg/daemon/digest.go` — DigestProducer struct, scan logic, product creation
- `pkg/daemon/digest_test.go` — Unit tests
- `cmd/orch/serve_digest.go` — API handlers for /api/digest endpoints
- `cmd/orch/serve.go` — Register new endpoints

**Key implementation details:**
- DigestProducer runs as periodic task (30m interval, configurable)
- Scans .kb/threads/, .kb/models/, .kb/investigations/ for file changes
- Compares SHA256 hashes against digest-state.json
- Creates product files in ~/.orch/digest/
- API handlers read/list/update product files

**Acceptance criteria:**
- `orch daemon run` produces digest products when KB artifacts change
- `/api/digest` returns product list with filtering
- `PATCH /api/digest/:id` updates product state
- Products auto-archive after 7 days

#### Phase 2: Dashboard Thinking Tab (frontend)

**Scope:** New SvelteKit route + Svelte store + components

**Files to create:**
- `web/src/routes/thinking/+page.svelte` — Page component
- `web/src/lib/stores/digest.ts` — Digest store (polling, 60s)
- `web/src/lib/components/digest-card/digest-card.svelte` — Product card component

**Files to modify:**
- `web/src/routes/+layout.svelte` — Add "Thinking" nav link with unread badge

**Key implementation details:**
- Store polls `/api/digest?state=new` every 60s
- Product cards show summary, significance badge, source path, action buttons
- Clicking card expands and fetches full source via `/api/file`
- Star/Archive actions call `PATCH /api/digest/:id`
- Unread badge in nav shows count from `/api/digest/stats`

**Acceptance criteria:**
- "Thinking" tab shows in dashboard navigation
- Products display with correct state indicators
- Star/Archive actions persist across page refresh
- Unread badge updates when products are read

#### Phase 3: Quality Gate + Feedback Loop (refinement)

**Scope:** Significance threshold tuning based on read/star behavior

**Files to modify:**
- `pkg/daemon/digest.go` — Add adaptive threshold logic

**Key implementation details:**
- Track read_rate and star_rate by product type in digest-state.json
- After 2 weeks, adjust significance thresholds:
  - Types with < 20% read rate → raise minimum delta threshold
  - Types with > 80% star rate → lower threshold (surface more)
- Weekly digest product rolls up the week's products into one summary

**Acceptance criteria:**
- Read/star rates tracked per product type
- Significance thresholds adapt over time
- Weekly digest summarizes the week

### Defect Class Exposure

| Code Path | Defect Class | Mitigation |
|-----------|-------------|------------|
| Digest scan → product creation | Class 6: Duplicate Action | Hash-based dedup in digest-state.json — same file hash never produces duplicate product |
| Product files in ~/.orch/digest/ | Class 3: Stale Artifact Accumulation | Auto-archive after 7 days; `archive-read` endpoint for batch cleanup |
| Multi-project scanning | Class 4: Cross-Project Boundary Bleed | Thread projectDir explicitly in scan paths — only scan current project's .kb/ |
| Digest state persistence | Class 5: Contradictory Authority Signals | Single digest-state.json as canonical source — no secondary caches |

### Alternative Approaches Considered

**Option B: Beads issues as reading queue**
- **Pros:** Already queryable, fits existing workflow, no new storage
- **Cons:** Mixes notifications with work items; reading queue lifecycle (read/archive) is fundamentally different from issue lifecycle (open/in-progress/closed); would pollute beads with non-actionable items
- **When to use instead:** For actionable items only (decision briefs that need Dylan's approval)

**Option C: Real-time SSE notification system**
- **Pros:** Immediate, no polling delay, desktop notifications
- **Cons:** Over-engineered for v1; Dylan reads on his schedule, not real-time; adds complexity to SSE infrastructure
- **When to use instead:** Phase 2+ enhancement for urgent pattern alerts only

**Option D: Email/Slack digest**
- **Pros:** Reaches Dylan anywhere, push-based
- **Cons:** Requires external service integration; overkill for single-user system; adds dependency
- **When to use instead:** If Dylan wants daily email digest — can be added as exporter from /api/digest

**Rationale for recommendation:** Option A (filesystem + dashboard) composes existing patterns, requires the least new infrastructure, works async by design, and avoids coupling to external services. The tradeoff of 30-minute latency is acceptable because Dylan reads on his schedule anyway.

---

### Implementation Details

**What to implement first:**
- Phase 1 (digest producer + API) is foundational — everything else depends on it
- Within Phase 1, start with thread_progression products (highest value to Dylan, simplest scan logic)
- Add model_update and decision_brief in same phase

**Things to watch out for:**
- ⚠️ File system scanning could be slow with 1,166+ investigations — use directory-level mtime check before individual file hashes
- ⚠️ ~/.orch/digest/ directory will grow — implement cleanup from day 1 (auto-archive + periodic purge of archived files > 30 days)
- ⚠️ Product summaries need to be genuinely useful, not just "file changed" — extract first 2-3 meaningful sentences from delta content

**Areas needing further investigation:**
- Mobile responsiveness of dashboard (current dashboard is desktop-optimized)
- Whether thread delta detection needs content-aware diff vs simple hash comparison
- Notification mechanism for urgent pattern alerts (desktop notification via pkg/notify/)

**Success criteria:**
- ✅ Dylan can see what the daemon learned without running any CLI command
- ✅ Products accumulate while Dylan is away and are waiting when he returns
- ✅ Star/archive feedback gives the system signal about what's valuable
- ✅ No new dependencies (composes existing daemon + filesystem + dashboard patterns)

---

## References

**Files Examined:**
- `pkg/daemon/reflect.go` — Periodic kb reflect integration (392 lines)
- `pkg/daemon/synthesis_auto_create.go` — Auto-create synthesis issues (264 lines)
- `pkg/daemon/knowledge_health.go` — Stale artifact detection
- `pkg/daemon/periodic.go` — Periodic task scheduler
- `cmd/orch/serve.go` — Dashboard API server setup (524 lines)
- `cmd/orch/serve_learn.go` — Reflect + gaps API handlers (212 lines)
- `cmd/orch/serve_system.go` — System endpoints including /api/file (539 lines)
- `cmd/orch/serve_beads.go` — Beads API handlers (747 lines)
- `web/src/routes/+layout.svelte` — Dashboard navigation (157 lines)
- `web/src/lib/stores/beads.ts` — Store polling pattern
- `web/src/lib/stores/harness.ts` — Store polling pattern
- `.kb/threads/` — 24 strategic thinking threads
- `.kb/models/` — 26 models with 140+ probes
- `~/.kb/principles.md` — Foundational principles (Session Amnesia, Provenance, Evidence Hierarchy)

**Related Artifacts:**
- **Decision:** `.kb/decisions/2026-02-25-continuous-knowledge-maintenance.md` — Auto-surface synthesis
- **Model:** `.kb/models/knowledge-accretion/model.md` — Orphan rate data, synthesis patterns
- **Model:** `.kb/models/dashboard-architecture/model.md` — Dashboard API patterns

---

## Investigation History

**2026-03-14 10:00:** Investigation started
- Initial question: How should the daemon package its thinking as consumable output for Dylan?
- Context: Subproblem 2 from daemon autonomous trigger layer architect task

**2026-03-14 10:30:** Codebase exploration complete
- Explored daemon package (120 files), dashboard API (40+ endpoints), frontend architecture (SvelteKit + stores)
- Key finding: infrastructure exists in pieces, needs product formatting layer

**2026-03-14 11:00:** Design forks navigated
- 6 decision forks resolved: product types, storage, production mechanism, consumption model, lifecycle, quality gate
- Substrate consulted: Session Amnesia principle, No Local Agent State constraint, existing filesystem-as-API patterns

**2026-03-14 11:30:** Investigation completed
- Status: Complete
- Key outcome: Filesystem-backed digest system with dashboard "Thinking" tab, 3-phase implementation plan
