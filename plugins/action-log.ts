/**
 * Plugin: Log tool action outcomes for behavioral pattern detection.
 *
 * Triggered by: tool.execute.before and tool.execute.after (Read, Glob, Grep, Bash)
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
 *
 * Key implementation note: The 'tool.execute.before' hook receives args in output.args,
 * while 'tool.execute.after' receives title/output/metadata. We store args from the
 * before hook using callID as the key, then retrieve them in the after hook.
 */

import type { Plugin } from "@opencode-ai/plugin"
import { appendFileSync, mkdirSync, existsSync } from "fs"
import { createHash } from "crypto"
import { homedir } from "os"
import { join, dirname } from "path"

/**
 * Deduplication using file-based lock files.
 * 
 * This handles the case where the plugin is loaded as separate modules
 * (which can happen if OpenCode loads from both ~/.config/opencode/plugin/
 * and follows symlinks separately). 
 * 
 * For each event, we create a short-lived lock file based on the event hash.
 * If the lock file exists, the event is a duplicate. Lock files are cleaned
 * up after a short TTL (1 second).
 */
const DEDUP_WINDOW_MS = 100 // Bucket timestamps to 100ms windows
const DEDUP_LOCK_DIR = join(homedir(), ".orch", ".action-log-locks")
const LOCK_TTL_MS = 1000 // Lock files expire after 1 second

// In-memory cache for fast lookups (avoids filesystem calls for same-instance duplicates)
const recentHashes = new Set<string>()
let lastCleanupTime = 0

/**
 * Generate a hash for deduplication.
 * Uses session_id, timestamp (bucketed to 100ms), tool, and target.
 */
function getEventHash(
  sessionId: string | undefined,
  tool: string,
  target: string
): string {
  // Bucket timestamp to 100ms windows to catch near-duplicates
  const timestampBucket = Math.floor(Date.now() / DEDUP_WINDOW_MS)
  const key = `${sessionId || ""}:${timestampBucket}:${tool}:${target}`
  return createHash("md5").update(key).digest("hex").slice(0, 16)
}

/**
 * Clean up old lock files periodically.
 */
function cleanupOldLocks(): void {
  const now = Date.now()
  // Only cleanup every 5 seconds to avoid excessive I/O
  if (now - lastCleanupTime < 5000) {
    return
  }
  lastCleanupTime = now
  
  try {
    if (!existsSync(DEDUP_LOCK_DIR)) return
    
    const { readdirSync, statSync, unlinkSync } = require("fs")
    const files = readdirSync(DEDUP_LOCK_DIR)
    
    for (const file of files) {
      try {
        const lockPath = join(DEDUP_LOCK_DIR, file)
        const stats = statSync(lockPath)
        if (now - stats.mtimeMs > LOCK_TTL_MS) {
          unlinkSync(lockPath)
        }
      } catch {
        // Ignore per-file errors
      }
    }
  } catch {
    // Ignore cleanup errors
  }
  
  // Also clean in-memory cache
  if (recentHashes.size > 500) {
    recentHashes.clear()
  }
}

/**
 * Check if this event was recently logged (dedupe across plugin instances).
 * Uses file-based lock files for cross-process deduplication.
 */
