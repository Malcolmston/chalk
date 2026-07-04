// Node smoke test: builds must be run first (see build.sh). Verifies the Go
// implementation is reachable from JS through wasm.
import assert from 'node:assert';
import { loadChalk } from './chalk.mjs';

const chalk = await loadChalk();

const red = chalk.red('error');
assert.ok(red.includes('\x1b['), 'expected ANSI escape in styled output');
assert.strictEqual(chalk.strip(red), 'error', 'strip should recover plain text');

const styled = chalk.style('hi', ['bold'], { hex: '#ff8800' });
assert.ok(styled.includes('\x1b[1m') || styled.includes('1;'), 'bold code present');
assert.strictEqual(chalk.strip(styled), 'hi');

const banner = chalk.figletFont('banner', 'GO');
assert.ok(banner.split('\n').length >= 5, 'figlet banner has multiple rows');

assert.ok(Array.isArray(chalk.fonts()) && chalk.fonts().includes('banner'));

console.log('chalk wasm adapter: all JS-side assertions passed');
console.log(chalk.style('  malcolmston/chalk in JS via wasm  ', ['bold'], { hex:'#6ea8ff' }));
process.exit(0);
