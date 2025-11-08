// @ts-check
import { defineConfig } from 'astro/config';
import solidJs from '@astrojs/solid-js';
import tailwindcss from '@tailwindcss/vite';

// https://astro.build/config
export default defineConfig({
  integrations: [
    solidJs()
  ],
  output: 'static',
  server: {
    port: 3000,
    host: true
  },
  vite: {
    plugins: [tailwindcss()]
  }
});