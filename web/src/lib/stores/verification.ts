import { writable } from 'svelte/store'

// API configuration - HTTPS for HTTP/2 multiplexing
const API_BASE = 'http://localhost:3348'

export type OverrideDirection = 'up' | 'down' | 'flat'

export interface VerificationOverrideTrend {
  window_days: number
  current_count: number
  previous_count: number
  delta: number
  direction: OverrideDirection
}

export interface VerificationStatus {
  unverified_count: number
  heartbeat_at?: string
  heartbeat_ago?: string
  daemon_paused?: boolean
  daemon_running?: boolean
  daemon_status?: string
  override_trend?: VerificationOverrideTrend
  project_dir?: string
  error?: string
}

function createVerificationStore() {
  const { subscribe, set } = writable<VerificationStatus | null>(null)

  return {
    subscribe,
    set,
    async fetch(projectDir?: string): Promise<void> {
      try {
        const params = new URLSearchParams()
        if (projectDir) {
          params.set('project_dir', projectDir)
        }
        const url = `${API_BASE}/api/verification${params.toString() ? '?' + params.toString() : ''}`
        const response = await fetch(url)
        if (!response.ok) {
          throw new Error(`HTTP ${response.status}: ${response.statusText}`)
        }
        const data = await response.json()
        set(data)
      } catch (error) {
        console.error('Failed to fetch verification status:', error)
        set({
          unverified_count: 0,
          error: String(error),
        })
      }
    },
  }
}

export const verification = createVerificationStore()
