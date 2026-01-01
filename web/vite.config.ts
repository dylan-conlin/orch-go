import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';

export default defineConfig({
	plugins: [sveltekit()],
	server: {
		port: 5188,
		proxy: {
			// Proxy API calls to orch serve
			'/api': {
				target: 'http://localhost:3348',
				changeOrigin: true
			}
		}
	}
});
