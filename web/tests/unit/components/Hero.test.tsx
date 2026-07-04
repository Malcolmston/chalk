import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import { Hero } from '../../../src/components/Hero';
import { LIBS } from '../../../src/data';

const chalk = LIBS[0];

describe('Hero', () => {
  beforeEach(() => {
    global.fetch = vi.fn().mockReturnValue(new Promise(() => {}));
  });

  it('renders the library name, module path and tagline', () => {
    render(<Hero lib={chalk} />);
    expect(screen.getByRole('heading', { level: 2, name: /chalk/ })).toBeInTheDocument();
    expect(screen.getByText(chalk.pkg)).toBeInTheDocument();
  });

  it('renders the GitHub link opening safely in a new tab', () => {
    render(<Hero lib={chalk} />);
    const github = screen.getByRole('link', { name: /GitHub/ });
    expect(github).toHaveAttribute('href', chalk.repo);
    expect(github).toHaveAttribute('target', '_blank');
    expect(github).toHaveAttribute('rel', expect.stringContaining('noopener'));
  });

  it('links the API docs to the co-located ./api/ path', () => {
    render(<Hero lib={chalk} />);
    expect(screen.getByRole('link', { name: /API docs/ })).toHaveAttribute('href', './api/');
  });
});
