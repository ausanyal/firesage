package tax

import (
	"math"

	"github.com/ausanyal/firesage/internal/data"
)

// EuropeTaxAnnual computes country- and canton-specific European retirement income, capital gains,
// and net wealth taxes across FireSage's calibrated European destinations:
//   - "CH-ZH" / "CH-TI" (Switzerland: Zurich Suburbs & Lugano): 0% private equity CGT + progressive
//     cantonal/federal pension tax + cantonal net wealth tax (Vermögenssteuer).
//   - "IT-SOUTH" (Southern Italy Mezzogiorno <30k-pop towns): Article 24-ter 7% Flat Foreign Retiree Tax
//     for the first 10 years of incoming residency (when enableTaxOpt & isIncomingForeignResident),
//     reverting thereafter to 26% Italian CGT + progressive IRPEF.
//   - "GR" (Chania/Crete & Peloponnese, Greece): Article 5B 7% Flat Foreign Retiree Tax for 15 years
//     (when enableTaxOpt & isIncomingForeignResident), or 0% UCITS ETF CGT / 15% investment tax + progressive pension slabs.
//   - "PT" (Algarve & Cascais/Lisbon, Portugal): Law 31/2024 holding-period CGT exclusion (19.6% effective
//     rate after 8+ years vs 28% base) + progressive IRS brackets on pension income; no wealth tax.
//   - "ES" (Valencia & Málaga/Costa del Sol, Spain): Progressive Renta del Ahorro CGT (19%–30%) + IRPF slabs
//     + Solidarity high-net-worth levy above €3.7M/person.
//   - "FR" (Nice & Aix-en-Provence, France): 30% PFU flat investment tax (22.5% optimized via Assurance-Vie/PEA
//     allowances) + 6.5% PUMA CSM health levy on high passive gains + Quotient Familial bareme.
func EuropeTaxAnnual(
	city data.CityProfile,
	ordinaryIncome, realizedLTCG, assessableNetWorthEUR float64,
	isCouple, enableTaxOpt, isIncomingForeignResident bool,
	yearsInRetirement, age int,
	cpiIndex float64,
) float64 {
	if cpiIndex <= 0 {
		cpiIndex = 1.0
	}
	if ordinaryIncome < 0 {
		ordinaryIncome = 0
	}
	if realizedLTCG < 0 {
		realizedLTCG = 0
	}
	if assessableNetWorthEUR < 0 {
		assessableNetWorthEUR = 0
	}

	adults := 1.0
	if isCouple {
		adults = 2.0
	}

	switch city.StateCode {
	case "CH-ZH", "CH-TI":
		// Switzerland: 0% Capital Gains Tax on private securities!
		// 1. Progressive Federal + Cantonal + Municipal income tax on ordinaryIncome (Pillar 2, state pension, bond interest)
		ded := 18_000.0 * cpiIndex
		if isCouple {
			ded = 32_000.0 * cpiIndex
		}
		taxableOrdPerAdult := math.Max(0, ordinaryIncome-ded) / adults
		ordTaxPerAdult := progressiveTax(taxableOrdPerAdult, []taxBand{
			{35_000 * cpiIndex, 0.07},
			{80_000 * cpiIndex, 0.12},
			{160_000 * cpiIndex, 0.17},
			{math.Inf(1), 0.22},
		})
		cantonIncomeMult := 1.00 // Zurich suburbs (Winterthur / Baden / Thalwil)
		cantonWealthMult := 1.00
		if city.StateCode == "CH-TI" {
			cantonIncomeMult = 1.09 // Ticino (Lugano) slightly higher cantonal multiplier
			cantonWealthMult = 1.18
		}
		ordTax := ordTaxPerAdult * adults * cantonIncomeMult

		// 2. Cantonal & Municipal Net Wealth Tax (Vermögenssteuer) on liquid/assessable net worth
		wealthExemption := 160_000.0 * cpiIndex * adults
		taxableWealth := math.Max(0, assessableNetWorthEUR-wealthExemption)
		wealthTax := progressiveTax(taxableWealth, []taxBand{
			{800_000 * cpiIndex, 0.0011},
			{2_000_000 * cpiIndex, 0.0021},
			{math.Inf(1), 0.0033},
		}) * cantonWealthMult

		return ordTax + wealthTax

	case "IT-SOUTH":
		// Southern Italy (Mezzogiorno <30k-pop municipalities):
		// Article 24-ter TUIR: 7% flat substitute tax on foreign pensions & investment gains for 10 tax years
		// (also exempts taxpayer from 0.20% IVAFE stamp duty on foreign financial assets).
		if isIncomingForeignResident && enableTaxOpt && yearsInRetirement >= 0 && yearsInRetirement < 10 {
			return (ordinaryIncome + realizedLTCG) * 0.07
		}
		// Standard Italian resident taxation (post-year-10 or domestic Italian resident):
		// 26% flat tax on capital gains + progressive IRPEF on ordinary/pension income + 0.20% IVAFE/bollo
		cgtTax := realizedLTCG * 0.26
		ordPerAdult := ordinaryIncome / adults
		irpef := progressiveTax(ordPerAdult, []taxBand{
			{8_500 * cpiIndex, 0.00}, // No-tax pension area
			{28_000 * cpiIndex, 0.23},
			{50_000 * cpiIndex, 0.35},
			{math.Inf(1), 0.43},
		}) * adults
		bollo := assessableNetWorthEUR * 0.0020
		return cgtTax + irpef + bollo

	case "GR":
		// Greece (Chania, Crete & Peloponnese):
		// Article 5B: 7% flat foreign retiree tax for 15 years on foreign-source income & gains
		if isIncomingForeignResident && enableTaxOpt && yearsInRetirement >= 0 && yearsInRetirement < 15 {
			return (ordinaryIncome + realizedLTCG) * 0.07
		}
		// Standard Greek regime: UCITS ETFs enjoy 0% CGT (blended ~5% dividend/rebalance drag if optimized, else 15%)
		cgtRate := 0.15
		if enableTaxOpt {
			cgtRate = 0.05
		}
		cgtTax := realizedLTCG * cgtRate
		ordPerAdult := ordinaryIncome / adults
		ordTax := progressiveTax(ordPerAdult, []taxBand{
			{10_000 * cpiIndex, 0.09},
			{20_000 * cpiIndex, 0.22},
			{30_000 * cpiIndex, 0.28},
			{40_000 * cpiIndex, 0.36},
			{math.Inf(1), 0.44},
		}) * adults
		return cgtTax + ordTax

	case "PT":
		// Portugal (Algarve & Cascais/Lisbon):
		// Law 31/2024 Capital Markets Reform: 30% CGT exclusion for securities held >= 8 years => 19.6% effective CGT (else 28%)
		cgtRate := 0.28
		if enableTaxOpt {
			cgtRate = 0.196
		}
		cgtTax := realizedLTCG * cgtRate
		// Portuguese IRS progressive brackets with Quociente Familiar (split by adults) after €4,350 specific deduction
		ordPerAdult := math.Max(0, ordinaryIncome/adults-4_350.0*cpiIndex)
		ordTax := progressiveTax(ordPerAdult, []taxBand{
			{7_703 * cpiIndex, 0.13},
			{11_623 * cpiIndex, 0.165},
			{16_472 * cpiIndex, 0.22},
			{21_321 * cpiIndex, 0.25},
			{27_146 * cpiIndex, 0.32},
			{39_791 * cpiIndex, 0.355},
			{80_000 * cpiIndex, 0.435},
			{math.Inf(1), 0.48},
		}) * adults
		return cgtTax + ordTax

	case "ES":
		// Spain (Valencia & Málaga / Costa del Sol):
		// Progressive Renta del Ahorro (Savings & Capital Gains base, split per partner on joint holdings)
		ltcgPerAdult := realizedLTCG / adults
		savingsTax := progressiveTax(ltcgPerAdult, []taxBand{
			{6_000 * cpiIndex, 0.19},
			{50_000 * cpiIndex, 0.21},
			{200_000 * cpiIndex, 0.23},
			{300_000 * cpiIndex, 0.27},
			{math.Inf(1), 0.30},
		}) * adults

		// General IRPF brackets after €5,550 personal allowance (higher €6,700 at 65+)
		allowance := 5_550.0 * cpiIndex
		if age >= 65 {
			allowance = 6_700.0 * cpiIndex
		}
		ordPerAdult := math.Max(0, ordinaryIncome/adults-allowance)
		ordTax := progressiveTax(ordPerAdult, []taxBand{
			{12_450 * cpiIndex, 0.19},
			{20_200 * cpiIndex, 0.24},
			{35_200 * cpiIndex, 0.30},
			{60_000 * cpiIndex, 0.37},
			{300_000 * cpiIndex, 0.45},
			{math.Inf(1), 0.47},
		}) * adults

		// Solidarity High-Net-Worth Tax (Impuesto de Solidaridad) only kicks in above €3.7M per person
		solidarity := 0.0
		wealthPerAdult := assessableNetWorthEUR / adults
		if wealthPerAdult > 3_700_000*cpiIndex {
			solidarity = (wealthPerAdult - 3_700_000*cpiIndex) * 0.0025 * adults
		}
		return savingsTax + ordTax + solidarity

	default:
		// France ("FR" — Nice & Aix-en-Provence) & general Western EU baseline:
		// 30% PFU flat tax (12.8% income + 17.2% social charges); Assurance-Vie / PEA wrapper reduces effective rate to ~22.5%
		pfuRate := 0.30
		if enableTaxOpt {
			pfuRate = 0.225
		}
		cgtTax := realizedLTCG * pfuRate

		// PUMA Cotisation Subsidiaire Maladie (6.5% healthcare levy on passive gains > €23,500 when pre-pension age < 65)
		pumaCSM := 0.0
		if age < 65 && ordinaryIncome < 9_200*cpiIndex*adults {
			pumaAbatement := 23_500.0 * cpiIndex * adults
			if realizedLTCG > pumaAbatement {
				pumaCSM = (realizedLTCG - pumaAbatement) * 0.065
			}
		}

		// Quotient Familial progressive Barème (after 10% pension abatement up to €4,321)
		abatedOrd := ordinaryIncome * 0.90
		partIncome := abatedOrd / adults
		ordTax := progressiveTax(partIncome, []taxBand{
			{11_497 * cpiIndex, 0.00},
			{29_315 * cpiIndex, 0.11},
			{83_823 * cpiIndex, 0.30},
			{180_294 * cpiIndex, 0.41},
			{math.Inf(1), 0.45},
		}) * adults

		return cgtTax + pumaCSM + ordTax
	}
}

