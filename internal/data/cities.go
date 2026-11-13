package data

// Cities returns all calibrated Tier-1 and Tier-2 city profiles for India and the USA.
func Cities() []CityProfile {
	return allCities
}

// CitiesByCountry filters city profiles by country code.
func CitiesByCountry(country CountryCode) []CityProfile {
	var out []CityProfile
	for _, c := range allCities {
		if c.Country == country {
			out = append(out, c)
		}
	}
	return out
}

// GetCity looks up a CityProfile by ID, falling back to the default Tier-1 city for the country.
func GetCity(id string, fallbackCountry CountryCode) CityProfile {
	for _, c := range allCities {
		if c.ID == id {
			return c
		}
	}
	switch fallbackCountry {
	case CountryUSA:
		return GetCity("us-san-jose", CountryIndia)
	case CountryAustralia:
		return GetCity("au-sydney", CountryIndia)
	case CountryEurope:
		return GetCity("eu-zurich-suburbs", CountryIndia)
	case CountryNewZealand:
		return GetCity("nz-auckland", CountryIndia)
	default:
		for _, c := range allCities {
			if c.ID == "in-bengaluru" {
				return c
			}
		}
		return allCities[0]
	}
}

// CrisesByCountry returns famous historical stress cohorts for a country.
func CrisesByCountry(country CountryCode) []HistoricalCrisis {
	switch country {
	case CountryUSA:
		return usCrises
	case CountryAustralia:
		return auCrises
	case CountryEurope:
		return euCrises
	case CountryNewZealand:
		return nzCrises
	default:
		return indiaCrises
	}
}

// DefaultUSDToINR is kept for backwards compatibility alongside FXRatesPerUSD.
const DefaultUSDToINR = 85.0

// FXRatesPerUSD returns reference nominal currency units per 1.00 USD across all 5 jurisdictions.
func FXRatesPerUSD() map[CountryCode]float64 {
	return map[CountryCode]float64{
		CountryUSA:        1.00,
		CountryIndia:      85.00,
		CountryAustralia:  1.46,
		CountryEurope:     0.94,
		CountryNewZealand: 1.65,
	}
}

// FXRate returns how many units of `to` currency equal 1.0 unit of `from` currency.
func FXRate(from, to CountryCode) float64 {
	if from == to {
		return 1.0
	}
	rates := FXRatesPerUSD()
	rFrom, ok1 := rates[from]
	rTo, ok2 := rates[to]
	if !ok1 || !ok2 || rFrom <= 0 {
		return 1.0
	}
	return rTo / rFrom
}

// ConvertCurrency converts a monetary amount between any two supported jurisdictions.
func ConvertCurrency(amount float64, from, to CountryCode) float64 {
	if from == to {
		return amount
	}
	return amount * FXRate(from, to)
}

