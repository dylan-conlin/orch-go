/**
 * Phase 2: Investigation circular pattern detection.
 * Parse investigation D.E.K.N. summaries and detect contradictions.
 */
import { existsSync, readFileSync, readdirSync, statSync } from "fs"
import { join } from "path"
import type { InvestigationRecommendation } from "./coaching-types"
import { log, DEBUG, LOG_PREFIX } from "./coaching-types"

/**
 * Parse D.E.K.N. Summary from investigation markdown.
 * Extracts the **Next:** field which contains architectural recommendations.
 */
export function parseDEKNSummary(content: string): string | null {
  const deknMatch = content.match(/## Summary \(D\.E\.K\.N\.\)([\s\S]*?)(?=\n##|$)/i)
  if (!deknMatch) {
    return null
  }

  const deknSection = deknMatch[1]
  const nextMatch = deknSection.match(/\*\*Next:\*\*\s+(.+?)(?=\n\*\*|$)/s)
  if (!nextMatch) {
    return null
  }

  return nextMatch[1].trim()
}

/**
 * Extract architectural keywords from recommendation text.
 */
export function extractKeywords(text: string): string[] {
  const lowerText = text.toLowerCase()
  const keywords: string[] = []

  const patterns = [
    "launchd",
    "overmind",
    "tmux",
    "systemd",
    "docker",
    "kubernetes",
    "procfile",
    "plist",
    "daemon",
    "supervisor",
  ]

  for (const pattern of patterns) {
    if (lowerText.includes(pattern)) {
      keywords.push(pattern)
    }
  }

  return keywords
}

/**
 * Load all investigation recommendations from .kb/investigations/.
 */
export function loadInvestigationRecommendations(directory: string): InvestigationRecommendation[] {
  const recommendations: InvestigationRecommendation[] = []
  const invDir = join(directory, ".kb", "investigations")

  if (!existsSync(invDir)) {
    log("Investigation directory not found:", invDir)
    return recommendations
  }

  try {
    const files = readdirSync(invDir)

    for (const file of files) {
      if (!file.endsWith(".md")) continue

      const filePath = join(invDir, file)
      const stat = statSync(filePath)
      if (!stat.isFile()) continue

      const dateMatch = file.match(/^(\d{4}-\d{2}-\d{2})/)
      if (!dateMatch) continue

      const content = readFileSync(filePath, "utf-8")
      const next = parseDEKNSummary(content)

      if (next) {
        const keywords = extractKeywords(next)
        recommendations.push({
          filePath,
          fileName: file,
          date: dateMatch[1],
          next,
          keywords,
        })

        log(`Loaded recommendation from ${file}: ${keywords.join(", ")}`)
      }
    }

    log(`Loaded ${recommendations.length} investigation recommendations`)
  } catch (err) {
    if (DEBUG) console.error(LOG_PREFIX, "Failed to load investigation recommendations:", err)
  }

  return recommendations
}

/**
 * Detect if command represents an architectural decision.
 * Returns extracted keywords if it's a decision, null otherwise.
 */
export function detectArchitecturalDecision(command: string): string[] | null {
  if (!command) return null

  if (command.includes("git commit")) {
    const messageMatch = command.match(/-m\s+["']([^"']+)["']/i)
    if (messageMatch) {
      return extractKeywords(messageMatch[1])
    }
  }

  if (
    command.includes(".plist") ||
    command.includes("Procfile") ||
    command.includes("launchd") ||
    command.includes("overmind")
  ) {
    return extractKeywords(command)
  }

  if (command.includes("bd create") && (command.includes("--type") || command.includes("-t"))) {
    return extractKeywords(command)
  }

  return null
}

/**
 * Check if decision keywords contradict recommendation keywords.
 */
export function findContradiction(
  decisionKeywords: string[],
  recommendations: InvestigationRecommendation[]
): InvestigationRecommendation | null {
  if (!decisionKeywords.length) return null

  const architectureGroups = {
    process_supervision: ["launchd", "overmind", "systemd", "supervisor", "daemon"],
    containerization: ["docker", "kubernetes"],
  }

  for (const group of Object.values(architectureGroups)) {
    const decisionInGroup = decisionKeywords.filter((k) => group.includes(k))
    if (!decisionInGroup.length) continue

    for (const rec of recommendations) {
      const recInGroup = rec.keywords.filter((k) => group.includes(k))
      if (!recInGroup.length) continue

      const hasContradiction = recInGroup.some((rk) => !decisionKeywords.includes(rk))

      if (hasContradiction) {
        log(
          `Potential circular pattern: Decision uses ${decisionInGroup.join(", ")} but ${rec.fileName} recommended ${recInGroup.join(", ")}`
        )
        return rec
      }
    }
  }

  return null
}
