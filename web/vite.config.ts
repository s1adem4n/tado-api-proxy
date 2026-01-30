import { svelte } from '@sveltejs/vite-plugin-svelte';
import tailwindcss from '@tailwindcss/vite';
import icons from 'unplugin-icons/vite';
import { defineConfig } from 'vite';

export default defineConfig({
	plugins: [svelte(), tailwindcss(), icons({ compiler: 'svelte' })],
	resolve: {
		alias: {
			'@': '/src'
		}
	},
	server: {
		proxy: {
			'/api': 'http://localhost:8090'
		}
	}
});
