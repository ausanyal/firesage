package copilot

import (
	"fmt"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/ausanyal/firesage/internal/data"
	"github.com/ausanyal/firesage/internal/engine"
)

// FormatCompact formats currency amounts identically to web/src/utils/format.ts across all 5 jurisdictions.
func FormatCompact(amount float64, country data.CountryCode) string {
	sign := ""
	if amount < 0 {
		sign = "-"
	}
	abs := math.Abs(amount)

	if country == data.CountryIndia {
		switch {
		case abs >= 1e7:
			cr := abs / 1e7
			if cr >= 100 {
				return fmt.Sprintf("%s₹%.0f Cr", sign, cr)
			}
			if cr >= 10 {
				return fmt.Sprintf("%s₹%.1f Cr", sign, cr)
			}
			return fmt.Sprintf("%s₹%.2f Cr", sign, cr)
		case abs >= 1e5:
			l := abs / 1e5
			if l >= 10 {
				return fmt.Sprintf("%s₹%.1f L", sign, l)
			}
			return fmt.Sprintf("%s₹%.2f L", sign, l)
		case abs >= 1e3:
			return fmt.Sprintf("%s₹%.0fK", sign, abs/1e3)
		default:
			return fmt.Sprintf("%s₹%.0f", sign, math.Round(abs))
		}
	}

	sym := "$"
	switch country {
	case data.CountryAustralia:
		sym = "A$"
	case data.CountryEurope:
		sym = "€"
	case data.CountryNewZealand:
		sym = "NZ$"
	}

	switch {
	case abs >= 1e9:
		return fmt.Sprintf("%s%s%.2fB", sign, sym, abs/1e9)
	case abs >= 1e6:
		m := abs / 1e6
		if m >= 100 {
			return fmt.Sprintf("%s%s%.0fM", sign, sym, m)
		}
		if m >= 10 {
			return fmt.Sprintf("%s%s%.1fM", sign, sym, m)
		}
		return fmt.Sprintf("%s%s%.2fM", sign, sym, m)
	case abs >= 1e3:
		k := abs / 1e3
		if k >= 100 {
			return fmt.Sprintf("%s%s%.0fK", sign, sym, k)
		}
		return fmt.Sprintf("%s%s%.1fK", sign, sym, k)
	default:
		return fmt.Sprintf("%s%s%.0f", sign, sym, math.Round(abs))
	}
}

// CountryUnlockAge returns the statutory penalty-free pension unlock age for a jurisdiction.
func CountryUnlockAge(c data.CountryCode) int {
	if c == data.CountryNewZealand {
		return 65
	}
	return 60
}

// CountryDisplayName returns flag + label for a jurisdiction.
func CountryDisplayName(c data.CountryCode) string {
	switch c {
	case data.CountryIndia:
		return "🇮🇳 India (INR ₹)"
	case data.CountryUSA:
		return "🇺🇸 USA (USD $)"
	case data.CountryAustralia:
		return "🇦🇺 Australia (AUD A$)"
	case data.CountryEurope:
		return "🇪🇺 Europe & CH (EUR €)"
	case data.CountryNewZealand:
		return "🇳🇿 New Zealand (NZD NZ$)"
	default:
		return string(c)
	}
}

type cityAliasEntry struct {
	alias  string
	cityID string
}

