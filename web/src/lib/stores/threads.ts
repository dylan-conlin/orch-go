import { writable } from 'svelte/store'

const API_BASE = 'http://localhost:3348'

export interface ThreadSummary {
	name: string
	title: string
	status: string
	created: string
	updated: string
	resolved_to?: string
	latest_entry: string
	entry_count: number
	filename: string
}

export interface ThreadEntry {
	date: string
	text: string
}

export interface ThreadDetail {
	slug: string
	title: string
	status: string
	created: string
	updated: string
	resolved_to?: string
	spawned_from?: string
	spawned?: string[]
	active_work?: string[]
	resolved_by?: string[]
	entries: ThreadEntry[]
	entry_count: number
	content: string
	filename: string
}

function createThreadsStore() {
	const { subscribe, set } = writable<ThreadSummary[]>([])

	return {
		subscribe,
		set,
		async fetch(status?: string): Promise<void> {
			try {
				const params = status ? `?status=${encodeURIComponent(status)}` : ''
				const response = await fetch(`${API_BASE}/api/threads${params}`)
				if (!response.ok) {
					throw new Error(`HTTP ${response.status}: ${response.statusText}`)
				}
				const data: ThreadSummary[] = await response.json()
				set(data)
			} catch (error) {
				console.error('Failed to fetch threads:', error)
				set([])
			}
		}
	}
}

function createThreadDetailStore() {
	const { subscribe, set } = writable<ThreadDetail | null>(null)

	return {
		subscribe,
		set,
		async fetch(slug: string): Promise<void> {
			try {
				const response = await fetch(`${API_BASE}/api/threads/${encodeURIComponent(slug)}`)
				if (!response.ok) {
					throw new Error(`HTTP ${response.status}: ${response.statusText}`)
				}
				const data: ThreadDetail = await response.json()
				set(data)
			} catch (error) {
				console.error('Failed to fetch thread detail:', error)
				set(null)
			}
		},
		clear() {
			set(null)
		}
	}
}

export const threads = createThreadsStore()
export const threadDetail = createThreadDetailStore()
