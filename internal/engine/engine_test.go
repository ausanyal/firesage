package engine

import (
	"testing"

	"github.com/ausanyal/firesage/internal/data"
)

func TestIndiaBengaluruToGoaFullAnalysis(t *testing.T) {
	in := PlanInput{
		Country:                  data.CountryIndia,
		CurrentAge:               33,
		RetirementAge:            45,
		HorizonAge:               90,
		IsCouple:                 true,
		CurrentCityID:            "in-bengaluru",
		RetirementCityID:         "in-goa",
		AutoScaleLifestyleOnMove: true,
		CurrentCorpus:            18_000_000, // ₹1.80 Cr
		MonthlySavings:           150_000,    // ₹1.50 L/mo
		AnnualSavingsStepUp:      0.07,
		DesiredMonthlySpend:      140_000,
		Mode:                     ModeBootstrap,
		NumPaths:                 1500,
		Seed:                     42,
	}

	res1 := RunFullAnalysis(in)
	res2 := RunFullAnalysis(in)

	// Verify determinism under identical seed
	if res1.SuccessRate != res2.SuccessRate || res1.CorpusAtFireRealP50 != res2.CorpusAtFireRealP50 {
		t.Fatalf("expected deterministic output for fixed seed: %.2f vs %.2f", res1.SuccessRate, res2.SuccessRate)
	}

	// Verify Geo-Arbitrage advantage: Retiring in Goa requires meaningfully less corpus & higher survival than staying in Bengaluru
	if res1.GeoArbitrage.MoveSuccessRate <= res1.GeoArbitrage.StaySuccessRate {
		t.Fatalf("expected Goa relocation survival (%.2f%%) > Bengaluru stay survival (%.2f%%)",
			res1.GeoArbitrage.MoveSuccessRate, res1.GeoArbitrage.StaySuccessRate)
	}
	if res1.GeoArbitrage.CorpusSavedReal <= 0 {
		t.Fatalf("expected positive real corpus savings moving Bengaluru -> Goa, got %.0f", res1.GeoArbitrage.CorpusSavedReal)
	}
	if len(res1.YearlyBands) != (90 - 33 + 1) {
		t.Fatalf("expected 58 yearly bands, got %d", len(res1.YearlyBands))
	}
	if len(res1.CrisisReplays) == 0 || len(res1.Frontier) == 0 || len(res1.ActionLevers) == 0 {
		t.Fatalf("expected populated crisis replays, frontier, and action levers")
	}
	t.Logf("🇮🇳 Bengaluru->Goa: Success=%.1f%%, MedianFIRE=₹%.2fCr, TargetFIREAge=%.1f, Saved=₹%.2fCr, Compute=%.1fms",
		res1.SuccessRate, res1.CorpusAtFireRealP50/1e7, res1.TargetFIREAge, res1.GeoArbitrage.CorpusSavedReal/1e7, res1.ComputeDurationMs)
}

func TestUSSanJoseToRaleighFullAnalysis(t *testing.T) {
	in := PlanInput{
		Country:                   data.CountryUSA,
		CurrentAge:                34,
		RetirementAge:             46,
		HorizonAge:                92,
		IsCouple:                  true,
		CurrentCityID:             "us-san-jose",
		RetirementCityID:          "us-raleigh",
		AutoScaleLifestyleOnMove:  true,
		CurrentCorpus:             650_000,
		MonthlySavings:            6_500,
		AnnualSavingsStepUp:       0.04,
		DesiredMonthlySpend:       10_500,
		SocialSecurityMonthlyReal: 3_400,
		SocialSecurityStartAge:    67,
		EnableTaxOptimization:     true,
		Mode:                      ModeBootstrap,
		NumPaths:                  1500,
		Seed:                      99,
	}

	res := RunFullAnalysis(in)
	if res.GeoArbitrage.MoveSuccessRate <= res.GeoArbitrage.StaySuccessRate {
		t.Fatalf("expected Raleigh NC relocation survival (%.2f%%) > San Jose CA stay survival (%.2f%%)",
			res.GeoArbitrage.MoveSuccessRate, res.GeoArbitrage.StaySuccessRate)
	}
	if res.GeoArbitrage.CorpusSavedReal <= 100_000 {
		t.Fatalf("expected substantial corpus savings relocating San Jose -> Raleigh, got $%.0f", res.GeoArbitrage.CorpusSavedReal)
	}
	t.Logf("🇺🇸 SanJose->Raleigh: Success=%.1f%%, MedianFIRE=$%.2fM, TargetFIREAge=%.1f, Saved=$%.2fM, Compute=%.1fms",
		res.SuccessRate, res.CorpusAtFireRealP50/1e6, res.TargetFIREAge, res.GeoArbitrage.CorpusSavedReal/1e6, res.ComputeDurationMs)
}

