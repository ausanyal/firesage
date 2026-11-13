package copilot

import "github.com/ausanyal/firesage/internal/engine"

// PersonaMode controls how Stage 3 synthesizes the quantitative tool outputs.
type PersonaMode string

const (
	PersonaAnalyst PersonaMode = "analyst"
	PersonaELI5    PersonaMode = "eli5"
	PersonaDebate  PersonaMode = "debate"
	PersonaDevil   PersonaMode = "devil"
)

// PlannerSkill describes an attached domain specialist invoked by Stage 1 Planner.
type PlannerSkill struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// SkillsRegistry mirrors StockSage's PLANNER_SKILLS_REGISTRY adapted for FireSage's 5-jurisdiction engine.
var SkillsRegistry = map[string]PlannerSkill{
	"skill_page_driver": {
		ID:          "skill_page_driver",
		Name:        "UI Page-Driver & Bi-Directional Plan Controller",
		Description: "Drives FireSage UI state: updates live PlanInput parameters, switches 5 jurisdictions & 34 global cities, toggles Real vs. Nominal basis, opens Trajectory Autopsy, switches Battle-Test Lab tabs, and scrolls to target charts.",
	},
	"skill_monte_carlo_srr": {
		ID:          "skill_monte_carlo_srr",
		Name:        "Stationary Block-Bootstrap & Sequence-of-Returns Auditor",
		Description: "Audits Politis-Romano (1994) block-bootstrap survival odds, CAPE/P-E valuation drag, implied initial SWR, and early-retirement sequence-of-returns tail risk.",
	},
	"skill_liquidity_tax_bridge": {
		ID:          "skill_liquidity_tax_bridge",
		Name:        "Statutory Lock-In, Liquidity Bridge & Global Tax Strategist",
		Description: "Audits pre-unlock liquid bridges (Age 60 IN/US/AU/EU, Age 65 NZ KiwiSaver), statutory withdrawal taxes (Sec 112A, 2026 ACA 400% FPL cliff, AU Super/Centrelink, CH LAMal/IT 7%/PT CGT, NZ FIF/PIE), and healthcare inflation.",
	},
	"skill_geo_arbitrage_scanner": {
		ID:          "skill_geo_arbitrage_scanner",
		Name:        "34-City Intra-Country & Cross-Border Geo-Arbitrage Comparator",
		Description: "Runs parallel stochastic simulations across domestic and cross-border retirement cities, isolating 0.65*COL + 0.35*Housing scaling, FX parity, and destination residency tax/medical regimes.",
	},
	"skill_goal_seek_solver": {
		ID:          "skill_goal_seek_solver",
		Name:        "CRN Secant/Bisection Goal-Seek Optimizer",
		Description: "Solves exact required monthly liquid SIP, lump-sum corpus today, sustainable monthly budget, and earliest 90%/95% FIRE age using Common Random Numbers.",
	},
	"skill_crisis_stress_tester": {
		ID:          "skill_crisis_stress_tester",
		Name:        "Historical Crisis Replay & Stagflation/Medical Shock Simulator",
		Description: "Replays famous historical crashes (1929, 1966, 1973, 1987, 1992, 2000, 2008, 2022) and runs custom Year-1 crash + multi-year inflation + medical shock stress tests.",
	},
	"skill_life_events_architect": {
		ID:          "skill_life_events_architect",
		Name:        "Custom Life-Event Cash-Flow & Coast-FIRE Bridge Modeler",
		Description: "Models multi-year consulting/Barista bridges, sabbaticals, kids' college outlays, weddings, and home-equity downsizing inflows.",
	},
}

// PlanStep represents one numbered step in the Planner's execution plan.
type PlanStep struct {
	Step   int    `json:"step"`
	Title  string `json:"title"`
	Detail string `json:"detail"`
	Status string `json:"status"`
}

// PlannerExecutionPlan holds Stage 1 Planner outputs rendered in the Copilot trace card.
type PlannerExecutionPlan struct {
	ThoughtSummary string     `json:"thought_summary"`
	Steps          []PlanStep `json:"steps"`
	Hypotheses     []string   `json:"hypotheses"`
	SkillsInvoked  []string   `json:"skills_invoked"`
	PlannedTools   []string   `json:"planned_tools"`
}

// ToolCallTrace records an executed Stage 2 quantitative tool call with timing and result summary.
type ToolCallTrace struct {
	ID          string  `json:"id"`
	Tool        string  `json:"tool"`
	ArgsSummary string  `json:"args_summary"`
	DurationMs  float64 `json:"duration_ms"`
	Status      string  `json:"status"`
	Summary     string  `json:"summary"`
}

// UICommand instructs the React frontend to drive or mutate the live dashboard.
type UICommand struct {
	Cmd       string            `json:"cmd"`
	Reason    string            `json:"reason"`
	Country   string            `json:"country,omitempty"`
	Basis     string            `json:"basis,omitempty"`
	PathLabel string            `json:"path_label,omitempty"`
	Tab       string            `json:"tab,omitempty"`
	Section   string            `json:"section,omitempty"`
	PlanInput *engine.PlanInput `json:"plan_input,omitempty"`
}

// CopilotAction represents a 1-click follow-up chip or UI mutation button attached to a bot reply.
type CopilotAction struct {
	Label     string     `json:"label"`
	Prompt    string     `json:"prompt,omitempty"`
	UICommand *UICommand `json:"ui_command,omitempty"`
}

// ChatTurn represents a prior message in conversation history.
type ChatTurn struct {
	Role string `json:"role"`
	Text string `json:"text"`
}

// ChatRequest is accepted by POST /api/v1/chat and WASM firesageRunChat.
type ChatRequest struct {
	Question   string           `json:"question"`
	Mode       string           `json:"mode"`
	Input      engine.PlanInput `json:"input"`
	ValueBasis string           `json:"value_basis"`
	History    []ChatTurn       `json:"history"`
	APIKey     string           `json:"api_key,omitempty"`
	Model      string           `json:"model,omitempty"`
}

// ChatResponse is returned by POST /api/v1/chat and WASM firesageRunChat.
type ChatResponse struct {
	OK           bool                 `json:"ok"`
	EngineLabel  string               `json:"engine_label"`
	UsedLLM      bool                 `json:"used_llm"`
	LLMNote      string               `json:"llm_note,omitempty"`
	Plan         PlannerExecutionPlan `json:"plan"`
	ToolTrace    []ToolCallTrace      `json:"tool_trace"`
	UICommands   []UICommand          `json:"ui_commands"`
	EvidencePack string               `json:"evidence_pack"`
	Markdown     string               `json:"markdown"`
	Actions      []CopilotAction      `json:"actions"`
}
