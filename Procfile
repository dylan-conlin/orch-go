api: orch serve
web: cd web && bun run dev
daemon: orch daemon run
# Unset ANTHROPIC_API_KEY to use OAuth stealth mode (Max subscription via OpenCode)
opencode: env -u ANTHROPIC_API_KEY BUN_JSC_heapSize=4096 ~/.bun/bin/opencode serve --port 4096
