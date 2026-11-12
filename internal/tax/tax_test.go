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
