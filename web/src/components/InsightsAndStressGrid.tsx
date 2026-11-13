import React, { useState } from 'react';
import type {
  ActionLever,
  CountryCode,
  CrisisReplayOutcome,
  LabTabId,
  SRRCell,
  YearBand,
} from '../types';
import { formatMoneyCompact, formatSignedPct } from '../utils/format';

interface InsightsAndStressGridProps {
  country: CountryCode;
  bands: YearBand[];
  crises: CrisisReplayOutcome[];
  srrHeatmap: SRRCell[];
  levers: ActionLever[];
  onApplyLever: (leverId: string) => void;
  activeTab?: LabTabId;
  onTabChange?: (tab: LabTabId) => void;
}

export const InsightsAndStressGrid: React.FC<InsightsAndStressGridProps> = ({
  country,
  bands,
  crises,
  srrHeatmap,
  levers,
  onApplyLever,
  activeTab: controlledTab,
  onTabChange,
}) => {
  const [internalTab, setInternalTab] = useState<LabTabId>('levers');
  const activeTab = controlledTab ?? internalTab;
  const setActiveTab = (tab: LabTabId) => {
    setInternalTab(tab);
    onTabChange?.(tab);
  };

  const retiredBands = bands.filter((b) => b.is_retired);

  // Extract unique bucket headers for 5x4 SRR Heatmap
  const retLabels = Array.from(new Set(srrHeatmap.map((c) => c.ret_bucket_label)));
  const infLabels = Array.from(new Set(srrHeatmap.map((c) => c.inf_bucket_label)));

  return (
    <section className="fs-card">
      <div className="fs-card-header fs-wrap-header">
        <div>
          <h2 className="fs-card-title">Goal-Seek Optimizer &amp; Historical Battle-Test Lab</h2>
          <p className="fs-card-subtitle">
            Concrete 1-click improvements, real historical crisis cohorts, spending guardrails &amp; sequence-risk matrix
          </p>
        </div>

        <div className="fs-segmented" role="radiogroup" aria-label="Analytical module selector">
          {(
            [
              ['levers', '🚀 Action Levers'],
              ['crises', '🏛️ Historical Crises'],
              ['guardrails', '🛡️ Spending Guardrails'],
              ['srr', '🔥 Sequence Risk Matrix'],
            ] as const
          ).map(([id, label]) => (
            <label key={id} className={`fs-seg-option ${activeTab === id ? 'active' : ''}`}>
              <input
                type="radio"
                name="lab_tab"
                checked={activeTab === id}
                onChange={() => setActiveTab(id)}
                className="visually-hidden"
              />
              {label}
            </label>
          ))}
        </div>
      </div>

      {/* TAB 1: ACTION LEVERS */}
      {activeTab === 'levers' && (
        <div className="fs-levers-list">
          {levers.map((lv) => {
            const positive = lv.success_delta_pct > 0.05 || lv.years_saved > 0.05;
            return (
              <div key={lv.id} className="fs-lever-card">
                <div className="fs-lever-info">
                  <div className="fs-lever-title-row">
                    <h3>{lv.title}</h3>
                    <div className="fs-lever-badges">
                      <span className={`fs-badge ${positive ? 'fs-badge-emerald' : 'fs-badge-neutral'}`}>
                        {formatSignedPct(lv.success_delta_pct)} Survival → {lv.new_success_rate.toFixed(1)}%
                      </span>
                      {lv.years_saved > 0 && (
                        <span className="fs-badge fs-badge-amber">
                          ⚡ -{lv.years_saved.toFixed(1)} yrs (FIRE Age {lv.new_target_fire_age.toFixed(1)})
                        </span>
                      )}
                    </div>
                  </div>
                  <p>{lv.description}</p>
                </div>
                <button
                  type="button"
                  className="fs-btn fs-btn-secondary"
                  onClick={() => onApplyLever(lv.id)}
                >
                  Apply Lever
                </button>
              </div>
            );
          })}
        </div>
      )}

      {/* TAB 2: HISTORICAL CRISIS BATTLE-TESTS */}
      {activeTab === 'crises' && (
        <div className="fs-crisis-grid">
          {crises.map((cr) => {
            const maxS = Math.max(1, ...cr.real_wealth_series);
            const pts = cr.real_wealth_series
              .map((v, i) => {
                const x = (i / Math.max(1, cr.real_wealth_series.length - 1)) * 220;
                const y = 52 - (Math.max(0, v) / maxS) * 46;
                return `${x.toFixed(1)},${y.toFixed(1)}`;
              })
              .join(' ');

            return (
              <article key={cr.crisis.id} className="fs-crisis-card">
                <div className="fs-crisis-top">
                  <div>
                    <h3>{cr.crisis.name}</h3>
                    <span className="fs-crisis-sub">{cr.crisis.subtitle}</span>
                  </div>
                  <span className={`fs-badge ${cr.survived ? 'fs-badge-emerald' : 'fs-badge-danger'}`}>
                    {cr.survived ? '✓ Survived' : `✕ Depleted Age ${cr.depletion_age}`}
                  </span>
                </div>
                <p className="fs-crisis-desc">{cr.crisis.description}</p>
                <svg viewBox="0 0 220 56" className="fs-sparkline" aria-hidden="true">
                  <polyline
                    fill="none"
                    stroke={cr.survived ? '#10b981' : '#ef4444'}
                    strokeWidth="2.2"
                    points={pts}
                  />
                </svg>
                <div className="fs-crisis-metrics">
                  <span>First 5Y Real CAGR: <strong>{formatSignedPct(cr.worst_5yr_real_ret)}</strong></span>
                  <span>Trough (Age {cr.min_wealth_age}): <strong>{formatMoneyCompact(cr.min_real_wealth, country)}</strong></span>
                  <span>Terminal: <strong>{formatMoneyCompact(cr.terminal_real, country)}</strong></span>
                </div>
              </article>
            );
          })}
        </div>
      )}

      {/* TAB 3: DYNAMIC GUARDRAIL LIVING REALITY BANDS */}
      {activeTab === 'guardrails' && retiredBands.length > 0 && (
        <div className="fs-guardrail-pane">
          <p className="fs-pane-note">
            Instead of rigid constant withdrawals until ruin, dynamic guardrails adjust discretionary spending during market extremes. Below is your <strong>real monthly purchasing-power budget</strong> across Bull (P90), Median (P50), and Stress (P10) market paths:
          </p>
          <GuardrailBandSVG bands={retiredBands} country={country} />
        </div>
      )}

      {/* TAB 4: SEQUENCE OF RETURNS RISK (SRR) 5x4 MATRIX */}
      {activeTab === 'srr' && (
        <div className="fs-srr-pane">
          <p className="fs-pane-note">
            Portfolio survival probability segmented by <strong>First 5-Year Real Portfolio Return</strong> vs. <strong>First 5-Year Inflation Regime</strong> right after retiring:
          </p>
          <div className="fs-table-scroll">
            <table className="fs-srr-table">
              <caption className="visually-hidden">
                Sequence-of-returns survival matrix by first 5-year real portfolio return and inflation regime
              </caption>
              <thead>
                <tr>
                  <th scope="col">First 5Y Real Return \ CPI</th>
                  {infLabels.map((il) => (
                    <th key={il} scope="col">{il}</th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {retLabels.map((rl, rIdx) => (
                  <tr key={rl}>
                    <th scope="row" className="fs-srr-row-hdr">{rl}</th>
                    {infLabels.map((il, cIdx) => {
                      const cell = srrHeatmap.find(
                        (x) => x.ret_bucket_idx === rIdx && x.inf_bucket_idx === cIdx,
                      );
                      if (!cell || cell.path_count === 0) {
                        return (
                          <td key={il} className="fs-srr-empty">
                            —
                          </td>
                        );
                      }
                      const tone =
                        cell.survival_pct >= 92
                          ? 'srr-good'
                          : cell.survival_pct >= 75
                            ? 'srr-warn'
                            : 'srr-bad';
                      return (
                        <td key={il} className={`fs-srr-cell ${tone}`}>
                          <strong>{cell.survival_pct.toFixed(0)}%</strong>
                          <span>({cell.path_count} paths)</span>
                        </td>
                      );
                    })}
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}
    </section>
  );
};

const GuardrailBandSVG: React.FC<{ bands: YearBand[]; country: CountryCode }> = ({ bands, country }) => {
  const w = 680;
  const h = 220;
  const pad = { top: 18, right: 20, bottom: 30, left: 68 };
  const plotW = w - pad.left - pad.right;
  const plotH = h - pad.top - pad.bottom;

  const maxSpend = Math.max(1, ...bands.map((b) => b.monthly_spend_real_p90)) * 1.12;
  const xAt = (i: number) => (bands.length <= 1 ? pad.left : pad.left + (i / (bands.length - 1)) * plotW);
  const yAt = (v: number) => pad.top + plotH - (Math.max(0, v) / maxSpend) * plotH;

  const bandArea =
    `M ` +
    bands.map((b, i) => `${xAt(i).toFixed(1)},${yAt(b.monthly_spend_real_p90).toFixed(1)}`).join(' L ') +
    ` L ` +
    [...bands]
      .reverse()
      .map((b, revIdx) => {
        const origIdx = bands.length - 1 - revIdx;
        return `${xAt(origIdx).toFixed(1)},${yAt(b.monthly_spend_real_p10).toFixed(1)}`;
      })
      .join(' L ') +
    ` Z`;

  return (
    <div className="fs-svg-wrap">
      <svg viewBox={`0 0 ${w} ${h}`} className="fs-svg-chart" role="img" aria-label="Dynamic monthly spending bands">
        {[0, 0.33, 0.66, 1].map((f) => {
          const val = maxSpend * f;
          const y = pad.top + plotH - f * plotH;
          return (
            <g key={f}>
              <line x1={pad.left} x2={w - pad.right} y1={y} y2={y} stroke="currentColor" strokeOpacity={0.1} />
              <text x={pad.left - 8} y={y + 4} textAnchor="end" className="fs-svg-axis">
                {formatMoneyCompact(val, country)}
              </text>
            </g>
          );
        })}

        <path d={bandArea} fill="#10b981" fillOpacity="0.16" />

        <polyline
          fill="none"
          stroke="#34d399"
          strokeWidth="1.6"
          strokeDasharray="4 4"
          points={bands.map((b, i) => `${xAt(i).toFixed(1)},${yAt(b.monthly_spend_real_p90).toFixed(1)}`).join(' ')}
        />
        <polyline
          fill="none"
          stroke="#10b981"
          strokeWidth="2.6"
          points={bands.map((b, i) => `${xAt(i).toFixed(1)},${yAt(b.monthly_spend_real_p50).toFixed(1)}`).join(' ')}
        />
        <polyline
          fill="none"
          stroke="#fb7185"
          strokeWidth="1.8"
          strokeDasharray="4 4"
          points={bands.map((b, i) => `${xAt(i).toFixed(1)},${yAt(b.monthly_spend_real_p10).toFixed(1)}`).join(' ')}
        />

        {bands.map((b, i) => {
          if (i === 0 || i === bands.length - 1 || b.age % 5 === 0) {
            return (
              <text key={b.age} x={xAt(i)} y={h - 8} textAnchor="middle" className="fs-svg-axis">
                Age {b.age}
              </text>
            );
          }
          return null;
        })}
      </svg>
      <div className="fs-legend-row">
        <span><i style={{ background: '#34d399' }} /> P90 Prosperity Monthly Budget</span>
        <span><i style={{ background: '#10b981' }} /> P50 Median Monthly Budget</span>
        <span><i style={{ background: '#fb7185' }} /> P10 Bear-Market Guardrail Floor</span>
      </div>
    </div>
  );
};
