package engine

import (
	"math"
	"math/rand/v2"
	"runtime"
	"slices"
	"sync"

	"github.com/ausanyal/firesage/internal/data"
	"github.com/ausanyal/firesage/internal/tax"
)

const baseSimulationYear = 2026

// StatutoryUnlockAge returns the statutory penalty-free retirement fund preservation/unlock age
// for the accumulation jurisdiction (Age 65 for NZ KiwiSaver; Age 60 for IN NPS, US 401k/IRA, AU Super, EU Pillar 2/3a).
func StatutoryUnlockAge(country data.CountryCode) int {
	if country == data.CountryNewZealand {
		return 65
	}
	return 60
}

// NormalizePlanInput fills defaults and clamps parameters into safe, consistent bounds.
func NormalizePlanInput(in PlanInput) PlanInput {
	if !data.IsValidCountry(in.Country) {
		in.Country = data.CountryIndia
	}
	if in.CurrentAge < 18 {
		in.CurrentAge = 32
	}
	if in.CurrentAge > 85 {
		in.CurrentAge = 85
	}
	if in.RetirementAge < in.CurrentAge {
		if in.RetirementAge <= 0 {
			in.RetirementAge = maxInt(in.CurrentAge, minInt(in.CurrentAge+12, StatutoryUnlockAge(in.Country)))
		} else {
			in.RetirementAge = in.CurrentAge
		}
	}
	if in.HorizonAge <= in.RetirementAge+2 {
		in.HorizonAge = maxInt(92, in.RetirementAge+15)
	}
	if in.HorizonAge > 105 {
		in.HorizonAge = 105
	}

	currCity := data.GetCity(in.CurrentCityID, in.Country)
	in.CurrentCityID = currCity.ID
	if in.RetirementCityID == "" {
		in.RetirementCityID = currCity.ID
	} else {
		// Allow any-to-any global cross-country retirement (e.g., India/US/AU/EU/NZ -> any of 34 cities).
		retCity := data.GetCity(in.RetirementCityID, currCity.Country)
		in.RetirementCityID = retCity.ID
	}

	if in.CurrentCorpus < 0 {
		in.CurrentCorpus = 0
	}
	if in.MonthlySavings < 0 {
		in.MonthlySavings = 0
	}
	if in.MonthlyLockedSavings < 0 {
		in.MonthlyLockedSavings = 0
	}
	if in.DesiredMonthlySpend <= 0 {
		in.DesiredMonthlySpend = currCity.MonthlyPresets.Standard
	}
	in.AnnualSavingsStepUp = clamp(in.AnnualSavingsStepUp, 0.0, 0.25)

	// Normalize asset allocation weights
	sumW := in.Allocation.Equity + in.Allocation.Debt + in.Allocation.Gold + in.Allocation.Cash
	if sumW <= 0.001 {
		switch in.Country {
		case data.CountryIndia:
			in.Allocation = AssetAllocation{Equity: 0.65, Debt: 0.22, Gold: 0.08, Cash: 0.05, UseBondTent: true}
		case data.CountryEurope:
			in.Allocation = AssetAllocation{Equity: 0.68, Debt: 0.23, Gold: 0.04, Cash: 0.05, UseBondTent: true}
		default:
			in.Allocation = AssetAllocation{Equity: 0.72, Debt: 0.20, Gold: 0.03, Cash: 0.05, UseBondTent: true}
		}
	} else {
		in.Allocation.Equity = math.Max(0, in.Allocation.Equity) / sumW
		in.Allocation.Debt = math.Max(0, in.Allocation.Debt) / sumW
		in.Allocation.Gold = math.Max(0, in.Allocation.Gold) / sumW
		in.Allocation.Cash = math.Max(0, in.Allocation.Cash) / sumW
	}

	if in.ExpenseRatioBps < 0 || in.ExpenseRatioBps > 300 {
		in.ExpenseRatioBps = 20
	}
	if in.WithdrawalStrategy == "" {
		in.WithdrawalStrategy = StrategyGuytonKlinger
	}
	if in.GuardrailFloorPct <= 0 || in.GuardrailFloorPct > 1.0 {
		in.GuardrailFloorPct = 0.82
	}
	if in.GuardrailCeilingPct < 1.0 || in.GuardrailCeilingPct > 2.0 {
		in.GuardrailCeilingPct = 1.25
	}
	if in.SocialSecurityStartAge <= 0 {
		switch in.Country {
		case data.CountryUSA, data.CountryAustralia:
			in.SocialSecurityStartAge = 67
		case data.CountryEurope, data.CountryNewZealand:
			in.SocialSecurityStartAge = 65
		default:
			in.SocialSecurityStartAge = 60
		}
	}

	if !in.Buckets.UseGranular {
		in.Buckets = defaultBuckets(in.Country, in.CurrentCorpus)
	} else {
		if in.Buckets.TaxableCostBasis <= 0 || in.Buckets.TaxableCostBasis > 1.0 {
			in.Buckets.TaxableCostBasis = 0.65
		}
		in.CurrentCorpus = in.Buckets.TaxableEquity + in.Buckets.TaxableDebtCash +
			in.Buckets.TaxDeferredLocked + in.Buckets.TaxFreeAccount + in.Buckets.PensionLocked
	}

	if in.Mode == "" {
		in.Mode = ModeBootstrap
	}
	if in.NumPaths <= 0 {
		in.NumPaths = 3000
	}
	if in.NumPaths < 100 {
		in.NumPaths = 100
	}
	if in.NumPaths > 10000 {
		in.NumPaths = 10000
	}
	if in.Seed == 0 {
		in.Seed = 20261112
	}
	return in
}

func defaultBuckets(country data.CountryCode, corpus float64) AccountBuckets {
	switch country {
	case data.CountryUSA:
		return AccountBuckets{
			UseGranular:       false,
			TaxableEquity:     corpus * 0.40,
			TaxableCostBasis:  0.65,
			TaxableDebtCash:   corpus * 0.08,
			TaxDeferredLocked: corpus * 0.38, // 401(k) / Traditional IRA
			TaxFreeAccount:    corpus * 0.14, // Roth IRA / HSA
			PensionLocked:     0,
		}
	case data.CountryAustralia:
		return AccountBuckets{
			UseGranular:       false,
			TaxableEquity:     corpus * 0.38, // Personal ETF / ASX Franked Brokerage
			TaxableCostBasis:  0.65,
			TaxableDebtCash:   corpus * 0.08, // High-Interest Cash / Offset
			TaxDeferredLocked: corpus * 0.54, // Superannuation (Preservation Age 60, 100% tax-free post-60)
			TaxFreeAccount:    0,
			PensionLocked:     0,
		}
	case data.CountryEurope:
		return AccountBuckets{
			UseGranular:       false,
			TaxableEquity:     corpus * 0.44, // UCITS ETF Brokerage
			TaxableCostBasis:  0.65,
			TaxableDebtCash:   corpus * 0.10, // Sovereign Bonds / Money Market
			TaxDeferredLocked: corpus * 0.32, // Swiss Pillar 2 (BVG) / Occupational Pension
			TaxFreeAccount:    corpus * 0.14, // Pillar 3a / Assurance-Vie / PEA Wrapper
			PensionLocked:     0,
		}
	case data.CountryNewZealand:
		return AccountBuckets{
			UseGranular:       false,
			TaxableEquity:     corpus * 0.46, // Liquid PIE / Global Equity Funds
			TaxableCostBasis:  0.65,
			TaxableDebtCash:   corpus * 0.10, // NZ Term Deposits / Cash PIE
			TaxDeferredLocked: corpus * 0.44, // KiwiSaver (Locked until Age 65, 100% tax-free unlock)
			TaxFreeAccount:    0,
			PensionLocked:     0,
		}
	default:
		return AccountBuckets{
			UseGranular:       false,
			TaxableEquity:     corpus * 0.48, // Equity + Arbitrage Mutual Funds
			TaxableCostBasis:  0.65,
			TaxableDebtCash:   corpus * 0.14, // Liquid / Debt Funds / FDs
			TaxDeferredLocked: corpus * 0.20, // EPF / VPF
			TaxFreeAccount:    corpus * 0.10, // PPF (EEE)
			PensionLocked:     corpus * 0.08, // NPS Tier-I (locked till 60)
		}
	}
}

