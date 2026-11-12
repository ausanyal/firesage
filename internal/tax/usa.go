package tax

import (
	"math"

	"github.com/ausanyal/firesage/internal/data"
)

// USFederalTax computes US Federal ordinary income tax + stacked Long-Term Capital Gains tax
// + 3.8% Net Investment Income Tax (NIIT), with brackets and standard deductions indexed by cpiIndex.
func USFederalTax(ordinaryIncome float64, ltcg float64, isJoint bool, cpiIndex float64) float64 {
	if cpiIndex <= 0 {
		cpiIndex = 1.0
	}
	if ordinaryIncome < 0 {
		ordinaryIncome = 0
	}
	if ltcg < 0 {
		ltcg = 0
	}

	stdDed := 15_400.0 * cpiIndex
	if isJoint {
		stdDed = 30_800.0 * cpiIndex
	}

	// Ordinary income absorbs standard deduction first; any unused deduction offsets LTCG
	taxableOrd := math.Max(0, ordinaryIncome-stdDed)
	unusedDed := math.Max(0, stdDed-ordinaryIncome)
	taxableLTCG := math.Max(0, ltcg-unusedDed)

	type bracket struct {
		upTo float64
		rate float64
	}
	var ordBrackets []bracket
	if isJoint {
		ordBrackets = []bracket{
			{23_850 * cpiIndex, 0.10},
			{96_950 * cpiIndex, 0.12},
			{206_700 * cpiIndex, 0.22},
			{394_600 * cpiIndex, 0.24},
			{501_050 * cpiIndex, 0.32},
			{751_600 * cpiIndex, 0.35},
			{math.Inf(1), 0.37},
		}
	} else {
		ordBrackets = []bracket{
			{11_925 * cpiIndex, 0.10},
			{48_475 * cpiIndex, 0.12},
			{103_350 * cpiIndex, 0.22},
			{197_300 * cpiIndex, 0.24},
			{250_525 * cpiIndex, 0.32},
			{626_350 * cpiIndex, 0.35},
			{math.Inf(1), 0.37},
		}
	}

	fedOrdTax := 0.0
	prev := 0.0
	for _, b := range ordBrackets {
		if taxableOrd <= prev {
			break
		}
		chunk := math.Min(taxableOrd, b.upTo) - prev
		if chunk > 0 {
			fedOrdTax += chunk * b.rate
		}
		prev = b.upTo
	}

	// LTCG stacks on top of taxableOrd
	zeroCap := 48_350.0 * cpiIndex
	fifteenCap := 533_400.0 * cpiIndex
	if isJoint {
		zeroCap = 96_700.0 * cpiIndex
		fifteenCap = 600_050.0 * cpiIndex
	}

	fedLTCGTax := 0.0
	if taxableLTCG > 0 {
		stackStart := taxableOrd
		stackEnd := taxableOrd + taxableLTCG

		// 15% band
		b15Start := math.Max(stackStart, zeroCap)
		b15End := math.Min(stackEnd, fifteenCap)
		if b15End > b15Start {
			fedLTCGTax += (b15End - b15Start) * 0.15
		}

		// 20% band
		b20Start := math.Max(stackStart, fifteenCap)
		if stackEnd > b20Start {
			fedLTCGTax += (stackEnd - b20Start) * 0.20
		}
	}

	// 3.8% Net Investment Income Tax (NIIT) — statutory $200k/$250k thresholds (unindexed by law,
	// though over multi-decade real comparisons we gently index half or statutory; IRS keeps nominal,
	// we index by cpiIndex so real comparison stays consistent across 50-year horizons).
	niitThreshold := 200_000.0 * cpiIndex
	if isJoint {
		niitThreshold = 250_000.0 * cpiIndex
	}
	magi := ordinaryIncome + ltcg
	niit := 0.0
	if magi > niitThreshold && ltcg > 0 {
		niit = math.Min(ltcg, magi-niitThreshold) * 0.038
	}

	return fedOrdTax + fedLTCGTax + niit
}

