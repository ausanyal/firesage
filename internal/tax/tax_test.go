package tax

import (
	"math"
	"testing"

	"github.com/ausanyal/firesage/internal/data"
)

func TestIndiaTaxRules(t *testing.T) {
	// Section 87A Rebate: Up to ₹12,00,000 slab income pays zero tax under New Regime
	if got := IndiaNewRegimeSlabTax(1_200_000, 1.0); got != 0 {
		t.Fatalf("expected ₹0 slab tax at ₹12L under 87A rebate, got %.2f", got)
	}

	// Above ₹12L: ₹20L slab income should pay progressive slab tax + 4% cess
	tax20L := IndiaNewRegimeSlabTax(2_000_000, 1.0)
	// 0-4L: 0, 4-8L: 20k, 8-12L: 40k, 12-16L: 60k, 16-20L: 80k => 200k * 1.04 = 208,000
	if math.Abs(tax20L-208_000) > 1.0 {
		t.Fatalf("expected ₹2,08,000 tax at ₹20L slab income, got %.2f", tax20L)
	}

	// Section 112A Equity LTCG: ₹1.25L tax-free, 12.5% + 4% cess (13.0%) above
	if got := IndiaEquityLTCGTax(125_000, 1.0); got != 0 {
		t.Fatalf("expected ₹0 LTCG tax at ₹1.25L gain, got %.2f", got)
	}
	// ₹11.25L gain -> ₹10L taxable * 13% = ₹1,30,000
	if got := IndiaEquityLTCGTax(1_125_000, 1.0); math.Abs(got-130_000) > 1.0 {
		t.Fatalf("expected ₹1,30,000 LTCG tax on ₹11.25L gain, got %.2f", got)
	}
}

func TestUSStateTaxAnd2026ACACliff(t *testing.T) {
	// California (San Jose) vs North Carolina (Raleigh) vs Texas (Austin) on $180k joint income ($80k ord + $100k LTCG)
	caTax := USStateTax("CA", 80_000, 100_000, true, 1.0)
	ncTax := USStateTax("NC", 80_000, 100_000, true, 1.0)
	txTax := USStateTax("TX", 80_000, 100_000, true, 1.0)

	if txTax != 0 {
		t.Fatalf("expected $0 TX state tax, got %.2f", txTax)
	}
	if !(caTax > ncTax && ncTax > txTax) {
		t.Fatalf("expected CA tax (%.2f) > NC tax (%.2f) > TX tax (%.2f)", caTax, ncTax, txTax)
	}

	// Verify 2026 Hard 400% FPL ACA Subsidy Cliff at age 58 for a couple ($21,150 * 4.0 = $84,600 cliff)
	sanJose := data.GetCity("us-san-jose", data.CountryUSA)
	justBelowCliff := USHealthcareAnnual(58, true, 82_000, sanJose, 1.0, 1.0)
	justAboveCliff := USHealthcareAnnual(58, true, 86_000, sanJose, 1.0, 1.0)

	// Crossing 400% FPL at age 58 in San Jose causes an immediate steep jump in annual healthcare cost!
	if justAboveCliff-justBelowCliff < 12_000 {
		t.Fatalf("expected steep 2026 ACA 400%% FPL subsidy cliff jump (>$12k), got below=%.2f above=%.2f", justBelowCliff, justAboveCliff)
	}

	// Verify SECURE 2.0 RMD start ages: born 1965 (age 73 -> 0 RMD; age 75 -> >0 RMD)
	if div := USRMDDivisor(73, 1965); div != 0 {
		t.Fatalf("expected 0 RMD divisor at age 73 for 1965 birth year under SECURE 2.0, got %.2f", div)
	}
	if div := USRMDDivisor(75, 1965); div <= 0 {
		t.Fatalf("expected positive RMD divisor at age 75 for 1965 birth year, got %.2f", div)
	}
}

func TestAustraliaTaxAndAgePension(t *testing.T) {
	// Couple realizing A$80,000 LTCG (50% discount => A$40k => A$20k/partner):
	// Under Stage 3 + LITO, A$20k/partner should pay essentially ~A$0 tax!
	lowTax := AustraliaIncomeTax(0, 80_000, true, true, 1.0)
	if lowTax > 100 {
		t.Fatalf("expected near-zero Australian tax for couple realizing A$80k LTCG, got %.2f", lowTax)
	}

	// Single realizing A$200k ordinary income pays substantial Stage 3 + 2% Medicare Levy
	highTax := AustraliaIncomeTax(200_000, 0, false, false, 1.0)
	if highTax < 55_000 || highTax > 66_000 {
		t.Fatalf("expected ~A$60k tax on A$200k single ordinary income, got %.2f", highTax)
	}

	// Age Pension: 0 at age 65, full A$48,516 at age 67 with A$400k assets, tapered at A$700k, 0 at A$1.2M
	if got := AustraliaAgePensionAnnual(65, true, 300_000, 1.0); got != 0 {
		t.Fatalf("expected 0 AU Age Pension at age 65, got %.2f", got)
	}
	if got := AustraliaAgePensionAnnual(67, true, 400_000, 1.0); math.Abs(got-48_516) > 1 {
		t.Fatalf("expected full A$48,516 couple Age Pension at A$400k assets, got %.2f", got)
	}
	tapered := AustraliaAgePensionAnnual(67, true, 700_000, 1.0)
	if !(tapered > 20_000 && tapered < 35_000) {
		t.Fatalf("expected tapered Age Pension around A$30.5k at A$700k assets, got %.2f", tapered)
	}
	if got := AustraliaAgePensionAnnual(67, true, 1_200_000, 1.0); got != 0 {
		t.Fatalf("expected 0 Age Pension above A$1.1M homeowner couple cutoff, got %.2f", got)
	}
}

