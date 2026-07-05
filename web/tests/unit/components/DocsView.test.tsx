import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import { DocsView } from '../../../src/components/DocsView';
import type { DocIndex } from 'go-ui';

// A minimal DocIndex the stubbed fetch returns for DocsApp's doc.json request.
const DOC_INDEX: DocIndex = {
  module: 'github.com/malcolmston/chalk',
  packages: [
    {
      importPath: 'github.com/malcolmston/chalk',
      name: 'chalk',
      synopsis: 'Package chalk is terminal string styling done right.',
      doc: 'Package chalk is terminal string styling done right.',
      consts: [],
      vars: [],
      types: [
        {
          name: 'Chalk',
          signature: 'type Chalk struct{}',
          doc: 'Chalk is a chainable, immutable style builder.',
          consts: [],
          vars: [],
          funcs: [],
          methods: [],
        },
      ],
      funcs: [{ name: 'New', signature: 'func New() *Chalk', doc: 'New creates a Chalk.' }],
    },
  ],
};

describe('DocsView', () => {
  beforeEach(() => {
    // DocsApp fetches doc.json; return the small index. Any non-doc request is
    // left pending so it never resolves.
    global.fetch = vi.fn((input: RequestInfo | URL) => {
      if (String(input).includes('doc.json')) {
        return Promise.resolve({ ok: true, json: () => Promise.resolve(DOC_INDEX) } as Response);
      }
      return new Promise<Response>(() => {});
    }) as unknown as typeof fetch;
  });

  it('renders the inline React API reference from the fetched doc.json', async () => {
    const { container } = render(<DocsView />);
    expect(container.querySelector('#view-docs')).not.toBeNull();
    expect(
      screen.getByRole('heading', { level: 2, name: /API documentation/ }),
    ).toBeInTheDocument();

    // DocsApp fetches asynchronously, then renders the package view + symbols.
    expect(await screen.findByRole('heading', { name: /package chalk/ })).toBeInTheDocument();
    expect(container.querySelector('#sym-New'), 'func New symbol card').not.toBeNull();
    expect(container.querySelector('#sym-Chalk'), 'type Chalk symbol card').not.toBeNull();

    // The secondary link to the raw generated static HTML remains.
    expect(screen.getByRole('link', { name: /raw generated HTML/i })).toHaveAttribute('href', './api/');
  });
});
