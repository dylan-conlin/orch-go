## Summary (D.E.K.N.)

**Delta:** `opencode run --attach` successfully sends messages and triggers agent execution on the specified server.

**Evidence:** Ran `opencode run --attach http://localhost:4096 "Reply with 'VERIFIED' then stop."` and received a successful response from a new session.

**Knowledge:** The `opencode` CLI's `run` command supports attaching to a remote/local server and sending positional message arguments as the initial prompt.

**Next:** Close investigation and proceed with session completion.

**Promote to Decision:** recommend-no

---

# Investigation: Verification Test Opencode Run Attach

**Question:** Can `opencode run --attach` send messages?

**Started:** 2026-01-09
**Updated:** 2026-01-09
**Owner:** Agent orch-go-lphj2
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: CLI Support for --attach

**Evidence:** `opencode run --help` shows the following option:
```
      --attach      attach to a running opencode server (e.g., http://localhost:4096)       [string]
```
It also defines positionals as:
```
Positionals:
  message  message to send                                                     [array] [default: []]
```

**Source:** `opencode run --help`

**Significance:** This confirms the CLI is designed to support sending messages while attached to a server.

---

### Finding 2: Successful Message Delivery and Execution

**Evidence:** Running the command:
`opencode run --attach http://localhost:4096 --format json "Reply with 'VERIFIED' then stop."`
Produced JSON events showing a new session was created (`ses_45c04a974ffeaJB5nqDYclCJsJ`) and the agent replied with "VERIFIED".

**Source:** Live CLI execution.

**Significance:** Proves that the mechanism works end-to-end: message is sent, session is established, and agent processes the message.

---

## Synthesis

**Key Insights:**

1. **Integrated Attach Mechanism** - `opencode run` combines session creation/attachment and message sending into a single command.
2. **Server-Side Compatibility** - The `orch-go` orchestrator server (running on 4096) correctly handles requests from the `opencode` CLI using the `--attach` flag.

**Answer to Investigation Question:**

Yes, `opencode run --attach` can send messages. This was verified by successfully sending a prompt to the local OpenCode server and receiving a response from a spawned agent.

---

## Structured Uncertainty

**What's tested:**

- ✅ `opencode run --attach` sends initial prompts (verified via dummy session test).
- ✅ Server responds to attached CLI requests (verified via port 4096 test).

**What's untested:**

- ⚠️ Sending messages to *existing* sessions via `--attach --session <id>` (not tested to avoid interrupting active agents, but supported by CLI help).

**What would change this:**

- Finding would be wrong if the message was ignored and only a TUI was opened (not the case in JSON format test).

---

## References

**Commands Run:**
```bash
# Check help
opencode run --help

# Test message sending
opencode run --attach http://localhost:4096 --format json "Reply with 'VERIFIED' then stop."
```

---

## Self-Review

- [x] Real test performed (not code review)
- [x] Conclusion from evidence (not speculation)
- [x] Question answered
- [x] File complete

**Self-Review Status:** PASSED
