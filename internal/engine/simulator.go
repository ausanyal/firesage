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

// NormalizePlanInput fills defaults and clamps parameters into safe, consistent bounds.
func NormalizePlanInput(in PlanInput) PlanInput {
	if in.Country != data.CountryUSA && in.Country != data.CountryIndia {
		in.Country = data.CountryIndia
	}
	if in.CurrentAge < 18 {
		in.CurrentAge = 32
	}
	if in.CurrentAge > 85 {
		in.CurrentAge = 85
	}
	if in.RetirementAge <= in.CurrentAge {
		in.RetirementAge = minInt(in.CurrentAge+12, 60)
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
		retCity := data.GetCity(in.RetirementCityID, in.Country)
		if retCity.Country != in.Country {
			in.RetirementCityID = currCity.ID
		} else {
			in.RetirementCityID = retCity.ID
		}
	}

	if in.CurrentCorpus < 0 {
		in.CurrentCorpus = 0
	}
	if in.MonthlySavings < 0 {
		in.MonthlySavings = 0
	}
	if in.DesiredMonthlySpend <= 0 {
		in.DesiredMonthlySpend = currCity.MonthlyPresets.Standard
	}
	in.AnnualSavingsStepUp = clamp(in.AnnualSavingsStepUp, 0.0, 0.25)

	// Normalize asset allocation weights
	sumW := in.Allocation.Equity + in.Allocation.Debt + in.Allocation.Gold + in.Allocation.Cash
	if sumW <= 0.001 {
		if in.Country == data.CountryIndia {
			in.Allocation = AssetAllocation{Equity: 0.65, Debt: 0.22, Gold: 0.08, Cash: 0.05, UseBondTent: true}
		} else {
			in.Allocation = AssetAllocation{Equity: 0.70, Debt: 0.22, Gold: 0.03, Cash: 0.05, UseBondTent: true}
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
		if in.Country == data.CountryUSA {
			in.SocialSecurityStartAge = 67
		} else {
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
	if country == data.CountryUSA {
		return AccountBuckets{
			UseGranular:       false,
			TaxableEquity:     corpus * 0.40,
			TaxableCostBasis:  0.65,
			TaxableDebtCash:   corpus * 0.08,
			TaxDeferredLocked: corpus * 0.38, // 401(k) / Traditional IRA
			TaxFreeAccount:    corpus * 0.14, // Roth IRA / HSA
			PensionLocked:     0,
		}
	}
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

func pregenerateProbeScenarios(in PlanInput, nProbe int) [][]data.MarketYear {
	history := data.HistoryByCountry(in.Country)
	mode := in.Mode
	if mode == ModeHistorical {
		mode = ModeBootstrap
		nProbe = minInt(nProbe, len(history))
	}
	numYears := in.HorizonAge - in.CurrentAge + 1
	scenarios := make([][]data.MarketYear, nProbe)
	for p := 0; p < nProbe; p++ {
		rng := rand.New(rand.NewPCG(in.Seed+uint64(p)*7919+31, uint64(p)*104729+17))
		scenarios[p] = generatePathScenario(mode, in.Country, history, numYears, p, rng)
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
	n := len(scenarios)
	if n == 0 {
		return 0
	}
	currCity := data.GetCity(in.CurrentCityID, in.Country)
	retCity := data.GetCity(in.RetirementCityID, in.Country)
	surv := 0
	for i := 0; i < n; i++ {
		ok, _ := simulateFastProbe(in, currCity, retCity, scenarios[i])
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
	feeDrag := in.ExpenseRatioBps / 10_000.0
	birthYear := baseSimulationYear - in.CurrentAge
	_, medianPE := data.CurrentValuation(in.Country)

	// Compute target baseline annual real spend in retirement
	colScale := 1.0
	if in.AutoScaleLifestyleOnMove && retCity.ID != currCity.ID && currCity.CostOfLivingIndex > 0 {
		colScale = retCity.CostOfLivingIndex / currCity.CostOfLivingIndex
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

		// Apply investment returns for year y (skipping t=0 pre-existing snapshot? No, year y represents age -> age+1)
		if y > 0 {
			cpiIndex *= (1.0 + cpi)
			medIndex *= (1.0 + medCPI)
		}

		// Track first 5 years of retirement returns for SRR Heatmap
		if isRetired && ret5Count < 5 {
			portReal := (1.0 + growthBlendRet) / math.Max(0.5, 1.0+cpi)
			ret5Prod *= portReal
			cpi5Sum += cpi
			ret5Count++
		}

		// Grow buckets for the year
		if taxableEq > 0 {
			taxableEq *= (1.0 + eqGoldBlendRet)
		}
		if taxableDebt > 0 {
			taxableDebt *= (1.0 + defensiveBlendRet)
		}
		if taxDeferred > 0 {
			if in.Country == data.CountryIndia {
				// EPF sovereign-backed debt-like steady return (~7.8%–8.4% linked to debt/cash regime)
				epfRate := clamp(0.65*debtRet+0.35*cashRet+0.008, 0.055, 0.095)
				taxDeferred *= (1.0 + epfRate)
			} else {
				taxDeferred *= (1.0 + growthBlendRet)
			}
		}
		if rothSeasoned+rothGrowthLocked+sumRungs(rothRungs) > 0 {
			if in.Country == data.CountryIndia {
				// PPF EEE sovereign rate (~7.1%–8.0%)
				ppfRate := clamp(0.50*debtRet+0.50*cashRet+0.005, 0.055, 0.088)
				rothGrowthLocked += (rothSeasoned + rothGrowthLocked) * ppfRate
			} else {
				// US Roth grows tax-free; growth accrues to rothGrowthLocked until 59.5
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
			// (Avoid forced taxable equity sales just to rebalance if unnecessary tax drag; drawdown handles trimming equity first when overweight!)
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

		if !isRetired {
			// ===================== ACCUMULATION PHASE =====================
			yearsWorked := age - in.CurrentAge
			stepUpMult := math.Pow(1.0+in.AnnualSavingsStepUp, float64(yearsWorked))
			// Note: AnnualSavingsStepUp is nominal career growth; ensure at least CPI pace if stepUp==0?
			// If user specifies AnnualSavingsStepUp (default 0.06), compound nominally, or if 0, inflate with CPI so real savings stay constant.
			if in.AnnualSavingsStepUp > 0 {
				savingsInNom = in.MonthlySavings * 12.0 * stepUpMult
			} else {
				savingsInNom = in.MonthlySavings * 12.0 * cpiIndex
			}

			netAddition := savingsInNom + milestoneInNom - milestoneOutNom
			if netAddition >= 0 {
				if in.Country == data.CountryUSA {
					toTaxableEq := netAddition * 0.42
					toTaxableDebt := netAddition * 0.08
					to401k := netAddition * 0.34
					toRoth := netAddition * 0.16
					taxableEq += toTaxableEq
					taxableBasis += toTaxableEq
					taxableDebt += toTaxableDebt
					taxDeferred += to401k
					rothSeasoned += toRoth
				} else {
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
				// Large pre-retirement milestone outflow exceeding annual savings (e.g. home down payment / sabbatical)
				deficit := -netAddition
				drawn, _ := drawFromLiquid(&taxableDebt, &taxableEq, &taxableBasis, deficit)
				rem := deficit - drawn
				if rem > 0 {
					pull := math.Min(rem, rothSeasoned)
					rothSeasoned -= pull
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
			totalStartNom := taxableEq + taxableDebt + taxDeferred + rothSeasoned + rothGrowthLocked + sumRungs(rothRungs) + pensionLocked
			totalStartReal := totalStartNom / cpiIndex

			// 1. Compute Core Living Spend in today's purchasing power per active WithdrawalStrategy
			targetSpendReal := baseRetireAnnualReal
			if in.UseSpendingSmile {
				// David Blanchett empirical "Retirement Spending Smile": active Go-Go early years,
				// gradual real decline across 60-78 Slow-Go years (while Healthcare separately models late-life spike)
				yearsInRet := age - in.RetirementAge
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
						// Capital preservation guardrail: trim discretionary spending smoothly down to floor
						ratio := (initWR * 1.20) / currWR
						targetSpendReal *= clamp(ratio, in.GuardrailFloorPct, 1.0)
					} else if currWR < initWR*0.78 {
						// Prosperity guardrail: raise lifestyle up to ceiling
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

			// 2. Guaranteed Income Floors (Social Security / Pension / NPS Annuity)
			if age >= in.SocialSecurityStartAge && in.SocialSecurityMonthlyReal > 0 {
				guaranteedInNom += in.SocialSecurityMonthlyReal * 12.0 * cpiIndex
			}
			guaranteedInNom += npsAnnuityAnnualNom

			// 3. Country-specific Healthcare, RMD, Tax-Smart Drawdown & Roth Ladder Sequencing
			if in.Country == data.CountryUSA {
				ordinaryIncome := milestoneTaxableInNom + guaranteedInNom*0.85 // Up to 85% of SS taxable
				realizedLTCG := 0.0

				// Enforce SECURE 2.0 Required Minimum Distributions (Age 73 or 75)
				rmdDiv := tax.USRMDDivisor(age, birthYear)
				mandatoryRMD := 0.0
				if rmdDiv > 0 && taxDeferred > 0 {
					mandatoryRMD = taxDeferred / rmdDiv
					taxDeferred -= mandatoryRMD
					taxableDebt += mandatoryRMD
					ordinaryIncome += mandatoryRMD
				}

				// Estimate baseline healthcare before discretionary conversions
				estHealth := tax.USHealthcareAnnual(age, in.IsCouple, ordinaryIncome, activeCity, cpiIndex, medIndex)
				grossNeed := coreLivingOutNom + estHealth + milestoneOutNom - guaranteedInNom - milestoneInNom
				if grossNeed < 0 {
					// Surplus guaranteed/milestone income gets reinvested into taxable
					surplus := -grossNeed
					taxableEq += surplus * 0.7
					taxableBasis += surplus * 0.7
					taxableDebt += surplus * 0.3
					grossNeed = 0
				}

				// Draw sequence:
				// Step A: Cash/Debt + Taxable Brokerage (basis-aware LTCG)
				drawnLiq, ltcgRealized := drawFromLiquid(&taxableDebt, &taxableEq, &taxableBasis, grossNeed)
				realizedLTCG += ltcgRealized
				portfolioDrawNom += drawnLiq
				remNeed := grossNeed - drawnLiq

				// Step B: Seasoned Roth principal & matured 5-year conversion ladder rungs (100% tax & penalty free!)
				if remNeed > 0 && rothSeasoned > 0 {
					pull := math.Min(remNeed, rothSeasoned)
					rothSeasoned -= pull
					portfolioDrawNom += pull
					remNeed -= pull
				}

				// Step C: Post-59.5 (age >= 60) Penalty-Free Traditional 401(k)/IRA + Roth Growth
				if remNeed > 0 && age >= 60 {
					if taxDeferred > 0 {
						// Gross up slightly for marginal tax
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

				// Step D: Pre-60 Emergency Liquidity Fallback!
				// If remNeed > 0 at age < 60 while locked 401(k)/Roth growth still exists, record Liquidity Crunch!
				if remNeed > 0 && age < 60 && (taxDeferred+rothGrowthLocked+sumRungs(rothRungs) > 1.0) {
					crunchThisYear = true
					hadLiquidityCrunch = true
					if firstCrunchAge == 0 {
						firstCrunchAge = age
					}
					// Forced early 401(k) tap incurs ordinary tax + 10% IRS early withdrawal penalty
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

				// Step E: Automated ACA-Cliff-Aware 5-Year Roth Conversion Ladder!
				if in.EnableTaxOptimization && taxDeferred > 0 && age < 73 {
					magiCeiling := tax.OptimalRothConversionCeiling(age, in.IsCouple, cpiIndex)
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

				// Final exact healthcare & Fed + State tax calculation based on realized MAGI
				finalMAGI := ordinaryIncome + realizedLTCG
				healthcareOutNom = tax.USHealthcareAnnual(age, in.IsCouple, finalMAGI, activeCity, cpiIndex, medIndex)
				extraHealth := math.Max(0, healthcareOutNom-estHealth)
				fedTax := tax.USFederalTax(ordinaryIncome, realizedLTCG, in.IsCouple, cpiIndex)
				stateTax := tax.USStateTax(activeCity.StateCode, ordinaryIncome, realizedLTCG, in.IsCouple, cpiIndex)
				totalIncomeTax := fedTax + stateTax
				taxesOutNom += totalIncomeTax

				// Pay tax + any healthcare delta from remaining accounts
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
			} else {
				// ===================== INDIA DECUMULATION & TAX SEQUENCING =====================
				healthcareOutNom = tax.IndiaHealthcareAnnual(age, in.IsCouple, activeCity, cpiIndex, medIndex)
				slabIncome := milestoneTaxableInNom + guaranteedInNom // NPS annuity + pension + consulting

				grossNeed := coreLivingOutNom + healthcareOutNom + milestoneOutNom - guaranteedInNom - milestoneInNom
				if grossNeed < 0 {
					surplus := -grossNeed
					taxableEq += surplus * 0.70
					taxableBasis += surplus * 0.70
					taxableDebt += surplus * 0.30
					grossNeed = 0
				}

				// Step A: Liquid Debt/Arbitrage/Equity MF Drawdown
				// Track implicit debt interest portion (~30% of debt draw) + equity LTCG
				debtBefore := taxableDebt
				drawnLiq, realizedEqGain := drawFromLiquid(&taxableDebt, &taxableEq, &taxableBasis, grossNeed)
				debtDrawn := debtBefore - taxableDebt
				slabIncome += debtDrawn * 0.28 // Accrued gain portion of Debt MF taxed at slab
				portfolioDrawNom += drawnLiq
				remNeed := grossNeed - drawnLiq

				// Step B: PPF (EEE Tax-Free) & Matured EPF post-55 (or bridge if retired, though NPS stays locked till 60!)
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

				// Pre-unlock Liquidity Check in India:
				// If liquid MFs run dry before Age 55/60 while EPF/PPF/NPS still hold locked funds, flag Liquidity Crunch!
				if remNeed > 0 && (taxDeferred+rothSeasoned+rothGrowthLocked+pensionLocked > 1.0) {
					crunchThisYear = true
					hadLiquidityCrunch = true
					if firstCrunchAge == 0 {
						firstCrunchAge = age
					}
					// EPF/PPF can be tapped in emergency post-employment, whereas NPS Tier-I (<60) forces 80% annuity lock!
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
					// Note: pensionLocked (NPS < 60) remains locked (only 20% lump sum allowed on premature exit)
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

				// Annual ₹1.25L Section 112A Tax-Gain Harvesting on unused exemption room
				exemption := tax.IndiaAnnualLTCGExemption(cpiIndex)
				if in.EnableTaxOptimization && realizedEqGain < exemption && taxableEq > taxableBasis {
					freeRoom := exemption - realizedEqGain
					harvest := math.Min(taxableEq-taxableBasis, freeRoom)
					taxableBasis += harvest
					rothOrHarvestNom = harvest
				}

				ltcgTax := tax.IndiaEquityLTCGTax(realizedEqGain, cpiIndex)
				slabTax := tax.IndiaNewRegimeSlabTax(slabIncome, cpiIndex)
				taxesOutNom = ltcgTax + slabTax

				if taxesOutNom > 0 {
					paid, _ := drawFromLiquid(&taxableDebt, &taxableEq, &taxableBasis, taxesOutNom)
					portfolioDrawNom += paid
					remTax := taxesOutNom - paid
					if remTax > 0 && taxDeferred > 0 {
						p := math.Min(remTax, taxDeferred)
						taxDeferred -= p
						portfolioDrawNom += p
						remTax -= p
					}
					remNeed += remTax
				}

				if remNeed > 1.0 && totalRemaining(taxableEq, taxableDebt, taxDeferred, rothSeasoned, rothGrowthLocked, rothRungs, pensionLocked) <= 1.0 {
					survived = false
					if depletionAge == 0 {
						depletionAge = age
					}
				}
			}
		}

		// Compute end-of-year liquid vs locked balances
		var liquidNom, lockedNom float64
		if in.Country == data.CountryUSA {
			if age >= 60 {
				liquidNom = taxableEq + taxableDebt + taxDeferred + rothSeasoned + rothGrowthLocked + sumRungs(rothRungs)
				lockedNom = 0
			} else {
				liquidNom = taxableEq + taxableDebt + rothSeasoned
				lockedNom = taxDeferred + rothGrowthLocked + sumRungs(rothRungs)
			}
		} else {
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
