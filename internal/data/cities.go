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
	if fallbackCountry == CountryUSA {
		return GetCity("us-san-jose", CountryUSA)
	}
	return GetCity("in-bengaluru", CountryIndia)
}

// CrisesByCountry returns famous historical stress cohorts for a country.
func CrisesByCountry(country CountryCode) []HistoricalCrisis {
	if country == CountryUSA {
		return usCrises
	}
	return indiaCrises
}

// DefaultUSDToINR is the baseline nominal FX parity used when modeling cross-border
// accumulation and retirement moves between India (INR) and the USA (USD).
const DefaultUSDToINR = 85.0

// ConvertCurrency converts a monetary amount between India (INR) and USA (USD)
// using DefaultUSDToINR.
func ConvertCurrency(amount float64, from, to CountryCode) float64 {
	if from == to {
		return amount
	}
	if from == CountryUSA && to == CountryIndia {
		return amount * DefaultUSDToINR
	}
	if from == CountryIndia && to == CountryUSA {
		return amount / DefaultUSDToINR
	}
	return amount
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
		Name:                   "Kochi",
		Region:                 "Kerala",
		Country:                CountryIndia,
		Tier:                   "Tier 2",
		Tagline:                "Coastal metro-lite with top-tier multi-specialty hospitals & international airport",
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
