import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { Features } from '../../../src/components/Features';
import { LIBS } from '../../../src/data';

const chalk = LIBS[0];

describe('Features', () => {
  it('renders one bullet per feature under the Features heading', () => {
    const { container } = render(<Features lib={chalk} />);
    expect(screen.getByRole('heading', { name: 'Features' })).toBeInTheDocument();
    expect(container.querySelector('#overview-feat')).not.toBeNull();
    expect(container.querySelectorAll('ul.feat li').length).toBe(chalk.features.length);
  });
});
