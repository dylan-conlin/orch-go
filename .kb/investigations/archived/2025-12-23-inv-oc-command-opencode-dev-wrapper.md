## Summary (D.E.K.N.)

**Delta:** The `oc` (opencode-dev wrapper) crashes with exit code 133 in directories containing a `.opencode/plugin/` folder when the plugin's `@opencode-ai/plugin` dependency is not installed.

**Evidence:** Ran `bun -e "import * from './.opencode/plugin/superpowers.js'"` in superpowers directory - got "Cannot find module '@opencode-ai/plugin/tool'". After `bun add @opencode-ai/plugin`, opencode launches successfully.

**Knowledge:** OpenCode's auto-install of plugin dependencies (via `installDependencies` in config.ts) uses `Promise.allSettled` and catches errors silently, allowing the TUI to launch before dependencies are installed - but the plugin import then fails with SIGTRAP (exit 133).

**Next:** The superpowers project needs to run `cd .opencode && bun add @opencode-ai/plugin` to install dependencies. Consider filing an issue with OpenCode about the race condition in plugin dependency installation.

**Confidence:** Very High (95%) - Root cause confirmed with reproduction and fix verification.

---

# Investigation: Oc Command Opencode Dev Wrapper

**Question:** Why does the `oc` command (opencode-dev wrapper) print session-context plugin logs but not launch the TUI in certain directories?

**Started:** 2025-12-23
**Updated:** 2025-12-24
**Owner:** Agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** Very High (95%)

---

## Findings

### Finding 1: The failure only occurs in directories with `.opencode/plugin/` 

**Evidence:** 
- Works in `/tmp` (no .opencode): TUI launches
- Works in `orch-go` (has .orch but no .opencode/plugin): TUI launches  
- Fails in `superpowers` (has `.opencode/plugin/superpowers.js`): Exit code 133

**Source:** 
```bash
# Superpowers (fails)
timeout 5 opencode-dev  # Exit code: 133

# orch-go (works) 
timeout 10 opencode-dev  # TUI launches

# /tmp (works)
timeout 10 opencode-dev  # TUI launches
```

**Significance:** The failure is directory-specific, tied to the presence of a plugin.

---

### Finding 2: Exit code 133 indicates SIGTRAP (signal 5) during plugin import

**Evidence:** 
- Exit code 133 = 128 + 5 = SIGTRAP
- Error log showed: `ERROR ... [object ErrorEvent]`
- Running `bun -e "import * from './.opencode/plugin/superpowers.js'"` revealed: `Cannot find module '@opencode-ai/plugin/tool'`

**Source:** 
```bash
cd ~/Documents/personal/superpowers
bun -e "import * as plugin from './.opencode/plugin/superpowers.js'"
# error: Cannot find module '@opencode-ai/plugin/tool'
```

**Significance:** The superpowers.js plugin imports `@opencode-ai/plugin/tool` at line 12, but the dependency wasn't installed in the `.opencode/` directory.

---

### Finding 3: Installing the missing dependency fixes the issue

**Evidence:**
```bash
cd ~/Documents/personal/superpowers/.opencode
bun add @opencode-ai/plugin
# installed @opencode-ai/plugin@1.0.193

# After installation
timeout 10 opencode-dev
# TUI launches successfully, no exit 133
```

**Source:** Manual test in superpowers directory

**Significance:** This confirms the root cause is the missing `@opencode-ai/plugin` dependency.

---

### Finding 4: OpenCode's dependency auto-install has a race condition

**Evidence:** In `/packages/opencode/src/config/config.ts:160-178`:
```typescript
async function installDependencies(dir: string) {
  if (Installation.isLocal()) return  // Skipped for local/dev installs!
  // ...
  await BunProc.run([...]).catch(() => {})  // Errors swallowed silently
}

// Called at line 101:
promises.push(installDependencies(dir))
// ...
result.plugin.push(...(await loadPlugin(dir)))  // Plugin loaded before install finishes

// At line 107:
await Promise.allSettled(promises)  // Install happens in parallel, may not complete
```

**Source:** `/Users/dylanconlin/Documents/personal/opencode/packages/opencode/src/config/config.ts:160-178`

**Significance:** When running from source (`opencode-dev`), `Installation.isLocal()` returns true and skips auto-install entirely. Even for production installs, the plugin is loaded before `installDependencies` completes.

---

## Synthesis

**Key Insights:**

1. **Plugin dependency race condition** - OpenCode's plugin loading happens before/in parallel with dependency installation, causing import failures when dependencies aren't pre-installed.

2. **Local development skips auto-install** - The `opencode-dev` script runs in local mode, which explicitly skips the `installDependencies` step entirely.

3. **The fix is simple** - Running `bun add @opencode-ai/plugin` in the `.opencode/` directory of any project with plugins resolves the issue permanently.

**Answer to Investigation Question:**