var cityAliases = []cityAliasEntry{
	// India (11 cities)
	{"bengaluru", "in-bengaluru"},
	{"bangalore", "in-bengaluru"},
	{"mumbai", "in-mumbai"},
	{"bombay", "in-mumbai"},
	{"delhi-ncr", "in-delhi-ncr"},
	{"delhi", "in-delhi-ncr"},
	{"gurugram", "in-delhi-ncr"},
	{"gurgaon", "in-delhi-ncr"},
	{"noida", "in-delhi-ncr"},
	{"hyderabad", "in-hyderabad"},
	{"kolkata", "in-kolkata"},
	{"calcutta", "in-kolkata"},
	{"goa", "in-goa"},
	{"panaji", "in-goa"},
	{"porvorim", "in-goa"},
	{"mysuru", "in-mysuru"},
	{"mysore", "in-mysuru"},
	{"coimbatore", "in-coimbatore"},
	{"kochi", "in-kochi"},
	{"cochin", "in-kochi"},
	{"dehradun", "in-dehradun"},
	{"kodaikanal", "in-kodaikanal"},
	{"kodai", "in-kodaikanal"},

	// USA (7 cities)
	{"san jose", "us-san-jose"},
	{"silicon valley", "us-san-jose"},
	{"bay area", "us-san-jose"},
	{"san francisco", "us-san-jose"},
	{"new york city", "us-nyc"},
	{"new york", "us-nyc"},
	{"nyc", "us-nyc"},
	{"manhattan", "us-nyc"},
	{"seattle", "us-seattle"},
	{"bellevue", "us-seattle"},
	{"raleigh", "us-raleigh"},
	{"research triangle", "us-raleigh"},
	{"cary", "us-raleigh"},
	{"austin", "us-austin"},
	{"tampa", "us-tampa"},
	{"sarasota", "us-tampa"},
	{"florida", "us-tampa"},
	{"denver", "us-denver"},
	{"boulder", "us-denver"},

	// Australia (4 cities)
	{"sydney", "au-sydney"},
	{"melbourne", "au-melbourne"},
	{"gold coast", "au-gold-coast"},
	{"sunshine coast", "au-gold-coast"},
	{"brisbane", "au-gold-coast"},
	{"adelaide", "au-adelaide"},

	// Europe & Switzerland (8 cities)
	{"zurich", "eu-zurich-suburbs"},
	{"zürich", "eu-zurich-suburbs"},
	{"winterthur", "eu-zurich-suburbs"},
	{"baden", "eu-zurich-suburbs"},
	{"thalwil", "eu-zurich-suburbs"},
	{"lugano", "eu-lugano"},
	{"ticino", "eu-lugano"},
	{"locarno", "eu-lugano"},
	{"algarve", "eu-algarve"},
	{"faro", "eu-algarve"},
	{"lagos", "eu-algarve"},
	{"tavira", "eu-algarve"},
	{"cascais", "eu-cascais-lisbon"},
	{"lisbon", "eu-cascais-lisbon"},
	{"portugal", "eu-algarve"},
	{"southern italy", "eu-south-italy"},
	{"south of italy", "eu-south-italy"},
	{"south italy", "eu-south-italy"},
	{"puglia", "eu-south-italy"},
	{"lecce", "eu-south-italy"},
	{"sicily", "eu-south-italy"},
	{"italy", "eu-south-italy"},
	{"valencia", "eu-valencia"},
	{"malaga", "eu-valencia"},
	{"málaga", "eu-valencia"},
	{"costa del sol", "eu-valencia"},
	{"spain", "eu-valencia"},
	{"nice", "eu-nice-provence"},
	{"provence", "eu-nice-provence"},
	{"aix-en-provence", "eu-nice-provence"},
	{"cote d'azur", "eu-nice-provence"},
	{"france", "eu-nice-provence"},
	{"chania", "eu-crete-greece"},
	{"crete", "eu-crete-greece"},
	{"peloponnese", "eu-crete-greece"},
	{"greece", "eu-crete-greece"},

	// New Zealand (4 cities)
	{"auckland", "nz-auckland"},
	{"wellington", "nz-wellington"},
	{"tauranga", "nz-tauranga-nelson"},
	{"nelson", "nz-tauranga-nelson"},
	{"bay of plenty", "nz-tauranga-nelson"},
	{"queenstown", "nz-queenstown"},
	{"wanaka", "nz-queenstown"},
}

// ExtractMentionedCities finds all global cities mentioned in question in exact appearance order.
func ExtractMentionedCities(question string) []data.CityProfile {
	qLow := strings.ToLower(question)
	type matchPos struct {
		pos    int
		length int
		cityID string
	}
	var hits []matchPos
	for _, entry := range cityAliases {
		pat := regexp.MustCompile(`\b` + regexp.QuoteMeta(entry.alias) + `\b`)
		loc := pat.FindStringIndex(qLow)
		if loc != nil {
			hits = append(hits, matchPos{pos: loc[0], length: loc[1] - loc[0], cityID: entry.cityID})
		}
	}
	sort.Slice(hits, func(i, j int) bool {
		if hits[i].pos != hits[j].pos {
			return hits[i].pos < hits[j].pos
		}
		return hits[i].length > hits[j].length
	})

	cityByID := make(map[string]data.CityProfile, 40)
	for _, c := range data.Cities() {
		cityByID[c.ID] = c
	}

	seen := make(map[string]bool)
	var out []data.CityProfile
	for _, h := range hits {
		if !seen[h.cityID] {
			if c, ok := cityByID[h.cityID]; ok {
				seen[h.cityID] = true
				out = append(out, c)
			}
		}
	}
	return out
}

