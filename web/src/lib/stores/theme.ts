import { writable, derived, get } from 'svelte/store';
import { browser } from '$app/environment';

// Import all theme JSON files
import aura from '$lib/themes/aura.json';
import ayu from '$lib/themes/ayu.json';
import catppuccin from '$lib/themes/catppuccin.json';
import catppuccinMacchiato from '$lib/themes/catppuccin-macchiato.json';
import cobalt2 from '$lib/themes/cobalt2.json';
import dracula from '$lib/themes/dracula.json';
import everforest from '$lib/themes/everforest.json';
import flexoki from '$lib/themes/flexoki.json';
import github from '$lib/themes/github.json';
import gruvbox from '$lib/themes/gruvbox.json';
import kanagawa from '$lib/themes/kanagawa.json';
import material from '$lib/themes/material.json';
import matrix from '$lib/themes/matrix.json';
import mercury from '$lib/themes/mercury.json';
import monokai from '$lib/themes/monokai.json';
import nightowl from '$lib/themes/nightowl.json';
import nord from '$lib/themes/nord.json';
import onedark from '$lib/themes/one-dark.json';
import opencode from '$lib/themes/opencode.json';
import orng from '$lib/themes/orng.json';
import palenight from '$lib/themes/palenight.json';
import rosepine from '$lib/themes/rosepine.json';
import solarized from '$lib/themes/solarized.json';
import synthwave84 from '$lib/themes/synthwave84.json';
import tokyonight from '$lib/themes/tokyonight.json';
import vercel from '$lib/themes/vercel.json';
import vesper from '$lib/themes/vesper.json';
import zenburn from '$lib/themes/zenburn.json';

// Type definitions matching OpenCode's theme schema
export type HexColor = `#${string}`;

export type Variant = {
	dark: HexColor | string;
	light: HexColor | string;
};

export type ColorValue = HexColor | string | Variant;

export interface ThemeJson {
	$schema?: string;
	defs?: Record<string, HexColor | string>;
	theme: {
		primary: ColorValue;
		secondary: ColorValue;
		accent: ColorValue;
		error: ColorValue;
		warning: ColorValue;
		success: ColorValue;
		info: ColorValue;
		text: ColorValue;
		textMuted: ColorValue;
		selectedListItemText?: ColorValue;
		background: ColorValue;
		backgroundPanel: ColorValue;
		backgroundElement: ColorValue;
		backgroundMenu?: ColorValue;
		border: ColorValue;
		borderActive: ColorValue;
		borderSubtle: ColorValue;
		// Diff colors
		diffAdded: ColorValue;
		diffRemoved: ColorValue;
		diffContext: ColorValue;
		diffHunkHeader: ColorValue;
		diffHighlightAdded: ColorValue;
		diffHighlightRemoved: ColorValue;
		diffAddedBg: ColorValue;
		diffRemovedBg: ColorValue;
		diffContextBg: ColorValue;
		diffLineNumber: ColorValue;
		diffAddedLineNumberBg: ColorValue;
		diffRemovedLineNumberBg: ColorValue;
		// Markdown colors
		markdownText: ColorValue;
		markdownHeading: ColorValue;
		markdownLink: ColorValue;
		markdownLinkText: ColorValue;
		markdownCode: ColorValue;
		markdownBlockQuote: ColorValue;
		markdownEmph: ColorValue;
		markdownStrong: ColorValue;
		markdownHorizontalRule: ColorValue;
		markdownListItem: ColorValue;
		markdownListEnumeration: ColorValue;
		markdownImage: ColorValue;
		markdownImageText: ColorValue;
		markdownCodeBlock: ColorValue;
		// Syntax colors
		syntaxComment: ColorValue;
		syntaxKeyword: ColorValue;
		syntaxFunction: ColorValue;
		syntaxVariable: ColorValue;
		syntaxString: ColorValue;
		syntaxNumber: ColorValue;
		syntaxType: ColorValue;
		syntaxOperator: ColorValue;
		syntaxPunctuation: ColorValue;
		thinkingOpacity?: number;
	};
}

