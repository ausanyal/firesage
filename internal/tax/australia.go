package tax

import (
	"math"

	"github.com/ausanyal/firesage/internal/data"
)

// AustraliaIncomeTax computes Australian resident income & capital gains tax under the
// FY 2025-26 Stage 3 statutory tax brackets, incorporating:
//   - 50% Capital Gains Tax (CGT) discount for assets held > 12 months
//   - Individual-level assessment (50/50 split across partners for couples on shared portfolio income)
//   - Low Income Tax Offset (LITO)
//   - 2.0% Medicare Levy with low-income phase-in shading
//   - Optional ASX dividend imputation / franking credit offset (enableFrankingCredits)
func AustraliaIncomeTax(ordinaryIncome, realizedLTCG float64, isCouple, enableFrankingCredits bool, cpiIndex float64) float64 {
	if cpiIndex <= 0 {
		cpiIndex = 1.0
	}
	if ordinaryIncome < 0 {
		ordinaryIncome = 0
	}
	if realizedLTCG < 0 {
		realizedLTCG = 0
	}

	// 50% CGT discount for individuals holding investments >= 12 months
	discountedCGT := realizedLTCG * 0.50
	totalTaxable := ordinaryIncome + discountedCGT
	if totalTaxable <= 0 {
		return 0
	}

	adults := 1.0
	if isCouple {
		adults = 2.0
	}
	perAdultTaxable := totalTaxable / adults

	// FY 2025-26 Stage 3 Resident Tax Brackets (indexed by cumulative CPI)
	type bracket struct {
		upTo float64
		rate float64
	}
	brackets := []bracket{
		{18_200 * cpiIndex, 0.00},
		{45_000 * cpiIndex, 0.16},
		{135_000 * cpiIndex, 0.30},
		{190_000 * cpiIndex, 0.37},
		{math.Inf(1), 0.45},
	}

	baseTax := 0.0
	prev := 0.0
	for _, b := range brackets {
		if perAdultTaxable <= prev {
			break
		}
		chunk := math.Min(perAdultTaxable, b.upTo) - prev
		if chunk > 0 {
			baseTax += chunk * b.rate
		}
		prev = b.upTo
	}

	// Low Income Tax Offset (LITO): max A$700 up to A$37,500, phases out by A$66,667
	lito := 0.0
	t1 := 37_500.0 * cpiIndex
	t2 := 45_000.0 * cpiIndex
	t3 := 66_667.0 * cpiIndex
	if perAdultTaxable <= t1 {
		lito = 700.0 * cpiIndex
	} else if perAdultTaxable <= t2 {
		lito = 700.0*cpiIndex - (perAdultTaxable-t1)*0.05
	} else if perAdultTaxable < t3 {
		lito = 325.0*cpiIndex - (perAdultTaxable-t2)*0.015
	}
	netIncomeTaxPerAdult := math.Max(0, baseTax-math.Max(0, lito))

	// 2.0% Medicare Levy with low-income threshold shade-in (10% of excess over threshold)
	medLow := 26_000.0 * cpiIndex
	if isCouple {
		// Family Medicare Levy threshold split per partner (~A$43,846 / 2)
		medLow = 21_923.0 * cpiIndex
	}
	medLevyPerAdult := 0.0
	if perAdultTaxable > medLow {
		shadeIn := (perAdultTaxable - medLow) * 0.10
		fullLevy := perAdultTaxable * 0.02
		medLevyPerAdult = math.Min(shadeIn, fullLevy)
	}

	householdTax := (netIncomeTaxPerAdult + medLevyPerAdult) * adults

	// Franking credit imputation offset on domestic ASX equity distributions (30% corporate imputation)
	if enableFrankingCredits && householdTax > 0 {
		// Calibrated imputation credit reduction (~18% effective tax offset on balanced AU/Global equity draw)
		householdTax *= 0.82
	}

	return householdTax
}

// AustraliaAgePensionAnnual computes the means-tested Services Australia Age Pension (from Age 67)
// for a homeowner household, applying the statutory Centrelink Assets Test taper
// (A$3.00/fortnight = A$78.00/year reduction per A$1,000 of assessable financial assets above threshold).
func AustraliaAgePensionAnnual(age int, isCouple bool, assessableAssets, cpiIndex float64) float64 {
	if age < 67 {
		return 0
	}
	if cpiIndex <= 0 {
		cpiIndex = 1.0
	}
	if assessableAssets < 0 {
		assessableAssets = 0
	}

	// Maximum 2025-26 full Age Pension including Pension & Energy Supplements
	maxAnnual := 32_214.0 * cpiIndex
	assetThreshold := 314_000.0 * cpiIndex
	if isCouple {
		maxAnnual = 48_516.0 * cpiIndex
		assetThreshold = 470_000.0 * cpiIndex
	}

	if assessableAssets <= assetThreshold {
		return maxAnnual
	}

	excessThousands := (assessableAssets - assetThreshold) / 1_000.0
	reduction := excessThousands * 78.0 // A$3/fortnight * 26 fortnights = A$78/yr per A$1,000
	return math.Max(0, maxAnnual-reduction)
}

// AustraliaSuperMinDrawRate returns the statutory minimum annual drawdown fraction for an
// Australian Account-Based Superannuation Pension (tax-free from Preservation Age 60+).
func AustraliaSuperMinDrawRate(age int) float64 {
	switch {
	case age < 60:
		return 0
	case age < 65:
		return 0.04
	case age < 75:
		return 0.05
	case age < 80:
		return 0.06
	case age < 85:
		return 0.07
	case age < 90:
		return 0.09
	case age < 95:
		return 0.11
	default:
		return 0.14
	}
}

// AustraliaHealthcareAnnual models Australian universal Medicare + Private Hospital (Silver+/Gold)
// & Extras cover (avoiding Medicare Levy Surcharge & LHC loading) + specialist gap/dental/aged-care OOP.
func AustraliaHealthcareAnnual(age int, isCouple bool, city data.CityProfile, cpiIndex, medicalIndex float64) float64 {
	if cpiIndex <= 0 {
		cpiIndex = 1.0
	}
	if medicalIndex <= 0 {
		medicalIndex = 1.0
	}

	var perAdultToday float64
	switch {
	case age < 45:
		perAdultToday = 3_550
	case age < 55:
		perAdultToday = 4_250
	case age < 65:
		perAdultToday = 5_100
	case age < 75:
		perAdultToday = 5_950
	default:
		perAdultToday = 7_600
	}

	householdMult := 1.0
	if isCouple {
		householdMult = 1.92
	}

	return perAdultToday*householdMult*city.HealthcareCostMult*medicalIndex + city.TertiaryMedicalBuffer*cpiIndex
}
