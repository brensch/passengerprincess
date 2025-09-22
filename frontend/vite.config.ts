import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
    plugins: [react()],
    build: {
        rollupOptions: {
            output: {
                manualChunks: {
                    'react-vendor': ['react', 'react-dom'],
                    'leaflet-vendor': ['leaflet']
                }
            }
        },
        minify: 'esbuild',
        cssMinify: true
    },
    server: {
        proxy: {
            '/api': 'http://127.0.0.1:8040',
            '/route': 'http://127.0.0.1:8040',
            '/superchargers': 'http://127.0.0.1:8040',
            '/autocomplete': 'http://127.0.0.1:8040',
            '/reverse-geocode': 'http://127.0.0.1:8040'
        }
    }
})
