<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Added `InferSkillFromLabels()` and `InferSkillFromIssue()` functions to pkg/daemon/daemon.go. Updated Preview(), Once(), and OnceWithSlot() to use new inference that respects skill:* labels.

**Evidence:** All daemon tests pass including new tests for skill label priority. Smoke tested with `orch daemon preview` on issue with skill:design-session label - correctly showed "Inferred skill: design-session" instead of type-based "feature-impl".

**Knowledge:** The original InferSkill() only mapped issue type to skill. Labels were already available in the Issue struct but weren't being used for skill inference. Priority order is now: skill:* label > issue type inference > error.

**Next:** Close - fix is implemented and tested.

---

# Investigation: Daemon Ignores skill:* Labels When Inferring Skill

**Question:** Why does the daemon ignore `skill:*` labels on issues when inferring which skill to use?

**Started:** 2025-12-28
**Updated:** 2025-12-28
**Owner:** Agent og-debug-daemon-ignores-skill-28dec
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: InferSkill only accepts issueType, ignores labels

**Evidence:** In `pkg/daemon/daemon.go` line 302-316, the `InferSkill()` function signature is:
```go
func InferSkill(issueType string) (string, error)
```

It only maps issue type to skill (bug→systematic-debugging, feature→feature-impl, etc.) and has no access to the issue's labels.

**Source:** 
- `pkg/daemon/daemon.go:302-316` - InferSkill function
- `pkg/daemon/daemon.go:281` - Preview() calls InferSkill with only issueType
- `pkg/daemon/daemon.go:574` - Once() calls InferSkill with only issueType

**Significance:** Issues with explicit `skill:research` labels are incorrectly assigned `feature-impl` based on type (e.g., task→feature-impl) instead of respecting the label.

---

### Finding 2: Issue struct already contains Labels field

**Evidence:** The `Issue` struct at line 55-63 already includes a `Labels []string` field, and the `HasLabel()` method exists for checking labels. The labels are already being used for the triage:ready filter.

**Source:** 
- `pkg/daemon/daemon.go:55-73` - Issue struct with Labels field and HasLabel method

**Significance:** The infrastructure to check labels already exists; we just need to use it in skill inference.

---

## Root Cause

The `InferSkill()` function was designed with only type-based inference. When `skill:*` labels were introduced as a mechanism for explicit skill selection, the inference logic wasn't updated to check labels first.

## Solution

Create a new function `InferSkillFromIssue(*Issue)` that:
1. Checks for `skill:*` labels first
2. Falls back to type-based inference if no skill label found

Expected priority: skill:* label > issue type inference > default

---

## Test Cases

- Issue with `skill:research` label and type `task` → should return `research`
- Issue with `skill:investigation` label and type `bug` → should return `investigation`
- Issue with no skill label and type `bug` → should return `systematic-debugging` (fallback)
- Issue with no skill label and type `feature` → should return `feature-impl` (fallback)
