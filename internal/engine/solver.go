package engine

import (
	"fmt"
	"math"
	"slices"
	"sync"
	"time"

	"github.com/ausanyal/firesage/internal/data"
)

// RunFullAnalysis executes the complete stochastic simulation, historical crisis stress-tests,
// Sequence-of-Returns heatmap, Readiness Age Frontier, Geo-Arbitrage comparison, and Goal-Seek Levers.
func RunFullAnalysis(rawInput PlanInput) FullAnalysisResponse {
	t0 := time.Now()
	in := NormalizePlanInput(rawInput)
	outcomes := runMonteCarloPaths(in, in.NumPaths)

	nPaths := len(outcomes)
	numYears := in.HorizonAge - in.CurrentAge + 1

	survivedCount := 0
	liqSafeCount := 0
	var crunchAges []float64
	fireReal := make([]float64, nPaths)
	fireNom := make([]float64, nPaths)
	termReal := make([]float64, nPaths)

	for i, o := range outcomes {
		if o.survived {
			survivedCount++
		}
		if o.survived && !o.hadLiquidityCrunch {
			liqSafeCount++
		}
		if o.hadLiquidityCrunch && o.firstCrunchAge > 0 {
			crunchAges = append(crunchAges, float64(o.firstCrunchAge))
		}
		fireReal[i] = o.corpusAtFireReal
		fireNom[i] = o.corpusAtFireNom
		termReal[i] = o.terminalReal
	}

	successRate := 100.0 * float64(survivedCount) / float64(nPaths)
	liqSafeRate := 100.0 * float64(liqSafeCount) / float64(nPaths)
	liqWarnAge := 0
	if len(crunchAges) >= maxInt(3, nPaths/25) {
		slices.Sort(crunchAges)
		liqWarnAge = int(math.Round(quantileSorted(crunchAges, 0.50)))
	}

	_, fireP10, _, fireP50, _, fireP90 := sortAndQuantiles(fireReal)
	_, _, _, fireNomP50, _, _ := sortAndQuantiles(fireNom)
	_, termP10, _, termP50, _, termP90 := sortAndQuantiles(termReal)

	// Build year-by-year percentile bands & median cash-flow stream
	yearlyBands := make([]YearBand, numYears)
	colBuf := make([]float64, nPaths)
	nomBuf := make([]float64, nPaths)
	liqBuf := make([]float64, nPaths)
	lockBuf := make([]float64, nPaths)
	spendBuf := make([]float64, nPaths)

	for y := 0; y < numYears; y++ {
		age := in.CurrentAge + y
		crunchCount := 0
		for p := 0; p < nPaths; p++ {
			rec := outcomes[p].years[y]
			colBuf[p] = rec.realWealth
			nomBuf[p] = rec.nomWealth
			liqBuf[p] = rec.liquidReal
			lockBuf[p] = rec.lockedReal
			spendBuf[p] = rec.monthlySpendReal
			if rec.hadLiqCrunch {
				crunchCount++
			}
		}
		rp5, rp10, rp25, rp50, rp75, rp90 := sortAndQuantiles(colBuf)
		_, np10, _, np50, _, np90 := sortAndQuantiles(nomBuf)
		_, _, _, liqP50, _, _ := sortAndQuantiles(liqBuf)
		_, _, _, lockP50, _, _ := sortAndQuantiles(lockBuf)
		_, sp10, _, sp50, _, sp90 := sortAndQuantiles(spendBuf)

		yearlyBands[y] = YearBand{
			Age:                   age,
			Year:                  baseSimulationYear + y,
			IsRetired:             age >= in.RetirementAge,
			RealP5:                rp5,
			RealP10:               rp10,
			RealP25:               rp25,
			RealP50:               rp50,
			RealP75:               rp75,
			RealP90:               rp90,
			NominalP10:            np10,
			NominalP50:            np50,
			NominalP90:            np90,
			LiquidRealP50:         liqP50,
			LockedRealP50:         lockP50,
			MonthlySpendRealP10:   sp10,
			MonthlySpendRealP50:   sp50,
			MonthlySpendRealP90:   sp90,
			FlowSavingsInReal:     medianMetric(outcomes, y, func(r pathYearRecord) float64 { return r.savingsInReal }),
			FlowPortfolioDrawReal: medianMetric(outcomes, y, func(r pathYearRecord) float64 { return r.portfolioDrawReal }),
			FlowGuaranteedInReal:  medianMetric(outcomes, y, func(r pathYearRecord) float64 { return r.guaranteedInReal }),
			FlowMilestoneInReal:   medianMetric(outcomes, y, func(r pathYearRecord) float64 { return r.milestoneInReal }),
			FlowRothOrHarvestReal: medianMetric(outcomes, y, func(r pathYearRecord) float64 { return r.rothOrHarvestReal }),
			FlowCoreLivingOutReal: medianMetric(outcomes, y, func(r pathYearRecord) float64 { return r.coreLivingOutReal }),
			FlowHealthcareOutReal: medianMetric(outcomes, y, func(r pathYearRecord) float64 { return r.healthcareOutReal }),
			FlowTaxesOutReal:      medianMetric(outcomes, y, func(r pathYearRecord) float64 { return r.taxesOutReal }),
			FlowMilestoneOutReal:  medianMetric(outcomes, y, func(r pathYearRecord) float64 { return r.milestoneOutReal }),
			LiquidityCrunchPct:    100.0 * float64(crunchCount) / float64(nPaths),
		}
	}

	// Implied initial Safe Withdrawal Rate at RetirementAge
	impliedSWR := 0.0
	retIdx := in.RetirementAge - in.CurrentAge
	if retIdx >= 0 && retIdx < numYears && fireP50 > 0 {
		firstRetDraw := yearlyBands[retIdx].FlowPortfolioDrawReal
		impliedSWR = 100.0 * firstRetDraw / fireP50
	}

	// Curate representative sample paths (Worst, P10, Median, P90 + 20 spaghetti trajectories)
	samplePaths := selectSamplePaths(in.CurrentAge, outcomes)

	// Pre-generate Common Random Numbers (CRN) scenario matrix once for smooth, ultra-fast solver & frontier probes
	crnFull := pregenerateProbeScenarios(in, 320)
	crnFast := crnFull[:minInt(140, len(crnFull))]

	// Run parallel sub-analyses: Historical Crises, SRR Heatmap, SafeSpend/RequiredCorpus, Frontier, GeoArb, Levers
	var (
		crisisReplays      []CrisisReplayOutcome
		srrHeatmap         []SRRCell
		safeMonthlySpend90 float64
		requiredCorpus90   float64
		frontier           []FrontierPoint
		coastAge           float64
		leanAge            float64
		targetAge          float64
		fortressAge        float64
		geoArb             GeoArbitrageSummary
		levers             []ActionLever
	)

	var wg sync.WaitGroup
	wg.Add(3)

	go func() {
		defer wg.Done()
		crisisReplays = computeCrisisReplays(in)
		srrHeatmap = buildSRRHeatmap(in.Country, outcomes)
	}()

	go func() {
		defer wg.Done()
		safeMonthlySpend90 = solveSafeMonthlySpendCRN(in, 90.0, crnFull)
		requiredCorpus90 = solveRequiredCorpusAtRetireCRN(in, 90.0, crnFull)
		frontier, targetAge, fortressAge = computeReadinessFrontierCRN(in, safeMonthlySpend90, fireP50, crnFast)
		coastAge = solveCoastFIREAgeCRN(in, crnFast)
		leanAge = solveLeanFIREAgeCRN(in, crnFast)
	}()

	go func() {
		defer wg.Done()
		geoArb = computeGeoArbitrageSummaryCRN(in, successRate, crnFull)
	}()

	wg.Wait()

	// Compute concrete Action Levers using baseline successRate and targetAge
	levers = computeActionLeversCRN(in, successRate, targetAge, crnFull)

	return FullAnalysisResponse{
		NormalizedInput:     in,
		SuccessRate:         round2(successRate),
		LiquiditySafeRate:   round2(liqSafeRate),
		LiquidityWarningAge: liqWarnAge,
		CorpusAtFireRealP10: math.Round(fireP10),
		CorpusAtFireRealP50: math.Round(fireP50),
		CorpusAtFireRealP90: math.Round(fireP90),
		CorpusAtFireNomP50:  math.Round(fireNomP50),
		TerminalRealP10:     math.Round(termP10),
		TerminalRealP50:     math.Round(termP50),
		TerminalRealP90:     math.Round(termP90),
		ImpliedInitialSWR:   round2(impliedSWR),
		SafeMonthlySpend90:  math.Round(safeMonthlySpend90),
		RequiredCorpus90:    math.Round(requiredCorpus90),
		CoastFIREAge:        round1(coastAge),
		LeanFIREAge:         round1(leanAge),
		TargetFIREAge:       round1(targetAge),
		FortressFIREAge:     round1(fortressAge),
		YearlyBands:         yearlyBands,
		SamplePaths:         samplePaths,
		CrisisReplays:       crisisReplays,
		SRRHeatmap:          srrHeatmap,
		Frontier:            frontier,
		ActionLevers:        levers,
		GeoArbitrage:        geoArb,
		ComputeDurationMs:   round2(float64(time.Since(t0).Microseconds()) / 1000.0),
	}
}

