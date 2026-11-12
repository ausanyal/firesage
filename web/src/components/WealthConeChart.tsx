import React, { useEffect, useMemo, useRef, useState } from 'react';
import type { CountryCode, SampleTrajectory, YearBand } from '../types';
import { formatMoneyCompact } from '../utils/format';

interface WealthConeChartProps {
  bands: YearBand[];
  samplePaths: SampleTrajectory[];
  country: CountryCode;
  retirementAge: number;
  valueBasis: 'real' | 'nominal';
  onChangeValueBasis: (basis: 'real' | 'nominal') => void;
  onScrubRetirementAge: (newAge: number) => void;
  onSelectTrajectory: (traj: SampleTrajectory) => void;
  isDark: boolean;
}

export const WealthConeChart: React.FC<WealthConeChartProps> = ({
  bands,
  samplePaths,
  country,
  retirementAge,
  valueBasis,
  onChangeValueBasis,
  onScrubRetirementAge,
  onSelectTrajectory,
  isDark,
}) => {
  const containerRef = useRef<HTMLDivElement | null>(null);
  const canvasRef = useRef<HTMLCanvasElement | null>(null);
  const [size, setSize] = useState({ width: 820, height: 390 });
  const [hoverIdx, setHoverIdx] = useState<number | null>(null);
  const [hoveredPathId, setHoveredPathId] = useState<number | null>(null);
  const [isDraggingFire, setIsDraggingFire] = useState(false);
  const [dragFireAge, setDragFireAge] = useState<number | null>(null);

  const activeFireAge = dragFireAge ?? retirementAge;

  useEffect(() => {
    setDragFireAge(null);
  }, [retirementAge]);

  useEffect(() => {
    const el = containerRef.current;
    if (!el) return;
    const ro = new ResizeObserver((entries) => {
      for (const entry of entries) {
        const w = Math.max(320, Math.floor(entry.contentRect.width));
        setSize({ width: w, height: w < 560 ? 320 : 390 });
      }
    });
    ro.observe(el);
    return () => ro.disconnect();
  }, []);

  const pad = useMemo(() => ({ top: 26, right: 24, bottom: 38, left: 74 }), []);

  const maxVal = useMemo(() => {
    if (bands.length === 0) return 1;
    let m = 0;
    for (const b of bands) {
      const v = valueBasis === 'real' ? b.real_p90 : b.nominal_p90;
      if (v > m) m = v;
    }
    return Math.max(1, m * 1.08);
  }, [bands, valueBasis]);

  const xForIndex = (idx: number) => {
    if (bands.length <= 1) return pad.left;
    const plotW = size.width - pad.left - pad.right;
    return pad.left + (idx / (bands.length - 1)) * plotW;
  };

  const yForValue = (v: number) => {
    const plotH = size.height - pad.top - pad.bottom;
    const clamped = Math.max(0, Math.min(maxVal, v));
    return pad.top + plotH - (clamped / maxVal) * plotH;
  };

  // Render High-DPI Canvas
  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas || bands.length === 0) return;
    const dpr = window.devicePixelRatio || 1;
    canvas.width = size.width * dpr;
    canvas.height = size.height * dpr;
    const ctx = canvas.getContext('2d');
    if (!ctx) return;
    ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
    ctx.clearRect(0, 0, size.width, size.height);

    const plotW = size.width - pad.left - pad.right;
    const plotH = size.height - pad.top - pad.bottom;

    // Subtle horizontal gridlines & Y labels
    const gridColor = isDark ? 'rgba(148, 163, 184, 0.11)' : 'rgba(15, 23, 42, 0.08)';
    const axisText = isDark ? '#94a3b8' : '#475569';
    ctx.font = '500 11px ui-sans-serif, system-ui, -apple-system, sans-serif';
    ctx.textAlign = 'right';
    ctx.textBaseline = 'middle';

    const ticks = 5;
    for (let t = 0; t <= ticks; t++) {
      const val = (maxVal * t) / ticks;
      const y = yForValue(val);
      ctx.strokeStyle = gridColor;
      ctx.lineWidth = 1;
      ctx.beginPath();
      ctx.moveTo(pad.left, y);
      ctx.lineTo(size.width - pad.right, y);
      ctx.stroke();

      ctx.fillStyle = axisText;
      ctx.fillText(formatMoneyCompact(val, country), pad.left - 10, y);
    }

    // X-axis age labels
    ctx.textAlign = 'center';
    ctx.textBaseline = 'top';
    const ageStep = bands.length > 45 ? 5 : 4;
    bands.forEach((b, idx) => {
      if (idx === 0 || idx === bands.length - 1 || b.age % ageStep === 0) {
        const x = xForIndex(idx);
        ctx.fillStyle = axisText;
        ctx.fillText(`${b.age}`, x, size.height - pad.bottom + 10);
      }
    });

    // Helper to fill polygon between upper and lower band getters
    const fillBand = (getUpper: (b: YearBand) => number, getLower: (b: YearBand) => number, fill: string) => {
      ctx.fillStyle = fill;
      ctx.beginPath();
      bands.forEach((b, i) => {
        const x = xForIndex(i);
        const y = yForValue(getUpper(b));
        if (i === 0) ctx.moveTo(x, y);
        else ctx.lineTo(x, y);
      });
      for (let i = bands.length - 1; i >= 0; i--) {
        const x = xForIndex(i);
        const y = yForValue(getLower(bands[i]));
        ctx.lineTo(x, y);
      }
      ctx.closePath();
      ctx.fill();
    };

    if (valueBasis === 'real') {
      // Outer P5–P90 cone
      fillBand(
        (b) => b.real_p90,
        (b) => b.real_p5,
        isDark ? 'rgba(16, 185, 129, 0.08)' : 'rgba(5, 150, 105, 0.09)',
      );
      // Middle P10–P75 cone
      fillBand(
        (b) => b.real_p75,
        (b) => b.real_p10,
        isDark ? 'rgba(16, 185, 129, 0.14)' : 'rgba(5, 150, 105, 0.14)',
      );
      // Inner interquartile P25–P75 cone
      fillBand(
        (b) => b.real_p75,
        (b) => b.real_p25,
        isDark ? 'rgba(20, 184, 166, 0.18)' : 'rgba(13, 148, 136, 0.17)',
      );

      // Individual sample spaghetti trajectories (Real mode)
      for (const sp of samplePaths) {
        const isHovered = sp.path_id === hoveredPathId;
        const isWorst = sp.label.includes('Worst');
        ctx.beginPath();
        sp.steps.forEach((st, idx) => {
          if (idx >= bands.length) return;
          const x = xForIndex(idx);
          const y = yForValue(st.real_wealth);
          if (idx === 0) ctx.moveTo(x, y);
          else ctx.lineTo(x, y);
        });
        if (isHovered) {
          ctx.strokeStyle = sp.survived ? '#f59e0b' : '#ef4444';
          ctx.lineWidth = 2.4;
        } else if (isWorst) {
          ctx.strokeStyle = isDark ? 'rgba(248, 113, 113, 0.58)' : 'rgba(220, 38, 38, 0.52)';
          ctx.lineWidth = 1.35;
        } else {
          ctx.strokeStyle = sp.survived
            ? isDark
              ? 'rgba(148, 163, 184, 0.15)'
              : 'rgba(71, 85, 105, 0.14)'
            : isDark
              ? 'rgba(248, 113, 113, 0.28)'
              : 'rgba(220, 38, 38, 0.25)';
          ctx.lineWidth = 1;
        }
        ctx.stroke();
      }
    } else {
      // Nominal P10–P90 cone
      fillBand(
        (b) => b.nominal_p90,
        (b) => b.nominal_p10,
        isDark ? 'rgba(56, 189, 248, 0.16)' : 'rgba(2, 132, 199, 0.15)',
      );
    }

    // Draw P10 downside guardrail dashed curve
    ctx.save();
    ctx.setLineDash([4, 4]);
    ctx.strokeStyle = isDark ? '#fb7185' : '#e11d48';
    ctx.lineWidth = 1.6;
    ctx.beginPath();
    bands.forEach((b, i) => {
      const v = valueBasis === 'real' ? b.real_p10 : b.nominal_p10;
      const x = xForIndex(i);
      const y = yForValue(v);
      if (i === 0) ctx.moveTo(x, y);
      else ctx.lineTo(x, y);
    });
    ctx.stroke();
    ctx.restore();

    // Draw P90 upside dashed curve
    ctx.save();
    ctx.setLineDash([3, 5]);
    ctx.strokeStyle = isDark ? 'rgba(52, 211, 153, 0.65)' : 'rgba(5, 150, 105, 0.65)';
    ctx.lineWidth = 1.3;
    ctx.beginPath();
    bands.forEach((b, i) => {
      const v = valueBasis === 'real' ? b.real_p90 : b.nominal_p90;
      const x = xForIndex(i);
      const y = yForValue(v);
      if (i === 0) ctx.moveTo(x, y);
      else ctx.lineTo(x, y);
    });
    ctx.stroke();
    ctx.restore();

    // Draw crisp Median P50 hero curve
    ctx.strokeStyle = valueBasis === 'real' ? (isDark ? '#34d399' : '#059669') : isDark ? '#38bdf8' : '#0284c7';
    ctx.lineWidth = 2.8;
    ctx.beginPath();
    bands.forEach((b, i) => {
      const v = valueBasis === 'real' ? b.real_p50 : b.nominal_p50;
      const x = xForIndex(i);
      const y = yForValue(v);
      if (i === 0) ctx.moveTo(x, y);
      else ctx.lineTo(x, y);
    });
    ctx.stroke();

    // Vertical Retirement Age Playhead
    const startAge = bands[0].age;
    const fireIdx = Math.max(0, Math.min(bands.length - 1, activeFireAge - startAge));
    const fireX = xForIndex(fireIdx);

    // Shade retirement zone subtly
    ctx.fillStyle = isDark ? 'rgba(245, 158, 11, 0.035)' : 'rgba(245, 158, 11, 0.045)';
    ctx.fillRect(fireX, pad.top, pad.left + plotW - fireX, plotH);

    ctx.save();
    ctx.strokeStyle = '#f59e0b';
    ctx.lineWidth = 2;
    ctx.setLineDash([6, 4]);
    ctx.beginPath();
    ctx.moveTo(fireX, pad.top);
    ctx.lineTo(fireX, size.height - pad.bottom);
    ctx.stroke();
    ctx.restore();

    // FIRE badge pill at top of playhead
    const pillLabel = `⇄ FIRE Age ${activeFireAge}`;
    ctx.font = '600 11px ui-sans-serif, system-ui, sans-serif';
    const tw = ctx.measureText(pillLabel).width + 16;
    const px = Math.max(pad.left, Math.min(size.width - pad.right - tw, fireX - tw / 2));
    ctx.fillStyle = '#f59e0b';
    ctx.beginPath();
    ctx.roundRect(px, 4, tw, 20, 10);
    ctx.fill();
    ctx.fillStyle = '#0f172a';
    ctx.textAlign = 'center';
    ctx.textBaseline = 'middle';
    ctx.fillText(pillLabel, px + tw / 2, 14);

    // Hover crosshair
    if (hoverIdx !== null && hoverIdx >= 0 && hoverIdx < bands.length) {
      const hx = xForIndex(hoverIdx);
      const hb = bands[hoverIdx];
      const medV = valueBasis === 'real' ? hb.real_p50 : hb.nominal_p50;
      const hy = yForValue(medV);

      ctx.strokeStyle = isDark ? 'rgba(226, 232, 240, 0.35)' : 'rgba(15, 23, 42, 0.25)';
      ctx.lineWidth = 1;
      ctx.beginPath();
      ctx.moveTo(hx, pad.top);
      ctx.lineTo(hx, size.height - pad.bottom);
      ctx.stroke();

      ctx.fillStyle = valueBasis === 'real' ? '#10b981' : '#0284c7';
      ctx.beginPath();
      ctx.arc(hx, hy, 5, 0, Math.PI * 2);
      ctx.fill();
      ctx.strokeStyle = isDark ? '#0f172a' : '#ffffff';
      ctx.lineWidth = 2;
      ctx.stroke();
    }
  }, [bands, samplePaths, size, maxVal, country, valueBasis, activeFireAge, hoverIdx, hoveredPathId, isDark, pad]);

  // Pointer interaction: hover crosshair, trajectory proximity hit-testing, and FIRE age scrubbing
  const ageFromClientX = (clientX: number): number => {
    const canvas = canvasRef.current;
    if (!canvas || bands.length <= 1) return retirementAge;
    const rect = canvas.getBoundingClientRect();
    const x = clientX - rect.left;
    const plotW = size.width - pad.left - pad.right;
    const frac = Math.max(0, Math.min(1, (x - pad.left) / plotW));
    const idx = Math.round(frac * (bands.length - 1));
    return bands[idx].age;
  };

  const handlePointerMove = (e: React.PointerEvent<HTMLCanvasElement>) => {
    const canvas = canvasRef.current;
    if (!canvas || bands.length <= 1) return;
    const rect = canvas.getBoundingClientRect();
    const x = e.clientX - rect.left;
    const y = e.clientY - rect.top;

    if (isDraggingFire) {
      const cand = Math.max(bands[0].age + 1, Math.min(bands[bands.length - 1].age - 3, ageFromClientX(e.clientX)));
      setDragFireAge(cand);
      return;
    }

    const plotW = size.width - pad.left - pad.right;
    if (x < pad.left - 8 || x > size.width - pad.right + 8) {
      setHoverIdx(null);
      setHoveredPathId(null);
      return;
    }

    const idx = Math.max(0, Math.min(bands.length - 1, Math.round(((x - pad.left) / plotW) * (bands.length - 1))));
    setHoverIdx(idx);

    // Find closest sample trajectory at idx (in real basis)
    if (valueBasis === 'real') {
      let bestDist = 16;
      let bestId: number | null = null;
      for (const sp of samplePaths) {
        const step = sp.steps[idx];
        if (!step) continue;
        const py = yForValue(step.real_wealth);
        const d = Math.abs(py - y);
        if (d < bestDist) {
          bestDist = d;
          bestId = sp.path_id;
        }
      }
      setHoveredPathId(bestId);
    } else {
      setHoveredPathId(null);
    }
  };

  const handlePointerDown = (e: React.PointerEvent<HTMLCanvasElement>) => {
    const canvas = canvasRef.current;
    if (!canvas || bands.length <= 1) return;
    const rect = canvas.getBoundingClientRect();
    const x = e.clientX - rect.left;
    const startAge = bands[0].age;
    const fireX = xForIndex(Math.max(0, Math.min(bands.length - 1, activeFireAge - startAge)));

    if (Math.abs(x - fireX) <= 22) {
      e.currentTarget.setPointerCapture(e.pointerId);
      setIsDraggingFire(true);
      return;
    }

    if (hoveredPathId !== null) {
      const found = samplePaths.find((s) => s.path_id === hoveredPathId);
      if (found) {
        onSelectTrajectory(found);
      }
    }
  };

  const handlePointerUp = (e: React.PointerEvent<HTMLCanvasElement>) => {
    if (isDraggingFire) {
      e.currentTarget.releasePointerCapture(e.pointerId);
      setIsDraggingFire(false);
      if (dragFireAge !== null && dragFireAge !== retirementAge) {
        onScrubRetirementAge(dragFireAge);
      }
    }
  };

  const hoveredBand = hoverIdx !== null ? bands[hoverIdx] : null;
  const hoveredTraj = hoveredPathId !== null ? samplePaths.find((p) => p.path_id === hoveredPathId) : null;

  return (
    <section className="fs-card fs-hero-chart-card" aria-label="Stochastic Wealth Trajectory Cone">
      <div className="fs-card-header">
        <div>
          <h2 className="fs-card-title">Stochastic Wealth Cone &amp; Trajectory Explorer</h2>
          <p className="fs-card-subtitle">
            Drag the amber <strong>⇄ FIRE Age</strong> line to scrub retirement timing • Click any trajectory line (or pill below) for a year-by-year autopsy
          </p>
        </div>
        <div className="fs-ToolbarGroup">
          <div className="fs-segmented" role="radiogroup" aria-label="Currency valuation basis">
            <label className={`fs-seg-option ${valueBasis === 'real' ? 'active' : ''}`}>
              <input
                type="radio"
                name="value_basis"
                value="real"
                checked={valueBasis === 'real'}
                onChange={() => onChangeValueBasis('real')}
                className="visually-hidden"
              />
              Today&apos;s {country === 'IN' ? '₹' : '$'} (Real)
            </label>
            <label className={`fs-seg-option ${valueBasis === 'nominal' ? 'active' : ''}`}>
              <input
                type="radio"
                name="value_basis"
                value="nominal"
                checked={valueBasis === 'nominal'}
                onChange={() => onChangeValueBasis('nominal')}
                className="visually-hidden"
              />
              Future {country === 'IN' ? '₹' : '$'} (Nominal)
            </label>
          </div>
        </div>
      </div>

      <div ref={containerRef} className="fs-canvas-wrap">
        <canvas
          ref={canvasRef}
          style={{
            width: `${size.width}px`,
            height: `${size.height}px`,
            cursor: isDraggingFire ? 'ew-resize' : hoveredPathId ? 'pointer' : 'crosshair',
          }}
          onPointerMove={handlePointerMove}
          onPointerDown={handlePointerDown}
          onPointerUp={handlePointerUp}
          onPointerLeave={() => {
            if (!isDraggingFire) {
              setHoverIdx(null);
              setHoveredPathId(null);
            }
          }}
        />

        {hoveredBand && (
          <div
            className="fs-chart-tooltip"
            style={{
              left: hoverIdx !== null && hoverIdx > bands.length * 0.62 ? '16px' : undefined,
              right: hoverIdx !== null && hoverIdx <= bands.length * 0.62 ? '16px' : undefined,
            }}
          >
            <div className="fs-tooltip-head">
              <strong>
                Age {hoveredBand.age} ({hoveredBand.year})
              </strong>
              <span className={`fs-badge ${hoveredBand.is_retired ? 'fs-badge-amber' : 'fs-badge-blue'}`}>
                {hoveredBand.is_retired ? 'FIRE Phase' : 'Accumulating'}
              </span>
            </div>
            <div className="fs-tooltip-grid">
              <span>P90 Upside:</span>
              <strong className="text-emerald">
                {formatMoneyCompact(valueBasis === 'real' ? hoveredBand.real_p90 : hoveredBand.nominal_p90, country)}
              </strong>
              <span>P50 Median:</span>
              <strong>
                {formatMoneyCompact(valueBasis === 'real' ? hoveredBand.real_p50 : hoveredBand.nominal_p50, country)}
              </strong>
              <span>P10 Stress:</span>
              <strong className="text-rose">
                {formatMoneyCompact(valueBasis === 'real' ? hoveredBand.real_p10 : hoveredBand.nominal_p10, country)}
              </strong>
              {valueBasis === 'real' && (
                <>
                  <span>Liquid / Locked:</span>
                  <span>
                    {formatMoneyCompact(hoveredBand.liquid_real_p50, country)} /{' '}
                    {formatMoneyCompact(hoveredBand.locked_real_p50, country)}
                  </span>
                </>
              )}
            </div>
            {hoveredTraj && (
              <div className="fs-tooltip-cta">
                🔍 Click to inspect <strong>{hoveredTraj.label}</strong>
              </div>
            )}
          </div>
        )}
      </div>

      {/* Quick-Inspect Key Path Chips */}
      <div className="fs-path-chips">
        <span className="fs-chip-caption">Inspect Trajectory Autopsy:</span>
        {samplePaths.slice(0, 5).map((sp) => (
          <button
            key={sp.path_id}
            type="button"
            className={`fs-chip ${!sp.survived ? 'fs-chip-danger' : ''}`}
            onMouseEnter={() => setHoveredPathId(sp.path_id)}
            onMouseLeave={() => setHoveredPathId(null)}
            onClick={() => onSelectTrajectory(sp)}
          >
            {sp.label} {!sp.survived ? `(Depleted Age ${sp.depletion_age})` : `(${formatMoneyCompact(sp.terminal_real, country)})`}
          </button>
        ))}
      </div>
    </section>
  );
};
