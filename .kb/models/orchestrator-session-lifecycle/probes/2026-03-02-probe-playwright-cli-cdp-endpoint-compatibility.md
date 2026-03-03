# Probe: Playwright CLI CDP Endpoint Compatibility

**Model:** orchestrator-session-lifecycle
**Date:** 2026-03-02
**Status:** Complete

---

## Question

Does `playwright-cli` support connecting to an existing CDP endpoint (e.g., `--cdp-endpoint http://localhost:9222`)? Or is CDP connection MCP-only? This determines whether the current MCP preset config is backward-compatible with playwright-cli for spawn defaults.

Tests claims from the prior probe (2026-02-28-probe-playwright-cli-vs-mcp-ux-audit.md), specifically the "backend independence" dimension — whether playwright-cli can operate as a drop-in replacement for MCP when CDP connection is required.

---

## What I Tested

### Test 1: Check playwright-cli help for CDP flags

```bash
playwright-cli --help          # No --cdp-endpoint in global options
playwright-cli --help open     # open: --browser, --config, --extension, --headed, --persistent, --profile only
```

### Test 2: Source code trace — CLI vs MCP code paths

Traced through the compiled source:

- `playwright-cli.js` → `playwright/lib/cli/client/program.js` (CLI client)
- CLI daemon: `playwright/lib/cli/daemon/daemon.js`, `daemon/program.js`, `daemon/commands.js`
- MCP server: `playwright/lib/mcp/program.js`, `mcp/browser/browserContextFactory.js`

```bash
# CDP references in MCP code only:
rg -i "cdp" playwright/lib/mcp/     # 6 files match
rg -i "cdp" playwright/lib/cli/     # 0 files match
```

### Test 3: Config file format supports CDP

Found in `mcp/browser/configIni.js` (line 113):
```javascript
const longhandTypes = {
  "browser.cdpEndpoint": "string",   // ← Supported in INI config
  "browser.cdpTimeout": "number",
  ...
};
```

JSON config file also works — `loadConfig()` (config.js:285-294) parses JSON first, falling back to INI. Both support `browser.cdpEndpoint`.

### Test 4: Environment variable test (functional)

```bash
cd /tmp/pw-cdp-test && PLAYWRIGHT_MCP_CDP_ENDPOINT=http://localhost:9999 playwright-cli open 2>&1
```

**Output:**
```
browserType.connectOverCDP: connect ECONNREFUSED ::1:9999
Call log:
  - <ws preparing> retrieving websocket url from http://localhost:9999
```

### Test 5: Config file test (functional)

```bash
# .playwright/cli.config.json:
# { "browser": { "cdpEndpoint": "http://localhost:9999" } }
cd /tmp/pw-cdp-test && playwright-cli open 2>&1
```

**Output:** Same `connectOverCDP: connect ECONNREFUSED` error — proving it read the config and attempted CDP connection.

### Test 6: Successful CDP connection

```bash
# With Chrome remote debugging on port 9222 (confirmed via lsof)
# .playwright/cli.config.json: { "browser": { "cdpEndpoint": "http://localhost:9222" } }
playwright-cli open
```

**Output:**
```
### Browser `default` opened with pid 91893.
- default:
  - browser-type: chrome
  - user-data-dir: <in-memory>
  - headed: false
```

Successfully connected to existing Chrome instance via CDP.

---

## What I Observed

### Architecture: Two separate code paths, shared config layer

| Component | CLI Flag | Config File | Env Var |
|-----------|---------|-------------|---------|
| **MCP server** | `--cdp-endpoint` | `browser.cdpEndpoint` | `PLAYWRIGHT_MCP_CDP_ENDPOINT` |
| **playwright-cli** | ❌ None | `browser.cdpEndpoint` ✅ | `PLAYWRIGHT_MCP_CDP_ENDPOINT` ✅ |

The MCP server defines `--cdp-endpoint` as a CLI option (in `mcp/program.js`). The CLI daemon does NOT expose this option.

However, both code paths share the same config resolution pipeline:
1. Default config
2. Config file (`.playwright/cli.config.json` or INI) — **supports `browser.cdpEndpoint`**
3. Environment variables — **supports `PLAYWRIGHT_MCP_CDP_ENDPOINT`**
4. CLI overrides (MCP has `--cdp-endpoint`, CLI daemon does not)

Both paths use `browserContextFactory.contextFactory(config)` which checks:
```javascript
if (config.browser.cdpEndpoint)
    return new CdpContextFactory(config);
```

### Key Finding: playwright-cli CAN connect via CDP

It just can't be specified via a command-line flag. It must come from:
1. **Config file**: `.playwright/cli.config.json` with `{ "browser": { "cdpEndpoint": "http://localhost:9222" } }`
2. **Environment variable**: `PLAYWRIGHT_MCP_CDP_ENDPOINT=http://localhost:9222`

### Backward Compatibility Assessment

The current MCP preset uses `--cdp-endpoint http://localhost:9222`. For playwright-cli:
- **Cannot use `--cdp-endpoint` flag** — it's not exposed in the CLI
- **CAN use config file or env var** to achieve the same behavior
- Both paths hit the same `CdpContextFactory` → `playwright.chromium.connectOverCDP()`

---

## Model Impact

- [x] **Confirms** invariant: Backend independence — playwright-cli provides MCP-independent browser automation via CDP
- [ ] **Contradicts** invariant: (none)
- [x] **Extends** model with:
  - CDP endpoint support is available in playwright-cli but through config file or env var, NOT a CLI flag
  - For spawn default design: if agents need CDP connection, set `PLAYWRIGHT_MCP_CDP_ENDPOINT` env var or provide a config file — don't assume `--cdp-endpoint` flag compatibility
  - The shared config layer (`configFromEnv` + `loadConfig`) is the interoperability surface between MCP and CLI

---

## Notes

- playwright-cli v1.59.0-alpha-1771104257000 (installed at `~/claude-npm-global/lib/node_modules/@playwright/cli/`)
- Device emulation is explicitly blocked with CDP: `if (cliOptions.device && cliOptions.cdpEndpoint) throw new Error("Device emulation is not supported with cdpEndpoint.")`
- The config precedence order means env var overrides config file, which means `PLAYWRIGHT_MCP_CDP_ENDPOINT` is the most flexible approach for spawn-time configuration
