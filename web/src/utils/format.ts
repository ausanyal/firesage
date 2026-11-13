import type { CountryCode } from '../types';

export interface CountryMeta {
  code: CountryCode;
  flag: string;
  name: string;
  shortBtn: string;
  currencyCode: string;
  symbol: string;
  locale: string;
  unlockAge: number;
  liquidShort: string;
  lockedShort: string;
  monthlyLockedLabel: string;
  bucketEqLabel: string;
  bucketDebtLabel: string;
  bucketDefLabel: string;
  bucketFreeLabel: string;
  hasFreeBucket: boolean;
  hasNPSBucket: boolean;
  pensionLabel: string;
  pensionShort: string;
  taxOptLabel: string;
  taxAlphaChip: string;
  healthcareShort: string;
  healthcareExplainer: string;
  residencySummary: string;
  defaultTier2CityId: string;
  corpusStep: number;
  savingsStep: number;
  spendStep: number;
  minSpend: number;
  defaultMilestoneAmt: number;
}

export const COUNTRY_META: Record<CountryCode, CountryMeta> = {
  IN: {
    code: 'IN',
    flag: '🇮🇳',
    name: 'India',
    shortBtn: '🇮🇳 IN (₹)',
    currencyCode: 'INR',
    symbol: '₹',
    locale: 'en-IN',
    unlockAge: 60,
    liquidShort: 'MF / Debt / FD',
    lockedShort: 'EPF / PPF / NPS',
    monthlyLockedLabel: 'EPF / VPF / NPS',
    bucketEqLabel: 'Taxable Equity/Arb MF',
    bucketDebtLabel: 'Liquid / Debt MF / FD',
    bucketDefLabel: 'EPF / VPF Balance',
    bucketFreeLabel: 'PPF (EEE Tax-Free)',
    hasFreeBucket: true,
    hasNPSBucket: true,
    pensionLabel: 'Monthly Pension / Rental',
    pensionShort: 'NPS / Pension',
    taxOptLabel: 'Auto ₹1.25L/yr Sec 112A Tax-Free Equity LTCG Harvesting',
    taxAlphaChip: 'Sec 112A Tax-Free LTCG Harvest',
    healthcareShort: 'Floater + Med CPI',
    healthcareExplainer: 'Family Floater + ₹1Cr Super Top-up & 11%–13% Medical CPI',
    residencySummary: 'New Regime ₹12L 87A rebate + 12.5% Equity LTCG & Family Floater medical',
    defaultTier2CityId: 'in-goa',
    corpusStep: 500_000,
    savingsStep: 5_000,
    spendStep: 5_000,
    minSpend: 5_000,
    defaultMilestoneAmt: -2_500_000,
  },
  US: {
    code: 'US',
    flag: '🇺🇸',
    name: 'USA',
    shortBtn: '🇺🇸 US ($)',
    currencyCode: 'USD',
    symbol: '$',
    locale: 'en-US',
    unlockAge: 60,
    liquidShort: 'Brokerage / Cash',
    lockedShort: '401k / IRA / Roth',
    monthlyLockedLabel: '401(k) / Roth',
    bucketEqLabel: 'Taxable Brokerage',
    bucketDebtLabel: 'Cash / T-Bills',
    bucketDefLabel: 'Traditional 401(k) / IRA',
    bucketFreeLabel: 'Roth IRA + HSA',
    hasFreeBucket: true,
    hasNPSBucket: false,
    pensionLabel: 'Monthly Social Security',
    pensionShort: 'Social Security',
    taxOptLabel: 'Auto Roth Conversion Ladder (capped below 2026 400% FPL ACA Cliff)',
    taxAlphaChip: 'ACA-Safe Roth Ladder Rung',
    healthcareShort: 'ACA / Medicare',
    healthcareExplainer: '2026 Hard 400% FPL ACA Subsidy Cliff (<65) + Medicare/IRMAA (65+)',
    residencySummary: 'Federal + State taxes & ACA 400% FPL cliff / Medicare',
    defaultTier2CityId: 'us-raleigh',
    corpusStep: 25_000,
    savingsStep: 250,
    spendStep: 250,
    minSpend: 500,
    defaultMilestoneAmt: -80_000,
  },
  AU: {
    code: 'AU',
    flag: '🇦🇺',
    name: 'Australia',
    shortBtn: '🇦🇺 AU (A$)',
    currencyCode: 'AUD',
    symbol: 'A$',
    locale: 'en-AU',
    unlockAge: 60,
    liquidShort: 'ASX/Global ETF / Cash',
    lockedShort: 'Superannuation (60)',
    monthlyLockedLabel: 'Super (SG + Sal. Sac.)',
    bucketEqLabel: 'Personal ETF / ASX Brokerage',
    bucketDebtLabel: 'High-Interest Cash / Offset',
    bucketDefLabel: 'Superannuation (Tax-Free @ 60)',
    bucketFreeLabel: 'Tax-Free Wrapper',
    hasFreeBucket: false,
    hasNPSBucket: false,
    pensionLabel: 'Extra Defined-Benefit Pension',
    pensionShort: 'Age Pension (67)',
    taxOptLabel: 'ASX Franking Credit Imputation Offset + 50% Long-Term CGT Discount',
    taxAlphaChip: 'ASX Franking + Super Alpha',
    healthcareShort: 'Medicare + PHI',
    healthcareExplainer: 'Universal Medicare + Private Hospital/Extras (MLS-exempt) & Gap OOP',
    residencySummary: 'Stage 3 brackets + 50% CGT discount, Age 67 Centrelink taper & Medicare + Private Hospital',
    defaultTier2CityId: 'au-gold-coast',
    corpusStep: 25_000,
    savingsStep: 250,
    spendStep: 250,
    minSpend: 600,
    defaultMilestoneAmt: -90_000,
  },
  EU: {
    code: 'EU',
    flag: '🇪🇺',
    name: 'Europe & CH',
    shortBtn: '🇪🇺 EU/CH (€)',
    currencyCode: 'EUR',
    symbol: '€',
    locale: 'en-IE',
    unlockAge: 60,
    liquidShort: 'UCITS ETF / Bonds',
    lockedShort: 'Pillar 2 / 3a / Pension',
    monthlyLockedLabel: 'Pillar 2 / 3a / Pension',
    bucketEqLabel: 'UCITS ETF Brokerage',
    bucketDebtLabel: 'Sovereign Bonds / Cash',
    bucketDefLabel: 'Pillar 2 (BVG) / Occ. Pension',
    bucketFreeLabel: 'Pillar 3a / PEA / Wrapper',
    hasFreeBucket: true,
    hasNPSBucket: false,
    pensionLabel: 'State AHV / Pillar 1 Pension',
    pensionShort: 'AHV / State Pension',
    taxOptLabel: 'Residency Tax Alpha (IT 7% 10yr / GR 7% 15yr / PT 19.6% 8yr+ / FR PEA)',
    taxAlphaChip: 'EU Residency Tax Alpha',
    healthcareShort: 'LAMal / SNS+Priv',
    healthcareExplainer: 'Swiss LAMal Per-Capita (CH) or Public SNS/SSN/PUMA + Private Hospital Cover',
    residencySummary: 'Canton/Country CGT & wealth rules (CH 0% CGT, IT/GR 7% flat, PT 19.6%, ES/FR) + local healthcare',
    defaultTier2CityId: 'eu-algarve',
    corpusStep: 25_000,
    savingsStep: 200,
    spendStep: 200,
    minSpend: 500,
    defaultMilestoneAmt: -75_000,
  },
  NZ: {
    code: 'NZ',
    flag: '🇳🇿',
    name: 'New Zealand',
    shortBtn: '🇳🇿 NZ (NZ$)',
    currencyCode: 'NZD',
    symbol: 'NZ$',
    locale: 'en-NZ',
    unlockAge: 65,
    liquidShort: 'PIE Funds / Cash',
    lockedShort: 'KiwiSaver (Age 65)',
    monthlyLockedLabel: 'KiwiSaver (Emp+Match)',
    bucketEqLabel: 'Liquid PIE / Global Shares',
    bucketDebtLabel: 'Term Deposits / Cash PIE',
    bucketDefLabel: 'KiwiSaver (Locked to Age 65)',
    bucketFreeLabel: 'Tax-Free Wrapper',
    hasFreeBucket: false,
    hasNPSBucket: false,
    pensionLabel: 'Universal NZ Super (Age 65)',
    pensionShort: 'NZ Super (65)',
    taxOptLabel: 'Multi-Rate PIE 28% Max PIR Cap on FIF 5% FDR (+ 4-Yr Migrant FIF Holiday)',
    taxAlphaChip: 'PIE 28% PIR Cap',
    healthcareShort: 'Health NZ + Surgical',
    healthcareExplainer: 'Public Health NZ + ACC + Age-Rated Southern Cross Private Surgical Cover',
    residencySummary: '0% general CGT + FIF 5% FDR (28% PIE cap) & Health NZ + Southern Cross surgical',
    defaultTier2CityId: 'nz-tauranga-nelson',
    corpusStep: 25_000,
    savingsStep: 250,
    spendStep: 250,
    minSpend: 600,
    defaultMilestoneAmt: -85_000,
  },
};

