Question
What does OpenCode /session/status return in the running server, and does it explain phantom accumulation by preventing accurate busy/idle tracking?

What I Tested
- `curl -s -S http://127.0.0.1:4096/session/status`

What I Observed
- Returned a non-empty JSON map with at least one session entry: `{"ses_38d5f01f6ffe2o0zxalpXLoSAh":{"type":"busy"}}`.

Model Impact
- Contradicts the assumption that `/session/status` is always empty on this server. The endpoint can return active session status, so phantom accumulation is likely driven by other lifecycle gaps (tmux/workspace/beads reconciliation or stale OpenCode sessions) rather than a permanently empty status map.

Status: Complete
