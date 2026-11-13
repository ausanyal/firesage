package engine

import (
	"math"
	"testing"

	"github.com/ausanyal/firesage/internal/data"
	"github.com/ausanyal/firesage/internal/tax"
)

func TestAll34CitiesAnd5CountriesIntegrity(t *testing.T) {
	allCities := data.Cities()
	if len(allCities) != 34 {
		t.Fatalf("expected exactly 34 global cities, got %d", len(allCities))
	}

	expectedCounts := map[data.CountryCode]int{
		data.CountryIndia:      11,
		data.CountryUSA:        7,
		data.CountryAustralia:  4,
		data.CountryEurope:     8,
		data.CountryNewZealand: 4,
	}

	seenIDs := make(map[string]bool)
	counts := make(map[data.CountryCode]int)
	for _, c := range allCities {
		if seenIDs[c.ID] {
			t.Fatalf("duplicate city ID: %s", c.ID)
		}
		seenIDs[c.ID] = true
		counts[c.Country]++

		if c.CostOfLivingIndex <= 0 || c.HousingCostIndex <= 0 || c.HealthcareCostMult <= 0 || c.TypicalHomePrice <= 0 {
			t.Fatalf("city %s has non-positive index/price: %+v", c.ID, c)
		}
		p := c.MonthlyPresets
		if !(p.Lean > 0 && p.Lean < p.Standard && p.Standard < p.Chubby && p.Chubby < p.Fat) {
			t.Fatalf("city %s has invalid lifestyle preset ordering: %+v", c.ID, p)
		}
	}

	for country, want := range expectedCounts {
		if got := counts[country]; got != want {
			t.Fatalf("country %s: expected %d cities, got %d", country, want, got)
		}
	}

	// Verify required newly added cities are all registered
	mustExist := []string{
		"in-kolkata", "in-dehradun", "in-kodaikanal", "in-kochi",
		"au-sydney", "au-melbourne", "au-adelaide", "au-gold-coast",
		"eu-zurich-suburbs", "eu-lugano", "eu-south-italy", "eu-cascais-lisbon", "eu-algarve",
		"nz-auckland", "nz-wellington", "nz-tauranga-nelson",
	}
	for _, id := range mustExist {
		if !seenIDs[id] {
			t.Fatalf("missing required city ID %q", id)
		}
	}

	// Verify intra-country EffectiveCOLScale matches documented 0.65 * COL + 0.35 * Housing formula
	mumbai := data.GetCity("in-mumbai", data.CountryIndia)
	kolkata := data.GetCity("in-kolkata", data.CountryIndia)
	gotScale := EffectiveCOLScale(mumbai, kolkata)
	wantScale := 0.65*(kolkata.CostOfLivingIndex/mumbai.CostOfLivingIndex) +
		0.35*(kolkata.HousingCostIndex/mumbai.HousingCostIndex)
	if math.Abs(gotScale-wantScale) > 1e-9 {
		t.Fatalf("Mumbai->Kolkata EffectiveCOLScale = %.6f, want %.6f (0.65*COL + 0.35*Housing)", gotScale, wantScale)
	}
}

func TestMonotonicityInvariantsAcrossParameters(t *testing.T) {
	base := DefaultCountryTemplate(data.CountryIndia)
	base.NumPaths = 800
	base.Seed = 2026

	// 1. AnnualSavingsStepUp must be strictly monotonic from 0% -> 1% -> 5% -> 10%
	stepUps := []float64{0.00, 0.01, 0.05, 0.10}
	prevFireMed := -1.0
	prevSuccess := -1.0
	for _, su := range stepUps {
		p := base
		p.AnnualSavingsStepUp = su
		res := RunFullAnalysis(p)
		if res.CorpusAtFireRealP50 <= prevFireMed {
			t.Fatalf("non-monotonic CorpusAtFireRealP50 at step-up %.0f%%: got %.0f <= prev %.0f",
				su*100, res.CorpusAtFireRealP50, prevFireMed)
		}
		if res.SuccessRate < prevSuccess-0.01 {
			t.Fatalf("SuccessRate decreased when raising step-up to %.0f%%: %.2f%% < %.2f%%",
				su*100, res.SuccessRate, prevSuccess)
		}
		prevFireMed = res.CorpusAtFireRealP50
		prevSuccess = res.SuccessRate
	}

	// 2. Increasing DesiredMonthlySpend must strictly increase RequiredCorpus90 and never increase SuccessRate
	spends := []float64{90_000, 140_000, 220_000}
	prevReq := 0.0
	prevRate := 101.0
	for _, sp := range spends {
		p := base
		p.DesiredMonthlySpend = sp
		res := RunFullAnalysis(p)
		if res.RequiredCorpus90 <= prevReq {
			t.Fatalf("expected RequiredCorpus90 to increase with spend %.0f: got %.0f <= %.0f",
				sp, res.RequiredCorpus90, prevReq)
		}
		if res.SuccessRate > prevRate+0.01 {
			t.Fatalf("expected SuccessRate not to increase with higher spend %.0f: %.2f%% > %.2f%%",
				sp, res.SuccessRate, prevRate)
		}
		prevReq = res.RequiredCorpus90
		prevRate = res.SuccessRate
	}
}

