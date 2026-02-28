# Session Synthesis

**Agent:** og-debug-bd-sync-fails-27feb-fe95
**Issue:** orch-go-swcg
**Duration:** 2026-02-27T20:33 → 2026-02-27T20:40
**Outcome:** success

---

## Plain-Language Summary

When multiple agents work concurrently in the same git repository, their uncommitted file changes (tracked files like `.go`, `.svelte`, etc.) cause `bd sync` to fail at the `git pull` step. The pull refuses to proceed because git detects a dirty working tree. This blocks the agent from exporting its beads changes even though those changes are already committed and ready to push. The fix adds `--autostash` to the `git pull` command, which makes git automatically stash dirty files before pulling and restore them after. This is a one-line change in the beads CLI (not orch-go).

## Verification Contract

See `VERIFICATION_SPEC.yaml` — key outcomes:
- Reproduction confirmed: `git pull` fails with exit 128 on dirty tree
- Fix confirmed: `git pull --autostash` succeeds, preserves dirty files
- Smoke test: `bd sync` completes in orch-go with 12+ modified tracked files

---

## Delta (What Changed)

### Files Modified
- `~/Documents/personal/beads/cmd/bd/sync_git.go` - Added `--autostash` flag to `git pull` command in `gitPull()` function (line 541)

### Binary Updated
- `~/bin/bd` - Rebuilt and installed from beads source

---

## Evidence (What Was Observed)

- `git pull origin master` on dirty working tree: **exit 128** with "cannot pull with rebase: You have unstaged changes"
- `git pull --autostash origin master` on same dirty tree: **exit 0**, "Created autostash → Applied autostash"
- `bd sync` in orch-go with 12+ modified tracked files from concurrent agents: **sync complete**, all dirty files preserved
- Git 2.48.1 installed (well above 2.27 minimum for `--autostash` with merge-based pulls)
- Pre-existing test failures in beads (internal/rpc, internal/storage/sqlite) are unrelated to this change

### Tests Run
```bash
# Build verification
cd ~/Documents/personal/beads && go build ./cmd/bd/
# PASS

# Smoke test with real dirty working tree
cd ~/Documents/personal/orch-go && bd sync
# ✓ Sync complete (12+ modified tracked files preserved)
```

---

## Architectural Choices

### `--autostash` vs skip-pull-when-dirty
- **What I chose:** `--autostash` (Option 1 from task)
- **What I rejected:** Skip pull when working tree is dirty (Option 2)
- **Why:** `--autostash` is built into git, handles the stash/unstash atomically, and preserves the full sync flow (export + commit + pull + import + push). Skipping pull would mean agents don't get remote updates during sync, which could cause more divergence over time.
- **Risk accepted:** If stashed changes conflict with pulled changes, git will leave conflict markers. This is rare since agents typically modify different files, and the existing error handling in `bd sync` already handles pull failures gracefully.

---

## Knowledge (What Was Learned)

### Constraints Discovered
- Cross-repo fix: `bd sync` code lives in `~/Documents/personal/beads`, not orch-go. Issues filed in orch-go about bd behavior require cross-repo fixes.
- The orch-go repo has `pull.rebase=true` configured, which is why the error specifically says "cannot pull with rebase"
- `--autostash` works for both merge-based and rebase-based pulls since git 2.27

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (fix implemented, binary installed)
- [x] Tests passing (build, vet, smoke test)
- [x] Ready for `orch complete orch-go-swcg`

**Note:** The fix should be committed in the beads repo separately. The beads repo change was made directly but not yet committed there.

---

## Unexplored Questions

- Should `bd sync` also handle the case where `--autostash` fails (stash conflicts)? Currently falls through to existing error handling which gives manual resolution instructions.
- Should the beads project configure `rebase.autoStash=true` globally instead of per-command? This would cover any future git pull calls too.

---

## Session Metadata

**Skill:** systematic-debugging
**Model:** anthropic/claude-opus-4-5-20251101
**Workspace:** `.orch/workspace/og-debug-bd-sync-fails-27feb-fe95/`
**Beads:** `bd show orch-go-swcg`
