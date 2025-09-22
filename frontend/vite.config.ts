import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import http from 'http'

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
            '/api': {
                target: 'http://127.0.0.1:8040',
                changeOrigin: true,
                agent: new http.Agent({ keepAlive: true })
            },
            '/route': {
                target: 'http://127.0.0.1:8040',
                changeOrigin: true,
                agent: new http.Agent({ keepAlive: true })
            },
            '/superchargers': {
                target: 'http://127.0.0.1:8040',
                changeOrigin: true,
                agent: new http.Agent({ keepAlive: true })
            },
            '/autocomplete': {
                target: 'http://127.0.0.1:8040',
                changeOrigin: true,
                agent: new http.Agent({ keepAlive: true })
            },
            '/reverse-geocode': {
                target: 'http://127.0.0.1:8040',
                changeOrigin: true,
                agent: new http.Agent({ keepAlive: true })
            }
        }
    }
})