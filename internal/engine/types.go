package engine

import "github.com/ausanyal/firesage/internal/data"

type SimulationMode string

const (
	ModeBootstrap  SimulationMode = "bootstrap"
	ModeHistorical SimulationMode = "historical"
	ModeRegimeMC   SimulationMode = "regime_mc"
	ModeStressTest SimulationMode = "stress_test"
)

type WithdrawalStrategy string

const (
	StrategyConstantReal  WithdrawalStrategy = "constant_real"
	StrategyGuytonKlinger WithdrawalStrategy = "guyton_klinger"
	StrategyCAPEDynamic   WithdrawalStrategy = "cape_dynamic"
)

// AccountBuckets holds granular tax/liquidity account balances in today's currency.
// When UseGranular is false, the engine derives realistic country-appropriate defaults from CurrentCorpus.
type AccountBuckets struct {
	UseGranular        bool    `json:"use_granular"`
	TaxableEquity      float64 `json:"taxable_equity"`       // US Brokerage / India Equity+Arbitrage MF
	TaxableCostBasis   float64 `json:"taxable_cost_basis"`   // Cost basis ratio (0.0–1.0, default 0.65)
	TaxableDebtCash    float64 `json:"taxable_debt_cash"`    // US Cash/Treasuries / India Debt MF & FDs
	TaxDeferredLocked  float64 `json:"tax_deferred_locked"`  // US Traditional 401(k)/IRA (59.5) / India EPF
	TaxFreeAccount     float64 `json:"tax_free_account"`     // US Roth IRA+HSA / India PPF (EEE)
	PensionLocked      float64 `json:"pension_locked"`       // India NPS Tier-I (locked to 60, 60/40 rule)
}

// AssetAllocation specifies target portfolio weights as decimals summing to 1.0.
type AssetAllocation struct {
	Equity     float64 `json:"equity"`
	Debt       float64 `json:"debt"`
	Gold       float64 `json:"gold"`
	Cash       float64 `json:"cash"`
	UseBondTent bool   `json:"use_bond_tent"` // ERN Rising Equity Glidepath: +18% bond/cash buffer at FIRE, gliding off over 10 yrs
}

// LifeEvent models one-time or multi-year custom cash flows (in today's purchasing power).
// Positive Amount = income/inflow (e.g. Barista consulting, downsizing home equity, inheritance).
// Negative Amount = expense/outflow (e.g. Kids' college, wedding, home purchase, sabbatical).
type LifeEvent struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	StartAge    int     `json:"start_age"`
	EndAge      int     `json:"end_age"`      // Inclusive; set equal to StartAge for one-time lump sum
	AnnualReal  float64 `json:"annual_real"`  // Signed annual amount in today's currency
	IsTaxableIn bool    `json:"is_taxable_in"` // True for Barista/consulting earned income
}

// StressConfig specifies custom shock parameters when SimulationMode == ModeStressTest (or overlaid).
type StressConfig struct {
	EnableCustomShock      bool    `json:"enable_custom_shock"`
	RetireYear1CrashPct    float64 `json:"retire_year1_crash_pct"`    // e.g. -0.35 for -35% crash in Year 1 of FIRE
	ExtraInflationPct      float64 `json:"extra_inflation_pct"`       // e.g. +0.04 (+4%/yr inflation spike)
	ExtraInflationYears    int     `json:"extra_inflation_years"`     // e.g. first 5 years of retirement
	MedicalEmergencyReal   float64 `json:"medical_emergency_real"`    // One-time uninsured shock in today's currency
	MedicalEmergencyAge    int     `json:"medical_emergency_age"`     // Default 72
}

// PlanInput encapsulates all Tier-1, Tier-2, and Tier-3 user parameters.
type PlanInput struct {
	Country                   data.CountryCode   `json:"country"`
	CurrentAge                int                `json:"current_age"`
	RetirementAge             int                `json:"retirement_age"`
	HorizonAge                int                `json:"horizon_age"`
	IsCouple                  bool               `json:"is_couple"`
	CurrentCityID             string             `json:"current_city_id"`
	RetirementCityID          string             `json:"retirement_city_id"`
	AutoScaleLifestyleOnMove  bool               `json:"auto_scale_lifestyle_on_move"`

	CurrentCorpus             float64            `json:"current_corpus"`
	Buckets                   AccountBuckets     `json:"buckets"`
	MonthlySavings            float64            `json:"monthly_savings"`
	AnnualSavingsStepUp       float64            `json:"annual_savings_step_up"` // e.g. 0.06 = 6%/yr career SIP step-up
	DesiredMonthlySpend       float64            `json:"desired_monthly_spend"`  // Today's purchasing power in CurrentCity (before move scaling)

	Allocation                AssetAllocation    `json:"allocation"`
	ExpenseRatioBps           float64            `json:"expense_ratio_bps"`
	WithdrawalStrategy        WithdrawalStrategy `json:"withdrawal_strategy"`
	GuardrailFloorPct         float64            `json:"guardrail_floor_pct"`    // e.g. 0.82 (max 18% discretionary belt-tightening)
	GuardrailCeilingPct       float64            `json:"guardrail_ceiling_pct"`  // e.g. 1.25 (max 25% lifestyle upgrade)
	UseSpendingSmile          bool               `json:"use_spending_smile"`     // Blanchett real retirement spending smile

	SocialSecurityMonthlyReal float64            `json:"social_security_monthly_real"` // US SSA or guaranteed pension in today's currency
	SocialSecurityStartAge    int                `json:"social_security_start_age"`    // Default 67 (US) or 60 (IN pension)
	EnableTaxOptimization     bool               `json:"enable_tax_optimization"`      // US MAGI-safe Roth Ladder / IN ₹1.25L LTCG harvesting

	LifeEvents                []LifeEvent        `json:"life_events"`
	Stress                    StressConfig       `json:"stress"`
	Mode                      SimulationMode     `json:"mode"`
	NumPaths                  int                `json:"num_paths"`
	Seed                      uint64             `json:"seed"`
}

