<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
-->

## Summary (D.E.K.N.)

**Delta:** The default model in orch-go is Claude Opus 4.5 (`claude-opus-4-5-20251101`).

**Evidence:** Read `pkg/model/model.go` lines 17-23 showing `DefaultModel` variable definition.

**Knowledge:** Opus is default because Max subscription covers unlimited Claude CLI usage; Sonnet requires pay-per-token API.

**Next:** Close - question answered, no action needed.

**Promote to Decision:** recommend-no (simple fact verification, not architectural)

---

# Investigation: Simple Test Read Pkg Model

**Question:** What is the default model in pkg/model/model.go?

**Started:** 2026-01-20
**Updated:** 2026-01-20
**Owner:** Worker agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Default model is Claude Opus 4.5

**Evidence:**
```go
// DefaultModel is used when no model is specified.
// Opus is the default (Max subscription covers unlimited Claude CLI usage).
// Sonnet requires pay-per-token API which needs explicit opt-in.
var DefaultModel = ModelSpec{
	Provider: "anthropic",
	ModelID:  "claude-opus-4-5-20251101",
}
```

**Source:** `pkg/model/model.go:17-23`

**Significance:** Confirms the default model is `anthropic/claude-opus-4-5-20251101`. The comment explains the rationale: Max subscription covers unlimited CLI usage.

---

## Structured Uncertainty

**What's tested:**

- ✅ Default model value (verified: read pkg/model/model.go directly)

**What's untested:**

- N/A (simple file read task)

**What would change this:**

- Someone modifying the DefaultModel variable in model.go

---

## References

**Files Examined:**
- `pkg/model/model.go` - Read to find DefaultModel definition

**Commands Run:**
```bash
# Verify project location
pwd

# Read model file
# (used Read tool)
```

---

## Investigation History

**2026-01-20:** Investigation completed
- Status: Complete
- Key outcome: Default model is `anthropic/claude-opus-4-5-20251101` (Claude Opus 4.5)
