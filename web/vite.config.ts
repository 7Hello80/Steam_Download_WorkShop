import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { resolve } from 'path'

export default defineConfig({
  plugins: [vue()],
  base: '/',
  resolve: {
    alias: {
      '@': resolve(__dirname, 'src'),
    },
  },
  build: {
    target: 'es2020',
    minify: 'esbuild',
    cssMinify: 'esbuild',
    chunkSizeWarningLimit: 600,
    rollupOptions: {
      output: {
        chunkFileNames: 'js/[name]-[hash].js',
        entryFileNames: 'js/[name]-[hash].js',
        assetFileNames: '[ext]/[name]-[hash].[ext]',
        manualChunks(id) {
          if (id.includes('node_modules')) {
            // Split element-plus separately (largest dep)
            if (id.includes('element-plus') || id.includes('@element-plus')) {
              return 'element-plus'
            }
            // Vue ecosystem
            if (id.includes('vue') || id.includes('pinia') || id.includes('@vueuse') || id.includes('@vue')) {
              return 'vue-vendor'
            }
            // Everything else from node_modules
            return 'vendor'
          }
          // Fix: Go's embed package ignores files starting with _ or .
          // The @vitejs/plugin-vue export helper chunk is named with a leading _ by default.
          // Give it an explicit name so Go embeds it properly.
          if (id.includes('plugin-vue_export-helper') || id.includes('plugin-vue-export-helper')) {
            return 'plugin-vue-export-helper'
          }
        },
      },
    },
  },
  server: {
    port: 3000,
    host: '127.0.0.1',
    proxy: {
      '/api': {
        target: 'http://127.0.0.1:8086',
        changeOrigin: true,
        ws: true,
      },
      '/static': {
        target: 'http://127.0.0.1:8086',
        changeOrigin: true,
      },
    },
  },
})
