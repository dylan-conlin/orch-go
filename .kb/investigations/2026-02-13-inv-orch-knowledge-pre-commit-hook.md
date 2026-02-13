## Summary (D.E.K.N.)

**Delta:** Pre-commit hook in orch-knowledge was broken by two bugs: `set -e` killing the script before error handling, and calling `orch lint --skills` which doesn't exist.

**Evidence:** Reproduced by committing staged skill files; fixed by removing `set -e` and adding subcommand existence check; verified with two successful commits.

**Knowledge:** Shell hooks that call external tools must gracefully handle missing subcommands, not just missing binaries.

**Next:** Close. Fix is committed and verified in orch-knowledge.

**Authority:** implementation - Localized bug fix in a shell script, no architectural impact.

---

# Investigation: orch-knowledge Pre-commit Hook Broken

**Question:** Why does the orch-knowledge pre-commit hook block all commits when skill files are staged?

**Started:** 2026-02-13
**Updated:** 2026-02-13
**Owner:** og-debug-orch-knowledge-pre-13feb-4963
**Phase:** Complete
**Next Step:** None
**Status:** Complete

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| N/A | - | - | - |

---

## Findings

### Finding 1: `set -e` kills script before error handling runs

**Evidence:** Line 12 of `scripts/pre-commit` had `set -e`. When `orch lint --skills` returns non-zero (unknown command), the shell exits immediately. The error-handling logic on lines 32-44 (checking for "Found N issues") never executes.

**Source:** `~/orch-knowledge/scripts/pre-commit:12` (original), `orch lint --skills` returns exit code 1 with "unknown command" error.

**Significance:** This is the primary bug. Even if `orch lint` existed and returned a non-zero exit for valid reasons (like finding issues), `set -e` would kill the script before the output parsing could differentiate between "command not found" and "issues found."

---

### Finding 2: `orch lint --skills` command doesn't exist

**Evidence:** Running `orch lint --skills` produces: `Error: unknown command "lint" for "orch-go"`. The `lint` subcommand was never implemented in orch-go.

**Source:** `orch lint --skills 2>&1` and `orch lint 2>&1` both fail. No `lint` command exists in `cmd/orch/`.

**Significance:** The hook was written anticipating a future command that was never built. Without the command, every commit with staged skill files fails.

---

### Finding 3: Hook chain structure works correctly

**Evidence:** Beads pre-commit hook at `.git/hooks/pre-commit` correctly chains to `pre-commit.old` (symlink to `scripts/pre-commit`). The chain itself is not broken - the bug is entirely in `scripts/pre-commit`.

**Source:** `.git/hooks/pre-commit` (beads hook), `.git/hooks/pre-commit.old` -> `../../scripts/pre-commit` (symlink).

**Significance:** Fix only needed in `scripts/pre-commit`, no hook plumbing changes required.

---

## Synthesis

**Key Insights:**

1. **Graceful degradation for aspirational features** - The hook referenced `orch lint --skills` that was planned but never built. Hooks should check for subcommand existence, not just binary existence.

2. **`set -e` incompatible with output-parsing error handling** - When you need to capture a command's output and parse it for specific patterns, `set -e` prevents the parsing from ever running on failure.

**Answer to Investigation Question:**

The hook blocks commits because `set -e` causes immediate script exit when `orch lint --skills` fails (the command doesn't exist). The fix: remove `set -e`, add an `orch lint --help` existence check, and capture the exit code separately from the output.

---

## Structured Uncertainty

**What's tested:**

- Verified: `orch lint --skills` returns non-zero (unknown command)
- Verified: Removed `set -e` + added subcommand check allows commits to succeed
- Verified: Two successful commits in orch-knowledge after fix (one with staged skill files, one with the fix itself)

**What's untested:**

- Future: When `orch lint` is eventually implemented, the hook should work correctly (untested since command doesn't exist)

**What would change this:**

- If `orch lint --help` returns non-zero even when the command exists, the check would incorrectly skip linting

---

## References

**Files Examined:**
- `~/orch-knowledge/.git/hooks/pre-commit` - Beads hook that chains to pre-commit.old
- `~/orch-knowledge/.git/hooks/pre-commit.old` - Symlink to scripts/pre-commit
- `~/orch-knowledge/scripts/pre-commit` - The broken hook (now fixed)

**Commands Run:**
```bash
# Verify orch lint doesn't exist
orch lint --skills 2>&1  # "unknown command"

# Test commit after fix
cd ~/orch-knowledge && git commit -m "fix: ..."  # Success
```