func solveSafeMonthlySpendCRN(in PlanInput, targetPct float64, scenarios [][]data.MarketYear) float64 {
	lo := in.DesiredMonthlySpend * 0.15
	hi := in.DesiredMonthlySpend * 2.8
	for iter := 0; iter < 11; iter++ {
		mid := 0.5 * (lo + hi)
		probe := in
		probe.DesiredMonthlySpend = mid
		if probeSuccessOnly(probe, scenarios) >= targetPct {
			lo = mid
		} else {
			hi = mid
		}
	}
	return 0.5 * (lo + hi)
}

func solveRequiredCorpusAtRetireCRN(in PlanInput, targetPct float64, scenarios [][]data.MarketYear) float64 {
	probeBase := in
	retOffset := maxInt(0, in.RetirementAge-in.CurrentAge)
	probeBase.CurrentAge = in.RetirementAge
	probeBase.MonthlySavings = 0
	probeBase.Buckets.UseGranular = false

	// Slice scenarios starting from RetirementAge so retirement horizon matches exact crisis dynamics
	subScenarios := make([][]data.MarketYear, len(scenarios))
	for i, sc := range scenarios {
		if retOffset < len(sc) {
			subScenarios[i] = sc[retOffset:]
		} else {
			subScenarios[i] = sc
		}
	}

	annualSpend := in.DesiredMonthlySpend * 12.0
	lo := annualSpend * 8.0
	hi := annualSpend * 52.0
	for iter := 0; iter < 11; iter++ {
		mid := 0.5 * (lo + hi)
		probe := probeBase
		probe.CurrentCorpus = mid
		probe.Buckets = defaultBuckets(probe.Country, mid)
		if probeSuccessOnly(probe, subScenarios) >= targetPct {
			hi = mid
		} else {
			lo = mid
		}
	}
	return 0.5 * (lo + hi)
}

