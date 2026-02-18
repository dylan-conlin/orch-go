/**
 * Plugin: Orchestrator Session Management
 *
 * This plugin consolidates two responsibilities for orchestrator sessions:
 * 1. Lazy-load orchestrator skill via system.transform hook (only for non-workers)
 * 2. Auto-start orchestrator session on session.created event
 *
 * Worker detection (progressive, per-session):
 * - tool.execute.before hook detects workers by examining tool arguments
 * - SPAWN_CONTEXT.md read (workers always read this file early)
 * - .orch/workspace/ path (workers operate in workspace directories)
 * - Results cached in Map<sessionID, boolean> for performance
 *
 * Lazy-loading implementation:
 * - Orchestrator skill content read fresh from disk on each system transform
 * - experimental.chat.system.transform hook injects skill per-session
 * - Worker sessions detected progressively skip skill injection
 * - Non-worker sessions receive full orchestrator skill in system prompt
 */

import type { Plugin } from "@opencode-ai/plugin"
import { access, readFile } from "fs/promises"
import { join, resolve } from "path"
import { homedir } from "os"

const LOG_PREFIX = "[orchestrator-session]"
const DEBUG = process.env.ORCH_PLUGIN_DEBUG === "1"

function log(...args: any[]) {
  if (DEBUG) console.log(LOG_PREFIX, ...args)
}

/**
 * Check if a file exists at the given path.
 */
async function exists(path: string): Promise<boolean> {
  try {
    await access(path)
    return true
  } catch {
    return false
  }
}

/**
 * Find .orch directory by walking up from startDir.
 * Returns the path to .orch directory if found, null otherwise.
 */
async function findOrchDirectory(startDir: string): Promise<string | null> {
  let currentDir = resolve(startDir)

  // Check current directory first
  const orchPath = join(currentDir, ".orch")
  if (await exists(orchPath)) {
    return orchPath
  }

  // Walk up to 10 levels
  for (let i = 0; i < 10; i++) {
    const parentDir = resolve(currentDir, "..")
    if (parentDir === currentDir) break // Reached root

    const parentOrchPath = join(parentDir, ".orch")
    if (await exists(parentOrchPath)) {
      return parentOrchPath
    }
    currentDir = parentDir
  }

  return null
}

/**
 * NOTE: Worker detection cannot be done at plugin init level.
 * Plugin runs in OpenCode server process which never sees ORCH_WORKER env var.
 * Worker sessions are identified per-event using sessionID and workspace paths.
 * See workerSessions tracking in the plugin below.
 */

/**
 * Check if an orchestrator session is already active.
 * Checks for session.json file existence instead of running orch command
 * to avoid stdout leaking into the TUI.
 */
async function hasActiveSession(): Promise<boolean> {
  const sessionFile = join(homedir(), ".orch", "session.json")
  if (!(await exists(sessionFile))) {
    return false
  }
  
  try {
    const { readFile } = await import("fs/promises")
    const content = await readFile(sessionFile, "utf-8")
    const data = JSON.parse(content)
    // Session is active if it has an id and started timestamp
    return !!(data.session?.id && data.session?.started)
  } catch {
    return false
  }
}

/**
 * OpenCode plugin that manages orchestrator sessions.
 *
 * On config:
 * - Inject orchestrator skill into instructions (for non-workers)
 *
 * On session.created:
 * - Auto-start orchestrator session via `orch session start` (for non-workers)
 */
