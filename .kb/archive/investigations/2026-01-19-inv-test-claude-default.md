## Summary (D.E.K.N.)

**Delta:** Default model is correctly set to Claude Opus 4.5 (`claude-opus-4-5-20251101`) with claude backend.

**Evidence:** Unit tests pass: `TestResolve_Empty` confirms DefaultModel is Opus, `TestModelAutoSelection/no_flags_defaults_to_claude` confirms default backend.

**Knowledge:** The model resolution chain: empty string → DefaultModel → `{anthropic, claude-opus-4-5-20251101}`. Backend defaults to `claude` (not `opencode`) when no flags specified.

**Next:** No action needed - implementation matches constraint.

**Promote to Decision:** recommend-no (verification only, no new decision made)

---

# Investigation: Test Claude Default

**Question:** Is the default model for orch-go correctly set to Claude Opus 4.5?

**Started:** 2026-01-19
**Updated:** 2026-01-19
**Owner:** agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: DefaultModel is Claude Opus 4.5

**Evidence:**
```go
// pkg/model/model.go:17-23
// DefaultModel is used when no model is specified.
// Opus is the default (Max subscription covers unlimited Claude CLI usage).
// Sonnet requires pay-per-token API which needs explicit opt-in.
var DefaultModel = ModelSpec{
    Provider: "anthropic",
    ModelID:  "claude-opus-4-5-20251101",
}
```

**Source:** `pkg/model/model.go:17-23`

**Significance:** The default model is explicitly set to Opus, matching the constraint "orch-go DefaultModel should be Opus (claude-opus-4-5-20251101), not Gemini".

---

### Finding 2: Default backend is `claude`

**Evidence:** Test case "no_flags_defaults_to_claude" in spawn_cmd_test.go:
```go
{
    name:            "no flags defaults to claude",
    modelFlag:       "",
    opusFlag:        false,
    expectedBackend: "claude",
}
```

Test output:
```
=== RUN   TestModelAutoSelection/no_flags_defaults_to_claude
--- PASS: TestModelAutoSelection/no_flags_defaults_to_claude (0.00s)
```

**Source:** `cmd/orch/spawn_cmd_test.go:116-120`

**Significance:** When no model flags are provided, the backend defaults to `claude` (not `opencode`), which uses Claude CLI with Max subscription.

---

### Finding 3: Model resolution chain verified

**Evidence:** Test case `TestResolve_Empty` passes:
```go
func TestResolve_Empty(t *testing.T) {
    result := Resolve("")
    if result != DefaultModel {
        t.Errorf("Expected DefaultModel, got %v", result)
    }
}
```

Test output:
```
=== RUN   TestResolve_Empty
--- PASS: TestResolve_Empty (0.00s)
```

**Source:** `pkg/model/model_test.go:5-10`

**Significance:** Empty model spec correctly resolves to DefaultModel (Opus), confirming the resolution chain works as expected.

---

## Synthesis

**Key Insights:**

1. **Default model is Opus** - The `DefaultModel` variable in pkg/model/model.go is set to `claude-opus-4-5-20251101`, satisfying the constraint.

2. **Default backend matches model** - The default backend is `claude` (not `opencode`), which is appropriate for Opus since Opus requires Claude CLI auth.

3. **Tests verify behavior** - Unit tests confirm both the default model resolution and backend selection work correctly.

**Answer to Investigation Question:**

Yes, the default model is correctly set to Claude Opus 4.5. The implementation in `pkg/model/model.go` defines `DefaultModel` as `{Provider: "anthropic", ModelID: "claude-opus-4-5-20251101"}`. This is verified by unit tests that pass.

---

## Structured Uncertainty

**What's tested:**

- ✅ DefaultModel resolves to Opus (verified: `TestResolve_Empty` passes)
- ✅ No flags defaults to claude backend (verified: `TestModelAutoSelection/no_flags_defaults_to_claude` passes)
- ✅ Opus model uses claude backend (verified: `TestModelAutoSelection/opus_model_auto-selects_claude` passes)

**What's untested:**

- ⚠️ End-to-end spawn with default model (not run - would require actual spawn)

**What would change this:**

- Finding would be wrong if DefaultModel variable was overridden elsewhere in the codebase
- Finding would be wrong if tests were not testing actual behavior

---

## References

**Files Examined:**
- `pkg/model/model.go` - DefaultModel definition
- `pkg/model/model_test.go` - Model resolution tests
- `cmd/orch/spawn_cmd.go` - Model resolution in spawn command
- `cmd/orch/spawn_cmd_test.go` - Spawn model selection tests

**Commands Run:**
```bash
# Check DefaultModel definition
rg "DefaultModel|default.*model" pkg/model --output-mode content

# Run model resolution tests
go test -v ./pkg/model/...

# Run model auto-selection tests
go test -v ./cmd/orch -run TestModelAutoSelection
go test -v ./cmd/orch -run TestValidateModeModelCombo
```

---

## Investigation History

**2026-01-19 15:35:** Investigation started
- Initial question: Is the default model for orch-go correctly set to Claude Opus 4.5?
- Context: Constraint says "orch-go DefaultModel should be Opus, not Gemini"

**2026-01-19 15:40:** Found DefaultModel definition
- Located in pkg/model/model.go:17-23
- Confirmed set to claude-opus-4-5-20251101

**2026-01-19 15:42:** Ran tests
- TestResolve_Empty passes
- TestModelAutoSelection passes
- All model/backend defaults verified

**2026-01-19 15:45:** Investigation completed
- Status: Complete
- Key outcome: Default model is correctly set to Opus
