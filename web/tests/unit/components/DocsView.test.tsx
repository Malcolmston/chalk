import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { DocsView } from '../../../src/components/DocsView';

describe('DocsView', () => {
  it('renders the docs heading and links to the co-located ./api/ reference', () => {
    const { container } = render(<DocsView />);
    expect(container.querySelector('#view-docs')).not.toBeNull();
    expect(screen.getByRole('heading', { level: 2, name: /API documentation/ })).toBeInTheDocument();
    const apiLink = screen.getByRole('link', { name: /Open the API reference/ });
    expect(apiLink).toHaveAttribute('href', './api/');
  });

  it('lists the chalk subpackages (prompts, figlet)', () => {
    const { container } = render(<DocsView />);
    const pkgList = container.querySelector('ul.feat');
    expect(pkgList).not.toBeNull();
    const bolds = Array.from(pkgList!.querySelectorAll('b')).map((b) => b.textContent);
    expect(bolds).toContain('chalk/prompts');
    expect(bolds).toContain('chalk/figlet');
  });
});