type pathYearRecord struct {
	realWealth       float64
	nomWealth        float64
	liquidReal       float64
	lockedReal       float64
	monthlySpendReal float64

	savingsInReal     float64
	portfolioDrawReal float64
	guaranteedInReal  float64
	milestoneInReal   float64
	rothOrHarvestReal float64
	coreLivingOutReal float64
	healthcareOutReal float64
	taxesOutReal      float64
	milestoneOutReal  float64

	hadLiqCrunch bool
	refYear      int
	eqRet        float64
	debtRet      float64
	cpi          float64
}

type singlePathOutcome struct {
	survived           bool
	depletionAge       int
	hadLiquidityCrunch bool
	firstCrunchAge     int
	terminalReal       float64
	corpusAtFireReal   float64
	corpusAtFireNom    float64
	first5YrRealRet    float64
	first5YrAvgCPI     float64
	years              []pathYearRecord
}

// runMonteCarloPaths executes nPaths concurrent simulations using a worker pool.
func runMonteCarloPaths(in PlanInput, nPaths int) []singlePathOutcome {
	history := data.HistoryByCountry(in.Country)
	currCity := data.GetCity(in.CurrentCityID, in.Country)
	retCity := data.GetCity(in.RetirementCityID, in.Country)

	if in.Mode == ModeHistorical {
		nPaths = len(history)
	}

	outcomes := make([]singlePathOutcome, nPaths)
	workers := runtime.NumCPU()
	if workers < 2 {
		workers = 2
	}
	if workers > nPaths {
		workers = nPaths
	}

	var wg sync.WaitGroup
	chunk := (nPaths + workers - 1) / workers
	for w := 0; w < workers; w++ {
		start := w * chunk
		end := minInt(start+chunk, nPaths)
		if start >= end {
			continue
		}
		wg.Add(1)
		go func(s, e int) {
			defer wg.Done()
			numYears := in.HorizonAge - in.CurrentAge + 1
			for p := s; p < e; p++ {
				rng := rand.New(rand.NewPCG(in.Seed+uint64(p)*7919, uint64(p)*104729+17))
				scenario := generatePathScenario(in.Mode, in.Country, history, numYears, p, rng)
				outcomes[p] = simulateSinglePath(in, currCity, retCity, scenario)
			}
		}(start, end)
	}
	wg.Wait()
	return outcomes
}

func simulateSinglePath(
	in PlanInput,
	currCity data.CityProfile,
	retCity data.CityProfile,
	scenario []data.MarketYear,
) singlePathOutcome {
	return simulatePathInternal(in, currCity, retCity, scenario, true)
}

func simulateFastProbe(
	in PlanInput,
	currCity data.CityProfile,
	retCity data.CityProfile,
	scenario []data.MarketYear,
) (survived bool, corpusAtFireReal float64) {
	out := simulatePathInternal(in, currCity, retCity, scenario, false)
	return out.survived, out.corpusAtFireReal
}

func pregenerateProbeScenarios(in PlanInput, maxProbe int) [][]data.MarketYear {
	history := data.HistoryByCountry(in.Country)
	mode := in.Mode
	nTotal := in.NumPaths
	if mode == ModeHistorical {
		nTotal = len(history)
	}
	nProbe := minInt(nTotal, maxProbe)
	if nProbe < 1 {
		nProbe = 1
	}
	numYears := in.HorizonAge - in.CurrentAge + 1
	scenarios := make([][]data.MarketYear, nProbe)
	for k := 0; k < nProbe; k++ {
		p := (k * nTotal) / nProbe
		rng := rand.New(rand.NewPCG(in.Seed+uint64(p)*7919, uint64(p)*104729+17))
		scenarios[k] = generatePathScenario(mode, in.Country, history, numYears, p, rng)
	}
	return scenarios
}

func probeSuccessAndMedianFire(in PlanInput, scenarios [][]data.MarketYear) (rate float64, medFire float64) {
	n := len(scenarios)
	if n == 0 {
		return 0, 0
	}
	currCity := data.GetCity(in.CurrentCityID, in.Country)
	retCity := data.GetCity(in.RetirementCityID, in.Country)
	surv := 0
	fireVals := make([]float64, n)
	for i := 0; i < n; i++ {
		ok, fReal := simulateFastProbe(in, currCity, retCity, scenarios[i])
		if ok {
			surv++
		}
		fireVals[i] = fReal
	}
	slices.Sort(fireVals)
	return 100.0 * float64(surv) / float64(n), quantileSorted(fireVals, 0.50)
}

func probeSuccessOnly(in PlanInput, scenarios [][]data.MarketYear) float64 {
	return probeSuccessWithInitMed(in, scenarios, nil)
}

func probeSuccessWithInitMed(in PlanInput, scenarios [][]data.MarketYear, initMedRatios []float64) float64 {
	n := len(scenarios)
	if n == 0 {
		return 0
	}
	currCity := data.GetCity(in.CurrentCityID, in.Country)
	retCity := data.GetCity(in.RetirementCityID, in.Country)
	surv := 0
	for i := 0; i < n; i++ {
		p := in
		if i < len(initMedRatios) && initMedRatios[i] > 0 {
			p.initMedIndex = initMedRatios[i]
		}
		ok, _ := simulateFastProbe(p, currCity, retCity, scenarios[i])
		if ok {
			surv++
		}
	}
	return 100.0 * float64(surv) / float64(n)
}

