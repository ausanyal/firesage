import React, { useCallback, useEffect, useId, useMemo, useRef, useState } from 'react';
import { runCopilotChat } from '../engine/client';
import type {
  ChatTurn,
  CityProfile,
  CopilotAction,
  CopilotMessage,
  FullAnalysisResponse,
  PersonaMode,
  PlanInput,
  UICommand,
} from '../types';
import { COUNTRY_META, formatMoneyCompact } from '../utils/format';
import { CopilotMarkdown } from './CopilotMarkdown';
import { LightDismissDialog } from './LightDismissDialog';

interface FireCopilotDrawerProps {
  open: boolean;
  onClose: () => void;
  input: PlanInput;
  analysis: FullAnalysisResponse;
  cities: CityProfile[];
  valueBasis: 'real' | 'nominal';
  onExecuteUICommands: (commands: UICommand[]) => string[];
  externalPromptRequest?: { id: number; prompt: string; mode?: PersonaMode } | null;
}

const STORAGE_KEY_API = 'firesage_gemini_api_key';
const STORAGE_KEY_MODEL = 'firesage_gemini_model';
const STORAGE_KEY_WIDE = 'firesage_copilot_wide';

const SKILL_BADGE_NAMES: Record<string, string> = {
  skill_page_driver: '🧭 UI Page-Driver',
  skill_monte_carlo_srr: '🎲 Block-Bootstrap & SRR',
  skill_liquidity_tax_bridge: '🏛️ Tax & Age-Unlock Bridge',
  skill_geo_arbitrage_scanner: '🌍 34-City Geo-Arbitrage',
  skill_goal_seek_solver: '🧮 CRN Goal-Seek Solver',
  skill_crisis_stress_tester: '⚡ Crisis & Stagflation Lab',
  skill_life_events_architect: '📅 Life-Events Modeler',
};

const PERSONA_OPTIONS: Array<{ id: PersonaMode; label: string; hint: string }> = [
  { id: 'analyst', label: '🧠 Deep Planner', hint: 'Institutional actuarial & tax synthesis' },
  { id: 'eli5', label: '👶 ELI5', hint: 'Plain-English everyday analogies' },
  { id: 'debate', label: '⚔️ Bull vs. Bear', hint: 'Committee bull case vs. bear case' },
  { id: 'devil', label: "🦈 Devil's Advocate", hint: 'Chief Risk Officer pre-mortem stress audit' },
];