// ParseMoneyToken converts a numeric string and optional unit ("cr", "l", "lakh", "m", "k") into raw units.
func ParseMoneyToken(numStr, unitStr string) (float64, bool) {
	clean := strings.ReplaceAll(strings.TrimSpace(numStr), ",", "")
	val, err := strconv.ParseFloat(clean, 64)
	if err != nil || val <= 0 || math.IsNaN(val) || math.IsInf(val, 0) {
		return 0, false
	}
	u := strings.ToLower(strings.TrimSpace(unitStr))
	switch u {
	case "cr", "crore", "crores":
		val *= 1e7
	case "l", "lakh", "lakhs", "lac", "lacs":
		val *= 1e5
	case "b", "bn", "billion":
		val *= 1e9
	case "m", "mn", "mil", "million":
		val *= 1e6
	case "k", "thousand":
		val *= 1e3
	}
	return val, true
}

// ParsedIntent captures structured user intents and parameter overrides extracted by Stage 1.
type ParsedIntent struct {
	IsExplicitUIMutation bool
	IsComparison         bool
	IsGoalSeek           bool
	IsStressOrCrisis     bool
	IsTaxOrLiquidity     bool
	IsLifeEvent          bool
	IsMethodologyOr35x   bool

	SwitchCountry *data.CountryCode
	MentionedCities []data.CityProfile

	NewCurrentAge     *int
	NewRetirementAge  *int
	NewHorizonAge     *int
	NewMonthlySpend   *float64
	NewCurrentCorpus  *float64
	NewMonthlySavings *float64
	NewEquityPct      *float64
	NewStrategy       *engine.WithdrawalStrategy
	EnableBondTent    *bool
	EnableSmile       *bool
	EnableTaxOpt      *bool

	TargetConfidence float64
	CrashPct         *float64
	ExtraInfPct      *float64
	MedicalShock     *float64
	MedicalShockAge  int

	ProposedEvent *engine.LifeEvent

	SwitchBasis   string // "real" | "nominal" | ""
	OpenAutopsy   string // "Worst Path" | "P10 Stress" | "Median Path" | "P90 Bull" | ""
	OpenLabTab    string // "levers" | "crises" | "guardrails" | "srr" | ""
	ScrollSection string // "wealth-cone" | "frontier" | "cashflow-bridge" | "battle-lab" | "kpi-strip" | ""
	OpenSources   bool
}

