package server

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/ausanyal/firesage/internal/data"
	"github.com/ausanyal/firesage/internal/engine"
)

// NewMux constructs the HTTP router serving both stateless /api/v1/* endpoints and embedded SPA assets.
func NewMux() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/v1/health", handleHealth)
	mux.HandleFunc("GET /api/v1/metadata", handleMetadata)
	mux.HandleFunc("POST /api/v1/analyze", handleAnalyze)

	// Embedded React/TypeScript SPA handler with client-side route fallback
	mux.Handle("/", newSPAHandler())

	return withCORSAndSecurityHeaders(mux)
}

func handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status":    "ok",
		"service":   "firesage",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

func handleMetadata(w http.ResponseWriter, _ *http.Request) {
	usCurrPE, usMedPE := data.CurrentValuation(data.CountryUSA)
	inCurrPE, inMedPE := data.CurrentValuation(data.CountryIndia)

	writeJSON(w, http.StatusOK, map[string]any{
		"cities": data.Cities(),
		"crises": map[string]any{
			"IN": data.CrisesByCountry(data.CountryIndia),
			"US": data.CrisesByCountry(data.CountryUSA),
		},
		"valuations": map[string]any{
			"IN": map[string]any{
				"metric":            "Nifty 50 Trailing P/E",
				"current":           inCurrPE,
				"historical_median": inMedPE,
				"history_span":      "1980–2025 (46 yrs: BSE Sensex TR / Nifty 50 TRI, RBI G-Sec, MoSPI CPI)",
			},
			"US": map[string]any{
				"metric":            "Shiller CAPE (P/E10)",
				"current":           usCurrPE,
				"historical_median": usMedPE,
				"history_span":      "1928–2025 (98 yrs: Shiller Yale / Damodaran NYU Stern, 10Y Treasury, BLS CPI)",
			},
		},
		"defaults": map[string]any{
			"IN": defaultCountryTemplate(data.CountryIndia),
			"US": defaultCountryTemplate(data.CountryUSA),
		},
		"fx_usd_to_inr": data.DefaultUSDToINR,
	})
}

func handleAnalyze(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1 MB max body
	var req engine.PlanInput
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid JSON request payload: " + err.Error(),
		})
		return
	}

	result := engine.RunFullAnalysis(req)
	writeJSON(w, http.StatusOK, result)
}

func defaultCountryTemplate(country data.CountryCode) engine.PlanInput {
	if country == data.CountryUSA {
		return engine.NormalizePlanInput(engine.PlanInput{
			Country:                  data.CountryUSA,
			CurrentAge:               34,
			RetirementAge:            46,
			HorizonAge:               92,
			IsCouple:                 true,
			CurrentCityID:            "us-san-jose",
			RetirementCityID:         "us-raleigh",
			AutoScaleLifestyleOnMove: true,
			CurrentCorpus:            650_000,
			Buckets: engine.AccountBuckets{
				UseGranular:       true,
				TaxableEquity:     260_000,
				TaxableCostBasis:  0.65,
				TaxableDebtCash:   52_000,
				TaxDeferredLocked: 247_000,
				TaxFreeAccount:    91_000,
				PensionLocked:     0,
			},
			MonthlySavings:            3_250,
			MonthlyLockedSavings:      3_250,
			AnnualSavingsStepUp:       0.04,
			DesiredMonthlySpend:       10_500, // San Jose standard; scales automatically on Raleigh move!
			Allocation:                engine.AssetAllocation{Equity: 0.72, Debt: 0.20, Gold: 0.03, Cash: 0.05, UseBondTent: true},
			ExpenseRatioBps:           12,
			WithdrawalStrategy:        engine.StrategyGuytonKlinger,
			GuardrailFloorPct:         0.82,
			GuardrailCeilingPct:       1.25,
			UseSpendingSmile:          true,
			SocialSecurityMonthlyReal: 3_400,
			SocialSecurityStartAge:    67,
			EnableTaxOptimization:     true,
			Mode:                      engine.ModeBootstrap,
			NumPaths:                  3000,
		})
	}

	return engine.NormalizePlanInput(engine.PlanInput{
		Country:                  data.CountryIndia,
		CurrentAge:               33,
		RetirementAge:            45,
		HorizonAge:               90,
		IsCouple:                 true,
		CurrentCityID:            "in-bengaluru",
		RetirementCityID:         "in-goa",
		AutoScaleLifestyleOnMove: true,
		CurrentCorpus:            18_000_000, // ₹1.80 Cr
		Buckets: engine.AccountBuckets{
			UseGranular:       true,
			TaxableEquity:     8_640_000,
			TaxableCostBasis:  0.65,
			TaxableDebtCash:   2_520_000,
			TaxDeferredLocked: 3_600_000,
			TaxFreeAccount:    1_800_000,
			PensionLocked:     1_440_000,
		},
		MonthlySavings:            105_000, // ₹1.05 L/mo liquid MF SIP
		MonthlyLockedSavings:      45_000,  // ₹45 K/mo EPF + VPF + NPS
		AnnualSavingsStepUp:       0.07,    // 7%/yr career step-up
		DesiredMonthlySpend:       140_000, // ₹1.40 L/mo Bengaluru equivalent (~₹1.09L in Goa)
		Allocation:                engine.AssetAllocation{Equity: 0.65, Debt: 0.22, Gold: 0.08, Cash: 0.05, UseBondTent: true},
		ExpenseRatioBps:           25,
		WithdrawalStrategy:        engine.StrategyGuytonKlinger,
		GuardrailFloorPct:         0.82,
		GuardrailCeilingPct:       1.25,
		UseSpendingSmile:          true,
		SocialSecurityMonthlyReal: 0,
		SocialSecurityStartAge:    60,
		EnableTaxOptimization:     true,
		Mode:                      engine.ModeBootstrap,
		NumPaths:                  3000,
	})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func withCORSAndSecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "SAMEORIGIN")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
