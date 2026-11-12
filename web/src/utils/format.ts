import type { CountryCode } from '../types';

export function formatMoneyCompact(amount: number, country: CountryCode): string {
  const sign = amount < 0 ? '-' : '';
  const abs = Math.abs(amount);

  if (country === 'IN') {
    if (abs >= 1e7) {
      const cr = abs / 1e7;
      return `${sign}₹${cr >= 100 ? cr.toFixed(0) : cr >= 10 ? cr.toFixed(1) : cr.toFixed(2)} Cr`;
    }
    if (abs >= 1e5) {
      const l = abs / 1e5;
      return `${sign}₹${l >= 10 ? l.toFixed(1) : l.toFixed(2)} L`;
    }
    if (abs >= 1e3) {
      return `${sign}₹${(abs / 1e3).toFixed(0)}K`;
    }
    return `${sign}₹${Math.round(abs).toLocaleString('en-IN')}`;
  }

  // United States ($ USD)
  if (abs >= 1e9) {
    return `${sign}$${(abs / 1e9).toFixed(2)}B`;
  }
  if (abs >= 1e6) {
    const m = abs / 1e6;
    return `${sign}$${m >= 100 ? m.toFixed(0) : m >= 10 ? m.toFixed(1) : m.toFixed(2)}M`;
  }
  if (abs >= 1e3) {
    const k = abs / 1e3;
    return `${sign}$${k >= 100 ? k.toFixed(0) : k.toFixed(1)}K`;
  }
  return `${sign}$${Math.round(abs).toLocaleString('en-US')}`;
}

export function formatMoneyFull(amount: number, country: CountryCode): string {
  const sign = amount < 0 ? '-' : '';
  const abs = Math.round(Math.abs(amount));
  if (country === 'IN') {
    return `${sign}₹${abs.toLocaleString('en-IN')}`;
  }
  return `${sign}$${abs.toLocaleString('en-US')}`;
}

export function currencySymbol(country: CountryCode): string {
  return country === 'IN' ? '₹' : '$';
}

export function formatSignedPct(val: number, digits = 1): string {
  const s = val > 0 ? '+' : '';
  return `${s}${val.toFixed(digits)}%`;
}