export interface ResolvedTheme {
	primary: string;
	secondary: string;
	accent: string;
	error: string;
	warning: string;
	success: string;
	info: string;
	text: string;
	textMuted: string;
	selectedListItemText: string;
	background: string;
	backgroundPanel: string;
	backgroundElement: string;
	backgroundMenu: string;
	border: string;
	borderActive: string;
	borderSubtle: string;
	// Diff colors
	diffAdded: string;
	diffRemoved: string;
	diffContext: string;
	diffHunkHeader: string;
	diffHighlightAdded: string;
	diffHighlightRemoved: string;
	diffAddedBg: string;
	diffRemovedBg: string;
	diffContextBg: string;
	diffLineNumber: string;
	diffAddedLineNumberBg: string;
	diffRemovedLineNumberBg: string;
	// Markdown colors
	markdownText: string;
	markdownHeading: string;
	markdownLink: string;
	markdownLinkText: string;
	markdownCode: string;
	markdownBlockQuote: string;
	markdownEmph: string;
	markdownStrong: string;
	markdownHorizontalRule: string;
	markdownListItem: string;
	markdownListEnumeration: string;
	markdownImage: string;
	markdownImageText: string;
	markdownCodeBlock: string;
	// Syntax colors
	syntaxComment: string;
	syntaxKeyword: string;
	syntaxFunction: string;
	syntaxVariable: string;
	syntaxString: string;
	syntaxNumber: string;
	syntaxType: string;
	syntaxOperator: string;
	syntaxPunctuation: string;
}

// All available themes
export const DEFAULT_THEMES: Record<string, ThemeJson> = {
	aura: aura as ThemeJson,
	ayu: ayu as ThemeJson,
	catppuccin: catppuccin as ThemeJson,
	'catppuccin-macchiato': catppuccinMacchiato as ThemeJson,
	cobalt2: cobalt2 as ThemeJson,
	dracula: dracula as ThemeJson,
	everforest: everforest as ThemeJson,
	flexoki: flexoki as ThemeJson,
	github: github as ThemeJson,
	gruvbox: gruvbox as ThemeJson,
	kanagawa: kanagawa as ThemeJson,
	material: material as ThemeJson,
	matrix: matrix as ThemeJson,
	mercury: mercury as ThemeJson,
	monokai: monokai as ThemeJson,
	nightowl: nightowl as ThemeJson,
	nord: nord as ThemeJson,
	'one-dark': onedark as ThemeJson,
	opencode: opencode as ThemeJson,
	orng: orng as ThemeJson,
	palenight: palenight as ThemeJson,
	rosepine: rosepine as ThemeJson,
	solarized: solarized as ThemeJson,
	synthwave84: synthwave84 as ThemeJson,
	tokyonight: tokyonight as ThemeJson,
	vercel: vercel as ThemeJson,
	vesper: vesper as ThemeJson,
	zenburn: zenburn as ThemeJson
};

// Get sorted theme names for the UI
export const themeNames = Object.keys(DEFAULT_THEMES).sort((a, b) =>
	a.localeCompare(b, undefined, { sensitivity: 'base' })
);

export type Mode = 'light' | 'dark' | 'system';

/**
 * Resolve a color value from the theme JSON, handling:
 * - Direct hex colors (#xxx or #xxxxxx)
 * - References to defs
 * - References to other theme properties
 * - Dark/light variants
 */
function resolveColor(
	colorValue: ColorValue,
	mode: 'dark' | 'light',
	defs: Record<string, string> = {},
	themeObj: Record<string, ColorValue> = {}
): string {
	// Handle transparent/none
	if (colorValue === 'transparent' || colorValue === 'none') {
		return 'transparent';
	}

	// Handle string values (hex colors or references)
	if (typeof colorValue === 'string') {
		// Direct hex color
		if (colorValue.startsWith('#')) {
			return colorValue;
		}

		// Reference to defs
		if (defs[colorValue] !== undefined) {
			return resolveColor(defs[colorValue], mode, defs, themeObj);
		}

		// Reference to another theme property
		if (themeObj[colorValue] !== undefined) {
			return resolveColor(themeObj[colorValue], mode, defs, themeObj);
		}

		// Unknown reference, return as-is (shouldn't happen with valid themes)
		console.warn(`Unknown color reference: ${colorValue}`);
		return colorValue;
	}

	// Handle variant objects { dark: ..., light: ... }
	if (typeof colorValue === 'object' && colorValue !== null) {
		const variantColor = colorValue[mode];
		return resolveColor(variantColor, mode, defs, themeObj);
	}

	// Fallback
	return '#000000';
}

/**
 * Resolve a full theme JSON into concrete hex colors for a given mode
 */
