import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import eslint from '@nabla/vite-plugin-eslint'
import vueI18n from '@intlify/unplugin-vue-i18n/vite'
import compression from 'vite-plugin-compression2'

// https://vite.dev/config/
export default defineConfig({
  build: {
    emptyOutDir: true,
    outDir: '../internal/server/assets/dist/'
  },
  plugins: [
    vue(),
    eslint(),
    compression({
      algorithms: ['gzip'],
      deleteOriginalAssets: true,
      threshold: 10240
    }),
    vueI18n()
  ],
  server: {
    proxy: {
      '^/api': {
        ws: true,
        target: 'http://127.0.0.1:5913'
      }
    }
  }
})
