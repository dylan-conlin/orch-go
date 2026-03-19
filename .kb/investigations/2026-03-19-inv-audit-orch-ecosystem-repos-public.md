# Audit: Orch Ecosystem Repos for Public Readiness

**TLDR:** orch-go needs significant scrubbing before public release — 1,041 tracked files contain `/Users/dylanconlin`, plus employer data (SCS/SendCutSend), personal emails, and 4,700-line gap-tracker with business context. harness is ready now. kb-cli and scrape need minor fixes (minutes each).

**Status:** Complete
**Date:** 2026-03-19
**Beads:** orch-go-gpy9d

## D.E.K.N. Summary

- **Delta:** Comprehensive 4-repo security audit with per-file findings and actionable remediation plan
- **Evidence:** `git ls-files | xargs grep` scans across all 4 repos, covering API keys, tokens, hardcoded paths, employer identifiers, email addresses, credential files
- **Knowledge:** The bulk of orch-go's exposure is in `.kb/` (1,952 tracked files) and `.orch/` archives (2,254 tracked files) — these are knowledge artifacts, not source code. Source code (cmd/, pkg/) has only 2 files with hardcoded paths. A strip-internal-refs approach (like kb-cli already has) would handle most of it.
- **Next:** Create scrubbing script for orch-go (modeled on kb-cli's `scripts/strip-internal-refs.sh`); decide whether `.kb/global/` and `.orch/archive/` should be public at all

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
| --- | --- | --- | --- |
| N/A - novel investigation | - | - | - |

## Question

Are the orch ecosystem repos (orch-go, kb-cli, harness, scrape) ready for public release? What secrets, sensitive paths, or personal data need to be scrubbed first?

## Scan Methodology

For each repo:
1. Check git sync status (unpushed commits vs remote)
2. Scan for API keys, tokens, passwords, OAuth secrets (`sk-`, `api_key`, `token`, `secret`, `credential`)
3. Scan for hardcoded paths containing usernames (`/Users/dylanconlin`)
4. Scan for account configs, SCS-specific data (`sendcutsend`, `scs`, employer email)
5. Check .gitignore coverage
6. Review sensitive tracked directories (`.beads/`, `.claude/`, `.orch/`, `.kb/`)
7. Verdict: ready / needs-scrubbing / keep-private

## Test Performed

```bash
# Per-repo scans (representative commands)
git ls-files | xargs grep -l '/Users/dylanconlin' 2>/dev/null | wc -l
git ls-files | xargs grep -l 'sendcutsend\|SendCutSend' 2>/dev/null
grep -ri 'dylan\.conlin@\|dylanconlin@' across tracked files
grep -ri 'sk-ant\|api_key\|secret_key\|password' across tracked files
git log --oneline origin/master..HEAD | wc -l
git ls-files '*.env' 'accounts.yaml' '*credentials*' '*secret*' '*.key' '*.pem'
```

## Findings

---

### 1. orch-go — NEEDS-SCRUBBING (significant)

**Git sync:** 479 commits ahead of origin/master
**Remote:** `git@github.com:dylan-conlin/orch-go.git`

#### CRITICAL: Employer/Personal Data

| Category | Count | Files | Severity |
|----------|-------|-------|----------|
| Files with `/Users/dylanconlin` | 1,041 | .kb/, .orch/, .beads/, Go source | HIGH |
| Files mentioning SendCutSend/SCS | 52+ | .kb/global/gap-tracker.json, investigations, models, tests | HIGH |
| Personal email addresses | 20+ files | .kb/ investigations, models, probes, workspace archives | HIGH |
| Work email (sendcutsend.com) | 15+ files | Same locations | HIGH |

#### Specific Critical Files

**Source code (cmd/, pkg/) — 2 files:**
- `cmd/orch/audit_cmd.go:212` — Hardcoded `/Users/dylanconlin` in launchd plist template PATH
- `pkg/verify/decision_patches_test.go` — Test file with hardcoded absolute paths

**Deploy config — 1 file:**
- `deploy/com.orch.entropy.plist` — All 7 paths hardcoded to `/Users/dylanconlin`

**Test data revealing employer structure:**
- `pkg/group/group_test.go` — References `scs` group, `scs-special-projects`, `toolshed`, `price-watch`, `sendassist`

**Tracked knowledge/metadata (bulk of exposure):**
- `.kb/global/gap-tracker.json` (4,701 lines) — 83 SCS references, includes "SendCutSend Employee Handbook" task, SCS scraper mappings, employer-specific business data
- `.kb/global/projects.json` — All project paths with `/Users/dylanconlin`
- `.kb/global/groups.yaml` — `scs` group with `parent: scs-special-projects`
- `.beads/issues.jsonl` (1,683 lines) — Issue tracking with absolute paths
- `.claude/projects/.../memory/` — Claude Code memory files (3 tracked)
- `.kb/threads/2026-03-05-making-price-watch-s-collection.md` — SCS-specific work context mentioning "Jim", pricing models
- `.kb/models/orchestration-cost-economics/model.md` — Contains `dylan@sendcutsend.com` example
- Multiple `.kb/investigations/` — Account distribution designs with both personal and work emails
- `.orch/workspace-archive/` — 2,254 tracked archive files, many with paths/emails

**Tracked directories breakdown:**
- `.kb/` — 1,952 tracked files
- `.orch/` — 2,254 tracked files (mostly archives)
- `.beads/` — 8 tracked files
- `.claude/` — 13 tracked files

#### What's Clean

- **No API keys/tokens** — All `sk-ant-*` patterns are placeholders
- **No credential files** — No `.env`, `accounts.yaml`, `*.key`, `*.pem` tracked
- **CLAUDE.md** — Clean (no personal paths or employer data)
- **Go source code** — Almost entirely clean (only 2 files need fixes)
- **`.gitignore`** — Properly excludes certificates, build artifacts, workspace
- **No private keys or SSH keys**

#### Remediation Plan

**Phase 1 — Quick source fixes (5 min):**
1. `cmd/orch/audit_cmd.go` — Use `os.UserHomeDir()` instead of hardcoded path
2. `deploy/com.orch.entropy.plist` — Use `$HOME` or make it a template
3. `pkg/verify/decision_patches_test.go` — Use `t.TempDir()` for paths
4. `pkg/group/group_test.go` — Rename `scs` → generic group name (e.g., `work-projects`)

**Phase 2 — Knowledge artifact scrubbing (30-60 min):**
1. Build strip script (model on kb-cli's `scripts/strip-internal-refs.sh`)
2. Strip `/Users/dylanconlin` → relative paths in all `.kb/` files
3. Redact email addresses (`dylan.conlin@sendcutsend.com` → `user@company.com`)
4. Strip SCS business data from `.kb/global/gap-tracker.json` or remove file

**Phase 3 — Architectural decision (needs orchestrator input):**
- Should `.orch/archive/` and `.orch/workspace-archive/` be public? (2,254 files, contains session context, synthesis files, spawn contexts with personal paths)
- Should `.beads/issues.jsonl` be public? (internal project management)
- Should `.kb/global/` be public? (cross-project knowledge, but contains employer data)
- Should `.claude/projects/.../memory/` be tracked? (agent memory with user context)

**Verdict: NEEDS-SCRUBBING — Significant but tractable. Source code is nearly clean. Bulk of work is in knowledge artifacts.**

---

### 2. kb-cli — NEEDS-SCRUBBING (minor)

**Git sync:** 3 commits ahead of origin/main
**Remote:** `git@github.com:dylan-conlin/kb-cli.git`

**Already has public release prep:** Commit 55cc54e created `scripts/strip-internal-refs.sh` with 17-test suite. All `.kb/` artifacts have been stripped of `/Users/dylanconlin` paths.

**Remaining issues:**
1. `.beads/issues.jsonl` — Contains absolute paths in issue metadata (tracked). Remove from tracking or strip.
2. `build/kb` and `kb` binaries — Committed with debug symbols containing build-time paths. Remove from tracking.

**Clean areas:**
- No API keys, tokens, or credentials
- No personal identifiers beyond GitHub username
- `.kb/` artifacts already stripped
- Good `.gitignore`

**Verdict: NEEDS-SCRUBBING (minor) — 5 minutes of git rm --cached commands**

---

### 3. harness — READY

**Git sync:** 3 commits ahead of origin/master
**Remote:** `git@github.com:dylan-conlin/harness.git`

**Scan results:** All clean.
- No API keys, tokens, or credentials
- No hardcoded user paths in source
- No personal identifiers
- No employer data
- Good `.gitignore` (excludes `.orch/`, `.beads/`, `.kb/`)
- `.harness/config.yaml` contains only accretion thresholds
- `.claude/settings.json` has hook config only

**Minor note:** Some `.orch/workspace/*/SYNTHESIS.md` files are tracked (agent work docs) — safe but could be excluded.

**Verdict: READY — No scrubbing required**

---

### 4. scrape — NEEDS-SCRUBBING (minor)

**Git sync:** 1 commit ahead of origin/master
**Remote:** `git@github.com:dylan-conlin/scrape.git`

**One issue:**
- `Makefile:3` — Hardcoded `/Users/dylanconlin/Documents/personal/harness/harness` path

**Clean areas:**
- API key references are env var names only (`ANTHROPIC_API_KEY`) — correct pattern
- No credentials, tokens, or secrets
- No employer data
- Good `.gitignore`

**Verdict: NEEDS-SCRUBBING (minor) — 1 line fix in Makefile**

---

## Conclusion

| Repo | Verdict | Commits Behind | Effort to Fix | Blocking Issues |
|------|---------|---------------|---------------|-----------------|
| **orch-go** | NEEDS-SCRUBBING | 479 ahead | 1-2 hours | SCS employer data, 1041 files with paths, personal emails |
| **kb-cli** | NEEDS-SCRUBBING (minor) | 3 ahead | 5 minutes | beads issues tracking, committed binaries |
| **harness** | READY | 3 ahead | 0 | None |
| **scrape** | NEEDS-SCRUBBING (minor) | 1 ahead | 2 minutes | Makefile hardcoded path |

**Priority order for Anthropic application:**
1. **orch-go** — centerpiece, needs most work but source code is nearly clean
2. **scrape** — quick 1-line fix
3. **kb-cli** — quick git rm commands
4. **harness** — already ready

**Key architectural decision needed:** Whether to include `.kb/`, `.orch/archive/`, and `.beads/` in the public repo at all. These directories contain 4,200+ tracked files with the bulk of the sensitive data. Options:
- **Strip and include** — Shows the knowledge system working (impressive for application)
- **Exclude entirely** — Simplest, add to `.gitignore` and `git rm --cached`
- **Selective include** — Keep guides/decisions/models, exclude investigations/archives/issues
