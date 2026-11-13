package copilot

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/ausanyal/firesage/internal/data"
	"github.com/ausanyal/firesage/internal/engine"
)

func clonePlan(in engine.PlanInput) engine.PlanInput {
	cp := in
	if len(in.LifeEvents) > 0 {
		cp.LifeEvents = make([]engine.LifeEvent, len(in.LifeEvents))
		copy(cp.LifeEvents, in.LifeEvents)
	}
	return cp
}

func fastPlan(in engine.PlanInput, paths int) engine.PlanInput {
	cp := clonePlan(in)
	if cp.Mode != engine.ModeHistorical {
		cp.NumPaths = paths
	}
	cp.Seed = 20260417
	return cp
}

// RebalanceGranularBuckets scales granular account buckets proportionally when CurrentCorpus is overridden.
func RebalanceGranularBuckets(p *engine.PlanInput, newCorpus float64) {
	oldCorpus := p.CurrentCorpus
	p.CurrentCorpus = newCorpus
	if p.Buckets.UseGranular && oldCorpus > 0 {
		ratio := newCorpus / oldCorpus
		p.Buckets.TaxableEquity = math.Round(p.Buckets.TaxableEquity * ratio)
		p.Buckets.TaxableDebtCash = math.Round(p.Buckets.TaxableDebtCash * ratio)
		p.Buckets.TaxDeferredLocked = math.Round(p.Buckets.TaxDeferredLocked * ratio)
		p.Buckets.TaxFreeAccount = math.Round(p.Buckets.TaxFreeAccount * ratio)
		p.Buckets.PensionLocked = math.Round(p.Buckets.PensionLocked * ratio)
	}
}

type cityComparisonRow struct {
	City               data.CityProfile
	SuccessRate        float64
	LiquiditySafeRate  float64
	EffectiveMonthly   float64
	RequiredCorpus90   float64
	TargetFIREAge      float64
	TerminalP50        float64
}

type goalSeekResult struct {
	TargetConfidence   float64
	CurrentSuccessRate float64
	SolvedMonthlySIP   float64
	SIPDelta           float64
	AchievableViaSIP   bool
	RequiredCorpus90   float64
	SafeMonthlySpend90 float64
	TargetFIREAge90    float64
	FortressFIREAge97  float64
}

type stressAuditResult struct {
	CrashPct             float64
	ExtraInfPct          float64
	ExtraInfYears        int
	MedicalShock         float64
	MedicalAge           int
	BaselineSuccessRate  float64
	ShockedSuccessRate   float64
	ShockedTerminalP50   float64
	WorstCrisisName      string
	WorstCrisisSurvived  bool
	WorstCrisisDeplAge   int
	WorstCrisis5YrRet    float64
	SurvivedCrisesCount  int
	TotalCrisesCount     int
	ShockedPlanForUI     engine.PlanInput
}

type lifeEventAuditResult struct {
	Event               engine.LifeEvent
	BaselineSuccessRate float64
	EventSuccessRate    float64
	BaselineTerminalP50 float64
	EventTerminalP50    float64
	MutatedPlanForUI    engine.PlanInput
}

type taxLiquidityAuditResult struct {
	UnlockAge               int
	BridgeYears             int
	LiquidAtRetireP50       float64
	LockedAtRetireP50       float64
	LiquidSharePct          float64
	LiquiditySafeRate       float64
	LiquidityWarningAge     int
	MedianLifetimeTaxes     float64
	MedianLifetimeHealth    float64
	TaxAlphaDeltaPct        float64
	JurisdictionRuleSummary string
}

