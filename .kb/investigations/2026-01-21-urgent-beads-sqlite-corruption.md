# URGENT: Beads SQLite Database Corruption

**Priority:** P0
**Status:** Open
**Discovered:** 2026-01-21

## Problem

Beads database is corrupted, blocking ALL bd operations:

```
Error: failed to open database: failed to enable WAL mode: sqlite3: database disk image is malformed
```

This affects:
- Pre-commit hooks (fail on every commit)
- All bd commands (create, list, show, etc.)
- No-db mode also fails due to mixed prefixes

## Impact

- Cannot create issues via bd
- Cannot track work properly
- Pre-commit hooks require `--no-verify` to bypass

## Likely Cause

SQLite database file corruption. Possible causes:
- Disk issue
- Interrupted write
- Concurrent access conflict

## Recovery Options

1. **Rebuild from JSONL** - `.beads/issues.jsonl` should have all data
2. **SQLite recovery tools** - `sqlite3 .beads/*.db ".recover"`
3. **Delete and reimport** - Remove db, let bd recreate from JSONL

## Location

Check both:
- `~/Documents/personal/beads/.beads/*.db`
- `~/Documents/personal/orch-go/.beads/*.db`

## Next Steps

1. Identify which db file(s) are corrupted
2. Backup the corrupted file
3. Attempt recovery or rebuild from JSONL
