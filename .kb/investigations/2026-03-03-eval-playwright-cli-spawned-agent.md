# Experiential Eval: playwright-cli from Spawned Agent

**Date:** 2026-03-03
**Status:** Complete

## What I Did

Smoke-tested playwright-cli from a Claude Code spawned agent (tmux backend, with `--mcp playwright` flag):

1. Verified `playwright-cli` binary available at `~/.bun/bin/playwright-cli` (symlink)
2. Opened http://localhost:5188 (Swarm Dashboard)
3. Took a screenshot of the Dashboard page
4. Clicked "Work Graph" navigation link
5. Took a screenshot of the Work Graph page
6. Closed the browser

## What Worked Well

- **Binary discovery:** `which playwright-cli` resolved immediately via the `~/.bun/bin` symlink. No PATH issues.
- **Page loading:** `playwright-cli open http://localhost:5188` loaded the page and produced an accessibility snapshot on the first try. Used `domcontentloaded` (not `networkidle`) per constraint — correct for SSE-heavy pages.
- **Screenshots:** `playwright-cli screenshot` produced clean PNGs. The Read tool rendered them inline, making visual verification seamless.
- **Navigation click:** `playwright-cli click e12` correctly resolved the ref to `getByRole('link', { name: 'Work Graph' })` and navigated. SPA routing worked — URL changed to `/work-graph`.
- **Browser cleanup:** `playwright-cli close` cleanly shut down the browser process.
- **Overall flow:** open → screenshot → click → screenshot → close. Five commands, all worked. Total wall-clock time under 30 seconds.

## What Didn't Work

- **`--ref` flag syntax:** I initially tried `playwright-cli click --ref e12` which failed with "unknown option." The correct syntax is `playwright-cli click e12` (positional arg, not flag). Minor — easy to learn, but not obvious from zero.
- **Port 3348 (orch serve API) returns HTTP error:** The task asked to open `localhost:3348` but that's the API server, not the web UI. `playwright-cli open http://localhost:3348` failed with `net::ERR_HTTP_RESPONSE_CODE_FAILURE`. The web UI is at port 5188. This is a task specification issue, not a tool issue.
- **Console errors:** 38-59 console errors on both pages. Likely SSE connection failures (dashboard shows "disconnected"). Not a playwright-cli issue — the app itself has console noise.

## What Surprised Me

- **Ref-based clicking is excellent.** The accessibility snapshot assigns refs (e3, e12, etc.) to every interactive element. Clicking by ref is far more reliable than CSS selectors or text matching. This is the right abstraction for agent-driven browser interaction.
- **No setup needed.** Zero configuration — no config files, no browser install step, no env vars. It just worked from the symlink.

## Would I Use This Again?

Yes, without hesitation. playwright-cli is production-ready for agent browser automation. The open → interact → screenshot → close loop is smooth and reliable. The ref-based interaction model is well-suited for LLM agents since the accessibility snapshot provides exactly the semantic information needed to choose what to click.

**Recommendation:** This tool chain works. No issues blocking adoption for spawned agent UI verification.