var allCities = []CityProfile{
	// ============================================================================
	// INDIA (₹ INR) — Tier-1 Metros & Tier-2 Lifestyle / Retirement Hubs
	// ============================================================================
	{
		ID:                     "in-bengaluru",
		Name:                   "Bengaluru",
		Region:                 "Karnataka",
		Country:                CountryIndia,
		Tier:                   "Tier 1",
		Tagline:                "India's tech capital — top accumulation hub, elevated metro housing & school costs",
		CostOfLivingIndex:      1.00,
		HousingCostIndex:       1.00,
		HealthcareCostMult:     1.00,
		TertiaryMedicalBuffer:  0,
		StateCode:              "KA",
		StateTopTaxRateDisplay: 0.0,
		PropertyTaxRate:        0.0025,
		MonthlyPresets: LifestylePresets{
			Lean:     75_000,
			Standard: 140_000,
			Chubby:   260_000,
			Fat:      500_000,
		},
		TypicalHomePrice: 22_500_000, // ₹2.25 Cr
	},
	{
		ID:                     "in-mumbai",
		Name:                   "Mumbai",
		Region:                 "Maharashtra",
		Country:                CountryIndia,
		Tier:                   "Tier 1",
		Tagline:                "Financial capital — highest Indian housing & Zone-1 healthcare costs",
		CostOfLivingIndex:      1.24,
		HousingCostIndex:       1.60,
		HealthcareCostMult:     1.15,
		TertiaryMedicalBuffer:  0,
		StateCode:              "MH",
		StateTopTaxRateDisplay: 0.0,
		PropertyTaxRate:        0.0030,
		MonthlyPresets: LifestylePresets{
			Lean:     105_000,
			Standard: 185_000,
			Chubby:   340_000,
			Fat:      650_000,
		},
		TypicalHomePrice: 38_000_000, // ₹3.80 Cr
	},
	{
		ID:                     "in-delhi-ncr",
		Name:                   "Delhi-NCR (Gurugram)",
		Region:                 "NCR",
		Country:                CountryIndia,
		Tier:                   "Tier 1",
		Tagline:                "Corporate & startup corridor — high club/gated-community & healthcare Zone-1 pricing",
		CostOfLivingIndex:      1.06,
		HousingCostIndex:       1.15,
		HealthcareCostMult:     1.10,
		TertiaryMedicalBuffer:  0,
		StateCode:              "DL",
		StateTopTaxRateDisplay: 0.0,
		PropertyTaxRate:        0.0025,
		MonthlyPresets: LifestylePresets{
			Lean:     82_000,
			Standard: 150_000,
			Chubby:   280_000,
			Fat:      540_000,
		},
		TypicalHomePrice: 26_000_000, // ₹2.60 Cr
	},
	{
		ID:                     "in-hyderabad",
		Name:                   "Hyderabad",
		Region:                 "Telangana",
		Country:                CountryIndia,
		Tier:                   "Tier 1",
		Tagline:                "High-infrastructure Tier-1 metro with strong hospital network & balanced living costs",
		CostOfLivingIndex:      0.94,
		HousingCostIndex:       0.92,
		HealthcareCostMult:     0.96,
		TertiaryMedicalBuffer:  0,
		StateCode:              "TG",
		StateTopTaxRateDisplay: 0.0,
		PropertyTaxRate:        0.0022,
		MonthlyPresets: LifestylePresets{
			Lean:     70_000,
			Standard: 130_000,
			Chubby:   240_000,
			Fat:      460_000,
		},
		TypicalHomePrice: 19_500_000, // ₹1.95 Cr
	},
	{
		ID:                     "in-goa",
		Name:                   "Goa (Panaji / Porvorim / North)",
		Region:                 "Goa",
		Country:                CountryIndia,
		Tier:                   "Tier 2",
		Tagline:                "Flagship coastal FIRE haven — ~22% lower living spend & ~35% lower housing vs Bengaluru",
		CostOfLivingIndex:      0.78,
		HousingCostIndex:       0.65,
		HealthcareCostMult:     0.88,
		TertiaryMedicalBuffer:  60_000, // Annual metro tertiary medical travel buffer
		StateCode:              "GA",
		StateTopTaxRateDisplay: 0.0,
		PropertyTaxRate:        0.0020,
		MonthlyPresets: LifestylePresets{
			Lean:     58_000,
			Standard: 108_000,
			Chubby:   200_000,
			Fat:      390_000,
		},
		TypicalHomePrice: 14_500_000, // ₹1.45 Cr
	},
	{
		ID:                     "in-mysuru",
		Name:                   "Mysuru",
		Region:                 "Karnataka",
		Country:                CountryIndia,
		Tier:                   "Tier 2",
		Tagline:                "Clean, calm heritage city 2 hrs from Bengaluru — ~32% lower living cost",
		CostOfLivingIndex:      0.68,
		HousingCostIndex:       0.48,
		HealthcareCostMult:     0.82,
		TertiaryMedicalBuffer:  35_000,
		StateCode:              "KA",
		StateTopTaxRateDisplay: 0.0,
		PropertyTaxRate:        0.0018,
		MonthlyPresets: LifestylePresets{
			Lean:     48_000,
			Standard: 92_000,
			Chubby:   170_000,
			Fat:      320_000,
		},
		TypicalHomePrice: 10_500_000, // ₹1.05 Cr
	},
	{
		ID:                     "in-coimbatore",
		Name:                   "Coimbatore",
		Region:                 "Tamil Nadu",
		Country:                CountryIndia,
		Tier:                   "Tier 2",
		Tagline:                "Premier South Indian retirement & medical hub near Nilgiris hills",
		CostOfLivingIndex:      0.70,
		HousingCostIndex:       0.50,
		HealthcareCostMult:     0.84,
		TertiaryMedicalBuffer:  20_000,
		StateCode:              "TN",
		StateTopTaxRateDisplay: 0.0,
		PropertyTaxRate:        0.0018,
		MonthlyPresets: LifestylePresets{
			Lean:     50_000,
			Standard: 95_000,
			Chubby:   175_000,
			Fat:      330_000,
		},
		TypicalHomePrice: 11_000_000, // ₹1.10 Cr
	},
	{
		ID:                     "in-kochi",
		Name:                   "Kochi (Cochin)",
		Region:                 "Kerala",
		Country:                CountryIndia,
		Tier:                   "Tier 2",
		Tagline:                "Coastal metro-lite with top-tier multi-specialty hospitals (Aster/Amrita) & international airport",
		CostOfLivingIndex:      0.74,
		HousingCostIndex:       0.55,
		HealthcareCostMult:     0.86,
		TertiaryMedicalBuffer:  15_000,
		StateCode:              "KL",
		StateTopTaxRateDisplay: 0.0,
		PropertyTaxRate:        0.0020,
		MonthlyPresets: LifestylePresets{
			Lean:     54_000,
			Standard: 102_000,
			Chubby:   188_000,
			Fat:      355_000,
		},
		TypicalHomePrice: 12_000_000, // ₹1.20 Cr
	},
	{
		ID:                     "in-dehradun",
		Name:                   "Dehradun",
		Region:                 "Uttarakhand",
		Country:                CountryIndia,
		Tier:                   "Tier 2",
		Tagline:                "Himalayan foothill valley hub — Max Super Speciality & AIIMS Rishikesh access, pleasant climate",
		CostOfLivingIndex:      0.72,
		HousingCostIndex:       0.56,
		HealthcareCostMult:     0.86,
		TertiaryMedicalBuffer:  18_000,
		StateCode:              "UK",
		StateTopTaxRateDisplay: 0.0,
		PropertyTaxRate:        0.0018,
		MonthlyPresets: LifestylePresets{
			Lean:     52_000,
			Standard: 98_000,
			Chubby:   180_000,
			Fat:      340_000,
		},
		TypicalHomePrice: 14_500_000, // ₹1.45 Cr
	},
	{
		ID:                     "in-kodaikanal",
		Name:                   "Kodaikanal",
		Region:                 "Tamil Nadu",
		Country:                CountryIndia,
		Tier:                   "Tier 2",
		Tagline:                "2,100m Palani Hills misty retreat — cool year-round climate; tertiary hospital care via Madurai (~3 hrs)",
		CostOfLivingIndex:      0.71,
		HousingCostIndex:       0.58,
		HealthcareCostMult:     0.84,
		TertiaryMedicalBuffer:  55_000, // Hill-to-Madurai/Coimbatore specialist & winter logistics buffer
		StateCode:              "TN",
		StateTopTaxRateDisplay: 0.0,
		PropertyTaxRate:        0.0018,
		MonthlyPresets: LifestylePresets{
			Lean:     50_000,
			Standard: 96_000,
			Chubby:   176_000,
			Fat:      335_000,
		},
		TypicalHomePrice: 16_500_000, // ₹1.65 Cr
	},

	// ============================================================================
	// UNITED STATES ($ USD) — Tier-1 VHCOL Metros & Tier-2 MCOL Hubs
	// ============================================================================
	{
		ID:                     "us-san-jose",
		Name:                   "San Jose / Silicon Valley",
		Region:                 "California",
		Country:                CountryUSA,
		Tier:                   "Tier 1",
		Tagline:                "Peak tech accumulation hub — VHCOL housing + CA state tax up to 13.3% on LTCG & Roth conversions",
		CostOfLivingIndex:      1.58,
		HousingCostIndex:       2.15,
		HealthcareCostMult:     1.18,
		TertiaryMedicalBuffer:  0,
		StateCode:              "CA",
		StateTopTaxRateDisplay: 13.3,
		PropertyTaxRate:        0.0075,
		MonthlyPresets: LifestylePresets{
			Lean:     6_200,
			Standard: 10_500,
			Chubby:   17_500,
			Fat:      30_000,
		},
		TypicalHomePrice: 1_750_000,
	},
	{
		ID:                     "us-nyc",
		Name:                   "New York City",
		Region:                 "New York",
		Country:                CountryUSA,
		Tier:                   "Tier 1",
		Tagline:                "Global finance & media hub — VHCOL living + combined NYS & NYC income/cap-gains taxes",
		CostOfLivingIndex:      1.62,
		HousingCostIndex:       2.25,
		HealthcareCostMult:     1.22,
		TertiaryMedicalBuffer:  0,
		StateCode:              "NY",
		StateTopTaxRateDisplay: 14.8,
		PropertyTaxRate:        0.0095,
		MonthlyPresets: LifestylePresets{
			Lean:     6_500,
			Standard: 11_000,
			Chubby:   18_500,
			Fat:      32_000,
		},
		TypicalHomePrice: 1_650_000,
	},
	{
		ID:                     "us-seattle",
		Name:                   "Seattle / Eastside",
		Region:                 "Washington",
		Country:                CountryUSA,
		Tier:                   "Tier 1",
		Tagline:                "HCOL Pacific NW hub — 0% state tax on Roth conversions/401k (7% WA excise only on high LTCG)",
		CostOfLivingIndex:      1.32,
		HousingCostIndex:       1.60,
		HealthcareCostMult:     1.08,
		TertiaryMedicalBuffer:  0,
		StateCode:              "WA",
		StateTopTaxRateDisplay: 0.0,
		PropertyTaxRate:        0.0092,
		MonthlyPresets: LifestylePresets{
			Lean:     5_200,
			Standard: 8_800,
			Chubby:   14_800,
			Fat:      25_500,
		},
		TypicalHomePrice: 1_150_000,
	},
	{
		ID:                     "us-raleigh",
		Name:                   "Raleigh / Research Triangle",
		Region:                 "North Carolina",
		Country:                CountryUSA,
		Tier:                   "Tier 2",
		Tagline:                "Premier MCOL hub — ~39% lower living cost vs San Jose + low ~4.25% flat NC state tax",
		CostOfLivingIndex:      0.96,
		HousingCostIndex:       0.88,
		HealthcareCostMult:     0.98,
		TertiaryMedicalBuffer:  0,
		StateCode:              "NC",
		StateTopTaxRateDisplay: 4.25,
		PropertyTaxRate:        0.0082,
		MonthlyPresets: LifestylePresets{
			Lean:     3_900,
			Standard: 6_500,
			Chubby:   11_000,
			Fat:      19_000,
		},
		TypicalHomePrice: 540_000,
	},
	{
		ID:                     "us-austin",
		Name:                   "Austin",
		Region:                 "Texas",
		Country:                CountryUSA,
		Tier:                   "Tier 2",
		Tagline:                "0% Texas state income & LTCG tax — MCOL lifestyle with higher ~1.65% property tax",
		CostOfLivingIndex:      1.02,
		HousingCostIndex:       0.95,
		HealthcareCostMult:     0.96,
		TertiaryMedicalBuffer:  0,
		StateCode:              "TX",
		StateTopTaxRateDisplay: 0.0,
		PropertyTaxRate:        0.0165,
		MonthlyPresets: LifestylePresets{
			Lean:     4_100,
			Standard: 6_900,
			Chubby:   11_600,
			Fat:      20_000,
		},
		TypicalHomePrice: 580_000,
	},
	{
		ID:                     "us-tampa",
		Name:                   "Tampa / Sarasota",
		Region:                 "Florida",
		Country:                CountryUSA,
		Tier:                   "Tier 2",
		Tagline:                "Classic coastal FIRE magnet — 0% Florida state income/LTCG tax & moderate living basket",
		CostOfLivingIndex:      0.98,
		HousingCostIndex:       0.92,
		HealthcareCostMult:     1.02,
		TertiaryMedicalBuffer:  0,
		StateCode:              "FL",
		StateTopTaxRateDisplay: 0.0,
		PropertyTaxRate:        0.0090,
		MonthlyPresets: LifestylePresets{
			Lean:     4_000,
			Standard: 6_600,
			Chubby:   11_200,
			Fat:      19_500,
		},
		TypicalHomePrice: 510_000,
	},
	{
		ID:                     "us-denver",
		Name:                   "Denver / Front Range",
		Region:                 "Colorado",
		Country:                CountryUSA,
		Tier:                   "Tier 2",
		Tagline:                "Active mountain lifestyle — flat 4.40% Colorado state tax & low 0.55% property tax",
		CostOfLivingIndex:      1.10,
		HousingCostIndex:       1.15,
		HealthcareCostMult:     1.00,
		TertiaryMedicalBuffer:  0,
		StateCode:              "CO",
		StateTopTaxRateDisplay: 4.40,
		PropertyTaxRate:        0.0055,
		MonthlyPresets: LifestylePresets{
			Lean:     4_400,
			Standard: 7_400,
			Chubby:   12_500,
			Fat:      21_500,
		},
		TypicalHomePrice: 660_000,
	},

	// ============================================================================
	// AUSTRALIA (A$ AUD) — Major Metros & Coastal Retirement Hubs
	// ============================================================================
	{
		ID:                     "au-sydney",
		Name:                   "Sydney",
		Region:                 "New South Wales",
		Country:                CountryAustralia,
		Tier:                   "Tier 1",
		Tagline:                "Australia's harbour & financial capital — high housing costs, Medicare + top private hospital network",
		CostOfLivingIndex:      1.08,
		HousingCostIndex:       1.28,
		HealthcareCostMult:     1.06,
		TertiaryMedicalBuffer:  0,
		StateCode:              "NSW",
		StateTopTaxRateDisplay: 47.0,
		PropertyTaxRate:        0.0035,
		MonthlyPresets: LifestylePresets{
			Lean:     4_600,
			Standard: 6_800,
			Chubby:   11_200,
			Fat:      19_000,
		},
		TypicalHomePrice: 1_650_000,
	},
	{
		ID:                     "au-melbourne",
		Name:                   "Melbourne",
		Region:                 "Victoria",
		Country:                CountryAustralia,
		Tier:                   "Tier 1",
		Tagline:                "Cultural & biomedical hub — ~12% lower housing than Sydney with world-class hospitals & trams",
		CostOfLivingIndex:      1.00,
		HousingCostIndex:       1.00,
		HealthcareCostMult:     1.00,
		TertiaryMedicalBuffer:  0,
		StateCode:              "VIC",
		StateTopTaxRateDisplay: 47.0,
		PropertyTaxRate:        0.0040,
		MonthlyPresets: LifestylePresets{
			Lean:     4_100,
			Standard: 6_100,
			Chubby:   10_200,
			Fat:      17_500,
		},
		TypicalHomePrice: 1_120_000,
	},
	{
		ID:                     "au-gold-coast",
		Name:                   "Gold Coast / Sunshine Coast",
		Region:                 "Queensland",
		Country:                CountryAustralia,
		Tier:                   "Tier 2",
		Tagline:                "Flagship subtropical beach FIRE belt — ~18% lower living costs vs Sydney + tax-free Super after 60",
		CostOfLivingIndex:      0.90,
		HousingCostIndex:       0.88,
		HealthcareCostMult:     0.97,
		TertiaryMedicalBuffer:  600,
		StateCode:              "QLD",
		StateTopTaxRateDisplay: 47.0,
		PropertyTaxRate:        0.0036,
		MonthlyPresets: LifestylePresets{
			Lean:     3_700,
			Standard: 5_500,
			Chubby:   9_200,
			Fat:      15_800,
		},
		TypicalHomePrice: 1_050_000,
	},
	{
		ID:                     "au-adelaide",
		Name:                   "Adelaide",
		Region:                 "South Australia",
		Country:                CountryAustralia,
		Tier:                   "Tier 2",
		Tagline:                "Mediterranean-climate wine capital — Australia's strongest metro affordability-to-healthcare ratio",
		CostOfLivingIndex:      0.85,
		HousingCostIndex:       0.78,
		HealthcareCostMult:     0.95,
		TertiaryMedicalBuffer:  0,
		StateCode:              "SA",
		StateTopTaxRateDisplay: 47.0,
		PropertyTaxRate:        0.0038,
		MonthlyPresets: LifestylePresets{
			Lean:     3_500,
			Standard: 5_200,
			Chubby:   8_600,
			Fat:      14_800,
		},
		TypicalHomePrice: 880_000,
	},

	// ============================================================================
	// EUROPE & SWITZERLAND (€ EUR) — Swiss Hubs & Mediterranean Retirement Havens
	// ============================================================================
	{
		ID:                     "eu-zurich-suburbs",
		Name:                   "Zurich Suburbs (Winterthur / Baden / Thalwil)",
		Region:                 "Zurich Region, Switzerland",
		Country:                CountryEurope,
		Tier:                   "Tier 1",
		Tagline:                "Peak European accumulation hub — 0% private equity CGT, low Swiss tax + wealth levy, mandatory LAMal insurance",
		CostOfLivingIndex:      1.55,
		HousingCostIndex:       1.70,
		HealthcareCostMult:     1.00,
		TertiaryMedicalBuffer:  0,
		StateCode:              "CH-ZH",
		StateTopTaxRateDisplay: 22.5,
		PropertyTaxRate:        0.0020,
		MonthlyPresets: LifestylePresets{
			Lean:     3_900,
			Standard: 5_600,
			Chubby:   9_200,
			Fat:      16_000,
		},
		TypicalHomePrice: 1_350_000,
	},
	{
		ID:                     "eu-lugano",
		Name:                   "Lugano & Lake Maggiore Suburbs",
		Region:                 "Ticino, Switzerland",
		Country:                CountryEurope,
		Tier:                   "Tier 1",
		Tagline:                "Sunniest Italian-speaking Swiss lake hub — 0% private CGT, Alpine safety, higher Ticino LAMal Zone-1 premiums",
		CostOfLivingIndex:      1.42,
		HousingCostIndex:       1.45,
		HealthcareCostMult:     1.28,
		TertiaryMedicalBuffer:  0,
		StateCode:              "CH-TI",
		StateTopTaxRateDisplay: 24.0,
		PropertyTaxRate:        0.0025,
		MonthlyPresets: LifestylePresets{
			Lean:     3_500,
			Standard: 5_100,
			Chubby:   8_500,
			Fat:      14_800,
		},
		TypicalHomePrice: 1_150_000,
	},
	{
		ID:                     "eu-algarve",
		Name:                   "Algarve (Lagos / Tavira / Faro)",
		Region:                 "Portugal",
		Country:                CountryEurope,
		Tier:                   "Tier 2",
		Tagline:                "300 days of Atlantic sunshine — 19.6% long-term CGT (8yr+ holding exclusion), no wealth tax, low private healthcare",
		CostOfLivingIndex:      0.78,
		HousingCostIndex:       0.72,
		HealthcareCostMult:     0.88,
		TertiaryMedicalBuffer:  500,
		StateCode:              "PT",
		StateTopTaxRateDisplay: 28.0,
		PropertyTaxRate:        0.0035,
		MonthlyPresets: LifestylePresets{
			Lean:     1_950,
			Standard: 2_850,
			Chubby:   4_800,
			Fat:      8_500,
		},
		TypicalHomePrice: 460_000,
	},
	{
		ID:                     "eu-cascais-lisbon",
		Name:                   "Cascais & Lisbon Coast",
		Region:                 "Portugal",
		Country:                CountryEurope,
		Tier:                   "Tier 1",
		Tagline:                "Cosmopolitan Portuguese riviera — international hub, top private hospitals (CUF/Luz), 19.6%–28% CGT",
		CostOfLivingIndex:      0.92,
		HousingCostIndex:       0.96,
		HealthcareCostMult:     0.95,
		TertiaryMedicalBuffer:  0,
		StateCode:              "PT",
		StateTopTaxRateDisplay: 28.0,
		PropertyTaxRate:        0.0038,
		MonthlyPresets: LifestylePresets{
			Lean:     2_350,
			Standard: 3_400,
			Chubby:   5_800,
			Fat:      10_200,
		},
		TypicalHomePrice: 680_000,
	},
	{
		ID:                     "eu-south-italy",
		Name:                   "Southern Italy (Puglia / Lecce & Sicily)",
		Region:                 "Mezzogiorno, Italy",
		Country:                CountryEurope,
		Tier:                   "Tier 2",
		Tagline:                "Eligible <30k-pop Mezzogiorno towns — 7% Flat Foreign Retiree Tax (10 yrs) or 26% CGT, ultra-low Mediterranean COL",
		CostOfLivingIndex:      0.62,
		HousingCostIndex:       0.48,
		HealthcareCostMult:     0.85,
		TertiaryMedicalBuffer:  750,
		StateCode:              "IT-SOUTH",
		StateTopTaxRateDisplay: 7.0,
		PropertyTaxRate:        0.0040,
		MonthlyPresets: LifestylePresets{
			Lean:     1_550,
			Standard: 2_250,
			Chubby:   3_800,
			Fat:      6_800,
		},
		TypicalHomePrice: 260_000,
	},
	{
		ID:                     "eu-valencia",
		Name:                   "Valencia & Costa del Sol (Málaga)",
		Region:                 "Spain",
		Country:                CountryEurope,
		Tier:                   "Tier 2",
		Tagline:                "Europe's #1 walkable Mediterranean beach cities — top Spanish healthcare, 19%–28% progressive savings tax",
		CostOfLivingIndex:      0.80,
		HousingCostIndex:       0.74,
		HealthcareCostMult:     0.90,
		TertiaryMedicalBuffer:  0,
		StateCode:              "ES",
		StateTopTaxRateDisplay: 28.0,
		PropertyTaxRate:        0.0045,
		MonthlyPresets: LifestylePresets{
			Lean:     2_000,
			Standard: 2_950,
			Chubby:   4_900,
			Fat:      8_800,
		},
		TypicalHomePrice: 410_000,
	},
	{
		ID:                     "eu-nice-provence",
		Name:                   "Nice & Aix-en-Provence (Côte d'Azur)",
		Region:                 "France",
		Country:                CountryEurope,
		Tier:                   "Tier 2",
		Tagline:                "French Riviera & Provence — gold-standard PUMA universal healthcare, 30% PFU flat investment tax",
		CostOfLivingIndex:      1.00,
		HousingCostIndex:       1.02,
		HealthcareCostMult:     1.00,
		TertiaryMedicalBuffer:  0,
		StateCode:              "FR",
		StateTopTaxRateDisplay: 30.0,
		PropertyTaxRate:        0.0055,
		MonthlyPresets: LifestylePresets{
			Lean:     2_500,
			Standard: 3_650,
			Chubby:   6_200,
			Fat:      11_000,
		},
		TypicalHomePrice: 590_000,
	},
	{
		ID:                     "eu-crete-greece",
		Name:                   "Chania (Crete) & Peloponnese",
		Region:                 "Greece",
		Country:                CountryEurope,
		Tier:                   "Tier 2",
		Tagline:                "Aegean coastal haven — 7% Flat Foreign Retiree Tax (15 yrs), low housing & longevity diet",
		CostOfLivingIndex:      0.65,
		HousingCostIndex:       0.52,
		HealthcareCostMult:     0.84,
		TertiaryMedicalBuffer:  700,
		StateCode:              "GR",
		StateTopTaxRateDisplay: 7.0,
		PropertyTaxRate:        0.0035,
		MonthlyPresets: LifestylePresets{
			Lean:     1_600,
			Standard: 2_350,
			Chubby:   3_950,
			Fat:      7_100,
		},
		TypicalHomePrice: 310_000,
	},

	// ============================================================================
	// NEW ZEALAND (NZ$ NZD) — North & South Island Lifestyle Hubs
	// ============================================================================
	{
		ID:                     "nz-auckland",
		Name:                   "Auckland",
		Region:                 "North Island",
		Country:                CountryNewZealand,
		Tier:                   "Tier 1",
		Tagline:                "NZ's harbour metropolis — KiwiSaver + universal NZ Super at 65, 0% domestic CGT (FIF 5% FDR on global stocks)",
		CostOfLivingIndex:      1.00,
		HousingCostIndex:       1.05,
		HealthcareCostMult:     1.02,
		TertiaryMedicalBuffer:  0,
		StateCode:              "NZ-AKL",
		StateTopTaxRateDisplay: 39.0,
		PropertyTaxRate:        0.0032,
		MonthlyPresets: LifestylePresets{
			Lean:     4_200,
			Standard: 6_000,
			Chubby:   9_800,
			Fat:      16_800,
		},
		TypicalHomePrice: 1_080_000,
	},
	{
		ID:                     "nz-wellington",
		Name:                   "Wellington",
		Region:                 "North Island",
		Country:                CountryNewZealand,
		Tier:                   "Tier 1",
		Tagline:                "Compact walkable harbour capital — vibrant cultural scene & tertiary public hospital hub",
		CostOfLivingIndex:      0.95,
		HousingCostIndex:       0.92,
		HealthcareCostMult:     1.00,
		TertiaryMedicalBuffer:  0,
		StateCode:              "NZ-WLG",
		StateTopTaxRateDisplay: 39.0,
		PropertyTaxRate:        0.0038,
		MonthlyPresets: LifestylePresets{
			Lean:     3_950,
			Standard: 5_700,
			Chubby:   9_300,
			Fat:      15_900,
		},
		TypicalHomePrice: 890_000,
	},
	{
		ID:                     "nz-tauranga-nelson",
		Name:                   "Tauranga / Nelson (Bay of Plenty & Sunny South)",
		Region:                 "Coastal New Zealand",
		Country:                CountryNewZealand,
		Tier:                   "Tier 2",
		Tagline:                "New Zealand's #1 sunny coastal retirement belt — mild winters, beaches & relaxed outdoor living",
		CostOfLivingIndex:      0.90,
		HousingCostIndex:       0.88,
		HealthcareCostMult:     0.98,
		TertiaryMedicalBuffer:  650,
		StateCode:              "NZ-BOP",
		StateTopTaxRateDisplay: 39.0,
		PropertyTaxRate:        0.0034,
		MonthlyPresets: LifestylePresets{
			Lean:     3_750,
			Standard: 5_400,
			Chubby:   8_800,
			Fat:      15_000,
		},
		TypicalHomePrice: 920_000,
	},
	{
		ID:                     "nz-queenstown",
		Name:                   "Queenstown / Wanaka",
		Region:                 "Otago Southern Alps",
		Country:                CountryNewZealand,
		Tier:                   "Tier 2",
		Tagline:                "Iconic alpine lake sanctuary — premium housing & winter heating; tertiary hospital care in Dunedin/Christchurch",
		CostOfLivingIndex:      1.12,
		HousingCostIndex:       1.25,
		HealthcareCostMult:     1.05,
		TertiaryMedicalBuffer:  1_400,
		StateCode:              "NZ-OTA",
		StateTopTaxRateDisplay: 39.0,
		PropertyTaxRate:        0.0030,
		MonthlyPresets: LifestylePresets{
			Lean:     4_800,
			Standard: 6_800,
			Chubby:   11_400,
			Fat:      19_500,
		},
		TypicalHomePrice: 1_550_000,
	},
}