func TestCrossBorderUSToIndiaAndIndiaToUS(t *testing.T) {
	// 1. Work in USA -> Retire in India (Goa): cross-border COL scaling & India residency tax/healthcare
	usToIn := RunFullAnalysis(PlanInput{
		Country:                  data.CountryUSA,
		CurrentAge:               34,
		RetirementAge:            46,
		HorizonAge:               90,
		IsCouple:                 true,
		CurrentCityID:            "us-san-jose",
		RetirementCityID:         "in-goa",
		AutoScaleLifestyleOnMove: true,
		CurrentCorpus:            650_000,
		MonthlySavings:           3_250,
		MonthlyLockedSavings:     3_250,
		AnnualSavingsStepUp:      0.04,
		DesiredMonthlySpend:      10_500,
		EnableTaxOptimization:    true,
		Mode:                     ModeBootstrap,
		NumPaths:                 1000,
		Seed:                     42,
	})
	if usToIn.GeoArbitrage.ComparisonCity.ID != "in-goa" || usToIn.GeoArbitrage.ComparisonCity.Country != data.CountryIndia {
		t.Fatalf("expected comparison city in-goa (IN), got %+v", usToIn.GeoArbitrage.ComparisonCity)
	}
	if usToIn.GeoArbitrage.MoveSuccessRate <= usToIn.GeoArbitrage.StaySuccessRate {
		t.Fatalf("expected San Jose -> Goa cross-border survival (%.1f%%) > staying in San Jose (%.1f%%)",
			usToIn.GeoArbitrage.MoveSuccessRate, usToIn.GeoArbitrage.StaySuccessRate)
	}
	if usToIn.GeoArbitrage.EffectiveRetireMonthly >= usToIn.GeoArbitrage.StayRetireMonthly*0.5 {
		t.Fatalf("expected lower USD monthly spend in Goa vs San Jose, got $%.0f vs $%.0f",
			usToIn.GeoArbitrage.EffectiveRetireMonthly, usToIn.GeoArbitrage.StayRetireMonthly)
	}

	// 2. Work in India -> Retire in USA (Raleigh): preserves RetirementCityID across countries & scales spend up in INR
	inToUs := NormalizePlanInput(PlanInput{
		Country:                  data.CountryIndia,
		CurrentAge:               33,
		RetirementAge:            48,
		HorizonAge:               90,
		IsCouple:                 true,
		CurrentCityID:            "in-bengaluru",
		RetirementCityID:         "us-raleigh",
		AutoScaleLifestyleOnMove: true,
		CurrentCorpus:            40_000_000,
		MonthlySavings:           150_000,
		MonthlyLockedSavings:     50_000,
		DesiredMonthlySpend:      140_000,
		NumPaths:                 600,
		Seed:                     42,
	})
	if inToUs.RetirementCityID != "us-raleigh" {
		t.Fatalf("expected NormalizePlanInput to preserve cross-border RetirementCityID=us-raleigh, got %q", inToUs.RetirementCityID)
	}
	resInToUs := RunFullAnalysis(inToUs)
	if resInToUs.GeoArbitrage.EffectiveRetireMonthly <= resInToUs.GeoArbitrage.StayRetireMonthly*2.0 {
		t.Fatalf("expected scaled INR monthly spend in Raleigh > 2x Bengaluru, got ₹%.0f vs ₹%.0f",
			resInToUs.GeoArbitrage.EffectiveRetireMonthly, resInToUs.GeoArbitrage.StayRetireMonthly)
	}
}

