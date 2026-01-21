/**
 * Plugin: Evidence Hierarchy - Warn on edit without prior search.
 *
 * Principle: "Code is truth. Artifacts are hypotheses."
 * The test: "Did the agent grep/search before claiming something exists or doesn't exist?"
 *
 * Mechanism:
 * - Track grep/glob/read calls via tool.execute.after (stores searched files/patterns)
 * - When Edit tool is called, check if the target file was searched/read first
 * - If not, inject warning via client.session.prompt suggesting to search first
 *
 * False positive mitigation:
 * - Skip warning if file was created in this session (new files don't need search)
 * - Skip warning if file was read (Read tool counts as evidence gathering)
 * - Skip warning for generated/config files (package.json, tsconfig.json, etc.)
 * - Track searches by directory prefix to handle pattern-based evidence
 *
 * Reference: ~/.kb/principles.md (Evidence Hierarchy section)
 */

import type { Plugin } from "@opencode-ai/plugin"
import { homedir } from "os"
import { join, dirname, basename } from "path"

const LOG_PREFIX = "[evidence-hierarchy]"
const DEBUG = process.env.ORCH_PLUGIN_DEBUG === "1"

function log(...args: any[]) {
  if (DEBUG) console.log(LOG_PREFIX, ...args)
}

/**
 * Files that don't need evidence gathering before editing.
 * These are typically generated or configuration files where
 * the agent knows the format from documentation.
 */
const EXEMPT_PATTERNS = [
  // Config files
  /package\.json$/,
  /tsconfig\.json$/,
  /\.json$/, // Most JSON config files
  /\.yaml$/,
  /\.yml$/,
  /\.toml$/,
  /\.env/,
  // Generated files
  /\.lock$/,
  /node_modules\//,
  /dist\//,
  /build\//,
  // Documentation that doesn't need code search
  /README\.md$/,
  /CHANGELOG\.md$/,
  // Git files
  /\.gitignore$/,
  /\.gitattributes$/,
  // Investigation/workspace files (agent-generated)
  /SYNTHESIS\.md$/,
  /SPAWN_CONTEXT\.md$/,
  /SESSION_HANDOFF\.md$/,
  /\.kb\/investigations\//,
  /\.kb\/decisions\//,
  /\.orch\/workspace\//,
]

/**
 * Check if a file path matches any exempt pattern.
 */
function isExemptFile(filePath: string): boolean {
  return EXEMPT_PATTERNS.some((pattern) => pattern.test(filePath))
}

/**
 * Normalize a file path for consistent matching.
 * Handles home directory expansion and removes trailing slashes.
 */
function normalizePath(filePath: string): string {
  if (!filePath) return ""
  // Expand ~ to home directory
  if (filePath.startsWith("~")) {
    filePath = join(homedir(), filePath.slice(1))
  }
  // Remove trailing slash
  return filePath.replace(/\/$/, "")
}

/**
 * Extract file path from Edit tool arguments.
 */
function extractEditFilePath(args: any): string | null {
  if (!args) return null
  return args.filePath || args.file_path || args.path || null
}

/**
 * Extract target from search/read tools.
 * Returns either a specific file path or a directory path (for pattern searches).
 */
function extractSearchTarget(tool: string, args: any): { files: string[]; directories: string[] } {
  const result: { files: string[]; directories: string[] } = { files: [], directories: [] }

  if (!args) return result

  switch (tool.toLowerCase()) {
    case "read":
      // Read targets specific files
      const readPath = args.filePath || args.file_path || args.path
      if (readPath) {
        result.files.push(normalizePath(readPath))
      }
      break

    case "grep":
      // Grep searches in a directory (or cwd if not specified)
      // Include pattern may narrow files, but we consider the directory searched
      const grepPath = args.path || process.cwd()
      result.directories.push(normalizePath(grepPath))
      // If there's an include pattern, we can extract file extensions being searched
      if (args.include) {
        log("Grep include pattern:", args.include)
      }
      break

    case "glob":
      // Glob finds files matching a pattern in a directory
      const globPath = args.path || process.cwd()
      result.directories.push(normalizePath(globPath))
      break

    case "bash":
      // Bash commands might include grep, find, etc.
      // Track the working directory as searched
      const bashPath = args.workdir || process.cwd()
      result.directories.push(normalizePath(bashPath))
      break
  }

  return result
}

/**
 * Check if a file was searched (either directly or via directory search).
 */
function wasFileSearched(
  filePath: string,
  searchedFiles: Set<string>,
  searchedDirectories: Set<string>
): boolean {
  const normalizedPath = normalizePath(filePath)

  // Check if file was directly read
  if (searchedFiles.has(normalizedPath)) {
    return true
  }

  // Check if any searched directory is a parent of the file
  const fileDir = dirname(normalizedPath)
  const dirs = Array.from(searchedDirectories)
  for (let i = 0; i < dirs.length; i++) {
    const dir = dirs[i]
    if (normalizedPath.startsWith(dir) || fileDir.startsWith(dir)) {
      return true
    }
  }

  return false
}

/**
 * Generate the warning message for editing without evidence.
 */
