import type { CSSProperties } from 'react';
import type { Lib } from '../data';

export interface FeaturesProps {
  lib: Lib;
}

// Features renders the accent-bulleted feature list for chalk.
export function Features({ lib }: FeaturesProps) {
  return (
    <>
      <div className="sec-h" id="overview-feat"><span className="bar" /><h3 style={{ margin: 0 }}>Features</h3></div>
      <ul className="feat" style={{ '--lib-accent': lib.accent } as CSSProperties}>
        {lib.features.map((f, i) => <li key={i} dangerouslySetInnerHTML={{ __html: f }} />)}
      </ul>
    </>
  );
}