func TestMonthlyLockedRetirementFundContributions(t *testing.T) {
	// Compare zero locked contributions vs active Age-60 401(k)/Roth contributions
	base := PlanInput{
		Country:             data.CountryUSA,
		CurrentAge:          32,
		RetirementAge:       50,
		HorizonAge:          88,
		CurrentCityID:       "us-seattle",
		RetirementCityID:    "us-seattle",
		CurrentCorpus:       300_000,
		Buckets: AccountBuckets{
			UseGranular:       true,
			TaxableEquity:     120_000,
			TaxableCostBasis:  0.70,
			TaxableDebtCash:   30_000,
			TaxDeferredLocked: 110_000,
			TaxFreeAccount:    40_000,
		},
		MonthlySavings:       2_500,
		MonthlyLockedSavings: 0,
		DesiredMonthlySpend:  7_000,
		Mode:                 ModeBootstrap,
		NumPaths:             600,
		Seed:                 77,
	}
	withoutLocked := RunFullAnalysis(base)

	withLocked := base
	withLocked.MonthlyLockedSavings = 2_500
	resLocked := RunFullAnalysis(withLocked)

	if resLocked.CorpusAtFireRealP50 <= withoutLocked.CorpusAtFireRealP50*1.25 {
		t.Fatalf("expected Age-60 monthly contributions to meaningfully grow corpus at FIRE: with=%.0f vs without=%.0f",
			resLocked.CorpusAtFireRealP50, withoutLocked.CorpusAtFireRealP50)
	}
	if resLocked.SuccessRate <= withoutLocked.SuccessRate {
		t.Fatalf("expected Age-60 monthly contributions to boost success rate: with=%.1f%% vs without=%.1f%%",
			resLocked.SuccessRate, withoutLocked.SuccessRate)
	}
}