export const OrchestratorSessionPlugin: Plugin = async ({
  directory,
  $,
}) => {
  log("Plugin initialized, directory:", directory)

  const workingDir = typeof directory === "string" ? directory : process.cwd()

  // Check if this is an orch project (has .orch directory)
  const orchDir = await findOrchDirectory(workingDir)
  log("Orch dir check:", orchDir)

  if (!orchDir) {
    log("Skipping - not an orch project")
    return {}
  }

  // Worker session tracking (per-session detection, not plugin-level)
  // Plugin runs in server process, can't see ORCH_WORKER env from spawned agents
  const workerSessions = new Map<string, boolean>() // sessionID -> isWorker

  // Check orchestrator skill path (read fresh per system transform)
  const skillPath = join(homedir(), ".claude", "skills", "meta", "orchestrator", "SKILL.md")
  log("Skill path:", skillPath)

  async function loadSkillContent(): Promise<string | null> {
    try {
      const content = await readFile(skillPath, "utf-8")
      log("Loaded orchestrator skill content:", content.length, "bytes")
      return content
    } catch (err) {
      if (DEBUG) console.error(`${LOG_PREFIX} Failed to read orchestrator skill:`, err)
      return null
    }
  }
  
  /**
   * Detect if a session is a worker by examining tool args.
   * Returns true if worker detected, false otherwise.
   * IMPORTANT: Only caches positive results (isWorker=true) to avoid
   * permanently misclassifying workers based on their first tool call.
   */
  function detectWorkerSession(sessionId: string, tool: string, args: any): boolean {
    // Check cache first - only returns early if we've confirmed this IS a worker
    const cached = workerSessions.get(sessionId)
    if (cached === true) {
      return true
    }

    let isWorker = false

    // Detection signal 1: read tool accessing SPAWN_CONTEXT.md
    // Workers ALWAYS read this file early in their session.
    if (tool === "read" && args?.filePath) {
      if (args.filePath.endsWith("SPAWN_CONTEXT.md")) {
        log(`Worker detected (SPAWN_CONTEXT.md read): session ${sessionId}, file: ${args.filePath}`)
        isWorker = true
      }
    }

    // Detection signal 2: any tool accessing files in .orch/workspace/
    // Workers operate on files within their workspace directory.
    if (args?.filePath && typeof args.filePath === "string") {
      if (args.filePath.includes(".orch/workspace/")) {
        log(`Worker detected (filePath in workspace): session ${sessionId}, file: ${args.filePath}`)
        isWorker = true
      }
    }
    if (args?.file_path && typeof args.file_path === "string") {
      if (args.file_path.includes(".orch/workspace/")) {
        log(`Worker detected (file_path in workspace): session ${sessionId}, file: ${args.file_path}`)
        isWorker = true
      }
    }

    // Only cache positive results - don't cache false
    // This allows detection to succeed on later tool calls if first tools don't match
    if (isWorker) {
      workerSessions.set(sessionId, true)
      log(`Session ${sessionId} marked as worker (will NOT load orchestrator skill)`)
    }

    return isWorker
  }

  return {
    /**
     * Tool hook: Detect worker sessions by examining tool arguments.
     * Populates workerSessions cache for use by system.transform hook.
     */
    "tool.execute.before": async (input, output) => {
      const { tool, sessionID } = input
      const { args } = output
      
      // Run worker detection on this tool call
      detectWorkerSession(sessionID, tool, args)
    },
    
    /**
     * System transform hook: Conditionally inject orchestrator skill for non-worker sessions.
     * This implements lazy-loading: skill content only added when needed.
     */
    "experimental.chat.system.transform": async (input, output) => {
      const { sessionID } = input
      const { system } = output
      
      // Check if this session is a known worker
      const isWorker = workerSessions.get(sessionID) === true
      
      if (isWorker) {
        log(`System transform: Skipping orchestrator skill for worker session ${sessionID}`)
        return
      }
      
      // Not a worker - inject orchestrator skill content (read fresh each time)
      const skillContent = await loadSkillContent()
      if (!skillContent) {
        log(`System transform: Skill content not available for session ${sessionID}`)
        return
      }
      
      // Add orchestrator skill to system prompt
      if (!system.includes(skillContent)) {
        system.push(skillContent)
        log(`System transform: Injected orchestrator skill for session ${sessionID} (${skillContent.length} bytes)`)
      }
    },

    /**
     * Event hook: Auto-start orchestrator session on session.created.
     * Per-session worker detection via sessionID.
     */
    event: async ({ event }) => {
      // Only handle session.created events
      if (event.type !== "session.created") {
        return
      }

      log("session.created event received")

      // Get sessionID from event properties
      const sessionId = (event as any).properties?.sessionID
      if (!sessionId) {
        log("Event: No sessionID in event properties, skipping")
        return
      }

      // Detect worker sessions by checking if session is in a workspace directory
      // Note: This is a heuristic - worker sessions typically start in .orch/workspace/
      // We could also check for SPAWN_CONTEXT.md reads in future tool calls
      // For now, we don't have enough info at session.created time to detect workers
      // So we proceed and let workers potentially trigger orch session start (harmless)
      // TODO: Add more robust worker detection via tool hooks if needed

      // Check if orchestrator session already exists
      if (await hasActiveSession()) {
        log("Event: Skipping - orchestrator session already active")
        return
      }

      // Start orchestrator session
      log("Event: Starting orchestrator session...")
      try {
        // Redirect stdout to /dev/null to prevent TUI pollution
        await $`orch session start > /dev/null 2>&1`
        log("Event: Orchestrator session started successfully")
      } catch (err) {
        if (DEBUG) console.error(`${LOG_PREFIX} Event: Failed to start session:`, err)
      }
    },
  }
}
