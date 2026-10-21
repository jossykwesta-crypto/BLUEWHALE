import React, { CSSProperties } from 'react';

/** Stellar memo types, as accepted by `extractRouting`'s `memoType`. */
export type MemoType = 'none' | 'id' | 'text' | 'hash' | 'return';

const MEMO_TYPE_LABELS: Record<MemoType, string> = {
  none: 'No memo',
  id: 'MEMO_ID',
  text: 'MEMO_TEXT',
  hash: 'MEMO_HASH',
  return: 'MEMO_RETURN',
};

const DEFAULT_MEMO_TYPES: MemoType[] = ['none', 'id', 'text', 'hash', 'return'];

/**
 * Minimum touch target edge, in px. WCAG 2.5.5 and the Apple and Material
 * guidelines all land on 44, so the select and the input never go below it.
 */
const MIN_TOUCH_TARGET = '44px';

interface MemoFieldProps {
  isVisible: boolean;
  value: string;
  onChange: (value: string) => void;
  /**
   * Selected memo type. The type selector is shown only when
   * `onMemoTypeChange` is also passed, so existing usage is unchanged.
   */
  memoType?: MemoType;
  /** Called with the new memo type. Passing it shows the type selector. */
  onMemoTypeChange?: (memoType: MemoType) => void;
  /** Memo types offered in the selector, in order. Defaults to all five. */
  memoTypes?: MemoType[];
  /** Additional CSS class names merged onto the root element. */
  className?: string;
  /** Inline styles applied to the root element. */
  style?: CSSProperties;
}

const controlStyle: CSSProperties = {
  minHeight: MIN_TOUCH_TARGET,
  padding: '0.5rem 0.75rem',
  border: '1px solid #cbd5e1',
  borderRadius: '0.375rem',
  fontSize: '1rem',
  boxSizing: 'border-box',
  outline: 'none',
  minWidth: 0,
};

export const MemoField: React.FC<MemoFieldProps> = ({
  isVisible,
  value,
  onChange,
  memoType = 'text',
  onMemoTypeChange,
  memoTypes = DEFAULT_MEMO_TYPES,
  className,
  style,
}) => {
  if (!isVisible) return null;

  const rootClass = ['bw-memo-field', className].filter(Boolean).join(' ');
  const showTypeSelect = typeof onMemoTypeChange === 'function';

  return (
    <div
      className={rootClass}
      style={{ marginTop: '0.75rem', display: 'flex', flexDirection: 'column', ...style }}
    >
      <label
        htmlFor="memo-input"
        className="bw-memo-field__label"
        style={{ fontSize: '0.875rem', fontWeight: '500', color: '#475569', marginBottom: '0.25rem' }}
      >
        Memo (Optional)
      </label>
      {/*
        The row wraps on its own: the select prefers ~10rem, the input ~14rem
        and grows far faster, so below roughly 24rem of width the select drops
        onto its own line above the input. That works with no stylesheet at
        all. styles/components.css adds an explicit 480px breakpoint for apps
        that import it.
      */}
      <div
        className="bw-memo-field__row"
        style={{ display: 'flex', flexWrap: 'wrap', gap: '0.5rem', alignItems: 'stretch' }}
      >
        {showTypeSelect && (
          <select
            aria-label="Memo type"
            className="bw-memo-field__type"
            value={memoType}
            onChange={(e) => onMemoTypeChange(e.target.value as MemoType)}
            style={{ ...controlStyle, flex: '1 1 10rem', backgroundColor: '#fff' }}
          >
            {memoTypes.map((type) => (
              <option key={type} value={type}>
                {MEMO_TYPE_LABELS[type]}
              </option>
            ))}
          </select>
        )}
        <input
          id="memo-input"
          type="text"
          placeholder="Enter memo here"
          value={value}
          onChange={(e) => onChange(e.target.value)}
          disabled={showTypeSelect && memoType === 'none'}
          className="bw-memo-field__input"
          style={{ ...controlStyle, flex: '999 1 14rem', width: '100%' }}
        />
      </div>
      <div
        className="bw-memo-field__hint"
        style={{ fontSize: '0.75rem', color: '#64748b', marginTop: '0.25rem' }}
      >
        A memo is sometimes required by exchanges.
      </div>
    </div>
  );
};
