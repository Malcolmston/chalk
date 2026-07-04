import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { NodeVsGo } from '../../../src/components/NodeVsGo';
import { LIBS } from '../../../src/data';

const chalk = LIBS[0];

describe('NodeVsGo', () => {
  it('renders both Node.js and Go comparison columns', () => {
    const { container } = render(<NodeVsGo lib={chalk} />);
    expect(container.querySelector('#overview-cmp')).not.toBeNull();
    expect(screen.getByText('Node.js')).toBeInTheDocument();
    expect(screen.getByText('Go')).toBeInTheDocument();
    // Two code cards inside the compare grid.
    expect(container.querySelectorAll('.compare .code').length).toBe(2);
  });
});
