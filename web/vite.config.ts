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
			// Proxy all API calls to orch serve (Go backend)
			// Go backend handles OpenCode proxying with proper error handling
			'/api': {
				target: 'http://localhost:3348',
				changeOrigin: true
			}
		}
	}
});
