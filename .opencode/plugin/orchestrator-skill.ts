/**
 * Plugin: Orchestrator Skill Injection
 *
 * Injects the orchestrator skill (SKILL.md) into the system prompt for
 * non-worker sessions. Workers (spawned with ORCH_WORKER=1) skip injection
 * to save context budget (~37k tokens).
 *
 * Triggered by: experimental.chat.system.transform
 * Action: Appends orchestrator skill content to system prompt array
 *
 * Detection: Uses client.session.get() to check session.metadata.role.
 * OpenCode sets metadata.role = 'worker' when session is created with
 * x-opencode-env-ORCH_WORKER=1 header (set by orch spawn).
 */

import type { Plugin } from "@opencode-ai/plugin"
import { readFileSync } from "fs"
import { homedir } from "os"
import { join } from "path"

const LOG_PREFIX = "[orchestrator-skill]"
const DEBUG = process.env.ORCH_PLUGIN_DEBUG === "1"
const SKILL_PATH = join(
  homedir(),
  ".claude",
  "skills",
  "meta",
  "orchestrator",
  "SKILL.md",
)

function log(...args: any[]) {
  if (DEBUG) console.log(LOG_PREFIX, ...args)
}

export const OrchestratorSkillPlugin: Plugin = async ({ client }) => {
  log("Plugin initialized")

  // Cache skill content (read once at plugin init)
  let skillContent: string | null = null
  try {
    skillContent = readFileSync(SKILL_PATH, "utf-8")
    log(`Loaded orchestrator skill: ${skillContent.length} chars`)
  } catch (err) {
    console.error(LOG_PREFIX, `Failed to read ${SKILL_PATH}:`, err)
  }

  // Cache worker status per session to avoid repeated API calls
  const sessionRoleCache = new Map<string, boolean>() // sessionID -> isWorker

  async function isWorkerSession(sessionID: string): Promise<boolean> {
    const cached = sessionRoleCache.get(sessionID)
    if (cached !== undefined) return cached

    try {
      const res = await client.session.get({ path: { id: sessionID } })
      const isWorker = res.data?.metadata?.role === "worker"
      sessionRoleCache.set(sessionID, isWorker)
      log(
        `Session ${sessionID}: ${isWorker ? "worker" : "orchestrator/interactive"}`,
      )

      // Bound cache size
      if (sessionRoleCache.size > 200) {
        const firstKey = sessionRoleCache.keys().next().value
        if (firstKey) sessionRoleCache.delete(firstKey)
      }

      return isWorker
    } catch (err) {
      log(`Failed to fetch session ${sessionID}, assuming non-worker:`, err)
      return false
    }
  }

  return {
    "experimental.chat.system.transform": async (input, output) => {
      if (!skillContent) return

      // No sessionID means agent generation or similar — skip injection
      if (!input.sessionID) {
        log("No sessionID, skipping")
        return
      }

      // Workers don't need the orchestrator skill
      if (await isWorkerSession(input.sessionID)) {
        log(`Skipping worker session ${input.sessionID}`)
        return
      }

      // Inject orchestrator skill into system prompt
      output.system.push(skillContent)
      log(`Injected orchestrator skill for session ${input.sessionID}`)
    },
  }
}