func computeReadinessFrontierCRN(in PlanInput, baseSafeMonthly float64, baseMedFire float64, scenarios [][]data.MarketYear) ([]FrontierPoint, float64, float64) {
	minAge := in.CurrentAge + 1
	maxAge := minInt(66, in.HorizonAge-5)
	if maxAge < minAge {
		maxAge = minAge
	}
	count := maxAge - minAge + 1
	pts := make([]FrontierPoint, count)

	// Anchor safe withdrawal yield from exact bisection at active RetirementAge
	safeMonthlyRatio := 0.036 / 12.0
	if baseMedFire > 0 && baseSafeMonthly > 0 {
		safeMonthlyRatio = clamp(baseSafeMonthly/baseMedFire, 0.018/12.0, 0.065/12.0)
	}

	var wg sync.WaitGroup
	for i := 0; i < count; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			candAge := minAge + idx
			probe := in
			probe.RetirementAge = candAge
			rate, medFire := probeSuccessAndMedianFire(probe, scenarios)

			stayRate := rate
			if in.RetirementCityID != in.CurrentCityID {
				stayProbe := probe
				stayProbe.RetirementCityID = in.CurrentCityID
				stayRate = probeSuccessOnly(stayProbe, scenarios)
			}

			// Horizon adjustment: retiring later means shorter drawdown horizon -> slightly higher sustainable SWR
			horizonDelta := float64(candAge - in.RetirementAge)
			horizonBoost := clamp(1.0+0.008*horizonDelta, 0.82, 1.35)
			safeSpend := medFire * safeMonthlyRatio * horizonBoost
			if candAge == in.RetirementAge && baseSafeMonthly > 0 {
				safeSpend = baseSafeMonthly
			}

			pts[idx] = FrontierPoint{
				RetirementAge:         candAge,
				SuccessRate:           round2(rate),
				StayInCurrentCityRate: round2(stayRate),
				SafeMonthlySpend90:    math.Round(safeSpend),
				MedianCorpusAtRetire:  math.Round(medFire),
			}
		}(i)
	}
	wg.Wait()

	targetAge := interpolateThresholdAge(pts, 90.0, float64(maxAge))
	fortressAge := interpolateThresholdAge(pts, 97.0, float64(maxAge))
	return pts, targetAge, fortressAge
}

