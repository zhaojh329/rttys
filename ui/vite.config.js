import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import eslint from '@nabla/vite-plugin-eslint'
import vueI18n from '@intlify/unplugin-vue-i18n/vite'
import compression from 'vite-plugin-compression2'

// https://vite.dev/config/
export default defineConfig({
  plugins: [
    vue(),
    eslint(),
    compression({
      deleteOriginalAssets: true,
      skipIfLargerOrEqual: true,
      threshold: 10240,
      filename: '[path][base]'
    }),
    vueI18n({
      compositionOnly: false
    })
  ],
  server: {
    proxy: {
      '/devs': {
        target: 'http://127.0.0.1:5913'
      },
      '/signin': {
        target: 'http://127.0.0.1:5913'
      },
      '/signout': {
        target: 'http://127.0.0.1:5913'
      },
      '/allowsignup': {
        target: 'http://127.0.0.1:5913'
      },
      '/alive': {
        target: 'http://127.0.0.1:5913'
      },
      '/signup': {
        target: 'http://127.0.0.1:5913'
      },
      '/isadmin': {
        target: 'http://127.0.0.1:5913'
      },
      '/users': {
        target: 'http://127.0.0.1:5913'
      },
      '/bind': {
        target: 'http://127.0.0.1:5913'
      },
      '/unbind': {
        target: 'http://127.0.0.1:5913'
      },
      '/delete': {
        target: 'http://127.0.0.1:5913'
      },
      '^/cmd/.*': {
        target: 'http://127.0.0.1:5913'
      },
      '^/connect/.*': {
        ws: true,
        target: 'http://127.0.0.1:5913'
      },
      '^/fontsize': {
        target: 'http://127.0.0.1:5913'
      },
      '^/authorized/.*': {
        target: 'http://127.0.0.1:5913'
      },
      '^/web/*': {
        target: 'http://127.0.0.1:5913'
      },
      '^/file/.*': {
        target: 'http://127.0.0.1:5913'
      }
    }
  }
})
