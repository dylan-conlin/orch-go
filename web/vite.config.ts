import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';

export default defineConfig({
	plugins: [sveltekit()],
	server: {
		port: 5188,
		// Prevent browser caching in dev mode - ensures UI updates are visible after server restart
		// Without this, browsers cache JS bundles and require hard refresh (Cmd+Shift+R) to see changes
		headers: {
			'Cache-Control': 'no-store'
		},
		proxy: {
			// Proxy SSE events from OpenCode
			'/api/events': {
				target: 'http://localhost:4096',
				changeOrigin: true,
				rewrite: (path) => path.replace(/^\/api/, '')
			},
			// Proxy API calls to OpenCode
			'/api': {
				target: 'http://localhost:4096',
				changeOrigin: true,
				rewrite: (path) => path.replace(/^\/api/, '')
			}
		}
	}
});
