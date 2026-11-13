import React, { useState } from 'react';
import type {
  CityProfile,
  LifeEvent,
  PlanInput,
  SimulationMode,
  WithdrawalStrategy,
} from '../types';
import { currencySymbol, formatMoneyCompact } from '../utils/format';
import { BufferedNumberInput } from './BufferedNumberInput';

interface InputControlRailProps {
  input: PlanInput;
  cities: CityProfile[];
  onChange: (next: PlanInput) => void;
}

export const InputControlRail: React.FC<InputControlRailProps> = ({ input, cities, onChange }) => {
  const [tier2Open, setTier2Open] = useState(false);
  const [tier3Open, setTier3Open] = useState(false);
  const [newEventName, setNewEventName] = useState('');
  const [newEventAge, setNewEventAge] = useState(input.retirement_age + 4);
  const [newEventAmt, setNewEventAmt] = useState(input.country === 'IN' ? -2_500_000 : -80_000);

  const countryCities = cities.filter((c) => c.country === input.country);
  const currentCity = countryCities.find((c) => c.id === input.current_city_id) || countryCities[0];
  const retireCity = countryCities.find((c) => c.id === input.retirement_city_id) || currentCity;

  const sym = currencySymbol(input.country);
  const corpusStep = input.country === 'IN' ? 500_000 : 25_000;
  const savingsStep = input.country === 'IN' ? 5_000 : 250;
  const spendStep = input.country === 'IN' ? 5_000 : 250;

  const update = (patch: Partial<PlanInput>) => {
    onChange({ ...input, ...patch });
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
          <legend className="fs-legend">1. Core Timeline &amp; Geo-Arbitrage</legend>

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

          {/* City Selectors (7-8 options -> <select> per modern-web-guidance `forms`) */}
          <div className="fs-field">
            <label htmlFor="fs-curr-city">Accumulation City (Working Years)</label>
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
            <label htmlFor="fs-ret-city">Retirement City (FIRE Years)</label>
            <select
              id="fs-ret-city"
              name="retirement_city_id"
              value={input.retirement_city_id}
              onChange={(e) => update({ retirement_city_id: e.target.value })}
            >
              {countryCities.map((c) => (
                <option key={c.id} value={c.id}>
                  {c.name} ({c.tier}
                  {c.country === 'US' ? ` • ${c.state_code} ${c.state_top_tax_rate_pct}% max tax` : ''})
                </option>
              ))}
            </select>
            {retireCity && <span className="fs-hint">{retireCity.tagline}</span>}
          </div>

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
          <legend className="fs-legend">2. Corpus, Savings &amp; Lifestyle</legend>

          <div className="fs-field">
            <div className="fs-label-split">
              <label htmlFor="fs-corpus">Current Investable Corpus ({sym})</label>
              <strong className="fs-val-callout">{formatMoneyCompact(input.current_corpus, input.country)}</strong>
            </div>
            <BufferedNumberInput
              id="fs-corpus"
              name="current_corpus"
              step={corpusStep}
              min={0}
              value={Math.round(input.current_corpus)}
              onCommit={(val) =>
                update({
                  current_corpus: val,
                  buckets: { ...input.buckets, use_granular: false },
                })
              }
            />
          </div>

          <div className="fs-field-row-2">
            <div className="fs-field">
              <div className="fs-label-split">
                <label htmlFor="fs-savings">Monthly SIP / Save</label>
              </div>
              <BufferedNumberInput
                id="fs-savings"
                name="monthly_savings"
                step={savingsStep}
                min={0}
                value={Math.round(input.monthly_savings)}
                onCommit={(v) => update({ monthly_savings: v })}
              />
              <span className="fs-hint">{formatMoneyCompact(input.monthly_savings, input.country)}/mo</span>
            </div>

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
              <span className="fs-hint">Career SIP growth/yr</span>
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
              min={input.country === 'IN' ? 5000 : 500}
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
                    className={`fs-preset-btn ${Math.abs(input.desired_monthly_spend - amt) < 500 ? 'active' : ''}`}
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
            <span>2B. Tax Buckets, {input.country === 'US' ? 'Social Security' : 'NPS/Pensions'} &amp; Life Events</span>
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
                <label htmlFor="fs-tax-opt">
                  {input.country === 'US'
                    ? 'Auto Roth Conversion Ladder (capped below 2026 400% FPL ACA Cliff)'
                    : 'Auto ₹1.25L/yr Sec 112A Tax-Free Equity LTCG Harvesting'}
                </label>
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
                    <label htmlFor="fs-b-eq">
                      {input.country === 'IN' ? 'Taxable Equity/Arb MF' : 'Taxable Brokerage'}
                    </label>
                    <BufferedNumberInput
                      id="fs-b-eq"
                      name="bucket_taxable_eq"
                      min={0}
                      step={corpusStep / 2}
                      value={Math.round(input.buckets.taxable_equity)}
                      onCommit={(v) =>
                        update({
                          buckets: { ...input.buckets, taxable_equity: v },
                        })
                      }
                    />
                  </div>
                  <div className="fs-field">
                    <label htmlFor="fs-b-debt">
                      {input.country === 'IN' ? 'Liquid / Debt MF / FD' : 'Cash / T-Bills'}
                    </label>
                    <BufferedNumberInput
                      id="fs-b-debt"
                      name="bucket_debt"
                      min={0}
                      step={corpusStep / 2}
                      value={Math.round(input.buckets.taxable_debt_cash)}
                      onCommit={(v) =>
                        update({
                          buckets: { ...input.buckets, taxable_debt_cash: v },
                        })
                      }
                    />
                  </div>
                  <div className="fs-field">
                    <label htmlFor="fs-b-def">
                      {input.country === 'IN' ? 'EPF / VPF Balance' : 'Traditional 401(k) / IRA'}
                    </label>
                    <BufferedNumberInput
                      id="fs-b-def"
                      name="bucket_def"
                      min={0}
                      step={corpusStep / 2}
                      value={Math.round(input.buckets.tax_deferred_locked)}
                      onCommit={(v) =>
                        update({
                          buckets: { ...input.buckets, tax_deferred_locked: v },
                        })
                      }
                    />
                  </div>
                  <div className="fs-field">
                    <label htmlFor="fs-b-free">
                      {input.country === 'IN' ? 'PPF (EEE Tax-Free)' : 'Roth IRA + HSA'}
                    </label>
                    <BufferedNumberInput
                      id="fs-b-free"
                      name="bucket_free"
                      min={0}
                      step={corpusStep / 2}
                      value={Math.round(input.buckets.tax_free_account)}
                      onCommit={(v) =>
                        update({
                          buckets: { ...input.buckets, tax_free_account: v },
                        })
                      }
                    />
                  </div>
                  {input.country === 'IN' && (
                    <div className="fs-field">
                      <label htmlFor="fs-b-nps">NPS Tier-I (Locked to 60)</label>
                      <BufferedNumberInput
                        id="fs-b-nps"
                        name="bucket_nps"
                        min={0}
                        step={corpusStep / 2}
                        value={Math.round(input.buckets.pension_locked)}
                        onCommit={(v) =>
                          update({
                            buckets: { ...input.buckets, pension_locked: v },
                          })
                        }
                      />
                    </div>
                  )}
                </div>
              )}

              <div className="fs-field-row-2">
                <div className="fs-field">
                  <label htmlFor="fs-ss-amt">
                    {input.country === 'US' ? 'Monthly Social Security' : 'Monthly Pension/Rental'}
                  </label>
                  <BufferedNumberInput
                    id="fs-ss-amt"
                    name="social_security_monthly_real"
                    step={200}
                    min={0}
                    value={Math.round(input.social_security_monthly_real)}
                    onCommit={(v) => update({ social_security_monthly_real: v })}
                  />
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
