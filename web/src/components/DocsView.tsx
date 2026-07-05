import { DocsApp } from 'go-ui';
import { LIBS } from '../data';

const CHALK = LIBS[0];

// DocsView is the "docs" tab. It renders the full, package-by-package Go API
// reference inline via the shared `DocsApp`, which fetches the generated
// `doc.json` (emitted by docs/gen) and shows a package sidebar + package view,
// hash-routable by import path. A secondary link points at the raw generated
// static HTML (`./api/`). It covers the root `chalk` package plus the
// `chalk/prompts` and `chalk/figlet` subpackages.
//
// `doc.json` is served at `<base>/doc.json`. If it is missing, DocsApp degrades
// gracefully (it renders an inline error/loading state rather than crashing).
export function DocsView() {
  return (
    <section className="view active" id="view-docs">
      <div className="sec-h"><span className="bar" /><h2 style={{ margin: 0 }}>API documentation</h2></div>
      <p className="muted">The complete package-by-package Go API reference, generated straight from the source
        with the dependency-free <code>go/doc</code> tool committed in <code>docs/gen</code>, so it never drifts
        from the code. It covers the root <code>chalk</code> package plus the <code>chalk/prompts</code> and
        <code>chalk/figlet</code> subpackages.</p>

      <div className="cta" style={{ margin: '1.6rem 0' }}>
        <a className="btn primary" href="./api/"><i className="fa-solid fa-book" />&nbsp;Open the raw generated HTML →</a>
        <a className="btn" href={CHALK.repo} target="_blank" rel="noopener"><i className="fa-brands fa-github" />&nbsp;Source on GitHub</a>
      </div>

      <DocsApp url={`${import.meta.env.BASE_URL}doc.json`} />
    </section>
  );
}
