import type { CSSProperties } from 'react';
import { LIBS } from '../data';

const CHALK = LIBS[0];

// DocsView is the "docs" tab: it points at the generated go/doc API reference,
// published alongside this landing site under ./api/ on GitHub Pages.
export function DocsView() {
  return (
    <section className="view active" id="view-docs">
      <div className="sec-h"><span className="bar" /><h2 style={{ margin: 0 }}>API documentation</h2></div>
      <p className="muted">The full API reference is generated straight from the source with the
        dependency-free <code>go/doc</code> tool committed in <code>docs/gen</code>, so it never drifts from the
        code. It is published alongside this site and covers the root <code>chalk</code> package plus the
        <code>chalk/prompts</code> and <code>chalk/figlet</code> subpackages.</p>

      <div className="cta" style={{ margin: '1.6rem 0' }}>
        <a className="btn primary" href="./api/"><i className="fa-solid fa-book" />&nbsp;Open the API reference →</a>
        <a className="btn" href={CHALK.repo} target="_blank" rel="noopener"><i className="fa-brands fa-github" />&nbsp;Source on GitHub</a>
      </div>

      <div className="sec-h"><span className="bar" /><h3 style={{ margin: 0 }}>Packages</h3></div>
      <ul className="feat" style={{ '--lib-accent': CHALK.accent } as CSSProperties}>
        <li><b>chalk</b> — chainable, immutable ANSI terminal styling (16 / 256 / truecolor).</li>
        <li><b>chalk/prompts</b> — inquirer-style interactive prompts (input, confirm, select, multiselect, password).</li>
        <li><b>chalk/figlet</b> — FIGfont ASCII-art banners with bundled fonts, gradients and rainbow coloring.</li>
      </ul>

      <div className="note">Can't reach the reference above? It's also served at{' '}
        <a href={CHALK.docs} target="_blank" rel="noopener">{CHALK.docs}</a>.</div>
    </section>
  );
}
