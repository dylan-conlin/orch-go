/**
 * Plugin: Orchestrator Tool Gate
 *
 * Prevents frame collapse by blocking orchestrators from using primitive tools.
 * Orchestrators should operate in meta-action space (orch, bd, kb) not primitives (edit, write, bash).
 *
 * Allowlist (meta-actions only):
 * - bash: orch spawn/complete/status/review, bd create/show/ready/close, kb context/quick, git status
 * - read: CLAUDE.md, .kb/*.md, .orch/*.md, SYNTHESIS.md (artifacts, not code)
 *
 * Block with explanation:
 * - edit: 'Orchestrators operate in meta-action space. Use orch spawn instead.'
 * - write: Same message
 * - read (code files): 'Orchestrators read artifacts, not code. Spawn a worker to investigate.'
 * - bash (most commands): 'Workers execute commands, not orchestrators.'
 *
 * Emergency override:
 * - Check args for force_primitive=true or similar flag
 * - Log override usage to metrics
 * - Should be <1% of orchestrator sessions
 *
 * Detection methods (reuses task-tool-gate.ts patterns):
 * 1. Skill loading: When "orchestrator" skill is loaded via Skill tool
 * 2. Session title pattern: Orchestrator sessions have titles starting with "og-" or "op-"
 *    WITHOUT beads ID brackets (workers have [beads-id])
 * 3. Workspace path: Workers operate in .orch/workspace/ directories
 *
 * Reference: orch-go-20962 (Phase 3: Registry-level tool gating for orchestrators)
 */

import type { Plugin } from "@opencode-ai/plugin"

const LOG_PREFIX = "[orchestrator-tool-gate]"
const DEBUG = process.env.ORCH_PLUGIN_DEBUG === "1"

function log(...args: any[]) {
  if (DEBUG) console.log(LOG_PREFIX, ...args)
}

/**
 * Track detected session types.
 * Maps sessionID -> type (orchestrator | worker | unknown).
 */
const sessionTypes = new Map<string, "orchestrator" | "worker" | "unknown">()

/**
 * Track warned sessions to avoid duplicate warnings.
 */
const warnedSessions = new Set<string>()

/**
 * Track emergency override usage for metrics.
 * Maps sessionID -> count of overrides.
 */
const overrideMetrics = new Map<string, number>()

/**
 * Allowlisted bash commands for orchestrators.
 * These are meta-actions that orchestrators are allowed to run.
 */
const BASH_ALLOWLIST_PATTERNS = [
  // orch commands
  /^orch\s+spawn\b/,
  /^orch\s+complete\b/,
  /^orch\s+status\b/,
  /^orch\s+review\b/,
  /^orch\s+servers\b/, // Added for server management
  /^orch\s+tail\b/,    // Added for monitoring spawns
  
  // bd (beads) commands
  /^bd\s+(create|show|ready|close|comment|comments|update)\b/,
  
  // kb commands
  /^kb\s+(context|quick)\b/,
  
  // git read-only commands
  /^git\s+status\b/,
  /^git\s+log\b/,
  /^git\s+diff\b/,
  /^git\s+show\b/,
  /^git\s+branch\b/,
  
  // File inspection (read-only, for debugging)
  /^ls\b/,
  /^pwd\b/,
  /^echo\b/,
  /^env\b/,
  /^printenv\b/,
  /^which\b/,
  /^whereis\b/,
  /^type\b/,
  
  // Process inspection
  /^ps\b/,
  /^pgrep\b/,
  
  // Help commands
  /^man\b/,
  /^--help\b/,
  /^help\b/,
]

/**
 * Allowlisted file patterns for read tool.
 * Orchestrators can read artifacts (documentation, config), not code.
 */
const READ_ALLOWLIST_PATTERNS = [
  // Documentation and artifacts
  /CLAUDE\.md$/i,
  /SYNTHESIS\.md$/i,
  /SPAWN_CONTEXT\.md$/i,
  /README\.md$/i,
  /\.kb\/.*\.md$/,
  /\.orch\/.*\.md$/,
  /docs\/.*\.md$/,
  
  // Config files (read-only inspection)
  /package\.json$/,
  /tsconfig\.json$/,
  /\.opencode\/opencode\.json$/,
  /Makefile$/,
  /go\.mod$/,
  /go\.sum$/,
  
  // Project metadata
  /\.git\//,
  /\.gitignore$/,
  /\.env\.example$/,
]

/**
 * Warning messages for different tool blocks.
 */