type taxBand struct {
	upTo float64
	rate float64
}

func progressiveTax(income float64, bands []taxBand) float64 {
	if income <= 0 {
		return 0
	}
	tax := 0.0
	prev := 0.0
	for _, b := range bands {
		if income <= prev {
			break
		}
		chunk := math.Min(income, b.upTo) - prev
		if chunk > 0 {
			tax += chunk * b.rate
		}
		prev = b.upTo
	}
	return tax
}

// EuropeHealthcareAnnual models mandatory Swiss KVG/LAMal individual insurance + franchise/OOP
// for Switzerland ("CH-ZH", "CH-TI") and universal SNS/SSN/PUMA + private top-up hospital cover
// for Portugal, Italy, Spain, France, and Greece.
func EuropeHealthcareAnnual(age int, isCouple bool, city data.CityProfile, cpiIndex, medicalIndex float64) float64 {
	if cpiIndex <= 0 {
		cpiIndex = 1.0
	}
	if medicalIndex <= 0 {
		medicalIndex = 1.0
	}

	if city.StateCode == "CH-ZH" || city.StateCode == "CH-TI" {
		// Swiss LAMal / KVG is strictly per-capita (no family/couple discount => 2.0x for couples!)
		var perAdultEUR float64
		switch {
		case age < 45:
			perAdultEUR = 5_200
		case age < 60:
			perAdultEUR = 5_900
		case age < 75:
			perAdultEUR = 6_800
		default:
			perAdultEUR = 8_600
		}
		adults := 1.0
		if isCouple {
			adults = 2.0
		}
		return perAdultEUR * adults * city.HealthcareCostMult * medicalIndex
	}

	// Mediterranean / Western EU public + private supplementary cover (Médis, Sanitas, Mutuelle, private clinic cover)
	var perAdultEUR float64
	switch {
	case age < 45:
		perAdultEUR = 1_450
	case age < 55:
		perAdultEUR = 1_950
	case age < 65:
		perAdultEUR = 2_750
	case age < 75:
		perAdultEUR = 3_650
	default:
		perAdultEUR = 4_950
	}
	householdMult := 1.0
	if isCouple {
		householdMult = 1.88
	}

	voluntarySSNFloor := 0.0
	if city.StateCode == "IT-SOUTH" && age < 65 {
		// Minimum €2,000/yr Italian voluntary SSN enrollment contribution for non-working elective/early retirees
		voluntarySSNFloor = 1_200.0 * cpiIndex
	}

	return perAdultEUR*householdMult*city.HealthcareCostMult*medicalIndex + voluntarySSNFloor + city.TertiaryMedicalBuffer*cpiIndex
}
