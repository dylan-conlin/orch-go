import { writable } from 'svelte/store'
import type { BriefResponse } from './beads'

const API_BASE = 'http://localhost:3348'

export interface BriefListItem {
	beads_id: string
	marked_read: boolean
	thread_slug?: string
	thread_title?: string
	has_tension?: boolean
}

function createBriefsStore() {
	const { subscribe, set, update } = writable<BriefListItem[]>([])

	return {
		subscribe,
		set,
		async fetch(projectDir?: string): Promise<void> {
			try {
				const params = projectDir ? `?project_dir=${encodeURIComponent(projectDir)}` : ''
				const response = await fetch(`${API_BASE}/api/briefs${params}`)
				if (!response.ok) {
					throw new Error(`HTTP ${response.status}: ${response.statusText}`)
				}
				const data: BriefListItem[] = await response.json()
				set(data)
			} catch (error) {
				console.error('Failed to fetch briefs list:', error)
				set([])
			}
		},
		async fetchBrief(beadsId: string, projectDir?: string): Promise<BriefResponse | null> {
			try {
				const params = projectDir ? `?project_dir=${encodeURIComponent(projectDir)}` : ''
				const response = await fetch(`${API_BASE}/api/briefs/${beadsId}${params}`)
				if (!response.ok) return null
				return await response.json()
			} catch (error) {
				console.error('Failed to fetch brief:', error)
				return null
			}
		},
		async markAsRead(beadsId: string, projectDir?: string): Promise<boolean> {
			try {
				const params = projectDir ? `?project_dir=${encodeURIComponent(projectDir)}` : ''
				const response = await fetch(`${API_BASE}/api/briefs/${beadsId}${params}`, {
					method: 'POST',
				})
				if (!response.ok) return false
				update(items =>
					items.map(item =>
						item.beads_id === beadsId
							? { ...item, marked_read: true }
							: item
					)
				)
				return true
			} catch (error) {
				console.error('Failed to mark brief as read:', error)
				return false
			}
		},
	}
}

export const briefs = createBriefsStore()
