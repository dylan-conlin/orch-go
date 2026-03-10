/**
 * Detection utilities: file classification, user message extraction,
 * premise-skipping detection, file line counting.
 */
import { existsSync, readFileSync } from "fs"
import { log, DEBUG, LOG_PREFIX } from "./coaching-types"

/**
 * Get line count for a file.
 * Returns the number of lines in the file, or null if the file cannot be read.
 */
export function getFileLineCount(filePath: string): number | null {
  if (!filePath || !existsSync(filePath)) {
    return null
  }

  try {
    const content = readFileSync(filePath, "utf-8")
    const lines = content.split("\n")
    return lines.length
  } catch (err) {
    if (DEBUG) console.error(LOG_PREFIX, `Failed to read file for line count: ${filePath}`, err)
    return null
  }
}

/**
 * Determine if a file path represents code (vs orchestration artifact).
 * Returns true if editing this file indicates frame collapse.
 */
export function isCodeFile(filePath: string): boolean {
  if (!filePath) return false

  const lowerPath = filePath.toLowerCase()

  // Orchestration directories - ALLOWED for orchestrators
  const orchestrationPaths = [
    "/.orch/",
    "/.kb/",
    "/.beads/",
    "/skills/",
    "/plugins/",
    "claude.md",
    "skill.md",
    "readme.md",
    "spawn_context.md",
    "synthesis.md",
    "session_handoff.md",
    "workspace.md",
    "agents.md",
  ]

  for (const orchPath of orchestrationPaths) {
    if (lowerPath.includes(orchPath)) {
      return false // Orchestration artifact, not code
    }
  }

  // Code file extensions - frame collapse indicators
  const codeExtensions = [
    ".go",
    ".ts",
    ".tsx",
    ".js",
    ".jsx",
    ".css",
    ".scss",
    ".less",
    ".sass",
    ".py",
    ".rb",
    ".java",
    ".rs",
    ".c",
    ".cpp",
    ".h",
    ".html",
    ".vue",
    ".svelte",
  ]

  for (const ext of codeExtensions) {
    if (lowerPath.endsWith(ext)) {
      return true // Code file
    }
  }

  return false // Unknown file type, not flagged
}

/**
 * Extract user messages from messages array.
 * Returns array of text content from user messages (role='user').
 */
export function extractUserMessages(
  messages: Array<{ info: any; parts: any[] }>
): Array<{ text: string; messageId: string }> {
  const userMessages: Array<{ text: string; messageId: string }> = []

  for (const msg of messages) {
    if (msg.info.role !== "user") continue

    for (const part of msg.parts) {
      if (part.type !== "text" || part.ignored || part.synthetic) continue
      if (part.text && part.text.trim()) {
        userMessages.push({
          text: part.text.trim(),
          messageId: msg.info.id || "unknown",
        })
      }
    }
  }

  return userMessages
}

/**
 * Detect premise-skipping question patterns.
 * Returns {matched: true, verb, extracted} if question assumes strategic direction
 * without premise validation.
 */
export function detectPremiseSkipping(text: string): { matched: boolean; verb?: string; extracted?: string } | null {
  const lowerText = text.toLowerCase()

  const strategicVerbs = [
    "migrate",
    "evolve",
    "fix",
    "centralize",
    "transition",
    "implement",
    "solve",
    "change",
    "shift",
    "move",
    "refactor",
    "rewrite",
    "rebuild",
  ]

  const howPatterns = [
    /how\s+(?:to|do\s+we|should\s+we|can\s+we)\s+(\w+)/gi,
  ]

  for (const pattern of howPatterns) {
    const matches = lowerText.matchAll(pattern)
    for (const match of matches) {
      const verb = match[1]

      if (strategicVerbs.includes(verb)) {
        if (lowerText.includes(" i ") || lowerText.includes("my ")) {
          continue // Skip tactical questions
        }

        return {
          matched: true,
          verb,
          extracted: match[0],
        }
      }
    }
  }

  return null
}
