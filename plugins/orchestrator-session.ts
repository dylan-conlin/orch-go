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
 * Detect if this session is a worker agent.
 *
 * Workers are identified by:
 * 1. ORCH_WORKER=1 environment variable (set by orch spawn)
 * 2. SPAWN_CONTEXT.md in the working directory
 * 3. Running from a .orch/workspace/ directory
 */
async function isWorker(directory: string | undefined): Promise<boolean> {
  // Check ORCH_WORKER env var (set by orch spawn)
  if (process.env.ORCH_WORKER === "1") {
    console.log(`${LOG_PREFIX} Worker detected: ORCH_WORKER=1`)
    return true
  }

  // Use process.cwd() if directory not provided
  const workDir = directory || process.cwd()

  // Check for SPAWN_CONTEXT.md (workers have this in their workspace)
  const spawnContextPath = join(workDir, "SPAWN_CONTEXT.md")
  if (await exists(spawnContextPath)) {
    console.log(`${LOG_PREFIX} Worker detected: SPAWN_CONTEXT.md found`)
    return true
  }

  // Check if path contains .orch/workspace/ (worker workspace directory)
  if (workDir.includes(".orch/workspace/")) {
    console.log(`${LOG_PREFIX} Worker detected: in .orch/workspace/`)
    return true
  }

  return false
}

/**
 * Check if an orchestrator session is already active.
 * Returns true if `orch session status` indicates an active session.
 */
async function hasActiveSession($: any): Promise<boolean> {
  try {
    const result = await $`orch session status 2>/dev/null`
    const output = result.stdout?.toString() || ""
    // If output contains "No active session", there's no active session
    return !output.includes("No active session")
  } catch {
    // Command failed - likely no session or orch not available
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
  console.log(`${LOG_PREFIX} Plugin initialized, directory:`, directory)

  const workingDir = typeof directory === "string" ? directory : process.cwd()

  // Check if this is an orch project (has .orch directory)
  const orchDir = await findOrchDirectory(workingDir)
  console.log(`${LOG_PREFIX} Orch dir check:`, orchDir)

  if (!orchDir) {
    console.log(`${LOG_PREFIX} Skipping - not an orch project`)
    return {}
  }

  // Check worker status once at init (shared across hooks)
  const worker = await isWorker(workingDir)
  console.log(`${LOG_PREFIX} Is worker:`, worker)

  if (worker) {
    console.log(`${LOG_PREFIX} Skipping - worker agent detected`)
    return {}
  }

  // Check if orchestrator skill exists
  const skillPath = join(homedir(), ".claude", "skills", "meta", "orchestrator", "SKILL.md")
  const skillExists = await exists(skillPath)
  console.log(`${LOG_PREFIX} Skill path:`, skillPath, "exists:", skillExists)

  return {
    /**
     * Config hook: Inject orchestrator skill into instructions.
     * Worker detection already handled at plugin init.
     */
    config: async (config) => {
      console.log(`${LOG_PREFIX} Config hook called`)

      // Skip if skill doesn't exist
      if (!skillExists) {
        console.log(`${LOG_PREFIX} Config: Orchestrator skill not found`)
        return
      }

      // Inject orchestrator skill into instructions
      if (!config.instructions) {
        config.instructions = []
      }
      if (!config.instructions.includes(skillPath)) {
        config.instructions.push(skillPath)
        console.log(`${LOG_PREFIX} Config: Added orchestrator skill to instructions`)
      }
    },

    /**
     * Event hook: Auto-start orchestrator session on session.created.
     * Worker detection already handled at plugin init.
     */
    event: async ({ event }) => {
      // Only handle session.created events
      if (event.type !== "session.created") {
        return
      }

      console.log(`${LOG_PREFIX} session.created event received`)

      // Check if orchestrator session already exists
      if (await hasActiveSession($)) {
        console.log(`${LOG_PREFIX} Event: Skipping - orchestrator session already active`)
        return
      }

      // Start orchestrator session
      console.log(`${LOG_PREFIX} Event: Starting orchestrator session...`)
      try {
        await $`orch session start`
        console.log(`${LOG_PREFIX} Event: Orchestrator session started successfully`)
      } catch (err) {
        console.error(`${LOG_PREFIX} Event: Failed to start session:`, err)
      }
    },
  }
}
