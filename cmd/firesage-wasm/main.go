//go:build js && wasm

package main

import (
	"encoding/json"
	"syscall/js"

	"github.com/ausanyal/firesage/internal/data"
	"github.com/ausanyal/firesage/internal/engine"
)

func main() {
	js.Global().Set("firesageGetMetadata", js.FuncOf(func(_ js.Value, _ []js.Value) any {
		usCurrPE, usMedPE := data.CurrentValuation(data.CountryUSA)
		inCurrPE, inMedPE := data.CurrentValuation(data.CountryIndia)

		payload := map[string]any{
			"cities": data.Cities(),
			"crises": map[string]any{
				"IN": data.CrisesByCountry(data.CountryIndia),
				"US": data.CrisesByCountry(data.CountryUSA),
			},
			"valuations": map[string]any{
				"IN": map[string]any{
					"metric":            "Nifty 50 Trailing P/E",
					"current":           inCurrPE,
					"historical_median": inMedPE,
					"history_span":      "1980–2025 (46 yrs: BSE Sensex TR / Nifty 50 TRI, RBI G-Sec, MoSPI CPI)",
				},
				"US": map[string]any{
					"metric":            "Shiller CAPE (P/E10)",
					"current":           usCurrPE,
					"historical_median": usMedPE,
					"history_span":      "1928–2025 (98 yrs: Shiller Yale / Damodaran NYU Stern, 10Y Treasury, BLS CPI)",
				},
			},
			"defaults": map[string]any{
				"IN": defaultTemplate(data.CountryIndia),
				"US": defaultTemplate(data.CountryUSA),
			},
			"fx_usd_to_inr": data.DefaultUSDToINR,
		}
		b, _ := json.Marshal(payload)
		return string(b)
	}))

	js.Global().Set("firesageRunAnalysis", js.FuncOf(func(_ js.Value, args []js.Value) any {
		if len(args) < 1 {
			return `{"error":"missing JSON input"}`
		}
		var req engine.PlanInput
		if err := json.Unmarshal([]byte(args[0].String()), &req); err != nil {
			errObj, _ := json.Marshal(map[string]string{"error": err.Error()})
			return string(errObj)
		}
		res := engine.RunFullAnalysis(req)
		b, _ := json.Marshal(res)
		return string(b)
	}))

	// Keep Go WASM runtime alive
	select {}
}

func defaultTemplate(country data.CountryCode) engine.PlanInput {
	if country == data.CountryUSA {
		return engine.NormalizePlanInput(engine.PlanInput{
			Country:                  data.CountryUSA,
			CurrentAge:               34,
			RetirementAge:            46,
			HorizonAge:               92,
			IsCouple:                 true,
			CurrentCityID:            "us-san-jose",
			RetirementCityID:         "us-raleigh",
			AutoScaleLifestyleOnMove: true,
			CurrentCorpus:            650_000,
			Buckets: engine.AccountBuckets{
				UseGranular:       true,
				TaxableEquity:     260_000,
				TaxableCostBasis:  0.65,
				TaxableDebtCash:   52_000,
				TaxDeferredLocked: 247_000,
				TaxFreeAccount:    91_000,
				PensionLocked:     0,
			},
			MonthlySavings:            3_250,
			MonthlyLockedSavings:      3_250,
			AnnualSavingsStepUp:       0.04,
			DesiredMonthlySpend:       10_500,
			Allocation:                engine.AssetAllocation{Equity: 0.72, Debt: 0.20, Gold: 0.03, Cash: 0.05, UseBondTent: true},
			ExpenseRatioBps:           12,
			WithdrawalStrategy:        engine.StrategyGuytonKlinger,
			GuardrailFloorPct:         0.82,
			GuardrailCeilingPct:       1.25,
			UseSpendingSmile:          true,
			SocialSecurityMonthlyReal: 3_400,
			SocialSecurityStartAge:    67,
			EnableTaxOptimization:     true,
			Mode:                      engine.ModeBootstrap,
			NumPaths:                  3000,
		})
	}
	return engine.NormalizePlanInput(engine.PlanInput{
		Country:                  data.CountryIndia,
		CurrentAge:               33,
		RetirementAge:            45,
		HorizonAge:               90,
		IsCouple:                 true,
		CurrentCityID:            "in-bengaluru",
		RetirementCityID:         "in-goa",
		AutoScaleLifestyleOnMove: true,
		CurrentCorpus:            18_000_000,
		Buckets: engine.AccountBuckets{
			UseGranular:       true,
			TaxableEquity:     8_640_000,
			TaxableCostBasis:  0.65,
			TaxableDebtCash:   2_520_000,
			TaxDeferredLocked: 3_600_000,
			TaxFreeAccount:    1_800_000,
			PensionLocked:     1_440_000,
		},
		MonthlySavings:            105_000,
		MonthlyLockedSavings:      45_000,
		AnnualSavingsStepUp:       0.07,
		DesiredMonthlySpend:       140_000,
		Allocation:                engine.AssetAllocation{Equity: 0.65, Debt: 0.22, Gold: 0.08, Cash: 0.05, UseBondTent: true},
		ExpenseRatioBps:           25,
		WithdrawalStrategy:        engine.StrategyGuytonKlinger,
		GuardrailFloorPct:         0.82,
		GuardrailCeilingPct:       1.25,
		UseSpendingSmile:          true,
		SocialSecurityMonthlyReal: 0,
		SocialSecurityStartAge:    60,
		EnableTaxOptimization:     true,
		Mode:                      engine.ModeBootstrap,
		NumPaths:                  3000,
	})
}