function isDuplicateEvent(hash: string): boolean {
  // Fast path: check in-memory cache first
  if (recentHashes.has(hash)) {
    return true
  }
  
  try {
    // Ensure lock directory exists
    mkdirSync(DEDUP_LOCK_DIR, { recursive: true })
    
    const lockPath = join(DEDUP_LOCK_DIR, hash)
    
    // Try to create lock file with exclusive flag (fails if exists)
    // Using 'wx' flag: write exclusive - fails if file exists
    const { openSync, closeSync } = require("fs")
    const fd = openSync(lockPath, "wx")
    closeSync(fd)
    
    // Successfully created lock - this is not a duplicate
    recentHashes.add(hash)
    
    // Periodically clean up old locks
    cleanupOldLocks()
    
    return false
  } catch (err: any) {
    // EEXIST means lock file already exists - this is a duplicate
    if (err.code === "EEXIST") {
      return true
    }
    // Other errors - proceed without dedup to avoid blocking logging
    return false
  }
}

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

  // For Read tool, check for successful file content markers FIRST
  // This prevents false positives from error pattern matching
  if (tool.toLowerCase() === "read") {
    // Read tool returns content wrapped in <file>...</file> tags on success
    if (output.includes("<file>") || output.startsWith("00001|")) {
      return { outcome: "success" }
    }
  }

  // Check for error patterns in output
  // Only check at the START of output to avoid matching content within files
  const outputStart = output.slice(0, 500).toLowerCase()
  for (const pattern of ERROR_PATTERNS) {
    // Check if error pattern appears near the start (not buried in file content)
    if (outputStart.includes(pattern.toLowerCase())) {
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
  if (tool.toLowerCase() === "read") {
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
 * Log action event to file with deduplication.
 * 
 * Uses a global hash-based deduplication to prevent the same event
 * from being logged twice (which can happen if the plugin is loaded
 * multiple times by OpenCode).
 */
function logAction(event: ActionEvent): void {
  // Generate hash for deduplication BEFORE writing
  const hash = getEventHash(event.session_id, event.tool, event.target)
  
  // Check for duplicate (uses global module-level Set)
  if (isDuplicateEvent(hash)) {
    return // Skip duplicate
  }
  
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
 *
 * Uses a Map to store args from tool.execute.before (which has output.args)
 * and retrieve them in tool.execute.after (which has output.title/output/metadata).
 * The callID is used as the key to correlate before/after calls.
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

  // Store args from before hook, keyed by callID
  // Also track logged callIDs to prevent duplicate logging
  const pendingArgs = new Map<string, any>()
  const loggedCalls = new Set<string>()

  return {
    // Capture args from before hook (output.args is available here)
    "tool.execute.before": async (input: any, output: any) => {
      const tool = input.tool?.toLowerCase()

      // Only track specific tools
      if (!TRACKED_TOOLS.includes(tool)) {
        return
      }

      // Store args keyed by callID for retrieval in after hook
      if (input.callID && output.args) {
        pendingArgs.set(input.callID, output.args)
      }
    },

    // Log action outcome using stored args
    "tool.execute.after": async (input: any, output: any) => {
      const tool = input.tool?.toLowerCase()

      // Only track specific tools
      if (!TRACKED_TOOLS.includes(tool)) {
        return
      }

      // Prevent duplicate logging (hook may be called multiple times)
      if (input.callID && loggedCalls.has(input.callID)) {
        return
      }

      // Retrieve stored args from before hook
      const args = input.callID ? pendingArgs.get(input.callID) : undefined

      // Clean up stored args
      if (input.callID) {
        pendingArgs.delete(input.callID)
        loggedCalls.add(input.callID)
      }

      // Get result from output (structure may vary)
      const resultStr = output.output || ""

      // Determine outcome
      const { outcome, error_message } = determineOutcome(
        tool,
        resultStr,
        undefined
      )

      // Extract target from stored args
      const target = extractTarget(tool, args)

      // Build event
      const event: ActionEvent = {
        timestamp: new Date().toISOString(),
        tool: tool.charAt(0).toUpperCase() + tool.slice(1), // Capitalize
        target,
        outcome,
        error_message,
        workspace,
        session_id: input.sessionID,
      }

      // Log it
      logAction(event)

      // Clean up old logged calls to prevent memory leak
      // Keep only last 1000 entries
      if (loggedCalls.size > 1000) {
        const iterator = loggedCalls.values()
        for (let i = 0; i < 500; i++) {
          const val = iterator.next().value
          if (val) loggedCalls.delete(val)
        }
      }
    },
  }
}
