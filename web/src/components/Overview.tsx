import { CodeBlock } from 'go-ui';
import { LIBS } from '../data';
import { Hero } from './Hero';
import { Install } from './Install';
import { QuickStart } from './QuickStart';
import { NodeVsGo } from './NodeVsGo';
import { Features } from './Features';

const CHALK = LIBS[0];

// Overview is the default tab: the library hero followed by install, quick
// start, a Node→Go comparison, a "going further" sample and the feature list.
export function Overview() {
  const lib = CHALK;
  return (
    <section className="view active" id="view-overview">
      <Hero lib={lib} />

      <p className="muted" dangerouslySetInnerHTML={{ __html: lib.blurb }} />
      <div className="onthispage">
        <a href="#overview-install">Install</a>
        <a href="#overview-quick">Quick start</a>
        <a href="#overview-cmp">Node → Go</a>
        <a href="#overview-more">Going further</a>
        <a href="#overview-feat">Features</a>
      </div>

      <Install lib={lib} />
      <QuickStart lib={lib} />
      <NodeVsGo lib={lib} />

      <div className="sec-h" id="overview-more"><span className="bar" /><h3 style={{ margin: 0 }}>Going further</h3></div>
      <CodeBlock lang="go" html={lib.integrate} />

      <Features lib={lib} />

      <div className="note">Full API reference &amp; runnable examples: <a href="./api/">{lib.docs}</a></div>
    </section>
  );
}
