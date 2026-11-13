package tax

import (
	"math"

	"github.com/ausanyal/firesage/internal/data"
)

// NZTaxAnnual computes New Zealand resident income & investment taxes under IRD 2025-26 rules:
//   - 0% General Capital Gains Tax on realized equity gains (realizedLTCG is exempt from CGT!)
//   - Foreign Investment Fund (FIF) 5% Fair Dividend Rate (FDR) deemed annual return on global equities
//     (~72% offshore equity share of taxableEquityValue) exceeding the NZ$50,000 single / NZ$100,000 couple
//     de minimis cost threshold.
//   - Portfolio Investment Entity (PIE) 28% maximum Prescribed Investor Rate (PIR) cap when enablePIE is true
//     (otherwise FIF deemed income stacks onto individual PAYE marginal rates up to 39%).
//   - Individual PAYE brackets (10.5% / 17.5% / 30% / 33% / 39%) on ordinaryIncome (NZ Superannuation,
//     bond/term-deposit interest, part-time income).
func NZTaxAnnual(ordinaryIncome, taxableEquityValue float64, isCouple, enablePIE bool, cpiIndex float64) float64 {
	if cpiIndex <= 0 {
		cpiIndex = 1.0
	}
	if ordinaryIncome < 0 {
		ordinaryIncome = 0
	}
	if taxableEquityValue < 0 {
		taxableEquityValue = 0
	}

	adults := 1.0
	if isCouple {
		adults = 2.0
	}

	// Global offshore share of NZ taxable equity portfolio (~72% global / ~28% FIF-exempt NZX & ASX listed shares)
	offshoreEquities := taxableEquityValue * 0.72
	deMinimis := 50_000.0 * cpiIndex * adults

	fifDeemedIncome := 0.0
	if offshoreEquities > deMinimis {
		// 5.0% Fair Dividend Rate (FDR) statutory deemed return on offshore share value
		fifDeemedIncome = offshoreEquities * 0.05
	}

	ordPerAdult := ordinaryIncome / adults
	fifPerAdult := fifDeemedIncome / adults

	// 2025-26 IRD Individual PAYE Tax Brackets (indexed by cumulative CPI)
	payeBands := []taxBand{
		{15_600 * cpiIndex, 0.105},
		{53_500 * cpiIndex, 0.175},
		{78_100 * cpiIndex, 0.300},
		{180_000 * cpiIndex, 0.330},
		{math.Inf(1), 0.390},
	}

	if enablePIE {
		// PIE multi-rate fund caps investment/FIF PIR at 10.5%, 17.5%, or max 28.0% based on individual income
		ordTaxPerAdult := progressiveTax(ordPerAdult, payeBands)
		totalTaxablePerAdult := ordPerAdult + fifPerAdult
		pir := 0.28
		switch {
		case ordPerAdult <= 15_600*cpiIndex && totalTaxablePerAdult <= 53_500*cpiIndex:
			pir = 0.105
		case ordPerAdult <= 53_500*cpiIndex && totalTaxablePerAdult <= 78_100*cpiIndex:
			pir = 0.175
		default:
			pir = 0.28
		}
		return (ordTaxPerAdult + fifPerAdult*pir) * adults
	}

	// Unoptimized direct brokerage holding: FIF deemed income stacks directly onto PAYE brackets (up to 39%)
	combinedTaxPerAdult := progressiveTax(ordPerAdult+fifPerAdult, payeBands)
	return combinedTaxPerAdult * adults
}

// NZHealthcareAnnual models New Zealand's universal public Health NZ + ACC acute system paired with
// Southern Cross / nib Private Surgical & Specialist insurance, capturing NZ's steep age-rated
// private surgical premium curve after age 60 plus GP/dental/aged-care out-of-pocket costs.
func NZHealthcareAnnual(age int, isCouple bool, city data.CityProfile, cpiIndex, medicalIndex float64) float64 {
	if cpiIndex <= 0 {
		cpiIndex = 1.0
	}
	if medicalIndex <= 0 {
		medicalIndex = 1.0
	}

	var perAdultNZD float64
	switch {
	case age < 45:
		perAdultNZD = 2_300
	case age < 55:
		perAdultNZD = 3_350
	case age < 65:
		perAdultNZD = 5_100
	case age < 75:
		perAdultNZD = 7_800
	default:
		perAdultNZD = 10_600
	}

	householdMult := 1.0
	if isCouple {
		householdMult = 1.90
	}

	return perAdultNZD*householdMult*city.HealthcareCostMult*medicalIndex + city.TertiaryMedicalBuffer*cpiIndex
}