var (
	reMutationVerb = regexp.MustCompile(`(?i)\b(set|change|switch|update|apply|move\s+to|relocate\s+to|retire\s+in|enable|turn\s+on|disable|turn\s+off|add\s+a|add\s+an|load|open|show\s+me)\b`)
	reWhatIf       = regexp.MustCompile(`(?i)\b(what\s+if|suppose|assuming|compare|vs\.?|versus|how\s+much|can\s+i|could\s+i|would\s+i)\b`)
	reCompare      = regexp.MustCompile(`(?i)\b(compare|vs\.?|versus|head-to-head|better\s+city|which\s+city|geo-?arbitrage|where\s+should\s+i\s+retire|cheaper\s+city|top\s+\d*\s*cit)\b`)
	reGoalSeek     = regexp.MustCompile(`(?i)\b(how\s+much|goal[\s-]*seek|extra\s+sip|extra\s+savings|monthly\s+sip|need\s+to\s+save|need\s+to\s+retire|required\s+corpus|when\s+can\s+i\s+retire|earliest\s+age|can\s+i\s+retire|safe\s+spend)\b`)
	reStress       = regexp.MustCompile(`(?i)\b(stress[\s-]*test|crash|bear\s+market|stagflation|recession|crisis|crises|1929|1966|1973|1987|1992|2000|2008|2022|medical\s+emergency|health\s+shock|drop|drops|plunge)\b`)
	reTaxLiq       = regexp.MustCompile(`(?i)\b(tax|taxes|ltcg|112a|87a|epf|ppf|nps|401k|401\(k\)|ira|roth|aca|medicare|irmaa|superannuation|centrelink|franking|kiwisaver|fif|pie|lamal|pillar|locked|liquid|liquidity|bridge|unlock|age\s*60|age\s*65)\b`)
	reLifeEvent    = regexp.MustCompile(`(?i)\b(sabbatical|career\s+break|college|university|education|wedding|marriage|barista|consulting\s+bridge|part[\s-]*time|downsize|sell\s+home|inheritance|windfall|life\s+event)\b`)
	reMethod35x    = regexp.MustCompile(`(?i)\b(35x|25x|30x|4%\s*rule|swr|multiplier|how\s+do\s+you\s+calculate|methodology|block\s+bootstrap|monte\s+carlo\s+work)\b`)

	reRetireAge  = regexp.MustCompile(`(?i)\b(?:retire\s+(?:at|by|age)|retirement\s+age\s*(?:to|of|=|is)?|fire\s+(?:at|age))\s*(\d{2})\b`)
	reCurrentAge = regexp.MustCompile(`(?i)\b(?:current\s+age\s*(?:to|of|=|is)?|i\s+am|i'm)\s*(\d{2})\b`)
	reHorizonAge = regexp.MustCompile(`(?i)\b(?:horizon\s*(?:age)?\s*(?:to|of|=)?|live\s+(?:until|to|till)\s*(?:age)?)\s*(\d{2,3})\b`)

	reMoneySpend  = regexp.MustCompile(`(?i)\b(?:spend|spending|budget|expenses?|lifestyle)\s*(?:of|to|at|=|is)?\s*(?:rs\.?|inr|₹|\$|a\$|au\$|nz\$|€|eur|usd|aud|nzd)?\s*(\d[\d,]*(?:\.\d+)?)\s*(cr|crore|crores|l|lakh|lakhs|lac|lacs|m|mn|million|k|thousand)?\b`)
	reMoneyCorpus = regexp.MustCompile(`(?i)\b(?:corpus|portfolio|nest\s+egg|net\s+worth|saved|have)\s*(?:of|to|at|=|is)?\s*(?:rs\.?|inr|₹|\$|a\$|au\$|nz\$|€|eur|usd|aud|nzd)?\s*(\d[\d,]*(?:\.\d+)?)\s*(cr|crore|crores|l|lakh|lakhs|lac|lacs|m|mn|million|k|thousand)?\b`)
	reMoneySave   = regexp.MustCompile(`(?i)\b(?:sip|monthly\s+savings?|save|saving|invest|contribute)\s*(?:of|to|at|=|is)?\s*(?:rs\.?|inr|₹|\$|a\$|au\$|nz\$|€|eur|usd|aud|nzd)?\s*(\d[\d,]*(?:\.\d+)?)\s*(cr|crore|crores|l|lakh|lakhs|lac|lacs|m|mn|million|k|thousand)?\b`)
	reGenericAmt  = regexp.MustCompile(`(?i)(?:rs\.?|inr|₹|\$|a\$|au\$|nz\$|€)\s*(\d[\d,]*(?:\.\d+)?)\s*(cr|crore|crores|l|lakh|lakhs|lac|lacs|m|mn|million|k|thousand)?`)

	reEquityAlloc = regexp.MustCompile(`(?i)\b(\d{2,3})\s*%\s*(?:in\s+)?(?:equity|equities|stocks)\b`)
	reConfidence  = regexp.MustCompile(`(?i)\b(80|85|90|92|95|97|98|99)\s*%\s*(?:confidence|survival|safe|success|odds|probability)\b`)
	reCrashPct    = regexp.MustCompile(`(?i)(-?\d{1,2}(?:\.\d+)?)\s*%\s*(?:market\s+|equity\s+|year[\s-]*1\s+)?(?:crash|drop|plunge|selloff|drawdown|shock)`)
	reCrashAlt    = regexp.MustCompile(`(?i)(?:crash|drop|drops|plunge|falls)\s*(?:by|of)?\s*(-?\d{1,2}(?:\.\d+)?)\s*%`)
	reExtraInf    = regexp.MustCompile(`(?i)\+?(\d{1,2}(?:\.\d+)?)\s*%\s*(?:extra\s+|higher\s+|spike\s+in\s+)?inflation`)
	reAtAge       = regexp.MustCompile(`(?i)\bat\s+age\s+(\d{2})\b`)
)

