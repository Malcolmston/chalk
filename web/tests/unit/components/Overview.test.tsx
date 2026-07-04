import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import { Overview } from '../../../src/components/Overview';
import { LIBS } from '../../../src/data';

const chalk = LIBS[0];

describe('Overview', () => {
  beforeEach(() => {
    global.fetch = vi.fn().mockReturnValue(new Promise(() => {}));
  });

  it('renders the overview view with all sub-sections', () => {
    const { container } = render(<Overview />);
    expect(container.querySelector('#view-overview')).not.toBeNull();
    expect(screen.getByRole('heading', { level: 2, name: /chalk/ })).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: 'Install' })).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: 'Quick start' })).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: 'Going further' })).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: 'Features' })).toBeInTheDocument();
    // Every feature bullet rendered.
    expect(container.querySelectorAll('ul.feat li').length).toBe(chalk.features.length);
  });

  it('every in-page jump-nav anchor resolves to an element id', () => {
    const { container } = render(<Overview />);
    const anchors = Array.from(container.querySelectorAll('.onthispage a')) as HTMLAnchorElement[];
    expect(anchors.length).toBeGreaterThan(0);
    for (const a of anchors) {
      const id = (a.getAttribute('href') ?? '').slice(1);
      expect(container.querySelector(`#${id}`), `missing target #${id}`).not.toBeNull();
    }
  });
});
