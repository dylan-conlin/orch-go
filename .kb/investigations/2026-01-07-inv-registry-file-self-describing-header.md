<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Added embedded `_schema` field to sessions.json that documents format, valid values, and safe operations.

**Evidence:** All 17 registry tests pass; new TestRegistrySchemaIncluded verifies schema presence and content.

**Knowledge:** Self-describing artifacts should embed documentation in the file itself, not in companion files that may get missed.

**Next:** Close - implementation complete and tested.

**Promote to Decision:** recommend-no - tactical fix following established JSON metadata patterns

---

# Investigation: Registry File Self Describing Header

**Question:** How to make ~/.orch/sessions.json self-describing so agents and humans can understand and safely modify it?

**Started:** 2026-01-07
**Updated:** 2026-01-07
**Owner:** feature-impl agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Registry stores orchestrator sessions with workspace_name as primary key

**Evidence:** `pkg/session/registry.go` defines `OrchestratorSession` struct with fields: `workspace_name`, `session_id`, `project_dir`, `spawn_time`, `goal`, `status`. The `Register()` method uses workspace_name to detect updates vs new entries.

**Source:** `pkg/session/registry.go:29-48`, `pkg/session/registry.go:156-176`

**Significance:** The primary key is not obvious from the raw JSON - it looks like session_id might be the key, but it's actually workspace_name.

---

### Finding 2: Status values are hardcoded strings without documentation

**Evidence:** Tests reference status values "active", "completed", "abandoned" but these are string literals, not constants or documented enums.

**Source:** `pkg/session/registry_test.go:284-289`

**Significance:** Anyone manually editing the file wouldn't know valid values without reading Go code.

---

### Finding 3: File uses locking for concurrent access

**Evidence:** `withLock()` method uses exclusive file creation for locking with stale lock detection (60s timeout).

**Source:** `pkg/session/registry.go:82-120`

**Significance:** Direct file modification bypasses locking and could corrupt state. Users should know to use orch commands.

---

## Synthesis

**Key Insights:**

1. **Self-describing artifacts need inline documentation** - Companion files (schemas, READMEs) can be missed. Embedding `_schema` in the JSON itself ensures the documentation travels with the data.

2. **Underscore prefix signals metadata** - Using `_schema` follows common JSON conventions (like `_id` in MongoDB) to distinguish metadata from data fields.

3. **Documentation should answer "how do I modify this safely?"** - Beyond schema, users need to know that direct modification is unsafe due to locking, and that orch commands are the proper interface.

**Answer to Investigation Question:**

Add an embedded `_schema` field to RegistryData that documents:
- Version (for future schema changes)
- Description (what this file is)
- Primary key (workspace_name)
- Valid status values (active | completed | abandoned)
- Safe operations (read-only)
- How to modify (use orch commands)

This approach is better than a companion schema file because:
1. Documentation travels with the file
2. Valid JSON (unlike JSON5 comments)
3. Any tool reading the file sees the documentation immediately
4. Works with jq: `jq '._schema' sessions.json`

---

## Structured Uncertainty

**What's tested:**

- ✅ Schema is included in saved files (verified: TestRegistrySchemaIncluded)
- ✅ All existing registry operations still work (verified: 17 tests pass)
- ✅ Schema contains expected fields (verified: test checks all fields)

**What's untested:**

- ⚠️ Existing sessions.json files will get schema on next save (not tested with production file)
- ⚠️ Other tools parsing sessions.json ignore _schema (assumed standard behavior)

**What would change this:**

- Finding would be wrong if tools parse _schema as session data (unlikely with underscore prefix)
- Finding would be wrong if JSON parsers can't handle the additional field (standard JSON, so unlikely)

---

## Implementation Recommendations

**Purpose:** Implementation is complete. This section documents what was done.

### Implemented Approach ⭐

**Embedded `_schema` field** - Add inline documentation to every save operation.

**Why this approach:**
- Documentation travels with the file - no separate file to miss
- Valid JSON - works with any tool
- Underscore prefix is conventional for metadata
- Answers the key questions: what is this, how to modify it

**Trade-offs accepted:**
- Slightly larger file size (~250 bytes)
- Schema repeated on every save (ensures always present)

**Implementation sequence:**
1. Added `RegistrySchema` struct to define documentation fields
2. Added `DefaultRegistrySchema()` to provide standard values
3. Modified `save()` to always include schema before writing
4. Added test to verify schema presence and content

---

## References

**Files Examined:**
- `pkg/session/registry.go` - Registry implementation
- `pkg/session/registry_test.go` - Existing tests and status value usage

**Commands Run:**
```bash
# Run all registry tests
go test ./pkg/session/... -v -run TestRegistry

# Verify schema output format
go run /tmp/test_schema.go
```

**Related Artifacts:**
- **Investigation origin:** Session ses_474f lines 230-265 showed jq failures
- **Principle applied:** Self-Describing Artifacts from SPAWN_CONTEXT

---

## Investigation History

**2026-01-07 17:10:** Investigation started
- Initial question: How to make sessions.json self-describing?
- Context: Orchestrator failed to edit registry with jq due to undocumented structure

**2026-01-07 17:25:** Design decision made
- Chose embedded `_schema` approach over companion schema file
- Reason: Documentation travels with data, always visible

**2026-01-07 17:45:** Implementation complete
- Status: Complete
- Key outcome: sessions.json now self-describes with inline _schema field
