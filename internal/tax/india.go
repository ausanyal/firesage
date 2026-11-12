package tax

import (
	"math"

	"github.com/ausanyal/firesage/internal/data"
)

// IndiaNewRegimeSlabTax computes Indian income tax under the FY 2025-26 / 2026 New Tax Regime
// on slab-taxable income (e.g., Debt MF gains, FD interest, NPS annuity, part-time consulting),
// with brackets and Section 87A rebate scaled by cumulative headline inflation (cpiIndex).
// Note: Per Section 112A of the Income Tax Act, Equity LTCG is NOT eligible for Section 87A rebate
// and is taxed separately via IndiaEquityLTCGTax.
func IndiaNewRegimeSlabTax(slabIncome float64, cpiIndex float64) float64 {
	if slabIncome <= 0 {
		return 0
	}
	if cpiIndex <= 0 {
		cpiIndex = 1.0
	}

	// Section 87A rebate threshold under New Regime: ₹12,00,000 real equivalent
	rebateLimit := 1_200_000.0 * cpiIndex
	if slabIncome <= rebateLimit {
		return 0
	}

	// Slabs in base-year rupees scaled by cpiIndex:
	// 0-4L: 0%, 4-8L: 5%, 8-12L: 10%, 12-16L: 15%, 16-20L: 20%, 20-24L: 25%, >24L: 30%
	type bracket struct {
		upTo float64
		rate float64
	}
	brackets := []bracket{
		{400_000 * cpiIndex, 0.00},
		{800_000 * cpiIndex, 0.05},
		{1_200_000 * cpiIndex, 0.10},
		{1_600_000 * cpiIndex, 0.15},
		{2_000_000 * cpiIndex, 0.20},
		{2_400_000 * cpiIndex, 0.25},
		{math.Inf(1), 0.30},
	}

	tax := 0.0
	prev := 0.0
	for _, b := range brackets {
		if slabIncome <= prev {
			break
		}
		taxable := math.Min(slabIncome, b.upTo) - prev
		if taxable > 0 {
			tax += taxable * b.rate
		}
		prev = b.upTo
	}

	// Marginal relief just above ₹12L threshold
	excessOverRebate := slabIncome - rebateLimit
	if tax > excessOverRebate {
		tax = excessOverRebate
	}

	// 4% Health & Education Cess
	return tax * 1.04
}

// IndiaEquityLTCGTax computes Section 112A Long-Term Capital Gains tax (12.5% + 4% cess = 13.0%)
// on realized equity / arbitrage gains exceeding the ₹1.25 Lakh annual exemption (inflation-indexed).
func IndiaEquityLTCGTax(realizedGain float64, cpiIndex float64) float64 {
	if realizedGain <= 0 {
		return 0
	}
	if cpiIndex <= 0 {
		cpiIndex = 1.0
	}
	exemption := 125_000.0 * cpiIndex
	if realizedGain <= exemption {
		return 0
	}
	return (realizedGain - exemption) * 0.125 * 1.04
}

// IndiaAnnualLTCGExemption returns the inflation-indexed ₹1.25 Lakh annual tax-free LTCG harvest allowance.
func IndiaAnnualLTCGExemption(cpiIndex float64) float64 {
	if cpiIndex <= 0 {
		cpiIndex = 1.0
	}
	return 125_000.0 * cpiIndex
}

// IndiaHealthcareAnnual computes realistic annual self-funded Indian healthcare expenditure
// (Retail Family Floater + ₹1Cr Super Top-up premiums + uninsured OOP diagnostics/care buffer),
// scaled by age band, household size, city Zone multiplier, tertiary travel buffer, and cumulative Medical CPI.
func IndiaHealthcareAnnual(age int, isCouple bool, city data.CityProfile, cpiIndex float64, medicalIndex float64) float64 {
	if medicalIndex <= 0 {
		medicalIndex = 1.0
	}
	if cpiIndex <= 0 {
		cpiIndex = 1.0
	}

	// Base annual insurance + OOP medical spend in today's rupees (Zone-1 Bengaluru reference)
	var baseToday float64
	switch {
	case age < 35:
		baseToday = 28_000
	case age < 45:
		baseToday = 44_000
	case age < 55:
		baseToday = 74_000
	case age < 65:
		baseToday = 128_000
	case age < 75:
		baseToday = 205_000
	default:
		baseToday = 315_000 // Includes senior diagnostics, home care & uncovered prescriptions
	}

	if isCouple {
		baseToday *= 1.65 // Family floater efficiency vs 2x standalone
	}

	zoneAdjusted := baseToday * city.HealthcareCostMult * medicalIndex
	tertiaryTravel := city.TertiaryMedicalBuffer * cpiIndex
	return zoneAdjusted + tertiaryTravel
}