export const FireCopilotDrawer: React.FC<FireCopilotDrawerProps> = ({
  open,
  onClose,
  input,
  analysis,
  cities,
  valueBasis,
  onExecuteUICommands,
  externalPromptRequest,
}) => {
  const drawerTitleId = useId();
  const promptInputId = useId();
  const apiKeyInputId = useId();
  const modelSelectId = useId();

  const [isWide, setIsWide] = useState<boolean>(() => localStorage.getItem(STORAGE_KEY_WIDE) === '1');
  const [mode, setMode] = useState<PersonaMode>('analyst');
  const [draftText, setDraftText] = useState('');
  const [isSending, setIsSending] = useState(false);
  const [settingsOpen, setSettingsOpen] = useState(false);
  const [apiKey, setApiKey] = useState<string>(() => localStorage.getItem(STORAGE_KEY_API) || '');
  const [selectedModel, setSelectedModel] = useState<string>(
    () => localStorage.getItem(STORAGE_KEY_MODEL) || 'gemini-2.5-flash',
  );
  const [politeAnnouncement, setPoliteAnnouncement] = useState('');
  const [lastUsedLLM, setLastUsedLLM] = useState(false);

  const curCity = useMemo(
    () => cities.find((c) => c.id === input.current_city_id) || analysis.geo_arbitrage.current_city,
    [cities, input.current_city_id, analysis.geo_arbitrage.current_city],
  );
  const retCity = useMemo(
    () => cities.find((c) => c.id === input.retirement_city_id) || analysis.geo_arbitrage.comparison_city,
    [cities, input.retirement_city_id, analysis.geo_arbitrage.comparison_city],
  );

  const buildWelcomeMessage = useCallback((): CopilotMessage => {
    const meta = COUNTRY_META[input.country];
    const surv = analysis.success_rate.toFixed(1);
    const req90 = formatMoneyCompact(analysis.required_corpus_90, input.country);
    const p50 = formatMoneyCompact(analysis.corpus_at_fire_real_p50, input.country);
    return {
      id: 'welcome',
      role: 'assistant',
      timestamp: 'Ready',
      engineLabel: '💬 Conversational FIRE Advisor + Go Quant Tools',
      text: [
        `### 👋 Hi! I'm your FIRE Copilot (${meta.flag} ${curCity.name} → ${retCity.name})`,
        '',
        `Ask me **anything in plain English** — whether it's a general FIRE concept (*"What's Coast vs. Barista FIRE?"*, *"Why not just use 35x?"*, *"Should I rent or buy?"*) or a deep dive into your own numbers:`,
        `- **Your Live Snapshot:** **${surv}% survival** retiring at Age ${input.retirement_age} • Projected median corpus **${p50}** vs. 90%-safe target **${req90}** (\`${analysis.implied_initial_swr.toFixed(2)}% SWR\`).`,
        `- **Simulate & Drive the App:** Ask me to **compare any of the 34 cities**, **solve your exact monthly SIP**, **stress-test market crashes + pre-Age-${meta.unlockAge} liquidity**, or **update sliders directly** (*"Set retirement age to 48 and move to Goa"*).`,
      ].join('\n'),
    };
  }, [
    input.country,
    input.retirement_age,
    analysis.success_rate,
    analysis.required_corpus_90,
    analysis.corpus_at_fire_real_p50,
    analysis.implied_initial_swr,
    curCity.name,
    retCity.name,
  ]);

  const [messages, setMessages] = useState<CopilotMessage[]>(() => [buildWelcomeMessage()]);
  const messagesEndRef = useRef<HTMLDivElement | null>(null);
  const textareaRef = useRef<HTMLTextAreaElement | null>(null);
  const handledExternalIdRef = useRef<number>(0);

  // Refresh initial welcome card when user switches country before chatting
  useEffect(() => {
    setMessages((prev) => {
      if (prev.length === 1 && prev[0].id === 'welcome') {
        return [buildWelcomeMessage()];
      }
      return prev;
    });
  }, [buildWelcomeMessage]);

  useEffect(() => {
    if (open) {
      messagesEndRef.current?.scrollIntoView({ behavior: 'smooth', block: 'end' });
    }
  }, [messages, isSending, open]);

  // Close drawer on Escape when settings modal is not open
  useEffect(() => {
    if (!open || settingsOpen) return;
    const onKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        onClose();
      }
    };
    window.addEventListener('keydown', onKeyDown);
    return () => window.removeEventListener('keydown', onKeyDown);
  }, [open, settingsOpen, onClose]);

  const quickChips = useMemo(() => {
    const comparePrompt = (() => {
      switch (input.country) {
        case 'IN':
          return `Compare retiring in ${curCity.name} vs Kolkata vs Goa vs Dehradun vs Kochi`;
        case 'US':
          return `Compare retiring in ${curCity.name} vs Raleigh vs Tampa vs Algarve vs Southern Italy`;
        case 'AU':
          return `Compare retiring in ${curCity.name} vs Melbourne vs Gold Coast vs Adelaide vs Tauranga`;
        case 'EU':
          return `Compare retiring in Zurich vs Lugano vs Algarve vs Southern Italy vs Crete`;
        case 'NZ':
          return `Compare retiring in ${curCity.name} vs Tauranga vs Wellington vs Gold Coast`;
      }
    })();
    const unlockAge = COUNTRY_META[input.country].unlockAge;

    return [
      {
        label: '🎯 Audit My Plan',
        prompt: `Audit my ${curCity.name} to ${retCity.name} retirement plan and rank my highest-impact action levers.`,
      },
      {
        label: '🌍 Compare Top Cities',
        prompt: comparePrompt,
      },
      {
        label: '🧮 Solve 90% SIP',
        prompt: `How much monthly liquid SIP do I need to hit 90% survival at Age ${input.retirement_age}?`,
      },
      {
        label: '⚡ -35% Crash + CPI Test',
        prompt: `Stress test my plan against a -35% Year-1 market crash, +3.5% inflation spike, and medical emergency.`,
      },
      {
        label: `🌉 Pre-${unlockAge} Liquidity`,
        prompt: `Audit my pre-Age-${unlockAge} liquid vs locked bridge and check if I risk an early-withdrawal crunch.`,
      },
      {
        label: '🏛️ Tax & Medical Audit',
        prompt: `Break down my lifetime retirement taxes, statutory lock-in rules, and healthcare inflation in ${retCity.name}.`,
      },
      {
        label: '📅 5-Yr Consulting Bridge',
        prompt: `What if I add a 5-year part-time consulting bridge right after retiring at Age ${input.retirement_age}?`,
      },
      {
        label: '📐 Why Not 35x?',
        prompt: `Why does FireSage compute a dynamic corpus target instead of a flat 35x annual expenses multiplier?`,
      },
    ];
  }, [input.country, input.retirement_age, curCity.name, retCity.name]);

  const sendPrompt = useCallback(
    async (questionText: string, overrideMode?: PersonaMode) => {
      const q = questionText.trim();
      if (!q || isSending) return;

      const activeMode = overrideMode ?? mode;
      if (overrideMode && overrideMode !== mode) {
        setMode(overrideMode);
      }

      const nowStr = new Date().toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
      const userMsg: CopilotMessage = {
        id: `u-${Date.now()}`,
        role: 'user',
        text: q,
        timestamp: nowStr,
      };

      const historyPayload: ChatTurn[] = messages
        .filter((m) => m.id !== 'welcome')
        .slice(-6)
        .map((m) => ({ role: m.role, text: m.text }));

      setMessages((prev) => [...prev, userMsg]);
      setDraftText('');
      setIsSending(true);

      try {
        const { result } = await runCopilotChat({
          question: q,
          mode: activeMode,
          input,
          value_basis: valueBasis,
          history: historyPayload,
          api_key: apiKey.trim() || undefined,
          model: selectedModel,
        });

        const executedSteps =
          result.ui_commands && result.ui_commands.length > 0
            ? onExecuteUICommands(result.ui_commands)
            : [];

        setLastUsedLLM(Boolean(result.used_llm));
        const botMsg: CopilotMessage = {
          id: `b-${Date.now()}`,
          role: 'assistant',
          text: result.markdown,
          timestamp: new Date().toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }),
          engineLabel: result.engine_label,
          usedLLM: result.used_llm,
          llmNote: result.llm_note,
          plan: result.plan,
          toolTrace: result.tool_trace,
          executedUISteps: executedSteps,
          actions: result.actions,
        };

        setMessages((prev) => [...prev, botMsg]);
        setPoliteAnnouncement(
          `FIRE Copilot finished analysis using ${result.tool_trace?.length || 1} quantitative tools.`,
        );
      } catch (err) {
        setMessages((prev) => [
          ...prev,
          {
            id: `err-${Date.now()}`,
            role: 'assistant',
            timestamp: nowStr,
            engineLabel: '⚠️ Copilot Error',
            text: `Unable to complete quantitative tool run: \`${err instanceof Error ? err.message : String(err)}\`. Please try again.`,
          },
        ]);
      } finally {
        setIsSending(false);
      }
    },
    [isSending, mode, messages, input, valueBasis, apiKey, selectedModel, onExecuteUICommands],
  );

  // Trigger external deep-dive prompt requests from dashboard buttons
  useEffect(() => {
    if (
      externalPromptRequest &&
      externalPromptRequest.id !== handledExternalIdRef.current &&
      open &&
      !isSending
    ) {
      handledExternalIdRef.current = externalPromptRequest.id;
      void sendPrompt(externalPromptRequest.prompt, externalPromptRequest.mode);
    }
  }, [externalPromptRequest, open, isSending, sendPrompt]);

  const handleActionClick = (act: CopilotAction) => {
    if (act.ui_command) {
      const steps = onExecuteUICommands([act.ui_command]);
      if (steps.length > 0) {
        setMessages((prev) => [
          ...prev,
          {
            id: `sys-${Date.now()}`,
            role: 'assistant',
            timestamp: new Date().toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }),
            engineLabel: '🧭 Page-Driver Action Applied',
            executedUISteps: steps,
            text: `Applied **${act.label}** directly to your live FireSage simulation. All charts and survival percentiles above have updated automatically.`,
          },
        ]);
      }
      return;
    }
    if (act.prompt) {
      void sendPrompt(act.prompt);
    }
  };

  const toggleWide = () => {
    const next = !isWide;
    setIsWide(next);
    localStorage.setItem(STORAGE_KEY_WIDE, next ? '1' : '0');
  };

  const handleClearChat = () => {
    setMessages([buildWelcomeMessage()]);
  };

  const handleSaveSettings = (e: React.FormEvent) => {
    e.preventDefault();
    const trimmed = apiKey.trim();
    if (trimmed) {
      localStorage.setItem(STORAGE_KEY_API, trimmed);
    } else {
      localStorage.removeItem(STORAGE_KEY_API);
    }
    localStorage.setItem(STORAGE_KEY_MODEL, selectedModel);
    setSettingsOpen(false);
  };

  const exportRetirementMemoMarkdown = () => {
    const meta = COUNTRY_META[input.country];
    const dateISO = new Date().toISOString().slice(0, 10);
    const lines: string[] = [
      `# 🔥 FireSage Institutional Retirement Committee Memo`,
      `**Generated:** ${new Date().toLocaleString()} | **Work Jurisdiction:** ${meta.flag} ${meta.name} | **Route:** ${curCity.name} → ${retCity.name}`,
      '',
      `## 1. Executive Actuarial Verdict`,
      `| Metric | Value | Notes |`,
      `| :--- | :--- | :--- |`,
      `| **Plan Survival (to Age ${input.horizon_age})** | **${analysis.success_rate.toFixed(1)}%** | ${analysis.success_rate >= 90 ? 'Clears 90% Institutional Target' : 'Below 90% Target — Optimization Recommended'} |`,
      `| **Pre-Age-${meta.unlockAge} Liquidity Safe** | **${analysis.liquidity_safe_rate.toFixed(1)}%** | Statutory penalty-free unlock at Age ${meta.unlockAge} |`,
      `| **Projected Median Corpus @ Age ${input.retirement_age}** | **${formatMoneyCompact(analysis.corpus_at_fire_real_p50, input.country)}** | P10–P90: ${formatMoneyCompact(analysis.corpus_at_fire_real_p10, input.country)} – ${formatMoneyCompact(analysis.corpus_at_fire_real_p90, input.country)} |`,
      `| **90%-Safe Required Corpus** | **${formatMoneyCompact(analysis.required_corpus_90, input.country)}** | Implied Initial SWR: ${analysis.implied_initial_swr.toFixed(2)}% |`,
      `| **Sustainable 90% Monthly Budget** | **${formatMoneyCompact(analysis.safe_monthly_spend_90, input.country)}/mo** | Terminal Median Legacy: ${formatMoneyCompact(analysis.terminal_real_p50, input.country)} |`,
      `| **Milestone FIRE Ages** | **Target 90%: Age ${analysis.target_fire_age.toFixed(1)}** | Coast: ${analysis.coast_fire_age.toFixed(1)} · Fortress (97%): ${analysis.fortress_fire_age.toFixed(1)} |`,
      '',
      `## 2. Ranked Action Levers`,
      ...analysis.action_levers.map(
        (lv) =>
          `- **${lv.title}**: ${lv.success_delta_pct >= 0 ? '+' : ''}${lv.success_delta_pct.toFixed(1)}% survival → **${lv.new_success_rate.toFixed(1)}%** (FIRE Age ${lv.new_target_fire_age.toFixed(1)}) — ${lv.description}`,
      ),
      '',
      `## 3. Historical Crisis Cohort Replays`,
      ...analysis.crisis_replays.map(
        (cr) =>
          `- **${cr.crisis.name} (${cr.crisis.start_year})**: ${cr.survived ? '✓ Survived' : `✕ Depleted at Age ${cr.depletion_age}`} | First 5Y Real CAGR: ${cr.worst_5yr_real_ret.toFixed(1)}% | Terminal Real: ${formatMoneyCompact(cr.terminal_real, input.country)}`,
      ),
      '',
      `## 4. FIRE Quant Copilot Deliberative Q&A Transcript`,
    ];

    for (const m of messages) {
      if (m.id === 'welcome') continue;
      lines.push('');
      lines.push(
        `### ${m.role === 'user' ? '🧑 Investor Question' : `🤖 FIRE Copilot (${m.engineLabel || 'Quant Engine'})`} [${m.timestamp}]`,
      );
      if (m.toolTrace && m.toolTrace.length > 0) {
        lines.push(
          `> **Executed Tools:** ${m.toolTrace.map((t) => `\`${t.tool}\` (${t.duration_ms}ms)`).join(', ')}`,
        );
      }
      lines.push('');
      lines.push(m.text);
    }

    const blob = new Blob([lines.join('\n')], { type: 'text/markdown;charset=utf-8' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `FireSage_Retirement_Memo_${input.country}_Age${input.retirement_age}_${dateISO}.md`;
    a.click();
    URL.revokeObjectURL(url);
  };

  if (!open) {
    return null;
  }

  const countryMeta = COUNTRY_META[input.country];
  const hasGeminiConfigured = Boolean(apiKey.trim()) || lastUsedLLM;

  return (
    <>
      <aside
        aria-labelledby={drawerTitleId}
        className={`fs-copilot-drawer ${isWide ? 'fs-copilot-wide' : ''}`}
      >
        {/* Polite screen-reader announcer */}
        <div className="visually-hidden" aria-live="polite" aria-atomic="true">
          {politeAnnouncement}
        </div>

        {/* DRAWER HEADER */}
        <div className="fs-copilot-head">
          <div className="fs-copilot-brand">
            <span className="fs-copilot-avatar" aria-hidden="true">
              🧠
            </span>
            <div>
              <h2 id={drawerTitleId} className="fs-copilot-title">
                FIRE Quant Copilot
              </h2>
              <p className="fs-copilot-sub">3-Stage Planner · Go Stochastic Tools · Page Driver</p>
            </div>
          </div>

          <div className="fs-copilot-toolbar">
            <button
              type="button"
              className="fs-copilot-toolbtn"
              onClick={toggleWide}
              aria-pressed={isWide}
              aria-label="Wide Copilot panel layout"
            >
              ⇔ Wide
            </button>
            <button
              type="button"
              className="fs-copilot-toolbtn fs-copilot-toolbtn-accent"
              onClick={exportRetirementMemoMarkdown}
            >
              📥 Memo
            </button>
            <button type="button" className="fs-copilot-toolbtn" onClick={handleClearChat}>
              ↺ Clear
            </button>
            <button
              type="button"
              className="fs-copilot-toolbtn"
              onClick={onClose}
              aria-label="Close FIRE Copilot drawer"
            >
              ✕
            </button>
          </div>
        </div>

        {/* LIVE PLAN CONTEXT + AI ENGINE PILL */}
        <div className="fs-copilot-ctxbar">
          <span className="fs-copilot-ctxtext">
            <span className="fs-copilot-dot" aria-hidden="true" />
            {countryMeta.flag} <strong>{curCity.name}</strong> → <strong>{retCity.name}</strong> •{' '}
            <strong>{analysis.success_rate.toFixed(1)}%</strong> @ Age {input.retirement_age}
          </span>
          <button
            type="button"
            className={`fs-copilot-modebadge ${hasGeminiConfigured ? 'is-llm' : ''}`}
            onClick={() => setSettingsOpen(true)}
          >
            {hasGeminiConfigured ? '🧠 Planner · Gemini' : '⚡ Planner · Quant Engine'} ⚙
          </button>
        </div>

        {/* 4 PERSONA MODES */}
        <fieldset className="fs-copilot-personas">
          <legend className="visually-hidden">Copilot reasoning persona</legend>
          {PERSONA_OPTIONS.map((p) => (
            <label
              key={p.id}
              className={`fs-persona-pill ${mode === p.id ? 'active' : ''}`}
            >
              <input
                type="radio"
                name="copilot_persona_mode"
                value={p.id}
                checked={mode === p.id}
                onChange={() => setMode(p.id)}
                className="visually-hidden"
              />
              {p.label}
            </label>
          ))}
        </fieldset>

        {/* 8 DYNAMIC CONTEXT-AWARE PROMPT CHIPS */}
        <ul className="fs-copilot-chips" role="list">
          {quickChips.map((chip) => (
            <li key={chip.label}>
              <button
                type="button"
                className="fs-copilot-chip"
                disabled={isSending}
                onClick={() => void sendPrompt(chip.prompt)}
              >
                {chip.label}
              </button>
            </li>
          ))}
        </ul>

        {/* CONVERSATION STREAM */}
        <div className="fs-copilot-messages">
          {messages.map((m) => (
            <article
              key={m.id}
              className={`fs-chat-msg ${m.role === 'user' ? 'fs-chat-user' : 'fs-chat-bot'}`}
            >
              <div className="fs-chat-meta">
                <span>
                  {m.role === 'user' ? '🧑 You' : m.engineLabel || '🤖 FIRE Copilot'}
                </span>
                <span>{m.timestamp}</span>
              </div>

              <div className="fs-chat-bubble">
                {/* Page-Driver Executed UI Banner */}
                {m.executedUISteps && m.executedUISteps.length > 0 && (
                  <div className="fs-pagedriver-banner">
                    <div className="fs-pagedriver-title">
                      🧭 Page-Driver UI Actions Executed Automatically
                    </div>
                    <ul className="fs-pagedriver-list">
                      {m.executedUISteps.map((st, idx) => (
                        <li key={idx}>✓ {st}</li>
                      ))}
                    </ul>
                  </div>
                )}

                {/* Native <details> + <summary> Deliberative Planner, Hypotheses & Tool Trace */}
                {m.plan && m.toolTrace && m.toolTrace.length > 0 && (
                  <details className="fs-planner-trace">
                    <summary className="fs-planner-summary">
                      🧠 Deliberative Planner, Hypotheses &amp; Tool Trace ({m.plan.steps.length}{' '}
                      Steps · {m.toolTrace.length} Tools)
                    </summary>
                    <div className="fs-planner-body">
                      <p className="fs-planner-thought">{m.plan.thought_summary}</p>

                      {m.plan.skills_invoked && m.plan.skills_invoked.length > 0 && (
                        <div className="fs-planner-skills">
                          {m.plan.skills_invoked.map((sk) => (
                            <span key={sk} className="fs-skill-tag">
                              {SKILL_BADGE_NAMES[sk] || sk}
                            </span>
                          ))}
                        </div>
                      )}

                      <div className="fs-planner-sec-label">Execution Plan</div>
                      <ol className="fs-planner-steps">
                        {m.plan.steps.map((st) => (
                          <li key={st.step}>
                            <strong>{st.title}:</strong> {st.detail}
                          </li>
                        ))}
                      </ol>

                      {m.plan.hypotheses && m.plan.hypotheses.length > 0 && (
                        <>
                          <div className="fs-planner-sec-label">Falsifiable Actuarial Hypotheses</div>
                          <ul className="fs-planner-hyps">
                            {m.plan.hypotheses.map((h, idx) => (
                              <li key={idx}>{h}</li>
                            ))}
                          </ul>
                        </>
                      )}

                      <div className="fs-planner-sec-label">
                        Executed Quantitative Go Engine Tools ({m.toolTrace.length})
                      </div>
                      <div className="fs-tooltrace-list">
                        {m.toolTrace.map((tc) => (
                          <div key={tc.id} className="fs-tooltrace-card">
                            <div className="fs-tooltrace-head">
                              <code>⚡ {tc.tool}</code>
                              <span className="fs-tooltrace-ms">{tc.duration_ms.toFixed(1)}ms</span>
                            </div>
                            <div className="fs-tooltrace-args">{tc.args_summary}</div>
                            <div className="fs-tooltrace-res">✓ {tc.summary}</div>
                          </div>
                        ))}
                      </div>
                    </div>
                  </details>
                )}

                {m.role === 'user' ? (
                  <p className="fs-md-p">{m.text}</p>
                ) : (
                  <CopilotMarkdown markdown={m.text} />
                )}

                {m.llmNote && <div className="fs-copilot-note">{m.llmNote}</div>}

                {/* 1-Click Follow-Up Prompts & Bi-Directional UI Mutation Buttons */}
                {m.actions && m.actions.length > 0 && (
                  <div className="fs-copilot-actions">
                    {m.actions.map((act, idx) => (
                      <button
                        key={idx}
                        type="button"
                        disabled={isSending}
                        className={`fs-action-chip ${act.ui_command ? 'is-mutation' : ''}`}
                        onClick={() => handleActionClick(act)}
                      >
                        {act.label}
                      </button>
                    ))}
                  </div>
                )}
              </div>
            </article>
          ))}

          {isSending && (
            <div className="fs-chat-msg fs-chat-bot">
              <div className="fs-chat-bubble fs-copilot-thinking">
                <strong>🧠 Running 3-Stage Deliberative FIRE Pipeline…</strong>
                <span>
                  1. Formulating actuarial hypotheses &amp; Page-Driver plan → 2. Executing Go CRN
                  Monte Carlo &amp; Geo-Arbitrage tools → 3. Synthesizing verdict
                </span>
              </div>
            </div>
          )}

          <div ref={messagesEndRef} />
        </div>

        {/* AUTO-GROWING COMPOSER (`field-sizing: content`) */}
        <form
          className="fs-copilot-composer"
          onSubmit={(e) => {
            e.preventDefault();
            void sendPrompt(draftText);
          }}
        >
          <label htmlFor={promptInputId} className="visually-hidden">
            Ask FIRE Quant Copilot a retirement, tax, geo-arbitrage, or scenario question
          </label>
          <textarea
            ref={textareaRef}
            id={promptInputId}
            name="copilot_prompt"
            className="fs-copilot-textarea"
            placeholder="Ask any FIRE question in plain English, compare cities, or drive sliders (e.g. 'What is Coast FIRE?', 'Compare Goa vs Algarve')…"
            value={draftText}
            onChange={(e) => setDraftText(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === 'Enter' && !e.shiftKey) {
                e.preventDefault();
                if (e.nativeEvent.isComposing || e.nativeEvent.keyCode === 229) {
                  return;
                }
                e.currentTarget.form?.requestSubmit();
              }
            }}
          />
          <button
            type="submit"
            className="fs-btn fs-btn-primary fs-copilot-sendbtn"
            disabled={isSending || !draftText.trim()}
          >
            Send ↵
          </button>
        </form>
      </aside>

      {/* GEMINI / VERTEX AI ENGINE SETTINGS MODAL */}
      <LightDismissDialog
        open={settingsOpen}
        onClose={() => setSettingsOpen(false)}
        title="FIRE Quant Copilot — AI Reasoner Settings"
        subtitle="Stage 1 (Strategic Planner) & Stage 2 (Parallel Go Stochastic Tools) run automatically with zero API keys. On Cloud Run, Stage 3 also auto-connects to Vertex AI Gemini — or you can supply an optional personal Gemini API key below."
        maxWidth="540px"
      >
        <form onSubmit={handleSaveSettings} style={{ display: 'flex', flexDirection: 'column', gap: '0.9rem' }}>
          <div className="fs-field" style={{ marginBottom: 0 }}>
            <label htmlFor={modelSelectId}>Preferred Stage-3 Gemini Model</label>
            <select
              id={modelSelectId}
              value={selectedModel}
              onChange={(e) => setSelectedModel(e.target.value)}
            >
              <option value="gemini-2.5-flash">gemini-2.5-flash (Fastest · Low Latency)</option>
              <option value="gemini-2.5-pro">gemini-2.5-pro (Deepest Actuarial Synthesis)</option>
            </select>
          </div>

          <div className="fs-field" style={{ marginBottom: 0 }}>
            <label htmlFor={apiKeyInputId}>Optional Gemini Developer API Key (Stored Only in Browser)</label>
            <input
              id={apiKeyInputId}
              type="password"
              name="gemini_api_key"
              autoComplete="off"
              placeholder="AIzaSy... (leave blank to use Cloud Run Vertex AI or Go Quant Engine)"
              value={apiKey}
              onChange={(e) => setApiKey(e.target.value)}
              style={{
                width: '100%',
                fontSize: '1rem',
                padding: '0.5rem 0.65rem',
                minHeight: '40px',
                borderRadius: '8px',
                border: '1px solid var(--border-color)',
                background: 'var(--bg-subtle)',
                color: 'var(--text-primary)',
              }}
            />
            <span className="fs-hint">
              Never logged or persisted on any server. Leave empty to rely on Cloud Run zero-key Vertex AI or
              deterministic Go/WASM synthesis.
            </span>
          </div>

          <div style={{ display: 'flex', justifyContent: 'flex-end', gap: '0.55rem' }}>
            {apiKey && (
              <button
                type="button"
                className="fs-btn fs-btn-ghost"
                onClick={() => {
                  setApiKey('');
                  localStorage.removeItem(STORAGE_KEY_API);
                }}
              >
                Clear Key
              </button>
            )}
            <button type="submit" className="fs-btn fs-btn-primary">
              Save Copilot Settings
            </button>
          </div>
        </form>
      </LightDismissDialog>
    </>
  );
};
