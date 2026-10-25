import { defineConfig } from 'vite'

export default defineConfig({
  root: '.',
  build: {
    outDir: 'dist',
    emptyOutDir: true,
  },
  server: {
    // Wails v3 开发代理对 localhost 强制 tcp4（见 assetserver/build_dev.go）。
    // Vite 默认只绑 [::1] 时，代理打 127.0.0.1 会 502，桌面窗白屏。
    host: '127.0.0.1',
    strictPort: true,
  },
})
