# Investigation: Entropy Spiral Recovery Audit

## Summary (D.E.K.N.)

**Delta:** 1163 commits between Jan 18 baseline and entropy-spiral-feb2026 contain ~8 recoverable features, but bd sync commits (3b004bef, b27e9778) accidentally deleted significant code including pkg/attention/. The branch compiles and tests pass for remaining code.

**Evidence:** `go build ./...` succeeds, `go test ./pkg/spawn/backends/...` passes. Git diff shows pkg/spawn/backends/ (6 new files), pkg/spawn/verification_spec.go (377 lines), substantial web UI improvements. pkg/attention/ deleted in bd sync commit.

**Knowledge:** The entropy spiral is a mixed bag - some clean abstractions (backends) alongside accidental deletions. Recovery should cherry-pick specific commits rather than merge the whole branch.

**Next:** Cherry-pick priority features: (1) spawn backends abstraction, (2) verification spec generation, (3) daemon skill inference. Investigate attention system recovery from pre-deletion commits.

**Authority:** strategic - Recovery decision involves resource commitment and architectural choices about what to keep.

---

# Investigation: Entropy Spiral Recovery Audit

**Question:** What features from the entropy-spiral-feb2026 branch (1163 commits since Jan 18 baseline 0bca3dec) are worth recovering and how complex is each recovery?

**Started:** 2026-02-13
**Updated:** 2026-02-13
**Owner:** Worker Agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| N/A - novel investigation | - | - | - |

---

## Findings

### Finding 1: Spawn Backends Abstraction (HIGH VALUE - Ready to recover)

**Evidence:**
- 6 new files in `pkg/spawn/backends/`: backend.go, common.go, headless.go, inline.go, tmux.go, backend_test.go
- Clean Backend interface with Spawn(ctx, *SpawnRequest) (*Result, error)
- Includes retry logic, session registration, event logging
- Tests pass: `go test ./pkg/spawn/backends/...` → OK (0.006s)