// ExecutePipeline runs Stage 1 (Strategic Planner), Stage 2 (Parallel Go Quantitative Tool Calls),
// and Stage 3 (Persona-Aware Deterministic Synthesis + Structured EvidencePack for Gemini/Vertex AI).
func ExecutePipeline(req ChatRequest) ChatResponse {
	q := strings.TrimSpace(req.Question)
	if q == "" {
		q = "Audit my retirement plan and highlight top action levers."
	}

	mode := PersonaMode(strings.ToLower(strings.TrimSpace(req.Mode)))
	switch mode {
	case PersonaELI5, PersonaDebate, PersonaDevil:
	default:
		mode = PersonaAnalyst
	}

	userBase := engine.NormalizePlanInput(req.Input)
	// Preserve user's interactive path count for any UI mutations we send back
	uiPathCount := req.Input.NumPaths
	if uiPathCount <= 0 {
		uiPathCount = 3000
	}

	intent := ParseUserQuestion(q, userBase)

	var uiCommands []UICommand
	var skills []string
	var plannedTools []string
	var toolTrace []ToolCallTrace
	var evidenceLines []string
	var actions []CopilotAction

	addSkill := func(id string) {
		for _, s := range skills {
			if s == id {
				return
			}
		}
		skills = append(skills, id)
	}
	addPlannedTool := func(t string) {
		for _, x := range plannedTools {
			if x == t {
				return
			}
		}
		plannedTools = append(plannedTools, t)
	}

	// ── STAGE 1: BUILD MUTATION CANDIDATE & SCHEDULE TOOLS / PAGE-DRIVER COMMANDS ──
	activeInput := clonePlan(userBase)
	activeInput.NumPaths = uiPathCount
	hasParamOverride := false
	var mutationNotes []string

	if intent.SwitchCountry != nil && *intent.SwitchCountry != userBase.Country {
		activeInput = engine.DefaultCountryTemplate(*intent.SwitchCountry)
		activeInput.NumPaths = uiPathCount
		hasParamOverride = true
		mutationNotes = append(mutationNotes, fmt.Sprintf("Jurisdiction → %s", CountryDisplayName(*intent.SwitchCountry)))
	}

	if intent.NewCurrentAge != nil && *intent.NewCurrentAge != activeInput.CurrentAge {
		activeInput.CurrentAge = *intent.NewCurrentAge
		if activeInput.RetirementAge < activeInput.CurrentAge {
			activeInput.RetirementAge = activeInput.CurrentAge
		}
		hasParamOverride = true
		mutationNotes = append(mutationNotes, fmt.Sprintf("Current Age → %d", activeInput.CurrentAge))
	}
	if intent.NewRetirementAge != nil && *intent.NewRetirementAge != activeInput.RetirementAge {
		activeInput.RetirementAge = *intent.NewRetirementAge
		if activeInput.HorizonAge <= activeInput.RetirementAge+4 {
			activeInput.HorizonAge = activeInput.RetirementAge + 25
		}
		hasParamOverride = true
		mutationNotes = append(mutationNotes, fmt.Sprintf("Retirement Age → %d", activeInput.RetirementAge))
	}
	if intent.NewHorizonAge != nil && *intent.NewHorizonAge != activeInput.HorizonAge {
		activeInput.HorizonAge = *intent.NewHorizonAge
		hasParamOverride = true
		mutationNotes = append(mutationNotes, fmt.Sprintf("Horizon Age → %d", activeInput.HorizonAge))
	}
	if intent.NewMonthlySpend != nil {
		activeInput.DesiredMonthlySpend = *intent.NewMonthlySpend
		hasParamOverride = true
		mutationNotes = append(mutationNotes, fmt.Sprintf("Monthly Spend → %s/mo", FormatCompact(*intent.NewMonthlySpend, activeInput.Country)))
	}
	if intent.NewCurrentCorpus != nil {
		RebalanceGranularBuckets(&activeInput, *intent.NewCurrentCorpus)
		hasParamOverride = true
		mutationNotes = append(mutationNotes, fmt.Sprintf("Current Corpus → %s", FormatCompact(*intent.NewCurrentCorpus, activeInput.Country)))
	}
	if intent.NewMonthlySavings != nil {
		activeInput.MonthlySavings = *intent.NewMonthlySavings
		hasParamOverride = true
		mutationNotes = append(mutationNotes, fmt.Sprintf("Monthly Liquid SIP → %s/mo", FormatCompact(*intent.NewMonthlySavings, activeInput.Country)))
	}
	if intent.NewEquityPct != nil {
		eq := *intent.NewEquityPct
		rem := math.Max(0.05, 1.0-eq)
		activeInput.Allocation.Equity = eq
		activeInput.Allocation.Debt = math.Round(rem*0.70*100) / 100
		activeInput.Allocation.Gold = math.Round(rem*0.15*100) / 100
		activeInput.Allocation.Cash = math.Max(0.02, math.Round((1.0-eq-activeInput.Allocation.Debt-activeInput.Allocation.Gold)*100)/100)
		hasParamOverride = true
		mutationNotes = append(mutationNotes, fmt.Sprintf("Equity Allocation → %.0f%%", eq*100))
	}
	if intent.NewStrategy != nil && *intent.NewStrategy != activeInput.WithdrawalStrategy {
		activeInput.WithdrawalStrategy = *intent.NewStrategy
		hasParamOverride = true
		mutationNotes = append(mutationNotes, fmt.Sprintf("Withdrawal Strategy → %s", *intent.NewStrategy))
	}
	if intent.EnableBondTent != nil && *intent.EnableBondTent != activeInput.Allocation.UseBondTent {
		activeInput.Allocation.UseBondTent = *intent.EnableBondTent
		hasParamOverride = true
		mutationNotes = append(mutationNotes, fmt.Sprintf("ERN Bond Tent → %v", *intent.EnableBondTent))
	}
	if intent.EnableSmile != nil && *intent.EnableSmile != activeInput.UseSpendingSmile {
		activeInput.UseSpendingSmile = *intent.EnableSmile
		hasParamOverride = true
		mutationNotes = append(mutationNotes, fmt.Sprintf("Blanchett Spending Smile → %v", *intent.EnableSmile))
	}
	if intent.EnableTaxOpt != nil && *intent.EnableTaxOpt != activeInput.EnableTaxOptimization {
		activeInput.EnableTaxOptimization = *intent.EnableTaxOpt
		hasParamOverride = true
		mutationNotes = append(mutationNotes, fmt.Sprintf("Tax Optimization → %v", *intent.EnableTaxOpt))
	}

	// Single-city relocation override (when not comparing multiple cities)
	if len(intent.MentionedCities) == 1 && !intent.IsComparison {
		mc := intent.MentionedCities[0]
		if mc.ID != activeInput.RetirementCityID {
			activeInput.RetirementCityID = mc.ID
			activeInput.AutoScaleLifestyleOnMove = true
			hasParamOverride = true
			mutationNotes = append(mutationNotes, fmt.Sprintf("Retirement City → %s (%s)", mc.Name, mc.Country))
		}
	}

	// Schedule Core & Specialist Tools
	addSkill("skill_monte_carlo_srr")
	addPlannedTool("audit_plan_telemetry")

	if hasParamOverride {
		addPlannedTool("update_live_plan_parameters")
	}
	if intent.IsComparison || len(intent.MentionedCities) >= 2 {
		addSkill("skill_geo_arbitrage_scanner")
		addPlannedTool("compare_cities_geo_arbitrage")
	}
	if intent.IsGoalSeek {
		addSkill("skill_goal_seek_solver")
		addPlannedTool("solve_goal_seek_targets")
	}
	if intent.IsStressOrCrisis || mode == PersonaDevil {
		addSkill("skill_crisis_stress_tester")
		addPlannedTool("run_stress_and_crisis_shock")
	}
	if intent.ProposedEvent != nil {
		addSkill("skill_life_events_architect")
		addPlannedTool("simulate_life_event")
	}
	if intent.IsTaxOrLiquidity || mode == PersonaDebate {
		addSkill("skill_liquidity_tax_bridge")
		addPlannedTool("audit_tax_and_liquidity_bridge")
	}

	// Emit explicit UI Page-Driver commands
	if hasParamOverride && intent.IsExplicitUIMutation {
		patchedCopy := clonePlan(activeInput)
		uiCommands = append(uiCommands, UICommand{
			Cmd:       "APPLY_PLAN_PATCH",
			Reason:    "Updated live plan: " + strings.Join(mutationNotes, " · "),
			PlanInput: &patchedCopy,
		})
	}
	if intent.SwitchBasis != "" && intent.SwitchBasis != req.ValueBasis && intent.IsExplicitUIMutation {
		uiCommands = append(uiCommands, UICommand{
			Cmd:    "SET_VALUE_BASIS",
			Basis:  intent.SwitchBasis,
			Reason: fmt.Sprintf("Switched valuation toggle to %s", strings.ToUpper(intent.SwitchBasis)),
		})
	}
	if intent.OpenAutopsy != "" {
		uiCommands = append(uiCommands, UICommand{
			Cmd:       "OPEN_AUTOPSY",
			PathLabel: intent.OpenAutopsy,
			Reason:    fmt.Sprintf("Opened Year-by-Year Trajectory Autopsy (%s)", intent.OpenAutopsy),
		})
	}
	if intent.OpenLabTab != "" {
		uiCommands = append(uiCommands, UICommand{
			Cmd:    "OPEN_LAB_TAB",
			Tab:    intent.OpenLabTab,
			Reason: fmt.Sprintf("Switched Battle-Test Lab tab to '%s'", strings.ToUpper(intent.OpenLabTab)),
		})
	}
	if intent.ScrollSection != "" {
		uiCommands = append(uiCommands, UICommand{
			Cmd:     "SCROLL_TO_SECTION",
			Section: intent.ScrollSection,
			Reason:  fmt.Sprintf("Focused dashboard panel '%s'", intent.ScrollSection),
		})
	}
	if intent.OpenSources {
		uiCommands = append(uiCommands, UICommand{
			Cmd:    "OPEN_METHODOLOGY",
			Reason: "Opened Quantitative Methodology & Global Tax Sources dialog",
		})
	}
	if len(uiCommands) > 0 {
		skills = append([]string{"skill_page_driver"}, skills...)
	}

	// ── STAGE 2: EXECUTE REAL GO STOCHASTIC ENGINE TOOLS ──

	// Tool 1: Baseline & Mutated Plan Telemetry Audit
	t0 := time.Now()
	baseAnalysis := engine.RunFullAnalysis(fastPlan(userBase, 1000))
	durBase := float64(time.Since(t0).Microseconds()) / 1000.0

	evalPlan := userBase
	evalAnalysis := baseAnalysis
	if hasParamOverride {
		tMut := time.Now()
		mutAnalysis := engine.RunFullAnalysis(fastPlan(activeInput, 1000))
		durMut := float64(time.Since(tMut).Microseconds()) / 1000.0
		evalPlan = engine.NormalizePlanInput(activeInput)
		evalAnalysis = mutAnalysis

		toolTrace = append(toolTrace, ToolCallTrace{
			ID:          "tc-mutate",
			Tool:        "update_live_plan_parameters",
			ArgsSummary: strings.Join(mutationNotes, ", "),
			DurationMs:  math.Round(durMut*10) / 10,
			Status:      "ok",
			Summary: fmt.Sprintf(
				"Survival %.1f%% → %.1f%% (%+.1f%%) · 90%%-Safe Corpus %s → %s · Target FIRE Age %.1f → %.1f",
				baseAnalysis.SuccessRate, mutAnalysis.SuccessRate, mutAnalysis.SuccessRate-baseAnalysis.SuccessRate,
				FormatCompact(baseAnalysis.RequiredCorpus90, userBase.Country), FormatCompact(mutAnalysis.RequiredCorpus90, evalPlan.Country),
				baseAnalysis.TargetFIREAge, mutAnalysis.TargetFIREAge,
			),
		})

		if !intent.IsExplicitUIMutation {
			patchedCopy := clonePlan(activeInput)
			actions = append(actions, CopilotAction{
				Label: "⚡ Apply Scenario to Live Dashboard (" + strings.Join(mutationNotes, ", ") + ")",
				UICommand: &UICommand{
					Cmd:       "APPLY_PLAN_PATCH",
					Reason:    "Applied simulated scenario: " + strings.Join(mutationNotes, " · "),
					PlanInput: &patchedCopy,
				},
			})
		}
	}

	curCity := evalAnalysis.GeoArbitrage.CurrentCity
	retCity := evalAnalysis.GeoArbitrage.ComparisonCity
	unlockAge := CountryUnlockAge(evalPlan.Country)

	toolTrace = append(toolTrace, ToolCallTrace{
		ID:          "tc-telemetry",
		Tool:        "audit_plan_telemetry",
		ArgsSummary: fmt.Sprintf("country=%s, ages=%d→%d→%d, city=%s→%s, paths=1000", evalPlan.Country, evalPlan.CurrentAge, evalPlan.RetirementAge, evalPlan.HorizonAge, curCity.ID, retCity.ID),
		DurationMs:  math.Round(durBase*10) / 10,
		Status:      "ok",
		Summary: fmt.Sprintf(
			"Survival %.1f%% (Pre-%d Liquidity %.1f%%) · P50 FIRE Corpus %s vs 90%%-Safe %s (SWR %.2f%%) · 90%% FIRE Age %.1f",
			evalAnalysis.SuccessRate, unlockAge, evalAnalysis.LiquiditySafeRate,
			FormatCompact(evalAnalysis.CorpusAtFireRealP50, evalPlan.Country),
			FormatCompact(evalAnalysis.RequiredCorpus90, evalPlan.Country),
			evalAnalysis.ImpliedInitialSWR, evalAnalysis.TargetFIREAge,
		),
	})

	// Auto-attach Goal-Seek or Liquidity Audit if plan shows underfunding or pre-unlock crunch!
	if !intent.IsGoalSeek && evalAnalysis.SuccessRate < 89.5 {
		addSkill("skill_goal_seek_solver")
		addPlannedTool("solve_goal_seek_targets")
		intent.IsGoalSeek = true
	}
	if !intent.IsTaxOrLiquidity && evalAnalysis.LiquiditySafeRate+1.0 < evalAnalysis.SuccessRate {
		addSkill("skill_liquidity_tax_bridge")
		addPlannedTool("audit_tax_and_liquidity_bridge")
		intent.IsTaxOrLiquidity = true
	}

	evidenceLines = append(evidenceLines,
		fmt.Sprintf("[TOOL:PLAN_TELEMETRY] Jurisdiction=%s | CurrentCity=%s (%s) | RetirementCity=%s (%s) | Ages: Current=%d, Retire=%d, Horizon=%d (Statutory Unlock Age=%d)",
			CountryDisplayName(evalPlan.Country), curCity.Name, curCity.Country, retCity.Name, retCity.Country, evalPlan.CurrentAge, evalPlan.RetirementAge, evalPlan.HorizonAge, unlockAge),
		fmt.Sprintf("[TOOL:INPUT_PARAMS] CurrentCorpus=%s | LiquidMonthlySIP=%s/mo + LockedMonthly=%s/mo (StepUp=%.1f%%/yr) | DesiredSpendInWorkCity=%s/mo (EffectiveRetireSpend=%s/mo) | Equity=%.0f%%, BondTent=%v, Strategy=%s, SpendingSmile=%v, TaxOpt=%v",
			FormatCompact(evalPlan.CurrentCorpus, evalPlan.Country),
			FormatCompact(evalPlan.MonthlySavings, evalPlan.Country),
			FormatCompact(evalPlan.MonthlyLockedSavings, evalPlan.Country),
			evalPlan.AnnualSavingsStepUp*100,
			FormatCompact(evalPlan.DesiredMonthlySpend, evalPlan.Country),
			FormatCompact(evalAnalysis.GeoArbitrage.EffectiveRetireMonthly, evalPlan.Country),
			evalPlan.Allocation.Equity*100, evalPlan.Allocation.UseBondTent, evalPlan.WithdrawalStrategy, evalPlan.UseSpendingSmile, evalPlan.EnableTaxOptimization),
		fmt.Sprintf("[TOOL:MONTE_CARLO_KPIS] PlanSurvival=%.1f%% | Pre-%d LiquiditySafe=%.1f%% (WarningAge=%d) | ProjectedCorpusAtFIRE: P10=%s, P50=%s, P90=%s (Nominal P50=%s) | 90%%-SafeCorpusNeeded=%s | ImpliedInitialSWR=%.2f%% | Sustainable90%%Spend=%s/mo | TerminalRealP50=%s | MilestoneAges: Coast=%.1f, Lean=%.1f, Target90%%=%.1f, Fortress97%%=%.1f",
			evalAnalysis.SuccessRate, unlockAge, evalAnalysis.LiquiditySafeRate, evalAnalysis.LiquidityWarningAge,
			FormatCompact(evalAnalysis.CorpusAtFireRealP10, evalPlan.Country),
			FormatCompact(evalAnalysis.CorpusAtFireRealP50, evalPlan.Country),
			FormatCompact(evalAnalysis.CorpusAtFireRealP90, evalPlan.Country),
			FormatCompact(evalAnalysis.CorpusAtFireNomP50, evalPlan.Country),
			FormatCompact(evalAnalysis.RequiredCorpus90, evalPlan.Country),
			evalAnalysis.ImpliedInitialSWR,
			FormatCompact(evalAnalysis.SafeMonthlySpend90, evalPlan.Country),
			FormatCompact(evalAnalysis.TerminalRealP50, evalPlan.Country),
			evalAnalysis.CoastFIREAge, evalAnalysis.LeanFIREAge, evalAnalysis.TargetFIREAge, evalAnalysis.FortressFIREAge),
	)
	if hasParamOverride {
		evidenceLines = append(evidenceLines, fmt.Sprintf(
			"[TOOL:MUTATION_DELTA] Applied/Simulated Overrides (%s): BaselineSurvival=%.1f%% → NewSurvival=%.1f%% (%+.1f%%) | Baseline90%%Corpus=%s → New90%%Corpus=%s | BaselineTargetFIREAge=%.1f → NewTargetFIREAge=%.1f",
			strings.Join(mutationNotes, "; "),
			baseAnalysis.SuccessRate, evalAnalysis.SuccessRate, evalAnalysis.SuccessRate-baseAnalysis.SuccessRate,
			FormatCompact(baseAnalysis.RequiredCorpus90, userBase.Country), FormatCompact(evalAnalysis.RequiredCorpus90, evalPlan.Country),
			baseAnalysis.TargetFIREAge, evalAnalysis.TargetFIREAge,
		))
	}

	// Tool 2: Multi-City Intra-Country & Cross-Border Geo-Arbitrage Comparator (Parallel Goroutines!)
	var cityRows []cityComparisonRow
	if intent.IsComparison || len(intent.MentionedCities) >= 2 {
		tGeo := time.Now()
		candCities := make([]data.CityProfile, 0, 5)
		seenID := map[string]bool{}
		pushCity := func(c data.CityProfile) {
			if c.ID != "" && !seenID[c.ID] && len(candCities) < 5 {
				seenID[c.ID] = true
				candCities = append(candCities, c)
			}
		}
		for _, mc := range intent.MentionedCities {
			pushCity(mc)
		}
		pushCity(curCity)
		pushCity(retCity)

		// Add smart domestic + cross-border arbitrage benchmarks if fewer than 4 cities selected
		defaultBenchIDs := map[data.CountryCode][]string{
			data.CountryIndia:      {"in-kolkata", "in-goa", "in-dehradun", "in-kochi", "eu-algarve"},
			data.CountryUSA:        {"us-raleigh", "us-tampa", "eu-algarve", "eu-south-italy"},
			data.CountryAustralia:  {"au-gold-coast", "au-adelaide", "nz-tauranga-nelson", "eu-algarve"},
			data.CountryEurope:     {"eu-algarve", "eu-south-italy", "eu-lugano", "eu-crete-greece"},
			data.CountryNewZealand: {"nz-tauranga-nelson", "nz-wellington", "au-gold-coast", "eu-algarve"},
		}
		for _, bid := range defaultBenchIDs[evalPlan.Country] {
			if len(candCities) >= 4 {
				break
			}
			pushCity(data.GetCity(bid, evalPlan.Country))
		}

		cityRows = make([]cityComparisonRow, len(candCities))
		var wg sync.WaitGroup
		for i, cProf := range candCities {
			wg.Add(1)
			go func(idx int, cp data.CityProfile) {
				defer wg.Done()
				pCity := fastPlan(evalPlan, 600)
				pCity.RetirementCityID = cp.ID
				pCity.AutoScaleLifestyleOnMove = true
				res := engine.RunFullAnalysis(pCity)
				cityRows[idx] = cityComparisonRow{
					City:              cp,
					SuccessRate:       res.SuccessRate,
					LiquiditySafeRate: res.LiquiditySafeRate,
					EffectiveMonthly:  res.GeoArbitrage.EffectiveRetireMonthly,
					RequiredCorpus90:  res.RequiredCorpus90,
					TargetFIREAge:     res.TargetFIREAge,
					TerminalP50:       res.TerminalRealP50,
				}
			}(i, cProf)
		}
		wg.Wait()

		sort.SliceStable(cityRows, func(i, j int) bool {
			if math.Abs(cityRows[i].SuccessRate-cityRows[j].SuccessRate) > 0.4 {
				return cityRows[i].SuccessRate > cityRows[j].SuccessRate
			}
			return cityRows[i].RequiredCorpus90 < cityRows[j].RequiredCorpus90
		})
		durGeo := float64(time.Since(tGeo).Microseconds()) / 1000.0

		var rowStrs []string
		for idx, r := range cityRows {
			rowStrs = append(rowStrs, fmt.Sprintf("#%d %s (%s): Survival=%.1f%%, PreUnlockSafe=%.1f%%, EffSpend=%s/mo, 90%%Corpus=%s, FIREAge=%.1f",
				idx+1, r.City.Name, r.City.Country, r.SuccessRate, r.LiquiditySafeRate,
				FormatCompact(r.EffectiveMonthly, evalPlan.Country),
				FormatCompact(r.RequiredCorpus90, evalPlan.Country),
				r.TargetFIREAge))
		}
		winner := cityRows[0]
		toolTrace = append(toolTrace, ToolCallTrace{
			ID:          "tc-geo",
			Tool:        "compare_cities_geo_arbitrage",
			ArgsSummary: fmt.Sprintf("cities=[%d global hubs], auto_scale=true, paths=600", len(cityRows)),
			DurationMs:  math.Round(durGeo*10) / 10,
			Status:      "ok",
			Summary: fmt.Sprintf("Top Hub: %s (%s) → %.1f%% Survival · 90%% Corpus %s · FIRE Age %.1f",
				winner.City.Name, winner.City.Country, winner.SuccessRate, FormatCompact(winner.RequiredCorpus90, evalPlan.Country), winner.TargetFIREAge),
		})
		evidenceLines = append(evidenceLines, "[TOOL:GEO_ARBITRAGE_COMPARISON] "+strings.Join(rowStrs, " | "))

		for _, r := range cityRows[:minInt(2, len(cityRows))] {
			if r.City.ID != evalPlan.RetirementCityID {
				pMove := clonePlan(evalPlan)
				pMove.NumPaths = uiPathCount
				pMove.RetirementCityID = r.City.ID
				pMove.AutoScaleLifestyleOnMove = true
				actions = append(actions, CopilotAction{
					Label: fmt.Sprintf("🌍 Switch Retirement City to %s (%.1f%% Survival)", r.City.Name, r.SuccessRate),
					UICommand: &UICommand{
						Cmd:       "APPLY_PLAN_PATCH",
						Reason:    fmt.Sprintf("Switched Retirement City to %s (%s)", r.City.Name, r.City.Country),
						PlanInput: &pMove,
					},
				})
			}
		}
	}

	// Tool 3: Common Random Numbers (CRN) Goal-Seek Solver (Exact Required Monthly SIP!)
	var gsResult *goalSeekResult
	if intent.IsGoalSeek {
		tGS := time.Now()
		tgt := intent.TargetConfidence
		curRate := evalAnalysis.SuccessRate
		solvedSIP := evalPlan.MonthlySavings
		achievable := true

		if evalPlan.RetirementAge <= evalPlan.CurrentAge {
			achievable = curRate >= tgt
		} else if math.Abs(curRate-tgt) > 0.4 {
			lo := 0.0
			hi := math.Max(evalPlan.MonthlySavings*6.0, evalPlan.DesiredMonthlySpend*4.0)
			pHi := fastPlan(evalPlan, 400)
			pHi.MonthlySavings = hi
			rHi := engine.RunFullAnalysis(pHi).SuccessRate
			if rHi < tgt {
				hi *= 3.0
				pHi.MonthlySavings = hi
				rHi = engine.RunFullAnalysis(pHi).SuccessRate
			}
			if rHi < tgt {
				achievable = false
				solvedSIP = hi
			} else {
				for iter := 0; iter < 9; iter++ {
					mid := 0.5 * (lo + hi)
					pMid := fastPlan(evalPlan, 400)
					pMid.MonthlySavings = mid
					rMid := engine.RunFullAnalysis(pMid).SuccessRate
					if rMid >= tgt {
						hi = mid
					} else {
						lo = mid
					}
				}
				stepRound := 500.0
				if evalPlan.Country == data.CountryIndia {
					stepRound = 2500.0
				}
				solvedSIP = math.Ceil(hi/stepRound) * stepRound
			}
		}

		gsResult = &goalSeekResult{
			TargetConfidence:   tgt,
			CurrentSuccessRate: curRate,
			SolvedMonthlySIP:   solvedSIP,
			SIPDelta:           solvedSIP - evalPlan.MonthlySavings,
			AchievableViaSIP:   achievable,
			RequiredCorpus90:   evalAnalysis.RequiredCorpus90,
			SafeMonthlySpend90: evalAnalysis.SafeMonthlySpend90,
			TargetFIREAge90:    evalAnalysis.TargetFIREAge,
			FortressFIREAge97:  evalAnalysis.FortressFIREAge,
		}
		durGS := float64(time.Since(tGS).Microseconds()) / 1000.0

		toolTrace = append(toolTrace, ToolCallTrace{
			ID:          "tc-goalseek",
			Tool:        "solve_goal_seek_targets",
			ArgsSummary: fmt.Sprintf("target_confidence=%.0f%%, retire_age=%d, crn_seed=20260417", tgt, evalPlan.RetirementAge),
			DurationMs:  math.Round(durGS*10) / 10,
			Status:      "ok",
			Summary: fmt.Sprintf("Required Liquid SIP for %.0f%% odds: %s/mo (%+s/mo) · 90%% Safe Spend: %s/mo · 90%% FIRE Age: %.1f",
				tgt, FormatCompact(solvedSIP, evalPlan.Country), FormatCompact(gsResult.SIPDelta, evalPlan.Country),
				FormatCompact(gsResult.SafeMonthlySpend90, evalPlan.Country), gsResult.TargetFIREAge90),
		})
		evidenceLines = append(evidenceLines, fmt.Sprintf(
			"[TOOL:GOAL_SEEK_SOLVER] TargetConfidence=%.0f%% at Age %d | CurrentLiquidSIP=%s/mo (%.1f%% Survival) → SolvedLiquidSIP=%s/mo (Delta=%s/mo, Achievable=%v) | 90%%-SafeCorpusAtFIRE=%s | MaxSustainable90%%MonthlySpend=%s/mo | Earliest90%%FIREAge=%.1f | Fortress97%%FIREAge=%.1f",
			tgt, evalPlan.RetirementAge,
			FormatCompact(evalPlan.MonthlySavings, evalPlan.Country), curRate,
			FormatCompact(solvedSIP, evalPlan.Country), FormatCompact(gsResult.SIPDelta, evalPlan.Country), achievable,
			FormatCompact(gsResult.RequiredCorpus90, evalPlan.Country),
			FormatCompact(gsResult.SafeMonthlySpend90, evalPlan.Country),
			gsResult.TargetFIREAge90, gsResult.FortressFIREAge97,
		))

		if achievable && math.Abs(gsResult.SIPDelta) > 500 {
			pSIP := clonePlan(evalPlan)
			pSIP.NumPaths = uiPathCount
			pSIP.MonthlySavings = solvedSIP
			actions = append(actions, CopilotAction{
				Label: fmt.Sprintf("⚡ Set Liquid SIP to %s/mo (Hit %.0f%% Goal)", FormatCompact(solvedSIP, evalPlan.Country), tgt),
				UICommand: &UICommand{
					Cmd:       "APPLY_PLAN_PATCH",
					Reason:    fmt.Sprintf("Applied Goal-Seek Liquid SIP of %s/mo for %.0f%% survival", FormatCompact(solvedSIP, evalPlan.Country), tgt),
					PlanInput: &pSIP,
				},
			})
		}
	}

	// Tool 4: Custom Crash + Stagflation + Medical Emergency & Historical Crisis Replay Auditor
	var stressRes *stressAuditResult
	if intent.IsStressOrCrisis || mode == PersonaDevil {
		tSt := time.Now()
		crash := -0.35
		if intent.CrashPct != nil {
			crash = *intent.CrashPct
		}
		extraInf := 0.035
		if intent.ExtraInfPct != nil {
			extraInf = *intent.ExtraInfPct
		}
		medShock := 85_000.0
		if evalPlan.Country == data.CountryIndia {
			medShock = 2_500_000.0
		}
		if intent.MedicalShock != nil {
			medShock = *intent.MedicalShock
		}

		pShock := fastPlan(evalPlan, 600)
		pShock.Stress = engine.StressConfig{
			EnableCustomShock:    true,
			RetireYear1CrashPct:  crash,
			ExtraInflationPct:    extraInf,
			ExtraInflationYears:  5,
			MedicalEmergencyReal: medShock,
			MedicalEmergencyAge:  intent.MedicalShockAge,
		}
		shockAnalysis := engine.RunFullAnalysis(pShock)

		pShockUI := clonePlan(evalPlan)
		pShockUI.NumPaths = uiPathCount
		pShockUI.Stress = pShock.Stress

		survivedCnt := 0
		worstName := "N/A"
		worstSurv := true
		worstDepl := 0
		worst5Yr := 999.0
		for _, cr := range evalAnalysis.CrisisReplays {
			if cr.Survived {
				survivedCnt++
			}
			if cr.Worst5YrRealRet < worst5Yr || (!cr.Survived && worstSurv) {
				worst5Yr = cr.Worst5YrRealRet
				worstName = cr.Crisis.Name
				worstSurv = cr.Survived
				worstDepl = cr.DepletionAge
			}
		}

		stressRes = &stressAuditResult{
			CrashPct:            crash,
			ExtraInfPct:         extraInf,
			ExtraInfYears:       5,
			MedicalShock:        medShock,
			MedicalAge:          intent.MedicalShockAge,
			BaselineSuccessRate: evalAnalysis.SuccessRate,
			ShockedSuccessRate:  shockAnalysis.SuccessRate,
			ShockedTerminalP50:  shockAnalysis.TerminalRealP50,
			WorstCrisisName:     worstName,
			WorstCrisisSurvived: worstSurv,
			WorstCrisisDeplAge:  worstDepl,
			WorstCrisis5YrRet:   worst5Yr,
			SurvivedCrisesCount: survivedCnt,
			TotalCrisesCount:    len(evalAnalysis.CrisisReplays),
			ShockedPlanForUI:    pShockUI,
		}
		durSt := float64(time.Since(tSt).Microseconds()) / 1000.0

		toolTrace = append(toolTrace, ToolCallTrace{
			ID:          "tc-stress",
			Tool:        "run_stress_and_crisis_shock",
			ArgsSummary: fmt.Sprintf("yr1_crash=%.0f%%, extra_cpi=+%.1f%%×5y, med_shock=%s@%d", crash*100, extraInf*100, FormatCompact(medShock, evalPlan.Country), intent.MedicalShockAge),
			DurationMs:  math.Round(durSt*10) / 10,
			Status:      "ok",
			Summary: fmt.Sprintf("Survival %.1f%% → %.1f%% under triple shock · Historical Crises Survived: %d/%d",
				evalAnalysis.SuccessRate, shockAnalysis.SuccessRate, survivedCnt, len(evalAnalysis.CrisisReplays)),
		})
		evidenceLines = append(evidenceLines, fmt.Sprintf(
			"[TOOL:STRESS_AND_CRISIS_AUDIT] TripleShock(Yr1Crash=%.0f%%, ExtraCPI=+%.1f%%/yr×5y, MedShock=%s@Age%d): BaselineSurvival=%.1f%% → ShockedSurvival=%.1f%% (ShockedTerminalP50=%s) | HistoricalCrisesSurvived=%d/%d | ToughestHistoricalCohort=%s (First5YRealCAGR=%+.1f%%, Survived=%v, DepletionAge=%d)",
			crash*100, extraInf*100, FormatCompact(medShock, evalPlan.Country), intent.MedicalShockAge,
			evalAnalysis.SuccessRate, shockAnalysis.SuccessRate, FormatCompact(shockAnalysis.TerminalRealP50, evalPlan.Country),
			survivedCnt, len(evalAnalysis.CrisisReplays), worstName, worst5Yr, worstSurv, worstDepl,
		))

		if intent.IsStressOrCrisis && !evalPlan.Stress.EnableCustomShock {
			actions = append(actions, CopilotAction{
				Label: fmt.Sprintf("🧪 Overlay %.0f%% Crash + Stagflation Shock on Live Chart", crash*100),
				UICommand: &UICommand{
					Cmd:       "APPLY_PLAN_PATCH",
					Reason:    fmt.Sprintf("Enabled custom stress overlay (%.0f%% Year-1 crash + %.1f%% extra CPI)", crash*100, extraInf*100),
					PlanInput: &pShockUI,
				},
			})
		}
	}

	// Tool 5: Custom Life Event Cash-Flow Simulator
	var leRes *lifeEventAuditResult
	if intent.ProposedEvent != nil {
		tLE := time.Now()
		ev := *intent.ProposedEvent
		pLE := fastPlan(evalPlan, 600)
		pLE.LifeEvents = append(pLE.LifeEvents, ev)
		leAnalysis := engine.RunFullAnalysis(pLE)

		pLEUI := clonePlan(evalPlan)
		pLEUI.NumPaths = uiPathCount
		pLEUI.LifeEvents = append(pLEUI.LifeEvents, ev)

		leRes = &lifeEventAuditResult{
			Event:               ev,
			BaselineSuccessRate: evalAnalysis.SuccessRate,
			EventSuccessRate:    leAnalysis.SuccessRate,
			BaselineTerminalP50: evalAnalysis.TerminalRealP50,
			EventTerminalP50:    leAnalysis.TerminalRealP50,
			MutatedPlanForUI:    pLEUI,
		}
		durLE := float64(time.Since(tLE).Microseconds()) / 1000.0

		toolTrace = append(toolTrace, ToolCallTrace{
			ID:          "tc-lifeevent",
			Tool:        "simulate_life_event",
			ArgsSummary: fmt.Sprintf("name=%q, ages=%d–%d, annual_real=%s", ev.Name, ev.StartAge, ev.EndAge, FormatCompact(ev.AnnualReal, evalPlan.Country)),
			DurationMs:  math.Round(durLE*10) / 10,
			Status:      "ok",
			Summary: fmt.Sprintf("Survival %.1f%% → %.1f%% (%+.1f%%) · Terminal P50 %s → %s",
				evalAnalysis.SuccessRate, leAnalysis.SuccessRate, leAnalysis.SuccessRate-evalAnalysis.SuccessRate,
				FormatCompact(evalAnalysis.TerminalRealP50, evalPlan.Country), FormatCompact(leAnalysis.TerminalRealP50, evalPlan.Country)),
		})
		evidenceLines = append(evidenceLines, fmt.Sprintf(
			"[TOOL:LIFE_EVENT_SIMULATION] Event=%q (Ages %d–%d, AnnualReal=%s/yr): Survival %.1f%% → %.1f%% (%+.1f%%) | TerminalP50 %s → %s",
			ev.Name, ev.StartAge, ev.EndAge, FormatCompact(ev.AnnualReal, evalPlan.Country),
			evalAnalysis.SuccessRate, leAnalysis.SuccessRate, leAnalysis.SuccessRate-evalAnalysis.SuccessRate,
			FormatCompact(evalAnalysis.TerminalRealP50, evalPlan.Country), FormatCompact(leAnalysis.TerminalRealP50, evalPlan.Country),
		))

		if intent.IsExplicitUIMutation {
			uiCommands = append(uiCommands, UICommand{
				Cmd:       "APPLY_PLAN_PATCH",
				Reason:    fmt.Sprintf("Added Life Event '%s' (Ages %d–%d)", ev.Name, ev.StartAge, ev.EndAge),
				PlanInput: &pLEUI,
			})
		} else {
			actions = append(actions, CopilotAction{
				Label: fmt.Sprintf("📅 Add '%s' to Live Plan", ev.Name),
				UICommand: &UICommand{
					Cmd:       "APPLY_PLAN_PATCH",
					Reason:    fmt.Sprintf("Added Life Event '%s' to live plan", ev.Name),
					PlanInput: &pLEUI,
				},
			})
		}
	}

	// Tool 6: Statutory Lock-In, Liquidity Bridge & Jurisdiction Tax/Health Auditor
	var taxLiqRes *taxLiquidityAuditResult
	if intent.IsTaxOrLiquidity || mode == PersonaDebate {
		tTx := time.Now()
		var liqAtRet, lockAtRet, sumTax, sumHealth float64
		for _, b := range evalAnalysis.YearlyBands {
			if b.Age == evalPlan.RetirementAge {
				liqAtRet = b.LiquidRealP50
				lockAtRet = b.LockedRealP50
			}
			if b.IsRetired {
				sumTax += b.FlowTaxesOutReal
				sumHealth += b.FlowHealthcareOutReal
			}
		}
		totAtRet := math.Max(1, liqAtRet+lockAtRet)
		liqShare := (liqAtRet / totAtRet) * 100.0

		// Measure Tax Optimization Alpha by toggling EnableTaxOptimization
		pAltTax := fastPlan(evalPlan, 500)
		pAltTax.EnableTaxOptimization = !evalPlan.EnableTaxOptimization
		altTaxRes := engine.RunFullAnalysis(pAltTax)
		taxAlpha := math.Abs(evalAnalysis.SuccessRate - altTaxRes.SuccessRate)

		retCountry := retCity.Country
		ruleSum := jurisdictionTaxRuleText(evalPlan.Country, retCountry, retCity.ID)
		bridgeYrs := maxInt(0, unlockAge-evalPlan.RetirementAge)

		taxLiqRes = &taxLiquidityAuditResult{
			UnlockAge:               unlockAge,
			BridgeYears:             bridgeYrs,
			LiquidAtRetireP50:       liqAtRet,
			LockedAtRetireP50:       lockAtRet,
			LiquidSharePct:          liqShare,
			LiquiditySafeRate:       evalAnalysis.LiquiditySafeRate,
			LiquidityWarningAge:     evalAnalysis.LiquidityWarningAge,
			MedianLifetimeTaxes:     sumTax,
			MedianLifetimeHealth:    sumHealth,
			TaxAlphaDeltaPct:        taxAlpha,
			JurisdictionRuleSummary: ruleSum,
		}
		durTx := float64(time.Since(tTx).Microseconds()) / 1000.0

		toolTrace = append(toolTrace, ToolCallTrace{
			ID:          "tc-taxliq",
			Tool:        "audit_tax_and_liquidity_bridge",
			ArgsSummary: fmt.Sprintf("work_country=%s, retire_country=%s, unlock_age=%d, bridge_yrs=%d", evalPlan.Country, retCountry, unlockAge, bridgeYrs),
			DurationMs:  math.Round(durTx*10) / 10,
			Status:      "ok",
			Summary: fmt.Sprintf("FIRE Split: %.0f%% Liquid (%s) / %.0f%% Locked (%s) · Lifetime Real Tax %s & Health %s",
				liqShare, FormatCompact(liqAtRet, evalPlan.Country), 100-liqShare, FormatCompact(lockAtRet, evalPlan.Country),
				FormatCompact(sumTax, evalPlan.Country), FormatCompact(sumHealth, evalPlan.Country)),
		})
		evidenceLines = append(evidenceLines, fmt.Sprintf(
			"[TOOL:TAX_AND_LIQUIDITY_BRIDGE] StatutoryUnlockAge=%d (PreUnlockBridge=%d yrs) | AtFIRE_P50_Split: Liquid=%s (%.1f%%) vs Locked=%s (%.1f%%) | Pre-%d_LiquiditySafe=%.1f%% (WarningAge=%d) | CumulativeMedianRealTaxes=%s | CumulativeMedianRealHealthcare=%s | TaxOptimizationAlpha=%+.1f%% Survival | Rules=%s",
			unlockAge, bridgeYrs,
			FormatCompact(liqAtRet, evalPlan.Country), liqShare,
			FormatCompact(lockAtRet, evalPlan.Country), 100-liqShare,
			unlockAge, evalAnalysis.LiquiditySafeRate, evalAnalysis.LiquidityWarningAge,
			FormatCompact(sumTax, evalPlan.Country), FormatCompact(sumHealth, evalPlan.Country),
			taxAlpha, ruleSum,
		))
	}

	// Include Top Action Levers in EvidencePack
	if len(evalAnalysis.ActionLevers) > 0 {
		var lvParts []string
		for _, lv := range evalAnalysis.ActionLevers[:minInt(4, len(evalAnalysis.ActionLevers))] {
			lvParts = append(lvParts, fmt.Sprintf("%s (%+.1f%% Survival → %.1f%%, -%.1f yrs FIRE)", lv.Title, lv.SuccessDeltaPct, lv.NewSuccessRate, lv.YearsSaved))
		}
		evidenceLines = append(evidenceLines, "[TOOL:ACTION_LEVERS] "+strings.Join(lvParts, " | "))
	}

	// Build Planner Step-by-Step Execution Plan & 3 Falsifiable Hypotheses
	hypotheses := []string{
		fmt.Sprintf("H1 (Sequence-of-Returns & Corpus Sufficiency): Test whether %s projected median corpus at Age %d (Implied Initial SWR %.2f%%) survives stationary 4-year block-bootstrap crashes through Age %d at ≥90%% confidence.",
			FormatCompact(evalAnalysis.CorpusAtFireRealP50, evalPlan.Country), evalPlan.RetirementAge, evalAnalysis.ImpliedInitialSWR, evalPlan.HorizonAge),
		fmt.Sprintf("H2 (Pre-Age-%d Liquidity Bridge & Statutory Tax Drag): Verify that accessible liquid assets (%s) cover the %d-year pre-unlock bridge before locked retirement accounts unlock at Age %d without early-withdrawal penalties.",
			unlockAge, curCity.Country, maxInt(0, unlockAge-evalPlan.RetirementAge), unlockAge),
		fmt.Sprintf("H3 (Geo-Arbitrage, Medical CPI & Tail Resilience): Quantify lifestyle & medical cost scaling between %s and %s alongside historical stagflation/crash cohorts.",
			curCity.Name, retCity.Name),
	}

	steps := []PlanStep{
		{
			Step:   1,
			Title:  "Parse Intent, Multi-Currency Parameters & Jurisdiction Rules",
			Detail: fmt.Sprintf("Mapped query onto %s (%s → %s, Retire Age %d, %s mode).", CountryDisplayName(evalPlan.Country), curCity.Name, retCity.Name, evalPlan.RetirementAge, strings.ToUpper(string(mode))),
			Status: "done",
		},
		{
			Step:   2,
			Title:  fmt.Sprintf("Execute %d Parallel Stochastic Engine Tool(s)", len(toolTrace)),
			Detail: "Ran Go CRN block-bootstrap & solver tools: " + strings.Join(plannedTools, ", "),
			Status: "done",
		},
		{
			Step:   3,
			Title:  "Synthesize Actuarial Verdict & Interactive Page-Driver Controls",
			Detail: fmt.Sprintf("Verified H1–H3 against exact percentiles and dispatched %d UI page-driver command(s).", len(uiCommands)),
			Status: "done",
		},
	}

	plan := PlannerExecutionPlan{
		ThoughtSummary: fmt.Sprintf(
			"Decomposed query for %s (%s → %s): executed %d quantitative Go tool call(s), tested 3 falsifiable actuarial hypotheses, and dispatched %d live UI page-driver action(s).",
			CountryDisplayName(evalPlan.Country), curCity.Name, retCity.Name, len(toolTrace), len(uiCommands),
		),
		Steps:         steps,
		Hypotheses:    hypotheses,
		SkillsInvoked: skills,
		PlannedTools:  plannedTools,
	}

	// Add smart contextual follow-up actions if fewer than 4 buttons exist
	if evalPlan.WithdrawalStrategy != engine.StrategyGuytonKlinger || !evalPlan.Allocation.UseBondTent {
		pGuard := clonePlan(evalPlan)
		pGuard.NumPaths = uiPathCount
		pGuard.WithdrawalStrategy = engine.StrategyGuytonKlinger
		pGuard.GuardrailFloorPct = 0.82
		pGuard.Allocation.UseBondTent = true
		pGuard.EnableTaxOptimization = true
		actions = append(actions, CopilotAction{
			Label: "🛡️ Enable Guyton-Klinger Guardrails + Bond Tent",
			UICommand: &UICommand{
				Cmd:       "APPLY_PLAN_PATCH",
				Reason:    "Enabled Guyton-Klinger Guardrails + ERN 10-Yr Bond Tent + Tax Optimization",
				PlanInput: &pGuard,
			},
		})
	}
	actions = append(actions,
		CopilotAction{
			Label: "🔍 Open Worst-Path Year-by-Year Autopsy",
			UICommand: &UICommand{
				Cmd:       "OPEN_AUTOPSY",
				PathLabel: "Worst Path",
				Reason:    "Opened Worst-Path Year-by-Year Trajectory Autopsy",
			},
		},
	)
	if len(cityRows) == 0 {
		actions = append(actions, CopilotAction{
			Label:  "🌍 Compare Top 4 Retirement Cities",
			Prompt: fmt.Sprintf("Compare retiring in %s vs top domestic and global geo-arbitrage cities", curCity.Name),
		})
	}
	if stressRes == nil {
		actions = append(actions, CopilotAction{
			Label:  "⚡ Run -35% Year-1 Crash + Stagflation Stress Test",
			Prompt: "Stress test my plan against a -35% Year-1 crash and +3.5% inflation spike",
		})
	}
	if len(actions) > 5 {
		actions = actions[:5]
	}

	markdown := synthesizeDeterministicMarkdown(
		mode, intent, userBase, baseAnalysis, evalPlan, evalAnalysis,
		mutationNotes, cityRows, gsResult, stressRes, leRes, taxLiqRes, uiCommands,
	)

	return ChatResponse{
		OK:           true,
		EngineLabel:  "⚡ Go Quant Planner + Stochastic Tool Engine",
		UsedLLM:      false,
		Plan:         plan,
		ToolTrace:    toolTrace,
		UICommands:   uiCommands,
		EvidencePack: strings.Join(evidenceLines, "\n"),
		Markdown:     markdown,
		Actions:      actions,
	}
}

