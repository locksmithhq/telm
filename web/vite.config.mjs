import Components from 'unplugin-vue-components/vite'
import Vue from '@vitejs/plugin-vue'
import Vuetify, { transformAssetUrls } from 'vite-plugin-vuetify'
import Fonts from 'unplugin-fonts/vite'

import { defineConfig } from 'vite'
import { fileURLToPath, URL } from 'node:url'

export default defineConfig({
  plugins: [
    Vue({ template: { transformAssetUrls } }),
    Vuetify(),
    Components(),
    Fonts({
      fontsource: {
        families: [
          { name: 'Roboto', weights: [300, 400, 500, 700], styles: ['normal'] },
        ],
      },
    }),
  ],
  optimizeDeps: { exclude: ['vuetify'] },
  define: { 'process.env': {} },
  resolve: {
    alias: { '@': fileURLToPath(new URL('src', import.meta.url)) },
    extensions: ['.js', '.json', '.jsx', '.mjs', '.ts', '.tsx', '.vue'],
  },
  server: {
    port: 3000,
    hmr: { clientPort: 4000 },
    host: '0.0.0.0',
  },
})
