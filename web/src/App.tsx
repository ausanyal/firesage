import React, { useCallback, useEffect, useRef, useState } from 'react';
import { CashFlowStreamChart } from './components/CashFlowStreamChart';
import { InputControlRail } from './components/InputControlRail';
import { InsightsAndStressGrid } from './components/InsightsAndStressGrid';
import { LightDismissDialog } from './components/LightDismissDialog';
import { ReadinessFrontierChart } from './components/ReadinessFrontierChart';
import { WealthConeChart } from './components/WealthConeChart';
import { type ExecutionBackend, fetchMetadata, runSimulationAnalysis } from './engine/client';
import { encodeShareableURL, planRepository } from './storage/planRepository';
import type {
  CountryCode,
  FullAnalysisResponse,
  MetadataResponse,
  PlanInput,
  SampleTrajectory,
  SavedPlanRecord,
} from './types';
import { formatMoneyCompact, formatMoneyFull } from './utils/format';

export const App: React.FC = () => {
  const [metadata, setMetadata] = useState<MetadataResponse | null>(null);
  const [input, setInput] = useState<PlanInput | null>(null);
  const [analysis, setAnalysis] = useState<FullAnalysisResponse | null>(null);
  const [backend, setBackend] = useState<ExecutionBackend>('go-local-wasm');
  const [isComputing, setIsComputing] = useState(false);
  const [valueBasis, setValueBasis] = useState<'real' | 'nominal'>('real');

  // Two-state color-scheme toggle respecting OS preference per modern-web-guidance `dark-mode`
  const [isDark, setIsDark] = useState<boolean>(() => {
    const pinned = localStorage.getItem('color-scheme');
    if (pinned === 'dark') return true;
    if (pinned === 'light') return false;
    return window.matchMedia('(prefers-color-scheme: dark)').matches;
  });

  // Modals
  const [selectedTraj, setSelectedTraj] = useState<SampleTrajectory | null>(null);
  const [sourcesOpen, setSourcesOpen] = useState(false);
  const [plansOpen, setPlansOpen] = useState(false);
  const [savedPlans, setSavedPlans] = useState<SavedPlanRecord[]>([]);
  const [saveName, setSaveName] = useState('');
  const [shareCopied, setShareCopied] = useState(false);

  const latestReqSeq = useRef(0);

  // Sync OS prefers-color-scheme listener when unpinned
  useEffect(() => {
    const mq = window.matchMedia('(prefers-color-scheme: dark)');
    const onChange = () => {
      const pinned = localStorage.getItem('color-scheme');
      if (!pinned) {
        setIsDark(mq.matches);
      }
    };
    mq.addEventListener('change', onChange);
    return () => mq.removeEventListener('change', onChange);
  }, []);

  const toggleTheme = () => {
    const nextScheme = isDark ? 'light' : 'dark';
    const osDark = window.matchMedia('(prefers-color-scheme: dark)').matches;
    const meta = document.querySelector('meta[name="color-scheme"]');

    // Two-state toggle: if user switches back to what matches OS, clear pin; else pin explicit choice
    if ((nextScheme === 'dark' && osDark) || (nextScheme === 'light' && !osDark)) {
      localStorage.removeItem('color-scheme');
      if (meta) meta.setAttribute('content', 'light dark');
      delete document.documentElement.dataset.theme;
    } else {
      localStorage.setItem('color-scheme', nextScheme);
      if (meta) meta.setAttribute('content', nextScheme);
      document.documentElement.dataset.theme = nextScheme;
    }
    setIsDark(nextScheme === 'dark');
  };

  // Bootstrap metadata + initial active draft
  useEffect(() => {
    let cancelled = false;
    (async () => {
      const { data, backend: detected } = await fetchMetadata();
      if (cancelled) return;
      setMetadata(data);
      setBackend(detected);

      const draft = planRepository.loadActiveDraft();
      const initial = draft ?? data.defaults.IN;
      setInput(initial);
      const saved = await planRepository.listPlans();
      if (!cancelled) setSavedPlans(saved);
    })();
    return () => {
      cancelled = true;
    };
  }, []);

  // Debounced reactive simulation runner
  useEffect(() => {
    if (!input) return;
    planRepository.saveActiveDraft(input);

    const seq = ++latestReqSeq.current;
    setIsComputing(true);
    const timer = setTimeout(async () => {
      try {
        const { result, backend: usedBackend } = await runSimulationAnalysis(input);
        if (seq === latestReqSeq.current) {
          setAnalysis(result);
          setBackend(usedBackend);
        }
      } finally {
        if (seq === latestReqSeq.current) {
          setIsComputing(false);
        }
      }
    }, 65);

    return () => clearTimeout(timer);
  }, [input]);

  const switchCountry = (country: CountryCode) => {
    if (!metadata || !input || input.country === country) return;
    setInput(metadata.defaults[country]);
  };

  const handleApplyLever = useCallback(
    (leverId: string) => {
      if (!input) return;
      switch (leverId) {
        case 'boost-savings-25':
          setInput({
            ...input,
            monthly_savings: Math.round(input.monthly_savings * 1.25),
            monthly_locked_savings: Math.round((input.monthly_locked_savings || 0) * 1.25),
          });
          break;
        case 'delay-retire-2yr':
          setInput({
            ...input,
            retirement_age: Math.min(input.horizon_age - 5, input.retirement_age + 2),
          });
          break;
        case 'barista-bridge-5yr':
          setInput({
            ...input,
            life_events: [
              ...(input.life_events || []),
              {
                id: `barista-${Date.now()}`,
                name: '5-Yr Consulting Bridge',
                start_age: input.retirement_age,
                end_age: input.retirement_age + 4,
                annual_real: Math.round(input.desired_monthly_spend * 12 * 0.35),
                is_taxable_in: true,
              },
            ],
          });
          break;
        case 'trim-spend-12':
          setInput({
            ...input,
            desired_monthly_spend: Math.round(input.desired_monthly_spend * 0.88),
          });
          break;
        case 'smart-guardrails-tent':
          setInput({
            ...input,
            withdrawal_strategy: 'guyton_klinger',
            guardrail_floor_pct: 0.8,
            allocation: { ...input.allocation, use_bond_tent: true },
            enable_tax_optimization: true,
          });
          break;
        case 'geo-arbitrage-tier2':
          setInput({
            ...input,
            retirement_city_id: input.country === 'US' ? 'us-raleigh' : 'in-goa',
            auto_scale_lifestyle_on_move: true,
          });
          break;
      }
    },
    [input],
  );

  const handleSaveScenario = async () => {
    if (!input || !saveName.trim()) return;
    const now = new Date().toISOString();
    const rec: SavedPlanRecord = {
      id: `plan-${Date.now()}`,
      name: saveName.trim(),
      schema_version: 1,
      created_at: now,
      updated_at: now,
      input,
    };
    await planRepository.savePlan(rec);
    setSavedPlans(await planRepository.listPlans());
    setSaveName('');
  };

  const handleExportJSON = () => {
    if (!input) return;
    const blob = new Blob([JSON.stringify({ schema_version: 1, input }, null, 2)], {
      type: 'application/json',
    });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `firesage-${input.country.toLowerCase()}-age${input.retirement_age}.json`;
    a.click();
    URL.revokeObjectURL(url);
  };

  const handleImportJSON = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;
    const reader = new FileReader();
    reader.onload = () => {
      try {
        const parsed = JSON.parse(String(reader.result));
        const loaded: PlanInput = parsed.input || parsed;
        if (loaded && (loaded.country === 'IN' || loaded.country === 'US')) {
          setInput(loaded);
          setPlansOpen(false);
        }
      } catch {
        // Ignore malformed file
      }
    };
    reader.readAsText(file);
  };

  const handleCopyShareLink = async () => {
    if (!input) return;
    const url = encodeShareableURL(input);
    await navigator.clipboard.writeText(url);
    setShareCopied(true);
    setTimeout(() => setShareCopied(false), 2200);
  };

  if (!metadata || !input || !analysis) {
    return (
      <div style={{ display: 'grid', placeItems: 'center', minHeight: '100vh', padding: '2rem' }}>
        <div style={{ textAlign: 'center' }}>
          <div className="fs-logo-mark" style={{ margin: '0 auto 0.8rem' }}>
            🔥
          </div>
          <strong>Bootstrapping FireSage Quantitative Engine&hellip;</strong>
        </div>
      </div>
    );
  }

  const country = input.country;
  const valInfo = metadata.valuations[country];
  const successTone =
    analysis.success_rate >= 90
      ? 'text-emerald'
      : analysis.success_rate >= 75
        ? 'text-amber'
        : 'text-rose';

  return (
    <div>
      {/* TOP HEADER BAR */}
      <header className="fs-topbar">
        <div className="fs-brand">
          <div className="fs-logo-mark" aria-hidden="true">
            🔥
          </div>
          <div>
            <h1 className="fs-brand-title">FireSage</h1>
            <div className="fs-brand-tag">
              Institutional Stochastic FIRE &amp; Geo-Arbitrage Simulator •{' '}
              {isComputing ? 'Simulating…' : `${analysis.compute_duration_ms.toFixed(0)}ms`}
            </div>
          </div>
        </div>

        <div className="fs-topbar-actions">
          {/* Country Toggle (2 options -> Radio per modern-web-guidance) */}
          <div className="fs-segmented" role="radiogroup" aria-label="Work portfolio currency and account jurisdiction">
            <label className={`fs-seg-option ${country === 'IN' ? 'active' : ''}`}>
              <input
                type="radio"
                name="country_switch"
                checked={country === 'IN'}
                onChange={() => switchCountry('IN')}
                className="visually-hidden"
              />
              🇮🇳 Work IN (₹ Cr/L)
            </label>
            <label className={`fs-seg-option ${country === 'US' ? 'active' : ''}`}>
              <input
                type="radio"
                name="country_switch"
                checked={country === 'US'}
                onChange={() => switchCountry('US')}
                className="visually-hidden"
              />
              🇺🇸 Work US ($M/K)
            </label>
          </div>

          <button type="button" className="fs-btn" onClick={() => setPlansOpen(true)}>
            💾 Plans ({savedPlans.length})
          </button>

          <button type="button" className="fs-btn" onClick={handleCopyShareLink}>
            {shareCopied ? '✓ Link Copied' : '🔗 Share Plan'}
          </button>

          <button type="button" className="fs-btn" onClick={() => setSourcesOpen(true)}>
            📐 Math &amp; Sources
          </button>

          <button
            type="button"
            className="fs-btn"
            onClick={toggleTheme}
            aria-label={isDark ? 'Switch to light color scheme' : 'Switch to dark color scheme'}
          >
            {isDark ? '☀️ Light' : '🌙 Dark'}
          </button>
        </div>
      </header>

      {/* MAIN TWO-COLUMN WORKSPACE */}
      <main className="fs-workspace">
        <InputControlRail input={input} cities={metadata.cities} onChange={setInput} />

        <div className="fs-main-canvas">
          {/* EXECUTIVE TELEMETRY KPI STRIP */}
          <section className="fs-kpi-grid" aria-label="Key FIRE Projection Metrics">
            <div className="fs-kpi-card">
              <div className="fs-kpi-label">Plan Survival (Age {input.horizon_age})</div>
              <div className={`fs-kpi-val ${successTone}`}>{analysis.success_rate.toFixed(1)}%</div>
              <div className="fs-kpi-sub">
                Pre-60 Liquidity Safe: <strong>{analysis.liquidity_safe_rate.toFixed(1)}%</strong>
              </div>
              <p className="fs-kpi-help">
                % of 3,000 market simulations lasting to Age {input.horizon_age}; Liquidity Safe avoids dipping into
                Age-60 funds early.
              </p>
            </div>

            <div className="fs-kpi-card">
              <div className="fs-kpi-label">90%-Confidence FIRE Age</div>
              <div className="fs-kpi-val">Age {analysis.target_fire_age.toFixed(1)}</div>
              <div className="fs-kpi-sub">
                Coast: <strong>{analysis.coast_fire_age.toFixed(1)}</strong> • Fortress:{' '}
                <strong>{analysis.fortress_fire_age.toFixed(1)}</strong>
              </div>
              <p className="fs-kpi-help">
                Earliest age to retire with 90%+ odds. Coast = stop saving now &amp; retire at 60; Fortress = 97%+ safe.
              </p>
            </div>

            <div className="fs-kpi-card">
              <div className="fs-kpi-label">Projected Corpus @ Age {input.retirement_age}</div>
              <div className="fs-kpi-val">
                {formatMoneyCompact(
                  valueBasis === 'real' ? analysis.corpus_at_fire_real_p50 : analysis.corpus_at_fire_nom_p50,
                  country,
                )}
              </div>
              <div className="fs-kpi-sub">
                Real P10–P90: {formatMoneyCompact(analysis.corpus_at_fire_real_p10, country)} –{' '}
                {formatMoneyCompact(analysis.corpus_at_fire_real_p90, country)}
              </div>
              <p className="fs-kpi-help">
                Expected {valueBasis === 'real' ? "today's value" : 'future nominal'} nest egg at FIRE (median P50,
                with 10th–90th market range).
              </p>
            </div>

            <div className="fs-kpi-card">
              <div className="fs-kpi-label">90%-Safe Corpus Needed</div>
              <div className="fs-kpi-val">{formatMoneyCompact(analysis.required_corpus_90, country)}</div>
              <div className="fs-kpi-sub">
                Implied Initial Draw: <strong>{analysis.implied_initial_swr.toFixed(2)}% SWR</strong>
              </div>
              <p className="fs-kpi-help">
                Portfolio target at Age {input.retirement_age} for 90% success, and the Year-1 withdrawal rate your
                budget implies.
              </p>
            </div>

            <div className="fs-kpi-card">
              <div className="fs-kpi-label">Sustainable 90% Spend</div>
              <div className="fs-kpi-val">{formatMoneyCompact(analysis.safe_monthly_spend_90, country)}/mo</div>
              <div className="fs-kpi-sub">
                Terminal P50 @ {input.horizon_age}:{' '}
                <strong>{formatMoneyCompact(analysis.terminal_real_p50, country)}</strong>
              </div>
              <p className="fs-kpi-help">
                Max inflation-adjusted monthly budget with 90% odds; Terminal P50 is median legacy wealth left at Age{' '}
                {input.horizon_age}.
              </p>
            </div>
          </section>

          {/* HERO CHART 1: STOCHASTIC WEALTH CONE & DRAGGABLE PLAYHEAD */}
          <WealthConeChart
            bands={analysis.yearly_bands}
            samplePaths={analysis.sample_paths}
            country={country}
            retirementAge={input.retirement_age}
            valueBasis={valueBasis}
            onChangeValueBasis={setValueBasis}
            onScrubRetirementAge={(newAge) => setInput({ ...input, retirement_age: newAge })}
            onSelectTrajectory={setSelectedTraj}
            isDark={isDark}
          />

          {/* CHARTS 2 & 3 SIDE-BY-SIDE: READINESS FRONTIER + LIFESPAN CASH-FLOW STREAM */}
          <div className="fs-dual-charts">
            <ReadinessFrontierChart
              frontier={analysis.frontier}
              country={country}
              retirementAge={input.retirement_age}
              coastAge={analysis.coast_fire_age}
              leanAge={analysis.lean_fire_age}
              targetAge={analysis.target_fire_age}
              fortressAge={analysis.fortress_fire_age}
              geoArb={analysis.geo_arbitrage}
              onSelectAge={(age) => setInput({ ...input, retirement_age: age })}
            />

            <CashFlowStreamChart
              bands={analysis.yearly_bands}
              country={country}
              retireCountry={
                metadata.cities.find((c) => c.id === input.retirement_city_id)?.country || country
              }
              retirementAge={input.retirement_age}
              liquidityWarningAge={analysis.liquidity_warning_age}
            />
          </div>

          {/* CHARTS 4, 5, 6 & 7: ACTION LEVERS, HISTORICAL CRISES, GUARDRAILS & SRR MATRIX */}
          <InsightsAndStressGrid
            country={country}
            bands={analysis.yearly_bands}
            crises={analysis.crisis_replays}
            srrHeatmap={analysis.srr_heatmap}
            levers={analysis.action_levers}
            onApplyLever={handleApplyLever}
          />

          <footer className="fs-status-footer">
            <span>
              Engine: <strong>{backend === 'go-cloud-api' ? 'Go Concurrent Server API' : 'Go Local-First WebAssembly Worker'}</strong> •{' '}
              {valInfo.metric}: <strong>{valInfo.current}x</strong> (Hist. Median {valInfo.historical_median}x)
            </span>
            <span>Dataset: {valInfo.history_span} • 100% Private Local-First Storage</span>
          </footer>
        </div>
      </main>

      {/* DIALOG 1: TRAJECTORY YEAR-BY-YEAR AUTOPSY */}
      <LightDismissDialog
        open={selectedTraj !== null}
        onClose={() => setSelectedTraj(null)}
        title={selectedTraj ? `Trajectory Autopsy — ${selectedTraj.label}` : 'Trajectory Autopsy'}
        subtitle={
          selectedTraj
            ? `${selectedTraj.survived ? '✓ Plan Survived through Horizon' : `✕ Depleted at Age ${selectedTraj.depletion_age}`} • Terminal Real Wealth: ${formatMoneyFull(selectedTraj.terminal_real, country)}`
            : undefined
        }
        maxWidth="920px"
      >
        {selectedTraj && (
          <div className="fs-table-scroll">
            <table className="fs-autopsy-table">
              <thead>
                <tr>
                  <th>Age</th>
                  <th>Sampled Hist. Yr</th>
                  <th>Equity Ret</th>
                  <th>Bond Ret</th>
                  <th>CPI</th>
                  <th>Core Spend (Real)</th>
                  <th>Healthcare</th>
                  <th>Taxes</th>
                  <th>Liquid Wealth</th>
                  <th>Total Real Wealth</th>
                </tr>
              </thead>
              <tbody>
                {selectedTraj.steps.map((st) => (
                  <tr key={st.age}>
                    <td>
                      <strong>{st.age}</strong>
                      {st.age === input.retirement_age ? ' 🔥' : ''}
                    </td>
                    <td>{st.historical_ref_year}</td>
                    <td className={st.equity_return < 0 ? 'text-rose' : 'text-emerald'}>
                      {st.equity_return > 0 ? '+' : ''}
                      {st.equity_return.toFixed(1)}%
                    </td>
                    <td>{st.debt_return.toFixed(1)}%</td>
                    <td>{st.headline_cpi.toFixed(1)}%</td>
                    <td>{formatMoneyCompact(st.real_annual_spend, country)}</td>
                    <td>{formatMoneyCompact(st.real_healthcare, country)}</td>
                    <td>{formatMoneyCompact(st.real_taxes, country)}</td>
                    <td>{formatMoneyCompact(st.liquid_real_wealth, country)}</td>
                    <td>
                      <strong>{formatMoneyCompact(st.real_wealth, country)}</strong>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </LightDismissDialog>

      {/* DIALOG 2: LOCAL-FIRST SAVED PLANS & JSON IMPORT/EXPORT */}
      <LightDismissDialog
        open={plansOpen}
        onClose={() => setPlansOpen(false)}
        title="Local-First Saved Scenarios &amp; JSON Portability"
        subtitle="Stored privately in your browser's IndexedDB — extensible to Google Sign-In + Cloud Firestore sync."
        maxWidth="680px"
      >
        <div style={{ display: 'flex', flexDirection: 'column', gap: '1rem' }}>
          <div className="fs-field-row-2">
            <input
              type="text"
              name="plan_save_name"
              aria-label="Scenario name"
              placeholder="e.g. Bengaluru → Goa at Age 45"
              value={saveName}
              onChange={(e) => setSaveName(e.target.value)}
              style={{
                fontSize: '1rem',
                padding: '0.5rem 0.7rem',
                borderRadius: '8px',
                border: '1px solid var(--border-color)',
                background: 'var(--bg-subtle)',
                color: 'var(--text-primary)',
              }}
            />
            <button type="button" className="fs-btn fs-btn-primary" onClick={handleSaveScenario}>
              Save Current Scenario
            </button>
          </div>

          <div style={{ display: 'flex', gap: '0.65rem', flexWrap: 'wrap' }}>
            <button type="button" className="fs-btn" onClick={handleExportJSON}>
              ⬇ Export JSON Blueprint
            </button>
            <label className="fs-btn" style={{ cursor: 'pointer' }}>
              ⬆ Import JSON Blueprint
              <input
                type="file"
                accept="application/json"
                onChange={handleImportJSON}
                className="visually-hidden"
              />
            </label>
          </div>

          <div>
            <strong style={{ fontSize: '0.85rem' }}>Saved Browser Scenarios ({savedPlans.length})</strong>
            {savedPlans.length === 0 ? (
              <p className="fs-hint">No saved scenarios yet. Save snapshots above to compare life plans.</p>
            ) : (
              <div style={{ display: 'flex', flexDirection: 'column', gap: '0.5rem', marginTop: '0.5rem' }}>
                {savedPlans.map((sp) => (
                  <div key={sp.id} className="fs-lever-card">
                    <div>
                      <strong>{sp.name}</strong>
                      <div className="fs-hint">
                        {sp.input.country === 'IN' ? '🇮🇳 India' : '🇺🇸 USA'} • Retire Age {sp.input.retirement_age} •
                        Corpus {formatMoneyCompact(sp.input.current_corpus, sp.input.country)}
                      </div>
                    </div>
                    <div style={{ display: 'flex', gap: '0.4rem' }}>
                      <button
                        type="button"
                        className="fs-btn fs-btn-secondary"
                        onClick={() => {
                          setInput(sp.input);
                          setPlansOpen(false);
                        }}
                      >
                        Load
                      </button>
                      <button
                        type="button"
                        className="fs-btn fs-btn-ghost"
                        onClick={async () => {
                          await planRepository.deletePlan(sp.id);
                          setSavedPlans(await planRepository.listPlans());
                        }}
                      >
                        Delete
                      </button>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>
      </LightDismissDialog>

      {/* DIALOG 3: INSTITUTIONAL METHODOLOGY & DATA SOURCES AUDIT */}
      <LightDismissDialog
        open={sourcesOpen}
        onClose={() => setSourcesOpen(false)}
        title="Quantitative Methodology, Tax Mechanics &amp; Institutional Sources"
        subtitle="How FireSage models sequence risk, inflation, tax drag, lock-ins, and healthcare cliffs"
        maxWidth="780px"
      >
        <div style={{ display: 'flex', flexDirection: 'column', gap: '0.85rem', fontSize: '0.86rem' }}>
          <div>
            <strong>1. Market &amp; Inflation Datasets</strong>
            <p className="fs-pane-note">
              <strong>USA (1928–2025):</strong> Aswath Damodaran (NYU Stern) S&amp;P 500 Total Return, 10Y US Treasury,
              3M T-Bill &amp; Gold series paired with Robert Shiller (Yale) CAPE (P/E10) and BLS Headline &amp; Medical
              Care CPI.
              <br />
              <strong>India (1980–2025):</strong> S&amp;P BSE Sensex Total Return spliced with NSE Nifty 50 TRI, RBI DBIE
              10Y GoI G-Sec &amp; 91-day T-Bill yields, INR Gold, and MoSPI CPI-Combined separated from 12%–14% Indian
              Medical Inflation.
            </p>
          </div>
          <div>
            <strong>2. Stochastic Engines &amp; Common Random Numbers (CRN)</strong>
            <p className="fs-pane-note">
              Default mode uses <strong>Politis &amp; Romano (1994) Stationary Block Bootstrap</strong> (geometric mean
              4-year blocks) to preserve fat-tail crashes, cross-asset crisis correlations, and multi-year stagflation
              waves without Gaussian IID assumptions. Goal-seek solvers apply Glasserman Common Random Numbers variance
              reduction.
            </p>
          </div>
          <div>
            <strong>3. Dual-Jurisdiction Tax, Lock-In &amp; Healthcare Cliffs (2026 Rules)</strong>
            <p className="fs-pane-note">
              <strong>India:</strong> New Regime slabs with ₹12L Section 87A rebate (properly excluding Section 112A
              equity gains), 12.5% Equity LTCG above ₹1.25L annual tax-free harvesting, slab-taxed Debt MFs, strict Age
              60 NPS 60/40 unlock, and Zone-calibrated retail Family Floater + Super Top-up medical escalation.
              <br />
              <strong>USA:</strong> Federal + State tax engines (CA 13.3%, NY, NC 4.25%, WA, CO, TX/FL 0%), 5-year
              seasoned Roth Conversion Ladder capped beneath the restored <strong>2026 hard 400% FPL ACA Subsidy Cliff</strong>,
              Age 65 Medicare + IRMAA transition, and SECURE 2.0 Age 73/75 RMDs.
            </p>
          </div>
        </div>
      </LightDismissDialog>
    </div>
  );
};