func interpolateThresholdAge(pts []FrontierPoint, threshold float64, fallback float64) float64 {
	if len(pts) == 0 {
		return fallback
	}
	if pts[0].SuccessRate >= threshold {
		return float64(pts[0].RetirementAge)
	}
	for i := 1; i < len(pts); i++ {
		if pts[i].SuccessRate >= threshold {
			prev := pts[i-1]
			curr := pts[i]
			span := curr.SuccessRate - prev.SuccessRate
			if span <= 0.01 {
				return float64(curr.RetirementAge)
			}
			frac := (threshold - prev.SuccessRate) / span
			return float64(prev.RetirementAge) + clamp(frac, 0, 1)
		}
	}
	return fallback
}

func solveCoastFIREAgeCRN(in PlanInput, scenarios [][]data.MarketYear) float64 {
	traditionalRetire := maxInt(60, in.CurrentAge+1)
	if traditionalRetire >= in.HorizonAge-3 {
		return float64(in.CurrentAge)
	}

	check := func(stopSavingsAge int) float64 {
		probe := in
		probe.RetirementAge = traditionalRetire
		if stopSavingsAge <= in.CurrentAge {
			probe.MonthlySavings = 0
		} else {
			annualSav := in.MonthlySavings * 12.0
			probe.MonthlySavings = 0
			probe.LifeEvents = append(slices.Clone(in.LifeEvents), LifeEvent{
				ID:          "coast-sav",
				Name:        "Career Savings Until Coast Age",
				StartAge:    in.CurrentAge,
				EndAge:      stopSavingsAge - 1,
				AnnualReal:  annualSav,
				IsTaxableIn: false,
			})
		}
		return probeSuccessOnly(probe, scenarios)
	}

	if check(in.CurrentAge) >= 90.0 {
		return float64(in.CurrentAge)
	}
	lo := in.CurrentAge
	hi := traditionalRetire
	for hi-lo > 1 {
		mid := (lo + hi) / 2
		if check(mid) >= 90.0 {
			hi = mid
		} else {
			lo = mid
		}
	}
	rLo := check(lo)
	rHi := check(hi)
	if rHi > rLo && rHi >= 90.0 {
		return float64(lo) + clamp((90.0-rLo)/(rHi-rLo), 0, 1)
	}
	return float64(hi)
}

func solveLeanFIREAgeCRN(in PlanInput, scenarios [][]data.MarketYear) float64 {
	retCity := data.GetCity(in.RetirementCityID, in.Country)
	probe := in
	probe.DesiredMonthlySpend = retCity.MonthlyPresets.Lean
	probe.AutoScaleLifestyleOnMove = false

	return solveTargetAgeQuickCRN(probe, 90.0, scenarios)
}

