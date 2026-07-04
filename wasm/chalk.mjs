// Idiomatic JS wrapper around the chalk WebAssembly adapter.
//
//   import { loadChalk } from './chalk.mjs';
//   const chalk = await loadChalk();        // browser (fetch) or Node
//   console.log(chalk.red('error!'));
//   console.log(chalk.style('hi', ['bold','underline'], { hex:'#ff8800' }));
//   console.log(chalk.figletFont('banner', 'GO'));
//
// The same Go implementation that powers the Go module runs here via wasm.

async function ensureGo() {
  if (typeof globalThis.Go === 'function') return;
  if (typeof window === 'undefined') {
    // Node: wasm_exec.js is a classic script that assigns globalThis.Go.
    const { readFileSync } = await import('node:fs');
    const { fileURLToPath } = await import('node:url');
    const path = fileURLToPath(new URL('./wasm_exec.js', import.meta.url));
    const { runInThisContext } = await import('node:vm');
    runInThisContext(readFileSync(path, 'utf8'));
  } else {
    await import('./wasm_exec.js');
  }
}

async function readWasm(wasmPath) {
  if (typeof window === 'undefined') {
    const { readFileSync } = await import('node:fs');
    const { fileURLToPath } = await import('node:url');
    const p = wasmPath ?? fileURLToPath(new URL('./chalk.wasm', import.meta.url));
    return readFileSync(p);
  }
  const res = await fetch(wasmPath ?? new URL('./chalk.wasm', import.meta.url));
  return new Uint8Array(await res.arrayBuffer());
}

export async function loadChalk(wasmPath) {
  await ensureGo();
  const go = new globalThis.Go();
  const bytes = await readWasm(wasmPath);
  const { instance } = await WebAssembly.instantiate(bytes, go.importObject);
  go.run(instance); // long-running; resolves when the module exits (it won't)
  const g = globalThis.__mgo_chalk;
  if (!g) throw new Error('chalk wasm did not register __mgo_chalk');

  const named = (name) => (text) => g.style(String(text), [name]);
  const api = {
    style: (text, styles = [], opts = {}) => g.style(String(text), styles, opts),
    hex: (text, hex) => g.style(String(text), [], { hex }),
    rgb: (text, r, gc, b) => g.style(String(text), [], { rgb: [r, gc, b] }),
    strip: (s) => g.strip(String(s)),
    visibleLength: (s) => g.visibleLength(String(s)),
    figlet: (t) => g.figlet(String(t)),
    figletFont: (font, t) => g.figletFont(String(font), String(t)),
    fonts: () => g.fonts(),
  };
  for (const n of ['red','green','blue','yellow','magenta','cyan','white','gray',
                    'bold','dim','italic','underline','inverse','strikethrough']) {
    api[n] = named(n);
  }
  return api;
}
