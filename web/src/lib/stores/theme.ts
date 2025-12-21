import { writable } from 'svelte/store';
import { browser } from '$app/environment';

export type Theme = 'light' | 'dark' | 'system';

function getInitialTheme(): Theme {
	if (!browser) return 'system';
	
	const stored = localStorage.getItem('theme');
	if (stored === 'light' || stored === 'dark' || stored === 'system') {
		return stored;
	}
	return 'system';
}

function getEffectiveTheme(theme: Theme): 'light' | 'dark' {
	if (theme === 'system') {
		if (!browser) return 'dark';
		return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
	}
	return theme;
}

function createThemeStore() {
	const { subscribe, set, update } = writable<Theme>(getInitialTheme());

	return {
		subscribe,
		set: (theme: Theme) => {
			if (browser) {
				localStorage.setItem('theme', theme);
				applyTheme(theme);
			}
			set(theme);
		},
		toggle: () => {
			update((current) => {
				const effective = getEffectiveTheme(current);
				const next: Theme = effective === 'dark' ? 'light' : 'dark';
				if (browser) {
					localStorage.setItem('theme', next);
					applyTheme(next);
				}
				return next;
			});
		},
		init: () => {
			if (browser) {
				const theme = getInitialTheme();
				applyTheme(theme);
				
				// Listen for system theme changes
				window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', () => {
					update((current) => {
						if (current === 'system') {
							applyTheme('system');
						}
						return current;
					});
				});
			}
		}
	};
}

function applyTheme(theme: Theme) {
	if (!browser) return;
	
	const effective = getEffectiveTheme(theme);
	const root = document.documentElement;
	
	if (effective === 'dark') {
		root.classList.add('dark');
	} else {
		root.classList.remove('dark');
	}
}

export const theme = createThemeStore();

// Derived store for the effective (resolved) theme
export function getEffective(t: Theme): 'light' | 'dark' {
	return getEffectiveTheme(t);
}
