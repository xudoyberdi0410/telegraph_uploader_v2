import { defineConfig } from 'vite'
import { svelte } from '@sveltejs/vite-plugin-svelte'
import { functionsMixins } from 'vite-plugin-functions-mixins'

// https://vitejs.dev/config/
export default defineConfig({
    plugins: [
        functionsMixins({ deps: ["m3-svelte"] }), // Этот плагин обязателен для m3-svelte
        svelte()
    ],
})