func simulatePathInternal(
	in PlanInput,
	currCity data.CityProfile,
	retCity data.CityProfile,
	scenario []data.MarketYear,
	recordYears bool,
) singlePathOutcome {
	numYears := in.HorizonAge - in.CurrentAge + 1
	var records []pathYearRecord
	if recordYears {
		records = make([]pathYearRecord, numYears)
	}

	// Initialize nominal account buckets at t = 0
	taxableEq := in.Buckets.TaxableEquity
	taxableBasis := taxableEq * clamp(in.Buckets.TaxableCostBasis, 0.10, 1.0)
	taxableDebt := in.Buckets.TaxableDebtCash
	taxDeferred := in.Buckets.TaxDeferredLocked // US 401k / IN EPF
	taxFree := in.Buckets.TaxFreeAccount        // US Roth / IN PPF
	pensionLocked := in.Buckets.PensionLocked   // IN NPS Tier-I

	// US Roth Conversion 5-year seasoning queue: rothSeasoned can be withdrawn penalty-free pre-59.5;
	// rothRungs[0] matures next year ... rothRungs[4] was converted this year.
	rothSeasoned := taxFree * 0.55 // Principal basis portion of existing Roth/PPF
	rothGrowthLocked := taxFree - rothSeasoned
	var rothRungs [5]float64

	cpiIndex := 1.0
	medIndex := 1.0
	if in.initMedIndex > 0 {
		medIndex = in.initMedIndex
	}
	feeDrag := in.ExpenseRatioBps / 10_000.0
	birthYear := baseSimulationYear - in.CurrentAge
	_, medianPE := data.CurrentValuation(in.Country)

	// Compute target baseline annual real spend in retirement
	colScale := 1.0
	if in.AutoScaleLifestyleOnMove && retCity.ID != currCity.ID {
		colScale = EffectiveCOLScale(currCity, retCity)
	}
	baseRetireAnnualReal := in.DesiredMonthlySpend * 12.0 * colScale

	survived := true
	depletionAge := 0
	hadLiquidityCrunch := false
	firstCrunchAge := 0
	var corpusAtFireReal, corpusAtFireNom float64
	npsAnnuityAnnualNom := 0.0

	// Track first 5 retirement years for Sequence-of-Returns Risk (SRR) Heatmap
	ret5Prod := 1.0
	cpi5Sum := 0.0
	ret5Count := 0

	initWR := 0.04

	for y := 0; y < numYears; y++ {
		age := in.CurrentAge + y
		isRetired := age >= in.RetirementAge
		activeCity := currCity
		if isRetired {
			activeCity = retCity
		}

		mkt := scenario[y]
		eqRet := mkt.Equity
		debtRet := mkt.Debt
		cashRet := mkt.Cash
		goldRet := mkt.Gold
		cpi := mkt.HeadlineCPI
		medCPI := mkt.MedicalCPI

		// Overlay user Stress Test shocks if enabled
		if in.Mode == ModeStressTest || in.Stress.EnableCustomShock {
			yearsIntoFire := age - in.RetirementAge
			if yearsIntoFire == 0 && in.Stress.RetireYear1CrashPct != 0 {
				eqRet = clamp(in.Stress.RetireYear1CrashPct, -0.75, 0.50)
			}
			if yearsIntoFire >= 0 && yearsIntoFire < in.Stress.ExtraInflationYears && in.Stress.ExtraInflationPct != 0 {
				cpi += in.Stress.ExtraInflationPct
				medCPI += in.Stress.ExtraInflationPct
			}
		}

		// Record wealth at start of retirement age before Year-1 decumulation
		if age == in.RetirementAge {
			startNom := taxableEq + taxableDebt + taxDeferred + rothSeasoned + rothGrowthLocked + sumRungs(rothRungs) + pensionLocked
			corpusAtFireNom = startNom
			corpusAtFireReal = startNom / cpiIndex
			if corpusAtFireReal > 0 {
				initWR = clamp(baseRetireAnnualReal/corpusAtFireReal, 0.02, 0.08)
			}
		}

		// Advance 5-year Roth Conversion seasoning queue at start of year
		rothSeasoned += rothRungs[0]
		rothRungs[0] = rothRungs[1]
		rothRungs[1] = rothRungs[2]
		rothRungs[2] = rothRungs[3]
		rothRungs[3] = rothRungs[4]
		rothRungs[4] = 0

		// India Age 60 NPS Tier-I Unlock: 60% tax-free lump sum + 40% taxable annuity (~6.6% yield)
		if in.Country == data.CountryIndia && age == 60 && pensionLocked > 0 {
			taxFreeLump := pensionLocked * 0.60
			annuityCorpus := pensionLocked * 0.40
			taxableDebt += taxFreeLump
			npsAnnuityAnnualNom += annuityCorpus * 0.066
			pensionLocked = 0
		}

		// Determine dynamic glidepath (ERN Bond Tent around RetirementAge if enabled)
		eqW := in.Allocation.Equity
		debtW := in.Allocation.Debt
		goldW := in.Allocation.Gold
		cashW := in.Allocation.Cash
		if in.Allocation.UseBondTent {
			distFromFire := age - in.RetirementAge
			var tentShift float64
			if distFromFire >= -3 && distFromFire < 0 {
				// Ramp bond tent over final 3 accumulation years
				tentShift = 0.18 * float64(3+distFromFire) / 3.0
			} else if distFromFire >= 0 && distFromFire <= 10 {
				// Glide back into equities over first 10 years of retirement
				tentShift = 0.18 * (1.0 - float64(distFromFire)/10.0)
			}
			tentShift = math.Min(tentShift, eqW*0.5)
			eqW -= tentShift
			debtW += tentShift * 0.70
			cashW += tentShift * 0.30
		}

		// Blended portfolio return vs defensive debt/cash return
		growthBlendRet := eqW*eqRet + goldW*goldRet + debtW*debtRet + cashW*cashRet - feeDrag
		eqGoldBlendRet := eqRet - feeDrag
		if eqW+goldW > 0.01 {
			eqGoldBlendRet = (eqW*eqRet+goldW*goldRet)/(eqW+goldW) - feeDrag
		}
		defensiveBlendRet := debtRet - feeDrag*0.5
		if debtW+cashW > 0.01 {
			defensiveBlendRet = (debtW*debtRet+cashW*cashRet)/(debtW+cashW) - feeDrag*0.5
		}

		// Compound headline and medical inflation over year y (age -> age+1)
		cpiIndex *= (1.0 + cpi)
		medIndex *= (1.0 + medCPI)
		yearsElapsed := in.preRetireYears + y + 1

		// Track first 5 years of retirement returns for SRR Heatmap
		if isRetired && ret5Count < 5 {
			portReal := (1.0 + growthBlendRet) / math.Max(0.5, 1.0+cpi)
			ret5Prod *= portReal
			cpi5Sum += cpi
			ret5Count++
		}

		// Grow buckets for the year
		if taxableEq > 0 {
			// In New Zealand during accumulation, ongoing PIE/FIF tax drag applies annually on global shares (~0.9% drag)
			nzAccFIFDrag := 0.0
			if !isRetired && in.Country == data.CountryNewZealand && taxableEq > 80_000*cpiIndex {
				nzAccFIFDrag = 0.009
			}
			taxableEq *= (1.0 + eqGoldBlendRet - nzAccFIFDrag)
		}
		if taxableDebt > 0 {
			taxableDebt *= (1.0 + defensiveBlendRet)
		}
		if taxDeferred > 0 {
			switch in.Country {
			case data.CountryIndia:
				// EPF sovereign-backed debt-like steady return (~7.8%–8.4% linked to debt/cash regime)
				epfRate := clamp(0.65*debtRet+0.35*cashRet+0.008, 0.055, 0.095)
				taxDeferred *= (1.0 + epfRate)
			case data.CountryAustralia:
				// Australian Superannuation: ~0.60% concessional fund earnings tax drag before age 60, 0% in retirement pension phase (60+)
				superDrag := 0.0
				if age < 60 || !isRetired {
					superDrag = 0.006
				}
				taxDeferred *= (1.0 + growthBlendRet - superDrag)
			case data.CountryNewZealand:
				// NZ KiwiSaver PIE multi-rate fund (~0.85% internal PIE/FIF tax drag; 100% tax-free on withdrawal at 65)
				taxDeferred *= (1.0 + growthBlendRet - 0.0085)
			default:
				taxDeferred *= (1.0 + growthBlendRet)
			}
		}
		if rothSeasoned+rothGrowthLocked+sumRungs(rothRungs) > 0 {
			if in.Country == data.CountryIndia {
				// PPF EEE sovereign rate (~7.1%–8.0%)
				ppfRate := clamp(0.50*debtRet+0.50*cashRet+0.005, 0.055, 0.088)
				rothGrowthLocked += (rothSeasoned + rothGrowthLocked) * ppfRate
			} else {
				// US Roth / EU Pillar 3a wrapper grows tax-free; growth accrues to rothGrowthLocked until unlock
				totRoth := rothSeasoned + rothGrowthLocked + sumRungs(rothRungs)
				gain := totRoth * growthBlendRet
				if gain >= 0 {
					rothGrowthLocked += gain
				} else {
					// Proportional drawdown across Roth sub-ledgers
					scale := math.Max(0, 1.0+growthBlendRet)
					rothSeasoned *= scale
					rothGrowthLocked *= scale
					for r := range rothRungs {
						rothRungs[r] *= scale
					}
				}
			}
		}
		if pensionLocked > 0 {
			pensionLocked *= (1.0 + growthBlendRet)
		}

		// Rebalance liquid taxable Equity vs Debt toward target allocation ratio
		liquidTot := taxableEq + taxableDebt
		if liquidTot > 0 {
			targetEqShare := clamp(eqW+goldW, 0.15, 0.92)
			desiredEq := liquidTot * targetEqShare
			deltaEq := desiredEq - taxableEq
			if deltaEq > 0 {
				// Buy equity from debt/cash -> increases basis dollar-for-dollar
				move := math.Min(deltaEq, taxableDebt)
				taxableDebt -= move
				taxableEq += move
				taxableBasis += move
			}
		}

		// Evaluate custom Life Events active at this age
		milestoneInNom := 0.0
		milestoneTaxableInNom := 0.0
		milestoneOutNom := 0.0
		for _, ev := range in.LifeEvents {
			endAge := ev.EndAge
			if endAge < ev.StartAge {
				endAge = ev.StartAge
			}
			if age >= ev.StartAge && age <= endAge {
				amtNom := ev.AnnualReal * cpiIndex
				if amtNom >= 0 {
					milestoneInNom += amtNom
					if ev.IsTaxableIn {
						milestoneTaxableInNom += amtNom
					}
				} else {
					milestoneOutNom += -amtNom
				}
			}
		}
		if (in.Mode == ModeStressTest || in.Stress.EnableCustomShock) &&
			in.Stress.MedicalEmergencyReal > 0 && age == in.Stress.MedicalEmergencyAge {
			milestoneOutNom += in.Stress.MedicalEmergencyReal * medIndex
		}

		var savingsInNom, portfolioDrawNom, guaranteedInNom, rothOrHarvestNom float64
		var coreLivingOutNom, healthcareOutNom, taxesOutNom float64
		crunchThisYear := false
		unlockAge := StatutoryUnlockAge(in.Country)

		if !isRetired {
			// ===================== ACCUMULATION PHASE =====================
			yearsWorked := age - in.CurrentAge
			savMult := math.Pow(1.0+in.AnnualSavingsStepUp, float64(yearsWorked))
			if in.stopSavingsAge > 0 && age >= in.stopSavingsAge {
				savMult = 0
			}

			if in.MonthlyLockedSavings > 0 || in.Buckets.UseGranular {
				liqSavNom := in.MonthlySavings * 12.0 * savMult
				lockedSavNom := in.MonthlyLockedSavings * 12.0 * savMult
				savingsInNom = liqSavNom + lockedSavNom

				netLiq := liqSavNom + milestoneInNom - milestoneOutNom
				if netLiq >= 0 {
					eqShare := clamp(in.Allocation.Equity+in.Allocation.Gold, 0.40, 0.90)
					toTaxableEq := netLiq * eqShare
					toTaxableDebt := netLiq * (1.0 - eqShare)
					taxableEq += toTaxableEq
					taxableBasis += toTaxableEq
					taxableDebt += toTaxableDebt
				} else {
					deficit := -netLiq
					drawn, _ := drawFromLiquid(&taxableDebt, &taxableEq, &taxableBasis, deficit)
					rem := deficit - drawn
					if rem > 0 {
						pull := math.Min(rem, rothSeasoned)
						rothSeasoned -= pull
					}
				}

				if lockedSavNom > 0 {
					switch in.Country {
					case data.CountryUSA:
						taxDeferred += lockedSavNom * 0.68
						rothSeasoned += lockedSavNom * 0.32
					case data.CountryAustralia:
						// Superannuation concessional contributions incur 15% entry tax inside the Super fund
						taxDeferred += lockedSavNom * 0.85
					case data.CountryEurope:
						// Split across Pillar 2 Occupational Pension (68%) + Pillar 3a / tax-advantaged wrapper (32%)
						taxDeferred += lockedSavNom * 0.68
						rothGrowthLocked += lockedSavNom * 0.32
					case data.CountryNewZealand:
						// KiwiSaver (after-tax / ESCT-netted contributions, preserved until Age 65)
						taxDeferred += lockedSavNom
					default:
						taxDeferred += lockedSavNom * 0.5625
						rothSeasoned += lockedSavNom * 0.21875
						pensionLocked += lockedSavNom * 0.21875
					}
				}
			} else {
				savingsInNom = in.MonthlySavings * 12.0 * savMult

				netAddition := savingsInNom + milestoneInNom - milestoneOutNom
				if netAddition >= 0 {
					switch in.Country {
					case data.CountryUSA:
						toTaxableEq := netAddition * 0.42
						toTaxableDebt := netAddition * 0.08
						to401k := netAddition * 0.34
						toRoth := netAddition * 0.16
						taxableEq += toTaxableEq
						taxableBasis += toTaxableEq
						taxableDebt += toTaxableDebt
						taxDeferred += to401k
						rothSeasoned += toRoth
					case data.CountryAustralia:
						toTaxableEq := netAddition * 0.50
						toTaxableDebt := netAddition * 0.10
						toSuper := netAddition * 0.40 * 0.85 // 15% Super entry tax
						taxableEq += toTaxableEq
						taxableBasis += toTaxableEq
						taxableDebt += toTaxableDebt
						taxDeferred += toSuper
					case data.CountryEurope:
						toTaxableEq := netAddition * 0.50
						toTaxableDebt := netAddition * 0.10
						taxableEq += toTaxableEq
						taxableBasis += toTaxableEq
						taxableDebt += toTaxableDebt
						taxDeferred += netAddition * 0.28
						rothGrowthLocked += netAddition * 0.12
					case data.CountryNewZealand:
						toTaxableEq := netAddition * 0.58
						toTaxableDebt := netAddition * 0.12
						taxableEq += toTaxableEq
						taxableBasis += toTaxableEq
						taxableDebt += toTaxableDebt
						taxDeferred += netAddition * 0.30
					default:
						toTaxableEq := netAddition * 0.56
						toTaxableDebt := netAddition * 0.12
						toEPF := netAddition * 0.18
						toPPF := netAddition * 0.07
						toNPS := netAddition * 0.07
						taxableEq += toTaxableEq
						taxableBasis += toTaxableEq
						taxableDebt += toTaxableDebt
						taxDeferred += toEPF
						rothSeasoned += toPPF
						pensionLocked += toNPS
					}
				} else {
					deficit := -netAddition
					drawn, _ := drawFromLiquid(&taxableDebt, &taxableEq, &taxableBasis, deficit)
					rem := deficit - drawn
					if rem > 0 {
						pull := math.Min(rem, rothSeasoned)
						rothSeasoned -= pull
					}
				}
			}

			// India annual ₹1.25L tax-free LTCG gain harvesting even during accumulation!
			if in.Country == data.CountryIndia && in.EnableTaxOptimization && taxableEq > taxableBasis {
				harvest := math.Min(taxableEq-taxableBasis, tax.IndiaAnnualLTCGExemption(cpiIndex))
				taxableBasis += harvest
				rothOrHarvestNom = harvest
			}

			coreLivingOutNom = in.DesiredMonthlySpend * 12.0 * cpiIndex
		} else {
			// ===================== RETIREMENT / DECUMULATION PHASE =====================
			yearsInRet := age - in.RetirementAge
			totalStartNom := taxableEq + taxableDebt + taxDeferred + rothSeasoned + rothGrowthLocked + sumRungs(rothRungs) + pensionLocked
			totalStartReal := totalStartNom / cpiIndex

			// 1. Compute Core Living Spend in today's purchasing power per active WithdrawalStrategy
			targetSpendReal := baseRetireAnnualReal
			if in.UseSpendingSmile {
				if age >= 58 && age <= 78 {
					smileMult := math.Max(0.86, 1.0-0.007*float64(minInt(yearsInRet, 20)))
					targetSpendReal *= smileMult
				} else if age > 78 {
					targetSpendReal *= 0.86
				}
			}

			switch in.WithdrawalStrategy {
			case StrategyGuytonKlinger:
				if totalStartReal > 0 {
					currWR := targetSpendReal / totalStartReal
					if currWR > initWR*1.20 {
						ratio := (initWR * 1.20) / currWR
						targetSpendReal *= clamp(ratio, in.GuardrailFloorPct, 1.0)
					} else if currWR < initWR*0.78 {
						ratio := (initWR * 0.78) / math.Max(0.005, currWR)
						targetSpendReal *= clamp(ratio, 1.0, in.GuardrailCeilingPct)
					}
				} else {
					targetSpendReal *= in.GuardrailFloorPct
				}
			case StrategyCAPEDynamic:
				if mkt.ValuationPE > 0 && medianPE > 0 {
					capeAdj := math.Sqrt(medianPE / mkt.ValuationPE)
					targetSpendReal *= clamp(capeAdj, in.GuardrailFloorPct, in.GuardrailCeilingPct)
				}
			}

			coreLivingOutNom = targetSpendReal * cpiIndex

			// 2. Guaranteed Income Floors (Social Security / State Pension / NPS Annuity / AU Centrelink Age Pension)
			if age >= in.SocialSecurityStartAge && in.SocialSecurityMonthlyReal > 0 {
				guaranteedInNom += in.SocialSecurityMonthlyReal * 12.0 * cpiIndex
			}
			guaranteedInNom += npsAnnuityAnnualNom

			// Australia automatic means-tested Centrelink Age Pension from Age 67 (for AU citizens/residents retiring in AU)
			if in.Country == data.CountryAustralia && activeCity.Country == data.CountryAustralia && age >= 67 {
				assessableAUD := data.ConvertCurrency(totalStartNom, in.Country, data.CountryAustralia)
				otherIncomeAUD := data.ConvertCurrency(guaranteedInNom+milestoneTaxableInNom, in.Country, data.CountryAustralia)
				auPensionAUD := tax.AustraliaAgePensionWithIncomeTest(age, in.IsCouple, assessableAUD, otherIncomeAUD, cpiIndex)
				auPensionNom := data.ConvertCurrency(auPensionAUD, data.CountryAustralia, in.Country)
				guaranteedInNom += auPensionNom
			}

			// 3. Country-specific Healthcare, RMD, Tax-Smart Drawdown & Unlock Sequencing
			switch in.Country {
			case data.CountryUSA:
				ordinaryIncome := milestoneTaxableInNom + guaranteedInNom*0.85 // Up to 85% of SS taxable
				realizedLTCG := 0.0

				// Enforce SECURE 2.0 Required Minimum Distributions (Age 73 or 75)
				rmdDiv := tax.USRMDDivisor(age, birthYear)
				if rmdDiv > 0 && taxDeferred > 0 {
					mandatoryRMD := taxDeferred / rmdDiv
					taxDeferred -= mandatoryRMD
					taxableDebt += mandatoryRMD
					ordinaryIncome += mandatoryRMD
				}

				estHealth := computeResidencyHealthcareNom(in.Country, activeCity, age, in.IsCouple, ordinaryIncome, cpiIndex, medIndex, yearsElapsed)
				grossNeed := coreLivingOutNom + estHealth + milestoneOutNom - guaranteedInNom - milestoneInNom
				if grossNeed < 0 {
					surplus := -grossNeed
					taxableEq += surplus * 0.7
					taxableBasis += surplus * 0.7
					taxableDebt += surplus * 0.3
					grossNeed = 0
				}

				drawnLiq, ltcgRealized := drawFromLiquid(&taxableDebt, &taxableEq, &taxableBasis, grossNeed)
				realizedLTCG += ltcgRealized
				portfolioDrawNom += drawnLiq
				remNeed := grossNeed - drawnLiq

				if remNeed > 0 && rothSeasoned > 0 {
					pull := math.Min(remNeed, rothSeasoned)
					rothSeasoned -= pull
					portfolioDrawNom += pull
					remNeed -= pull
				}

				if remNeed > 0 && age >= 60 {
					if taxDeferred > 0 {
						pull := math.Min(remNeed*1.16, taxDeferred)
						taxDeferred -= pull
						ordinaryIncome += pull
						portfolioDrawNom += pull
						used := math.Min(remNeed, pull)
						taxableDebt += pull - used
						remNeed -= used
					}
					if remNeed > 0 {
						availRoth := rothGrowthLocked + sumRungs(rothRungs)
						if availRoth > 0 {
							pull := math.Min(remNeed, availRoth)
							drainRothPool(&rothGrowthLocked, &rothRungs, pull)
							portfolioDrawNom += pull
							remNeed -= pull
						}
					}
				}

				if remNeed > 0 && age < 60 && (taxDeferred+rothGrowthLocked+sumRungs(rothRungs) > 1.0) {
					crunchThisYear = true
					hadLiquidityCrunch = true
					if firstCrunchAge == 0 {
						firstCrunchAge = age
					}
					if taxDeferred > 0 {
						grossPull := math.Min(taxDeferred, remNeed*1.28)
						taxDeferred -= grossPull
						penalty := grossPull * 0.10
						taxesOutNom += penalty
						ordinaryIncome += grossPull
						portfolioDrawNom += grossPull
						netCovered := math.Min(remNeed, grossPull*0.90)
						remNeed -= netCovered
					}
					if remNeed > 0 {
						availRoth := rothGrowthLocked + sumRungs(rothRungs)
						if availRoth > 0 {
							pull := math.Min(availRoth, remNeed*1.12)
							drainRothPool(&rothGrowthLocked, &rothRungs, pull)
							taxesOutNom += pull * 0.10
							portfolioDrawNom += pull
							remNeed -= math.Min(remNeed, pull*0.90)
						}
					}
				}

				if in.EnableTaxOptimization && taxDeferred > 0 && age < 73 {
					convAge := age
					if activeCity.Country != data.CountryUSA && convAge < 65 {
						// Non-US residents do not buy US ACA Marketplace coverage; fill full 12% federal bracket
						convAge = 65
					}
					magiCeiling := tax.OptimalRothConversionCeiling(convAge, in.IsCouple, cpiIndex)
					currentMAGI := ordinaryIncome + realizedLTCG
					room := magiCeiling - currentMAGI
					if room > 1_000*cpiIndex {
						convertAmt := math.Min(taxDeferred, room)
						taxDeferred -= convertAmt
						rothRungs[4] += convertAmt
						ordinaryIncome += convertAmt
						rothOrHarvestNom = convertAmt
					}
				}

				finalMAGI := ordinaryIncome + realizedLTCG
				healthcareOutNom = computeResidencyHealthcareNom(in.Country, activeCity, age, in.IsCouple, finalMAGI, cpiIndex, medIndex, yearsElapsed)
				extraHealth := math.Max(0, healthcareOutNom-estHealth)
				assessableNW := taxableEq + taxableDebt + rothSeasoned
				if age >= 60 {
					assessableNW += taxDeferred + rothGrowthLocked
				}
				totalIncomeTax := computeResidencyTaxNom(
					in.Country, currCity, activeCity, in.IsCouple, in.EnableTaxOptimization,
					ordinaryIncome, realizedLTCG, assessableNW, taxableEq, yearsInRet, age, cpiIndex,
				)
				taxesOutNom += totalIncomeTax

				taxAndHealthBill := totalIncomeTax + extraHealth
				if taxAndHealthBill > 0 {
					paid, extraGain := drawFromLiquid(&taxableDebt, &taxableEq, &taxableBasis, taxAndHealthBill)
					portfolioDrawNom += paid
					if extraGain > 0 {
						taxesOutNom += extraGain * 0.15
					}
					remBill := taxAndHealthBill - paid
					if remBill > 0 && rothSeasoned > 0 {
						p := math.Min(remBill, rothSeasoned)
						rothSeasoned -= p
						portfolioDrawNom += p
						remBill -= p
					}
					if remBill > 0 && taxDeferred > 0 {
						p := math.Min(remBill*1.15, taxDeferred)
						taxDeferred -= p
						portfolioDrawNom += p
						remBill = 0
					}
					remNeed += remBill
				}

				if remNeed > 1.0 && totalRemaining(taxableEq, taxableDebt, taxDeferred, rothSeasoned, rothGrowthLocked, rothRungs, pensionLocked) <= 1.0 {
					survived = false
					if depletionAge == 0 {
						depletionAge = age
					}
				}

			case data.CountryAustralia, data.CountryEurope, data.CountryNewZealand:
				// ===================== AU / EU / NZ DECUMULATION =====================
				ordinaryIncome := milestoneTaxableInNom + guaranteedInNom
				realizedEqGain := 0.0

				estHealth := computeResidencyHealthcareNom(in.Country, activeCity, age, in.IsCouple, ordinaryIncome, cpiIndex, medIndex, yearsElapsed)
				grossNeed := coreLivingOutNom + estHealth + milestoneOutNom - guaranteedInNom - milestoneInNom
				if grossNeed < 0 {
					surplus := -grossNeed
					taxableEq += surplus * 0.70
					taxableBasis += surplus * 0.70
					taxableDebt += surplus * 0.30
					grossNeed = 0
				}

				remNeed := grossNeed

				// Australia Account-Based Super Pension mandatory minimum drawdown from Age 60+ (100% tax-free!)
				if in.Country == data.CountryAustralia && age >= 60 && taxDeferred > 0 {
					minSuper := taxDeferred * tax.AustraliaSuperMinDrawRate(age)
					if minSuper > 0 {
						taxDeferred -= minSuper
						used := math.Min(remNeed, minSuper)
						portfolioDrawNom += used
						remNeed -= used
						if minSuper > used {
							taxableDebt += minSuper - used
						}
					}
				}

				// Step A: Liquid Taxable Portfolio (Cash/Bonds + Brokerage/PIE)
				if remNeed > 0 {
					debtBefore := taxableDebt
					drawnLiq, eqGain := drawFromLiquid(&taxableDebt, &taxableEq, &taxableBasis, remNeed)
					debtDrawn := debtBefore - taxableDebt
					ordinaryIncome += debtDrawn * 0.22 // Interest/coupon portion of defensive fixed-income draw
					realizedEqGain += eqGain
					portfolioDrawNom += drawnLiq
					remNeed -= drawnLiq
				}

				// Step B: Seasoned Tax-Free Wrapper
				if remNeed > 0 && rothSeasoned > 0 {
					pull := math.Min(remNeed, rothSeasoned)
					rothSeasoned -= pull
					portfolioDrawNom += pull
					remNeed -= pull
				}

				// Step C: Statutory Retirement Fund Unlock (Age 60 for AU Super & EU Pillar 2/3a; Age 65 for NZ KiwiSaver)
				if remNeed > 0 && age >= unlockAge {
					if in.Country == data.CountryAustralia || in.Country == data.CountryNewZealand {
						// AU Superannuation (60+) and NZ KiwiSaver (65+) withdrawals are 100% TAX-FREE!
						if taxDeferred > 0 {
							pull := math.Min(remNeed, taxDeferred)
							taxDeferred -= pull
							portfolioDrawNom += pull
							remNeed -= pull
						}
					} else {
						// Europe: Pillar 3a / tax-advantaged wrapper first (low ~6% separate lump-sum tax or tax-free PEA), then Pillar 2 pension
						if rothGrowthLocked > 0 {
							pull := math.Min(remNeed*1.06, rothGrowthLocked)
							rothGrowthLocked -= pull
							taxesOutNom += pull * 0.055 // Modest Swiss Pillar 3a separate exit tax / wrapper exit
							used := math.Min(remNeed, pull*0.945)
							portfolioDrawNom += pull
							remNeed -= used
						}
						if remNeed > 0 && taxDeferred > 0 {
							pull := math.Min(remNeed*1.16, taxDeferred)
							taxDeferred -= pull
							ordinaryIncome += pull // Pillar 2 / occupational pension taxed as ordinary income
							portfolioDrawNom += pull
							used := math.Min(remNeed, pull)
							taxableDebt += pull - used
							remNeed -= used
						}
					}
				}

				// Step D: Pre-Unlock Liquidity Crunch Check (< Age 60 for AU/EU; < Age 65 for NZ KiwiSaver!)
				if remNeed > 0 && age < unlockAge && (taxDeferred+rothGrowthLocked > 1.0) {
					crunchThisYear = true
					hadLiquidityCrunch = true
					if firstCrunchAge == 0 {
						firstCrunchAge = age
					}
					// Strict preservation laws in AU/NZ/EU: hardship early release incurs ~22% penalty/tax friction
					if rothGrowthLocked > 0 {
						grossPull := math.Min(rothGrowthLocked, remNeed*1.22)
						rothGrowthLocked -= grossPull
						taxesOutNom += grossPull * 0.18
						portfolioDrawNom += grossPull
						remNeed -= math.Min(remNeed, grossPull*0.82)
					}
					if remNeed > 0 && taxDeferred > 0 {
						grossPull := math.Min(taxDeferred, remNeed*1.28)
						taxDeferred -= grossPull
						taxesOutNom += grossPull * 0.22
						portfolioDrawNom += grossPull
						remNeed -= math.Min(remNeed, grossPull*0.78)
					}
				}

				// Step E: Residency Tax & Healthcare Settlement
				// Note: Ununlocked Pillar 2/3a / Super / KiwiSaver are exempt from Swiss/EU net wealth taxes until unlocked
				assessableNW := taxableEq + taxableDebt + rothSeasoned
				if age >= unlockAge {
					assessableNW += taxDeferred + rothGrowthLocked
				}
				resTax := computeResidencyTaxNom(
					in.Country, currCity, activeCity, in.IsCouple, in.EnableTaxOptimization,
					ordinaryIncome, realizedEqGain, assessableNW, taxableEq, yearsInRet, age, cpiIndex,
				)
				taxesOutNom += resTax

				extraHealth := 0.0
				healthcareOutNom = estHealth
				if activeCity.Country == data.CountryUSA {
					finalMAGI := ordinaryIncome + realizedEqGain
					healthcareOutNom = computeResidencyHealthcareNom(in.Country, activeCity, age, in.IsCouple, finalMAGI, cpiIndex, medIndex, yearsElapsed)
					extraHealth = math.Max(0, healthcareOutNom-estHealth)
				}

				bill := resTax + extraHealth
				if bill > 0 {
					paid, _ := drawFromLiquid(&taxableDebt, &taxableEq, &taxableBasis, bill)
					portfolioDrawNom += paid
					remBill := bill - paid
					if remBill > 0 && age >= unlockAge && taxDeferred > 0 {
						p := math.Min(remBill, taxDeferred)
						taxDeferred -= p
						portfolioDrawNom += p
						remBill -= p
					}
					if remBill > 0 && rothGrowthLocked > 0 && age >= unlockAge {
						p := math.Min(remBill, rothGrowthLocked)
						rothGrowthLocked -= p
						portfolioDrawNom += p
						remBill -= p
					}
					remNeed += remBill
				}

				if remNeed > 1.0 && totalRemaining(taxableEq, taxableDebt, taxDeferred, rothSeasoned, rothGrowthLocked, rothRungs, pensionLocked) <= 1.0 {
					survived = false
					if depletionAge == 0 {
						depletionAge = age
					}
				}

			default:
				// ===================== INDIA ACCUMULATION ACCOUNTS DECUMULATION =====================
				estMAGI := milestoneTaxableInNom + guaranteedInNom
				healthcareOutNom = computeResidencyHealthcareNom(in.Country, activeCity, age, in.IsCouple, estMAGI, cpiIndex, medIndex, yearsElapsed)
				slabIncome := milestoneTaxableInNom + guaranteedInNom

				grossNeed := coreLivingOutNom + healthcareOutNom + milestoneOutNom - guaranteedInNom - milestoneInNom
				if grossNeed < 0 {
					surplus := -grossNeed
					taxableEq += surplus * 0.70
					taxableBasis += surplus * 0.70
					taxableDebt += surplus * 0.30
					grossNeed = 0
				}

				debtBefore := taxableDebt
				drawnLiq, realizedEqGain := drawFromLiquid(&taxableDebt, &taxableEq, &taxableBasis, grossNeed)
				debtDrawn := debtBefore - taxableDebt
				slabIncome += debtDrawn * 0.28
				portfolioDrawNom += drawnLiq
				remNeed := grossNeed - drawnLiq

				if remNeed > 0 && age >= 50 && rothSeasoned+rothGrowthLocked > 0 {
					availPPF := rothSeasoned + rothGrowthLocked
					pull := math.Min(remNeed, availPPF)
					scale := (availPPF - pull) / availPPF
					rothSeasoned *= scale
					rothGrowthLocked *= scale
					portfolioDrawNom += pull
					remNeed -= pull
				}
				if remNeed > 0 && age >= 55 && taxDeferred > 0 {
					pull := math.Min(remNeed, taxDeferred)
					taxDeferred -= pull
					portfolioDrawNom += pull
					remNeed -= pull
				}

				if remNeed > 0 && (taxDeferred+rothSeasoned+rothGrowthLocked+pensionLocked > 1.0) {
					crunchThisYear = true
					hadLiquidityCrunch = true
					if firstCrunchAge == 0 {
						firstCrunchAge = age
					}
					if taxDeferred > 0 {
						pull := math.Min(remNeed, taxDeferred)
						taxDeferred -= pull
						portfolioDrawNom += pull
						remNeed -= pull
					}
					if remNeed > 0 && rothSeasoned+rothGrowthLocked > 0 {
						availPPF := rothSeasoned + rothGrowthLocked
						pull := math.Min(remNeed, availPPF)
						scale := (availPPF - pull) / availPPF
						rothSeasoned *= scale
						rothGrowthLocked *= scale
						portfolioDrawNom += pull
						remNeed -= pull
					}
					if remNeed > 0 && age < 60 && pensionLocked > 0 {
						emergencyLump := pensionLocked * 0.20
						npsAnnuityAnnualNom += (pensionLocked * 0.80) * 0.06
						pensionLocked = 0
						used := math.Min(remNeed, emergencyLump)
						taxableDebt += emergencyLump - used
						portfolioDrawNom += used
						remNeed -= used
					}
				}

				exemption := tax.IndiaAnnualLTCGExemption(cpiIndex)
				if in.EnableTaxOptimization && realizedEqGain < exemption && taxableEq > taxableBasis {
					freeRoom := exemption - realizedEqGain
					harvest := math.Min(taxableEq-taxableBasis, freeRoom)
					taxableBasis += harvest
					rothOrHarvestNom = harvest
				}

				assessableNW := taxableEq + taxableDebt
				if age >= 60 {
					assessableNW += taxDeferred + rothSeasoned + rothGrowthLocked
				}
				taxesOutNom = computeResidencyTaxNom(
					in.Country, currCity, activeCity, in.IsCouple, in.EnableTaxOptimization,
					slabIncome, realizedEqGain, assessableNW, taxableEq, yearsInRet, age, cpiIndex,
				)
				extraHealth := 0.0
				if activeCity.Country == data.CountryUSA {
					finalMAGI := slabIncome + realizedEqGain
					finalHealth := computeResidencyHealthcareNom(in.Country, activeCity, age, in.IsCouple, finalMAGI, cpiIndex, medIndex, yearsElapsed)
					extraHealth = math.Max(0, finalHealth-healthcareOutNom)
					healthcareOutNom = finalHealth
				}

				bill := taxesOutNom + extraHealth
				if bill > 0 {
					paid, _ := drawFromLiquid(&taxableDebt, &taxableEq, &taxableBasis, bill)
					portfolioDrawNom += paid
					remBill := bill - paid
					if remBill > 0 && taxDeferred > 0 {
						p := math.Min(remBill, taxDeferred)
						taxDeferred -= p
						portfolioDrawNom += p
						remBill -= p
					}
					remNeed += remBill
				}

				if remNeed > 1.0 && totalRemaining(taxableEq, taxableDebt, taxDeferred, rothSeasoned, rothGrowthLocked, rothRungs, pensionLocked) <= 1.0 {
					survived = false
					if depletionAge == 0 {
						depletionAge = age
					}
				}
			}
		}

		// Compute end-of-year liquid vs locked balances across all 5 jurisdictions
		var liquidNom, lockedNom float64
		switch in.Country {
		case data.CountryUSA:
			if age >= 60 {
				liquidNom = taxableEq + taxableDebt + taxDeferred + rothSeasoned + rothGrowthLocked + sumRungs(rothRungs)
				lockedNom = 0
			} else {
				liquidNom = taxableEq + taxableDebt + rothSeasoned
				lockedNom = taxDeferred + rothGrowthLocked + sumRungs(rothRungs)
			}
		case data.CountryAustralia, data.CountryEurope, data.CountryNewZealand:
			if age >= unlockAge {
				liquidNom = taxableEq + taxableDebt + taxDeferred + rothSeasoned + rothGrowthLocked + sumRungs(rothRungs) + pensionLocked
				lockedNom = 0
			} else {
				liquidNom = taxableEq + taxableDebt + rothSeasoned
				lockedNom = taxDeferred + rothGrowthLocked + sumRungs(rothRungs) + pensionLocked
			}
		default:
			if age >= 60 {
				liquidNom = taxableEq + taxableDebt + taxDeferred + rothSeasoned + rothGrowthLocked + pensionLocked
				lockedNom = 0
			} else if age >= 55 {
				liquidNom = taxableEq + taxableDebt + taxDeferred + rothSeasoned + rothGrowthLocked
				lockedNom = pensionLocked
			} else {
				liquidNom = taxableEq + taxableDebt
				lockedNom = taxDeferred + rothSeasoned + rothGrowthLocked + pensionLocked
			}
		}

		if !survived {
			liquidNom = 0
			lockedNom = 0
		}
		totNom := math.Max(0, liquidNom+lockedNom)
		totReal := totNom / cpiIndex

		if recordYears {
			records[y] = pathYearRecord{
				realWealth:        totReal,
				nomWealth:         totNom,
				liquidReal:        math.Max(0, liquidNom) / cpiIndex,
				lockedReal:        math.Max(0, lockedNom) / cpiIndex,
				monthlySpendReal:  (coreLivingOutNom / cpiIndex) / 12.0,
				savingsInReal:     savingsInNom / cpiIndex,
				portfolioDrawReal: portfolioDrawNom / cpiIndex,
				guaranteedInReal:  guaranteedInNom / cpiIndex,
				milestoneInReal:   milestoneInNom / cpiIndex,
				rothOrHarvestReal: rothOrHarvestNom / cpiIndex,
				coreLivingOutReal: coreLivingOutNom / cpiIndex,
				healthcareOutReal: healthcareOutNom / cpiIndex,
				taxesOutReal:      taxesOutNom / cpiIndex,
				milestoneOutReal:  milestoneOutNom / cpiIndex,
				hadLiqCrunch:      crunchThisYear,
				refYear:           mkt.Year,
				eqRet:             eqRet,
				debtRet:           debtRet,
				cpi:               cpi,
			}
		} else if !survived {
			// Early exit once ruined in fast probe mode!
			break
		}
	}

	cagr5 := 0.0
	avgCPI5 := 0.0
	if ret5Count > 0 {
		cagr5 = math.Pow(math.Max(0.05, ret5Prod), 1.0/float64(ret5Count)) - 1.0
		avgCPI5 = cpi5Sum / float64(ret5Count)
	}

	termReal := 0.0
	if recordYears && numYears > 0 {
		termReal = records[numYears-1].realWealth
	}

	return singlePathOutcome{
		survived:           survived,
		depletionAge:       depletionAge,
		hadLiquidityCrunch: hadLiquidityCrunch,
		firstCrunchAge:     firstCrunchAge,
		terminalReal:       termReal,
		corpusAtFireReal:   corpusAtFireReal,
		corpusAtFireNom:    corpusAtFireNom,
		first5YrRealRet:    cagr5,
		first5YrAvgCPI:     avgCPI5,
		years:              records,
	}
}

