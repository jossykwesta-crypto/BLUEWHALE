/**
 * Ladle stories for individual sub-components:
 * TypeBadge, WarningList, MemoField.
 */
import React, { useState } from 'react';
import type { Story } from '@ladle/react';
import { TypeBadge } from '../components/TypeBadge';
import { WarningList } from '../components/WarningList';
import { MemoField, type MemoType } from '../components/MemoField';

// ─── TypeBadge ────────────────────────────────────────────────────────────────

export const TypeBadgeAll: Story = () => (
  <div style={{ display: 'flex', gap: '0.5rem', fontFamily: 'sans-serif', padding: '1rem' }}>
    <TypeBadge type="G" />
    <TypeBadge type="M" />
    <TypeBadge type="C" />
  </div>
);
TypeBadgeAll.storyName = 'TypeBadge – all variants';

export const TypeBadgeCustom: Story = () => (
  <TypeBadge
    type="G"
    style={{ backgroundColor: '#dcfce7', color: '#14532d', fontSize: '1rem' }}
    className="my-custom-badge"
  />
);
TypeBadgeCustom.storyName = 'TypeBadge – custom style';

// ─── WarningList ──────────────────────────────────────────────────────────────

export const WarningListBasic: Story = () => (
  <WarningList
    warnings={[
      'Contract addresses cannot be used for standard payments.',
      'Memo is ignored when using an M-address.',
    ]}
  />
);
WarningListBasic.storyName = 'WarningList – two warnings';

export const WarningListEmpty: Story = () => <WarningList warnings={[]} />;
WarningListEmpty.storyName = 'WarningList – empty (renders nothing)';

export const WarningListCustom: Story = () => (
  <WarningList
    warnings={['This is a custom-styled warning.']}
    style={{ backgroundColor: '#fef2f2', borderColor: '#fca5a5', color: '#7f1d1d' }}
    className="my-warning"
  />
);
WarningListCustom.storyName = 'WarningList – custom style';

// ─── MemoField ────────────────────────────────────────────────────────────────

export const MemoFieldVisible: Story = () => {
  const [value, setValue] = useState('');
  return (
    <div style={{ maxWidth: '24rem', padding: '1rem', fontFamily: 'sans-serif' }}>
      <MemoField isVisible value={value} onChange={setValue} />
      <p style={{ color: '#64748b', fontSize: '0.875rem', marginTop: '0.5rem' }}>
        Current value: <code>{value || '(empty)'}</code>
      </p>
    </div>
  );
};
MemoFieldVisible.storyName = 'MemoField – visible';

export const MemoFieldWithType: Story = () => {
  const [value, setValue] = useState('');
  const [memoType, setMemoType] = useState<MemoType>('id');
  return (
    <div style={{ maxWidth: '24rem', padding: '1rem', fontFamily: 'sans-serif' }}>
      <MemoField
        isVisible
        value={value}
        onChange={setValue}
        memoType={memoType}
        onMemoTypeChange={setMemoType}
      />
      <p style={{ color: '#64748b', fontSize: '0.875rem', marginTop: '0.5rem' }}>
        Resize below 480px: the type selector stacks above the input.
      </p>
    </div>
  );
};
MemoFieldWithType.storyName = 'MemoField – with memo type';

export const MemoFieldHidden: Story = () => (
  <div style={{ padding: '1rem', fontFamily: 'sans-serif', color: '#64748b' }}>
    <MemoField isVisible={false} value="" onChange={() => {}} />
    <em>MemoField is hidden — nothing rendered above.</em>
  </div>
);
MemoFieldHidden.storyName = 'MemoField – hidden';
