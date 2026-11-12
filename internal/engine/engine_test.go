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
