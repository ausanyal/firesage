package engine

import "github.com/ausanyal/firesage/internal/data"

// BuildMetadataPayload constructs the complete /api/v1/metadata & WASM metadata object
// across all 5 supported jurisdictions (India, USA, Australia, Europe & Switzerland, New Zealand).
func BuildMetadataPayload() map[string]any {
	inCurrPE, inMedPE := data.CurrentValuation(data.CountryIndia)
	usCurrPE, usMedPE := data.CurrentValuation(data.CountryUSA)
	auCurrPE, auMedPE := data.CurrentValuation(data.CountryAustralia)
	euCurrPE, euMedPE := data.CurrentValuation(data.CountryEurope)
	nzCurrPE, nzMedPE := data.CurrentValuation(data.CountryNewZealand)

	return map[string]any{
		"cities": data.Cities(),
		"crises": map[string]any{
			"IN": data.CrisesByCountry(data.CountryIndia),
			"US": data.CrisesByCountry(data.CountryUSA),
			"AU": data.CrisesByCountry(data.CountryAustralia),
			"EU": data.CrisesByCountry(data.CountryEurope),
			"NZ": data.CrisesByCountry(data.CountryNewZealand),
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
			"AU": map[string]any{
				"metric":            "ASX 200 Trailing P/E",
				"current":           auCurrPE,
				"historical_median": auMedPE,
				"history_span":      "1980–2025 (46 yrs: ASX All Ords / ASX 200 Accumulation, RBA 10Y CGS, ABS CPI)",
			},
			"EU": map[string]any{
				"metric":            "MSCI Europe / SPI CAPE-P/E",
				"current":           euCurrPE,
				"historical_median": euMedPE,
				"history_span":      "1980–2025 (46 yrs: MSCI Europe / STOXX 600 TR, 10Y Bund/Sovereign, Eurostat HICP)",
			},
			"NZ": map[string]any{
				"metric":            "NZX 50 / Global PIE P/E",
				"current":           nzCurrPE,
				"historical_median": nzMedPE,
				"history_span":      "1980–2025 (46 yrs: NZX 50 Gross / Global Equity, RBNZ 10Y NZGB, Stats NZ CPI)",
			},
		},
		"defaults": map[string]any{
			"IN": DefaultCountryTemplate(data.CountryIndia),
			"US": DefaultCountryTemplate(data.CountryUSA),
			"AU": DefaultCountryTemplate(data.CountryAustralia),
			"EU": DefaultCountryTemplate(data.CountryEurope),
			"NZ": DefaultCountryTemplate(data.CountryNewZealand),
		},
		"fx_usd_to_inr":    data.DefaultUSDToINR,
		"fx_rates_per_usd": data.FXRatesPerUSD(),
	}
}