// YearBand captures cross-sectional percentiles & cash-flow medians at a specific age.
type YearBand struct {
	Age                 int     `json:"age"`
	Year                int     `json:"year"`
	IsRetired           bool    `json:"is_retired"`
	RealP5              float64 `json:"real_p5"`
	RealP10             float64 `json:"real_p10"`
	RealP25             float64 `json:"real_p25"`
	RealP50             float64 `json:"real_p50"`
	RealP75             float64 `json:"real_p75"`
	RealP90             float64 `json:"real_p90"`
	NominalP10          float64 `json:"nominal_p10"`
	NominalP50          float64 `json:"nominal_p50"`
	NominalP90          float64 `json:"nominal_p90"`
	LiquidRealP50       float64 `json:"liquid_real_p50"`
	LockedRealP50       float64 `json:"locked_real_p50"`
	MonthlySpendRealP10 float64 `json:"monthly_spend_real_p10"`
	MonthlySpendRealP50 float64 `json:"monthly_spend_real_p50"`
	MonthlySpendRealP90 float64 `json:"monthly_spend_real_p90"`

	// Median annual real cash-flow components for Lifespan Streamgraph / Sankey
	FlowSavingsInReal      float64 `json:"flow_savings_in_real"`
	FlowPortfolioDrawReal  float64 `json:"flow_portfolio_draw_real"`
	FlowGuaranteedInReal   float64 `json:"flow_guaranteed_in_real"`
	FlowMilestoneInReal    float64 `json:"flow_milestone_in_real"`
	FlowRothOrHarvestReal  float64 `json:"flow_roth_or_harvest_real"`
	FlowCoreLivingOutReal  float64 `json:"flow_core_living_out_real"`
	FlowHealthcareOutReal  float64 `json:"flow_healthcare_out_real"`
	FlowTaxesOutReal       float64 `json:"flow_taxes_out_real"`
	FlowMilestoneOutReal   float64 `json:"flow_milestone_out_real"`
	LiquidityCrunchPct     float64 `json:"liquidity_crunch_pct"` // % of paths experiencing pre-60 liquid depletion at this age
}

// TrajectoryStep stores year-by-year details for interactive Click-to-Autopsy inspection.
type TrajectoryStep struct {
	Age               int     `json:"age"`
	HistoricalRefYear int     `json:"historical_ref_year"`
	EquityReturn      float64 `json:"equity_return"`
	DebtReturn        float64 `json:"debt_return"`
	HeadlineCPI       float64 `json:"headline_cpi"`
	RealWealth        float64 `json:"real_wealth"`
	LiquidRealWealth  float64 `json:"liquid_real_wealth"`
	RealAnnualSpend   float64 `json:"real_annual_spend"`
	RealHealthcare    float64 `json:"real_healthcare"`
	RealTaxes         float64 `json:"real_taxes"`
}

// SampleTrajectory represents an individual simulated life path for spaghetti overlay & autopsy.
type SampleTrajectory struct {
	PathID        int              `json:"path_id"`
	Label         string           `json:"label"` // e.g. "Worst Path", "P10 Stress", "Median Path", "P90 Bull", "Sample"
	Survived      bool             `json:"survived"`
	DepletionAge  int              `json:"depletion_age"` // 0 if survived
	TerminalReal  float64          `json:"terminal_real"`
	Steps         []TrajectoryStep `json:"steps"`
}

// CrisisReplayOutcome summarizes how the user's exact plan fared in a famous historical crisis cohort.
type CrisisReplayOutcome struct {
	Crisis            data.HistoricalCrisis `json:"crisis"`
	Survived          bool                  `json:"survived"`
	DepletionAge      int                   `json:"depletion_age"`
	MinRealWealth     float64               `json:"min_real_wealth"`
	MinWealthAge      int                   `json:"min_wealth_age"`
	TerminalReal      float64               `json:"terminal_real"`
	Worst5YrRealRet   float64               `json:"worst_5yr_real_ret"`
	RealWealthSeries  []float64             `json:"real_wealth_series"`
}

