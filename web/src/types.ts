export type CountryCode = 'IN' | 'US' | 'AU' | 'EU' | 'NZ';
export const SUPPORTED_COUNTRIES: CountryCode[] = ['IN', 'US', 'AU', 'EU', 'NZ'];

export function isValidCountry(val: unknown): val is CountryCode {
  return typeof val === 'string' && (SUPPORTED_COUNTRIES as string[]).includes(val);
}

export type SimulationMode = 'bootstrap' | 'historical' | 'regime_mc' | 'stress_test';
export type WithdrawalStrategy = 'constant_real' | 'guyton_klinger' | 'cape_dynamic';

export interface LifestylePresets {
  lean: number;
  standard: number;
  chubby: number;
  fat: number;
}

export interface CityProfile {
  id: string;
  name: string;
  region: string;
  country: CountryCode;
  tier: string;
  tagline: string;
  cost_of_living_index: number;
  housing_cost_index: number;
  healthcare_cost_mult: number;
  tertiary_medical_buffer: number;
  state_code: string;
  state_top_tax_rate_pct: number;
  property_tax_rate: number;
  monthly_presets: LifestylePresets;
  typical_home_price: number;
}

export interface HistoricalCrisis {
  id: string;
  name: string;
  country: CountryCode;
  start_year: number;
  subtitle: string;
  description: string;
}

export interface AccountBuckets {
  use_granular: boolean;
  taxable_equity: number;
  taxable_cost_basis: number;
  taxable_debt_cash: number;
  tax_deferred_locked: number;
  tax_free_account: number;
  pension_locked: number;
}

export interface AssetAllocation {
  equity: number;
  debt: number;
  gold: number;
  cash: number;
  use_bond_tent: boolean;
}

export interface LifeEvent {
  id: string;
  name: string;
  start_age: number;
  end_age: number;
  annual_real: number;
  is_taxable_in: boolean;
}

export interface StressConfig {
  enable_custom_shock: boolean;
  retire_year1_crash_pct: number;
  extra_inflation_pct: number;
  extra_inflation_years: number;
  medical_emergency_real: number;
  medical_emergency_age: number;
}

export interface PlanInput {
  country: CountryCode;
  current_age: number;
  retirement_age: number;
  horizon_age: number;
  is_couple: boolean;
  current_city_id: string;
  retirement_city_id: string;
  auto_scale_lifestyle_on_move: boolean;
  current_corpus: number;
  buckets: AccountBuckets;
  monthly_savings: number;
  monthly_locked_savings?: number;
  annual_savings_step_up: number;
  desired_monthly_spend: number;
  allocation: AssetAllocation;
  expense_ratio_bps: number;
  withdrawal_strategy: WithdrawalStrategy;
  guardrail_floor_pct: number;
  guardrail_ceiling_pct: number;
  use_spending_smile: boolean;
  social_security_monthly_real: number;
  social_security_start_age: number;
  enable_tax_optimization: boolean;
  life_events: LifeEvent[];
  stress: StressConfig;
  mode: SimulationMode;
  num_paths: number;
  seed?: number;
}

export interface YearBand {
  age: number;
  year: number;
  is_retired: boolean;
  real_p5: number;
  real_p10: number;
  real_p25: number;
  real_p50: number;
  real_p75: number;
  real_p90: number;
  nominal_p10: number;
  nominal_p50: number;
  nominal_p90: number;
  liquid_real_p50: number;
  locked_real_p50: number;
  monthly_spend_real_p10: number;
  monthly_spend_real_p50: number;
  monthly_spend_real_p90: number;
  flow_savings_in_real: number;
  flow_portfolio_draw_real: number;
  flow_guaranteed_in_real: number;
  flow_milestone_in_real: number;
  flow_roth_or_harvest_real: number;
  flow_core_living_out_real: number;
  flow_healthcare_out_real: number;
  flow_taxes_out_real: number;
  flow_milestone_out_real: number;
  liquidity_crunch_pct: number;
}

export interface TrajectoryStep {
  age: number;
  historical_ref_year: number;
  equity_return: number;
  debt_return: number;
  headline_cpi: number;
  real_wealth: number;
  liquid_real_wealth: number;
  real_annual_spend: number;
  real_healthcare: number;
  real_taxes: number;
}

export interface SampleTrajectory {
  path_id: number;
  label: string;
  survived: boolean;
  depletion_age: number;
  terminal_real: number;
  steps: TrajectoryStep[];
}

export interface CrisisReplayOutcome {
  crisis: HistoricalCrisis;
  survived: boolean;
  depletion_age: number;
  min_real_wealth: number;
  min_wealth_age: number;
  terminal_real: number;
  worst_5yr_real_ret: number;
  real_wealth_series: number[];
}

