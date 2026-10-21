# @redishfish/bluewhale

React UI components for Stellar address input, validation, and routing ID display.

## Install

```bash
npm install @redishfish/bluewhale
```

> **Peer dependencies:** React ≥ 17

## Quick Start

```tsx
import { AddressInput } from '@redishfish/bluewhale';

function WithdrawalForm() {
  const [canSubmit, setCanSubmit] = React.useState(false);

  return (
    <form>
      <AddressInput
        showCopyButton
        onValidationChange={(isValid) => setCanSubmit(isValid)}
      />
      <button type="submit" disabled={!canSubmit}>
        Submit
      </button>
    </form>
  );
}
```

---

## API

### `<AddressInput />`

| Prop | Type | Default | Description |
|---|---|---|---|
| `onValidationChange` | `(isValid: boolean, result: RoutingResult) => void` | – | Fired on every input change. `isValid` is `false` for C-addresses and unrecognized prefixes. |
| `showCopyButton` | `boolean` | `false` | Renders one-click copy buttons for the resolved base account and routing ID, with a `✓ Copied!` tooltip. |
| `className` | `string` | – | Extra CSS classes merged onto the root `<div>`. Combine with Tailwind or your own CSS. |
| `style` | `CSSProperties` | – | Inline styles applied to the root element (merged with defaults). |

#### `RoutingResult`

```ts
interface RoutingResult {
  destinationBaseAccount: string | null;
  routingId: string | null;
  warnings: string[];
  isValid: boolean;
}
```

### `<TypeBadge type="G" | "M" | "C" | "UNKNOWN" />`

| Prop | Type | Description |
|---|---|---|
| `type` | `AddressType` | Address type to display. Renders `null` for `"UNKNOWN"`. |
| `className` | `string` | Extra CSS classes. |
| `style` | `CSSProperties` | Inline style overrides. |

### `<WarningList warnings={[...]} />`

| Prop | Type | Description |
|---|---|---|
| `warnings` | `string[]` | List of warning strings. Renders `null` when empty. |
| `className` | `string` | Extra CSS classes. |
| `style` | `CSSProperties` | Inline style overrides. |

### `<MemoField isVisible value onChange />`

| Prop | Type | Description |
|---|---|---|
| `isVisible` | `boolean` | Whether to render the field. |
| `value` | `string` | Controlled value. |
| `onChange` | `(v: string) => void` | Change handler. |
| `memoType` | `MemoType` | Selected memo type (`none`, `id`, `text`, `hash`, `return`). Defaults to `text`. |
| `onMemoTypeChange` | `(t: MemoType) => void` | Passing this shows the memo type selector. |
| `memoTypes` | `MemoType[]` | Types offered in the selector. Defaults to all five. |
| `className` | `string` | Extra CSS classes. |
| `style` | `CSSProperties` | Inline style overrides. |

The type selector and the memo input share one row, which wraps so the selector
sits above the input on narrow screens. Both controls keep a 44×44px minimum
touch target. With `memoType="none"`, the memo input is disabled.

### Optional stylesheet

The components need no CSS import. To pin the mobile layout to an explicit
breakpoint (the memo row stacks below 480px), import the stylesheet once:

```ts
import '@redishfish/bluewhale/styles.css';
```

---

## Styling Customization

### BEM class names

Every component exposes stable BEM class names for CSS targeting. Inline styles are additive — you can override any style property.

| Class | Element |
|---|---|
| `bw-address-input` | Root wrapper of `<AddressInput>` |
| `bw-address-input__input` | The address `<input>` |
| `bw-address-input__badge` | Badge container |
| `bw-address-input__field-wrap` | Flex wrapper around input + badge |
| `bw-type-badge` | `<TypeBadge>` root |
| `bw-type-badge--g` | Modifier for G-addresses |
| `bw-type-badge--m` | Modifier for M-addresses |
| `bw-type-badge--c` | Modifier for C-addresses |
| `bw-warning-list` | `<WarningList>` root |
| `bw-warning-list__title` | "⚠️ Warnings:" heading |
| `bw-warning-list__items` | `<ul>` container |
| `bw-warning-item` | Individual `<li>` warning |
| `bw-memo-field` | `<MemoField>` root |
| `bw-memo-field__label` | Label element |
| `bw-memo-field__row` | Row holding the type selector and the input |
| `bw-memo-field__type` | Memo type `<select>` |
| `bw-memo-field__input` | Memo `<input>` |
| `bw-memo-field__hint` | Hint text below the input |
| `bw-routing-summary` | Resolved address summary box |
| `bw-routing-summary__row` | Individual summary row |
| `bw-routing-summary__value` | Value span in a summary row |
| `bw-copy-button` | Copy-to-clipboard button |

### Tailwind CSS example

```tsx
<AddressInput
  className="rounded-2xl border border-indigo-300 p-4 shadow-md"
  style={{ maxWidth: '28rem' }}
/>
```

Override individual sub-elements via global CSS:

```css
/* Your global stylesheet */
.bw-address-input__input {
  @apply rounded-xl border-indigo-300 focus:ring-2 focus:ring-indigo-500;
}

.bw-type-badge--m {
  @apply bg-purple-200 text-purple-900;
}
```

### Headless / unstyled mode

Override _all_ inline styles by passing an empty `style` prop and relying entirely on your own classes:

```tsx
<AddressInput
  style={{}}             /* clears maxWidth / margin defaults */
  className="my-address-input"
/>
```

Then in your CSS:

```css
.my-address-input { /* ... */ }
.my-address-input .bw-address-input__input { /* ... */ }
```

### Dark mode example

```tsx
<AddressInput showCopyButton className="bw-dark" />
```

```css
.bw-dark .bw-address-input__input {
  background-color: #1e293b;
  color: #e2e8f0;
  border-color: #334155;
}

.bw-dark .bw-routing-summary {
  background-color: #1e293b;
  border-color: #334155;
  color: #e2e8f0;
}
```

---

## Interactive Preview (Ladle)

A full component sandbox is available via [Ladle](https://ladle.dev):

```bash
# Install dev dependencies, then:
npm run ladle

# Build static storybook:
npm run build-storybook
```

Stories cover: G-address, M-address, C-address, memo warnings, dark mode, copy buttons, and `onValidationChange` callback.

---

## Development

```bash
# Build
npm run build

# Tests
npm test

# Watch mode
npm run test:watch
```

## License

MIT