function resolveTheme(themeJson: ThemeJson, mode: 'dark' | 'light'): ResolvedTheme {
	const defs = themeJson.defs ?? {};
	const themeObj = themeJson.theme as Record<string, ColorValue>;

	const resolve = (key: keyof typeof themeJson.theme): string => {
		const value = themeJson.theme[key];
		if (value === undefined) {
			// Handle optional properties with defaults
			if (key === 'selectedListItemText') {
				return resolve('background');
			}
			if (key === 'backgroundMenu') {
				return resolve('backgroundElement');
			}
			return '#000000';
		}
		return resolveColor(value, mode, defs, themeObj);
	};

	return {
		primary: resolve('primary'),
		secondary: resolve('secondary'),
		accent: resolve('accent'),
		error: resolve('error'),
		warning: resolve('warning'),
		success: resolve('success'),
		info: resolve('info'),
		text: resolve('text'),
		textMuted: resolve('textMuted'),
		selectedListItemText: resolve('selectedListItemText'),
		background: resolve('background'),
		backgroundPanel: resolve('backgroundPanel'),
		backgroundElement: resolve('backgroundElement'),
		backgroundMenu: resolve('backgroundMenu'),
		border: resolve('border'),
		borderActive: resolve('borderActive'),
		borderSubtle: resolve('borderSubtle'),
		diffAdded: resolve('diffAdded'),
		diffRemoved: resolve('diffRemoved'),
		diffContext: resolve('diffContext'),
		diffHunkHeader: resolve('diffHunkHeader'),
		diffHighlightAdded: resolve('diffHighlightAdded'),
		diffHighlightRemoved: resolve('diffHighlightRemoved'),
		diffAddedBg: resolve('diffAddedBg'),
		diffRemovedBg: resolve('diffRemovedBg'),
		diffContextBg: resolve('diffContextBg'),
		diffLineNumber: resolve('diffLineNumber'),
		diffAddedLineNumberBg: resolve('diffAddedLineNumberBg'),
		diffRemovedLineNumberBg: resolve('diffRemovedLineNumberBg'),
		markdownText: resolve('markdownText'),
		markdownHeading: resolve('markdownHeading'),
		markdownLink: resolve('markdownLink'),
		markdownLinkText: resolve('markdownLinkText'),
		markdownCode: resolve('markdownCode'),
		markdownBlockQuote: resolve('markdownBlockQuote'),
		markdownEmph: resolve('markdownEmph'),
		markdownStrong: resolve('markdownStrong'),
		markdownHorizontalRule: resolve('markdownHorizontalRule'),
		markdownListItem: resolve('markdownListItem'),
		markdownListEnumeration: resolve('markdownListEnumeration'),
		markdownImage: resolve('markdownImage'),
		markdownImageText: resolve('markdownImageText'),
		markdownCodeBlock: resolve('markdownCodeBlock'),
		syntaxComment: resolve('syntaxComment'),
		syntaxKeyword: resolve('syntaxKeyword'),
		syntaxFunction: resolve('syntaxFunction'),
		syntaxVariable: resolve('syntaxVariable'),
		syntaxString: resolve('syntaxString'),
		syntaxNumber: resolve('syntaxNumber'),
		syntaxType: resolve('syntaxType'),
		syntaxOperator: resolve('syntaxOperator'),
		syntaxPunctuation: resolve('syntaxPunctuation')
	};
}

/**
 * Convert hex color to HSL values for CSS variables
 */
function hexToHsl(hex: string): string {
	// Remove # if present
	hex = hex.replace('#', '');

	// Handle shorthand hex (e.g., #fff)
	if (hex.length === 3) {
		hex = hex
			.split('')
			.map((c) => c + c)
			.join('');
	}

	const r = parseInt(hex.substring(0, 2), 16) / 255;
	const g = parseInt(hex.substring(2, 4), 16) / 255;
	const b = parseInt(hex.substring(4, 6), 16) / 255;

	const max = Math.max(r, g, b);
	const min = Math.min(r, g, b);
	let h = 0;
	let s = 0;
	const l = (max + min) / 2;

	if (max !== min) {
		const d = max - min;
		s = l > 0.5 ? d / (2 - max - min) : d / (max + min);
		switch (max) {
			case r:
				h = ((g - b) / d + (g < b ? 6 : 0)) / 6;
				break;
			case g:
				h = ((b - r) / d + 2) / 6;
				break;
			case b:
				h = ((r - g) / d + 4) / 6;
				break;
		}
	}

	// Return as "h s% l%" format for CSS hsl()
	return `${Math.round(h * 360)} ${Math.round(s * 100)}% ${Math.round(l * 100)}%`;
}

/**
 * Apply theme CSS variables to the document
 */
