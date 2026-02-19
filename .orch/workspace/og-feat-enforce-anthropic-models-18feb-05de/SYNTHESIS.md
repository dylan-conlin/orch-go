# Session Synthesis

**Agent:** og-feat-enforce-anthropic-models-18feb-05de
**Issue:** orch-go-1082
**Outcome:** success

---

## Plain-Language Summary

Spawn resolution now blocks Anthropic models on the OpenCode backend with a clear error unless users explicitly opt in via config. Added an `allow_anthropic_opencode` user config flag to override the guard for legacy behavior and updated tests to cover the new compatibility checks.

---

## Verification Contract

`/Users/dylanconlin/Documents/personal/orch-go/.orch/workspace/og-feat-enforce-anthropic-models-18feb-05de/VERIFICATION_SPEC.yaml`

---

## Tests Run

```bash
go test ./pkg/spawn -run TestResolve_AnthropicModel -v
go test ./pkg/userconfig -run TestLoadWithMetaExistingConfig -v
```