// SRRCell represents one cell in the Sequence-of-Returns Risk 2D survival heatmap.
type SRRCell struct {
	RetBucketIdx   int     `json:"ret_bucket_idx"`
	InfBucketIdx   int     `json:"inf_bucket_idx"`
	RetBucketLabel string  `json:"ret_bucket_label"`
	InfBucketLabel string  `json:"inf_bucket_label"`
	PathCount      int     `json:"path_count"`
	SurvivalPct    float64 `json:"survival_pct"`
}

// FrontierPoint represents plan viability if retiring at a specific candidate age.
type FrontierPoint struct {
	RetirementAge         int     `json:"retirement_age"`
	SuccessRate           float64 `json:"success_rate"`
	StayInCurrentCityRate float64 `json:"stay_in_current_city_rate"`
	SafeMonthlySpend90    float64 `json:"safe_monthly_spend_90"`
	MedianCorpusAtRetire  float64 `json:"median_corpus_at_retire"`
}

// ActionLever quantifies the impact of a concrete user adjustment on FIRE readiness.
type ActionLever struct {
	ID                 string  `json:"id"`
	Title              string  `json:"title"`
	Description        string  `json:"description"`
	NewSuccessRate     float64 `json:"new_success_rate"`
	SuccessDeltaPct    float64 `json:"success_delta_pct"`
	NewTargetFIREAge   float64 `json:"new_target_fire_age"`
	YearsSaved         float64 `json:"years_saved"`
}

// GeoArbitrageSummary quantifies the impact of retiring in RetirementCity vs CurrentCity.
type GeoArbitrageSummary struct {
	CurrentCity               data.CityProfile `json:"current_city"`
	ComparisonCity            data.CityProfile `json:"comparison_city"`
	IsRelocating              bool             `json:"is_relocating"`
	EffectiveRetireMonthly    float64          `json:"effective_retire_monthly"`
	StayRetireMonthly         float64          `json:"stay_retire_monthly"`
	StaySuccessRate           float64          `json:"stay_success_rate"`
	MoveSuccessRate           float64          `json:"move_success_rate"`
	RequiredCorpusStay90      float64          `json:"required_corpus_stay_90"`
	RequiredCorpusMove90      float64          `json:"required_corpus_move_90"`
	CorpusSavedReal           float64          `json:"corpus_saved_real"`
	YearsEarlierFIRE          float64          `json:"years_earlier_fire"`
}

// FullAnalysisResponse is the complete payload returned by /api/v1/analyze.
type FullAnalysisResponse struct {
	NormalizedInput      PlanInput             `json:"normalized_input"`
	SuccessRate          float64               `json:"success_rate"`
	LiquiditySafeRate    float64               `json:"liquidity_safe_rate"`
	LiquidityWarningAge  int                   `json:"liquidity_warning_age"` // 0 if no pre-unlock crunch
	CorpusAtFireRealP10  float64               `json:"corpus_at_fire_real_p10"`
	CorpusAtFireRealP50  float64               `json:"corpus_at_fire_real_p50"`
	CorpusAtFireRealP90  float64               `json:"corpus_at_fire_real_p90"`
	CorpusAtFireNomP50   float64               `json:"corpus_at_fire_nom_p50"`
	TerminalRealP10      float64               `json:"terminal_real_p10"`
	TerminalRealP50      float64               `json:"terminal_real_p50"`
	TerminalRealP90      float64               `json:"terminal_real_p90"`
	ImpliedInitialSWR    float64               `json:"implied_initial_swr"`
	SafeMonthlySpend90   float64               `json:"safe_monthly_spend_90"`
	RequiredCorpus90     float64               `json:"required_corpus_90"`

	// Milestone FIRE Ages solved along the Readiness Frontier
	CoastFIREAge         float64               `json:"coast_fire_age"`    // Stop saving now, retire at 60 with 90% confidence
	LeanFIREAge          float64               `json:"lean_fire_age"`     // 90% confidence at city's Lean lifestyle
	TargetFIREAge        float64               `json:"target_fire_age"`   // 90% confidence at user's desired lifestyle
	FortressFIREAge      float64               `json:"fortress_fire_age"` // 97% confidence at user's desired lifestyle

	YearlyBands          []YearBand            `json:"yearly_bands"`
	SamplePaths          []SampleTrajectory    `json:"sample_paths"`
	CrisisReplays        []CrisisReplayOutcome `json:"crisis_replays"`
	SRRHeatmap           []SRRCell             `json:"srr_heatmap"`
	Frontier             []FrontierPoint       `json:"frontier"`
	ActionLevers         []ActionLever         `json:"action_levers"`
	GeoArbitrage         GeoArbitrageSummary   `json:"geo_arbitrage"`
	ComputeDurationMs    float64               `json:"compute_duration_ms"`
}