func TestEuropeSubJurisdictions(t *testing.T) {
	zurich := data.GetCity("eu-zurich-suburbs", data.CountryEurope)
	lugano := data.GetCity("eu-lugano", data.CountryEurope)
	puglia := data.GetCity("eu-south-italy", data.CountryEurope)
	algarve := data.GetCity("eu-algarve", data.CountryEurope)

	// Zurich has 0% CGT on €100k LTCG, paying only modest cantonal wealth tax on €1.5M net worth
	chTax := EuropeTaxAnnual(zurich, 0, 100_000, 1_500_000, true, true, false, 2, 52, 1.0)
	if chTax <= 0 || chTax > 3_500 {
		t.Fatalf("expected modest Swiss wealth-only tax (<€3.5k) on €100k LTCG + €1.5M wealth, got %.2f", chTax)
	}

	// Lugano LAMal healthcare should exceed Zurich suburbs due to Ticino Zone-1 multiplier (1.28x)
	zhHealth := EuropeHealthcareAnnual(55, true, zurich, 1.0, 1.0)
	tiHealth := EuropeHealthcareAnnual(55, true, lugano, 1.0, 1.0)
	if tiHealth <= zhHealth*1.20 {
		t.Fatalf("expected Lugano LAMal (%.2f) > 1.2x Zurich suburbs (%.2f)", tiHealth, zhHealth)
	}

	// Southern Italy Article 24-ter: 7% flat tax in year 3 for incoming retiree vs 26% CGT in year 12!
	itYr3 := EuropeTaxAnnual(puglia, 20_000, 60_000, 1_000_000, true, true, true, 3, 55, 1.0)
	itYr12 := EuropeTaxAnnual(puglia, 20_000, 60_000, 1_000_000, true, true, true, 12, 64, 1.0)
	if math.Abs(itYr3-5_600) > 1.0 {
		t.Fatalf("expected €5,600 (7%% of €80k) under Italy Art. 24-ter in yr 3, got %.2f", itYr3)
	}
	if itYr12 <= itYr3*2.5 {
		t.Fatalf("expected post-10yr Italian tax (%.2f) to jump well above 7%% flat regime (%.2f)", itYr12, itYr3)
	}

	// Portugal Law 31/2024: 19.6% long-term CGT when optimized vs 28% unoptimized
	ptOpt := EuropeTaxAnnual(algarve, 0, 100_000, 1_000_000, true, true, true, 5, 55, 1.0)
	ptUnopt := EuropeTaxAnnual(algarve, 0, 100_000, 1_000_000, true, false, true, 5, 55, 1.0)
	if math.Abs(ptOpt-19_600) > 1 || math.Abs(ptUnopt-28_000) > 1 {
		t.Fatalf("expected PT CGT €19,600 (opt) vs €28,000 (unopt), got %.2f vs %.2f", ptOpt, ptUnopt)
	}
}

func TestNewZealandFIFAndHealthcare(t *testing.T) {
	auckland := data.GetCity("nz-auckland", data.CountryNewZealand)

	// NZ$1.2M equity portfolio (72% = NZ$864k offshore > NZ$100k couple de minimis => NZ$43.2k FDR deemed income)
	// With PIE 28% cap vs unoptimized on top of high ordinary income
	pieTax := NZTaxAnnual(180_000, 1_200_000, false, true, 1.0)
	directTax := NZTaxAnnual(180_000, 1_200_000, false, false, 1.0)
	if !(pieTax > 0 && directTax > pieTax) {
		t.Fatalf("expected NZ PIE 28%% cap (%.2f) < direct 39%% marginal FIF tax (%.2f)", pieTax, directTax)
	}

	// Verify steep Southern Cross age-rated healthcare jump from age 50 to age 70
	h50 := NZHealthcareAnnual(50, true, auckland, 1.0, 1.0)
	h70 := NZHealthcareAnnual(70, true, auckland, 1.0, 1.0)
	if h70 < h50*2.0 {
		t.Fatalf("expected NZ age-70 healthcare (%.2f) >= 2x age-50 healthcare (%.2f)", h70, h50)
	}
}