const MESSAGES = {
  EDIT: `## Orchestrator Tool Gate: Edit Blocked

You attempted to use the **Edit tool** in an orchestrator context.

**Why this is blocked:**
Orchestrators operate in meta-action space, not implementation space.

**Correct approach:**
\`\`\`bash
# Spawn a worker to make the change
orch spawn feature-impl "description of change" --issue BEADS-ID

# Or for quick fixes
orch spawn surgical-change "specific edit needed" --issue BEADS-ID
\`\`\`

**Emergency override:**
If you have a legitimate reason to edit directly (e.g., fixing CLAUDE.md or .kb/ files):
Add \`force_primitive: true\` to your tool arguments.

**This override is logged and should be rare (<1% of sessions).**
`,

  WRITE: `## Orchestrator Tool Gate: Write Blocked

You attempted to use the **Write tool** in an orchestrator context.

**Why this is blocked:**
Orchestrators operate in meta-action space, not implementation space.

**Correct approach:**
\`\`\`bash
# Spawn a worker to create the file
orch spawn feature-impl "create new file with X" --issue BEADS-ID
\`\`\`

**Emergency override:**
If you have a legitimate reason to write directly (e.g., creating .kb/ documentation):
Add \`force_primitive: true\` to your tool arguments.

**This override is logged and should be rare (<1% of sessions).**
`,

  READ_CODE: `## Orchestrator Tool Gate: Read Code Blocked

You attempted to **read code files** in an orchestrator context.

**Why this is blocked:**
Orchestrators read artifacts (CLAUDE.md, .kb/, .orch/), not implementation code.

**Correct approach:**
\`\`\`bash
# Spawn a worker to investigate the code
orch spawn investigation "explore [topic]" --issue BEADS-ID

# Or for debugging
orch spawn systematic-debugging "symptom description" --issue BEADS-ID
\`\`\`

**Allowed files:**
- Documentation: CLAUDE.md, README.md, .kb/*.md, .orch/*.md
- Config: package.json, tsconfig.json, go.mod (read-only inspection)

**Emergency override:**
If you have a legitimate reason to read code directly:
Add \`force_primitive: true\` to your tool arguments.

**This override is logged and should be rare (<1% of sessions).**
`,

  BASH_COMMAND: `## Orchestrator Tool Gate: Bash Command Blocked

You attempted to run a **non-meta command** in an orchestrator context.

**Why this is blocked:**
Workers execute commands, not orchestrators.

**Allowed commands (meta-actions only):**
- orch spawn, complete, status, review, servers, tail
- bd create, show, ready, close, comment, update
- kb context, quick
- git status, log, diff, show (read-only)
- ls, pwd, echo (inspection only)

**Correct approach:**
\`\`\`bash
# Spawn a worker to run commands and implement
orch spawn feature-impl "task description" --issue BEADS-ID
\`\`\`

**Emergency override:**
If you have a legitimate reason to run this command directly:
Add \`force_primitive: true\` to your tool arguments.

**This override is logged and should be rare (<1% of sessions).**
`,
}

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
 * When the "orchestrator" skill is loaded, mark the session as orchestrator.
 */
