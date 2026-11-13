package copilot

import (
	"strings"
	"testing"

	"github.com/ausanyal/firesage/internal/data"
	"github.com/ausanyal/firesage/internal/engine"
)

func TestExtractMentionedCitiesGlobal34(t *testing.T) {
	cities := ExtractMentionedCities("Compare retiring in Bangalore vs Calcutta vs Cochin vs Algarve vs South of Italy vs Lugano")
	wantIDs := []string{"in-bengaluru", "in-kolkata", "in-kochi", "eu-algarve", "eu-south-italy", "eu-lugano"}
	if len(cities) != len(wantIDs) {
		t.Fatalf("got %d cities (%+v), want %d", len(cities), cities, len(wantIDs))
	}
	for i, w := range wantIDs {
		if cities[i].ID != w {
			t.Fatalf("city[%d] = %s, want %s", i, cities[i].ID, w)
		}
	}
}

func TestExecutePipelinePlannerToolsAndPersonas(t *testing.T) {
	base := engine.DefaultCountryTemplate(data.CountryIndia)

	// 1. Multi-city comparison + Goal-Seek in Analyst mode
	resp := ExecutePipeline(ChatRequest{
		Question:   "Compare retiring in Bengaluru vs Kolkata vs Goa vs Dehradun and tell me how much monthly SIP I need for 90% confidence",
		Mode:       "analyst",
		Input:      base,
		ValueBasis: "real",
	})
	if !resp.OK {
		t.Fatalf("expected OK=true")
	}
	if len(resp.Plan.Steps) != 3 || len(resp.Plan.Hypotheses) != 3 {
		t.Fatalf("expected 3 plan steps and 3 hypotheses, got %d and %d", len(resp.Plan.Steps), len(resp.Plan.Hypotheses))
	}
	hasGeoTool := false
	hasGoalSeek := false
	for _, tc := range resp.ToolTrace {
		if tc.Tool == "compare_cities_geo_arbitrage" {
			hasGeoTool = true
		}
		if tc.Tool == "solve_goal_seek_targets" {
			hasGoalSeek = true
		}
	}
	if !hasGeoTool || !hasGoalSeek {
		t.Fatalf("expected compare_cities_geo_arbitrage and solve_goal_seek_targets in tool_trace, got %+v", resp.ToolTrace)
	}
	if !strings.Contains(resp.Markdown, "Kolkata") || !strings.Contains(resp.Markdown, "Multi-City Geo-Arbitrage Scorecard") {
		t.Fatalf("markdown missing Kolkata geo-arbitrage scorecard:\n%s", resp.Markdown)
	}

	// 2. Explicit UI Page-Driver mutation ("Set retirement age to 48 and switch to Goa and open historical crises")
	mutResp := ExecutePipeline(ChatRequest{
		Question:   "Set retirement age to 48, move to Goa, and open historical crises",
		Mode:       "debate",
		Input:      base,
		ValueBasis: "real",
	})
	if len(mutResp.UICommands) < 2 {
		t.Fatalf("expected at least 2 UICommands (APPLY_PLAN_PATCH + OPEN_LAB_TAB), got %+v", mutResp.UICommands)
	}
	foundPatch := false
	foundTab := false
	for _, cmd := range mutResp.UICommands {
		if cmd.Cmd == "APPLY_PLAN_PATCH" && cmd.PlanInput != nil {
			if cmd.PlanInput.RetirementAge != 48 || cmd.PlanInput.RetirementCityID != "in-goa" {
				t.Fatalf("unexpected patched PlanInput: age=%d city=%s", cmd.PlanInput.RetirementAge, cmd.PlanInput.RetirementCityID)
			}
			foundPatch = true
		}
		if cmd.Cmd == "OPEN_LAB_TAB" && cmd.Tab == "crises" {
			foundTab = true
		}
	}
	if !foundPatch || !foundTab {
		t.Fatalf("missing APPLY_PLAN_PATCH or OPEN_LAB_TAB in %+v", mutResp.UICommands)
	}
	if !strings.Contains(mutResp.Markdown, "The Bull Case") || !strings.Contains(mutResp.Markdown, "The Bear Case") {
		t.Fatalf("debate persona missing Bull/Bear sections:\n%s", mutResp.Markdown)
	}

	// 3. Devil's Advocate stress shock & Life Event on New Zealand plan (Age-65 KiwiSaver unlock)
	nzPlan := engine.DefaultCountryTemplate(data.CountryNewZealand)
	devilResp := ExecutePipeline(ChatRequest{
		Question:   "What if we add a consulting bridge and face a -40% crash?",
		Mode:       "devil",
		Input:      nzPlan,
		ValueBasis: "real",
	})
	hasStress := false
	hasEvent := false
	for _, tc := range devilResp.ToolTrace {
		if tc.Tool == "run_stress_and_crisis_shock" {
			hasStress = true
		}
		if tc.Tool == "simulate_life_event" {
			hasEvent = true
		}
	}
	if !hasStress || !hasEvent {
		t.Fatalf("expected stress and life_event tools in devil mode, got %+v", devilResp.ToolTrace)
	}

	// 4. ELI5 + 35x multiplier explanation on Europe plan
	euPlan := engine.DefaultCountryTemplate(data.CountryEurope)
	eli5Resp := ExecutePipeline(ChatRequest{
		Question:   "Why not just use 35x annual expenses?",
		Mode:       "eli5",
		Input:      euPlan,
		ValueBasis: "real",
	})
	if !strings.Contains(eli5Resp.Markdown, "Plain-English Breakdown") || !strings.Contains(eli5Resp.Markdown, "35x") {
		t.Fatalf("ELI5 35x explanation missing expected sections:\n%s", eli5Resp.Markdown)
	}
}