// ParseUserQuestion extracts all structured intents, city references, parameter tweaks, and UI commands.
func ParseUserQuestion(question string, base engine.PlanInput) ParsedIntent {
	qLow := strings.ToLower(question)
	intent := ParsedIntent{
		TargetConfidence: 90.0,
		MedicalShockAge:  72,
	}

	hasMut := reMutationVerb.MatchString(question)
	hasWhatIf := reWhatIf.MatchString(question)
	intent.IsExplicitUIMutation = hasMut && !strings.HasPrefix(strings.TrimSpace(qLow), "what if")
	intent.IsComparison = reCompare.MatchString(question)
	intent.IsGoalSeek = reGoalSeek.MatchString(question)
	intent.IsStressOrCrisis = reStress.MatchString(question)
	intent.IsTaxOrLiquidity = reTaxLiq.MatchString(question)
	intent.IsLifeEvent = reLifeEvent.MatchString(question)
	intent.IsMethodologyOr35x = reMethod35x.MatchString(question)

	intent.MentionedCities = ExtractMentionedCities(question)
	if len(intent.MentionedCities) >= 2 && (strings.Contains(qLow, " or ") || strings.Contains(qLow, " vs") || hasWhatIf) {
		intent.IsComparison = true
	}

	// Explicit country switch detection (e.g., "switch to Australia", "change country to New Zealand")
	if strings.Contains(qLow, "switch to ") || strings.Contains(qLow, "change country to ") || strings.Contains(qLow, "load default ") {
		switch {
		case strings.Contains(qLow, "australia"):
			c := data.CountryAustralia
			intent.SwitchCountry = &c
		case strings.Contains(qLow, "new zealand"):
			c := data.CountryNewZealand
			intent.SwitchCountry = &c
		case strings.Contains(qLow, "europe") || strings.Contains(qLow, "switzerland"):
			c := data.CountryEurope
			intent.SwitchCountry = &c
		case strings.Contains(qLow, "usa") || strings.Contains(qLow, "united states") || strings.Contains(qLow, "us plan"):
			c := data.CountryUSA
			intent.SwitchCountry = &c
		case strings.Contains(qLow, "india"):
			c := data.CountryIndia
			intent.SwitchCountry = &c
		}
	}

	if m := reConfidence.FindStringSubmatch(question); len(m) >= 2 {
		if v, err := strconv.ParseFloat(m[1], 64); err == nil {
			intent.TargetConfidence = v
		}
	}

	if m := reCurrentAge.FindStringSubmatch(question); len(m) >= 2 {
		if age, err := strconv.Atoi(m[1]); err == nil && age >= 18 && age <= 80 {
			intent.NewCurrentAge = &age
		}
	}
	if m := reRetireAge.FindStringSubmatch(question); len(m) >= 2 {
		if age, err := strconv.Atoi(m[1]); err == nil && age >= 22 && age <= 85 {
			intent.NewRetirementAge = &age
		}
	}
	if m := reHorizonAge.FindStringSubmatch(question); len(m) >= 2 {
		if age, err := strconv.Atoi(m[1]); err == nil && age >= 65 && age <= 105 {
			intent.NewHorizonAge = &age
		}
	}

	if m := reMoneySpend.FindStringSubmatch(question); len(m) >= 2 {
		if val, ok := ParseMoneyToken(m[1], m[2]); ok && val >= 400 {
			intent.NewMonthlySpend = &val
		}
	}
	if m := reMoneyCorpus.FindStringSubmatch(question); len(m) >= 2 {
		if val, ok := ParseMoneyToken(m[1], m[2]); ok && val >= 5_000 {
			intent.NewCurrentCorpus = &val
		}
	}
	if m := reMoneySave.FindStringSubmatch(question); len(m) >= 2 && !intent.IsLifeEvent {
		if val, ok := ParseMoneyToken(m[1], m[2]); ok && val >= 100 {
			intent.NewMonthlySavings = &val
		}
	}

	if m := reEquityAlloc.FindStringSubmatch(question); len(m) >= 2 {
		if pct, err := strconv.ParseFloat(m[1], 64); err == nil && pct >= 10 && pct <= 95 {
			frac := pct / 100.0
			intent.NewEquityPct = &frac
		}
	}

	// Withdrawal strategy & toggles
	if strings.Contains(qLow, "guyton") || strings.Contains(qLow, "klinger") || strings.Contains(qLow, "dynamic guardrail") {
		s := engine.StrategyGuytonKlinger
		intent.NewStrategy = &s
	} else if strings.Contains(qLow, "cape dynamic") || strings.Contains(qLow, "valuation dynamic") {
		s := engine.StrategyCAPEDynamic
		intent.NewStrategy = &s
	} else if strings.Contains(qLow, "constant real") || strings.Contains(qLow, "fixed 4%") || strings.Contains(qLow, "rigid spend") {
		s := engine.StrategyConstantReal
		intent.NewStrategy = &s
	}

	if strings.Contains(qLow, "bond tent") || strings.Contains(qLow, "glidepath") {
		v := !strings.Contains(qLow, "disable") && !strings.Contains(qLow, "without")
		intent.EnableBondTent = &v
	}
	if strings.Contains(qLow, "spending smile") || strings.Contains(qLow, "blanchett") {
		v := !strings.Contains(qLow, "disable") && !strings.Contains(qLow, "without")
		intent.EnableSmile = &v
	}
	if strings.Contains(qLow, "tax optimization") || strings.Contains(qLow, "tax harvesting") || strings.Contains(qLow, "roth ladder") {
		v := !strings.Contains(qLow, "disable") && !strings.Contains(qLow, "without")
		intent.EnableTaxOpt = &v
	}

	// Stress shock parameters
	if m := reCrashPct.FindStringSubmatch(question); len(m) >= 2 {
		if v, err := strconv.ParseFloat(m[1], 64); err == nil {
			frac := -math.Abs(v) / 100.0
			intent.CrashPct = &frac
			intent.IsStressOrCrisis = true
		}
	} else if m := reCrashAlt.FindStringSubmatch(question); len(m) >= 2 {
		if v, err := strconv.ParseFloat(m[1], 64); err == nil {
			frac := -math.Abs(v) / 100.0
			intent.CrashPct = &frac
			intent.IsStressOrCrisis = true
		}
	}
	if m := reExtraInf.FindStringSubmatch(question); len(m) >= 2 {
		if v, err := strconv.ParseFloat(m[1], 64); err == nil && v > 0 && v <= 20 {
			frac := v / 100.0
			intent.ExtraInfPct = &frac
			intent.IsStressOrCrisis = true
		}
	}
	if m := reAtAge.FindStringSubmatch(question); len(m) >= 2 {
		if age, err := strconv.Atoi(m[1]); err == nil && age >= base.CurrentAge && age <= base.HorizonAge {
			intent.MedicalShockAge = age
		}
	}
	if strings.Contains(qLow, "medical emergency") || strings.Contains(qLow, "health shock") || strings.Contains(qLow, "hospital shock") {
		intent.IsStressOrCrisis = true
		if m := reGenericAmt.FindStringSubmatch(question); len(m) >= 2 {
			if val, ok := ParseMoneyToken(m[1], m[2]); ok {
				intent.MedicalShock = &val
			}
		}
		if intent.MedicalShock == nil {
			defShock := 85_000.0
			if base.Country == data.CountryIndia {
				defShock = 2_500_000.0
			}
			intent.MedicalShock = &defShock
		}
	}

	// Life Event extraction
	if intent.IsLifeEvent {
		evAge := base.RetirementAge + 2
		if intent.MedicalShockAge != 72 {
			evAge = intent.MedicalShockAge
		}
		var parsedAmt float64
		if m := reGenericAmt.FindStringSubmatch(question); len(m) >= 2 {
			if val, ok := ParseMoneyToken(m[1], m[2]); ok {
				parsedAmt = val
			}
		}

		switch {
		case strings.Contains(qLow, "barista") || strings.Contains(qLow, "consulting") || strings.Contains(qLow, "part-time") || strings.Contains(qLow, "part time"):
			ann := parsedAmt
			if ann <= 0 {
				ann = math.Round(base.DesiredMonthlySpend * 12 * 0.35)
			}
			intent.ProposedEvent = &engine.LifeEvent{
				ID:          "copilot-barista-bridge",
				Name:        "5-Yr Consulting / Coast Bridge",
				StartAge:    base.RetirementAge,
				EndAge:      minInt(base.HorizonAge-2, base.RetirementAge+4),
				AnnualReal:  math.Abs(ann),
				IsTaxableIn: true,
			}
		case strings.Contains(qLow, "downsize") || strings.Contains(qLow, "sell home") || strings.Contains(qLow, "inheritance") || strings.Contains(qLow, "windfall"):
			amt := parsedAmt
			if amt <= 0 {
				amt = base.DesiredMonthlySpend * 12 * 3.5
			}
			intent.ProposedEvent = &engine.LifeEvent{
				ID:          "copilot-inflow-event",
				Name:        "Home Downsize / Capital Inflow",
				StartAge:    maxInt(base.RetirementAge, evAge),
				EndAge:      maxInt(base.RetirementAge, evAge),
				AnnualReal:  math.Abs(amt),
				IsTaxableIn: false,
			}
		case strings.Contains(qLow, "sabbatical") || strings.Contains(qLow, "career break"):
			start := maxInt(base.CurrentAge+1, minInt(base.RetirementAge-2, evAge))
			amt := parsedAmt
			if amt <= 0 {
				amt = base.DesiredMonthlySpend * 12
			}
			intent.ProposedEvent = &engine.LifeEvent{
				ID:          "copilot-sabbatical",
				Name:        "2-Yr Pre-FIRE Sabbatical",
				StartAge:    start,
				EndAge:      start + 1,
				AnnualReal:  -math.Abs(amt),
				IsTaxableIn: false,
			}
		default:
			// Kids' college / wedding / milestone outflow
			label := "Kids' Higher Education / Milestone"
			if strings.Contains(qLow, "wedding") || strings.Contains(qLow, "marriage") {
				label = "Family Wedding Milestone"
			}
			amt := parsedAmt
			if amt <= 0 {
				if base.Country == data.CountryIndia {
					amt = 4_000_000 // ₹40 L
				} else {
					amt = 100_000
				}
			}
			intent.ProposedEvent = &engine.LifeEvent{
				ID:          "copilot-milestone-outflow",
				Name:        label,
				StartAge:    evAge,
				EndAge:      evAge,
				AnnualReal:  -math.Abs(amt),
				IsTaxableIn: false,
			}
		}
	}

	// View / Navigation Page-Driver detection
	if strings.Contains(qLow, "nominal") || strings.Contains(qLow, "future value") || strings.Contains(qLow, "future currency") {
		intent.SwitchBasis = "nominal"
	} else if strings.Contains(qLow, "today's purchasing") || strings.Contains(qLow, "real purchasing") || strings.Contains(qLow, "switch to real") {
		intent.SwitchBasis = "real"
	}

	if strings.Contains(qLow, "autopsy") || strings.Contains(qLow, "year-by-year table") {
		switch {
		case strings.Contains(qLow, "p10"):
			intent.OpenAutopsy = "P10 Stress"
		case strings.Contains(qLow, "median") || strings.Contains(qLow, "p50"):
			intent.OpenAutopsy = "Median Path"
		case strings.Contains(qLow, "bull") || strings.Contains(qLow, "p90"):
			intent.OpenAutopsy = "P90 Bull"
		default:
			intent.OpenAutopsy = "Worst Path"
		}
	}

	switch {
	case strings.Contains(qLow, "crisis") || strings.Contains(qLow, "crises") || strings.Contains(qLow, "1929") || strings.Contains(qLow, "2008"):
		intent.OpenLabTab = "crises"
		intent.ScrollSection = "battle-lab"
	case strings.Contains(qLow, "sequence risk") || strings.Contains(qLow, "srr") || strings.Contains(qLow, "heatmap"):
		intent.OpenLabTab = "srr"
		intent.ScrollSection = "battle-lab"
	case strings.Contains(qLow, "guardrail") && strings.Contains(qLow, "band"):
		intent.OpenLabTab = "guardrails"
		intent.ScrollSection = "battle-lab"
	case strings.Contains(qLow, "action lever"):
		intent.OpenLabTab = "levers"
		intent.ScrollSection = "battle-lab"
	case intent.IsTaxOrLiquidity && (strings.Contains(qLow, "bridge") || strings.Contains(qLow, "liquid") || strings.Contains(qLow, "locked") || strings.Contains(qLow, "cash flow") || strings.Contains(qLow, "cash-flow")):
		intent.ScrollSection = "cashflow-bridge"
	case strings.Contains(qLow, "frontier") || strings.Contains(qLow, "when can i retire"):
		intent.ScrollSection = "frontier"
	case strings.Contains(qLow, "sources") || strings.Contains(qLow, "datasets"):
		if hasMut {
			intent.OpenSources = true
		}
	}

	return intent
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