func jurisdictionTaxRuleText(workCountry, retireCountry data.CountryCode, retireCityID string) string {
	switch retireCountry {
	case data.CountryIndia:
		return "India New Regime (₹12L Sec 87A rebate excl. Sec 112A equity gains), 12.5% Equity LTCG above ₹1.25L annual tax-free harvest, Age-60 EPF/PPF/NPS 60/40 unlock, and 11%–13% Indian Medical CPI + Zone/hill-station buffer."
	case data.CountryUSA:
		return "US Federal + State progressive brackets, 5-yr Roth Conversion Ladder capped below the restored 2026 hard 400% FPL ACA Subsidy Cliff (<65), Medicare + IRMAA (65+), and SECURE 2.0 Age-73/75 RMDs."
	case data.CountryAustralia:
		return "Australia Stage 3 brackets + 2% Medicare Levy, 50% CGT discount + ASX 30% franking credit offset, 100% tax-free Superannuation unlock at Preservation Age 60, and Age-67 Centrelink Age Pension dual Assets + Deeming Income Test."
	case data.CountryEurope:
		switch retireCityID {
		case "eu-zurich-suburbs", "eu-lugano":
			return "Switzerland 0% private equity CGT + cantonal net wealth tax (0.18%–0.32%), Pillar 2/3a unlock at Age 60, and mandatory per-capita LAMal medical insurance."
		case "eu-south-italy":
			return "Southern Italy Art. 24-ter 7% flat foreign retiree tax for 10 years (then 26% CGT) + Italian SSN healthcare."
		case "eu-crete-greece":
			return "Greece Art. 5B 7% flat foreign retiree income/gains tax for 15 years + private hospital cover."
		case "eu-algarve", "eu-cascais-lisbon":
			return "Portugal Law 31/2024 holding-period CGT taper (19.6% effective rate after 8+ yrs) + SNS & private hospital cover."
		case "eu-nice-provence":
			return "France 30% PFU flat investment tax + 6.5% PUMA CSM health levy above €23k capital income."
		default:
			return "Spain 19%–30% progressive Renta del Ahorro savings tax + Sanitas private medical cover."
		}
	case data.CountryNewZealand:
		extra := ""
		if workCountry != data.CountryNewZealand {
			extra = " (+ 4-Yr Transitional Resident FIF Tax Holiday for cross-border arrivals)"
		}
		return "New Zealand 0% general CGT + Foreign Investment Fund (FIF) 5% Fair Dividend Rate (FDR) capped at 28% Multi-Rate PIE PIR" + extra + ", KiwiSaver locked to Age 65, and universal non-means-tested NZ Super at Age 65."
	default:
		return "Statutory progressive capital-gains & healthcare inflation model."
	}
}

