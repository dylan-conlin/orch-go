/**
 * Plugin: Task Tool Gate
 *
 * Warns/blocks Task tool usage in orchestrator context.
 * Orchestrators should use `orch spawn` instead of Task tool because:
 * - Task tool lacks workspace setup and beads tracking
 * - Task tool spawns aren't visible in dashboard
 * - Task tool bypasses completion verification workflow
 *
 * Detection methods:
 * 1. Session title pattern: Orchestrator sessions have titles starting with "og-" or "op-"
 *    (orch-go spawned sessions) but WITHOUT beads ID brackets (workers have [beads-id])
 * 2. Environment check: ORCH_WORKER env var (set by orch spawn for workers)
 * 3. Workspace path: Workers operate in .orch/workspace/ directories
 *
 * Triggered by: tool.execute.before (Task tool)
 * Action: Inject warning message explaining why to use orch spawn
 *
 * Note: Task tool is also denied via .opencode/opencode.json permission config.
 * This plugin provides the educational warning explaining WHY.
 *
 * Reference: .kb/investigations/2026-01-20-research-disable-task-tool-opencode-orchestrator.md
 */

import type { Plugin } from "@opencode-ai/plugin"

const LOG_PREFIX = "[task-tool-gate]"
const DEBUG = process.env.ORCH_PLUGIN_DEBUG === "1"

function log(...args: any[]) {
  if (DEBUG) console.log(LOG_PREFIX, ...args)
}

/**
 * Warning message injected when Task tool is invoked in orchestrator context.
 */
const TASK_TOOL_WARNING = `## Task Tool Warning

You attempted to use the **Task tool** in an orchestrator context.

**Why this is blocked:**
Orchestrators should use \`orch spawn\` instead of Task tool because:

1. **No workspace setup** - Task tool spawns agents without proper workspace directories
2. **No beads tracking** - Spawned work isn't linked to beads issues
3. **No dashboard visibility** - Agents spawned via Task tool don't appear in \`orch status\`
4. **No completion verification** - Bypasses the \`orch complete\` verification workflow
5. **No skill context** - Task tool doesn't inject skill-specific spawn context

**Correct approach:**
\`\`\`bash
# Spawn a worker with proper tracking
orch spawn feature-impl "task description" --issue BEADS-ID

# Or for investigation work
orch spawn investigation "what to explore" --issue BEADS-ID
\`\`\`

**Reference:** See CLAUDE.md "Tool Restrictions" section for details.
`

/**
 * Track warned sessions to avoid duplicate warnings.
 */
const warnedSessions = new Set<string>()

/**
 * Track detected orchestrator sessions.
 * Maps sessionID -> isOrchestrator (true if confirmed orchestrator).
 * Workers are detected progressively via tool arguments.
 */
const sessionTypes = new Map<string, "orchestrator" | "worker" | "unknown">()

/**
 * Detect if a session is a worker by examining tool arguments.
 * Workers have these characteristics:
 * 1. They read SPAWN_CONTEXT.md early in their session
 * 2. They operate on files in .orch/workspace/ directories
 * 3. They have session titles with beads ID brackets like [orch-go-xyz]
 */
function isWorkerSession(sessionId: string, tool?: string, args?: any): boolean {
  // Check cache
  const cached = sessionTypes.get(sessionId)
  if (cached === "worker") return true
  if (cached === "orchestrator") return false

  let isWorker = false

  // Detection signal 1: read tool accessing SPAWN_CONTEXT.md
  if (tool === "read" && args?.filePath) {
    if (args.filePath.endsWith("SPAWN_CONTEXT.md")) {
      log(`Worker detected (SPAWN_CONTEXT.md read): session ${sessionId}`)
      isWorker = true
    }
  }

  // Detection signal 2: any tool accessing files in .orch/workspace/
  const filePath = args?.filePath || args?.file_path
  if (typeof filePath === "string" && filePath.includes(".orch/workspace/")) {
    log(`Worker detected (workspace path): session ${sessionId}`)
    isWorker = true
  }

  // Cache positive result
  if (isWorker) {
    sessionTypes.set(sessionId, "worker")
  }

  return isWorker
}

/**
 * Detect if session is an orchestrator based on session title patterns.
 * Called via event hook when session.created fires.
 *
 * Orchestrator session title patterns:
 * - Direct orchestrator: "orchestrator", "orch-*"
 * - Spawned orchestrator: "og-orch-*", "op-orch-*" (without beads ID)
 *
 * Worker session title patterns:
 * - Have beads ID in brackets: "[orch-go-xyz]", "[proj-123]"
 * - Start with "og-inv-*", "og-feat-*", "op-feat-*" etc.
 */