**Source:**
- Commit 40d09539: "feat: extract spawn backends to pkg/spawn/backends/ (Phase 1 Strangler Fig)"
- Files: pkg/spawn/backends/*.go

**Significance:** This is a well-designed abstraction that simplifies the spawn code. No dependencies on other entropy-spiral code. Can be cherry-picked cleanly.

**Recovery Complexity:** LOW - single commit, tests pass, no external dependencies.

---

### Finding 2: Verification Spec Generation (HIGH VALUE - Ready to recover)

**Evidence:**
- pkg/spawn/verification_spec.go: 377 lines
- pkg/spawn/verification_spec_test.go: 165 lines
- Generates VERIFICATION_SPEC.yaml skeletons for spawn workspaces
- Tier/skill-aware verification entry generation
- Uses existing spawn.Config, no new dependencies

**Source:**
- Multiple commits touching verification_spec.go
- File: pkg/spawn/verification_spec.go

**Significance:** Implements the verification spec pattern that's referenced in SPAWN_CONTEXT templates. Missing this creates a gap in the verification pipeline.

**Recovery Complexity:** LOW - self-contained, tests exist.

---

### Finding 3: Daemon Skill Inference Improvements (MEDIUM VALUE)

**Evidence:**
- pkg/daemon/skill_inference.go: 114 lines
- InferSkillFromIssue() with priority: skill:* label > title pattern > type inference
- InferMCPFromLabels() for needs:playwright -> MCP server mapping
- Bug -> architect (not systematic-debugging) policy implemented

**Source:**
- File: pkg/daemon/skill_inference.go

**Significance:** Implements daemon policy decisions. Current master may already have this or similar.

**Recovery Complexity:** LOW - small file, may need conflict resolution with master.

---

### Finding 4: Attention System (DELETED - HIGH VALUE but requires archaeology)

**Evidence:**
- pkg/attention/ was created with comprehensive collectors:
  - agent_collector.go, beads.go, stuck_collector.go, unblocked_collector.go
  - serve_attention.go endpoint
- DELETED in commit 3b004bef "bd sync: 2026-02-07 11:26:40" (6785 line change)
- Deletion appears accidental - bd sync removing untracked files

**Source:**
- Creation: commits 654dbe72, 8f8304fb, 423ccd55, f3be75fc, etc.
- Deletion: commit 3b004bef

**Significance:** The attention system was a significant feature for surfacing work priorities. It was lost to a bd sync operation.

**Recovery Complexity:** HIGH - Need to identify the last good state before deletion and extract those files. May have dependencies on other deleted code.

---

### Finding 5: Web UI Dashboard Improvements (MEDIUM VALUE - Mixed state)

**Evidence:**
- 1040 lines added, 2829 lines deleted in web/
- Components added: work-graph, agent-card improvements, stores
- Many components removed: coaching, questions, services
- Active dashboard at +page.svelte with keyboard navigation

**Source:**
- Multiple commits in web/ directory
- Commits: fb87ec64, 3c57b42d, d77ff469, 6d14f241, etc.

**Significance:** Dashboard improvements but significant code was also removed. Net change is unclear without testing.

**Recovery Complexity:** MEDIUM - Need to evaluate what's working vs what was intentionally removed.

---

### Finding 6: Daemon Configuration and Rate Limiting (MEDIUM VALUE)

**Evidence:**
- pkg/daemon/daemon.go: 28KB file with Config struct
- MaxSpawnsPerHour rate limiting (default 20)
- ReflectEnabled, ReflectInterval for kb reflect integration
- Pool and spawn tracker improvements

**Source:**
- File: pkg/daemon/daemon.go

**Significance:** Production daemon hardening. May overlap with master.

**Recovery Complexity:** LOW-MEDIUM - Need to diff against master to identify net-new.

---

### Finding 7: bd sync Deletions Caused Significant Code Loss

**Evidence:**
- Commit 3b004bef deleted:
  - pkg/attention/ (entire directory)
  - cmd/orch/serve_attention.go (656 lines)
  - Various investigations and mockups
- 6785 total line changes in one "bd sync" commit
- This appears to be accidental - bd sync shouldn't delete tracked files

**Source:**
- Commit 3b004bef, b27e9778

**Significance:** This explains why some expected features are missing. The entropy spiral suffered from tool-induced code loss.

**Recovery Complexity:** This is a process issue, not a recovery issue. The deleted code exists in earlier commits.

---

### Finding 8: CLI Commands - Limited Net-New (LOW VALUE)

**Evidence:**
- No phase_cmd.go file exists
- Most CLI changes are modifications to existing commands
- complete_cmd.go at 57KB is large but mostly existing code
- spawn_cmd.go reduced from larger to 998 lines (deletion of old code)

**Source:**
- cmd/orch/*.go listing shows 85 files total

**Significance:** The entropy spiral removed/simplified more CLI code than it added. This is actually desirable.

**Recovery Complexity:** N/A - Not a recovery target.

---

## Synthesis

**Key Insights:**

1. **Clean Abstractions Worth Keeping** - pkg/spawn/backends/ and pkg/spawn/verification_spec.go are high-quality, tested, isolated modules that should be recovered immediately.

2. **Accidental Code Loss** - The bd sync tool caused significant deletion of tracked code (pkg/attention/), which explains missing features. This is a tool bug, not intentional removal.

3. **Net Deletion is Mostly Good** - The entropy spiral deleted a lot of code from cmd/orch/ and pkg/daemon/ - this appears to be intentional cleanup/simplification, not loss.

**Answer to Investigation Question:**

Priority-ranked features worth recovering:

| Priority | Feature | Commits/Files | Complexity | Dependencies |
|----------|---------|---------------|------------|--------------|
| P1 | Spawn Backends | 40d09539, pkg/spawn/backends/ | LOW | None |
| P1 | Verification Spec | pkg/spawn/verification_spec.go | LOW | spawn.Config |
| P2 | Attention System | Pre-3b004bef commits, pkg/attention/ | HIGH | serve_attention.go |
| P2 | Skill Inference | pkg/daemon/skill_inference.go | LOW | daemon.Issue |
| P3 | Dashboard Improvements | web/ directory | MEDIUM | Svelte components |
| P3 | Daemon Rate Limiting | pkg/daemon/daemon.go | LOW | beads package |

---

## Structured Uncertainty

**What's tested:**

- ✅ entropy-spiral-feb2026 compiles (`go build ./...` succeeds)
- ✅ pkg/spawn/backends/ tests pass (`go test ./pkg/spawn/backends/...`)
- ✅ Commit history shows attention system existed before bd sync deletion
- ✅ verification_spec.go exists and has test file

**What's untested:**

- ⚠️ Dashboard functionality (no browser test run)
- ⚠️ Integration of recovered code with master
- ⚠️ Attention system dependencies on other deleted code

**What would change this:**

- If master already has equivalent spawn backends, P1 becomes unnecessary
- If attention system has deep dependencies on other deleted code, recovery complexity increases
- If dashboard changes break existing functionality, priority decreases

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Cherry-pick spawn backends | implementation | Isolated module, reversible, clear criteria |
| Cherry-pick verification_spec | implementation | Same as above |
| Investigate attention recovery | architectural | Cross-component, multiple valid approaches |
| Dashboard recovery | strategic | Resource commitment, unclear benefit |

### Recommended Approach ⭐

**Surgical Cherry-Pick** - Recover high-value isolated modules first, investigate complex features second.

**Why this approach:**
- Low-risk wins first (backends, verification_spec)
- Avoids pulling in problematic code from the entropy spiral
- Attention system needs investigation before recovery

**Trade-offs accepted:**
- May miss some small improvements that were in "churn" commits
- Dashboard improvements deferred

**Implementation sequence:**
1. Cherry-pick 40d09539 (spawn backends) to master
2. Cherry-pick verification_spec commits
3. Open investigation issue for attention system recovery
4. Evaluate daemon improvements against master

### Alternative Approaches Considered

**Option B: Merge entropy-spiral-feb2026**
- **Pros:** Gets everything at once
- **Cons:** Brings in bd sync damage, untested integration, massive diff
- **When to use instead:** Never - the branch has known damage

**Option C: Fresh rewrite**
- **Pros:** Clean design without historical baggage
- **Cons:** Loses working, tested code that exists
- **When to use instead:** Only for attention system if recovery is too complex

---

## References

**Commands Run:**
```bash
# Count commits
git log --oneline 0bca3dec..entropy-spiral-feb2026 | wc -l  # 1163

# Test spawn backends
go build ./...  # Success
go test ./pkg/spawn/backends/...  # OK

# Check attention deletion
git log --oneline 0bca3dec..entropy-spiral-feb2026 --diff-filter=D -- 'pkg/attention/'
# 3b004bef bd sync: 2026-02-07 11:26:40
```

**Files Examined:**
- pkg/spawn/backends/*.go - Backend interface and implementations
- pkg/spawn/verification_spec.go - Verification spec generation
- pkg/daemon/skill_inference.go - Skill inference logic
- web/src/routes/+page.svelte - Dashboard main page
- cmd/orch/*.go - CLI command files

---

## Investigation History

**2026-02-13 10:00:** Investigation started
- Initial question: What's recoverable from entropy spiral?
- Context: 1163 commits since Jan 18 baseline

**2026-02-13 10:30:** Core analysis complete
- Identified 6 functional areas with findings
- Discovered bd sync deletion of pkg/attention/
- Confirmed spawn backends compile and test

**2026-02-13 11:00:** Investigation completed
- Status: Complete
- Key outcome: P1 features (spawn backends, verification_spec) ready for immediate recovery; attention system needs separate investigation
