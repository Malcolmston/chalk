# chalk — JavaScript adapter (WebAssembly)

Run the **same Go implementation** of chalk from JavaScript — in the browser or
Node — via WebAssembly. No reimplementation: `main.go` exposes chalk's portable,
pure functions to JS and `chalk.mjs` wraps them in an idiomatic API.

## Build

```sh
./build.sh          # produces chalk.wasm (+ copies the Go wasm_exec.js runtime)
```

## Use (Node or browser)

```js
import { loadChalk } from './chalk.mjs';
const chalk = await loadChalk();

chalk.red('error!');                                   // ANSI string
chalk.style('hi', ['bold', 'underline'], { hex: '#ff8800' });
chalk.strip(chalk.red('x'));                           // 'x'
chalk.figletFont('banner', 'GO');                      // ASCII-art banner
chalk.fonts();                                         // bundled figlet fonts
```

## Verify

```sh
./build.sh && node test.mjs
```

The adapter is compiled with `GOOS=js GOARCH=wasm`; on normal platforms `stub.go`
keeps `go build ./...` and CI green. Build artifacts (`*.wasm`) are gitignored.
