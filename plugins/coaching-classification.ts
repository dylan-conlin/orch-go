/**
 * Semantic command classification for bash commands.
 */
import type { SemanticGroup } from "./coaching-types"

/**
 * Patterns for semantic command classification.
 * Order matters - first match wins.
 */
const SEMANTIC_PATTERNS: Array<{ group: SemanticGroup; patterns: RegExp[] }> = [
  {
    group: "process_mgmt",
    patterns: [
      /\bovermind\b/,
      /\btmux\b/,
      /\blaunchd\b/,
      /\blaunchctl\b/,
      /\bsystemctl\b/,
      /\bservice\b/,
      /\bkill\b/,
      /\bpkill\b/,
      /\bps\b.*aux/,
      /\bpgrep\b/,
      /\blsof\b/,
    ],
  },
  {
    group: "git",
    patterns: [/\bgit\b/],
  },
  // Test patterns must come BEFORE build patterns (npm test vs npm install)
  {
    group: "test",
    patterns: [/\bgo test\b/, /\bnpm test\b/, /\bjest\b/, /\bpytest\b/, /\bvitest\b/],
  },
  {
    group: "build",
    patterns: [/\bmake\b/, /\bgo build\b/, /\bgo install\b/, /\bnpm\b/, /\bbun\b/, /\bcargo\b/],
  },
  {
    group: "knowledge",
    patterns: [/\bkb\b/, /\bbd\b/],
  },
  // Orch pattern: must be command start, not part of a path like ~/.orch/
  {
    group: "orch",
    patterns: [/^orch\b/, /\borch spawn\b/, /\borch status\b/, /\borch complete\b/],
  },
  {
    group: "file_ops",
    patterns: [/\bls\b/, /\bcat\b/, /\bmkdir\b/, /\bcp\b/, /\bmv\b/, /\brm\b/, /\bfind\b/, /\brg\b/],
  },
  {
    group: "network",
    patterns: [/\bcurl\b/, /\bwget\b/, /\bnc\b/, /\bssh\b/, /\bhttp\b/],
  },
]

/**
 * Classify a bash command into a semantic group.
 */
export function classifyBashCommand(command: string): SemanticGroup {
  if (!command) return "other"

  for (const { group, patterns } of SEMANTIC_PATTERNS) {
    for (const pattern of patterns) {
      if (pattern.test(command)) {
        return group
      }
    }
  }

  return "other"
}

/**
 * Detect if a bash command is a spawn.
 */
export function isSpawn(command: string): boolean {
  if (!command) return false
  return command.includes("orch spawn")
}
