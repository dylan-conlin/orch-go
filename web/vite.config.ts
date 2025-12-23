import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';

export default defineConfig({
	plugins: [sveltekit()],
	server: {
		port: 5188,
		proxy: {
			// Proxy SSE events from OpenCode
			'/api/events': {
				target: 'http://127.0.0.1:4096',
				changeOrigin: true,
				rewrite: (path) => path.replace(/^\/api/, '')
			},
			// Proxy API calls to OpenCode
			'/api': {
				target: 'http://127.0.0.1:4096',
				changeOrigin: true,
				rewrite: (path) => path.replace(/^\/api/, '')
			}
		}
	}
});
