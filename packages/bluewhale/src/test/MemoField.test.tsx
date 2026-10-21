import React from 'react';
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoField } from '../components/MemoField';
import { AddressInput } from '../AddressInput';

const M_PREFIX = 'M';

describe('MemoField', () => {
  it('renders nothing when isVisible is false', () => {
    const { container } = render(<MemoField isVisible={false} value="" onChange={() => {}} />);
    expect(container).toBeEmptyDOMElement();
  });

  it('renders the labelled input when isVisible is true', () => {
    render(<MemoField isVisible value="abc" onChange={() => {}} />);
    const input = screen.getByLabelText(/memo \(optional\)/i);
    expect(input).toHaveValue('abc');
  });

  it('calls onChange with the typed value', async () => {
    const onChange = vi.fn();
    render(<MemoField isVisible value="" onChange={onChange} />);
    await userEvent.type(screen.getByLabelText(/memo/i), '7');
    expect(onChange).toHaveBeenCalledWith('7');
  });

  it('merges className onto the root element', () => {
    const { container } = render(
      <MemoField isVisible value="" onChange={() => {}} className="custom" />,
    );
    expect(container.firstElementChild).toHaveClass('bw-memo-field', 'custom');
  });
});

describe('MemoField memo type selector (#237)', () => {
  it('is not rendered unless onMemoTypeChange is passed', () => {
    render(<MemoField isVisible value="" onChange={() => {}} />);
    expect(screen.queryByRole('combobox', { name: /memo type/i })).not.toBeInTheDocument();
  });

  it('renders every memo type and reports the selection', async () => {
    const onMemoTypeChange = vi.fn();
    render(
      <MemoField
        isVisible
        value=""
        onChange={() => {}}
        memoType="text"
        onMemoTypeChange={onMemoTypeChange}
      />,
    );
    const select = screen.getByRole('combobox', { name: /memo type/i });
    expect(select).toHaveValue('text');
    expect(screen.getAllByRole('option').map((o) => o.getAttribute('value'))).toEqual([
      'none',
      'id',
      'text',
      'hash',
      'return',
    ]);
    await userEvent.selectOptions(select, 'id');
    expect(onMemoTypeChange).toHaveBeenCalledWith('id');
  });

  it('limits the options to memoTypes', () => {
    render(
      <MemoField
        isVisible
        value=""
        onChange={() => {}}
        memoType="id"
        memoTypes={['id', 'text']}
        onMemoTypeChange={() => {}}
      />,
    );
    expect(screen.getAllByRole('option')).toHaveLength(2);
  });

  it('disables the value input when the type is none', () => {
    render(
      <MemoField isVisible value="" onChange={() => {}} memoType="none" onMemoTypeChange={() => {}} />,
    );
    expect(screen.getByLabelText(/memo \(optional\)/i)).toBeDisabled();
  });
});

describe('MemoField mobile layout (#237)', () => {
  const renderWithType = () =>
    render(
      <MemoField isVisible value="" onChange={() => {}} memoType="id" onMemoTypeChange={() => {}} />,
    );

  it('puts the selector and the input in a wrapping row', () => {
    const { container } = renderWithType();
    const row = container.querySelector('.bw-memo-field__row') as HTMLElement;
    expect(row).not.toBeNull();
    expect(row.style.flexWrap).toBe('wrap');
    // Selector first, so it lands above the input when the row wraps.
    expect(row.firstElementChild).toHaveClass('bw-memo-field__type');
    expect(row.lastElementChild).toHaveClass('bw-memo-field__input');
  });

  it('gives both controls a 44px minimum touch target', () => {
    renderWithType();
    expect(screen.getByRole('combobox', { name: /memo type/i }).style.minHeight).toBe('44px');
    expect(screen.getByLabelText(/memo \(optional\)/i).style.minHeight).toBe('44px');
  });

  it('ships a stylesheet that stacks the row below 480px', () => {
    // Vitest runs from the package root; jsdom's import.meta.url is not a file URL.
    const css = readFileSync(resolve(process.cwd(), 'src/styles/components.css'), 'utf8');
    expect(css).toMatch(/@media \(max-width: 479\.98px\)/);
    expect(css).toMatch(/\.bw-memo-field__row\s*{[^}]*flex-direction: column/);
    expect(css).toMatch(/min-height: 44px/);
  });
});

describe('MemoField inside AddressInput', () => {
  it('is hidden until an M-address prefix is detected', async () => {
    render(<AddressInput />);
    expect(screen.queryByLabelText(/memo/i)).not.toBeInTheDocument();
    await userEvent.type(screen.getByRole('textbox', { name: /stellar address/i }), M_PREFIX);
    expect(screen.queryByLabelText(/memo/i)).toBeInTheDocument();
  });
});