func computeGeoArbitrageSummaryCRN(in PlanInput, currentPlanRate float64, scenarios [][]data.MarketYear) GeoArbitrageSummary {
	currCity := data.GetCity(in.CurrentCityID, in.Country)
	compCity := data.GetCity(in.RetirementCityID, in.Country)
	isRelocating := compCity.ID != currCity.ID

	if !isRelocating {
		if in.Country == data.CountryUSA {
			if currCity.ID == "us-raleigh" {
				compCity = data.GetCity("us-austin", data.CountryUSA)
			} else {
				compCity = data.GetCity("us-raleigh", data.CountryUSA)
			}
		} else {
			if currCity.ID == "in-goa" {
				compCity = data.GetCity("in-mysuru", data.CountryIndia)
			} else {
				compCity = data.GetCity("in-goa", data.CountryIndia)
			}
		}
	}

	stayProbe := in
	stayProbe.RetirementCityID = currCity.ID
	stayProbe.AutoScaleLifestyleOnMove = false

	moveProbe := in
	moveProbe.RetirementCityID = compCity.ID
	moveProbe.AutoScaleLifestyleOnMove = true

	var stayRate, moveRate float64
	if isRelocating && in.AutoScaleLifestyleOnMove {
		moveRate = currentPlanRate
		stayRate = probeSuccessOnly(stayProbe, scenarios)
	} else if !isRelocating {
		stayRate = currentPlanRate
		moveRate = probeSuccessOnly(moveProbe, scenarios)
	} else {
		stayRate = probeSuccessOnly(stayProbe, scenarios)
		moveRate = probeSuccessOnly(moveProbe, scenarios)
	}

	reqStay := solveRequiredCorpusAtRetireCRN(stayProbe, 90.0, scenarios)
	reqMove := solveRequiredCorpusAtRetireCRN(moveProbe, 90.0, scenarios)

	stayAge := solveTargetAgeQuickCRN(stayProbe, 90.0, scenarios)
	moveAge := solveTargetAgeQuickCRN(moveProbe, 90.0, scenarios)

	colScale := 1.0
	if currCity.CostOfLivingIndex > 0 {
		colScale = compCity.CostOfLivingIndex / currCity.CostOfLivingIndex
	}

	return GeoArbitrageSummary{
		CurrentCity:            currCity,
		ComparisonCity:         compCity,
		IsRelocating:           isRelocating,
		EffectiveRetireMonthly: math.Round(in.DesiredMonthlySpend * colScale),
		StayRetireMonthly:      math.Round(in.DesiredMonthlySpend),
		StaySuccessRate:        round2(stayRate),
		MoveSuccessRate:        round2(moveRate),
		RequiredCorpusStay90:   math.Round(reqStay),
		RequiredCorpusMove90:   math.Round(reqMove),
		CorpusSavedReal:        math.Round(reqStay - reqMove),
		YearsEarlierFIRE:       round1(stayAge - moveAge),
	}
}

func solveTargetAgeQuickCRN(in PlanInput, targetPct float64, scenarios [][]data.MarketYear) float64 {
	lo := in.CurrentAge + 1
	hi := minInt(68, in.HorizonAge-5)
	check := func(age int) float64 {
		p := in
		p.RetirementAge = age
		return probeSuccessOnly(p, scenarios)
	}
	if check(lo) >= targetPct {
		return float64(lo)
	}
	for hi-lo > 1 {
		mid := (lo + hi) / 2
		if check(mid) >= targetPct {
			hi = mid
		} else {
			lo = mid
		}
	}
	rLo := check(lo)
	rHi := check(hi)
	if rHi > rLo && rHi >= targetPct {
		return float64(lo) + clamp((targetPct-rLo)/(rHi-rLo), 0, 1)
	}
	return float64(hi)
}

