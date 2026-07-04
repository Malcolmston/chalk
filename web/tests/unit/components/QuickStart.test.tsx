import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { QuickStart } from '../../../src/components/QuickStart';
import { LIBS } from '../../../src/data';

const chalk = LIBS[0];

describe('QuickStart', () => {
  it('renders the quick-start heading and Go snippet', () => {
    const { container } = render(<QuickStart lib={chalk} />);
    expect(screen.getByRole('heading', { name: 'Quick start' })).toBeInTheDocument();
    expect(container.querySelector('#overview-quick')).not.toBeNull();
    // A recognisable token from the chalk go_code sample.
    expect(container.querySelector('pre code')?.textContent).toMatch(/chalk\.New\(\)/);
  });
});