var usCrises = []HistoricalCrisis{
	{
		ID:          "us-1929",
		Name:        "1929 Great Depression",
		Country:     CountryUSA,
		StartYear:   1929,
		Subtitle:    "CAPE 32.6 • -64% Cumulative S&P Collapse (1929–1932)",
		Description: "Retiring at the September 1929 market peak right before four consecutive brutal equity drawdown years.",
	},
	{
		ID:          "us-1966",
		Name:        "1966 Stagflation Era (Bengen's Worst Cohort)",
		Country:     CountryUSA,
		StartYear:   1966,
		Subtitle:    "16 Years of Flat Real Stocks + 1970s Double-Digit Inflation",
		Description: "Historically deadlier than 1929 for 4% retirees because soaring 1973–1981 CPI compounded withdrawals while both stocks and bonds lost real value.",
	},
	{
		ID:          "us-1973",
		Name:        "1973–74 Oil Embargo & Stagflation",
		Country:     CountryUSA,
		StartYear:   1973,
		Subtitle:    "-37% Nominal Equity Drop + 11%–13% Runaway CPI",
		Description: "Immediate back-to-back bear market paired with persistent stagflation through 1981.",
	},
	{
		ID:          "us-2000",
		Name:        "2000 Dot-Com Peak + 2008 GFC ('Lost Decade')",
		Country:     CountryUSA,
		StartYear:   2000,
		Subtitle:    "CAPE 44.2 • Two -45%+ Crashes in First 9 Years",
		Description: "Retiring at all-time valuation highs into 2000–2002 (-43%) followed shortly by the 2008 financial crisis (-37%).",
	},
	{
		ID:          "us-2008",
		Name:        "2008 Global Financial Crisis",
		Country:     CountryUSA,
		StartYear:   2008,
		Subtitle:    "-36.6% Year-1 Equity Shock + Zero-Yield ZIRP Era",
		Description: "Immediate Year-1 banking panic drawdown followed by a decade of near-zero T-Bill yields.",
	},
}

