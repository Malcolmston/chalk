import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { Install } from '../../../src/components/Install';
import { LIBS } from '../../../src/data';

const chalk = LIBS[0];

describe('Install', () => {
  it('renders the go get install command and heading', () => {
    const { container } = render(<Install lib={chalk} />);
    expect(screen.getByRole('heading', { name: 'Install' })).toBeInTheDocument();
    expect(container.querySelector('#overview-install')).not.toBeNull();
    expect(screen.getByText(new RegExp(`go get ${chalk.pkg}`))).toBeInTheDocument();
  });
});
