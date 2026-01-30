/**
 * Plugin: Dynamic HUD (Heads-Up Display) for Orchestrators and Workers
 *
 * Triggered by: experimental.chat.system.transform
 * When: Every LLM call (per-turn injection)
 * Purpose: Inject real-time contextual information for orchestrators and workers
 *
 * For orchestrators: spawn state (orch frontier), backlog health
 * For workers: beads context (bd show), active constraints
 *
 * ⚠️ EXPERIMENTAL API: The experimental.chat.system.transform hook may change
 * or be removed in future OpenCode versions.
 *
 * Reference: .kb/investigations/2026-01-27-inv-design-exploration-dynamic-hud-pattern.md
 */

import type { Plugin } from "@opencode-ai/plugin"
import { readFile, access } from "fs/promises"
import { join, dirname, resolve } from "path"

const LOG_PREFIX = "[orch-hud]"
const DEBUG = process.env.ORCH_PLUGIN_DEBUG === "1"
const MAX_HUD_TOKENS = 500 // Keep HUD under 500 tokens (~2000 chars)

function log(...args: any[]) {
  if (DEBUG) console.error(LOG_PREFIX, ...args)
}

// Role cache per session - stores "orchestrator" | "worker" | null (unknown)
const sessionRoles = new Map<string, "orchestrator" | "worker" | null>()
// Track sessions we've already checked via API (to avoid repeated lookups)
const sessionAPIChecked = new Set<string>()

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
 * Read file content safely, returning null on error.
 */
async function readFileSafe(path: string): Promise<string | null> {
  try {
    return (await readFile(path, "utf-8")).trim()
  } catch {
    return null
  }
}

/**
 * Find workspace directory by walking up from startDir.
 * Workspace is identified by presence of .tier or .beads_id file.
 */
async function findWorkspaceDir(startDir: string): Promise<string | null> {
  let currentDir = resolve(startDir)

  // Check current directory first
  if (await exists(join(currentDir, ".tier"))) {
    return currentDir
  }

  // Walk up to 10 levels
  for (let i = 0; i < 10; i++) {
    const parentDir = dirname(currentDir)
    if (parentDir === currentDir) break // Reached root

    if (await exists(join(parentDir, ".tier"))) {
      return parentDir
    }
    currentDir = parentDir
  }

  return null
}

/**
 * Detect worker from session title pattern.
 * Workers have beads ID [xxx-yyy] in title and are NOT orchestrators (-orch-).
 * 
 * Pattern copied from coaching.ts - same detection logic.
 */
function isWorkerByTitle(title: string): boolean {
  if (!title) return false
  const hasBeadsId = /\[[\w-]+-\w+\]/.test(title)
  const isOrchestratorTitle = /-orch-/.test(title) || /^meta-/.test(title)
  return hasBeadsId && !isOrchestratorTitle
}

/**
 * Async role detection via session API lookup.
 * Called once per session to get definitive answer from session title.
 * Returns "worker" | "orchestrator" based on title pattern.
 */
async function detectRoleViaAPI(sessionId: string, client: any): Promise<"orchestrator" | "worker" | null> {
  try {
    const sessions = await client.session.list()
    const session = sessions?.find((s: any) => s.id === sessionId)
    if (session?.title) {
      const isWorker = isWorkerByTitle(session.title)
      const role = isWorker ? "worker" : "orchestrator"
      log(`Role detected via API: ${sessionId} title="${session.title}" -> ${role}`)
      return role
    }
    log(`Session ${sessionId} not found or no title`)
  } catch (err) {
    log(`Failed to detect role via API: ${err}`)
  }
  return null
}

/**
 * Get beads issue ID from workspace .beads_id file.
 */
async function getBeadsIssueId(workspaceDir: string): Promise<string | null> {
  return readFileSafe(join(workspaceDir, ".beads_id"))
}

/**
 * Get beads issue context for workers.
 */
