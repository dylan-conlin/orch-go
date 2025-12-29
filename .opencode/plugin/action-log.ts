/**
 * Plugin: Log tool action outcomes for behavioral pattern detection.
 *
 * Triggered by: tool.execute.after (Read, Glob, Grep, Bash)
 * Purpose: Track action outcomes (success/empty/error) to enable detection
 * of repeated futile actions by the orchestrator.
 *
 * This addresses the gap identified in investigation orch-go-eq8k:
 * - Current mechanisms track knowledge state but not action outcomes
 * - Tool failures are ephemeral and untracked
 * - Behavioral pattern detection requires observing action outcomes
 *
 * Integration:
 * - Logs to ~/.orch/action-log.jsonl (JSONL format)
 * - Patterns surfaced via 'orch patterns' command
 * - Uses pkg/action ActionEvent format
 */

import type { Plugin } from "@opencode-ai/plugin"
import { appendFileSync, mkdirSync } from "fs"
import { homedir } from "os"
import { join, dirname } from "path"

// ActionEvent matches the Go struct in pkg/action/action.go
interface ActionEvent {
  timestamp: string
  tool: string
  target: string
  outcome: "success" | "empty" | "error" | "fallback"
  error_message?: string
  fallback_action?: string
  session_id?: string
  workspace?: string
  context?: string
}

// Tools we want to track for behavioral patterns
const TRACKED_TOOLS = ["read", "glob", "grep", "bash"]

// Patterns that indicate empty results
const EMPTY_PATTERNS = [
  "no matches found",
  "no files found",
  "file is empty",
  "0 results",
  "No results",
  "Pattern not found",
]

// Patterns that indicate errors
const ERROR_PATTERNS = [
  "Error:",
  "error:",
  "ENOENT",
  "EACCES",
  "command not found",
  "No such file or directory",
  "Permission denied",
  "failed to",
]

/**
 * Determine the outcome based on tool output.
 */
function determineOutcome(
  tool: string,
  output: string | undefined,
  error?: string
): { outcome: ActionEvent["outcome"]; error_message?: string } {
  // Explicit error takes priority
  if (error) {
    return { outcome: "error", error_message: error }
  }

  // No output or empty output
  if (!output || output.trim() === "") {
    return { outcome: "empty" }
  }

  // Check for error patterns in output
  for (const pattern of ERROR_PATTERNS) {
    if (output.includes(pattern)) {
      return { outcome: "error", error_message: output.slice(0, 200) }
    }
  }

  // Check for empty patterns in output
  for (const pattern of EMPTY_PATTERNS) {
    if (output.toLowerCase().includes(pattern.toLowerCase())) {
      return { outcome: "empty" }
    }
  }

  // For Read tool, check if content is meaningful
  if (tool === "read") {
    // Very short output might indicate an empty or near-empty file
    if (output.trim().length < 10) {
      return { outcome: "empty" }
    }
  }

  return { outcome: "success" }
}

/**
 * Extract target from tool arguments.
 */
function extractTarget(tool: string, args: any): string {
  if (!args) return "unknown"

  switch (tool.toLowerCase()) {
    case "read":
      return args.filePath || args.file_path || args.path || "unknown"
    case "glob":
      return args.pattern || "unknown"
    case "grep":
      return `${args.pattern || "*"}${args.path ? ` in ${args.path}` : ""}`
    case "bash":
      const cmd = args.command || ""
      // Truncate long commands
      return cmd.length > 100 ? cmd.slice(0, 97) + "..." : cmd
    default:
      return JSON.stringify(args).slice(0, 100)
  }
}

/**
 * Log action event to file.
 */
function logAction(event: ActionEvent): void {
  const logPath = join(homedir(), ".orch", "action-log.jsonl")

  try {
    // Ensure directory exists
    mkdirSync(dirname(logPath), { recursive: true })

    // Append JSON line
    const line = JSON.stringify(event) + "\n"
    appendFileSync(logPath, line)
  } catch (err) {
    // Silently fail - don't disrupt agent execution
    console.error("[action-log] Failed to log action:", err)
  }
}

/**
 * OpenCode plugin that logs tool action outcomes.
 */
export const ActionLogPlugin: Plugin = async ({
  project,
  client,
  $,
  directory,
  worktree,
}) => {
  // Detect workspace context from directory
  let workspace: string | undefined
  const workspaceMatch = directory?.match(/\.orch\/workspace\/([^/]+)/)
  if (workspaceMatch) {
    workspace = workspaceMatch[1]
  }

  return {
    "tool.execute.after": async (input: any, output: any) => {
      const tool = input.tool?.toLowerCase()

      // Only track specific tools
      if (!TRACKED_TOOLS.includes(tool)) {
        return
      }

      // Get result from output (structure depends on tool)
      const result = output.result
      const resultStr =
        typeof result === "string"
          ? result
          : result?.output || result?.content || JSON.stringify(result || "")

      // Determine outcome
      const { outcome, error_message } = determineOutcome(
        tool,
        resultStr,
        output.error
      )

      // Extract target
      const target = extractTarget(tool, output.args)

      // Build event
      const event: ActionEvent = {
        timestamp: new Date().toISOString(),
        tool: tool.charAt(0).toUpperCase() + tool.slice(1), // Capitalize
        target,
        outcome,
        error_message,
        workspace,
        session_id: input.session_id,
      }

      // Log it
      logAction(event)
    },
  }
}
