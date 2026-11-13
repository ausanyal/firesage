import React, { useEffect, useState } from 'react';

interface BufferedNumberInputProps {
  id?: string;
  name: string;
  value: number;
  min?: number;
  max?: number;
  step?: number;
  ariaLabel?: string;
  onCommit: (nextValue: number) => void;
}

/**
 * Accessible numeric input that buffers raw string keystrokes while focused so users
 * can freely erase all digits and type a new value from scratch without premature
 * min/default clamping, committing in-range edits live and clamping on blur.
 */
export const BufferedNumberInput: React.FC<BufferedNumberInputProps> = ({
  id,
  name,
  value,
  min,
  max,
  step,
  ariaLabel,
  onCommit,
}) => {
  const [focused, setFocused] = useState(false);
  const [draft, setDraft] = useState<string>(() => String(value));

  useEffect(() => {
    if (!focused) {
      setDraft(String(value));
    }
  }, [value, focused]);

  const clampValue = (raw: number): number => {
    let v = raw;
    if (min !== undefined && v < min) v = min;
    if (max !== undefined && v > max) v = max;
    return v;
  };

  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const rawStr = e.target.value;
    setDraft(rawStr);

    const trimmed = rawStr.trim();
    if (trimmed === '' || trimmed === '-' || trimmed === '+') {
      return;
    }

    const parsed = Number(trimmed);
    if (!Number.isFinite(parsed)) {
      return;
    }

    const aboveMin = min === undefined || parsed >= min;
    const belowMax = max === undefined || parsed <= max;
    if (aboveMin && belowMax) {
      onCommit(parsed);
    }
  };

  const finalize = () => {
    setFocused(false);
    const trimmed = draft.trim();
    if (trimmed === '' || trimmed === '-' || trimmed === '+') {
      setDraft(String(value));
      return;
    }
    const parsed = Number(trimmed);
    if (!Number.isFinite(parsed)) {
      setDraft(String(value));
      return;
    }
    const clamped = clampValue(parsed);
    setDraft(String(clamped));
    if (clamped !== value) {
      onCommit(clamped);
    }
  };

  const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Enter') {
      if (e.nativeEvent.isComposing || e.keyCode === 229) {
        return;
      }
      e.preventDefault();
      e.currentTarget.blur();
    }
  };

  return (
    <input
      id={id}
      name={name}
      type="number"
      min={min}
      max={max}
      step={step}
      aria-label={ariaLabel}
      enterKeyHint="done"
      value={focused ? draft : String(value)}
      onFocus={() => {
        setFocused(true);
        setDraft(String(value));
      }}
      onChange={handleChange}
      onBlur={finalize}
      onKeyDown={handleKeyDown}
    />
  );
};