function applyCssVariables(theme: ResolvedTheme) {
	if (!browser) return;

	const root = document.documentElement;

	// Map theme colors to CSS variables used by shadcn/ui
	// Using HSL format as expected by Tailwind/shadcn
	root.style.setProperty('--background', hexToHsl(theme.background));
	root.style.setProperty('--foreground', hexToHsl(theme.text));
	root.style.setProperty('--card', hexToHsl(theme.backgroundPanel));
	root.style.setProperty('--card-foreground', hexToHsl(theme.text));
	root.style.setProperty('--popover', hexToHsl(theme.backgroundElement));
	root.style.setProperty('--popover-foreground', hexToHsl(theme.text));
	root.style.setProperty('--primary', hexToHsl(theme.primary));
	root.style.setProperty('--primary-foreground', hexToHsl(theme.background));
	root.style.setProperty('--secondary', hexToHsl(theme.secondary));
	root.style.setProperty('--secondary-foreground', hexToHsl(theme.text));
	root.style.setProperty('--muted', hexToHsl(theme.backgroundElement));
	root.style.setProperty('--muted-foreground', hexToHsl(theme.textMuted));
	root.style.setProperty('--accent', hexToHsl(theme.accent));
	root.style.setProperty('--accent-foreground', hexToHsl(theme.text));
	root.style.setProperty('--destructive', hexToHsl(theme.error));
	root.style.setProperty('--destructive-foreground', hexToHsl(theme.background));
	root.style.setProperty('--border', hexToHsl(theme.border));
	root.style.setProperty('--input', hexToHsl(theme.border));
	root.style.setProperty('--ring', hexToHsl(theme.primary));

	// Additional semantic colors for custom components
	root.style.setProperty('--success', hexToHsl(theme.success));
	root.style.setProperty('--warning', hexToHsl(theme.warning));
	root.style.setProperty('--info', hexToHsl(theme.info));
}

// Initialize stores
function getStoredValue<T>(key: string, defaultValue: T): T {
	if (!browser) return defaultValue;
	const stored = localStorage.getItem(key);
	if (stored === null) return defaultValue;
	try {
		return JSON.parse(stored) as T;
	} catch {
		return stored as unknown as T;
	}
}

function getSystemMode(): 'dark' | 'light' {
	if (!browser) return 'dark';
	return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
}

// Theme name store
const activeTheme = writable<string>(getStoredValue('theme', 'opencode'));

// Mode store (light/dark/system)
const themeMode = writable<Mode>(getStoredValue('theme_mode', 'system'));

// Effective mode (resolved system to actual light/dark)
const effectiveMode = derived(themeMode, ($mode) => {
	if ($mode === 'system') {
		return getSystemMode();
	}
	return $mode;
});

// Resolved theme colors
const resolvedTheme = derived([activeTheme, effectiveMode], ([$theme, $mode]) => {
	const themeJson = DEFAULT_THEMES[$theme] ?? DEFAULT_THEMES['opencode'];
	return resolveTheme(themeJson, $mode);
});

// Apply CSS when resolved theme changes
if (browser) {
	resolvedTheme.subscribe((theme) => {
		applyCssVariables(theme);
	});

	// Listen for system theme changes
	window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', () => {
		const currentMode = get(themeMode);
		if (currentMode === 'system') {
			// Force re-resolution
			themeMode.set('system');
		}
	});
}

// Exported store API
export const theme = {
	// Current theme name
	subscribe: activeTheme.subscribe,

	// Set theme by name
	set: (name: string) => {
		if (DEFAULT_THEMES[name]) {
			activeTheme.set(name);
			if (browser) {
				localStorage.setItem('theme', JSON.stringify(name));
			}
		}
	},

	// Get all theme names
	all: () => themeNames,

	// Initialize (apply stored theme on mount)
	init: () => {
		if (browser) {
			const theme = get(resolvedTheme);
			applyCssVariables(theme);
		}
	}
};

export const mode = {
	subscribe: themeMode.subscribe,

	set: (m: Mode) => {
		themeMode.set(m);
		if (browser) {
			localStorage.setItem('theme_mode', JSON.stringify(m));
			// Update dark class on root
			const effective = m === 'system' ? getSystemMode() : m;
			if (effective === 'dark') {
				document.documentElement.classList.add('dark');
			} else {
				document.documentElement.classList.remove('dark');
			}
		}
	},

	// Get effective mode (light or dark, with system resolved)
	effective: effectiveMode
};

// For backwards compatibility
export type Theme = Mode;
export function getEffective(t: Mode): 'light' | 'dark' {
	if (t === 'system') {
		return getSystemMode();
	}
	return t;
}