func computeActionLeversCRN(in PlanInput, baseRate float64, baseTargetAge float64, scenarios [][]data.MarketYear) []ActionLever {
	type leverSpec struct {
		id     string
		title  string
		desc   string
		mutate func(p PlanInput) PlanInput
	}

	currCity := data.GetCity(in.CurrentCityID, in.Country)
	altCityID := "in-goa"
	altCityName := "Goa"
	if in.Country == data.CountryUSA {
		altCityID = "us-raleigh"
		altCityName = "Raleigh, NC"
	}

	specs := []leverSpec{
		{
			id:    "boost-savings-25",
			title: "Boost Monthly Investment by +25%",
			desc:  "Accelerates pre-FIRE compounding and raises your retirement safety cushion.",
			mutate: func(p PlanInput) PlanInput {
				p.MonthlySavings *= 1.25
				return p
			},
		},
		{
			id:    "delay-retire-2yr",
			title: "Extend Career Horizon by +2 Years",
			desc:  "Adds 2 years of peak compounding while cutting 2 early-withdrawal sequence-risk years.",
			mutate: func(p PlanInput) PlanInput {
				p.RetirementAge = minInt(p.HorizonAge-5, p.RetirementAge+2)
				return p
			},
		},
		{
			id:    "barista-bridge-5yr",
			title: "5-Year Coast / Consulting Bridge (35% Spend Cover)",
			desc:  "Part-time advisory or passion income during your first 5 FIRE years slashes early sequence risk.",
			mutate: func(p PlanInput) PlanInput {
				p.LifeEvents = append(slices.Clone(p.LifeEvents), LifeEvent{
					ID:          "lever-barista",
					Name:        "5-Yr Consulting Bridge",
					StartAge:    p.RetirementAge,
					EndAge:      p.RetirementAge + 4,
					AnnualReal:  p.DesiredMonthlySpend * 12.0 * 0.35,
					IsTaxableIn: true,
				})
				return p
			},
		},
		{
			id:    "trim-spend-12",
			title: "Optimize Baseline Retirement Spend by -12%",
			desc:  "Permanently lowers both annual portfolio withdrawals and capital-gains/MAGI tax drag.",
			mutate: func(p PlanInput) PlanInput {
				p.DesiredMonthlySpend *= 0.88
				return p
			},
		},
		{
			id:    "smart-guardrails-tent",
			title: "Enable Guyton-Klinger Guardrails + ERN Bond Tent + Tax Alpha",
			desc:  "Combines rising-equity bond tent, dynamic bear-market belt-tightening, and Roth/LTCG tax optimization.",
			mutate: func(p PlanInput) PlanInput {
				p.WithdrawalStrategy = StrategyGuytonKlinger
				p.GuardrailFloorPct = 0.80
				p.Allocation.UseBondTent = true
				p.EnableTaxOptimization = true
				return p
			},
		},
	}

	if in.RetirementCityID == in.CurrentCityID && currCity.Tier == "Tier 1" {
		specs = append(specs, leverSpec{
			id:    "geo-arbitrage-tier2",
			title: fmt.Sprintf("Geo-Arbitrage: Accumulate in %s → Retire in %s", currCity.Name, altCityName),
			desc:  "Unlocks Tier-2 housing/lifestyle scaling and lower state/regional healthcare overhead at retirement.",
			mutate: func(p PlanInput) PlanInput {
				p.RetirementCityID = altCityID
				p.AutoScaleLifestyleOnMove = true
				return p
			},
		})
	}

	baseCRNRate := probeSuccessOnly(in, scenarios)
	out := make([]ActionLever, len(specs))
	var wg sync.WaitGroup
	for i, sp := range specs {
		wg.Add(1)
		go func(idx int, s leverSpec) {
			defer wg.Done()
			mod := s.mutate(in)
			crnRate := probeSuccessOnly(mod, scenarios)
			delta := crnRate - baseCRNRate
			newRate := clamp(baseRate+delta, 0.0, 100.0)
			newAge := baseTargetAge
			if s.id != "delay-retire-2yr" {
				newAge = solveTargetAgeQuickCRN(mod, 90.0, scenarios)
			}
			out[idx] = ActionLever{
				ID:               s.id,
				Title:            s.title,
				Description:      s.desc,
				NewSuccessRate:   round2(newRate),
				SuccessDeltaPct:  round2(newRate - baseRate),
				NewTargetFIREAge: round1(newAge),
				YearsSaved:       round1(math.Max(0, baseTargetAge-newAge)),
			}
		}(i, sp)
	}
	wg.Wait()
	return out
}