// drawFromLiquid draws cash needed harmoniously across taxable debt/cash and taxable equity,
// tracking proportional cost-basis reduction and returning (totalDrawn, realizedEquityGain).
func drawFromLiquid(debt *float64, eq *float64, basis *float64, amount float64) (float64, float64) {
	if amount <= 0 {
		return 0, 0
	}
	tot := *debt + *eq
	if tot <= 0 {
		return 0, 0
	}
	if amount >= tot {
		gain := math.Max(0, *eq-*basis)
		*debt = 0
		*eq = 0
		*basis = 0
		return tot, gain
	}

	// Draw proportionally so asset mix doesn't drift wildly, slightly favoring debt buffer in drawdowns
	debtShare := *debt / tot
	fromDebt := math.Min(*debt, amount*math.Min(1.0, debtShare*1.25))
	fromEq := amount - fromDebt
	if fromEq > *eq {
		fromEq = *eq
		fromDebt = math.Min(*debt, amount-fromEq)
	}

	realizedGain := 0.0
	if fromEq > 0 && *eq > 0 {
		basisRatio := clamp(*basis / *eq, 0.0, 1.0)
		basisUsed := fromEq * basisRatio
		realizedGain = math.Max(0, fromEq-basisUsed)
		*eq -= fromEq
		*basis = math.Max(0, *basis-basisUsed)
	}
	*debt -= fromDebt
	return fromDebt + fromEq, realizedGain
}