async function getBeadsContext($: any, issueId: string): Promise<{ phase: string; title: string; status: string } | null> {
  try {
    const result = await $`bd show ${issueId} --json`.quiet()
    const output = result.stdout.toString().trim()
    if (!output) return null

    const parsed = JSON.parse(output)
    // bd show returns an array with a single issue object
    const data = Array.isArray(parsed) ? parsed[0] : parsed
    if (!data) return null
    
    // Extract latest phase from comments
    let phase = "unknown"
    const comments = data.comments || []
    for (let i = comments.length - 1; i >= 0; i--) {
      const comment = comments[i]
      const text = typeof comment === "string" ? comment : comment.text || ""
      const phaseMatch = text.match(/Phase:\s*(\w+)/)
      if (phaseMatch) {
        phase = phaseMatch[1]
        break
      }
    }
    
    return {
      phase,
      title: data.title || "Unknown",
      status: data.status || "unknown"
    }
  } catch (err) {
    log("Failed to get beads context:", err)
    return null
  }
}

/**
 * Get spawn state for orchestrators via orch frontier.
 */
async function getSpawnState($: any): Promise<{ active: number; ready: number; blocked: number } | null> {
  try {
    const result = await $`orch frontier --json`.quiet()
    const output = result.stdout.toString().trim()
    if (!output) return null

    const data = JSON.parse(output)
    
    return {
      active: data.active?.length || 0,
      ready: data.ready?.length || 0,
      blocked: data.blocked?.length || 0
    }
  } catch (err) {
    log("Failed to get spawn state:", err)
    return null
  }
}

/**
 * Get backlog health for orchestrators.
 */
async function getBacklogHealth($: any): Promise<{ triageReady: number } | null> {
  try {
    const result = await $`bd list -l triage:ready --json`.quiet()
    const output = result.stdout.toString().trim()
    if (!output) return null

    const data = JSON.parse(output)
    
    return {
      triageReady: Array.isArray(data) ? data.length : 0
    }
  } catch (err) {
    log("Failed to get backlog health:", err)
    return null
  }
}

/**
 * Get recent constraints from kb.
 */
async function getRecentConstraints($: any): Promise<string[]> {
  try {
    const result = await $`kb quick list --type constraint --json`.quiet()
    const output = result.stdout.toString().trim()
    if (!output) return []

    const data = JSON.parse(output)
    // Take first 3 constraints
    const constraints = Array.isArray(data) ? data.slice(0, 3) : []
    return constraints.map((c: any) => c.content || c.text || c.id)
  } catch (err) {
    log("Failed to get constraints:", err)
    return []
  }
}

/**
 * Build HUD content for orchestrators.
 */
function buildOrchestratorHUD(data: {
  spawnState: { active: number; ready: number; blocked: number } | null
  backlogHealth: { triageReady: number } | null
  constraints: string[]
}): string {
  const lines: string[] = []

  lines.push("---")
  lines.push("## 🎯 ORCHESTRATOR HUD")
  lines.push("")

  // Spawn state
  if (data.spawnState) {
    const { active, ready, blocked } = data.spawnState
    lines.push("### Active Spawns")
    lines.push(`- **Active:** ${active} agent${active !== 1 ? 's' : ''}`)
    lines.push(`- **Ready:** ${ready} issue${ready !== 1 ? 's' : ''}`)
    if (blocked > 0) {
      lines.push(`- ⚠️ **Blocked:** ${blocked} issue${blocked !== 1 ? 's' : ''}`)
    }
    lines.push("")
  }

  // Backlog health
  if (data.backlogHealth) {
    const { triageReady } = data.backlogHealth
    lines.push("### Backlog")
    lines.push(`- **Triage Ready:** ${triageReady} issue${triageReady !== 1 ? 's' : ''}`)
    if (triageReady === 0) {
      lines.push("  - ⚠️ Daemon queue is empty")
    }
    lines.push("")
  }

  // Constraints
  if (data.constraints.length > 0) {
    lines.push("### Active Constraints")
    for (const c of data.constraints.slice(0, 2)) {
      // Truncate long constraints
      const truncated = c.length > 80 ? c.substring(0, 77) + "..." : c
      lines.push(`- ${truncated}`)
    }
    lines.push("")
  }

  lines.push("*Auto-updated per turn via experimental.chat.system.transform*")
  lines.push("---")

  return lines.join("\n")
}

/**
 * Build HUD content for workers.
 */