The `oc` command fails to launch the TUI in directories with `.opencode/plugin/` because:
1. Plugins are loaded before their dependencies are installed
2. When running from source (`opencode-dev`), auto-installation is skipped entirely
3. Missing `@opencode-ai/plugin` causes a SIGTRAP (exit 133) during import

The session-context plugin logs appear because it's loaded from `~/.config/opencode/plugin/` (which already has dependencies installed), but the project-local superpowers plugin fails.

---

## Confidence Assessment

**Current Confidence:** Very High (95%)

**Why this level?**

Root cause was identified through systematic elimination, confirmed with direct testing of the import failure, and verified by installing the missing dependency.

**What's certain:**

- The missing `@opencode-ai/plugin` dependency causes the crash
- Installing the dependency fixes the issue
- The `opencode-dev` script skips auto-installation because `Installation.isLocal()` returns true

**What's uncertain:**

- Whether this affects production installs (likely not, auto-install should work there)
- Whether there are other directories with the same issue

**What would increase confidence to 100%:**

- Testing the same fix in other projects with local plugins
- Confirming with OpenCode maintainers that this is the expected behavior for local development

---

## Implementation Recommendations

### Recommended Approach: Add dependency installation instructions

**Why this approach:**
- Simple one-time fix per project
- Doesn't require changes to OpenCode
- Works for all projects with local plugins

**Trade-offs accepted:**
- Manual step required for each project with plugins
- Could be automated with a script

**Implementation sequence:**
1. For superpowers (already done): `cd ~/.../superpowers/.opencode && bun add @opencode-ai/plugin`
2. For any other project with plugins: same pattern
3. Optionally: create a script to auto-detect and install

### Alternative Approaches Considered

**Option B: Modify opencode-dev to install dependencies**
- **Pros:** Fixes the root cause for local development
- **Cons:** Requires changes to OpenCode source
- **When to use instead:** If many projects have this issue

---

## Test performed

**Test:** Installed `@opencode-ai/plugin` in superpowers `.opencode/` directory and re-ran opencode-dev

**Result:** TUI launched successfully. No more exit code 133.

```bash
cd ~/Documents/personal/superpowers/.opencode
bun add @opencode-ai/plugin
# installed @opencode-ai/plugin@1.0.193

timeout 10 /Users/dylanconlin/Documents/personal/opencode/packages/opencode/opencode-dev
# Session-context logs appear, then TUI launches and stays running until timeout
```

---

## Conclusion

The issue is caused by missing `@opencode-ai/plugin` dependency in the `.opencode/` directory. When OpenCode tries to load local plugins, the import fails because the dependency isn't available. Running `bun add @opencode-ai/plugin` in the `.opencode/` directory fixes the issue permanently.

This is a race condition/local-dev issue in OpenCode:
- Production installs auto-install dependencies (though there's still a race)
- Local development (`opencode-dev`) skips auto-install entirely
- The fix is simple manual dependency installation

---

## Self-Review

- [x] Real test performed (not code review)
- [x] Conclusion from evidence (not speculation)
- [x] Question answered
- [x] File complete

**Self-Review Status:** PASSED

---

## References

**Files Examined:**
- `/Users/dylanconlin/Documents/personal/opencode/packages/opencode/opencode-dev` - The wrapper script
- `/Users/dylanconlin/Documents/personal/opencode/packages/opencode/src/config/config.ts` - Plugin loading logic
- `/Users/dylanconlin/Documents/personal/opencode/packages/opencode/src/plugin/index.ts` - Plugin initialization
- `/Users/dylanconlin/Documents/personal/superpowers/.opencode/plugin/superpowers.js` - The failing plugin

**Commands Run:**
```bash
# Test in different directories
timeout 10 opencode-dev  # orch-go: works, superpowers: fails

# Identify missing dependency
bun -e "import * from './.opencode/plugin/superpowers.js'"
# Cannot find module '@opencode-ai/plugin/tool'

# Fix
cd ~/Documents/personal/superpowers/.opencode && bun add @opencode-ai/plugin

# Verify fix
timeout 10 opencode-dev  # Now works!
```

---

## Investigation History

**2025-12-24 03:45:** Investigation started
- Initial question: Why does oc command hang in superpowers but work elsewhere?
- Context: Dylan reported oc prints plugin logs but doesn't launch TUI

**2025-12-24 03:50:** Identified directory-specific failure
- Works in orch-go and /tmp, fails in superpowers
- Exit code 133 = SIGTRAP

**2025-12-24 03:52:** Found missing dependency
- Tested plugin import directly with bun -e
- Error: Cannot find module '@opencode-ai/plugin/tool'

**2025-12-24 03:53:** Verified fix
- Installed @opencode-ai/plugin in .opencode directory
- TUI now launches successfully

**2025-12-24 03:55:** Investigation completed
- Final confidence: Very High (95%)
- Status: Complete
- Key outcome: Missing dependency in .opencode/, fixed with bun add
