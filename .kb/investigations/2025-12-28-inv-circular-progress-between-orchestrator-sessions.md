# Investigation: Circular Progress Between Orchestrator Sessions

## Summary (D.E.K.N.)

**Delta:** Two orchestrator sessions (A: ses_4996, B: ses_4994) exhibited circular debugging where Session B re-discovered and re-fixed the same root cause (stale binary) that Session A had found but Session B was unaware of due to running the stale binary itself.

**Evidence:** 
- Session A fixed `serve.go` at 12:23:58 (commit d948e5d6)
- Session B started at 12:43:14 with the OLD binary (c06db83c missing d948e5d6)
- Session B spent ~30 minutes debugging "sessions died silently" before discovering it was using stale binary
- Session B's fix (auto-rebuild) at 13:14:24 would have prevented Session B's own confusion

**Knowledge:** The stale binary problem is self-hiding: when you run `orch` with a stale binary, you can't see the fix that would show you the problem exists. This creates circular debugging patterns where sessions solve the same problem repeatedly.

**Next:** The auto-rebuild fix (f0d8b823) is now in place. Consider adding a "session overlap warning" when multiple orchestrator sessions exist in the same project.

---

# Investigation: Circular Progress Between Orchestrator Sessions

**Question:** What caused circular progress between sessions A (ses_4996) and B (ses_4994), what should have been fixed in what sequence, and where did the confusion originate?

**Status:** Complete ✅

## Timeline Reconstruction

### Session A (ses_4996): 12:05-12:37

| Time | Event | Outcome |
|------|-------|---------|
| 12:05 | Session started | Investigating spawned agent not visible |
| 12:10 | Discovered `orch serve` calls `ListSessions("")` | Root cause identified |
| 12:23:58 | **Committed fix (d948e5d6)** | `serve.go` now queries per-directory |
| 12:27 | Committed port fix (cadeb6fb) | 3333→3348 in orchestrator skill |
| 12:29 | Updated SYNTHESIS | Stale build artifact cleanup noted |
| 12:33 | Documented feature gaps | Investigation file committed |
| 12:37 | Session ended | **Did NOT run `make install`** |

### Session B (ses_4994): 12:43-13:18

| Time | Event | Binary State |
|------|-------|--------------|
| 12:43:14 | Session started | **Stale** - missing d948e5d6 |
| ~12:50 | "Are agents running?" - Mixed picture | Stale binary couldn't see them |
| ~12:55 | "Sessions died silently" - Wrong conclusion | Actually visible, binary just old |
| ~13:00 | Direct investigation authorized | Spent 20+ min on wrong path |
| ~13:05 | Discovery: `go run` shows 6, `orch` shows 0 | **Eureka moment** |
| 13:10 | Root cause confirmed: stale binary | Same root cause as Session A! |
| 13:14:24 | **Fixed: Auto-rebuild feature (f0d8b823)** | Would have prevented own confusion |

## The Circular Pattern

```
Session A fixes serve.go → Doesn't rebuild installed binary
                              ↓
Session B starts with stale binary → Can't see agents
                              ↓
Session B investigates "dead sessions" for 30 min
                              ↓
Session B discovers it's the same stale binary problem
                              ↓
Session B implements auto-rebuild → Prevents future recurrence
```

## What Should Have Been Fixed, In What Sequence

### Ideal Sequence

1. **First:** Auto-rebuild mechanism (so stale binaries never persist)
2. **Second:** Multi-directory session query (serve.go fix)
3. **Third:** Documentation and feature gap issues

### What Actually Happened

1. Session A fixed serve.go but didn't ensure the fix was deployed
2. Session B inherited a stale binary and couldn't benefit from Session A's fix
3. Session B spent significant time debugging a phantom problem
4. Session B implemented auto-rebuild (which should have been first)

## Where Confusion Originated

### Primary Origin: No "Push to Remote" Mental Model for Local Development

Session A committed the fix but:
- Did not run `make install` after committing
- Did not verify the fix was deployed
- Session transcript shows "commits ready to push" but no deployment verification

### Secondary Origin: Silent Stale Binary

The stale binary problem is **self-hiding**:
- Running `orch status` with stale binary → shows 0 agents
- Running `go run ./cmd/orch status` with source → shows correct agents
- But you only think to do the second when you already suspect staleness

### Tertiary Origin: No Session Overlap Detection

Both sessions ran in the same project during overlapping time windows:
- Session A: 12:05-12:37
- Session B: 12:43-13:18 (started 6 minutes after A ended)

No mechanism warned Session B that:
- Another session had just made changes
- Those changes weren't in the installed binary
- Session A's fix was available but not deployed

## Findings

### Finding 1: Commit ≠ Deploy

Session A's transcript shows clear completion:
- "Committed fix: query OpenCode sessions per-project-directory"
- "The fix is working"
- Dashboard screenshot confirmed agents visible

But the fix only worked because Session A was likely running `go run` during development, not the installed binary.

### Finding 2: Session B Wasted ~30 Minutes

Timeline from Session B:
- 12:43 → 13:05 (~22 minutes) spent on wrong hypothesis ("sessions died silently")
- Would have been 0 minutes if binary was current

### Finding 3: The Meta-Irony Was Recognized

Session B explicitly noted: "The irony is perfect - we just lived through exactly the problem we're trying to solve."

The design session and stale binary issues were spawned BEFORE Session B discovered it was running stale itself.

### Finding 4: Dashboard vs CLI Mismatch Persisted

Even after both fixes:
- Session A: Fixed serve.go (per-directory queries)
- Session B: Added auto-rebuild

Session B still saw dashboard/status mismatch at the end and spawned yet another investigation. The root cause may have layers.

## Test Performed

**Test:** Reconstructed timeline from git log and session transcripts
**Result:** Confirmed circular pattern:
- d948e5d6 (Session A's fix) committed at 12:23:58
- Session B started at 12:43:14 with pre-fix binary
- Session B's revelation at ~13:05 matched exactly when they tried `go run` vs installed binary

## Conclusion

The circular progress was caused by **stale binary inheritance across sessions**. Session A fixed a visibility problem but didn't deploy the fix. Session B started 6 minutes later using the old binary and couldn't see agents, leading to 30 minutes of debugging the wrong problem.

The auto-rebuild mechanism (f0d8b823) is the correct fix and is now in place. However, three process gaps remain:

1. **Pre-commit hook exists but...** - The project has a pre-commit hook that rebuilds, but it didn't prevent Session B from running stale. This suggests Session B may have started before Session A committed, or the hook doesn't cover all paths.

2. **No session handoff protocol** - When Session A ended, there was no mechanism to ensure Session B inherited current state.

3. **Dashboard still has issues** - Session B spawned another investigation at the end (orch-go-bgf5), suggesting dashboard/CLI mismatch has additional causes.

## Self-Review

- [x] Real test performed (timeline reconstruction from git/transcripts)
- [x] Conclusion from evidence (specific commits, timestamps matched)
- [x] Question answered (identified sequence, confusion origin)
- [x] File complete (all sections filled)

**Self-Review Status:** PASSED

## Discovered Work

1. **Potential issue:** Pre-commit hook didn't prevent Session B staleness - may need investigation
2. **Potential issue:** Dashboard/CLI mismatch spawned orch-go-bgf5 - check if resolved
3. **Process gap:** No session overlap warning mechanism exists