async function detectOrchestratorFromSkill(
  sessionId: string,
  tool: string,
  args?: any,
): Promise<boolean> {
  if (tool.toLowerCase() !== "skill") return false

  // Skill tool uses 'skill' parameter (from SPAWN_CONTEXT) or 'name' parameter (from OpenCode)
  const skillName = args?.skill || args?.name
  if (!skillName) return false

  // Check if orchestrator skill is being loaded
  if (skillName.toLowerCase() === "orchestrator") {
    log(`Orchestrator detected (skill loading): session ${sessionId}`)
    sessionTypes.set(sessionId, "orchestrator")
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
 * Check if emergency override flag is set in tool arguments.
 */
function hasEmergencyOverride(args?: any): boolean {
  if (!args) return false
  
  // Check for force_primitive flag in various forms
  return !!(
    args.force_primitive === true ||
    args.forcePrimitive === true ||
    args.override === true ||
    args.emergency_override === true
  )
}

/**
 * Log emergency override usage for metrics.
 */
function logOverride(sessionId: string, tool: string, reason: string) {
  const count = (overrideMetrics.get(sessionId) || 0) + 1
  overrideMetrics.set(sessionId, count)
  
  console.log(`${LOG_PREFIX} EMERGENCY OVERRIDE: session=${sessionId} tool=${tool} count=${count} reason=${reason}`)
  
  // Clean up old entries if too many
  if (overrideMetrics.size > 100) {
    const toDelete = Array.from(overrideMetrics.keys()).slice(0, 50)
    toDelete.forEach((id) => overrideMetrics.delete(id))
  }
}

/**
 * Check if a bash command is allowed for orchestrators.
 */
function isBashCommandAllowed(command: string): boolean {
  if (!command) return false
  
  // Trim the command for pattern matching
  const trimmed = command.trim()
  
  // Check against allowlist patterns
  return BASH_ALLOWLIST_PATTERNS.some(pattern => pattern.test(trimmed))
}

/**
 * Check if a file path is allowed for read tool.
 */
function isReadPathAllowed(filePath: string): boolean {
  if (!filePath) return false
  
  // Check against allowlist patterns
  return READ_ALLOWLIST_PATTERNS.some(pattern => pattern.test(filePath))
}

/**
 * Inject warning message to session.
 */
async function injectWarning(
  client: any,
  sessionId: string,
  message: string,
  tool: string
): Promise<void> {
  // Avoid duplicate warnings
  const key = `${sessionId}:${tool}`
  if (warnedSessions.has(key)) {
    log(`Already warned session ${sessionId} about ${tool}`)
    return
  }

  log(`Injecting ${tool} warning for session ${sessionId}`)

  // Mark as warned
  warnedSessions.add(key)

  // Clean up old entries if too many
  if (warnedSessions.size > 100) {
    const toDelete = Array.from(warnedSessions).slice(0, 50)
    toDelete.forEach((k) => warnedSessions.delete(k))
  }

  // Inject warning message
  try {
    await client.session.prompt({
      path: { id: sessionId },
      body: {
        noReply: true,
        parts: [
          {
            type: "text",
            text: message,
          },
        ],
      },
    })
    log(`Warning injected for session ${sessionId}`)
  } catch (err) {
    if (DEBUG) console.error(LOG_PREFIX, "Failed to inject warning:", err)
  }
}

/**
 * OpenCode plugin that gates tool usage for orchestrators.
 */
export const OrchestratorToolGatePlugin: Plugin = async ({
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
     * Tool hook: Detect orchestrator context and gate tool usage.
     */
    "tool.execute.before": async (input, output) => {
      const { tool, sessionID } = input
      const { args } = output

      // Always run worker detection to populate cache
      const isWorker = isWorkerSession(sessionID, tool, args)

      // Detection: Check for orchestrator skill loading
      await detectOrchestratorFromSkill(sessionID, tool, args)

      // Skip gating for workers
      if (isWorker) {
        return
      }

      // Check if this is an orchestrator session
      const sessionType = sessionTypes.get(sessionID)
      
      // Only gate confirmed orchestrators
      if (sessionType !== "orchestrator") {
        return
      }

      log(`Tool gate check: session=${sessionID} tool=${tool} type=${sessionType}`)

      // Check for emergency override
      if (hasEmergencyOverride(args)) {
        logOverride(sessionID, tool, `override requested via args`)
        return // Allow with logging
      }

      // Gate tools based on type
      const lowerTool = tool.toLowerCase()

      // Block Edit tool
      if (lowerTool === "edit") {
        await injectWarning(client, sessionID, MESSAGES.EDIT, tool)
        throw new Error("Orchestrator tool gate: Edit tool blocked. Use 'orch spawn' instead or add force_primitive: true to override.")
      }

      // Block Write tool
      if (lowerTool === "write") {
        await injectWarning(client, sessionID, MESSAGES.WRITE, tool)
        throw new Error("Orchestrator tool gate: Write tool blocked. Use 'orch spawn' instead or add force_primitive: true to override.")
      }

      // Gate Read tool (allow artifacts, block code)
      if (lowerTool === "read") {
        const filePath = args?.filePath || args?.file_path
        if (filePath && !isReadPathAllowed(filePath)) {
          await injectWarning(client, sessionID, MESSAGES.READ_CODE, tool)
          throw new Error(`Orchestrator tool gate: Read blocked for code file '${filePath}'. Spawn a worker to investigate or add force_primitive: true to override.`)
        }
      }

      // Gate Bash tool (allow meta-commands, block implementation commands)
      if (lowerTool === "bash") {
        const command = args?.command
        if (command && !isBashCommandAllowed(command)) {
          await injectWarning(client, sessionID, MESSAGES.BASH_COMMAND, tool)
          throw new Error(`Orchestrator tool gate: Bash command '${command}' blocked. Use allowed meta-commands or add force_primitive: true to override.`)
        }
      }
    },
  }
}