export interface SRRCell {
  ret_bucket_idx: number;
  inf_bucket_idx: number;
  ret_bucket_label: string;
  inf_bucket_label: string;
  path_count: number;
  survival_pct: number;
}

export interface FrontierPoint {
  retirement_age: number;
  success_rate: number;
  stay_in_current_city_rate: number;
  safe_monthly_spend_90: number;
  median_corpus_at_retire: number;
}

export interface ActionLever {
  id: string;
  title: string;
  description: string;
  new_success_rate: number;
  success_delta_pct: number;
  new_target_fire_age: number;
  years_saved: number;
}

export interface GeoArbitrageSummary {
  current_city: CityProfile;
  comparison_city: CityProfile;
  is_relocating: boolean;
  effective_retire_monthly: number;
  stay_retire_monthly: number;
  stay_success_rate: number;
  move_success_rate: number;
  required_corpus_stay_90: number;
  required_corpus_move_90: number;
  corpus_saved_real: number;
  years_earlier_fire: number;
}

export interface FullAnalysisResponse {
  normalized_input: PlanInput;
  success_rate: number;
  liquidity_safe_rate: number;
  liquidity_warning_age: number;
  corpus_at_fire_real_p10: number;
  corpus_at_fire_real_p50: number;
  corpus_at_fire_real_p90: number;
  corpus_at_fire_nom_p50: number;
  terminal_real_p10: number;
  terminal_real_p50: number;
  terminal_real_p90: number;
  implied_initial_swr: number;
  safe_monthly_spend_90: number;
  required_corpus_90: number;
  coast_fire_age: number;
  lean_fire_age: number;
  target_fire_age: number;
  fortress_fire_age: number;
  yearly_bands: YearBand[];
  sample_paths: SampleTrajectory[];
  crisis_replays: CrisisReplayOutcome[];
  srr_heatmap: SRRCell[];
  frontier: FrontierPoint[];
  action_levers: ActionLever[];
  geo_arbitrage: GeoArbitrageSummary;
  compute_duration_ms: number;
}

export interface ValuationInfo {
  metric: string;
  current: number;
  historical_median: number;
  history_span: string;
}

export interface MetadataResponse {
  cities: CityProfile[];
  crises: Record<CountryCode, HistoricalCrisis[]>;
  valuations: Record<CountryCode, ValuationInfo>;
  defaults: Record<CountryCode, PlanInput>;
  fx_usd_to_inr?: number;
  fx_rates_per_usd?: Record<CountryCode, number>;
}

export interface SavedPlanRecord {
  id: string;
  name: string;
  schema_version: number;
  created_at: string;
  updated_at: string;
  input: PlanInput;
}

export type LabTabId = 'levers' | 'crises' | 'guardrails' | 'srr';
export type PersonaMode = 'analyst' | 'eli5' | 'debate' | 'devil';

export interface PlanStep {
  step: number;
  title: string;
  detail: string;
  status: string;
}

export interface PlannerExecutionPlan {
  thought_summary: string;
  steps: PlanStep[];
  hypotheses: string[];
  skills_invoked: string[];
  planned_tools: string[];
}

export interface ToolCallTrace {
  id: string;
  tool: string;
  args_summary: string;
  duration_ms: number;
  status: string;
  summary: string;
}

export interface UICommand {
  cmd:
    | 'APPLY_PLAN_PATCH'
    | 'SET_VALUE_BASIS'
    | 'OPEN_AUTOPSY'
    | 'OPEN_LAB_TAB'
    | 'SCROLL_TO_SECTION'
    | 'OPEN_METHODOLOGY';
  reason: string;
  country?: CountryCode;
  basis?: 'real' | 'nominal';
  path_label?: string;
  tab?: LabTabId;
  section?: string;
  plan_input?: PlanInput;
}

export interface CopilotAction {
  label: string;
  prompt?: string;
  ui_command?: UICommand;
}

export interface ChatTurn {
  role: 'user' | 'assistant';
  text: string;
}

export interface ChatRequest {
  question: string;
  mode: PersonaMode;
  input: PlanInput;
  value_basis: 'real' | 'nominal';
  history: ChatTurn[];
  api_key?: string;
  model?: string;
}

export interface ChatResponse {
  ok: boolean;
  engine_label: string;
  used_llm: boolean;
  llm_note?: string;
  plan: PlannerExecutionPlan;
  tool_trace: ToolCallTrace[];
  ui_commands: UICommand[];
  evidence_pack: string;
  markdown: string;
  actions: CopilotAction[];
}

export interface CopilotMessage {
  id: string;
  role: 'user' | 'assistant';
  text: string;
  timestamp: string;
  engineLabel?: string;
  usedLLM?: boolean;
  llmNote?: string;
  plan?: PlannerExecutionPlan;
  toolTrace?: ToolCallTrace[];
  executedUISteps?: string[];
  actions?: CopilotAction[];
}