func TestSolverSelfConsistencyAcrossAll5JurisdictionsAndCrossBorder(t *testing.T) {
	countries := []data.CountryCode{
		data.CountryIndia,
		data.CountryUSA,
		data.CountryAustralia,
		data.CountryEurope,
		data.CountryNewZealand,
	}

	for _, cc := range countries {
		plan := DefaultCountryTemplate(cc)
		plan.NumPaths = 600
		plan.Seed = 777
		res := RunFullAnalysis(plan)

		// A. SafeMonthlySpend90 self-consistency: setting DesiredMonthlySpend = SafeMonthlySpend90 should yield ~90% survival
		checkSafe := plan
		checkSafe.DesiredMonthlySpend = res.SafeMonthlySpend90
		resSafe := RunFullAnalysis(checkSafe)
		if resSafe.SuccessRate < 89.0 || resSafe.SuccessRate > 91.0 {
			t.Fatalf("[%s] SafeMonthlySpend90=%.0f yielded SuccessRate=%.2f%%, expected ~90%% (89–91%%)",
				cc, res.SafeMonthlySpend90, resSafe.SuccessRate)
		}

		// B. RequiredCorpus90 self-consistency when retiring immediately (CurrentAge == RetirementAge)
		nowPlan := plan
		nowPlan.CurrentAge = 50
		nowPlan.RetirementAge = 50
		nowPlan.MonthlySavings = 0
		nowPlan.MonthlyLockedSavings = 0
		resNow := RunFullAnalysis(nowPlan)

		verifyCorpus := nowPlan
		verifyCorpus.CurrentCorpus = resNow.RequiredCorpus90
		verifyCorpus.Buckets = scaleBucketsForCorpus(resNow.NormalizedInput, resNow.RequiredCorpus90)
		resVerify := RunFullAnalysis(verifyCorpus)
		if resVerify.SuccessRate < 89.0 || resVerify.SuccessRate > 91.0 {
			t.Fatalf("[%s] RequiredCorpus90=%.0f at immediate retirement yielded SuccessRate=%.2f%%, expected ~90%% (89–91%%)",
				cc, resNow.RequiredCorpus90, resVerify.SuccessRate)
		}

		// C. Milestone ordering: LeanFIREAge <= TargetFIREAge <= FortressFIREAge
		if res.LeanFIREAge > res.TargetFIREAge+0.2 || res.TargetFIREAge > res.FortressFIREAge+0.2 {
			t.Fatalf("[%s] invalid milestone ordering: Lean=%.1f, Target=%.1f, Fortress=%.1f",
				cc, res.LeanFIREAge, res.TargetFIREAge, res.FortressFIREAge)
		}
	}

	// D. Cross-Border LeanFIREAge currency conversion check: Working in India (INR) -> Retiring in Cascais/Lisbon (EUR)
	inToLisbon := DefaultCountryTemplate(data.CountryIndia)
	inToLisbon.RetirementCityID = "eu-cascais-lisbon"
	inToLisbon.AutoScaleLifestyleOnMove = true
	inToLisbon.CurrentCorpus = 25_000_000 // ₹2.5 Cr (~€263k) — NOT enough to Lean-FIRE in Lisbon tomorrow at Age 34!
	inToLisbon.NumPaths = 500
	inToLisbon.Seed = 888
	resCross := RunFullAnalysis(inToLisbon)
	if resCross.LeanFIREAge <= float64(inToLisbon.CurrentAge+2) {
		t.Fatalf("expected IN->Lisbon LeanFIREAge > %d (EUR->INR conversion check), got %.1f",
			inToLisbon.CurrentAge+2, resCross.LeanFIREAge)
	}
}