func computeCrisisReplays(in PlanInput) []CrisisReplayOutcome {
	crises := data.CrisesByCountry(in.Country)
	history := data.HistoryByCountry(in.Country)
	currCity := data.GetCity(in.CurrentCityID, in.Country)
	retCity := data.GetCity(in.RetirementCityID, in.Country)
	numYears := in.HorizonAge - in.CurrentAge + 1

	// Anchor crisis StartYear right at the user's RetirementAge so we test retiring ON THE EVE of each famous crisis!
	retOffset := in.RetirementAge - in.CurrentAge
	out := make([]CrisisReplayOutcome, len(crises))

	for i, cr := range crises {
		crisisIdx := 0
		for k, h := range history {
			if h.Year == cr.StartYear {
				crisisIdx = k
				break
			}
		}
		startIdx := (crisisIdx - retOffset) % len(history)
		if startIdx < 0 {
			startIdx += len(history)
		}
		scenario := make([]data.MarketYear, numYears)
		for y := 0; y < numYears; y++ {
			scenario[y] = history[(startIdx+y)%len(history)]
		}

		probe := in
		probe.Mode = ModeHistorical
		probe.Stress.EnableCustomShock = false
		res := simulateSinglePath(probe, currCity, retCity, scenario)

		series := make([]float64, numYears)
		minW := math.Inf(1)
		minAge := in.RetirementAge
		for y := 0; y < numYears; y++ {
			w := math.Round(res.years[y].realWealth)
			series[y] = w
			age := in.CurrentAge + y
			if age >= in.RetirementAge && w < minW {
				minW = w
				minAge = age
			}
		}
		if math.IsInf(minW, 1) {
			minW = 0
		}

		out[i] = CrisisReplayOutcome{
			Crisis:           cr,
			Survived:         res.survived,
			DepletionAge:     res.depletionAge,
			MinRealWealth:    minW,
			MinWealthAge:     minAge,
			TerminalReal:     math.Round(res.terminalReal),
			Worst5YrRealRet:  round2(res.first5YrRealRet * 100.0),
			RealWealthSeries: series,
		}
	}
	return out
}

func buildSRRHeatmap(country data.CountryCode, outcomes []singlePathOutcome) []SRRCell {
	// 5 First-5-Year Real Return buckets x 4 First-5-Year Inflation Regime buckets
	retEdges := []float64{-0.06, -0.01, 0.03, 0.07}
	retLabels := []string{"<-6%/yr Crash", "-6% to -1%", "-1% to +3%", "+3% to +7%", ">+7%/yr Bull"}

	var infEdges []float64
	var infLabels []string
	if country == data.CountryIndia {
		infEdges = []float64{0.05, 0.075, 0.10}
		infLabels = []string{"<5% Benign CPI", "5%–7.5% Normal", "7.5%–10% Elevated", ">10% Stagflation"}
	} else {
		infEdges = []float64{0.02, 0.038, 0.06}
		infLabels = []string{"<2% Low CPI", "2%–3.8% Moderate", "3.8%–6% Sticky", ">6% Stagflation"}
	}

	var counts [5][4]int
	var survs [5][4]int

	for _, o := range outcomes {
		rIdx := len(retEdges)
		for i, edge := range retEdges {
			if o.first5YrRealRet < edge {
				rIdx = i
				break
			}
		}
		iIdx := len(infEdges)
		for j, edge := range infEdges {
			if o.first5YrAvgCPI < edge {
				iIdx = j
				break
			}
		}
		counts[rIdx][iIdx]++
		if o.survived {
			survs[rIdx][iIdx]++
		}
	}

	cells := make([]SRRCell, 0, 20)
	for r := 0; r < 5; r++ {
		for c := 0; c < 4; c++ {
			pct := 0.0
			if counts[r][c] > 0 {
				pct = round2(100.0 * float64(survs[r][c]) / float64(counts[r][c]))
			}
			cells = append(cells, SRRCell{
				RetBucketIdx:   r,
				InfBucketIdx:   c,
				RetBucketLabel: retLabels[r],
				InfBucketLabel: infLabels[c],
				PathCount:      counts[r][c],
				SurvivalPct:    pct,
			})
		}
	}
	return cells
}