function detectOrchestratorFromTitle(sessionTitle: string): boolean {
  if (!sessionTitle) return false

  const lowerTitle = sessionTitle.toLowerCase()

  // Direct orchestrator indicators
  if (lowerTitle === "orchestrator" || lowerTitle.startsWith("orch-")) {
    return true
  }

  // Worker indicator: has beads ID in brackets like [orch-go-xyz]
  // This is a strong worker signal - workers spawned by orch have this pattern
  if (/\[[\w-]+-\w+\]/.test(sessionTitle)) {
    return false // Worker, not orchestrator
  }

  // Spawned prefixes that could be orchestrator OR worker
  // og = orch-go, op = opencode
  // Check for orch-specific suffixes
  if (
    lowerTitle.startsWith("og-orch") ||
    lowerTitle.startsWith("op-orch")
  ) {
    return true
  }

  // Other og-* or op-* prefixes without orch are likely workers
  // e.g., og-feat-*, og-inv-*, op-feat-*
  if (/^(og|op)-/.test(lowerTitle)) {
    return false // Likely worker
  }

  // Default: unknown, will be detected via tool usage
  return false
}

/**
 * OpenCode plugin that warns about Task tool usage in orchestrator context.
 */
export const TaskToolGatePlugin: Plugin = async ({
  client,
  directory,
}) => {
  log("Plugin initialized, directory:", directory)

  return {
    /**
     * Event hook: Detect orchestrator sessions via session title.
     */
    event: async ({ event }) => {
      if (event.type !== "session.created") return

      const info = (event as any).properties?.info
      if (!info) return

      const sessionId = info.id
      const sessionTitle = info.title || ""

      if (!sessionId) return

      log(`Session created: ${sessionId}, title: "${sessionTitle}"`)

      // Detect orchestrator from title
      if (detectOrchestratorFromTitle(sessionTitle)) {
        sessionTypes.set(sessionId, "orchestrator")
        log(`Orchestrator session detected via title: ${sessionId}`)
      }

      // Detect worker from title patterns
      if (/\[[\w-]+-\w+\]/.test(sessionTitle) || /^(og|op)-(?!orch)/.test(sessionTitle.toLowerCase())) {
        sessionTypes.set(sessionId, "worker")
        log(`Worker session detected via title: ${sessionId}`)
      }
    },

    /**
     * Tool hook: Detect Task tool usage and inject warning for orchestrators.
     */
    "tool.execute.before": async (input, output) => {
      const { tool, sessionID } = input
      const { args } = output

      // Always run worker detection to populate cache
      const isWorker = isWorkerSession(sessionID, tool, args)

      // Only intercept Task tool
      if (tool.toLowerCase() !== "task") {
        return
      }

      log(`Task tool invoked in session ${sessionID}`)

      // Skip warning for confirmed workers
      if (isWorker) {
        log(`Skipping warning for worker session ${sessionID}`)
        return
      }

      // Check if this is an orchestrator session
      const sessionType = sessionTypes.get(sessionID)

      // If confirmed orchestrator OR unknown (conservative approach)
      // Unknown sessions that use Task tool are likely orchestrators
      // because workers shouldn't be using Task tool either
      if (sessionType === "worker") {
        log(`Skipping warning for worker session ${sessionID}`)
        return
      }

      // Avoid duplicate warnings
      if (warnedSessions.has(sessionID)) {
        log(`Already warned session ${sessionID}`)
        return
      }

      log(`Injecting Task tool warning for session ${sessionID} (type: ${sessionType || "unknown"})`)

      // Mark as warned
      warnedSessions.add(sessionID)

      // Clean up old entries if too many
      if (warnedSessions.size > 100) {
        const toDelete = Array.from(warnedSessions).slice(0, 50)
        toDelete.forEach((id) => warnedSessions.delete(id))
      }

      // Inject warning message
      try {
        await client.session.prompt({
          path: { id: sessionID },
          body: {
            noReply: true,
            parts: [
              {
                type: "text",
                text: TASK_TOOL_WARNING,
              },
            ],
          },
        })
        log(`Warning injected for session ${sessionID}`)
      } catch (err) {
        if (DEBUG) console.error(LOG_PREFIX, "Failed to inject warning:", err)
      }
    },
  }
}