export function currencySymbol(country: CountryCode): string {
  return COUNTRY_META[country]?.symbol || '$';
}

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

  const meta = COUNTRY_META[country] || COUNTRY_META.US;
  const sym = meta.symbol;
  if (abs >= 1e9) {
    return `${sign}${sym}${(abs / 1e9).toFixed(2)}B`;
  }
  if (abs >= 1e6) {
    const m = abs / 1e6;
    return `${sign}${sym}${m >= 100 ? m.toFixed(0) : m >= 10 ? m.toFixed(1) : m.toFixed(2)}M`;
  }
  if (abs >= 1e3) {
    const k = abs / 1e3;
    return `${sign}${sym}${k >= 100 ? k.toFixed(0) : k.toFixed(1)}K`;
  }
  return `${sign}${sym}${Math.round(abs).toLocaleString(meta.locale)}`;
}

export function formatMoneyFull(amount: number, country: CountryCode): string {
  const sign = amount < 0 ? '-' : '';
  const abs = Math.round(Math.abs(amount));
  const meta = COUNTRY_META[country] || COUNTRY_META.US;
  return `${sign}${meta.symbol}${abs.toLocaleString(meta.locale)}`;
}

export function formatSignedPct(val: number, digits = 1): string {
  const s = val > 0 ? '+' : '';
  return `${s}${val.toFixed(digits)}%`;
}

const DEFAULT_FX_PER_USD: Record<CountryCode, number> = {
  US: 1.0,
  IN: 85.0,
  AU: 1.46,
  EU: 0.94,
  NZ: 1.65,
};

export function formatFXParity(
  from: CountryCode,
  to: CountryCode,
  ratesPerUSD?: Record<CountryCode, number>,
): string {
  if (from === to) return '1:1';
  const rates = ratesPerUSD || DEFAULT_FX_PER_USD;
  const rFrom = rates[from] || 1;
  const rTo = rates[to] || 1;
  const fromMeta = COUNTRY_META[from];
  const toMeta = COUNTRY_META[to];

  // Pick direction where rate >= 1 for cleanest human readability (e.g., 1 EUR = ₹90.4 instead of 1 INR = €0.011)
  if (rTo / rFrom >= 1) {
    const mult = rTo / rFrom;
    return `1 ${fromMeta.currencyCode} = ${toMeta.symbol}${mult >= 10 ? mult.toFixed(1) : mult.toFixed(2)}`;
  }
  const inv = rFrom / rTo;
  return `1 ${toMeta.currencyCode} = ${fromMeta.symbol}${inv >= 10 ? inv.toFixed(1) : inv.toFixed(2)}`;
}