// USStateTax computes state income & capital gains tax for major Tier-1 and Tier-2 US states.
func USStateTax(stateCode string, ordinaryIncome float64, ltcg float64, isJoint bool, cpiIndex float64) float64 {
	if cpiIndex <= 0 {
		cpiIndex = 1.0
	}
	totalIncome := math.Max(0, ordinaryIncome) + math.Max(0, ltcg)
	if totalIncome <= 0 {
		return 0
	}

	switch stateCode {
	case "TX", "FL", "NV", "TN", "NH":
		return 0

	case "WA":
		// Washington: 0% on ordinary income / Roth conversions / 401(k); 7% excise on LTCG > ~$270k
		waThreshold := 270_000.0 * cpiIndex
		if ltcg > waThreshold {
			return (ltcg - waThreshold) * 0.07
		}
		return 0

	case "NC":
		// North Carolina (Raleigh): Flat 4.25% after NC standard deduction ($12,750 Single / $25,500 MFJ)
		ded := 12_750.0 * cpiIndex
		if isJoint {
			ded = 25_500.0 * cpiIndex
		}
		return math.Max(0, totalIncome-ded) * 0.0425

	case "CO":
		// Colorado (Denver): Flat 4.40% after federal-linked standard deduction
		ded := 15_400.0 * cpiIndex
		if isJoint {
			ded = 30_800.0 * cpiIndex
		}
		return math.Max(0, totalIncome-ded) * 0.044

	case "CA":
		// California (San Jose / Silicon Valley): Taxes LTCG & Roth conversions as ordinary income!
		ded := 5_540.0 * cpiIndex
		mult := 1.0
		if isJoint {
			ded = 11_080.0 * cpiIndex
			mult = 2.0
		}
		taxable := math.Max(0, totalIncome-ded)
		type band struct {
			upTo float64
			rate float64
		}
		bands := []band{
			{10_756 * mult * cpiIndex, 0.01},
			{25_499 * mult * cpiIndex, 0.02},
			{40_245 * mult * cpiIndex, 0.04},
			{55_866 * mult * cpiIndex, 0.06},
			{70_606 * mult * cpiIndex, 0.08},
			{360_659 * mult * cpiIndex, 0.093},
			{432_787 * mult * cpiIndex, 0.103},
			{721_314 * mult * cpiIndex, 0.113},
			{1_000_000 * cpiIndex, 0.123},
			{math.Inf(1), 0.133},
		}
		tax := 0.0
		prev := 0.0
		for _, b := range bands {
			if taxable <= prev {
				break
			}
			chunk := math.Min(taxable, b.upTo) - prev
			if chunk > 0 {
				tax += chunk * b.rate
			}
			prev = b.upTo
		}
		return tax

	case "NY":
		// Combined NYS + NYC progressive approximation
		ded := 8_000.0 * cpiIndex
		if isJoint {
			ded = 16_050.0 * cpiIndex
		}
		taxable := math.Max(0, totalIncome-ded)
		if taxable <= 50_000*cpiIndex {
			return taxable * 0.082
		}
		if taxable <= 250_000*cpiIndex {
			return 50_000*cpiIndex*0.082 + (taxable-50_000*cpiIndex)*0.102
		}
		return 50_000*cpiIndex*0.082 + 200_000*cpiIndex*0.102 + (taxable-250_000*cpiIndex)*0.135

	default:
		// Moderate US state fallback (~4.5%)
		ded := 12_000.0 * cpiIndex
		if isJoint {
			ded = 24_000.0 * cpiIndex
		}
		return math.Max(0, totalIncome-ded) * 0.045
	}
}

