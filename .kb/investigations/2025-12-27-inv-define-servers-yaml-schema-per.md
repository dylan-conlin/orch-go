<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Created `pkg/servers` package with a rich schema for per-project server declarations in `.orch/servers.yaml`.

**Evidence:** All 13 test cases pass; schema supports command/docker/launchd types with http/tcp/command health checks.

**Knowledge:** The schema extends the simple `servers: {name: port}` format in `.orch/config.yaml` with type-specific fields, health checks, dependencies, and environment variables.

**Next:** Integrate with `orch servers` commands to use the new schema for lifecycle management.

---

# Investigation: Define servers.yaml Schema for Per-Project Server Declarations

**Question:** What schema should `.orch/servers.yaml` use to declare servers with types, commands, health checks, and ports for the orch servers switchboard?

**Started:** 2025-12-27
**Updated:** 2025-12-27
**Owner:** feature-impl agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Current System Uses Simple Port Mapping

**Evidence:** The existing `.orch/config.yaml` uses a flat `servers: {name: port}` mapping:
```yaml
servers:
  web: 5188
  api: 3348
```

**Source:** `pkg/config/config.go:22`, `.orch/config.yaml`

**Significance:** The current schema is too simple for lifecycle management. It lacks:
- How to start servers (command vs docker vs launchd)
- Health check definitions
- Dependencies between servers
- Environment variable configuration

---

### Finding 2: Port Registry is Separate from Server Config

**Evidence:** Port allocation is handled by `pkg/port/port.go` which manages a global `~/.orch/ports.yaml` registry. Project configs reference ports but don't own them.

**Source:** `pkg/port/port.go:91-94`

**Significance:** The new servers.yaml should work with the port registry but focus on server definitions rather than port allocation.

---

### Finding 3: orch servers Commands Need Richer Metadata

**Evidence:** The `cmd/orch/servers.go` commands (list, start, stop, attach, open, status) currently rely on tmuxinator configs for starting servers. A richer schema enables:
- Native server management without tmuxinator
- Health check monitoring
- Dependency-aware startup order

**Source:** `cmd/orch/servers.go:52-65`

**Significance:** The new schema should support the existing commands while enabling new capabilities.

---

## Synthesis

**Key Insights:**

1. **Type-based server definitions** - Servers can be started via shell command, Docker container, or launchd service. Each type has different required fields.

2. **Health check polymorphism** - Health checks can be HTTP endpoint checks, TCP port checks, or command-based checks. This covers most use cases.

3. **Dependency ordering** - Servers can declare dependencies on other servers, enabling proper startup sequencing (e.g., database before API).

**Answer to Investigation Question:**

The schema uses a list of server objects with:
- Required: `name`, `port`
- Type-specific: `type` (command|docker|launchd), `command`, `image`, `launchd_label`
- Optional: `health` (with `type`, `path`, `command`, `interval`, `timeout`, `retries`)
- Optional: `env`, `workdir`, `depends_on`

---

## Structured Uncertainty

**What's tested:**

- ✅ Full config parsing with all field types (verified: TestLoad_FullConfig)
- ✅ Default value application (verified: TestLoad_Defaults)
- ✅ Duration parsing for intervals (verified: TestDuration_YAML)
- ✅ Validation of required fields (verified: TestValidate_MissingRequired)
- ✅ Duplicate name/port detection (verified: TestValidate_DuplicateName, TestValidate_DuplicatePort)

**What's untested:**

- ⚠️ Integration with orch servers commands (not implemented yet)
- ⚠️ Actual health check execution (schema only, no runtime)
- ⚠️ Docker/launchd type execution (schema only, no runtime)

**What would change this:**

- If orch servers needs additional fields for runtime management (e.g., restart policy, resource limits)
- If health checks need additional types (e.g., gRPC, custom protocols)

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation.

### Recommended Approach ⭐

**New servers.yaml schema in pkg/servers** - Created with full type definitions and validation.

**Why this approach:**
- Extends rather than replaces existing config.yaml
- Supports all three server types (command/docker/launchd)
- Includes health check definitions for monitoring

**Trade-offs accepted:**
- Not integrated with orch servers commands yet (future work)
- No runtime execution - schema only

**Implementation sequence:**
1. ✅ Create pkg/servers package with types
2. ✅ Add Load/Save/Validate functions
3. ✅ Write comprehensive tests
4. Future: Integrate with orch servers commands

### Alternative Approaches Considered

**Option B: Extend config.yaml**
- **Pros:** Single config file
- **Cons:** Mixing concerns; config.yaml is for static declarations
- **When to use instead:** Never - separation is cleaner

**Option C: Use Docker Compose format**
- **Pros:** Familiar format
- **Cons:** Different purpose; over-engineered for our needs
- **When to use instead:** If Docker-only deployment

---

### Implementation Details

**What to implement first:**
- ✅ pkg/servers package with schema types
- ✅ Load/Save functions
- ✅ Validation logic

**Things to watch out for:**
- ⚠️ Backward compatibility with config.yaml (keep both)
- ⚠️ Port conflicts between servers.yaml and port registry

**Areas needing further investigation:**
- How to migrate from config.yaml servers to servers.yaml
- Whether to deprecate config.yaml servers section

**Success criteria:**
- ✅ Schema parses full server definitions with all field types
- ✅ Defaults are applied correctly
- ✅ Validation catches errors early
- ✅ All tests pass

---

## References

**Files Examined:**
- `pkg/config/config.go` - Current simple server config
- `pkg/port/port.go` - Port registry management
- `cmd/orch/servers.go` - Existing server commands

**Commands Run:**
```bash
# Run tests
go test ./pkg/servers/... -v

# Verify build
go build ./...
```

**Related Artifacts:**
- **Package:** `pkg/servers/servers.go` - New server schema
- **Tests:** `pkg/servers/servers_test.go` - Comprehensive test coverage

---

## Investigation History

**2025-12-27:** Investigation started
- Initial question: What schema should servers.yaml use?
- Context: Need richer server config for orch servers switchboard

**2025-12-27:** Schema designed and implemented
- Created pkg/servers package
- Added types for Server, HealthCheck, Duration
- Implemented Load/Save/Validate

**2025-12-27:** Investigation completed
- Status: Complete
- Key outcome: Full schema with 13 passing tests
