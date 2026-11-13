import React, { useEffect, useState } from 'react';
import {
  SUPPORTED_COUNTRIES,
  type CityProfile,
  type CountryCode,
  type LifeEvent,
  type PlanInput,
  type SimulationMode,
  type WithdrawalStrategy,
} from '../types';
import {
  COUNTRY_META,
  currencySymbol,
  formatFXParity,
  formatMoneyCompact,
} from '../utils/format';
import { BufferedNumberInput } from './BufferedNumberInput';

interface InputControlRailProps {
  input: PlanInput;
  cities: CityProfile[];
  fxRates?: Record<CountryCode, number>;
  onChange: (next: PlanInput) => void;
}

function formatRetireCityOption(c: CityProfile): string {
  if (c.country === 'US') {
    return `${c.name} (${c.tier} • ${c.state_code} ${c.state_top_tax_rate_pct}% max tax)`;
  }
  if (c.country === 'EU') {
    if (c.state_code === 'CH-ZH' || c.state_code === 'CH-TI') {
      return `${c.name} (${c.tier} • 0% CGT + Wealth Tax)`;
    }
    if (c.state_code === 'IT-SOUTH' || c.state_code === 'GR') {
      return `${c.name} (${c.tier} • 7% Flat Retiree Tax)`;
    }
    if (c.state_code === 'PT') {
      return `${c.name} (${c.tier} • Portugal 19.6%–28% CGT)`;
    }
    return `${c.name} (${c.tier} • ${c.region} ${c.state_top_tax_rate_pct}% CGT)`;
  }
  if (c.country === 'AU') {
    return `${c.name} (${c.tier} • ${c.region})`;
  }
  if (c.country === 'NZ') {
    return `${c.name} (${c.tier} • ${c.region})`;
  }
  return `${c.name} (${c.tier} • ${c.region})`;
}