func sumRungs(r [5]float64) float64 {
	return r[0] + r[1] + r[2] + r[3] + r[4]
}

func drainRothPool(growth *float64, rungs *[5]float64, amount float64) {
	rem := amount
	for i := 0; i < 5 && rem > 0; i++ {
		p := math.Min(rem, rungs[i])
		rungs[i] -= p
		rem -= p
	}
	if rem > 0 {
		*growth = math.Max(0, *growth-rem)
	}
}

func totalRemaining(eq, debt, def, rothS, rothG float64, rungs [5]float64, pen float64) float64 {
	return eq + debt + def + rothS + rothG + sumRungs(rungs) + pen
}

func quantileSorted(sorted []float64, q float64) float64 {
	n := len(sorted)
	if n == 0 {
		return 0
	}
	if q <= 0 {
		return sorted[0]
	}
	if q >= 1 {
		return sorted[n-1]
	}
	pos := q * float64(n-1)
	lo := int(math.Floor(pos))
	hi := int(math.Ceil(pos))
	if lo == hi {
		return sorted[lo]
	}
	frac := pos - float64(lo)
	return sorted[lo]*(1-frac) + sorted[hi]*frac
}

func sortAndQuantiles(vals []float64) (p5, p10, p25, p50, p75, p90 float64) {
	cp := make([]float64, len(vals))
	copy(cp, vals)
	slices.Sort(cp)
	return quantileSorted(cp, 0.05),
		quantileSorted(cp, 0.10),
		quantileSorted(cp, 0.25),
		quantileSorted(cp, 0.50),
		quantileSorted(cp, 0.75),
		quantileSorted(cp, 0.90)
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// EffectiveCOLScale computes the retirement spending multiplier when relocating from
// currCity to retCity, handling both intra-country weighted (65% non-housing COL + 35% housing)
// index ratios and cross-country purchasing-power ratios using calibrated Standard presets + FX parity.
func EffectiveCOLScale(currCity, retCity data.CityProfile) float64 {
	if retCity.ID == currCity.ID {
		return 1.0
	}
	if currCity.Country == retCity.Country {
		if currCity.CostOfLivingIndex > 0 && currCity.HousingCostIndex > 0 && retCity.HousingCostIndex > 0 {
			return 0.65*(retCity.CostOfLivingIndex/currCity.CostOfLivingIndex) +
				0.35*(retCity.HousingCostIndex/currCity.HousingCostIndex)
		}
		if currCity.CostOfLivingIndex > 0 {
			return retCity.CostOfLivingIndex / currCity.CostOfLivingIndex
		}
		return 1.0
	}
	if currCity.MonthlyPresets.Standard > 0 && retCity.MonthlyPresets.Standard > 0 {
		retStdInCurr := data.ConvertCurrency(retCity.MonthlyPresets.Standard, retCity.Country, currCity.Country)
		return retStdInCurr / currCity.MonthlyPresets.Standard
	}
	return 1.0
}

// realMedicalExcessRate returns the calibrated long-run annual excess medical cost inflation
// above headline CPI for the destination retirement jurisdiction.
func realMedicalExcessRate(country data.CountryCode) float64 {
	switch country {
	case data.CountryIndia:
		return 0.042 // ~10.4–11.5% medical inflation vs ~6.2% headline CPI
	case data.CountryUSA:
		return 0.016 // ~4.6% ACA/Medicare/OOP trend vs ~3.0% headline CPI
	case data.CountryAustralia:
		return 0.015 // ~4.3% PHI/specialist gap trend vs ~2.8% headline CPI
	case data.CountryEurope:
		return 0.012 // ~3.4% Swiss LAMal / EU mutuelle trend vs ~2.2% headline CPI
	case data.CountryNewZealand:
		return 0.018 // ~4.6% Southern Cross medical trend vs ~2.8% headline CPI
	default:
		return 0.018
	}
}

func computeResidencyHealthcareNom(
	acctCountry data.CountryCode,
	retCity data.CityProfile,
	age int,
	isCouple bool,
	magiAcctCurrency float64,
	cpiIndex, medIndex float64,
	yearsElapsed int,
) float64 {
	effMedIndex := medIndex
	if acctCountry != retCity.Country {
		// Prevent applying the origin country's nominal medical inflation excess (e.g., India ~11.5%/yr)
		// to a foreign retirement destination's healthcare system (e.g., Swiss LAMal or US Medicare).
		if yearsElapsed < 0 {
			yearsElapsed = 0
		}
		effMedIndex = cpiIndex * math.Pow(1.0+realMedicalExcessRate(retCity.Country), float64(yearsElapsed))
	}

	switch retCity.Country {
	case data.CountryUSA:
		magiUSD := data.ConvertCurrency(magiAcctCurrency, acctCountry, data.CountryUSA)
		hcUSD := tax.USHealthcareAnnual(age, isCouple, magiUSD, retCity, cpiIndex, effMedIndex)
		return data.ConvertCurrency(hcUSD, data.CountryUSA, acctCountry)
	case data.CountryAustralia:
		hcAUD := tax.AustraliaHealthcareAnnual(age, isCouple, retCity, cpiIndex, effMedIndex)
		return data.ConvertCurrency(hcAUD, data.CountryAustralia, acctCountry)
	case data.CountryEurope:
		hcEUR := tax.EuropeHealthcareAnnual(age, isCouple, retCity, cpiIndex, effMedIndex)
		return data.ConvertCurrency(hcEUR, data.CountryEurope, acctCountry)
	case data.CountryNewZealand:
		hcNZD := tax.NZHealthcareAnnual(age, isCouple, retCity, cpiIndex, effMedIndex)
		return data.ConvertCurrency(hcNZD, data.CountryNewZealand, acctCountry)
	default:
		hcINR := tax.IndiaHealthcareAnnual(age, isCouple, retCity, cpiIndex, effMedIndex)
		return data.ConvertCurrency(hcINR, data.CountryIndia, acctCountry)
	}
}

func computeResidencyTaxNom(
	acctCountry data.CountryCode,
	currCity, retCity data.CityProfile,
	isCouple, enableTaxOpt bool,
	ordinaryAcctCurrency, ltcgAcctCurrency, assessableNWAcctCurrency, taxableEqAcctCurrency float64,
	yearsInRetirement, age int,
	cpiIndex float64,
) float64 {
	ordConv := data.ConvertCurrency(ordinaryAcctCurrency, acctCountry, retCity.Country)
	ltcgConv := data.ConvertCurrency(ltcgAcctCurrency, acctCountry, retCity.Country)

	switch retCity.Country {
	case data.CountryUSA:
		fed := tax.USFederalTax(ordConv, ltcgConv, isCouple, cpiIndex)
		st := tax.USStateTax(retCity.StateCode, ordConv, ltcgConv, isCouple, cpiIndex)
		return data.ConvertCurrency(fed+st, data.CountryUSA, acctCountry)

	case data.CountryAustralia:
		// ASX franking credit offset applies when holding domestic Australian equity portfolio
		hasFranking := enableTaxOpt && acctCountry == data.CountryAustralia
		auTax := tax.AustraliaIncomeTax(ordConv, ltcgConv, isCouple, hasFranking, cpiIndex)
		return data.ConvertCurrency(auTax, data.CountryAustralia, acctCountry)

	case data.CountryEurope:
		nwEUR := data.ConvertCurrency(assessableNWAcctCurrency, acctCountry, data.CountryEurope)
		isIncomingForeign := acctCountry != data.CountryEurope || currCity.StateCode != retCity.StateCode
		euTax := tax.EuropeTaxAnnual(
			retCity, ordConv, ltcgConv, nwEUR,
			isCouple, enableTaxOpt, isIncomingForeign,
			yearsInRetirement, age, cpiIndex,
		)
		return data.ConvertCurrency(euTax, data.CountryEurope, acctCountry)

	case data.CountryNewZealand:
		eqNZD := data.ConvertCurrency(taxableEqAcctCurrency, acctCountry, data.CountryNewZealand)
		// NZ IRD Transitional Tax Resident 48-month (4-yr) FIF exemption for incoming cross-border retirees
		if acctCountry != data.CountryNewZealand && enableTaxOpt && yearsInRetirement >= 0 && yearsInRetirement < 4 {
			eqNZD = 0
		}
		nzTax := tax.NZTaxAnnual(ordConv, eqNZD, isCouple, enableTaxOpt, cpiIndex)
		return data.ConvertCurrency(nzTax, data.CountryNewZealand, acctCountry)

	default:
		ltcg := tax.IndiaEquityLTCGTax(ltcgConv, cpiIndex)
		slab := tax.IndiaNewRegimeSlabTax(ordConv, cpiIndex)
		return data.ConvertCurrency(ltcg+slab, data.CountryIndia, acctCountry)
	}
}