var indiaCrises = []HistoricalCrisis{
	{
		ID:          "in-1992",
		Name:        "1992 Harshad Mehta Crash & High Inflation",
		Country:     CountryIndia,
		StartYear:   1992,
		Subtitle:    "Post-Boom Valuation Unwind + 11.8% CPI Inflation",
		Description: "Retiring at peak post-liberalization euphoria right before the 1992–93 securities market slump and double-digit inflation.",
	},
	{
		ID:          "in-2000",
		Name:        "2000 Ketan Parekh / Dot-Com Meltdown",
		Country:     CountryIndia,
		StartYear:   2000,
		Subtitle:    "3-Year Equity Bear Market (-20.6%, -16.2%) Through 2002",
		Description: "Multi-year early-retirement sequence shock across Indian equities before the 2003 recovery.",
	},
	{
		ID:          "in-2008",
		Name:        "2008 Global Financial Crisis (Nifty -51.8%)",
		Country:     CountryIndia,
		StartYear:   2008,
		Subtitle:    "Sharpest Single-Year Indian Equity Drawdown + Rising CPI",
		Description: "Year-1 Nifty 50 TRI crash of -51.8% followed quickly by the 2010–2013 sticky inflation wave.",
	},
	{
		ID:          "in-2010",
		Name:        "2010–2013 Sticky Double-Digit CPI Stagflation",
		Country:     CountryIndia,
		StartYear:   2010,
		Subtitle:    "4 Years of 9.5%–12% CPI + -24% 2011 Drawdown",
		Description: "Classic Indian sequence risk: living & medical expenses surged ~55% in 4 years while nominal Nifty returns stagnated.",
	},
}

