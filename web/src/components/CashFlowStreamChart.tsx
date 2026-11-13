import React, { useState } from 'react';
import type { CountryCode, YearBand } from '../types';
import { formatMoneyCompact } from '../utils/format';

interface CashFlowStreamChartProps {
  bands: YearBand[];
  country: CountryCode;
  retireCountry?: CountryCode;
  retirementAge?: number;
  liquidityWarningAge: number;
}

export const CashFlowStreamChart: React.FC<CashFlowStreamChartProps> = ({
  bands,
  country,
  retireCountry,
  retirementAge,
  liquidityWarningAge,
}) => {
  const [viewMode, setViewMode] = useState<'outflows' | 'liquidity'>('outflows');
  const [hoverIdx, setHoverIdx] = useState<number | null>(null);
  const effRetCountry = retireCountry || country;

  if (bands.length === 0) return null;

  const w = 620;
  const h = 250;
  const pad = { top: 18, right: 16, bottom: 32, left: 64 };
  const plotW = w - pad.left - pad.right;
  const plotH = h - pad.top - pad.bottom;

  const maxOutflow = Math.max(
    1,
    ...bands.map(
      (b) =>
        b.flow_core_living_out_real +
        b.flow_healthcare_out_real +
        b.flow_taxes_out_real +
        b.flow_milestone_out_real,
    ),
  );

  const maxWealthSplit = Math.max(1, ...bands.map((b) => b.liquid_real_p50 + b.locked_real_p50));
  const maxY = (viewMode === 'outflows' ? maxOutflow : maxWealthSplit) * 1.08;

  const barW = Math.max(2, plotW / bands.length - 1.5);
  const hovered = hoverIdx !== null ? bands[hoverIdx] : bands.find((b) => b.is_retired) || bands[0];

  return (
    <section className="fs-card" aria-label="Lifespan Cash-Flow, Taxes & Liquidity Bridge">
      <div className="fs-card-header">
        <div>
          <h2 className="fs-card-title">Lifespan Cash-Flow, Tax &amp; Liquidity Bridge</h2>
          <p className="fs-card-subtitle">
            Annual real expenses, healthcare &amp; tax drag vs. liquid vs. locked retirement buckets
          </p>
        </div>
        <div className="fs-segmented" role="radiogroup" aria-label="Cash flow view mode">
          <label className={`fs-seg-option ${viewMode === 'outflows' ? 'active' : ''}`}>
            <input
              type="radio"
              name="cf_mode"
              checked={viewMode === 'outflows'}
              onChange={() => setViewMode('outflows')}
              className="visually-hidden"
            />
            Spend + Health + Tax
          </label>
          <label className={`fs-seg-option ${viewMode === 'liquidity' ? 'active' : ''}`}>
            <input
              type="radio"
              name="cf_mode"
              checked={viewMode === 'liquidity'}
              onChange={() => setViewMode('liquidity')}
              className="visually-hidden"
            />
            Liquid vs Locked
          </label>
        </div>
      </div>

      {liquidityWarningAge > 0 && (
        <div className="fs-alert-warning" role="status">
          ⚠️ <strong>Early-Retirement Liquidity Bridge Alert (Median Age {liquidityWarningAge}):</strong> A meaningful share of stress paths exhaust liquid taxable/seasoned funds before statutory unlock age (60) while wealth remains locked in {country === 'IN' ? 'EPF / PPF / NPS' : 'Traditional 401(k) / unseasoned Roth'}. Increase taxable bridge share or enable {country === 'US' ? 'Roth Conversion Ladder' : 'Arbitrage/Equity bridge'}!
        </div>
      )}

      {/* Why These 2 Modes Exist */}
      <div className="fs-mode-explainer">
        {viewMode === 'outflows' ? (
          <span>
            <strong>Why Spend + Health + Tax exists:</strong> Your monthly budget is only part of the bill. This view
            exposes hidden retirement drag from rising <strong>Healthcare</strong> (
            {effRetCountry === 'US' ? 'ACA 2026 Cliff + Medicare' : 'Family Floater + 9% Medical Inflation'}) and{' '}
            <strong>Income / Capital Gains Taxes</strong>, plus how guaranteed income offsets them.
          </span>
        ) : (
          <span>
            <strong>Why Liquid vs Locked exists (Early-Retirement Bridge Check):</strong> If you FIRE before Age 60,{' '}
            <strong className="text-emerald">Spendable Liquid Funds</strong> must carry every year until Age 60 without
            triggering penalties or lock-ins on <strong style={{ color: '#818cf8' }}>Locked Retirement Accounts</strong> (
            {country === 'IN' ? 'EPF / PPF / NPS' : '401(k) / Unseasoned Roth'}).
          </span>
        )}
      </div>

      <div className="fs-svg-wrap">
        <svg viewBox={`0 0 ${w} ${h}`} className="fs-svg-chart" role="img" aria-label="Stacked lifespan cash flows">
          {[0, 0.25, 0.5, 0.75, 1].map((f) => {
            const val = maxY * f;
            const y = pad.top + plotH - f * plotH;
            return (
              <g key={f}>
                <line
                  x1={pad.left}
                  x2={w - pad.right}
                  y1={y}
                  y2={y}
                  stroke="currentColor"
                  strokeOpacity={0.09}
                />
                <text x={pad.left - 6} y={y + 4} textAnchor="end" className="fs-svg-axis">
                  {formatMoneyCompact(val, country)}
                </text>
              </g>
            );
          })}

          {bands.map((b, idx) => {
            const x = pad.left + (idx / bands.length) * plotW;
            const segments =
              viewMode === 'outflows'
                ? [
                    { val: b.flow_core_living_out_real, color: b.is_retired ? '#10b981' : '#64748b' },
                    { val: b.flow_healthcare_out_real, color: '#0ea5e9' },
                    { val: b.flow_taxes_out_real, color: '#f59e0b' },
                    { val: b.flow_milestone_out_real, color: '#ec4899' },
                  ]
                : [
                    { val: b.liquid_real_p50, color: '#10b981' },
                    { val: b.locked_real_p50, color: '#6366f1' },
                  ];

            let yCursor = pad.top + plotH;
            const showAge = idx === 0 || idx === bands.length - 1 || b.age % 5 === 0;

            return (
              <g
                key={b.age}
                onMouseEnter={() => setHoverIdx(idx)}
                onMouseLeave={() => setHoverIdx(null)}
              >
                {segments.map((seg, sIdx) => {
                  const bh = Math.max(0, (seg.val / maxY) * plotH);
                  yCursor -= bh;
                  return (
                    <rect
                      key={sIdx}
                      x={x}
                      y={yCursor}
                      width={barW}
                      height={bh}
                      fill={seg.color}
                      opacity={hoverIdx === idx ? 1 : 0.85}
                      rx={1}
                    />
                  );
                })}
                {showAge && (
                  <text x={x + barW / 2} y={h - 8} textAnchor="middle" className="fs-svg-axis">
                    {b.age}
                  </text>
                )}
              </g>
            );
          })}

          {/* Vertical Reference Markers for FIRE Age & Age 60 Statutory Unlock */}
          {(() => {
            const fireIdx = retirementAge !== undefined ? bands.findIndex((b) => b.age === retirementAge) : -1;
            const unlockIdx = bands.findIndex((b) => b.age === 60);
            return (
              <>
                {fireIdx >= 0 && (
                  <g>
                    <line
                      x1={pad.left + (fireIdx / bands.length) * plotW}
                      x2={pad.left + (fireIdx / bands.length) * plotW}
                      y1={pad.top}
                      y2={pad.top + plotH}
                      stroke="#f59e0b"
                      strokeWidth="1.5"
                      strokeDasharray="4 3"
                    />
                    <text
                      x={pad.left + (fireIdx / bands.length) * plotW + 3}
                      y={pad.top + 10}
                      fill="#f59e0b"
                      fontSize="9.5"
                      fontWeight="700"
                    >
                      FIRE {retirementAge}
                    </text>
                  </g>
                )}
                {unlockIdx >= 0 && (
                  <g>
                    <line
                      x1={pad.left + (unlockIdx / bands.length) * plotW}
                      x2={pad.left + (unlockIdx / bands.length) * plotW}
                      y1={pad.top}
                      y2={pad.top + plotH}
                      stroke="#818cf8"
                      strokeWidth="1.5"
                      strokeDasharray="3 3"
                    />
                    <text
                      x={pad.left + (unlockIdx / bands.length) * plotW + 3}
                      y={pad.top + 10}
                      fill="#818cf8"
                      fontSize="9.5"
                      fontWeight="700"
                    >
                      Age 60 Unlock
                    </text>
                  </g>
                )}
              </>
            );
          })()}

          {/* Overlay Guaranteed Income Floor line in Outflows view */}
          {viewMode === 'outflows' && (
            <polyline
              fill="none"
              stroke="#a855f7"
              strokeWidth="2"
              points={bands
                .map((b, idx) => {
                  const x = pad.left + (idx / bands.length) * plotW + barW / 2;
                  const y = pad.top + plotH - Math.min(1, b.flow_guaranteed_in_real / maxY) * plotH;
                  return `${x.toFixed(1)},${y.toFixed(1)}`;
                })
                .join(' ')}
            />
          )}
        </svg>
      </div>

      {/* Interactive Legend + Hovered Year Readout */}
      <div className="fs-cf-footer">
        {viewMode === 'outflows' ? (
          <div className="fs-legend-row">
            <span><i style={{ background: '#10b981' }} /> Core Living: <strong>{formatMoneyCompact(hovered.flow_core_living_out_real, country)}/yr</strong></span>
            <span><i style={{ background: '#0ea5e9' }} /> Healthcare ({effRetCountry === 'US' && hovered.age < 65 ? 'ACA 2026' : effRetCountry === 'US' ? 'Medicare' : 'Floater+Med'}): <strong>{formatMoneyCompact(hovered.flow_healthcare_out_real, country)}/yr</strong></span>
            <span><i style={{ background: '#f59e0b' }} /> Taxes: <strong>{formatMoneyCompact(hovered.flow_taxes_out_real, country)}/yr</strong></span>
            <span><i style={{ background: '#a855f7' }} /> {country === 'US' ? 'Social Security' : 'NPS/Pension'}: <strong>{formatMoneyCompact(hovered.flow_guaranteed_in_real, country)}/yr</strong></span>
          </div>
        ) : (
          <div className="fs-legend-row">
            <span><i style={{ background: '#10b981' }} /> Spendable Liquid at Age {hovered.age}: <strong>{formatMoneyCompact(hovered.liquid_real_p50, country)}</strong></span>
            <span><i style={{ background: '#6366f1' }} /> Statutory Locked ({country === 'IN' ? 'EPF/PPF/NPS <60' : '401k/Unseasoned Roth <59.5'}): <strong>{formatMoneyCompact(hovered.locked_real_p50, country)}</strong></span>
            {hovered.flow_roth_or_harvest_real > 0 && (
              <span>✨ {country === 'US' ? 'ACA-Safe Roth Ladder Rung' : 'Sec 112A Tax-Free LTCG Harvest'}: <strong>{formatMoneyCompact(hovered.flow_roth_or_harvest_real, country)}</strong></span>
            )}
          </div>
        )}
      </div>
    </section>
  );
};
