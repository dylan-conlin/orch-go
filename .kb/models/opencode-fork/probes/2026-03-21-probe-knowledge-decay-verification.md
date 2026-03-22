# Probe: Knowledge Decay Verification — OpenCode Fork

**Date:** 2026-03-21
**Trigger:** 30 days since last probe (2026-02-20)
**Model:** opencode-fork/model.md (last updated 2026-03-06)

---

## Claims Verified

### 1. Custom Commit Count: **STALE**
- **Model claims:** 16 custom commits ahead of upstream
- **Actual:** 32 custom commits ahead of `upstream/dev`
- **What happened:** A major rebase (commit b6c4605d5, ~2026-02-25) incorporated 268 upstream commits while preserving 22 custom commits. Since then, 10 more custom commits added (OpenAPI fixes, GitHub Actions cleanup, etc.)
- **Impact:** The "16 commits" figure is misleading. The rebase concern about "any more and rebasing becomes painful" is now directly relevant at 32.

### 2. File Paths: **STALE (major restructure)**
- **Model claims:** Key files at flat paths like `src/instance.ts`, `src/server.ts`, `src/session.ts`, `src/session.sql.ts`
- **Actual paths (post-upstream rebase):**
  - `src/instance.ts` → `src/project/instance.ts`
  - `src/server.ts` → `src/server/server.ts`
  - `src/session.ts` → split into: `src/session/index.ts`, `src/server/routes/session.ts`, `src/cli/cmd/session.ts`, `src/acp/session.ts`
  - `src/session.sql.ts` → `src/session/session.sql.ts`
  - TTL cleanup extracted to `src/session/cleanup.ts`
- **Impact:** All References section paths in the model are broken. Anyone following them will get "file not found."

### 3. Instance LRU/TTL Eviction: **CONFIRMED**
- `src/project/instance.ts` is exactly 350 lines (matches model claim)
- LRU/TTL eviction logic intact

### 4. ORCH_WORKER Server-Side Code: **CORRECTED (no longer lost)**
- **Model claims:** "corresponding server-side code was not found in the current codebase — may have been lost during an upstream rebase"
- **Actual:** Server-side code EXISTS at `src/server/routes/session.ts:245-249`. Commits 8f806313e and afa30a639 re-implemented the ORCH_WORKER header reading and metadata.role setting.
- **Impact:** The "lost code" caveat is no longer accurate — this was fixed post-model-creation.

### 5. Session TTL Cleanup: **CONFIRMED (restructured)**
- TTL cleanup still runs every 5 minutes, now in dedicated `src/session/cleanup.ts` (extracted from monolithic session.ts)
- Functional behavior matches model description

### 6. Session Metadata API: **CONFIRMED**
- Metadata accepted in POST (create) and PATCH (update) routes
- `Session.setMetadata()` used for updates
- Metadata surfaced in session list responses

### 7. Monorepo Structure: **STALE (incomplete)**
- Model lists 8 packages. Actual has ~18 packages including: `script`, `function`, `identity`, `extensions`, `console`, `docs`, `tauri`, `slack`, `enterprise`, `containers`
- Not critical for orch-go integration but the model's structure diagram is incomplete

### 8. Remote Setup: **MINOR STALE**
- Model says `fork → git@github.com:user/opencode.git`
- Actual: `fork → git@github.com:dylan-conlin/opencode.git`
- Cosmetic inaccuracy

### 9. Upstream Sync Status: **STALE**
- Model says "Last upstream fetch: Feb 5, 2026"
- Actual: A full rebase onto upstream/dev incorporating 268 commits occurred ~Feb 25, 2026
- The fork is now much more current with upstream

### 10. GET /session/status Integration: **STILL PENDING**
- Model correctly notes orch-go integration is pending
- Still pending as of this probe

---

## Verdict

**Model is PARTIALLY STALE.** Core functional claims (TTL, metadata, LRU eviction, status endpoint) are confirmed. But:

1. **File paths are completely wrong** — upstream restructured into subdirectories
2. **Custom commit count doubled** (16 → 32) — rebase pain threshold concern is now active
3. **ORCH_WORKER "lost code" caveat is resolved** — server-side code was re-implemented
4. **Upstream sync is much more current** than model states

**Recommended model update:** Medium — fix all file paths in References, update commit count, remove "lost code" caveat, update sync status, note the session module extraction.
