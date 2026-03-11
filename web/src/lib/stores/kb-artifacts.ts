import { writable } from 'svelte/store'

// API configuration - HTTPS for HTTP/2 multiplexing
const API_BASE = 'http://localhost:3348'

// Artifact from /api/kb/artifacts
export interface ArtifactFeedItem {
  path: string // Relative path from project root
  title: string // From frontmatter or filename
  type: string // investigation, decision, model, guide, principle
  status: string // Status field from frontmatter
  date: string // Date from frontmatter or filename
  summary: string // First paragraph or summary from frontmatter
  recommendation: boolean // True if investigation has recommendation section
  modified_at: string // File modification time (ISO 8601)
  relative_time: string // Human-readable relative time (e.g., "2h ago")
}

// Artifacts response from /api/kb/artifacts
export interface KBArtifactsResponse {
  needs_decision: ArtifactFeedItem[]
  recent: ArtifactFeedItem[]
  by_type: Record<string, ArtifactFeedItem[]>
  project_dir?: string
  error?: string
}

// KB artifacts store
function createKBArtifactsStore() {
  const { subscribe, set, update } = writable<KBArtifactsResponse | null>(null)
  let currentSince = '7d'
  let controller: AbortController | null = null
  let request = 0

  return {
    subscribe,
    set,
    update,
    getSince(): string {
      return currentSince
    },
    // Fetch KB artifacts from orch-go API.
    // projectDir: Optional project directory to query.
    // since: Optional time filter; when omitted, reuses the active filter.
    async fetch(projectDir?: string, since?: string): Promise<void> {
      request += 1
      const id = request

      if (controller) {
        controller.abort()
      }

      controller = new AbortController()

      try {
        const effectiveSince = since ?? currentSince
        currentSince = effectiveSince

        const params = new URLSearchParams()
        if (projectDir) {
          params.set('project_dir', projectDir)
        }
        params.set('since', effectiveSince)
        const url = `${API_BASE}/api/kb/artifacts${params.toString() ? '?' + params.toString() : ''}`
        const response = await fetch(url, { signal: controller.signal })
        if (!response.ok) {
          throw new Error(`HTTP ${response.status}: ${response.statusText}`)
        }
        const data = await response.json()
        if (id !== request) {
          return
        }
        set(data)
      } catch (error) {
        if (id !== request) {
          return
        }
        if (error instanceof Error && error.name === 'AbortError') {
          return
        }
        console.error('Failed to fetch KB artifacts:', error)
        set({
          needs_decision: [],
          recent: [],
          by_type: {},
          error: String(error),
        })
      } finally {
        if (id === request) {
          controller = null
        }
      }
    },
  }
}

export const kbArtifacts = createKBArtifactsStore()

// Artifact content response from /api/kb/artifact/content
export interface ArtifactContentResponse {
  path: string
  content: string
  error?: string
}

// Fetch full content of a specific artifact
export async function fetchArtifactContent(
  path: string,
  projectDir?: string,
): Promise<ArtifactContentResponse> {
  try {
    const params = new URLSearchParams()
    params.set('path', path)
    if (projectDir) {
      params.set('project_dir', projectDir)
    }
    const url = `${API_BASE}/api/kb/artifact/content?${params.toString()}`
    const response = await fetch(url)
    if (!response.ok) {
      throw new Error(`HTTP ${response.status}: ${response.statusText}`)
    }
    return await response.json()
  } catch (error) {
    console.error('Failed to fetch artifact content:', error)
    return {
      path,
      content: '',
      error: String(error),
    }
  }
}
