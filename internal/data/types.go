package data

// CountryCode identifies a supported jurisdiction.
type CountryCode string

const (
	CountryIndia      CountryCode = "IN"
	CountryUSA        CountryCode = "US"
	CountryAustralia  CountryCode = "AU"
	CountryEurope     CountryCode = "EU"
	CountryNewZealand CountryCode = "NZ"
)

// IsValidCountry returns true if c is one of the 5 supported jurisdictions.
func IsValidCountry(c CountryCode) bool {
	switch c {
	case CountryIndia, CountryUSA, CountryAustralia, CountryEurope, CountryNewZealand:
		return true
	default:
		return false
	}
}

// MarketYear represents a single historical calendar year's joint nominal asset class
// total returns, inflation rates, and starting valuation multiple.
// All return and inflation values are expressed as decimals (e.g. 0.125 = 12.5%).
type MarketYear struct {
	Year        int     `json:"year"`
	Equity      float64 `json:"equity"`       // Broad equity total return (S&P 500 TR / Sensex-Nifty 50 TRI)
	Debt        float64 `json:"debt"`         // 10Y Sovereign bond total return (10Y US Treasury / 10Y India G-Sec)
	Cash        float64 `json:"cash"`         // 3M T-Bill / Liquid yield
	Gold        float64 `json:"gold"`         // Gold return in local currency (USD or INR)
	HeadlineCPI float64 `json:"headline_cpi"` // Headline consumer inflation (CPI-U / MoSPI CPI-Combined)
	MedicalCPI  float64 `json:"medical_cpi"`  // Healthcare inflation (US Medical Care CPI / India Medical CPI)
	ValuationPE float64 `json:"valuation_pe"` // Starting Shiller CAPE (US) or Nifty/Sensex Trailing P/E (India)
}

// LifestylePresets provides monthly spending benchmarks in today's purchasing power
// (excluding income taxes and standalone healthcare premiums, which are modeled explicitly).
type LifestylePresets struct {
	Lean     float64 `json:"lean"`
	Standard float64 `json:"standard"`
	Chubby   float64 `json:"chubby"`
	Fat      float64 `json:"fat"`
}

// CityProfile captures regional cost-of-living, housing, healthcare, and state-tax parameters
// for Tier-1 metros and Tier-2 retirement/lifestyle hubs in India and the USA.
type CityProfile struct {
	ID                     string           `json:"id"`
	Name                   string           `json:"name"`
	Region                 string           `json:"region"` // State or UT/Province
	Country                CountryCode      `json:"country"`
	Tier                   string           `json:"tier"` // "Tier 1" or "Tier 2"
	Tagline                string           `json:"tagline"`
	CostOfLivingIndex      float64          `json:"cost_of_living_index"`      // 1.00 = National/Tier-1 reference
	HousingCostIndex       float64          `json:"housing_cost_index"`       // Relative rent/housing cost
	HealthcareCostMult     float64          `json:"healthcare_cost_mult"`     // ACA Silver regional mult (US) or Zone mult (IN)
	TertiaryMedicalBuffer  float64          `json:"tertiary_medical_buffer"`  // Annual travel/metro-care buffer in local currency
	StateCode              string           `json:"state_code"`               // e.g. "CA", "NC", "TX", "KA", "GA"
	StateTopTaxRateDisplay float64          `json:"state_top_tax_rate_pct"`   // Display summary of top state marginal tax %
	PropertyTaxRate        float64          `json:"property_tax_rate"`        // Effective annual property tax decimal
	MonthlyPresets         LifestylePresets `json:"monthly_presets"`          // Calibrated monthly spend in local currency
	TypicalHomePrice       float64          `json:"typical_home_price"`       // Reference 3BHK / 3BR home price in today's money
}

// HistoricalCrisis defines a famous market/inflation stress regime for historical cohort replay.
type HistoricalCrisis struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	Country     CountryCode `json:"country"`
	StartYear   int         `json:"start_year"`
	Subtitle    string      `json:"subtitle"`
	Description string      `json:"description"`
}
