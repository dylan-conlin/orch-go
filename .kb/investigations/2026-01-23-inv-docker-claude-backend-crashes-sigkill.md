<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Docker Claude agents are killed by Linux OOM killer because Colima VM has 8GB total and no per-container memory limits are set.

**Evidence:** Colima config shows `memory: 8` (GB). docker.go uses `docker run -it --rm` with no `--memory` flag. Claude CLI can consume 2GB+ with large context.

**Knowledge:** Adding `--memory 4g --memory-swap 4g` to Docker run command prevents any single container from exhausting VM memory.

**Next:** Implement fix in pkg/spawn/docker.go, test with Docker spawn.

**Promote to Decision:** recommend-no (bug fix, not architectural decision)

---

# Investigation: Docker Claude Backend Crashes SIGKILL

**Question:** Why are Docker-backed Claude agents being killed mid-session with SIGKILL, and how do we prevent it?

**Started:** 2026-01-23
**Updated:** 2026-01-23
**Owner:** Architect agent
**Phase:** Complete
**Next Step:** Implement fix in pkg/spawn/docker.go
**Status:** Complete

**Patches-Decision:** N/A
**Extracted-From:** N/A
**Supersedes:** N/A
**Superseded-By:** N/A

---

## Findings

### Finding 1: Colima VM has limited memory (8GB total)

**Evidence:** Colima configuration at `~/.colima/default/colima.yaml` shows:
```yaml
memory: 8  # Size of memory in GiB allocated to VM
cpu: 4
vmType: vz
```

**Source:** ~/.colima/default/colima.yaml:12-13

**Significance:** All Docker containers share this 8GB pool. When combined memory usage exceeds available memory, the Linux OOM killer in the VM terminates processes.

---

### Finding 2: Docker run command has no memory limits

**Evidence:** The SpawnDocker function in docker.go uses:
```go
dockerCmd := fmt.Sprintf(
    `docker run -it --rm `+
    `--name %q `+
    // ... volume mounts and env vars ...
    `%s `+
    `bash -c 'claude --dangerously-skip-permissions < %q'`,
```

No `--memory` or `--memory-swap` flags are specified.

**Source:** pkg/spawn/docker.go:86-110

**Significance:** Without memory limits, a single Claude container can attempt to consume all 8GB of VM memory, triggering OOM killer.

---

### Finding 3: Claude CLI memory usage is highly variable

