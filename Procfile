api: orch serve
daemon: orch daemon run
doctor: orch doctor --daemon
# Unset API keys so OpenCode uses OAuth (Max/Pro subscriptions) instead of pay-per-token billing
opencode: env -u ANTHROPIC_API_KEY -u OPENAI_API_KEY ~/.bun/bin/opencode serve --port 4096