var auCrises = []HistoricalCrisis{
	{
		ID:          "au-1982",
		Name:        "1981–1982 Resource Slump & Double-Digit Inflation",
		Country:     CountryAustralia,
		StartYear:   1981,
		Subtitle:    "Back-to-Back ASX Drops (-12.9%, -13.8%) + 11.1% CPI",
		Description: "Severe Australian stagflation shock combining back-to-back equity losses with double-digit consumer and medical inflation.",
	},
	{
		ID:          "au-1987",
		Name:        "1987 Black Monday & 1990 Recession",
		Country:     CountryAustralia,
		StartYear:   1987,
		Subtitle:    "Post-Boom Unwind + -17.5% 1990 Recession Slump",
		Description: "Retiring into the October 1987 crash followed by high late-80s CPI and Australia's 1990–91 recession.",
	},
	{
		ID:          "au-2008",
		Name:        "2008 Global Financial Crisis (ASX -38.4%)",
		Country:     CountryAustralia,
		StartYear:   2008,
		Subtitle:    "Sharpest Modern ASX Accumulation Drawdown + 2011 Dip",
		Description: "Immediate -38.4% Year-1 Australian equity crash followed by a sluggish 2010–2011 recovery.",
	},
	{
		ID:          "au-2022",
		Name:        "2022 Joint Stock/Bond Drop & 6.6% CPI Spike",
		Country:     CountryAustralia,
		StartYear:   2022,
		Subtitle:    "Commonwealth Bonds -10.2% + Post-COVID Inflation Surge",
		Description: "Simultaneous bond and equity drawdown paired with the sharpest Australian inflation wave in three decades.",
	},
}