func TestGlobalJurisdictionsAustraliaEuropeNZAndNewIndianCities(t *testing.T) {
	// 1. Australia: Sydney -> Melbourne & Gold Coast + Superannuation Age 60 unlock + Age 67 Centrelink Age Pension
	auPlan := DefaultCountryTemplate(data.CountryAustralia)
	auPlan.NumPaths = 800
	auPlan.Seed = 101
	resAU := RunFullAnalysis(auPlan)
	if resAU.GeoArbitrage.MoveSuccessRate <= resAU.GeoArbitrage.StaySuccessRate {
		t.Fatalf("expected AU Sydney -> Gold Coast survival (%.1f%%) > Sydney stay (%.1f%%)",
			resAU.GeoArbitrage.MoveSuccessRate, resAU.GeoArbitrage.StaySuccessRate)
	}
	// Check Melbourne city lookup & simulation
	auMelb := auPlan
	auMelb.RetirementCityID = "au-melbourne"
	resMelb := RunFullAnalysis(auMelb)
	if resMelb.SuccessRate <= 0 || len(resMelb.CrisisReplays) != 4 {
		t.Fatalf("expected valid AU Melbourne analysis & 4 AU historical crisis replays, got %.1f%% / %d",
			resMelb.SuccessRate, len(resMelb.CrisisReplays))
	}

	// 2. Europe & Switzerland: Zurich Suburbs -> Algarve (Portugal) & Southern Italy (7% Flat Tax)
	euPlan := DefaultCountryTemplate(data.CountryEurope)
	euPlan.NumPaths = 800
	euPlan.Seed = 202
	resEU := RunFullAnalysis(euPlan)
	if resEU.GeoArbitrage.MoveSuccessRate <= resEU.GeoArbitrage.StaySuccessRate {
		t.Fatalf("expected Zurich Suburbs -> Algarve survival (%.1f%%) > Zurich stay (%.1f%%)",
			resEU.GeoArbitrage.MoveSuccessRate, resEU.GeoArbitrage.StaySuccessRate)
	}
	if resEU.GeoArbitrage.CorpusSavedReal < 250_000 {
		t.Fatalf("expected huge corpus savings moving Zurich Suburbs -> Algarve, got €%.0f", resEU.GeoArbitrage.CorpusSavedReal)
	}

	// Lugano -> Southern Italy (Puglia/Sicily)
	luganoToItaly := euPlan
	luganoToItaly.CurrentCityID = "eu-lugano"
	luganoToItaly.RetirementCityID = "eu-south-italy"
	resLugIt := RunFullAnalysis(luganoToItaly)
	if resLugIt.GeoArbitrage.MoveSuccessRate <= resLugIt.GeoArbitrage.StaySuccessRate {
		t.Fatalf("expected Lugano -> Southern Italy survival (%.1f%%) > Lugano stay (%.1f%%)",
			resLugIt.GeoArbitrage.MoveSuccessRate, resLugIt.GeoArbitrage.StaySuccessRate)
	}

	// 3. New Zealand: Auckland -> Tauranga/Nelson & KiwiSaver Age 65 unlock check
	nzPlan := DefaultCountryTemplate(data.CountryNewZealand)
	nzPlan.NumPaths = 800
	nzPlan.Seed = 303
	resNZ := RunFullAnalysis(nzPlan)
	if resNZ.SuccessRate <= 0 {
		t.Fatalf("expected positive NZ FIRE success rate, got %.1f%%", resNZ.SuccessRate)
	}
	// Verify KiwiSaver stays locked at age 62 (<65) and unlocks at age 65!
	idx62 := 62 - nzPlan.CurrentAge
	idx65 := 65 - nzPlan.CurrentAge
	if resNZ.YearlyBands[idx62].LockedRealP50 <= 0 {
		t.Fatalf("expected NZ KiwiSaver to remain locked at age 62, got LockedRealP50=%.0f", resNZ.YearlyBands[idx62].LockedRealP50)
	}
	if resNZ.YearlyBands[idx65].LockedRealP50 != 0 {
		t.Fatalf("expected NZ KiwiSaver to unlock completely at age 65, got LockedRealP50=%.0f", resNZ.YearlyBands[idx65].LockedRealP50)
	}

	// 4. New Indian Cities: Dehradun, Kodaikanal, Kochi (Cochin) + Cross-Border India -> Portugal
	for _, cityID := range []string{"in-dehradun", "in-kodaikanal", "in-kochi"} {
		inCityPlan := DefaultCountryTemplate(data.CountryIndia)
		inCityPlan.RetirementCityID = cityID
		inCityPlan.NumPaths = 400
		resInCity := RunFullAnalysis(inCityPlan)
		if resInCity.GeoArbitrage.MoveSuccessRate <= resInCity.GeoArbitrage.StaySuccessRate {
			t.Fatalf("expected Bengaluru -> %s survival (%.1f%%) > Bengaluru stay (%.1f%%)",
				cityID, resInCity.GeoArbitrage.MoveSuccessRate, resInCity.GeoArbitrage.StaySuccessRate)
		}
	}

	t.Logf("🌏 Global Check: AU=%.1f%%, EU(ZH->PT)=%.1f%%, EU(Lugano->Puglia)=%.1f%%, NZ=%.1f%%",
		resAU.SuccessRate, resEU.SuccessRate, resLugIt.SuccessRate, resNZ.SuccessRate)
}
