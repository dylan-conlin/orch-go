/**
 * Plugin: Slow Find Warning
 *
 * Mechanizes the "Gate Over Remind" principle for slow filesystem operations.
 * 
 * Detects when an agent is about to run a broad `find` command that will be slow
 * (e.g., `find ~/Documents` without maxdepth) and surfaces a warning before execution.
 *
 * Triggered by: tool.execute.before (Bash tool)
 * Action: Inject warning via client.session.prompt (does not block execution)
 * 
 * Problem: Claude autonomously generates `find ~/Documents` commands that take ~32s each.
 * Evidence: 68 occurrences in action log = 36+ min cumulative.
 * 
 * Reference: Task "Add hook to warn on slow find ~/Documents commands"
 */

import type { Plugin } from "@opencode-ai/plugin"

const LOG_PREFIX = "[slow-find-warn]"
const DEBUG = process.env.ORCH_PLUGIN_DEBUG === "1"

function log(...args: any[]) {
  if (DEBUG) console.log(LOG_PREFIX, ...args)
}

// Track commands we've already warned about in this session to avoid spam
const warnedCommands = new Set<string>()

/**
 * Check if a bash command is a slow find operation.
 * Returns true if command matches patterns like:
 * - find ~/Documents ... (without -maxdepth)
 * - find ~/ ... (without -maxdepth)
 * - find ~ ... (without -maxdepth)
 */
function isSlowFindCommand(command: string): boolean {
  if (!command) return false

  const trimmed = command.trim()

  // Must contain "find"
  if (!trimmed.includes("find")) return false

  // Must NOT already have maxdepth (they're already being careful)
  if (trimmed.includes("-maxdepth") || trimmed.includes("--maxdepth")) {
    return false
  }

  // Check for broad search patterns
  const slowPatterns = [
    /\bfind\s+~\/Documents/,  // find ~/Documents
    /\bfind\s+~\/(?:\s|$)/,   // find ~/ (followed by space or end)
    /\bfind\s+~\s/,           // find ~ (with space after)
    /\bfind\s+~$/,            // find ~ (end of line)
  ]

  for (const pattern of slowPatterns) {
    if (pattern.test(trimmed)) {
      return true
    }
  }

  return false
}

/**
 * Generate a normalized command key for deduplication.
 * E.g., "find ~/Documents -name foo" and "find ~/Documents -name bar" 
 * should be treated as the same pattern for warning purposes.
 */
function normalizeCommand(command: string): string {
  // Extract just the find command structure (base path + flags, not the search term)
  const match = command.match(/find\s+(~[^\s]*)\s+.*/)
  if (match) {
    return `find ${match[1]}` // e.g., "find ~/Documents"
  }
  return command
}

/**
 * Slow Find Warning Plugin
 *
 * Warns when agents run broad find commands without maxdepth.
 */
export const SlowFindWarnPlugin: Plugin = async ({
  project,
  client,
  $,
  directory,
  worktree,
}) => {
  log("Plugin initialized")

  return {
    "tool.execute.before": async (input, output) => {
      // Only check Bash tool calls
      if (input.tool !== "bash") {
        return
      }

      // Get the command being executed
      const command = output.args?.command as string | undefined
      if (!command) {
        return
      }

      log("Bash command detected:", command.substring(0, 100))

      // Check if this is a slow find command
      if (!isSlowFindCommand(command)) {
        log("Not a slow find command")
        return
      }

      log("Slow find command detected:", command)

      // Normalize command for deduplication
      const normalizedKey = normalizeCommand(command)

      // Skip if we've already warned about this pattern in this session
      if (warnedCommands.has(normalizedKey)) {
        log("Already warned about pattern:", normalizedKey)
        return
      }

      // Mark as warned
      warnedCommands.add(normalizedKey)

      // Clear set if it gets too large (memory management)
      if (warnedCommands.size > 100) {
        warnedCommands.clear()
        log("Cleared warned commands set (exceeded 100 entries)")
      }

      // Inject the warning
      // Note: We use noReply: true so this doesn't interrupt the command
      // The agent will see this warning but the command will still execute
      try {
        const warningMessage = `⚠️ **Slow Find Warning**

This command will be slow (~30s):
\`\`\`bash
${command}
\`\`\`

**Why:** Broad \`find\` searches without \`-maxdepth\` scan entire directory trees.

**Consider:**
- Add \`-maxdepth 3\` to limit depth
- Use project-scoped path instead of \`~/Documents\`
- Use \`orch locate <workspace-name>\` for workspace lookups (O(1))

**Example:**
\`\`\`bash
# Instead of:
find ~/Documents -name "*.md"

# Use:
find ~/Documents -maxdepth 3 -name "*.md"
# or
find /Users/dylanconlin/Documents/personal/orch-go -name "*.md"
\`\`\`

*This warning is informational - your command will still execute.*`

        await client.session.prompt({
          path: { id: input.sessionID },
          body: {
            noReply: true,
            parts: [
              {
                type: "text",
                text: warningMessage,
              },
            ],
          },
        })
        log("Warning injected for:", normalizedKey)
      } catch (err) {
        if (DEBUG) console.error(LOG_PREFIX, "Failed to inject warning:", err)
      }
    },
  }
}
