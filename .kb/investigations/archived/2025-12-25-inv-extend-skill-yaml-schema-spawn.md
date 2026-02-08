## Summary (D.E.K.N.)

**Delta:** Extended skillc manifest.go with SpawnRequires struct supporting 6 spawn-time behavior fields.

**Evidence:** All tests pass in skillc; investigation skill.yaml compiles successfully with new spawn_requires section.

**Knowledge:** Using pointer types for bool fields enables distinguishing "unset" from "explicitly false"; helper methods provide sensible defaults.

**Next:** Close - schema is ready for orch spawn template to consume.

**Confidence:** Very High (95%) - All acceptance criteria met, backward compatibility verified.

---

# Investigation: Extend Skill Yaml Schema Spawn

**Question:** How to extend skill.yaml schema with spawn_requires section for spawn-time behavior configuration?

**Started:** 2025-12-25
**Updated:** 2025-12-25
**Owner:** Agent (feature-impl)
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** Very High (95%)

---

## Findings

### Finding 1: SpawnRequires struct added to skillc manifest.go

**Evidence:** Added struct with 6 fields:
- `authority_level` (string): worker|elevated|orchestrator
- `kb_context` (*bool): gather kb context before spawn
- `beads_tracking` (*bool): use beads for progress tracking
- `servers` (string): none|auto|required
- `synthesis` (string): none|light|full
- `phase_reporting` (*bool): inject phase reporting instructions

**Source:** `/Users/dylanconlin/Documents/personal/skillc/pkg/compiler/manifest.go:29-55`

**Significance:** Using *bool allows distinguishing nil (use default) from false (explicitly disabled).

---

### Finding 2: Helper methods provide sensible defaults

**Evidence:** Added Get* methods that handle nil gracefully:
- GetAuthorityLevel() returns "worker" if nil/empty
- GetKBContext() returns true if nil
- GetServers() returns "auto" if nil/empty
- GetSynthesis() returns "full" if nil/empty

**Source:** `/Users/dylanconlin/Documents/personal/skillc/pkg/compiler/manifest.go:105-155`

**Significance:** Downstream code (orch spawn template) can call these methods without nil checks.

---

### Finding 3: Backward compatible - skills without spawn_requires work fine

**Evidence:** feature-impl skill (no spawn_requires) builds successfully:
```
✓ Compiled .skillc to SKILL.md
Token counts:
  feature-impl: 2872 tokens
```

**Source:** `skillc build` output for feature-impl skill

**Significance:** Existing skills don't need updates.

---

## References

**Files Modified:**
- `/Users/dylanconlin/Documents/personal/skillc/pkg/compiler/manifest.go` - SpawnRequires struct + helper methods
- `/Users/dylanconlin/Documents/personal/skillc/pkg/compiler/manifest_test.go` - 7 new tests
- `/Users/dylanconlin/orch-knowledge/skills/src/worker/investigation/.skillc/skill.yaml` - Example usage

**Commands Run:**
```bash
# Build and test skillc
cd ~/Documents/personal/skillc && go build ./... && go test ./...

# Verify backward compatibility
cd ~/orch-knowledge/skills/src/worker/feature-impl && skillc build

# Verify new schema works
cd ~/orch-knowledge/skills/src/worker/investigation && skillc build
```
