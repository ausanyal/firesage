package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ausanyal/firesage/internal/copilot"
	"github.com/ausanyal/firesage/internal/data"
	"github.com/ausanyal/firesage/internal/engine"
)

type testMetadataResponse struct {
	Cities        []data.CityProfile                     `json:"cities"`
	Crises        map[string][]data.HistoricalCrisis     `json:"crises"`
	Valuations    map[string]map[string]any              `json:"valuations"`
	Defaults      map[data.CountryCode]engine.PlanInput  `json:"defaults"`
	FXRatesPerUSD map[data.CountryCode]float64           `json:"fx_rates_per_usd"`
}

func TestHTTPServerEndpointsEndToEnd(t *testing.T) {
	mux := NewMux()

	// 1. Health endpoint & security headers
	reqHealth := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	recHealth := httptest.NewRecorder()
	mux.ServeHTTP(recHealth, reqHealth)
	if recHealth.Code != http.StatusOK {
		t.Fatalf("GET /api/v1/health status = %d, want 200", recHealth.Code)
	}
	if recHealth.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("missing X-Content-Type-Options: nosniff header")
	}

	// 2. Metadata endpoint: 5 countries, 34 global cities, FX rates, templates
	reqMeta := httptest.NewRequest(http.MethodGet, "/api/v1/metadata", nil)
	recMeta := httptest.NewRecorder()
	mux.ServeHTTP(recMeta, reqMeta)
	if recMeta.Code != http.StatusOK {
		t.Fatalf("GET /api/v1/metadata status = %d, want 200", recMeta.Code)
	}
	var meta testMetadataResponse
	if err := json.Unmarshal(recMeta.Body.Bytes(), &meta); err != nil {
		t.Fatalf("unmarshal metadata failed: %v", err)
	}
	if len(meta.Defaults) != 5 || len(meta.Valuations) != 5 || len(meta.FXRatesPerUSD) != 5 {
		t.Fatalf("expected 5 jurisdictions in metadata defaults/valuations/fx, got %d/%d/%d",
			len(meta.Defaults), len(meta.Valuations), len(meta.FXRatesPerUSD))
	}
	if len(meta.Cities) != 34 {
		t.Fatalf("expected 34 cities in /api/v1/metadata, got %d", len(meta.Cities))
	}

	// 3. POST /api/v1/analyze: Bengaluru -> Kolkata & cross-border Bengaluru -> Zurich Suburbs
	plan := meta.Defaults[data.CountryIndia]
	plan.RetirementCityID = "in-kolkata"
	plan.AutoScaleLifestyleOnMove = true
	plan.NumPaths = 400
	body, err := json.Marshal(plan)
	if err != nil {
		t.Fatalf("marshal plan: %v", err)
	}
	reqAnalyze := httptest.NewRequest(http.MethodPost, "/api/v1/analyze", bytes.NewReader(body))
	reqAnalyze.Header.Set("Content-Type", "application/json")
	recAnalyze := httptest.NewRecorder()
	mux.ServeHTTP(recAnalyze, reqAnalyze)
	if recAnalyze.Code != http.StatusOK {
		t.Fatalf("POST /api/v1/analyze status = %d, body = %s", recAnalyze.Code, recAnalyze.Body.String())
	}
	var analysis engine.FullAnalysisResponse
	if err := json.Unmarshal(recAnalyze.Body.Bytes(), &analysis); err != nil {
		t.Fatalf("unmarshal FullAnalysisResponse failed: %v", err)
	}
	if analysis.GeoArbitrage.ComparisonCity.ID != "in-kolkata" {
		t.Fatalf("expected ComparisonCity in-kolkata, got %s", analysis.GeoArbitrage.ComparisonCity.ID)
	}
	if analysis.SuccessRate <= 0 || analysis.RequiredCorpus90 <= 0 || analysis.SafeMonthlySpend90 <= 0 {
		t.Fatalf("unexpected non-positive KPI outputs: %+v", analysis)
	}

	// 4. Malformed JSON returns 400 Bad Request
	reqBad := httptest.NewRequest(http.MethodPost, "/api/v1/analyze", bytes.NewBufferString("{bad-json"))
	recBad := httptest.NewRecorder()
	mux.ServeHTTP(recBad, reqBad)
	if recBad.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 on malformed JSON, got %d", recBad.Code)
	}

	// 5. OPTIONS preflight returns 204 No Content
	reqOpt := httptest.NewRequest(http.MethodOptions, "/api/v1/analyze", nil)
	recOpt := httptest.NewRecorder()
	mux.ServeHTTP(recOpt, reqOpt)
	if recOpt.Code != http.StatusNoContent {
		t.Fatalf("expected 204 on OPTIONS preflight, got %d", recOpt.Code)
	}

	// 6. SPA fallback returns 200 text/html
	reqSPA := httptest.NewRequest(http.MethodGet, "/", nil)
	recSPA := httptest.NewRecorder()
	mux.ServeHTTP(recSPA, reqSPA)
	if recSPA.Code != http.StatusOK {
		t.Fatalf("GET / status = %d, want 200", recSPA.Code)
	}

	// 7. POST /api/v1/chat runs Stage 1 Planner + Stage 2 Quantitative Tools + Stage 3 Synthesis
	chatReqObj := copilot.ChatRequest{
		Question:   "Compare retiring in Bengaluru vs Kolkata vs Algarve and solve required SIP for 90% survival",
		Mode:       "analyst",
		Input:      plan,
		ValueBasis: "real",
	}
	chatBytes, _ := json.Marshal(chatReqObj)
	reqChat := httptest.NewRequest(http.MethodPost, "/api/v1/chat", bytes.NewReader(chatBytes))
	reqChat.Header.Set("Content-Type", "application/json")
	recChat := httptest.NewRecorder()
	mux.ServeHTTP(recChat, reqChat)
	if recChat.Code != http.StatusOK {
		t.Fatalf("POST /api/v1/chat status = %d, body = %s", recChat.Code, recChat.Body.String())
	}
	var chatResp copilot.ChatResponse
	if err := json.Unmarshal(recChat.Body.Bytes(), &chatResp); err != nil {
		t.Fatalf("unmarshal ChatResponse failed: %v", err)
	}
	if !chatResp.OK || len(chatResp.Plan.Steps) != 3 || len(chatResp.ToolTrace) < 3 || chatResp.Markdown == "" {
		t.Fatalf("unexpected ChatResponse: %+v", chatResp)
	}
}