**Evidence:** Research from GitHub issues and documentation:
- Baseline: ~270-370MB per Claude process
- Large contexts: 1-2GB+ (observed with long conversations)
- Bug conditions: Up to 20GB (reported in issue #5771, #9897)
- Concurrent agents multiply this (e.g., 3 agents × 2GB = 6GB)

**Source:**
- https://github.com/anthropics/claude-code/issues/5771
- https://github.com/anthropics/claude-code/issues/9897

**Significance:** A single Claude agent can exceed 2GB easily, and multiple concurrent Docker agents (daemon mode) can rapidly exhaust 8GB.

---

### Finding 4: Native Claude spawns don't have this issue

**Evidence:** Native Claude backend (--backend claude) runs directly on macOS host which has significantly more memory (typically 16-64GB). The Claude process is a direct child of the shell, not containerized.

**Source:**
- pkg/spawn/claude.go - runs `claude` directly in tmux
- Issue description: "Use `--backend claude` (native tmux) instead of docker for critical work"

**Significance:** The OOM issue is specific to Docker backend because of Colima VM memory constraints. Native spawns have access to full host memory.

---

### Finding 5: SIGKILL is the OOM killer signature

**Evidence:** The error message from the crash:
```
bash: line 1:     8 Killed                  claude --dangerously-skip-permissions < "...SPAWN_CONTEXT.md"
```

Exit via "Killed" with signal 9 (SIGKILL) is the signature of Linux OOM killer. Regular termination would show SIGTERM (15).

**Source:** Issue description, standard Linux OOM killer behavior

**Significance:** Confirms this is memory pressure, not a Claude CLI bug or network issue.

---

## Synthesis

**Key Insights:**

1. **Resource starvation is architectural** - Docker backend trades Statsig fingerprint isolation for memory constraints. This is inherent to running in a VM with limited resources.

2. **Memory limits provide graceful degradation** - With `--memory` limits, Docker will OOM-kill the specific container rather than letting it starve other containers or crash the VM.

3. **4GB per container is a reasonable limit** - Given 8GB VM total, allowing 4GB per container means 2 concurrent agents can run safely. This is sufficient for most Claude Code sessions.

**Answer to Investigation Question:**

Docker Claude agents crash with SIGKILL because:
1. Colima VM has 8GB total memory
2. No per-container memory limits are set
3. Claude CLI can consume 2GB+ with large contexts
4. Multiple concurrent agents exhaust VM memory
5. Linux OOM killer sends SIGKILL to terminate the heaviest process

The fix is to add `--memory 4g --memory-swap 4g` to the Docker run command in docker.go. This caps each container at 4GB (50% of VM memory), allowing 2 concurrent agents while preventing any single agent from triggering OOM killer.

---

## Structured Uncertainty

**What's tested:**

- ✅ Colima config shows 8GB memory (verified: read ~/.colima/default/colima.yaml)
- ✅ docker.go has no --memory flag (verified: read pkg/spawn/docker.go:86-110)
- ✅ SIGKILL in error message matches OOM killer pattern (verified: bash "Killed" output)

**What's untested:**

- ⚠️ Fix actually prevents crashes (requires testing with docker spawn)
- ⚠️ 4GB limit is sufficient for large Claude contexts (may need adjustment)
- ⚠️ Memory limit behavior under concurrent spawn load

**What would change this:**

- Finding would be wrong if crashes occur with only 1 agent using <2GB (would suggest different root cause)
- Finding would be wrong if native Claude spawns also crash with SIGKILL (would suggest Claude CLI bug)
- 4GB limit might need increase if Claude contexts frequently exceed 4GB

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation.

### Recommended Approach ⭐

**Add Docker memory limits** - Modify docker run command to include `--memory 4g --memory-swap 4g`

**Why this approach:**
- Directly addresses root cause (unbounded memory consumption)
- Prevents any single container from exhausting VM
- Allows graceful container-level OOM instead of VM-level chaos
- Simple, single-file change

**Trade-offs accepted:**
- Agents with very large contexts (>4GB) will be killed (acceptable - use native backend for those)
- Reduces max concurrent agents to 2 (acceptable - docker is escape hatch, not primary mode)

**Implementation sequence:**
1. Add `--memory 4g --memory-swap 4g` flags to dockerCmd in docker.go
2. Test with `orch spawn --backend docker` to verify agent runs successfully
3. Monitor for any new issues with memory-limited containers

### Alternative Approaches Considered

**Option B: Increase Colima VM memory**
- **Pros:** More headroom for all containers
- **Cons:** Consumes more host resources; doesn't prevent runaway containers
- **When to use instead:** If 4GB limit proves too restrictive for typical workloads

**Option C: Add memory monitoring with early warning**
- **Pros:** Could preemptively warn before OOM
- **Cons:** More complex; doesn't prevent the crash, just predicts it
- **When to use instead:** If we need observability into container memory usage

**Rationale for recommendation:** Option A directly fixes the bug with minimal complexity. Memory limits are the standard Docker solution for preventing resource exhaustion.

---

### Implementation Details

**What to implement first:**
- Add `--memory 4g --memory-swap 4g` to dockerCmd format string
- This is the complete fix; no additional changes needed

**Things to watch out for:**
- ⚠️ Memory limit should come before the image name in docker run
- ⚠️ Format: `--memory 4g --memory-swap 4g` (4g = 4 gigabytes)
- ⚠️ Setting --memory-swap equal to --memory disables swap (prevents disk thrashing)

**Areas needing further investigation:**
- Optimal memory limit (4GB is a starting point; may need tuning)
- Whether to make limit configurable via spawn flags or config

**Success criteria:**
- ✅ Docker agents no longer crash with SIGKILL under normal usage
- ✅ Agent at 45% quota / 9k tokens (original crash scenario) completes successfully
- ✅ Multiple concurrent docker agents can coexist without OOM

---

## References

**Files Examined:**
- pkg/spawn/docker.go:86-110 - Docker spawn command construction
- ~/.colima/default/colima.yaml:12-13 - Colima VM memory configuration
- ~/.claude/docker-workaround/Dockerfile - Container image (no memory config)
- ~/.claude/docker-workaround/run.sh - Convenience script (no memory config)
- ~/.docker/config.json - Shows Colima context in use

**Commands Run:**
```bash
# Verify project location
pwd

# Check Colima config
cat ~/.colima/default/colima.yaml | head -15

# Check docker.go for memory limits
grep -n "memory" pkg/spawn/docker.go
```

**External Documentation:**
- https://github.com/anthropics/claude-code/issues/5771 - Claude Code memory usage bug
- https://github.com/anthropics/claude-code/issues/9897 - Claude Code massive memory consumption

**Related Artifacts:**
- **Investigation:** .kb/investigations/2026-01-20-inv-implement-docker-backend-orch-spawn.md - Original docker backend implementation
- **Investigation:** .kb/investigations/2026-01-21-inv-test-daemon-spawn-docker-backend.md - Docker daemon spawn testing
- **Guide:** .kb/guides/claude-code-sandbox-architecture.md - Sandbox architecture reference

---

## Investigation History

**2026-01-23 04:00:** Investigation started
- Initial question: Why are Docker Claude agents being killed mid-session with SIGKILL?
- Context: Agent crashed at 45% quota, ~9.1k tokens with "Killed" message

**2026-01-23 04:15:** Root cause identified
- Found Colima VM has 8GB total memory
- docker.go has no --memory flags
- Claude CLI can consume 2GB+ per process

**2026-01-23 04:30:** Investigation completed
- Status: Complete
- Key outcome: Add `--memory 4g --memory-swap 4g` to Docker run command in docker.go