export const InputControlRail: React.FC<InputControlRailProps> = ({
  input,
  cities,
  fxRates,
  onChange,
}) => {
  const meta = COUNTRY_META[input.country] || COUNTRY_META.IN;
  const [tier2Open, setTier2Open] = useState(false);
  const [tier3Open, setTier3Open] = useState(false);
  const [newEventName, setNewEventName] = useState('');
  const [newEventAge, setNewEventAge] = useState(input.retirement_age + 4);
  const [newEventAmt, setNewEventAmt] = useState(meta.defaultMilestoneAmt);

  useEffect(() => {
    setNewEventAmt(COUNTRY_META[input.country]?.defaultMilestoneAmt ?? -80_000);
  }, [input.country]);

  const countryCities = cities.filter((c) => c.country === input.country);
  const currentCity = countryCities.find((c) => c.id === input.current_city_id) || countryCities[0];
  const retireCity = cities.find((c) => c.id === input.retirement_city_id) || currentCity;
  const isCrossBorder = Boolean(currentCity && retireCity && currentCity.country !== retireCity.country);
  const retireMeta = retireCity ? COUNTRY_META[retireCity.country] : meta;

  const sym = currencySymbol(input.country);
  const { corpusStep, savingsStep, spendStep, unlockAge } = meta;

  // Put active working country first in Retirement City dropdown, followed by all 4 cross-border jurisdictions
  const orderedCountryGroups = [
    input.country,
    ...SUPPORTED_COUNTRIES.filter((code) => code !== input.country),
  ].map((code) => {
    const cm = COUNTRY_META[code];
    const isHome = code === input.country;
    return {
      code,
      label: `${cm.flag} ${cm.name} (${cm.symbol} ${cm.currencyCode})${isHome ? '' : ' — Cross-Border'}`,
      list: cities.filter((c) => c.country === code),
    };
  });

  const liquidCorpus = Math.max(
    0,
    Math.round((input.buckets?.taxable_equity || 0) + (input.buckets?.taxable_debt_cash || 0)),
  );
  const lockedCorpus = Math.max(
    0,
    Math.round(
      (input.buckets?.tax_deferred_locked || 0) +
        (input.buckets?.tax_free_account || 0) +
        (input.buckets?.pension_locked || 0),
    ),
  );
  const lockedSavings = Math.max(0, Math.round(input.monthly_locked_savings || 0));

  const update = (patch: Partial<PlanInput>) => {
    onChange({ ...input, ...patch });
  };

  const updateBucketsAndCorpus = (nextBuckets: PlanInput['buckets']) => {
    const nextTotal =
      Math.max(0, nextBuckets.taxable_equity) +
      Math.max(0, nextBuckets.taxable_debt_cash) +
      Math.max(0, nextBuckets.tax_deferred_locked) +
      Math.max(0, nextBuckets.tax_free_account) +
      Math.max(0, nextBuckets.pension_locked);
    update({
      current_corpus: Math.round(nextTotal),
      buckets: { ...nextBuckets, use_granular: true },
    });
  };

  const setLiquidCorpusTotal = (nextLiq: number) => {
    const prevLiq = input.buckets.taxable_equity + input.buckets.taxable_debt_cash;
    const eqRatio =
      prevLiq > 1 ? input.buckets.taxable_equity / prevLiq : input.country === 'IN' ? 0.77 : 0.82;
    const taxable_equity = Math.round(nextLiq * eqRatio);
    const taxable_debt_cash = Math.max(0, Math.round(nextLiq - taxable_equity));
    updateBucketsAndCorpus({
      ...input.buckets,
      taxable_equity,
      taxable_debt_cash,
    });
  };

  const setLockedCorpusTotal = (nextLocked: number) => {
    const prevLocked =
      input.buckets.tax_deferred_locked + input.buckets.tax_free_account + input.buckets.pension_locked;
    if (prevLocked > 1) {
      const defShare = input.buckets.tax_deferred_locked / prevLocked;
      const freeShare = input.buckets.tax_free_account / prevLocked;
      const tax_deferred_locked = Math.round(nextLocked * defShare);
      const tax_free_account = Math.round(nextLocked * freeShare);
      const pension_locked = Math.max(0, Math.round(nextLocked - tax_deferred_locked - tax_free_account));
      updateBucketsAndCorpus({
        ...input.buckets,
        tax_deferred_locked,
        tax_free_account,
        pension_locked,
      });
    } else if (!meta.hasFreeBucket && !meta.hasNPSBucket) {
      updateBucketsAndCorpus({
        ...input.buckets,
        tax_deferred_locked: Math.round(nextLocked),
        tax_free_account: 0,
        pension_locked: 0,
      });
    } else if (!meta.hasNPSBucket) {
      const tax_deferred_locked = Math.round(nextLocked * 0.72);
      const tax_free_account = Math.max(0, Math.round(nextLocked - tax_deferred_locked));
      updateBucketsAndCorpus({
        ...input.buckets,
        tax_deferred_locked,
        tax_free_account,
        pension_locked: 0,
      });
    } else {
      const tax_deferred_locked = Math.round(nextLocked * 0.53);
      const tax_free_account = Math.round(nextLocked * 0.26);
      const pension_locked = Math.max(0, Math.round(nextLocked - tax_deferred_locked - tax_free_account));
      updateBucketsAndCorpus({
        ...input.buckets,
        tax_deferred_locked,
        tax_free_account,
        pension_locked,
      });
    }
  };

  const applyLifestylePreset = (amount: number) => {
    update({ desired_monthly_spend: amount });
  };

  const addLifeEvent = () => {
    if (!newEventName.trim()) return;
    const ev: LifeEvent = {
      id: `ev-${Date.now()}`,
      name: newEventName.trim(),
      start_age: Math.max(input.current_age, Math.min(input.horizon_age, newEventAge)),
      end_age: Math.max(input.current_age, Math.min(input.horizon_age, newEventAge)),
      annual_real: newEventAmt,
      is_taxable_in: newEventAmt > 0,
    };
    update({ life_events: [...(input.life_events || []), ev] });
    setNewEventName('');
  };

  const removeLifeEvent = (id: string) => {
    update({ life_events: (input.life_events || []).filter((e) => e.id !== id) });
  };

  return (
    <aside className="fs-rail" aria-label="Simulation Input Controls">
      <form onSubmit={(e) => e.preventDefault()} className="fs-rail-form">
        {/* TIER 1: 60-SECOND CORE INPUTS */}
        <fieldset className="fs-fieldset">
          <legend className="fs-legend">1. Core Timeline &amp; Global Geo-Arbitrage</legend>

          <div className="fs-field-row-2">
            <div className="fs-field">
              <label htmlFor="fs-curr-age">Current Age</label>
              <BufferedNumberInput
                id="fs-curr-age"
                name="current_age"
                min={18}
                max={80}
                value={input.current_age}
                onCommit={(v) => {
                  const nextRetire = Math.max(v + 1, input.retirement_age);
                  const nextHorizon = Math.max(nextRetire + 5, input.horizon_age);
                  update({
                    current_age: v,
                    retirement_age: nextRetire,
                    horizon_age: nextHorizon,
                  });
                }}
              />
            </div>
            <div className="fs-field">
              <label htmlFor="fs-horizon-age">Plan Until Age</label>
              <BufferedNumberInput
                id="fs-horizon-age"
                name="horizon_age"
                min={input.retirement_age + 5}
                max={105}
                value={input.horizon_age}
                onCommit={(v) => update({ horizon_age: v })}
              />
            </div>
          </div>

          <div className="fs-field">
            <div className="fs-label-split">
              <label htmlFor="fs-retire-age">Target FIRE Age</label>
              <strong className="fs-val-callout">
                Age {input.retirement_age} ({Math.max(1, input.retirement_age - input.current_age)} yrs)
              </strong>
            </div>
            <input
              id="fs-retire-age"
              name="retirement_age"
              type="range"
              min={input.current_age + 1}
              max={Math.min(72, input.horizon_age - 5)}
              value={input.retirement_age}
              onChange={(e) => update({ retirement_age: Number(e.target.value) })}
            />
          </div>

          {/* Household Type (2 options -> Radio per modern-web-guidance `forms`) */}
          <div className="fs-field">
            <span className="fs-group-label">Household Structure</span>
            <div className="fs-segmented fs-full-seg" role="radiogroup" aria-label="Household structure">
              <label className={`fs-seg-option ${!input.is_couple ? 'active' : ''}`}>
                <input
                  type="radio"
                  name="household_type"
                  checked={!input.is_couple}
                  onChange={() => update({ is_couple: false })}
                  className="visually-hidden"
                />
                👤 Individual
              </label>
              <label className={`fs-seg-option ${input.is_couple ? 'active' : ''}`}>
                <input
                  type="radio"
                  name="household_type"
                  checked={input.is_couple}
                  onChange={() => update({ is_couple: true })}
                  className="visually-hidden"
                />
                👥 Couple / Family
              </label>
            </div>
          </div>

          {/* City Selectors (34 global options -> <select> with <optgroup> per modern-web-guidance `forms`) */}
          <div className="fs-field">
            <label htmlFor="fs-curr-city">Accumulation City ({meta.flag} Working Years)</label>
            <select
              id="fs-curr-city"
              name="current_city_id"
              value={input.current_city_id}
              onChange={(e) => update({ current_city_id: e.target.value })}
            >
              {countryCities.map((c) => (
                <option key={c.id} value={c.id}>
                  {c.name} ({c.tier} • COL {c.cost_of_living_index.toFixed(2)}x)
                </option>
              ))}
            </select>
          </div>

          <div className="fs-field">
            <label htmlFor="fs-ret-city">Retirement City (Global {cities.length} Cities)</label>
            <select
              id="fs-ret-city"
              name="retirement_city_id"
              value={input.retirement_city_id}
              onChange={(e) => update({ retirement_city_id: e.target.value })}
            >
              {orderedCountryGroups.map((grp) => (
                <optgroup key={grp.code} label={grp.label}>
                  {grp.list.map((c) => (
                    <option key={c.id} value={c.id}>
                      {formatRetireCityOption(c)}
                    </option>
                  ))}
                </optgroup>
              ))}
            </select>
            {retireCity && <span className="fs-hint">{retireCity.tagline}</span>}
          </div>

          {isCrossBorder && currentCity && retireCity && (
            <div className="fs-crossborder-note" role="status">
              🌐 <strong>Cross-Border Move ({meta.flag} {currentCity.country} → {retireMeta.flag} {retireCity.country}):</strong>{' '}
              Portfolio stays in {sym} ({meta.lockedShort}). At Age {input.retirement_age}, expenses convert at{' '}
              <strong>{formatFXParity(currentCity.country, retireCity.country, fxRates)}</strong> under{' '}
              <strong>{retireCity.name}</strong> ({retireMeta.residencySummary}).
            </div>
          )}

          {input.current_city_id !== input.retirement_city_id && (
            <div className="fs-checkbox-row">
              <input
                id="fs-autoscale-move"
                name="auto_scale_lifestyle_on_move"
                type="checkbox"
                checked={input.auto_scale_lifestyle_on_move}
                onChange={(e) => update({ auto_scale_lifestyle_on_move: e.target.checked })}
              />
              <label htmlFor="fs-autoscale-move">
                Auto-scale lifestyle by city COL ({currentCity?.name} → {retireCity?.name})
              </label>
            </div>
          )}
        </fieldset>

        {/* CORPUS, SAVINGS & LIFESTYLE */}
        <fieldset className="fs-fieldset">
          <div className="fs-legend-split">
            <legend className="fs-legend">2. Corpus, Retirement Funds &amp; Lifestyle</legend>
            <strong className="fs-val-callout">
              Total {formatMoneyCompact(liquidCorpus + lockedCorpus, input.country)}
            </strong>
          </div>

          <div className="fs-field-row-2">
            <div className="fs-field">
              <label htmlFor="fs-liquid-corpus">Liquid Corpus ({meta.liquidShort})</label>
              <BufferedNumberInput
                id="fs-liquid-corpus"
                name="liquid_corpus"
                step={corpusStep}
                min={0}
                value={liquidCorpus}
                onCommit={setLiquidCorpusTotal}
              />
              <span className="fs-hint">
                {formatMoneyCompact(liquidCorpus, input.country)} spendable pre-{unlockAge}
              </span>
            </div>

            <div className="fs-field">
              <label htmlFor="fs-locked-corpus">
                Age-{unlockAge} Funds ({meta.lockedShort})
              </label>
              <BufferedNumberInput
                id="fs-locked-corpus"
                name="locked_corpus"
                step={corpusStep}
                min={0}
                value={lockedCorpus}
                onCommit={setLockedCorpusTotal}
              />
              <span className="fs-hint">
                {formatMoneyCompact(lockedCorpus, input.country)} unlocks Age {unlockAge}
              </span>
            </div>
          </div>

          <div className="fs-field-row-2">
            <div className="fs-field">
              <label htmlFor="fs-savings">Monthly Liquid Invest / Save</label>
              <BufferedNumberInput
                id="fs-savings"
                name="monthly_savings"
                step={savingsStep}
                min={0}
                value={Math.round(input.monthly_savings)}
                onCommit={(v) => update({ monthly_savings: v })}
              />
              <span className="fs-hint">{formatMoneyCompact(input.monthly_savings, input.country)}/mo liquid</span>
            </div>

            <div className="fs-field">
              <label htmlFor="fs-locked-savings">Monthly {meta.monthlyLockedLabel}</label>
              <BufferedNumberInput
                id="fs-locked-savings"
                name="monthly_locked_savings"
                step={savingsStep}
                min={0}
                value={lockedSavings}
                onCommit={(v) => update({ monthly_locked_savings: v })}
              />
              <span className="fs-hint">
                {formatMoneyCompact(lockedSavings, input.country)}/mo age-{unlockAge}
              </span>
            </div>
          </div>

          <div className="fs-field-row-2">
            <div className="fs-field">
              <label htmlFor="fs-stepup">Annual Step-Up %</label>
              <BufferedNumberInput
                id="fs-stepup"
                name="annual_savings_step_up"
                step={1}
                min={0}
                max={25}
                value={Math.round(input.annual_savings_step_up * 100)}
                onCommit={(v) => update({ annual_savings_step_up: v / 100 })}
              />
              <span className="fs-hint">Career save growth/yr</span>
            </div>
            <div className="fs-field">
              <span className="fs-group-label">Combined Monthly Save</span>
              <div className="fs-summary-box">
                <strong>{formatMoneyCompact(input.monthly_savings + lockedSavings, input.country)}</strong>
                <small>/mo total</small>
              </div>
              <span className="fs-hint">Fine-tune splits in 2B</span>
            </div>
          </div>

          <div className="fs-field">
            <div className="fs-label-split">
              <label htmlFor="fs-spend">Target Monthly Spend (Today&apos;s {sym})</label>
              <strong className="fs-val-callout">
                {formatMoneyCompact(input.desired_monthly_spend, input.country)}/mo
              </strong>
            </div>
            <BufferedNumberInput
              id="fs-spend"
              name="desired_monthly_spend"
              step={spendStep}
              min={meta.minSpend}
              value={Math.round(input.desired_monthly_spend)}
              onCommit={(v) => update({ desired_monthly_spend: v })}
            />
            {currentCity && (
              <div className="fs-preset-bar" aria-label="City lifestyle presets">
                {(
                  [
                    ['Lean', currentCity.monthly_presets.lean],
                    ['Standard', currentCity.monthly_presets.standard],
                    ['Chubby', currentCity.monthly_presets.chubby],
                    ['Fat', currentCity.monthly_presets.fat],
                  ] as const
                ).map(([label, amt]) => (
                  <button
                    key={label}
                    type="button"
                    className={`fs-preset-btn ${Math.abs(input.desired_monthly_spend - amt) < meta.spendStep / 2 ? 'active' : ''}`}
                    onClick={() => applyLifestylePreset(amt)}
                  >
                    <span>{label}</span>
                    <small>{formatMoneyCompact(amt, input.country)}</small>
                  </button>
                ))}
              </div>
            )}
          </div>

          <div className="fs-field">
            <div className="fs-label-split">
              <label htmlFor="fs-eq-slider">Equity Allocation</label>
              <strong className="fs-val-callout">{Math.round(input.allocation.equity * 100)}% Equity</strong>
            </div>
            <input
              id="fs-eq-slider"
              name="equity_pct"
              type="range"
              min={25}
              max={90}
              value={Math.round(input.allocation.equity * 100)}
              onChange={(e) => {
                const eq = Number(e.target.value) / 100;
                const rem = Math.max(0, 1 - eq);
                const gold = input.country === 'IN' ? Math.min(0.1, rem * 0.25) : Math.min(0.05, rem * 0.12);
                const cash = Math.min(0.06, rem * 0.18);
                const debt = Math.max(0, rem - gold - cash);
                update({
                  allocation: {
                    ...input.allocation,
                    equity: eq,
                    debt,
                    gold,
                    cash,
                  },
                });
              }}
            />
            <span className="fs-hint">
              Mix: {Math.round(input.allocation.equity * 100)}% Equity •{' '}
              {Math.round(input.allocation.debt * 100)}% Sovereign/Debt •{' '}
              {Math.round(input.allocation.gold * 100)}% Gold • {Math.round(input.allocation.cash * 100)}% Cash
            </span>
          </div>
        </fieldset>

        {/* TIER 2 PROGRESSIVE DISCLOSURE: TAX BUCKETS, INCOME FLOORS & LIFE EVENTS */}
        <div className="fs-accordion">
          <button
            type="button"
            className="fs-accordion-trigger"
            aria-expanded={tier2Open}
            aria-controls="fs-tier2-panel"
            onClick={() => setTier2Open((o) => !o)}
          >
            <span>2B. Tax Buckets, {meta.pensionShort} &amp; Life Events</span>
            <span>{tier2Open ? '▾' : '▸'}</span>
          </button>

          {tier2Open && (
            <div id="fs-tier2-panel" className="fs-accordion-body">
              <div className="fs-checkbox-row">
                <input
                  id="fs-tax-opt"
                  name="enable_tax_optimization"
                  type="checkbox"
                  checked={input.enable_tax_optimization}
                  onChange={(e) => update({ enable_tax_optimization: e.target.checked })}
                />
                <label htmlFor="fs-tax-opt">{meta.taxOptLabel}</label>
              </div>

              <div className="fs-checkbox-row">
                <input
                  id="fs-granular-buckets"
                  name="use_granular_buckets"
                  type="checkbox"
                  checked={input.buckets.use_granular}
                  onChange={(e) =>
                    update({
                      buckets: { ...input.buckets, use_granular: e.target.checked },
                    })
                  }
                />
                <label htmlFor="fs-granular-buckets">Customize Granular Account Tax Buckets</label>
              </div>

              {input.buckets.use_granular && (
                <div className="fs-subgrid">
                  <div className="fs-field">
                    <label htmlFor="fs-b-eq">{meta.bucketEqLabel}</label>
                    <BufferedNumberInput
                      id="fs-b-eq"
                      name="bucket_taxable_eq"
                      min={0}
                      step={corpusStep / 2}
                      value={Math.round(input.buckets.taxable_equity)}
                      onCommit={(v) =>
                        updateBucketsAndCorpus({ ...input.buckets, taxable_equity: v })
                      }
                    />
                  </div>
                  <div className="fs-field">
                    <label htmlFor="fs-b-debt">{meta.bucketDebtLabel}</label>
                    <BufferedNumberInput
                      id="fs-b-debt"
                      name="bucket_debt"
                      min={0}
                      step={corpusStep / 2}
                      value={Math.round(input.buckets.taxable_debt_cash)}
                      onCommit={(v) =>
                        updateBucketsAndCorpus({ ...input.buckets, taxable_debt_cash: v })
                      }
                    />
                  </div>
                  <div className="fs-field">
                    <label htmlFor="fs-b-def">{meta.bucketDefLabel}</label>
                    <BufferedNumberInput
                      id="fs-b-def"
                      name="bucket_def"
                      min={0}
                      step={corpusStep / 2}
                      value={Math.round(input.buckets.tax_deferred_locked)}
                      onCommit={(v) =>
                        updateBucketsAndCorpus({ ...input.buckets, tax_deferred_locked: v })
                      }
                    />
                  </div>
                  {meta.hasFreeBucket && (
                    <div className="fs-field">
                      <label htmlFor="fs-b-free">{meta.bucketFreeLabel}</label>
                      <BufferedNumberInput
                        id="fs-b-free"
                        name="bucket_free"
                        min={0}
                        step={corpusStep / 2}
                        value={Math.round(input.buckets.tax_free_account)}
                        onCommit={(v) =>
                          updateBucketsAndCorpus({ ...input.buckets, tax_free_account: v })
                        }
                      />
                    </div>
                  )}
                  {meta.hasNPSBucket && (
                    <div className="fs-field">
                      <label htmlFor="fs-b-nps">NPS Tier-I (Locked to 60)</label>
                      <BufferedNumberInput
                        id="fs-b-nps"
                        name="bucket_nps"
                        min={0}
                        step={corpusStep / 2}
                        value={Math.round(input.buckets.pension_locked)}
                        onCommit={(v) =>
                          updateBucketsAndCorpus({ ...input.buckets, pension_locked: v })
                        }
                      />
                    </div>
                  )}
                </div>
              )}

              <div className="fs-field-row-2">
                <div className="fs-field">
                  <label htmlFor="fs-ss-amt">{meta.pensionLabel}</label>
                  <BufferedNumberInput
                    id="fs-ss-amt"
                    name="social_security_monthly_real"
                    step={200}
                    min={0}
                    value={Math.round(input.social_security_monthly_real)}
                    onCommit={(v) => update({ social_security_monthly_real: v })}
                  />
                  {input.country === 'AU' && (
                    <span className="fs-hint">Centrelink Age Pension @ 67 is auto-modeled via Assets Test!</span>
                  )}
                </div>
                <div className="fs-field">
                  <label htmlFor="fs-ss-age">Start Age</label>
                  <BufferedNumberInput
                    id="fs-ss-age"
                    name="social_security_start_age"
                    min={50}
                    max={75}
                    value={input.social_security_start_age}
                    onCommit={(v) => update({ social_security_start_age: v })}
                  />
                </div>
              </div>

              {/* Custom Life Events */}
              <div className="fs-events-box">
                <span className="fs-group-label">Custom Life Milestones (+Inflow / -Outflow)</span>
                {(input.life_events || []).map((ev) => (
                  <div key={ev.id} className="fs-event-item">
                    <span>
                      <strong>Age {ev.start_age}:</strong> {ev.name} (
                      {formatMoneyCompact(ev.annual_real, input.country)})
                    </span>
                    <button
                      type="button"
                      className="fs-btn-icon"
                      onClick={() => removeLifeEvent(ev.id)}
                      aria-label={`Remove ${ev.name}`}
                    >
                      ✕
                    </button>
                  </div>
                ))}
                <div className="fs-event-adder">
                  <input
                    type="text"
                    name="new_event_name"
                    aria-label="Milestone name"
                    placeholder="e.g. Kid's College / Downsize"
                    value={newEventName}
                    onChange={(e) => setNewEventName(e.target.value)}
                  />
                  <div className="fs-field-row-2">
                    <BufferedNumberInput
                      name="new_event_age"
                      ariaLabel="Milestone age"
                      min={input.current_age}
                      max={input.horizon_age}
                      value={newEventAge}
                      onCommit={(v) => setNewEventAge(v)}
                    />
                    <BufferedNumberInput
                      name="new_event_amount"
                      ariaLabel="Signed real amount"
                      step={corpusStep / 2}
                      value={newEventAmt}
                      onCommit={(v) => setNewEventAmt(v)}
                    />
                  </div>
                  <button type="button" className="fs-btn fs-btn-ghost fs-full-btn" onClick={addLifeEvent}>
                    + Add Milestone Event
                  </button>
                </div>
              </div>
            </div>
          )}
        </div>

        {/* TIER 3 PROGRESSIVE DISCLOSURE: QUANT ENGINE, GUARDRAILS & STRESS LAB */}
        <div className="fs-accordion">
          <button
            type="button"
            className="fs-accordion-trigger"
            aria-expanded={tier3Open}
            aria-controls="fs-tier3-panel"
            onClick={() => setTier3Open((o) => !o)}
          >
            <span>3. Quant Engine, Guardrails &amp; Black-Swan Lab</span>
            <span>{tier3Open ? '▾' : '▸'}</span>
          </button>

          {tier3Open && (
            <div id="fs-tier3-panel" className="fs-accordion-body">
              <div className="fs-field">
                <span className="fs-group-label">Stochastic Return Engine</span>
                <div className="fs-radio-stack" role="radiogroup" aria-label="Simulation engine mode">
                  {(
                    [
                      ['bootstrap', 'Stationary Block Bootstrap (Politis-Romano)'],
                      ['regime_mc', 'Valuation/CAPE Markov Regime-Switching MC'],
                      ['historical', 'Pure Historical Rolling Cohort Replay'],
                      ['stress_test', 'Custom Black-Swan Stress Shock Mode'],
                    ] as [SimulationMode, string][]
                  ).map(([val, label]) => (
                    <label key={val} className="fs-radio-line">
                      <input
                        type="radio"
                        name="sim_mode"
                        checked={input.mode === val}
                        onChange={() => update({ mode: val })}
                      />
                      <span>{label}</span>
                    </label>
                  ))}
                </div>
              </div>

              <div className="fs-field">
                <span className="fs-group-label">Retirement Withdrawal Strategy</span>
                <div className="fs-radio-stack" role="radiogroup" aria-label="Withdrawal guardrail strategy">
                  {(
                    [
                      ['guyton_klinger', 'Guyton-Klinger Dynamic Guardrails'],
                      ['cape_dynamic', 'ERN Valuation / CAPE-Conditioned Dynamic'],
                      ['constant_real', 'Classic Constant Real Spending (4%-Style)'],
                    ] as [WithdrawalStrategy, string][]
                  ).map(([val, label]) => (
                    <label key={val} className="fs-radio-line">
                      <input
                        type="radio"
                        name="withdrawal_strategy"
                        checked={input.withdrawal_strategy === val}
                        onChange={() => update({ withdrawal_strategy: val })}
                      />
                      <span>{label}</span>
                    </label>
                  ))}
                </div>
              </div>

              <div className="fs-checkbox-row">
                <input
                  id="fs-bond-tent"
                  name="use_bond_tent"
                  type="checkbox"
                  checked={input.allocation.use_bond_tent}
                  onChange={(e) =>
                    update({
                      allocation: { ...input.allocation, use_bond_tent: e.target.checked },
                    })
                  }
                />
                <label htmlFor="fs-bond-tent">
                  ERN Rising-Equity Bond Tent (+18% defensive buffer around FIRE date)
                </label>
              </div>

              <div className="fs-checkbox-row">
                <input
                  id="fs-smile"
                  name="use_spending_smile"
                  type="checkbox"
                  checked={input.use_spending_smile}
                  onChange={(e) => update({ use_spending_smile: e.target.checked })}
                />
                <label htmlFor="fs-smile">Blanchett Empirical &ldquo;Spending Smile&rdquo; Age Curve</label>
              </div>

              {(input.mode === 'stress_test' || input.stress.enable_custom_shock) && (
                <div className="fs-stress-box">
                  <div className="fs-field">
                    <label htmlFor="fs-yr1-crash">
                      Year-1 Retirement Crash: {Math.round(input.stress.retire_year1_crash_pct * 100)}%
                    </label>
                    <input
                      id="fs-yr1-crash"
                      name="retire_year1_crash_pct"
                      type="range"
                      min={-55}
                      max={0}
                      value={Math.round(input.stress.retire_year1_crash_pct * 100)}
                      onChange={(e) =>
                        update({
                          stress: {
                            ...input.stress,
                            enable_custom_shock: true,
                            retire_year1_crash_pct: Number(e.target.value) / 100,
                          },
                        })
                      }
                    />
                  </div>
                  <div className="fs-field">
                    <label htmlFor="fs-stag-boost">
                      5-Year Stagflation Spike: +{Math.round(input.stress.extra_inflation_pct * 100)}%/yr CPI
                    </label>
                    <input
                      id="fs-stag-boost"
                      name="extra_inflation_pct"
                      type="range"
                      min={0}
                      max={8}
                      value={Math.round(input.stress.extra_inflation_pct * 100)}
                      onChange={(e) =>
                        update({
                          stress: {
                            ...input.stress,
                            enable_custom_shock: true,
                            extra_inflation_pct: Number(e.target.value) / 100,
                            extra_inflation_years: 5,
                          },
                        })
                      }
                    />
                  </div>
                </div>
              )}
            </div>
          )}
        </div>
      </form>
    </aside>
  );
};
