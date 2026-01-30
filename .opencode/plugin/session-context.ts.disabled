/**
 * Plugin: Session Context
 *
 * Ensures orchestrator-spawned sessions automatically behave as orchestrators
 * without manual prompting, while avoiding frame-mismatch for normal chats.
 *
 * Triggered by: session.start
 * Action: Inject orchestrator role signal via client.session.prompt
 *
 * Problem: Orchestrator sessions started via `orch spawn orchestrator` require
 * manual "act as orchestrator" prompting. Agents spawned as workers with
 * ORCH_WORKER=1 should skip orchestrator skill (saves ~37k tokens).
 *
 * Decision kb-4eb82b: Add explicit orchestrator role signal when 'orch session'
 * is started so agent behaves as orchestrator without manual prompting, but do
 * not force orchestrator mode for every chat in orch-go root.
 *
 * Reference: .kb/decisions/kb-4eb82b.json
 */

import type { Plugin } from "@opencode-ai/plugin"

const LOG_PREFIX = "[session-context]"
const DEBUG = process.env.ORCH_PLUGIN_DEBUG === "1"

function log(...args: any[]) {
  if (DEBUG) console.log(LOG_PREFIX, ...args)
}

/**
 * Session Context Plugin
 *
 * Auto-loads orchestrator role for orchestrator sessions based on CLAUDE_CONTEXT env var.
 */
export const SessionContextPlugin: Plugin = async ({
  project,
  client,
  $,
  directory,
  worktree,
}) => {
  log("Plugin initialized")

  return {
    "session.start": async (input, output) => {
      log("Session starting:", input.sessionID)

      // Check environment variables to determine session type
      const claudeContext = process.env.CLAUDE_CONTEXT
      const orchWorker = process.env.ORCH_WORKER

      log("CLAUDE_CONTEXT:", claudeContext)
      log("ORCH_WORKER:", orchWorker)

      // Skip if not an orchestrator session OR if this is a worker session
      if (
        !claudeContext ||
        (claudeContext !== "orchestrator" && claudeContext !== "meta-orchestrator") ||
        orchWorker === "1"
      ) {
        log("Skipping orchestrator skill loading:", {
          claudeContext,
          orchWorker,
          reason:
            orchWorker === "1"
              ? "Worker session (ORCH_WORKER=1)"
              : "Not an orchestrator session",
        })
        return
      }

      log("Loading orchestrator skill for", claudeContext, "session")

      // Inject orchestrator role instructions
      // This ensures the agent behaves as an orchestrator without manual prompting
      try {
        const orchestratorInstruction = `# Orchestrator Role Signal

You are operating as an **${claudeContext}** session. This session was spawned via \`orch spawn ${claudeContext}\`.

**Your role:**
- Manage and coordinate worker agents via \`orch spawn\`
- Make tactical execution decisions
- Synthesize findings from completed agents
- Track progress and maintain session state in SYNTHESIS.md

**Key tools:**
- \`orch spawn <skill> "task"\` - Delegate work to worker agents
- \`orch complete <agent>\` - Complete agent sessions after verifying deliverables
- \`orch status\` - Check active agents and project state
- \`bd ready\` - Find available work
- \`bd show <id>\` - View issue details

**Session management:**
- Progressive documentation in SYNTHESIS.md (fill AS YOU WORK, not at end)
- Use beads for task tracking (\`bd create\`, \`bd update\`, \`bd close\`)
- Escalate to human for strategic decisions or scope changes

For detailed orchestrator guidance, the orchestrator skill has been auto-loaded for this session.`

        await client.session.prompt({
          path: { id: input.sessionID },
          body: {
            noReply: true,
            parts: [
              {
                type: "text",
                text: orchestratorInstruction,
              },
            ],
          },
        })
        log("Orchestrator role signal injected for session:", input.sessionID)
      } catch (err) {
        if (DEBUG) console.error(LOG_PREFIX, "Failed to inject orchestrator role:", err)
      }
    },
  }
}