func selectSamplePaths(startAge int, outcomes []singlePathOutcome) []SampleTrajectory {
	n := len(outcomes)
	if n == 0 {
		return nil
	}
	idxByTerm := make([]int, n)
	for i := range idxByTerm {
		idxByTerm[i] = i
	}
	slices.SortFunc(idxByTerm, func(a, b int) int {
		if outcomes[a].terminalReal != outcomes[b].terminalReal {
			if outcomes[a].terminalReal < outcomes[b].terminalReal {
				return -1
			}
			return 1
		}
		// Tie-break earlier depletion age as worse
		return outcomes[a].depletionAge - outcomes[b].depletionAge
	})

	type pick struct {
		rank  int
		label string
	}
	keyPicks := []pick{
		{0, "Worst Tail Path"},
		{maxInt(0, int(float64(n-1)*0.10)), "P10 Stress Path"},
		{maxInt(0, int(float64(n-1)*0.25)), "P25 Conservative Path"},
		{maxInt(0, int(float64(n-1)*0.50)), "Median (P50) Path"},
		{maxInt(0, int(float64(n-1)*0.75)), "P75 Strong Path"},
		{maxInt(0, int(float64(n-1)*0.90)), "P90 Bull Path"},
	}

	used := make(map[int]bool)
	var samples []SampleTrajectory

	appendSample := func(origIdx int, label string) {
		if used[origIdx] {
			return
		}
		used[origIdx] = true
		o := outcomes[origIdx]
		steps := make([]TrajectoryStep, len(o.years))
		for y, rec := range o.years {
			steps[y] = TrajectoryStep{
				Age:               startAge + y,
				HistoricalRefYear: rec.refYear,
				EquityReturn:      round2(rec.eqRet * 100.0),
				DebtReturn:        round2(rec.debtRet * 100.0),
				HeadlineCPI:       round2(rec.cpi * 100.0),
				RealWealth:        math.Round(rec.realWealth),
				LiquidRealWealth:  math.Round(rec.liquidReal),
				RealAnnualSpend:   math.Round(rec.coreLivingOutReal),
				RealHealthcare:    math.Round(rec.healthcareOutReal),
				RealTaxes:         math.Round(rec.taxesOutReal),
			}
		}
		samples = append(samples, SampleTrajectory{
			PathID:       origIdx + 1,
			Label:        label,
			Survived:     o.survived,
			DepletionAge: o.depletionAge,
			TerminalReal: math.Round(o.terminalReal),
			Steps:        steps,
		})
	}

	for _, kp := range keyPicks {
		appendSample(idxByTerm[kp.rank], kp.label)
	}

	// Add 18 evenly spaced spaghetti paths across the distribution
	step := maxInt(1, n/20)
	for i := step / 2; i < n && len(samples) < 24; i += step {
		origIdx := idxByTerm[i]
		appendSample(origIdx, fmt.Sprintf("Simulated Path #%d", origIdx+1))
	}

	return samples
}

func medianMetric(outcomes []singlePathOutcome, yearIdx int, fn func(pathYearRecord) float64) float64 {
	buf := make([]float64, len(outcomes))
	for i := range outcomes {
		buf[i] = fn(outcomes[i].years[yearIdx])
	}
	slices.Sort(buf)
	return math.Round(quantileSorted(buf, 0.50))
}

func round1(v float64) float64 {
	return math.Round(v*10.0) / 10.0
}

func round2(v float64) float64 {
	return math.Round(v*100.0) / 100.0
}
