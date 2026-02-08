**TLDR:** Question: Set up Go CLI project structure with cobra, Makefile, and proper directory layout. Answer: Created cmd/orch/main.go with cobra CLI, Makefile with build/install targets, and organized code into cmd/, pkg/, internal/ structure. High confidence (95%) - all tests pass and build produces working binary.

---

# Investigation: CLI Project Scaffolding and Build Setup

**Question:** How should we set up the Go project structure for orch-go with cobra CLI and Makefile?

**Started:** 2025-12-19
**Updated:** 2025-12-19
**Owner:** Agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** Very High (95%)

---

## Findings

### Finding 1: Project already had working codebase

**Evidence:** main.go had complete implementations for spawn, ask, monitor commands with working tests in main_test.go.

**Source:** main.go (519 lines), main_test.go (447 lines)

**Significance:** Did not need to rewrite logic - just reorganize into proper Go project structure.

---

### Finding 2: Package structure created successfully

**Evidence:** Created:
- `cmd/orch/main.go` - CLI entry point with cobra commands
- `pkg/opencode/` - Client and SSE handling (types.go, client.go, sse.go)
- `pkg/events/` - Event logging (logger.go)
- `internal/` - Empty, for private packages

**Source:** Directory structure verified via `ls -la`

**Significance:** Follows Go best practices for CLI tools with reusable packages.

---

### Finding 3: Cobra CLI integration working

**Evidence:** 
```
$ ./build/orch-go --help
Available Commands:
  ask         Send a message to an existing session
  completion  Generate the autocompletion script for the specified shell
  help        Help about any command
  monitor     Monitor SSE events for session completion
  spawn       Spawn a new OpenCode session
```

**Source:** `./build/orch-go --help` output

**Significance:** CLI is fully functional with spawn, ask, monitor commands plus automatic completion support.

---

## Synthesis

**Key Insights:**

1. **Incremental refactoring worked** - Instead of rewriting, we restructured existing working code into the new layout.

2. **Cobra provides value** - Beyond just argument parsing, we get help text, shell completion, and consistent UX.

3. **Package separation enables reuse** - The `pkg/opencode` package can be imported by other Go projects.

**Answer to Investigation Question:**

The project structure follows standard Go CLI patterns: `cmd/orch/main.go` as entry point, reusable packages in `pkg/`, and a Makefile for build/install. Build produces a working binary that passes all tests.

---

## Confidence Assessment

**Current Confidence:** Very High (95%)

**Why this level?**

Build succeeds, all tests pass (both original main_test.go and new pkg/opencode tests), and CLI shows correct help output.

**What's certain:**

- ✅ Build succeeds: `make build` produces working binary
- ✅ Tests pass: `go test -v ./...` shows all green
- ✅ CLI works: `./build/orch-go --help` shows expected commands

**What's uncertain:**

- ⚠️ Integration with real opencode server not tested (would need running opencode instance)
- ⚠️ CI/CD (goreleaser) not implemented yet - out of scope for this task

**What would increase confidence to 100%:**

- Run end-to-end test with actual opencode server
- Add goreleaser config for releases

---

## Deliverables

**Created:**
- `cmd/orch/main.go` - Cobra CLI entry point
- `Makefile` - Build, test, install targets

**Modified:**
- `go.mod` / `go.sum` - Added cobra dependency

**Directory structure:**
```
orch-go/
├── cmd/orch/          # CLI entry point
│   └── main.go
├── pkg/               # Public packages
│   ├── opencode/      # OpenCode client
│   │   ├── types.go
│   │   ├── client.go
│   │   ├── client_test.go
│   │   ├── sse.go
│   │   └── sse_test.go
│   └── events/        # Event logging
│       └── logger.go
├── internal/          # Private packages (empty)
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

---

## Commands Run

```bash
# Create directory structure
mkdir -p cmd/orch pkg/opencode pkg/events pkg/spawn internal

# Add cobra dependency
go get github.com/spf13/cobra@latest

# Build
make build

# Test
go test -v ./...

# Verify CLI
./build/orch-go --help
./build/orch-go --version
```

---

## Investigation History

**2025-12-19:** Investigation started
- Initial question: Set up Go CLI project structure
- Context: Part of orch-go Phase 1 scaffolding

**2025-12-19:** Implementation complete
- Created cmd/orch/main.go with cobra CLI
- Created Makefile with build/install targets
- All tests passing

**2025-12-19:** Investigation completed
- Final confidence: Very High (95%)
- Status: Complete
- Key outcome: Working Go CLI with proper project structure