function generateWarning(filePath: string): string {
  const fileName = basename(filePath)
  return `<system-reminder>
⚠️ **Evidence Hierarchy Warning**

You are editing \`${fileName}\` without first searching/reading it in this session.

**The Evidence Hierarchy principle states:** "Code is truth. Artifacts are hypotheses."

**The test:** Did you grep/search before making claims about what exists or doesn't exist?

**Recommendation:** Before editing unfamiliar code, use:
- \`grep\` to search for patterns/function names
- \`glob\` to find related files
- \`read\` to understand existing implementation

This warning appears because no search/read of this file or its directory was detected in this session.

**Why this matters:** Agents can hallucinate or make incorrect assumptions about code without first verifying. Searching first creates provenance for your changes.

If you have already gathered evidence through other means (e.g., conversation context), you can proceed with the edit.
</system-reminder>`
}

/**
 * OpenCode plugin that enforces the Evidence Hierarchy principle.
 *
 * Tracks search/read operations and warns when editing files that
 * haven't been searched in the current session.
 */
export const EvidenceHierarchyPlugin: Plugin = async ({
  project,
  client,
  $,
  directory,
  worktree,
}) => {
  log("Plugin initialized, directory:", directory)

  // Track files and directories that have been searched/read in this session
  // These are session-local (reset on plugin reload)
  const searchedFiles = new Set<string>()
  const searchedDirectories = new Set<string>()

  // Track files created in this session (don't warn for new files)
  const createdFiles = new Set<string>()

  // Track files that have already triggered a warning (only warn once per file)
  const warnedFiles = new Set<string>()

  // Store args from before hook for retrieval in after hook
  const pendingArgs = new Map<string, any>()

  return {
    /**
     * Before hook: Capture args and check Edit operations.
     * This is where we can inject warnings before the edit happens.
     */
    "tool.execute.before": async (input: any, output: any) => {
      const tool = input.tool?.toLowerCase()

      // Store args for all tools (needed in after hook)
      if (input.callID && output.args) {
        pendingArgs.set(input.callID, output.args)
      }

      // Only check Edit tool
      if (tool !== "edit") {
        return
      }

      const args = output.args
      const filePath = extractEditFilePath(args)

      if (!filePath) {
        log("Edit: No file path found in args")
        return
      }

      const normalizedPath = normalizePath(filePath)
      log("Edit detected for:", normalizedPath)

      // Skip if file is exempt
      if (isExemptFile(normalizedPath)) {
        log("Edit: File is exempt:", normalizedPath)
        return
      }

      // Skip if file was created this session (Write tool)
      if (createdFiles.has(normalizedPath)) {
        log("Edit: File was created this session:", normalizedPath)
        return
      }

      // Skip if already warned about this file
      if (warnedFiles.has(normalizedPath)) {
        log("Edit: Already warned about this file:", normalizedPath)
        return
      }

      // Check if file was searched/read
      if (wasFileSearched(normalizedPath, searchedFiles, searchedDirectories)) {
        log("Edit: File was searched:", normalizedPath)
        return
      }

      // File wasn't searched - inject warning
      log("Edit: File NOT searched, injecting warning:", normalizedPath)
      warnedFiles.add(normalizedPath)

      try {
        // Use client.session.prompt to inject a warning into the session
        // The noReply: true option means the agent won't be expected to respond
        if (client?.session?.prompt) {
          await client.session.prompt({
            path: { id: input.sessionID },
            body: {
              noReply: true,
              parts: [
                {
                  type: "text",
                  text: generateWarning(normalizedPath),
                },
              ],
            },
          })
          log("Warning injected successfully")
        }
      } catch (err) {
        log("Failed to inject warning:", err)
      }
    },

    /**
     * After hook: Track search/read operations and file creations.
     */
    "tool.execute.after": async (input: any, output: any) => {
      const tool = input.tool?.toLowerCase()

      // Retrieve stored args
      const args = input.callID ? pendingArgs.get(input.callID) : undefined

      // Clean up stored args
      if (input.callID) {
        pendingArgs.delete(input.callID)
      }

      // Track Write operations (new file creation)
      if (tool === "write") {
        const writePath = args?.filePath || args?.file_path || args?.path
        if (writePath) {
          const normalizedPath = normalizePath(writePath)
          createdFiles.add(normalizedPath)
          log("Write: Tracked new file:", normalizedPath)
        }
        return
      }

      // Track search/read operations
      const searchTools = ["read", "grep", "glob", "bash"]
      if (!searchTools.includes(tool)) {
        return
      }

      const targets = extractSearchTarget(tool, args)

      // Add searched files
      for (const file of targets.files) {
        searchedFiles.add(file)
        log(`${tool}: Tracked file:`, file)
      }

      // Add searched directories
      for (const dir of targets.directories) {
        searchedDirectories.add(dir)
        log(`${tool}: Tracked directory:`, dir)
      }

      // Clean up old entries to prevent memory leak
      // Keep only last 500 files and 100 directories
      if (searchedFiles.size > 500) {
        const iterator = searchedFiles.values()
        for (let i = 0; i < 250; i++) {
          const val = iterator.next().value
          if (val) searchedFiles.delete(val)
        }
      }
      if (searchedDirectories.size > 100) {
        const iterator = searchedDirectories.values()
        for (let i = 0; i < 50; i++) {
          const val = iterator.next().value
          if (val) searchedDirectories.delete(val)
        }
      }
    },
  }
}
