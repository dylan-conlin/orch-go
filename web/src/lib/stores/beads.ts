import { writable } from 'svelte/store'

// API configuration - HTTPS for HTTP/2 multiplexing
const API_BASE = 'http://localhost:3348'

// Beads stats response from /api/beads
export interface BeadsStats {
  total_issues: number
  open_issues: number
  in_progress_issues: number
  blocked_issues: number
  ready_issues: number
  closed_issues: number
  avg_lead_time_hours?: number
  project_dir?: string
  error?: string
}

// Ready issue from /api/beads/ready
export interface ReadyIssue {
  id: string
  title: string
  priority: number
  issue_type: string
  labels?: string[]
  created_at?: string
}

// Ready issues response from /api/beads/ready
export interface BeadsReadyResponse {
  issues: ReadyIssue[]
  count: number
  project_dir?: string
  error?: string
}

// Review queue issue from /api/beads/review-queue
// Uses verify.ListUnverifiedWork() as the canonical source — same as daemon counter
export interface ReviewQueueIssue {
  id: string
  title: string
  issue_type: string
  tier: number    // 1=feature/bug, 2=investigation, 3=task
  gate1: boolean  // Comprehension gate passed
  gate2: boolean  // Behavioral gate passed
}

// Review queue response from /api/beads/review-queue
export interface ReviewQueueResponse {
  issues: ReviewQueueIssue[]
  count: number
  project_dir?: string
  error?: string
}

// Beads store
function createBeadsStore() {
  const { subscribe, set } = writable<BeadsStats | null>(null)

  return {
    subscribe,
    set,
    // Fetch beads stats from orch-go API
    // projectDir: Optional project directory to query (for following orchestrator context)
    async fetch(projectDir?: string): Promise<void> {
      try {
        const params = new URLSearchParams()
        if (projectDir) {
          params.set('project_dir', projectDir)
        }
        const url = `${API_BASE}/api/beads${params.toString() ? '?' + params.toString() : ''}`
        const response = await fetch(url)
        if (!response.ok) {
          throw new Error(`HTTP ${response.status}: ${response.statusText}`)
        }
        const data = await response.json()
        set(data)
      } catch (error) {
        console.error('Failed to fetch beads stats:', error)
        set({
          total_issues: 0,
          open_issues: 0,
          in_progress_issues: 0,
          blocked_issues: 0,
          ready_issues: 0,
          closed_issues: 0,
          error: String(error),
        })
      }
    },
  }
}

// Ready issues store for dashboard queue visibility
function createReadyIssuesStore() {
  const { subscribe, set } = writable<BeadsReadyResponse | null>(null)

  return {
    subscribe,
    set,
    // Fetch ready issues from orch-go API
    // projectDir: Optional project directory to query (for following orchestrator context)
    async fetch(projectDir?: string): Promise<void> {
      try {
        const params = new URLSearchParams()
        if (projectDir) {
          params.set('project_dir', projectDir)
        }
        const url = `${API_BASE}/api/beads/ready${params.toString() ? '?' + params.toString() : ''}`
        const response = await fetch(url)
        if (!response.ok) {
          throw new Error(`HTTP ${response.status}: ${response.statusText}`)
        }
        const data = await response.json()
        set(data)
      } catch (error) {
        console.error('Failed to fetch ready issues:', error)
        set({
          issues: [],
          count: 0,
          error: String(error),
        })
      }
    },
  }
}

// Review queue store for issues awaiting review
function createReviewQueueStore() {
  const { subscribe, set } = writable<ReviewQueueResponse | null>(null)

  return {
    subscribe,
    set,
    // Fetch review queue issues from orch-go API
    // projectDir: Optional project directory to query (for following orchestrator context)
    async fetch(projectDir?: string): Promise<void> {
      try {
        const params = new URLSearchParams()
        if (projectDir) {
          params.set('project_dir', projectDir)
        }
        const url = `${API_BASE}/api/beads/review-queue${params.toString() ? '?' + params.toString() : ''}`
        const response = await fetch(url)
        if (!response.ok) {
          throw new Error(`HTTP ${response.status}: ${response.statusText}`)
        }
        const data = await response.json()
        set(data)
      } catch (error) {
        console.error('Failed to fetch review queue:', error)
        set({
          issues: [],
          count: 0,
          error: String(error),
        })
      }
    },
  }
}

export const beads = createBeadsStore()
export const readyIssues = createReadyIssuesStore()
export const reviewQueue = createReviewQueueStore()
