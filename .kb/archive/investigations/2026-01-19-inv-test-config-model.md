<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Config model loading and spawn integration work correctly, but test coverage for model fields is missing.

**Evidence:** Created two test programs that verified config loading (3 test cases) and spawn integration (5 test cases) all pass.

**Knowledge:** Two config systems exist: pkg/config (project-level with model settings) and pkg/userconfig (user-level with daemon settings). OpenCode.Model defaults to "flash" which is blocked in spawn.

**Next:** Add unit tests to config_test.go for model fields (TestApplyDefaultsModels, TestLoadConfigWithModels).

**Promote to Decision:** recommend-no - This is a test coverage gap fix, not an architectural decision.

<!--
Example D.E.K.N.:
Delta: Test-running guidance is missing from spawn prompts and CLAUDE.md.
Evidence: Searched 5 agent sessions - none ran tests; guidance exists in separate docs but isn't loaded.
Knowledge: Agents follow documentation literally; guidance must be in loaded context to be followed.
Next: Add test-running instruction to SPAWN_CONTEXT.md template.
Promote to Decision: recommend-no (tactical fix, not architectural)

Guidelines:
- Keep each line to ONE sentence
- Delta answers "What did we find?"
- Evidence answers "How do we know?"
- Knowledge answers "What does this mean?"
- Next answers "What should happen now?"
- Promote to Decision: flag for orchestrator/human - recommend-yes if this establishes a pattern, constraint, or architectural choice worth preserving
- Enable 30-second understanding for fresh Claude
-->

---

# Investigation: Test Config Model

**Question:** Do the config package tests cover model-related fields (Claude.Model, OpenCode.Model) and their defaults?

**Started:** 2026-01-19
**Updated:** 2026-01-19
**Owner:** spawned-agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Patches-Decision:** [Path to decision document this investigation patches/extends, if applicable - enables review triggers]
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]
**Supersedes:** [Path to artifact this replaces, if applicable]
**Superseded-By:** [Path to artifact that replaced this, if applicable]

---

## Findings

### Finding 1: Test coverage gap - model fields not tested

**Evidence:** Existing config_test.go has tests for servers, load/save, and basic functionality, but no tests for:
- `ClaudeConfig.Model` field
- `ClaudeConfig.TmuxSession` field
- `OpenCodeConfig.Model` field
- `OpenCodeConfig.Server` field
- `SpawnMode` field
- `ApplyDefaults()` model defaults

**Source:** `pkg/config/config_test.go:1-158` - all tests focus on servers map only

**Significance:** Model-related config fields could have bugs that tests wouldn't catch. Default values (opus for claude, flash for opencode) are untested.

---

### Finding 2: Config model loading works correctly (tested)

**Evidence:** Created test program that verified:
- Test 1: Explicit model values in config are loaded correctly
  - SpawnMode: "claude", Claude.Model: "sonnet", OpenCode.Model: "pro" - all PASS
- Test 2: Defaults are applied when config has minimal values
  - SpawnMode: "opencode", Claude.Model: "opus", OpenCode.Model: "flash" - all PASS
- Test 3: Empty config gets all defaults - all PASS

**Source:** `/tmp/test_config_model.go` - ran via `go run`

**Significance:** Core config loading for model fields is functional. Defaults are correct (opus for claude, flash for opencode).

---

### Finding 3: Spawn integration with config model works (tested)

**Evidence:** Created test of `resolveModelWithConfig` function:
- Test 1: Explicit --model flag takes priority over config - PASS
- Test 2: Claude backend uses config's claude.model (opus) - PASS
- Test 3: OpenCode backend uses config's opencode.model (sonnet) - PASS
- Test 4: No config uses default model (opus) - PASS
- Test 5: Config with empty model field uses default - PASS

**Source:** `/tmp/test_spawn_config_model.go` - copied `resolveModelWithConfig` logic from `cmd/orch/spawn_cmd.go:777-796`

**Significance:** The priority chain works correctly: --model flag > config backend-specific model > default model

---

## Synthesis

**Key Insights:**

1. **Functionality works but tests are missing** - Config model loading and spawn integration both work correctly (Finding 2, 3), but the test suite doesn't cover model fields (Finding 1).

2. **Two config systems exist** - `pkg/config` handles project-level config (.orch/config.yaml with model settings), while `pkg/userconfig` handles user-level config (~/.orch/config.yaml with daemon settings). CLI `config show` only shows user config.

3. **Default chain is correct** - Model resolution follows: --model flag > config's backend-specific model > default (opus for anthropic).