func TestCrossBorderHealthcareAndTaxAccountingAccuracy(t *testing.T) {
	// 1. Work in India (Bengaluru) -> Retire in Zurich Suburbs (Switzerland):
	// Verify Swiss LAMal healthcare at Age 75 is NOT distorted by Indian 11.5% medical CPI!
	inToZurich := PlanInput{
		Country:                  data.CountryIndia,
		CurrentAge:               35,
		RetirementAge:            50,
		HorizonAge:               85,
		IsCouple:                 true,
		CurrentCityID:            "in-bengaluru",
		RetirementCityID:         "eu-zurich-suburbs",
		AutoScaleLifestyleOnMove: true,
		CurrentCorpus:            250_000_000, // ₹25 Cr (~€2.63M)
		MonthlySavings:           500_000,
		DesiredMonthlySpend:      150_000,
		Mode:                     ModeBootstrap,
		NumPaths:                 400,
		Seed:                     123,
	}
	resZH := RunFullAnalysis(inToZurich)
	idx75 := 75 - inToZurich.CurrentAge
	hcRealINR := resZH.YearlyBands[idx75].FlowHealthcareOutReal
	hcRealEUR := data.ConvertCurrency(hcRealINR, data.CountryIndia, data.CountryEurope)
	// Today's Swiss couple age 75 healthcare in Zurich Suburbs: 7550 * 2.0 * 1.15 = €17,365/yr;
	// after 41 years of +1.2%/yr real medical excess (~1.63x), expected ~€28,300/yr in real EUR (definitely between €18k and €40k, NOT €120k+!).
	if hcRealEUR < 18_000 || hcRealEUR > 40_000 {
		t.Fatalf("IN->Zurich real healthcare at Age 75 = €%.0f/yr (₹%.0f), expected realistic €18k–€40k (check cross-border medCPI)",
			hcRealEUR, hcRealINR)
	}

	// 2. Work in India -> Retire in Raleigh, USA (pre-65 ACA):
	// Verify extraHealth is NOT double-counted inside FlowTaxesOutReal!
	inToUS := PlanInput{
		Country:                  data.CountryIndia,
		CurrentAge:               45,
		RetirementAge:            50,
		HorizonAge:               85,
		IsCouple:                 true,
		CurrentCityID:            "in-bengaluru",
		RetirementCityID:         "us-raleigh",
		AutoScaleLifestyleOnMove: true,
		CurrentCorpus:            300_000_000, // ₹30 Cr (~$3.47M)
		Buckets: AccountBuckets{
			UseGranular:      true,
			TaxableEquity:    200_000_000,
			TaxableCostBasis: 0.95, // Almost all principal -> near-zero realized LTCG!
			TaxableDebtCash:  100_000_000,
		},
		DesiredMonthlySpend: 150_000,
		Mode:                ModeBootstrap,
		NumPaths:            300,
		Seed:                321,
	}
	resUS := RunFullAnalysis(inToUS)
	idx55 := 55 - inToUS.CurrentAge
	taxRealUSD := data.ConvertCurrency(resUS.YearlyBands[idx55].FlowTaxesOutReal, data.CountryIndia, data.CountryUSA)
	hcRealUSD := data.ConvertCurrency(resUS.YearlyBands[idx55].FlowHealthcareOutReal, data.CountryIndia, data.CountryUSA)
	if hcRealUSD < 12_000 {
		t.Fatalf("expected meaningful pre-65 US ACA healthcare at Age 55, got $%.0f", hcRealUSD)
	}
	if taxRealUSD > 8_000 {
		t.Fatalf("expected low US tax on high-basis portfolio at Age 55 (no ACA healthcare double-count), got $%.0f", taxRealUSD)
	}
}

func TestAustraliaCentrelinkAdditiveIncomeTestAndStatutoryUnlockAges(t *testing.T) {
	// Adding A$12,000/yr private pension at A$250k assets must strictly increase total household guaranteed income
	basePension := tax.AustraliaAgePensionWithIncomeTest(68, false, 250_000, 0, 1.0)
	withPrivate := 12_000.0 + tax.AustraliaAgePensionWithIncomeTest(68, false, 250_000, 12_000, 1.0)
	if withPrivate <= basePension {
		t.Fatalf("expected private income + tapered Age Pension (A$%.0f) > base Age Pension alone (A$%.0f)",
			withPrivate, basePension)
	}

	// Verify StatutoryUnlockAge across all 5 jurisdictions
	if StatutoryUnlockAge(data.CountryNewZealand) != 65 {
		t.Fatalf("expected NZ KiwiSaver unlock age 65, got %d", StatutoryUnlockAge(data.CountryNewZealand))
	}
	for _, cc := range []data.CountryCode{data.CountryIndia, data.CountryUSA, data.CountryAustralia, data.CountryEurope} {
		if StatutoryUnlockAge(cc) != 60 {
			t.Fatalf("expected %s unlock age 60, got %d", cc, StatutoryUnlockAge(cc))
		}
	}
}
