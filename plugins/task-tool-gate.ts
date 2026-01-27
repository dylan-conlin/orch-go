/**
 * Plugin: Task Tool Gate
 *
 * Warns/blocks Task tool usage in orchestrator context.
 * Orchestrators should use `orch spawn` instead of Task tool because:
 * - Task tool lacks workspace setup and beads tracking
 * - Task tool spawns aren't visible in dashboard
 * - Task tool bypasses completion verification workflow
 *
 * Detection methods (in priority order):
 * 1. Skill loading: When "orchestrator" skill is loaded via Skill tool, mark session
 * 2. Session title pattern: Orchestrator sessions have titles starting with "og-" or "op-"
 *    (orch-go spawned sessions) but WITHOUT beads ID brackets (workers have [beads-id])
 * 3. Environment check: ORCH_WORKER env var (set by orch spawn for workers)
 * 4. Workspace path: Workers operate in .orch/workspace/ directories
 *
 * Triggered by: tool.execute.before (Task tool, Skill tool)
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
 * Detect if a session is an orchestrator via Skill tool loading.
 * When the "orchestrator" skill is loaded, mark the session as orchestrator
 * AND update the session metadata so the registry gate can block Task tool.
 *
 * This is the primary detection method for interactive sessions where
 * the user invokes the orchestrator skill manually.
 */
async function detectOrchestratorFromSkill(
  sessionId: string,
  tool: string,
  args?: any,
  updateSessionMetadata?: (sessionId: string, role: "orchestrator") => Promise<void>
): Promise<boolean> {
  if (tool.toLowerCase() !== "skill") return false

  // Skill tool uses 'skill' parameter (from SPAWN_CONTEXT) or 'name' parameter (from OpenCode)
  const skillName = args?.skill || args?.name
  if (!skillName) return false

  // Check if orchestrator skill is being loaded
  if (skillName.toLowerCase() === "orchestrator") {
    log(`Orchestrator detected (skill loading): session ${sessionId}`)
    sessionTypes.set(sessionId, "orchestrator")

    // Update session metadata so registry gate can block Task tool
    if (updateSessionMetadata) {
      try {
        await updateSessionMetadata(sessionId, "orchestrator")
        log(`Session metadata updated for ${sessionId}: role=orchestrator`)
      } catch (err) {
        if (DEBUG) console.error(LOG_PREFIX, "Failed to update session metadata:", err)
      }
    }

    return true
  }

  return false
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

  /**
   * Update session metadata to set the role field.
   * This allows the registry gate to block Task tool for orchestrators.
   *
   * Uses the OpenCode PATCH /session/:sessionID API endpoint.
   */
  async function updateSessionMetadata(sessionId: string, role: "orchestrator") {
    try {
      // Make HTTP request to update session metadata
      // Use localhost since plugin runs in same process as server
      const response = await fetch(`https://localhost:3348/session/${sessionId}`, {
        method: "PATCH",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({ metadata: { role } }),
      })

      if (response.ok) {
        log(`Session metadata updated via API: ${sessionId} role=${role}`)
      } else {
        const text = await response.text()
        throw new Error(`API returned ${response.status}: ${text}`)
      }
    } catch (err) {
      if (DEBUG) console.error(LOG_PREFIX, "Failed to update session metadata:", err)
    }
  }

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
        // Update session metadata immediately when detected via title
        await updateSessionMetadata(sessionId, "orchestrator")
      }

      // Detect worker from title patterns
      if (/\[[\w-]+-\w+\]/.test(sessionTitle) || /^(og|op)-(?!orch)/.test(sessionTitle.toLowerCase())) {
        sessionTypes.set(sessionId, "worker")
        log(`Worker session detected via title: ${sessionId}`)
      }
    },

    /**
     * Tool hook: Detect orchestrator context via Skill tool, and warn about Task tool usage.
     */
    "tool.execute.before": async (input, output) => {
      const { tool, sessionID } = input
      const { args } = output

      // Always run worker detection to populate cache
      const isWorker = isWorkerSession(sessionID, tool, args)

      // Detection: Check for orchestrator skill loading
      // This is the primary detection method for interactive sessions
      await detectOrchestratorFromSkill(sessionID, tool, args, updateSessionMetadata)

      // Only intercept Task tool for warnings
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