// USHealthcareAnnual computes pre-65 ACA Marketplace health costs (including the restored 2026
// hard 400% FPL subsidy cliff!) and age 65+ Medicare Part B/D + Medigap + IRMAA surcharges.
func USHealthcareAnnual(age int, isCouple bool, magi float64, city data.CityProfile, cpiIndex float64, medicalIndex float64) float64 {
	if cpiIndex <= 0 {
		cpiIndex = 1.0
	}
	if medicalIndex <= 0 {
		medicalIndex = 1.0
	}

	// Age 65+: Medicare Part B + Part D + Medigap Plan G + OOP dental/vision + IRMAA
	if age >= 65 {
		perPersonToday := 5_100.0
		if age >= 78 {
			perPersonToday = 6_800.0 // Higher late-life OOP / long-term care tail
		}
		adults := 1.0
		if isCouple {
			adults = 2.0
		}
		baseMedicare := perPersonToday * adults * medicalIndex

		// IRMAA high-MAGI surcharge
		irmaaThreshold := 106_000.0 * cpiIndex
		if isCouple {
			irmaaThreshold = 212_000.0 * cpiIndex
		}
		irmaa := 0.0
		if magi > irmaaThreshold {
			irmaa = 2_400.0 * adults * medicalIndex
		}
		return baseMedicare + irmaa
	}

	// Pre-65 Early Retirement: ACA Marketplace with 2026 Hard 400% FPL Subsidy Cliff
	fpl := 15_650.0 * cpiIndex
	if isCouple {
		fpl = 21_150.0 * cpiIndex
	}

	// Unsubsidized Silver benchmark premium by age curve (in today's dollars per adult)
	var unsubPerAdultToday float64
	switch {
	case age < 35:
		unsubPerAdultToday = 4_900
	case age < 45:
		unsubPerAdultToday = 6_200
	case age < 55:
		unsubPerAdultToday = 9_400
	case age < 60:
		unsubPerAdultToday = 12_200
	default:
		unsubPerAdultToday = 14_600
	}
	adults := 1.0
	if isCouple {
		adults = 2.0
	}
	unsubsidizedPremium := unsubPerAdultToday * adults * city.HealthcareCostMult * medicalIndex
	oopAllowance := 2_100.0 * adults * medicalIndex

	fplRatio := 0.0
	if fpl > 0 {
		fplRatio = magi / fpl
	}

	// 2026 Hard 400% FPL Cliff: crossing 4.0x FPL loses 100% of premium tax credits!
	if fplRatio > 4.0 {
		return unsubsidizedPremium + oopAllowance
	}

	// Below 400% FPL: statutory sliding-scale expected contribution % of MAGI (2.0% up to 9.96%)
	var expectedPct float64
	switch {
	case fplRatio <= 1.5:
		expectedPct = 0.02
	case fplRatio <= 2.0:
		expectedPct = 0.02 + (fplRatio-1.5)*0.09 // 2% -> 6.5%
	case fplRatio <= 3.0:
		expectedPct = 0.065 + (fplRatio-2.0)*0.025 // 6.5% -> 9.0%
	default:
		expectedPct = 0.09 + (fplRatio-3.0)*0.0096 // 9.0% -> 9.96%
	}

	cappedPremium := math.Min(unsubsidizedPremium, math.Max(0, magi)*expectedPct)
	return cappedPremium + oopAllowance
}

// OptimalRothConversionCeiling computes the maximum tax-smart & ACA-safe MAGI target
// for automated Roth Conversion Ladder rungs during retirement.
func OptimalRothConversionCeiling(age int, isCouple bool, cpiIndex float64) float64 {
	if cpiIndex <= 0 {
		cpiIndex = 1.0
	}
	if age < 65 {
		// Stay safely at 385% FPL just underneath the 2026 400% FPL ACA healthcare subsidy cliff!
		fpl := 15_650.0 * cpiIndex
		if isCouple {
			fpl = 21_150.0 * cpiIndex
		}
		return 3.85 * fpl
	}
	// Ages 65+: ACA cliff is gone (Medicare active); fill top of 12% federal bracket + std deduction
	if isCouple {
		return (30_800.0 + 96_950.0) * cpiIndex
	}
	return (15_400.0 + 48_475.0) * cpiIndex
}

// USRMDDivisor returns the IRS Uniform Lifetime Table divisor under SECURE 2.0
// (Age 75 RMD start for individuals born 1960+, Age 73 for born 1951–1959).
func USRMDDivisor(age int, birthYear int) float64 {
	rmdStart := 75
	if birthYear < 1960 {
		rmdStart = 73
	}
	if age < rmdStart {
		return 0
	}
	// IRS Uniform Lifetime Table approximation (exact at key ages: 73->26.5, 75->24.6, 80->20.2, 85->16.0, 90->12.2, 95->8.9)
	divisor := 24.6 - float64(age-75)*0.84
	if divisor < 4.0 {
		divisor = 4.0
	}
	return divisor
}