var euCrises = []HistoricalCrisis{
	{
		ID:          "eu-1980",
		Name:        "1980–1982 Second Oil Shock Stagflation",
		Country:     CountryEurope,
		StartYear:   1980,
		Subtitle:    "3 Years of 8.8%–10.8% European Inflation",
		Description: "Persistent high European inflation eroding bond and cash purchasing power in early retirement.",
	},
	{
		ID:          "eu-2000",
		Name:        "2000–2002 European Telecom & Dot-Com Crash",
		Country:     CountryEurope,
		StartYear:   2000,
		Subtitle:    "3-Year Bear Market (-2.2%, -15.6%, -30.7%) + 2008 GFC",
		Description: "Brutal 'Lost Decade' sequence: three straight negative equity years followed six years later by the -43.6% GFC.",
	},
	{
		ID:          "eu-2008",
		Name:        "2008 GFC (-43.6%) & 2011 Euro Sovereign Debt Crisis",
		Country:     CountryEurope,
		StartYear:   2008,
		Subtitle:    "Deep Equity Shock + Negative ECB Deposit Yield Era",
		Description: "Double-dip European banking and sovereign crisis followed by a decade of zero-to-negative safe yields.",
	},
	{
		ID:          "eu-2022",
		Name:        "2022 European Energy Shock & Bund Rout",
		Country:     CountryEurope,
		StartYear:   2022,
		Subtitle:    "STOXX -10.6% • 10Y Bund -16.8% • 8.4% HICP",
		Description: "Classic 60/40 stress test where European bonds suffered historic losses alongside energy-driven inflation.",
	},
}

