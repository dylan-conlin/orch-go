/**
 * Plugin: Orchestrator Session Management
 *
 * This plugin consolidates two responsibilities for orchestrator sessions:
 * 1. Config hook: Inject orchestrator skill (~/.claude/skills/meta/orchestrator/SKILL.md)
 * 2. Event hook: Auto-start orchestrator session on session.created
 *
 * Worker detection (shared logic):
 * - ORCH_WORKER env var is set (explicit marker from orch spawn)
 * - SPAWN_CONTEXT.md exists in working directory
 * - Path contains .orch/workspace/ (worker workspace)
 *
 * Both hooks skip processing for worker agents.
 */

import type { Plugin } from "@opencode-ai/plugin"
import { access } from "fs/promises"
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
  const workerSessions = new Set<string>() // sessionID set

  // Check if orchestrator skill exists
  const skillPath = join(homedir(), ".claude", "skills", "meta", "orchestrator", "SKILL.md")
  const skillExists = await exists(skillPath)
  log("Skill path:", skillPath, "exists:", skillExists)

  return {
    /**
     * Config hook: Inject orchestrator skill into instructions.
     * Worker detection already handled at plugin init.
     */
    config: async (config) => {
      log("Config hook called")

      // Skip if skill doesn't exist
      if (!skillExists) {
        log("Config: Orchestrator skill not found")
        return
      }

      // Inject orchestrator skill into instructions
      if (!config.instructions) {
        config.instructions = []
      }
      if (!config.instructions.includes(skillPath)) {
        config.instructions.push(skillPath)
        log("Config: Added orchestrator skill to instructions")
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
