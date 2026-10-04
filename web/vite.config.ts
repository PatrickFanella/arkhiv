import { defineConfig, loadEnv, type ViteDevServer } from 'vite';
import react from '@vitejs/plugin-react';
import { cpSync, mkdirSync, createReadStream, existsSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { createRequire } from 'node:module';

const require = createRequire(import.meta.url);
function pdfAssets() {
  const root = dirname(require.resolve('pdfjs-dist/package.json'));
  return {name:'pdf-reader-assets',configureServer(server: ViteDevServer) {
    server.middlewares.use((request,response,next)=> {
      const path = request.url?.split('?')[0] || '';
      if (!path.startsWith('/pdfjs/')) return next();
      if (!/^\/pdfjs\/(cmaps|standard_fonts|wasm)\/[A-Za-z0-9_-]+\.[A-Za-z0-9]+$/.test(path)) {response.statusCode=404;response.end();return;}
      const file = join(root,path.slice(7));
      if (!existsSync(file)) {response.statusCode=404;response.end();return;}
      response.setHeader('Content-Type',path.endsWith('.wasm') ? 'application/wasm' : 'application/octet-stream');
      response.setHeader('X-Content-Type-Options','nosniff');
      const stream = createReadStream(file);
      stream.on('error',()=>response.destroy());stream.pipe(response);
    });
  },closeBundle() {
    mkdirSync('dist/pdfjs',{recursive:true});
    for (const directory of ['cmaps','standard_fonts','wasm']) cpSync(join(root,directory),join('dist/pdfjs',directory),{recursive:true});
  }};
}

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, '.', 'VHNESS_');
  const proxy = { '/api': env.VHNESS_API || 'http://127.0.0.1:8080' };
  return { plugins: [react(),pdfAssets()], server: { port: 5173, strictPort: true, proxy }, preview: { proxy } };
});