**Answer to Investigation Question:**

No, the config package tests do not cover model-related fields. The existing tests (`config_test.go`) only test the servers map and basic load/save functionality. However, manual testing confirms that:
- Model fields are correctly loaded from YAML
- ApplyDefaults() correctly sets opus for claude.model and flash for opencode.model
- Spawn correctly uses config model values when no --model flag is provided

**Recommendation:** Add unit tests to config_test.go for model fields and defaults.

---

## Structured Uncertainty

**What's tested:**

- ✅ Config loads model fields correctly (verified: test_config_model.go)
- ✅ ApplyDefaults sets correct model defaults (verified: test_config_model.go)
- ✅ resolveModelWithConfig uses config model when no --model flag (verified: test_spawn_config_model.go)
- ✅ Existing config tests pass (verified: go test ./pkg/config/...)

**What's untested:**

- ⚠️ Full spawn command doesn't have dry-run mode to verify end-to-end config usage
- ⚠️ No automated tests for model field loading in config_test.go
- ⚠️ CLI `config get` doesn't support nested keys (claude.model, opencode.model)

**What would change this:**

- Finding would be wrong if actual spawn creates agent with wrong model (would need live test)
- Add unit tests to config_test.go to prevent regression on model fields

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation.

### Recommended Approach ⭐

**Add model field tests to config_test.go** - Extend existing test coverage to include model-related fields.

**Why this approach:**
- Fills identified test gap (Finding 1)
- Builds on existing test patterns already in config_test.go
- Prevents regression on model defaults which spawn depends on

**Trade-offs accepted:**
- Won't add end-to-end spawn tests (too complex, requires daemon)
- Won't modify CLI to expose nested config keys (separate concern)

**Implementation sequence:**
1. Add TestLoadConfigWithModels - test loading config with model fields set
2. Add TestApplyDefaultsModels - test that ApplyDefaults sets correct model defaults
3. Add TestConfigRoundTripModels - test save/load preserves model fields

### Alternative Approaches Considered

**Option B: Add integration tests in spawn_cmd_test.go**
- **Pros:** Tests the actual integration point
- **Cons:** spawn_cmd.go has many dependencies, harder to test in isolation
- **When to use instead:** If config tests pass but spawn still uses wrong model

**Rationale for recommendation:** Unit tests in config package are simpler, more targeted, and match existing test patterns

---

### Implementation Details

**What to implement first:**
- TestApplyDefaultsModels - ensures defaults don't regress
- TestLoadConfigWithModels - ensures YAML parsing works

**Things to watch out for:**
- ⚠️ Default for OpenCode.Model is "flash" but flash is actually blocked in spawn (TPM limits)
- ⚠️ Two config systems (pkg/config vs pkg/userconfig) can be confusing

**Areas needing further investigation:**
- Should OpenCode.Model default be changed from "flash" to something usable?
- Should CLI `config get` support nested keys?

**Success criteria:**
- ✅ `go test ./pkg/config/...` passes with new tests
- ✅ Tests verify ApplyDefaults sets Claude.Model=opus, OpenCode.Model=flash
- ✅ Tests verify round-trip save/load preserves model fields

---

## References

**Files Examined:**
- `pkg/config/config.go` - Config struct, Load, Save, ApplyDefaults
- `pkg/config/config_test.go` - Existing tests (only cover servers)
- `cmd/orch/spawn_cmd.go:777-796` - resolveModelWithConfig function
- `.orch/config.yaml` - Project config with model values

**Commands Run:**
```bash
# Run existing config tests
go test ./pkg/config/... -v

# Test config model loading
go run /tmp/test_config_model.go

# Test spawn config integration
go run /tmp/test_spawn_config_model.go

# Check config CLI
./orch config show
./orch config get spawn_mode
```

**Related Artifacts:**
- **Decision:** None directly related
- **Investigation:** `.kb/investigations/2026-01-14-design-spawn-context-model-inclusion.md` - prior model investigation

---

## Investigation History

**2026-01-19 15:30:** Investigation started
- Initial question: Do config package tests cover model-related fields?
- Context: Spawned to test config model functionality

**2026-01-19 15:35:** Identified test coverage gap
- Existing config_test.go only covers servers, not model fields

**2026-01-19 15:40:** Verified config model loading works
- Created test_config_model.go, all tests pass

**2026-01-19 15:45:** Verified spawn integration works
- Created test_spawn_config_model.go, all tests pass

**2026-01-19 15:50:** Investigation completed
- Status: Complete
- Key outcome: Config model functionality works but lacks test coverage