// DefaultCountryTemplate returns a realistic, domain-calibrated starting household FIRE scenario
// for each of the 5 supported jurisdictions.
func DefaultCountryTemplate(country data.CountryCode) PlanInput {
	switch country {
	case data.CountryUSA:
		return NormalizePlanInput(PlanInput{
			Country:                  data.CountryUSA,
			CurrentAge:               34,
			RetirementAge:            46,
			HorizonAge:               92,
			IsCouple:                 true,
			CurrentCityID:            "us-san-jose",
			RetirementCityID:         "us-raleigh",
			AutoScaleLifestyleOnMove: true,
			CurrentCorpus:            650_000,
			Buckets: AccountBuckets{
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
			Allocation:                AssetAllocation{Equity: 0.72, Debt: 0.20, Gold: 0.03, Cash: 0.05, UseBondTent: true},
			ExpenseRatioBps:           12,
			WithdrawalStrategy:        StrategyGuytonKlinger,
			GuardrailFloorPct:         0.82,
			GuardrailCeilingPct:       1.25,
			UseSpendingSmile:          true,
			SocialSecurityMonthlyReal: 3_400,
			SocialSecurityStartAge:    67,
			EnableTaxOptimization:     true,
			Mode:                      ModeBootstrap,
			NumPaths:                  3000,
		})

	case data.CountryAustralia:
		return NormalizePlanInput(PlanInput{
			Country:                  data.CountryAustralia,
			CurrentAge:               34,
			RetirementAge:            48,
			HorizonAge:               92,
			IsCouple:                 true,
			CurrentCityID:            "au-sydney",
			RetirementCityID:         "au-gold-coast",
			AutoScaleLifestyleOnMove: true,
			CurrentCorpus:            580_000,
			Buckets: AccountBuckets{
				UseGranular:       true,
				TaxableEquity:     220_000,
				TaxableCostBasis:  0.65,
				TaxableDebtCash:   48_000,
				TaxDeferredLocked: 312_000, // Superannuation (Preservation Age 60)
				TaxFreeAccount:    0,
				PensionLocked:     0,
			},
			MonthlySavings:            3_200,
			MonthlyLockedSavings:      2_300, // Super Guarantee (12%) + Salary Sacrifice
			AnnualSavingsStepUp:       0.035,
			DesiredMonthlySpend:       6_800, // Sydney Standard (~A$5,670/mo on Gold Coast)
			Allocation:                AssetAllocation{Equity: 0.74, Debt: 0.18, Gold: 0.03, Cash: 0.05, UseBondTent: true},
			ExpenseRatioBps:           18,
			WithdrawalStrategy:        StrategyGuytonKlinger,
			GuardrailFloorPct:         0.82,
			GuardrailCeilingPct:       1.25,
			UseSpendingSmile:          true,
			SocialSecurityMonthlyReal: 0, // Centrelink Age Pension (Age 67) is auto-computed via Assets Test!
			SocialSecurityStartAge:    67,
			EnableTaxOptimization:     true,
			Mode:                      ModeBootstrap,
			NumPaths:                  3000,
		})

	case data.CountryEurope:
		return NormalizePlanInput(PlanInput{
			Country:                  data.CountryEurope,
			CurrentAge:               35,
			RetirementAge:            48,
			HorizonAge:               92,
			IsCouple:                 true,
			CurrentCityID:            "eu-zurich-suburbs",
			RetirementCityID:         "eu-algarve",
			AutoScaleLifestyleOnMove: true,
			CurrentCorpus:            540_000,
			Buckets: AccountBuckets{
				UseGranular:       true,
				TaxableEquity:     238_000,
				TaxableCostBasis:  0.65,
				TaxableDebtCash:   54_000,
				TaxDeferredLocked: 173_000, // Swiss Pillar 2 (BVG) / Occupational Pension
				TaxFreeAccount:    75_000,  // Pillar 3a / Tax-Advantaged Wrapper
				PensionLocked:     0,
			},
			MonthlySavings:            2_900,
			MonthlyLockedSavings:      1_900,
			AnnualSavingsStepUp:       0.03,
			DesiredMonthlySpend:       5_600, // Zurich Suburbs Standard -> scales down sharply in Algarve/South Italy!
			Allocation:                AssetAllocation{Equity: 0.68, Debt: 0.23, Gold: 0.04, Cash: 0.05, UseBondTent: true},
			ExpenseRatioBps:           18,
			WithdrawalStrategy:        StrategyGuytonKlinger,
			GuardrailFloorPct:         0.82,
			GuardrailCeilingPct:       1.25,
			UseSpendingSmile:          true,
			SocialSecurityMonthlyReal: 2_100, // State AHV / European Pillar 1 Pension at 65
			SocialSecurityStartAge:    65,
			EnableTaxOptimization:     true,
			Mode:                      ModeBootstrap,
			NumPaths:                  3000,
		})

	case data.CountryNewZealand:
		return NormalizePlanInput(PlanInput{
			Country:                  data.CountryNewZealand,
			CurrentAge:               34,
			RetirementAge:            49,
			HorizonAge:               92,
			IsCouple:                 true,
			CurrentCityID:            "nz-auckland",
			RetirementCityID:         "nz-tauranga-nelson",
			AutoScaleLifestyleOnMove: true,
			CurrentCorpus:            540_000,
			Buckets: AccountBuckets{
				UseGranular:       true,
				TaxableEquity:     248_000,
				TaxableCostBasis:  0.65,
				TaxableDebtCash:   54_000,
				TaxDeferredLocked: 238_000, // KiwiSaver (Preserved until Age 65!)
				TaxFreeAccount:    0,
				PensionLocked:     0,
			},
			MonthlySavings:            2_900,
			MonthlyLockedSavings:      1_450, // Employee + Employer KiwiSaver
			AnnualSavingsStepUp:       0.035,
			DesiredMonthlySpend:       6_000, // Auckland Standard
			Allocation:                AssetAllocation{Equity: 0.72, Debt: 0.20, Gold: 0.03, Cash: 0.05, UseBondTent: true},
			ExpenseRatioBps:           22,
			WithdrawalStrategy:        StrategyGuytonKlinger,
			GuardrailFloorPct:         0.82,
			GuardrailCeilingPct:       1.25,
			UseSpendingSmile:          true,
			SocialSecurityMonthlyReal: 3_700, // Universal non-means-tested NZ Superannuation (Couple) at Age 65
			SocialSecurityStartAge:    65,
			EnableTaxOptimization:     true,
			Mode:                      ModeBootstrap,
			NumPaths:                  3000,
		})

	default:
		return NormalizePlanInput(PlanInput{
			Country:                  data.CountryIndia,
			CurrentAge:               33,
			RetirementAge:            45,
			HorizonAge:               90,
			IsCouple:                 true,
			CurrentCityID:            "in-bengaluru",
			RetirementCityID:         "in-goa",
			AutoScaleLifestyleOnMove: true,
			CurrentCorpus:            18_000_000,
			Buckets: AccountBuckets{
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
			Allocation:                AssetAllocation{Equity: 0.65, Debt: 0.22, Gold: 0.08, Cash: 0.05, UseBondTent: true},
			ExpenseRatioBps:           25,
			WithdrawalStrategy:        StrategyGuytonKlinger,
			GuardrailFloorPct:         0.82,
			GuardrailCeilingPct:       1.25,
			UseSpendingSmile:          true,
			SocialSecurityMonthlyReal: 0,
			SocialSecurityStartAge:    60,
			EnableTaxOptimization:     true,
			Mode:                      ModeBootstrap,
			NumPaths:                  3000,
		})
	}
}