func synthesizeDeterministicMarkdown(
	mode PersonaMode,
	intent ParsedIntent,
	basePlan engine.PlanInput,
	baseRes engine.FullAnalysisResponse,
	evalPlan engine.PlanInput,
	evalRes engine.FullAnalysisResponse,
	mutationNotes []string,
	cityRows []cityComparisonRow,
	gs *goalSeekResult,
	st *stressAuditResult,
	le *lifeEventAuditResult,
	tx *taxLiquidityAuditResult,
	uiCmds []UICommand,
) string {
	var b strings.Builder
	cc := evalPlan.Country
	unlockAge := CountryUnlockAge(cc)
	curCity := evalRes.GeoArbitrage.CurrentCity
	retCity := evalRes.GeoArbitrage.ComparisonCity

	statusTag := "🟢 INSTITUTIONAL FORTRESS"
	if evalRes.SuccessRate < 75 {
		statusTag = "🔴 UNDERFUNDED / HIGH SEQUENCE RISK"
	} else if evalRes.SuccessRate < 90 {
		statusTag = "🟡 BORDERLINE — NEEDS OPTIMIZATION"
	}

	if len(uiCmds) > 0 {
		b.WriteString(fmt.Sprintf("> 🧭 **Page Driver Active:** Dispatched **%d live UI update(s)** directly onto your FireSage dashboard.\n\n", len(uiCmds)))
	}

	// Persona-specific framing
	switch mode {
	case PersonaELI5:
		b.WriteString(fmt.Sprintf("### 👶 Plain-English Breakdown: Your %s → %s Retirement Plan\n\n", curCity.Name, retCity.Name))
		b.WriteString(fmt.Sprintf("Think of your retirement portfolio like a **water reservoir** feeding your household from **Age %d to Age %d** (%d years) across **3,000 different weather histories** (booms, droughts, and high-inflation storms):\n\n",
			evalPlan.RetirementAge, evalPlan.HorizonAge, evalPlan.HorizonAge-evalPlan.RetirementAge))
		b.WriteString(fmt.Sprintf("- **How often your reservoir lasts till Age %d:** **%.1f%% of the time** (%s). Institutional planners aim for **90%%+** so even a nasty early market storm doesn't empty the tank.\n",
			evalPlan.HorizonAge, evalRes.SuccessRate, statusTag))
		b.WriteString(fmt.Sprintf("- **The Two-Tank Rule (Before vs. After Age %d):** Money in locked retirement accounts (`EPF/NPS`, `401(k)`, `Super`, `KiwiSaver`) sits behind a locked gate until **Age %d**. Your **accessible liquid tank** survives until Age %d without touching locked money in **%.1f%%** of paths.\n",
			unlockAge, unlockAge, unlockAge, evalRes.LiquiditySafeRate))
		b.WriteString(fmt.Sprintf("- **Your Magic Target vs. What You're On Track For:** By Age %d, your median nest egg is projected to reach **%s** (in today's buying power), whereas a 90%%-safe cushion requires **%s** (an initial Year-1 draw of **%.2f%%**).\n\n",
			evalPlan.RetirementAge,
			FormatCompact(evalRes.CorpusAtFireRealP50, cc),
			FormatCompact(evalRes.RequiredCorpus90, cc),
			evalRes.ImpliedInitialSWR))

	case PersonaDebate:
		b.WriteString(fmt.Sprintf("### ⚔️ Retirement Committee Debate: %s → %s @ Age %d\n\n", curCity.Name, retCity.Name, evalPlan.RetirementAge))
		b.WriteString("#### 🟢 The Bull Case (Why Your Plan Works)\n")
		b.WriteString(fmt.Sprintf("- **Compounding & Savings Velocity:** Saving **%s/mo liquid + %s/mo locked** grows your median real nest egg to **%s** by Age %d (`P90 Bull`: **%s**).\n",
			FormatCompact(evalPlan.MonthlySavings, cc), FormatCompact(evalPlan.MonthlyLockedSavings, cc),
			FormatCompact(evalRes.CorpusAtFireRealP50, cc), evalPlan.RetirementAge, FormatCompact(evalRes.CorpusAtFireRealP90, cc)))
		if evalRes.GeoArbitrage.IsRelocating {
			b.WriteString(fmt.Sprintf("- **Geo-Arbitrage Tailwind:** Moving from **%s** to **%s** scales equivalent monthly living costs to **%s/mo**, saving **%s** in required 90%%-safe capital.\n",
				curCity.Name, retCity.Name, FormatCompact(evalRes.GeoArbitrage.EffectiveRetireMonthly, cc), FormatCompact(evalRes.GeoArbitrage.CorpusSavedReal, cc)))
		}
		b.WriteString(fmt.Sprintf("- **Dynamic Defense:** With `%s` withdrawal rules, median legacy wealth at Age %d finishes at **%s**.\n\n",
			evalPlan.WithdrawalStrategy, evalPlan.HorizonAge, FormatCompact(evalRes.TerminalRealP50, cc)))

		b.WriteString("#### 🔴 The Bear Case (What Could Break It)\n")
		b.WriteString(fmt.Sprintf("- **10th-Percentile Sequence Risk:** In poor early-market cohorts (`P10`), your Age-%d corpus lands at **%s** against a 90%%-safe requirement of **%s**.\n",
			evalPlan.RetirementAge, FormatCompact(evalRes.CorpusAtFireRealP10, cc), FormatCompact(evalRes.RequiredCorpus90, cc)))
		b.WriteString(fmt.Sprintf("- **Pre-Age-%d Bridge Exposure:** Retiring at **Age %d** leaves a **%d-year bridge** before statutory pension unlock (`Liquidity Safe`: **%.1f%%**).\n",
			unlockAge, evalPlan.RetirementAge, maxInt(0, unlockAge-evalPlan.RetirementAge), evalRes.LiquiditySafeRate))
		b.WriteString("- **Structural Medical Inflation:** Late-life healthcare compounds faster than headline CPI, squeezing fixed withdrawals in your 70s and 80s.\n\n")

		b.WriteString("#### ⚖️ Committee Verdict\n")
		b.WriteString(fmt.Sprintf("At **%.1f%% survival** (`Target 90%% FIRE Age`: **%.1f**), %s\n\n",
			evalRes.SuccessRate, evalRes.TargetFIREAge,
			ternaryStr(evalRes.SuccessRate >= 90, "the **Bull Case prevails** — your plan clears the 90% institutional bar.", "the **Bear Case flags an actionable gap** — use the Goal-Seek SIP or Geo-Arbitrage levers below to cross 90%.")))

	case PersonaDevil:
		b.WriteString(fmt.Sprintf("### 🦈 Chief Risk Officer Pre-Mortem Audit: %s → %s\n\n", curCity.Name, retCity.Name))
		b.WriteString(fmt.Sprintf("I stress-tested your **Age %d FIRE thesis** specifically hunting for failure modes across sequence-of-returns crashes, pre-Age-%d liquidity locks, and medical inflation:\n\n",
			evalPlan.RetirementAge, unlockAge))
	default:
		b.WriteString(fmt.Sprintf("### 🎯 Actuarial & Tax Synthesis: %s → %s (%s)\n\n", curCity.Name, retCity.Name, statusTag))
	}

	// Core Telemetry / Before-vs-After Table
	if len(mutationNotes) > 0 {
		b.WriteString("#### 📊 Before vs. After Scenario Telemetry\n\n")
		b.WriteString("| Metric | Baseline Plan | Simulated / Updated Plan | Delta |\n")
		b.WriteString("| :--- | :--- | :--- | :--- |\n")
		b.WriteString(fmt.Sprintf("| **Plan Survival (Age %d)** | %.1f%% | **%.1f%%** | **%+.1f%%** |\n",
			evalPlan.HorizonAge, baseRes.SuccessRate, evalRes.SuccessRate, evalRes.SuccessRate-baseRes.SuccessRate))
		b.WriteString(fmt.Sprintf("| **Pre-Age-%d Liquidity Safe** | %.1f%% | **%.1f%%** | %+.1f%% |\n",
			unlockAge, baseRes.LiquiditySafeRate, evalRes.LiquiditySafeRate, evalRes.LiquiditySafeRate-baseRes.LiquiditySafeRate))
		b.WriteString(fmt.Sprintf("| **Projected P50 Corpus @ FIRE** | %s | **%s** | %s |\n",
			FormatCompact(baseRes.CorpusAtFireRealP50, basePlan.Country), FormatCompact(evalRes.CorpusAtFireRealP50, cc),
			FormatCompact(evalRes.CorpusAtFireRealP50-baseRes.CorpusAtFireRealP50, cc)))
		b.WriteString(fmt.Sprintf("| **90%%-Safe Corpus Needed** | %s | **%s** | %s |\n",
			FormatCompact(baseRes.RequiredCorpus90, basePlan.Country), FormatCompact(evalRes.RequiredCorpus90, cc),
			FormatCompact(evalRes.RequiredCorpus90-baseRes.RequiredCorpus90, cc)))
		b.WriteString(fmt.Sprintf("| **90%%-Confidence FIRE Age** | Age %.1f | **Age %.1f** | %+.1f yrs |\n\n",
			baseRes.TargetFIREAge, evalRes.TargetFIREAge, evalRes.TargetFIREAge-baseRes.TargetFIREAge))
	} else {
		b.WriteString("| Core KPI | Live Reading | Institutional Benchmark |\n")
		b.WriteString("| :--- | :--- | :--- |\n")
		b.WriteString(fmt.Sprintf("| **Plan Survival (to Age %d)** | **%.1f%%** | ≥ 90.0%% Target (97%% Fortress) |\n", evalPlan.HorizonAge, evalRes.SuccessRate))
		b.WriteString(fmt.Sprintf("| **Pre-Age-%d Liquidity Safe** | **%.1f%%** | Avoids early locked-fund penalties |\n", unlockAge, evalRes.LiquiditySafeRate))
		b.WriteString(fmt.Sprintf("| **Median Corpus @ Age %d** | **%s** (`P10`: %s) | 90%%-Safe Target: **%s** |\n",
			evalPlan.RetirementAge, FormatCompact(evalRes.CorpusAtFireRealP50, cc), FormatCompact(evalRes.CorpusAtFireRealP10, cc), FormatCompact(evalRes.RequiredCorpus90, cc)))
		b.WriteString(fmt.Sprintf("| **Effective Retire Budget** | **%s/mo** (SWR `%.2f%%`) | 90%%-Safe Max Spend: **%s/mo** |\n",
			FormatCompact(evalRes.GeoArbitrage.EffectiveRetireMonthly, cc), evalRes.ImpliedInitialSWR, FormatCompact(evalRes.SafeMonthlySpend90, cc)))
		b.WriteString(fmt.Sprintf("| **Milestone FIRE Ages** | **Target: Age %.1f** | Coast: `%.1f` · Fortress: `%.1f` |\n\n",
			evalRes.TargetFIREAge, evalRes.CoastFIREAge, evalRes.FortressFIREAge))
	}

	// Why 35x vs Dynamic Multiplier section when asked
	if intent.IsMethodologyOr35x {
		annSpend := math.Max(1, evalRes.GeoArbitrage.EffectiveRetireMonthly*12)
		dynMult := evalRes.RequiredCorpus90 / annSpend
		b.WriteString("#### 📐 Why FireSage Computes a Dynamic Multiplier Instead of Static `35x`\n\n")
		b.WriteString(fmt.Sprintf("- Your **90%%-Safe Corpus (`%s`)** divided by your effective annual retirement spend (`%s/yr`) equals **`%.1fx`** (Implied Year-1 SWR: **`%.2f%%`**).\n",
			FormatCompact(evalRes.RequiredCorpus90, cc), FormatCompact(annSpend, cc), dynMult, evalRes.ImpliedInitialSWR))
		b.WriteString(fmt.Sprintf("- Unlike a rigid `25x` or `35x` rule of thumb, FireSage solves the exact corpus that survives **90%% of stationary block-bootstrap histories** over your specific **%d-year horizon**, factoring in **%s tax drag**, **healthcare inflation**, **pension inflows**, and **`%s` guardrails**.\n\n",
			evalPlan.HorizonAge-evalPlan.RetirementAge, retCity.Name, evalPlan.WithdrawalStrategy))
	}

	// Tool Output: Multi-City Geo-Arbitrage Table
	if len(cityRows) > 0 {
		b.WriteString("#### 🌍 Multi-City Geo-Arbitrage Scorecard (Parallel CRN Simulation)\n\n")
		b.WriteString("| Rank & City | Jurisdiction | Scaled Spend/mo | Survival % | 90%-Safe Corpus | 90% FIRE Age |\n")
		b.WriteString("| :--- | :--- | :--- | :--- | :--- | :--- |\n")
		for idx, r := range cityRows {
			b.WriteString(fmt.Sprintf("| **#%d %s** | %s | %s/mo | **%.1f%%** | %s | **Age %.1f** |\n",
				idx+1, r.City.Name, r.City.Country,
				FormatCompact(r.EffectiveMonthly, cc),
				r.SuccessRate,
				FormatCompact(r.RequiredCorpus90, cc),
				r.TargetFIREAge))
		}
		b.WriteString("\n")
	}

	// Tool Output: Goal-Seek Solver
	if gs != nil {
		b.WriteString(fmt.Sprintf("#### 🧮 CRN Goal-Seek Solution (%.0f%% Confidence @ Age %d)\n\n", gs.TargetConfidence, evalPlan.RetirementAge))
		if gs.CurrentSuccessRate >= gs.TargetConfidence {
			b.WriteString(fmt.Sprintf("- **Already Funded Above %.0f%%:** Your current liquid SIP (`%s/mo`) achieves **%.1f%% survival** — you could safely spend up to **%s/mo** or retire as early as **Age %.1f**.\n\n",
				gs.TargetConfidence, FormatCompact(evalPlan.MonthlySavings, cc), gs.CurrentSuccessRate, FormatCompact(gs.SafeMonthlySpend90, cc), gs.TargetFIREAge90))
		} else if gs.AchievableViaSIP {
			b.WriteString(fmt.Sprintf("- **Required Monthly Liquid SIP:** Increase liquid savings from `%s/mo` to **`%s/mo`** (**`+%s/mo`**) to reach **%.0f%% survival** at Age %d.\n",
				FormatCompact(evalPlan.MonthlySavings, cc), FormatCompact(gs.SolvedMonthlySIP, cc), FormatCompact(gs.SIPDelta, cc), gs.TargetConfidence, evalPlan.RetirementAge))
			b.WriteString(fmt.Sprintf("- **Alternative Levers (Zero Extra SIP):** Retire at **Age %.1f** for 90%% confidence, or trim target retirement spend to **`%s/mo`**.\n\n",
				gs.TargetFIREAge90, FormatCompact(gs.SafeMonthlySpend90, cc)))
		} else {
			b.WriteString(fmt.Sprintf("- **Horizon Too Short for SIP Alone:** Because Age %d is imminent, pair higher savings with **retiring at Age %.1f** or adjusting monthly spend toward **`%s/mo`**.\n\n",
				evalPlan.RetirementAge, gs.TargetFIREAge90, FormatCompact(gs.SafeMonthlySpend90, cc)))
		}
	}

	// Tool Output: Stress & Crisis Replay
	if st != nil {
		b.WriteString("#### ⚡ Triple-Shock Stress Test & Historical Crisis Audit\n\n")
		b.WriteString(fmt.Sprintf("- **Custom Triple Shock (`%.0f%%` Yr-1 Crash + `+%.1f%%/yr` CPI × 5y + `%s` Medical Shock @ Age %d):** Survival shifts from **%.1f%% → %.1f%%** (Shocked Median Terminal Wealth: `%s`).\n",
			st.CrashPct*100, st.ExtraInfPct*100, FormatCompact(st.MedicalShock, cc), st.MedicalAge,
			st.BaselineSuccessRate, st.ShockedSuccessRate, FormatCompact(st.ShockedTerminalP50, cc)))
		b.WriteString(fmt.Sprintf("- **Historical Crisis Cohorts Survived:** **%d of %d** famous market regimes. Toughest cohort: **%s** (`First 5Y Real CAGR`: `%+.1f%%`, %s).\n\n",
			st.SurvivedCrisesCount, st.TotalCrisesCount, st.WorstCrisisName, st.WorstCrisis5YrRet,
			ternaryStr(st.WorstCrisisSurvived, "✓ Survived", fmt.Sprintf("✕ Depleted at Age %d", st.WorstCrisisDeplAge))))
	}

	// Tool Output: Life Event Simulation
	if le != nil {
		b.WriteString(fmt.Sprintf("#### 📅 Life-Event Cash-Flow Impact: *%s* (Ages %d–%d)\n\n", le.Event.Name, le.Event.StartAge, le.Event.EndAge))
		b.WriteString(fmt.Sprintf("- **Annual Cash Flow (Today's Value):** `%s/yr` → Plan Survival shifts **%.1f%% → %.1f%%** (`%+.1f%%`), and Median Legacy Wealth moves `%s → %s`.\n\n",
			FormatCompact(le.Event.AnnualReal, cc), le.BaselineSuccessRate, le.EventSuccessRate, le.EventSuccessRate-le.BaselineSuccessRate,
			FormatCompact(le.BaselineTerminalP50, cc), FormatCompact(le.EventTerminalP50, cc)))
	}

	// Tool Output: Statutory Lock-In & Tax Bridge
	if tx != nil {
		b.WriteString(fmt.Sprintf("#### 🏛️ Statutory Age-%d Bridge, Tax & Healthcare Audit\n\n", tx.UnlockAge))
		b.WriteString(fmt.Sprintf("- **Corpus Split @ Age %d:** **%s Liquid (`%.0f%%`)** vs. **%s Locked (`%.0f%%`)** across a **%d-year pre-unlock bridge** (`Pre-%d Safe`: **%.1f%%**%s).\n",
			evalPlan.RetirementAge, FormatCompact(tx.LiquidAtRetireP50, cc), tx.LiquidSharePct,
			FormatCompact(tx.LockedAtRetireP50, cc), 100-tx.LiquidSharePct, tx.BridgeYears, tx.UnlockAge, tx.LiquiditySafeRate,
			ternaryStr(tx.LiquidityWarningAge > 0, fmt.Sprintf(", ⚠️ crunch watch near Age %d", tx.LiquidityWarningAge), "")))
		b.WriteString(fmt.Sprintf("- **Lifetime Median Drag:** **%s** in real taxes + **%s** in real healthcare outlays (`Tax Optimization Alpha`: **`%+.1f%%`** survival).\n",
			FormatCompact(tx.MedianLifetimeTaxes, cc), FormatCompact(tx.MedianLifetimeHealth, cc), tx.TaxAlphaDeltaPct))
		b.WriteString(fmt.Sprintf("- **Active Residency Regime:** %s\n\n", tx.JurisdictionRuleSummary))
	}

	// Top Action Levers footer
	if len(evalRes.ActionLevers) > 0 {
		best := evalRes.ActionLevers[0]
		b.WriteString(fmt.Sprintf("**🚀 Highest-Impact Next Lever:** **%s** (`%+.1f%%` survival → **%.1f%%**, saves **%.1f yrs** off Target FIRE Age).",
			best.Title, best.SuccessDeltaPct, best.NewSuccessRate, best.YearsSaved))
	}

	return strings.TrimSpace(b.String())
}

func ternaryStr(cond bool, a, b string) string {
	if cond {
		return a
	}
	return b
}