var nzCrises = []HistoricalCrisis{
	{
		ID:          "nz-1980",
		Name:        "1980–1986 Pre-RBNZ Double-Digit Inflation Era",
		Country:     CountryNewZealand,
		StartYear:   1980,
		Subtitle:    "15%–17% Annual NZ CPI Before Inflation Targeting",
		Description: "Extreme purchasing-power erosion testing whether equity and property-linked growth can outpace runaway inflation.",
	},
	{
		ID:          "nz-1987",
		Name:        "1987–1990 NZ Sharemarket Meltdown (-65%+ Real)",
		Country:     CountryNewZealand,
		StartYear:   1987,
		Subtitle:    "-48.2% (1987), -12.4% (1988), -27.6% (1990) + 15.7% CPI",
		Description: "One of the harshest multi-year equity crashes in OECD history — New Zealand's ultimate sequence-of-returns stress test.",
	},
	{
		ID:          "nz-2008",
		Name:        "2008 Global Financial Crisis & Finance Co. Failures",
		Country:     CountryNewZealand,
		StartYear:   2008,
		Subtitle:    "-32.8% NZX Drop + Domestic Credit Crunch",
		Description: "Sharp Year-1 NZX drawdown paired with New Zealand finance-company freezes and 4% CPI.",
	},
	{
		ID:          "nz-2021",
		Name:        "2021–2023 High-OCR NZX Drawdown & 7.2% CPI",
		Country:     CountryNewZealand,
		StartYear:   2021,
		Subtitle:    "Flat/Negative NZX + Bond Losses + Steep Medical Inflation",
		Description: "Multi-year real portfolio contraction as RBNZ hiked rates aggressively into 7.2% headline and 8.6% medical inflation.",
	},
}
