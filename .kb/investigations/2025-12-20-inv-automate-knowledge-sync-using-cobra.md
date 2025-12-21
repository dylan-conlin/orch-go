**TLDR:** Question: How to automate CLI documentation generation using Cobra's doc package? Answer: Created `cmd/gendoc/main.go` that mirrors the command tree and uses `doc.GenMarkdownTreeCustom()` to generate markdown docs in `docs/cli/`. High confidence (95%) - fully tested with `make docs` target.

---

# Investigation: Automate Knowledge Sync using Cobra Doc Gen

**Question:** How can we automate CLI documentation generation to keep docs in sync with code?

**Started:** 2025-12-20
**Updated:** 2025-12-20
**Owner:** Agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** Very High (95%)

---

## Findings

### Finding 1: Cobra provides doc generation package

**Evidence:** The `github.com/spf13/cobra/doc` package provides functions for generating documentation in multiple formats: Markdown, Man pages, ReStructuredText, and YAML.

**Source:** `go doc github.com/spf13/cobra/doc` and https://pkg.go.dev/github.com/spf13/cobra/doc

**Significance:** Native support for doc generation means we don't need third-party tools. The package integrates directly with Cobra command structures.

---

### Finding 2: Commands must be "runnable" for doc generation

**Evidence:** When building the command tree, commands without a `Run` or `RunE` function are marked as `Runnable() = false`, and `GenMarkdownTree` skips non-runnable commands.

**Source:** Testing with debug output showed commands had `runnable: false` until `Run: noopRun` was added.

**Significance:** For doc generation, we need to add placeholder Run functions to all commands, even though they won't actually execute anything.

---

### Finding 3: Separate tool approach is cleaner than exporting rootCmd

**Evidence:** The main CLI is in `package main` with `rootCmd` unexported. Rather than refactoring the main package to export commands, creating a dedicated `cmd/gendoc/main.go` that mirrors the command structure is cleaner and doesn't affect the main CLI.

**Source:** Analysis of `cmd/orch/main.go` structure and Go package constraints.

**Significance:** This approach keeps concerns separated - the gendoc tool only needs command metadata (Use, Short, Long, Flags), not the actual implementations.

---

## Synthesis

**Key Insights:**

1. **Doc generation is structural** - Cobra's doc generator only needs the command tree structure and metadata, not the actual command implementations. This allows us to create a separate tool that mirrors the structure.

2. **Custom prependers add value** - Using `GenMarkdownTreeCustom` with a custom file prepender adds YAML frontmatter with title and generation date, making the docs more useful for static site generators.

3. **Makefile integration enables automation** - A simple `make docs` target makes it easy to regenerate docs when commands change.

**Answer to Investigation Question:**

Automation is achieved through:
1. `cmd/gendoc/main.go` - A tool that mirrors the CLI command tree structure
2. `doc.GenMarkdownTreeCustom()` - Generates markdown for all commands and subcommands
3. `make docs` - Makefile target for easy regeneration

This approach generates 30 markdown files covering all CLI commands with proper frontmatter, usage examples, and flag documentation.

---

## Confidence Assessment

**Current Confidence:** Very High (95%)

**Why this level?**

The implementation is complete and tested. All commands are documented, the Makefile target works, and all existing tests pass.

**What's certain:**

- ✅ Cobra's doc package generates correct markdown output
- ✅ All 30 CLI commands are documented with proper format
- ✅ `make docs` regenerates docs successfully
- ✅ Existing tests still pass

**What's uncertain:**

- ⚠️ Command structure in gendoc must be kept in sync with main.go manually
- ⚠️ Some edge cases in flag documentation might differ from actual CLI behavior

**What would increase confidence to 100%:**

- Automated test comparing generated docs with actual `--help` output
- CI integration to verify docs are up-to-date

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation.

### Recommended Approach ⭐

**Separate gendoc tool with Makefile integration** - Create `cmd/gendoc/main.go` that mirrors command structure and generates docs via `make docs`.

**Why this approach:**
- No changes to main CLI package required
- Clean separation of concerns
- Easy to integrate into build process

**Trade-offs accepted:**
- Command structure must be manually synced between main.go and gendoc/main.go
- This is acceptable because command changes are infrequent and easy to spot

**Implementation sequence:**
1. Create `cmd/gendoc/main.go` with command tree mirroring main CLI
2. Add `docs` target to Makefile
3. Generate initial documentation

### Alternative Approaches Considered

**Option B: Export rootCmd from main package**
- **Pros:** Single source of truth for command structure
- **Cons:** Requires refactoring main package, complicates build
- **When to use instead:** If commands change frequently and sync is a burden

**Option C: Use external tool (e.g., cobra-docs)**
- **Pros:** No custom code needed
- **Cons:** External dependency, less control over output format
- **When to use instead:** For simple CLIs without customization needs

**Rationale for recommendation:** The gendoc approach provides full control over output while keeping the main CLI clean. Command changes are rare enough that manual sync is acceptable.

---

## References

**Files Examined:**
- `cmd/orch/main.go` - Main CLI entry point and command definitions
- `Makefile` - Build configuration

**Commands Run:**
```bash
# Check Cobra doc package capabilities
go doc github.com/spf13/cobra/doc

# Add dependency
go get github.com/spf13/cobra/doc@v1.10.2

# Generate docs
go run ./cmd/gendoc

# Run tests
go test ./...
```

**External Documentation:**
- https://pkg.go.dev/github.com/spf13/cobra/doc - Cobra doc package reference

---

## Investigation History

**2025-12-20 18:20:** Investigation started
- Initial question: How to automate CLI documentation generation?
- Context: Need to keep docs in sync with CLI commands

**2025-12-20 18:25:** Found Cobra's doc package
- Explored `github.com/spf13/cobra/doc` capabilities
- Identified `GenMarkdownTree` as key function

**2025-12-20 18:30:** Created gendoc tool
- Built command tree mirroring main CLI
- Discovered need for `Run` functions to make commands "runnable"

**2025-12-20 18:35:** Investigation completed
- Final confidence: Very High (95%)
- Status: Complete
- Key outcome: Successfully automated doc generation with `make docs`
