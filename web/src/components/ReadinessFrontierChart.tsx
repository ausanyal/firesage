import React, { useState } from 'react';
import type { CountryCode, FrontierPoint, GeoArbitrageSummary } from '../types';
import { formatMoneyCompact } from '../utils/format';

interface ReadinessFrontierChartProps {
  frontier: FrontierPoint[];
  country: CountryCode;
  retirementAge: number;
  coastAge: number;
  leanAge: number;
  targetAge: number;
  fortressAge: number;
  geoArb: GeoArbitrageSummary;
  onSelectAge: (age: number) => void;
}

export const ReadinessFrontierChart: React.FC<ReadinessFrontierChartProps> = ({
  frontier,
  country,
  retirementAge,
  coastAge,
  leanAge,
  targetAge,
  fortressAge,
  geoArb,
  onSelectAge,
}) => {
  const [hoverIdx, setHoverIdx] = useState<number | null>(null);

  if (frontier.length === 0) return null;

  const w = 620;
  const h = 250;
  const pad = { top: 20, right: 20, bottom: 32, left: 46 };
  const plotW = w - pad.left - pad.right;
  const plotH = h - pad.top - pad.bottom;

  const xAt = (idx: number) =>
    frontier.length <= 1 ? pad.left : pad.left + (idx / (frontier.length - 1)) * plotW;
  const yAt = (pct: number) => pad.top + plotH - (Math.max(0, Math.min(100, pct)) / 100) * plotH;

  const mainPoints = frontier.map((pt, i) => `${xAt(i).toFixed(1)},${yAt(pt.success_rate).toFixed(1)}`).join(' ');
  const stayPoints = frontier
    .map((pt, i) => `${xAt(i).toFixed(1)},${yAt(pt.stay_in_current_city_rate).toFixed(1)}`)
    .join(' ');

  const areaPath =
    `M ${xAt(0).toFixed(1)},${(pad.top + plotH).toFixed(1)} L ` +
    frontier.map((pt, i) => `${xAt(i).toFixed(1)},${yAt(pt.success_rate).toFixed(1)}`).join(' L ') +
    ` L ${xAt(frontier.length - 1).toFixed(1)},${(pad.top + plotH).toFixed(1)} Z`;

  const activeIdx = frontier.findIndex((f) => f.retirement_age === retirementAge);
  const hovered = hoverIdx !== null ? frontier[hoverIdx] : activeIdx >= 0 ? frontier[activeIdx] : frontier[0];

  return (
    <section className="fs-card" aria-label="Retirement Age Readiness Frontier">
      <div className="fs-card-header">
        <div>
          <h2 className="fs-card-title">&ldquo;When Can I Retire?&rdquo; Readiness Frontier</h2>
          <p className="fs-card-subtitle">
            Survival odds across every candidate retirement age • Click any point to jump to that age
          </p>
        </div>
        {hovered && (
          <div className="fs-mini-stat">
            <span>Age {hovered.retirement_age} Safe 90% Spend:</span>
            <strong>{formatMoneyCompact(hovered.safe_monthly_spend_90, country)}/mo</strong>
          </div>
        )}
      </div>

      {/* Four Milestone Badges */}
      <div className="fs-milestone-strip">
        <div className="fs-ms-pill">
          <span className="fs-ms-label">⛵ Coast FIRE</span>
          <strong className="fs-ms-val">Age {coastAge.toFixed(1)}</strong>
        </div>
        <div className="fs-ms-pill">
          <span className="fs-ms-label">🌱 Lean FIRE (90%)</span>
          <strong className="fs-ms-val">Age {leanAge.toFixed(1)}</strong>
        </div>
        <div className="fs-ms-pill fs-ms-highlight">
          <span className="fs-ms-label">🎯 Target FIRE (90%)</span>
          <strong className="fs-ms-val">Age {targetAge.toFixed(1)}</strong>
        </div>
        <div className="fs-ms-pill">
          <span className="fs-ms-label">🏰 Fortress FIRE (97%)</span>
          <strong className="fs-ms-val">Age {fortressAge.toFixed(1)}</strong>
        </div>
      </div>

      <div className="fs-svg-wrap">
        <svg viewBox={`0 0 ${w} ${h}`} className="fs-svg-chart" role="img" aria-label="Retirement probability by age curve">
          <defs>
            <linearGradient id="frontierGrad" x1="0" y1="0" x2="0" y2="1">
              <stop offset="0%" stopColor="#10b981" stopOpacity="0.28" />
              <stop offset="100%" stopColor="#10b981" stopOpacity="0.02" />
            </linearGradient>
          </defs>

          {[0, 25, 50, 75, 90, 100].map((pct) => (
            <g key={pct}>
              <line
                x1={pad.left}
                x2={w - pad.right}
                y1={yAt(pct)}
                y2={yAt(pct)}
                stroke={pct === 90 ? '#10b981' : 'currentColor'}
                strokeOpacity={pct === 90 ? 0.55 : 0.1}
                strokeDasharray={pct === 90 ? '4 4' : undefined}
              />
              <text x={pad.left - 8} y={yAt(pct) + 4} textAnchor="end" className="fs-svg-axis">
                {pct}%
              </text>
            </g>
          ))}

          <path d={areaPath} fill="url(#frontierGrad)" />

          {geoArb.is_relocating && (
            <polyline
              fill="none"
              stroke="#94a3b8"
              strokeWidth="1.8"
              strokeDasharray="4 4"
              points={stayPoints}
            />
          )}

          <polyline fill="none" stroke="#10b981" strokeWidth="2.6" points={mainPoints} />

          {frontier.map((pt, idx) => {
            const cx = xAt(idx);
            const cy = yAt(pt.success_rate);
            const isSelected = pt.retirement_age === retirementAge;
            const showTick = idx === 0 || idx === frontier.length - 1 || pt.retirement_age % 4 === 0;
            return (
              <g
                key={pt.retirement_age}
                style={{ cursor: 'pointer' }}
                onMouseEnter={() => setHoverIdx(idx)}
                onMouseLeave={() => setHoverIdx(null)}
                onClick={() => onSelectAge(pt.retirement_age)}
              >
                <rect x={cx - 8} y={pad.top} width={16} height={plotH} fill="transparent" />
                {isSelected && (
                  <line
                    x1={cx}
                    x2={cx}
                    y1={pad.top}
                    y2={pad.top + plotH}
                    stroke="#f59e0b"
                    strokeWidth="2"
                    strokeDasharray="4 3"
                  />
                )}
                <circle
                  cx={cx}
                  cy={cy}
                  r={isSelected ? 5.5 : hoverIdx === idx ? 4.5 : 2.2}
                  fill={isSelected ? '#f59e0b' : '#10b981'}
                />
                {showTick && (
                  <text x={cx} y={h - 8} textAnchor="middle" className="fs-svg-axis">
                    {pt.retirement_age}
                  </text>
                )}
              </g>
            );
          })}
        </svg>
      </div>

      {/* Geo-Arbitrage Callout Footer */}
      <div className="fs-geo-banner">
        <div>
          <strong>
            🌍 {geoArb.is_relocating ? 'Active Geo-Arbitrage:' : 'Geo-Arbitrage Opportunity:'}{' '}
            {geoArb.current_city.name} → {geoArb.comparison_city.name}
          </strong>
          <p>
            {geoArb.is_relocating
              ? `Relocating at retirement scales monthly spend from ${formatMoneyCompact(geoArb.stay_retire_monthly, country)} to ${formatMoneyCompact(geoArb.effective_retire_monthly, country)} (${geoArb.comparison_city.tier}) — lifting survival from ${geoArb.stay_success_rate.toFixed(1)}% to ${geoArb.move_success_rate.toFixed(1)}%.`
              : `Retiring in ${geoArb.comparison_city.name} (${geoArb.comparison_city.tier}) instead of ${geoArb.current_city.name} lifts survival from ${geoArb.stay_success_rate.toFixed(1)}% → ${geoArb.move_success_rate.toFixed(1)}% and cuts 90%-confidence corpus by ${formatMoneyCompact(Math.max(0, geoArb.corpus_saved_real), country)}.`}
          </p>
        </div>
        {geoArb.years_earlier_fire > 0 && (
          <span className="fs-badge fs-badge-emerald">
            ⚡ {geoArb.years_earlier_fire.toFixed(1)} yrs earlier FIRE
          </span>
        )}
      </div>
    </section>
  );
};