function buildWorkerHUD(data: {
  beadsId: string | null
  beadsContext: { phase: string; title: string; status: string } | null
  constraints: string[]
}): string {
  const lines: string[] = []

  lines.push("---")
  lines.push("## 🔧 WORKER HUD")
  lines.push("")

  // Beads issue context
  if (data.beadsId && data.beadsContext) {
    const { phase, title, status } = data.beadsContext
    lines.push("### Current Issue")
    lines.push(`- **ID:** ${data.beadsId}`)
    lines.push(`- **Title:** ${title}`)
    lines.push(`- **Status:** ${status}`)
    lines.push(`- **Phase:** ${phase}`)
    lines.push("")
    lines.push("**Progress Reporting:**")
    lines.push(`\`bd comment ${data.beadsId} "Phase: ... - [details]"\``)
    lines.push("")
  }

  // Constraints
  if (data.constraints.length > 0) {
    lines.push("### Active Constraints")
    for (const c of data.constraints.slice(0, 3)) {
      // Truncate long constraints
      const truncated = c.length > 80 ? c.substring(0, 77) + "..." : c
      lines.push(`- ${truncated}`)
    }
    lines.push("")
  }

  lines.push("**Completion Protocol:**")
  lines.push("1. Report: `bd comment <id> \"Phase: Complete - [summary]\"`")
  lines.push("2. Commit all changes")
  lines.push("3. Run `/exit`")
  lines.push("")
  lines.push("*Auto-updated per turn via experimental.chat.system.transform*")
  lines.push("---")

  return lines.join("\n")
}

/**
 * Truncate HUD content to stay under token limit.
 * Rough estimate: 1 token ≈ 4 characters
 */
function truncateHUD(content: string, maxTokens: number): string {
  const maxChars = maxTokens * 4
  if (content.length <= maxChars) {
    return content
  }
  
  log(`HUD content truncated from ${content.length} to ${maxChars} chars`)
  return content.substring(0, maxChars - 50) + "\n\n...[truncated]...\n---"
}

/**
 * Orch HUD Plugin
 *
 * Injects dynamic HUD content on every LLM call using experimental.chat.system.transform.
 * Provides orchestrators with spawn state and backlog health.
 * Provides workers with beads context and active constraints.
 * 
 * NOTE: Role detection uses session title via API, not ORCH_WORKER env var.
 * Plugin runs in server process, can't see env vars from spawned agents.
 */
export const OrchHUDPlugin: Plugin = async ({
  directory,
  $,
  client,
}) => {
  log("Plugin initialized, directory:", directory)

  const workingDir = typeof directory === "string" ? directory : process.cwd()
  const workspaceDir = await findWorkspaceDir(workingDir)

  return {
    /**
     * Hook: experimental.chat.system.transform
     *
     * Called on every LLM call. We inject HUD content into the system prompt.
     * Role detection happens per-session via title pattern matching.
     */
    "experimental.chat.system.transform": async (
      input: { sessionID?: string; model: { providerID: string; modelID: string } },
      output: { system: string[] }
    ) => {
      const sessionId = input.sessionID || "unknown"
      log("System transform triggered for session:", sessionId)

      // Get role from cache or detect via API
      let role = sessionRoles.get(sessionId)
      if (role === undefined && !sessionAPIChecked.has(sessionId)) {
        sessionAPIChecked.add(sessionId)
        role = await detectRoleViaAPI(sessionId, client)
        if (role) {
          sessionRoles.set(sessionId, role)
        }
      }

      // If still unknown, skip HUD injection
      if (!role) {
        log("Role unknown for session:", sessionId, "- skipping HUD")
        return
      }

      log("Role for session:", sessionId, "is:", role)

      let hudContent = ""

      if (role === "orchestrator") {
        // Get orchestrator-specific data
        const spawnState = await getSpawnState($)
        const backlogHealth = await getBacklogHealth($)
        const constraints = await getRecentConstraints($)

        hudContent = buildOrchestratorHUD({
          spawnState,
          backlogHealth,
          constraints
        })
      } else if (role === "worker" && workspaceDir) {
        // Get worker-specific data
        const beadsId = await getBeadsIssueId(workspaceDir)
        const beadsContext = beadsId ? await getBeadsContext($, beadsId) : null
        const constraints = await getRecentConstraints($)

        hudContent = buildWorkerHUD({
          beadsId,
          beadsContext,
          constraints
        })
      }

      // Truncate if needed
      hudContent = truncateHUD(hudContent, MAX_HUD_TOKENS)

      // Inject into system prompt array
      if (hudContent) {
        output.system.push(hudContent)
        log("Injected HUD content for", role)
      }
    },
  }
}
