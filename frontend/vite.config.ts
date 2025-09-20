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
            '/api': 'http://localhost:8040',
            '/route': 'http://localhost:8040',
            '/superchargers': 'http://localhost:8040',
            '/autocomplete': 'http://localhost:8040',
            '/reverse-geocode': 'http://localhost:8040'
        }
    }
})
