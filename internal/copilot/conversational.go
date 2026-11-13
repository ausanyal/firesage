package copilot

import (
	"fmt"
	"math"
	"regexp"
	"strings"

	"github.com/ausanyal/firesage/internal/data"
	"github.com/ausanyal/firesage/internal/engine"
)

// QueryKind distinguishes casual/meta chat and conceptual FIRE questions from heavy simulation runs.
type QueryKind string

const (
	KindConversationalMeta      QueryKind = "conversational_meta"
	KindConceptualFIRE          QueryKind = "conceptual_fire"
	KindFollowUp                QueryKind = "follow_up"
	KindConversationalPlanCheck QueryKind = "plan_check"
	KindQuantitativeDeepDive    QueryKind = "quant_deep_dive"
)

var (
	reGreetingOrMeta = regexp.MustCompile(`(?i)^\s*(hi|hello|hey|howdy|yo|good\s+(morning|afternoon|evening)|greetings|namaste|sup)\b|[?\s]*(can\s+i\s+ask|may\s+i\s+ask|can\s+you\s+(answer|help|explain|chat|talk)|what\s+can\s+you\s+(do|answer|help\s+with)|who\s+are\s+you|what\s+are\s+you|are\s+you\s+an?\s+(ai|bot|chatbot|llm)|how\s+do\s+i\s+use\s+(this|you)|help\s*me\s*get\s*started|^\s*help\s*[?!.]*$|^\s*(thanks|thank\s+you|thx|ty|cool|awesome|great|got\s+it|makes\s+sense|ok|okay)\b)`)
	reThanksOnly     = regexp.MustCompile(`(?i)^\s*(thanks|thank\s+you|thx|ty|awesome|great|perfect|got\s+it|makes\s+sense|super\s+helpful)[!.\s]*$`)
	reFollowUpShort  = regexp.MustCompile(`(?i)^\s*(yes|yep|yeah|sure|please|go\s+ahead|do\s+it|tell\s+me\s+more|explain\s+more|elaborate|give\s+(me\s+)?an?\s+example|why\s*\??|how\s+so\s*\??|in\s+simple\s+(words|english|terms)|simplify(\s+that)?)\s*[?!.]*$`)

	// General / Conceptual FIRE question cues (when no personal slider mutation or multi-city comparison is requested)
	reConceptualCue = regexp.MustCompile(`(?i)\b(what\s+is|what's|what\s+are|what\s+does|meaning\s+of|define|explain|difference\s+between|how\s+does\s+.*\s+work|why\s+do\s+people|why\s+is\s+.*\s+important|is\s+it\s+better\s+to|should\s+one|pros\s+and\s+cons|rule\s+of\s+thumb|guide\s+to|teach\s+me|understand)\b`)
	rePersonalPlan  = regexp.MustCompile(`(?i)\b(my\s+(plan|corpus|portfolio|sip|savings|survival|budget|retirement|numbers|target)|am\s+i\s+on\s+track|is\s+my\s+|audit\s+my|roast\s+my|how\s+does\s+my\s+plan|review\s+my|check\s+my|for\s+me\b)\b`)
)

// ClassifyQueryKind determines whether the user is chatting casually, asking a conceptual FIRE question,
// following up on the prior turn, asking a conversational check on their plan, or requesting a deep tool run.
func ClassifyQueryKind(question string, intent ParsedIntent, history []ChatTurn) (QueryKind, string) {
	q := strings.TrimSpace(question)
	qLow := strings.ToLower(q)

	hasHardQuantTrigger := intent.IsExplicitUIMutation ||
		intent.IsComparison ||
		len(intent.MentionedCities) >= 2 ||
		intent.NewCurrentAge != nil ||
		intent.NewRetirementAge != nil ||
		intent.NewHorizonAge != nil ||
		intent.NewMonthlySpend != nil ||
		intent.NewCurrentCorpus != nil ||
		intent.NewMonthlySavings != nil ||
		intent.NewEquityPct != nil ||
		intent.CrashPct != nil ||
		intent.ExtraInfPct != nil ||
		intent.MedicalShock != nil ||
		intent.SwitchCountry != nil ||
		intent.OpenAutopsy != "" ||
		intent.OpenLabTab != ""

	if hasHardQuantTrigger {
		return KindQuantitativeDeepDive, ""
	}

	// 1. Short gratitude / acknowledgment
	if reThanksOnly.MatchString(q) {
		return KindConversationalMeta, "thanks"
	}

	// 2. Greeting, capability check ("can i ask u FIRE related questions"), or "what can you do?"
	if reGreetingOrMeta.MatchString(q) && !rePersonalPlan.MatchString(q) && !intent.IsGoalSeek && !intent.IsStressOrCrisis {
		return KindConversationalMeta, "greeting_or_capabilities"
	}
	// Specifically catch "can i ask ... fire ... questions" even if "fire" triggered a keyword
	if strings.Contains(qLow, "can i ask") || strings.Contains(qLow, "may i ask") || strings.Contains(qLow, "can we talk") || strings.Contains(qLow, "can we chat") || strings.Contains(qLow, "ask u ") || strings.Contains(qLow, "ask you ") {
		if strings.Contains(qLow, "question") || strings.Contains(qLow, "about fire") || strings.Contains(qLow, "anything") || len(q) < 90 {
			return KindConversationalMeta, "greeting_or_capabilities"
		}
	}

	// 3. Short conversational follow-up ("tell me more", "give me an example", "yes", "explain simply")
	if reFollowUpShort.MatchString(q) && len(history) > 0 {
		return KindFollowUp, "follow_up"
	}

	// 4. Conceptual / Educational FIRE & Personal Finance Topics
	topic := DetectFIREConceptTopic(qLow)
	if topic != "" && !rePersonalPlan.MatchString(q) {
		// If they didn't explicitly ask "how much SIP do I need" or "audit my plan", answer conceptually + tie into their plan!
		if reConceptualCue.MatchString(q) || !intent.IsGoalSeek {
			return KindConceptualFIRE, topic
		}
	}

	// 5. General natural-language plan check ("am i on track?", "is my plan safe?", "how am i doing?", "what's my biggest risk?")
	if !intent.IsComparison && !intent.IsStressOrCrisis && !intent.IsLifeEvent {
		return KindConversationalPlanCheck, ""
	}

	return KindQuantitativeDeepDive, ""
}

// DetectFIREConceptTopic identifies core FIRE, investing, withdrawal, tax, housing, and lifestyle concepts.
func DetectFIREConceptTopic(qLow string) string {
	switch {
	case strings.Contains(qLow, "coast fire") || strings.Contains(qLow, "lean fire") || strings.Contains(qLow, "fat fire") ||
		strings.Contains(qLow, "chubby fire") || strings.Contains(qLow, "barista fire") || strings.Contains(qLow, "flamingo fire") ||
		strings.Contains(qLow, "types of fire") || strings.Contains(qLow, "flavors of fire") || strings.Contains(qLow, "kinds of fire"):
		return "fire_types"

	case (strings.Contains(qLow, "what is fire") || strings.Contains(qLow, "what's fire") || strings.Contains(qLow, "meaning of fire") ||
		strings.Contains(qLow, "explain fire") || strings.Contains(qLow, "new to fire") || strings.Contains(qLow, "how does fire work") ||
		strings.Contains(qLow, "financial independence")):
		return "fire_overview"

	case strings.Contains(qLow, "4% rule") || strings.Contains(qLow, "four percent rule") || strings.Contains(qLow, "trinity study") ||
		strings.Contains(qLow, "safe withdrawal rate") || strings.Contains(qLow, "what is swr") || strings.Contains(qLow, "35x") ||
		strings.Contains(qLow, "25x") || strings.Contains(qLow, "30x") || strings.Contains(qLow, "33x") || strings.Contains(qLow, "multiplier"):
		return "swr_multipliers"

	case strings.Contains(qLow, "sequence of return") || strings.Contains(qLow, "sequence risk") || strings.Contains(qLow, "sorr") ||
		strings.Contains(qLow, "bond tent") || strings.Contains(qLow, "glidepath") || strings.Contains(qLow, "bucket strategy") ||
		strings.Contains(qLow, "three bucket") || strings.Contains(qLow, "3 bucket") || strings.Contains(qLow, "cash wedge"):
		return "sorr_bond_tent"

	case strings.Contains(qLow, "guyton") || strings.Contains(qLow, "klinger") || strings.Contains(qLow, "spending smile") ||
		strings.Contains(qLow, "blanchett") || (strings.Contains(qLow, "guardrail") && reConceptualCue.MatchString(qLow)) ||
		(strings.Contains(qLow, "withdrawal strateg") && reConceptualCue.MatchString(qLow)):
		return "withdrawal_guardrails"

	case strings.Contains(qLow, "rent vs buy") || strings.Contains(qLow, "rent or buy") || strings.Contains(qLow, "buy a house") ||
		strings.Contains(qLow, "buy a home") || strings.Contains(qLow, "mortgage") || strings.Contains(qLow, "home loan") ||
		strings.Contains(qLow, "primary residence"):
		return "rent_vs_buy"

	case strings.Contains(qLow, "asset allocation") || strings.Contains(qLow, "stocks vs bonds") || strings.Contains(qLow, "equity vs debt") ||
		strings.Contains(qLow, "index fund") || strings.Contains(qLow, "gold in") || strings.Contains(qLow, "rebalanc") ||
		strings.Contains(qLow, "lump sum vs sip") || strings.Contains(qLow, "dca vs lump"):
		return "asset_allocation"

	case strings.Contains(qLow, "roth ladder") || strings.Contains(qLow, "roth conversion") || strings.Contains(qLow, "72(t)") ||
		strings.Contains(qLow, "72t") || strings.Contains(qLow, "sepp") || strings.Contains(qLow, "rule of 55") ||
		(reConceptualCue.MatchString(qLow) && (strings.Contains(qLow, "epf") || strings.Contains(qLow, "ppf") || strings.Contains(qLow, "nps") ||
			strings.Contains(qLow, "401k") || strings.Contains(qLow, "401(k)") || strings.Contains(qLow, "superannuation") || strings.Contains(qLow, "kiwisaver"))):
		return "pension_unlock_strategies"

	case strings.Contains(qLow, "medical inflation") || strings.Contains(qLow, "healthcare inflation") || strings.Contains(qLow, "health insurance") ||
		strings.Contains(qLow, "super top") || strings.Contains(qLow, "aca cliff") || strings.Contains(qLow, "aca subsidy") ||
		strings.Contains(qLow, "irmaa") || strings.Contains(qLow, "lamal"):
		return "healthcare_inflation"

	case reConceptualCue.MatchString(qLow) && (strings.Contains(qLow, "tax") || strings.Contains(qLow, "ltcg") || strings.Contains(qLow, "capital gain") || strings.Contains(qLow, "inflation")):
		return "tax_and_inflation"

	case strings.Contains(qLow, "geo arbitrage") || strings.Contains(qLow, "geo-arbitrage") || strings.Contains(qLow, "retire abroad") ||
		strings.Contains(qLow, "rnor") || strings.Contains(qLow, "nri") || strings.Contains(qLow, "7% flat tax") || strings.Contains(qLow, "golden visa"):
		return "geo_arbitrage_concept"

	case strings.Contains(qLow, "savings rate") || strings.Contains(qLow, "retire faster") || strings.Contains(qLow, "step-up sip") ||
		strings.Contains(qLow, "step up sip") || strings.Contains(qLow, "lifestyle creep"):
		return "savings_velocity"

	case strings.Contains(qLow, "one more year") || strings.Contains(qLow, "bored in retirement") || strings.Contains(qLow, "die with zero") ||
		strings.Contains(qLow, "psycholog") || strings.Contains(qLow, "purpose in retirement") || strings.Contains(qLow, "fear of running out"):
		return "psychology_fire"

	case strings.Contains(qLow, "kids") || strings.Contains(qLow, "children") || strings.Contains(qLow, "aging parents") ||
		strings.Contains(qLow, "dependent parents") || strings.Contains(qLow, "spouse"):
		return "family_planning"

	case strings.Contains(qLow, "block bootstrap") || strings.Contains(qLow, "politis") || strings.Contains(qLow, "how does monte carlo") ||
		strings.Contains(qLow, "shiller cape") || strings.Contains(qLow, "cape ratio"):
		return "simulation_methodology"
	}
	return ""
}

// ExecuteConversationalTurn handles greetings, capability checks, conceptual FIRE Q&A, and conversational follow-ups
// without dumping robotic telemetry tables.
func ExecuteConversationalTurn(req ChatRequest, kind QueryKind, subtopic string) ChatResponse {
	userBase := engine.NormalizePlanInput(req.Input)
	curCity := data.GetCity(userBase.CurrentCityID, userBase.Country)
	retCity := data.GetCity(userBase.RetirementCityID, userBase.Country)
	cc := userBase.Country
	unlockAge := CountryUnlockAge(cc)

	// Run a super-lightweight 250-path snapshot (~4ms) ONLY so we can personalize 1 sentence or EvidencePack for Gemini
	snap := engine.RunFullAnalysis(fastPlan(userBase, 250))

	evidence := strings.Join([]string{
		fmt.Sprintf("[CONVERSATIONAL_CONTEXT] QueryKind=%s | Subtopic=%s | UserQuestion=%q", kind, subtopic, req.Question),
		fmt.Sprintf("[USER_ACTIVE_PLAN_SNAPSHOT] WorkJurisdiction=%s | CurrentCity=%s | RetireCity=%s | Ages=%d→%d→%d (UnlockAge=%d) | Corpus=%s | LiquidSIP=%s/mo + Locked=%s/mo | Spend=%s/mo (EffRetire=%s/mo) | Survival=%.1f%% | Pre-%dSafe=%.1f%% | P50CorpusAtFIRE=%s vs 90%%Required=%s (ImpliedSWR=%.2f%%) | CoastAge=%.1f, Target90%%Age=%.1f, Fortress97%%Age=%.1f",
			CountryDisplayName(cc), curCity.Name, retCity.Name,
			userBase.CurrentAge, userBase.RetirementAge, userBase.HorizonAge, unlockAge,
			FormatCompact(userBase.CurrentCorpus, cc),
			FormatCompact(userBase.MonthlySavings, cc),
			FormatCompact(userBase.MonthlyLockedSavings, cc),
			FormatCompact(userBase.DesiredMonthlySpend, cc),
			FormatCompact(snap.GeoArbitrage.EffectiveRetireMonthly, cc),
			snap.SuccessRate, unlockAge, snap.LiquiditySafeRate,
			FormatCompact(snap.CorpusAtFireRealP50, cc),
			FormatCompact(snap.RequiredCorpus90, cc),
			snap.ImpliedInitialSWR,
			snap.CoastFIREAge, snap.TargetFIREAge, snap.FortressFIREAge,
		),
	}, "\n")

	var md string
	var actions []CopilotAction

	switch kind {
	case KindConversationalMeta:
		if subtopic == "thanks" {
			md = fmt.Sprintf(
				"Anytime! Happy to think through things with you. Whenever you want, we can explore general FIRE concepts, stress-test market crashes, compare cities for your **%s → %s** plan, or tweak your sliders together — just ask in plain English.",
				curCity.Name, retCity.Name,
			)
		} else {
			md = strings.Join([]string{
				"Yes, of course! You can ask me **any FIRE, investing, tax, or retirement lifestyle question** in plain, everyday English — we don't have to stick to rigid commands or spreadsheets.",
				"",
				"Here are a few ways people like to chat with me:",
				"- **FIRE Concepts & Rules of Thumb:** *\"What's the difference between Coast FIRE, Lean FIRE, and Barista FIRE?\"*, *\"Why is 35x better or worse than the 4% rule?\"*, or *\"Should I pay off my home loan before retiring?\"*",
				fmt.Sprintf("- **Tax, Lock-Ins & Healthcare:** *\"How do I bridge early retirement before Age %d without touching locked accounts?\"* or *\"How bad is medical inflation in my 70s?\"*", unlockAge),
				fmt.Sprintf("- **Your Live Plan (%s → %s):** *\"Am I realistically on track to retire at %d?\"*, *\"How much monthly SIP gets me to 90%% safety?\"*, or *\"Compare retiring in %s vs Goa vs Algarve.\"*", curCity.Name, retCity.Name, userBase.RetirementAge, curCity.Name),
				"- **Hands-Free Dashboard Control:** Say *\"Set my retirement age to 48\"* or *\"Open the historical crises tab\"* and I'll drive the page for you.",
				"",
				"What's on your mind right now — a general FIRE concept, or a question about your own plan?",
			}, "\n")
		}
		actions = []CopilotAction{
			{Label: "💬 What are Coast, Lean, Barista & Fat FIRE?", Prompt: "What is the difference between Coast FIRE, Lean FIRE, Barista FIRE, and Fat FIRE?"},
			{Label: "💬 Why not just use the 4% rule or 35x?", Prompt: "Why does the 4% rule or 35x multiplier break down for early retirees?"},
			{Label: fmt.Sprintf("🎯 Am I on track to retire at %d?", userBase.RetirementAge), Prompt: fmt.Sprintf("Am I realistically on track to retire at Age %d in %s?", userBase.RetirementAge, retCity.Name)},
			{Label: "💬 Should I rent or buy a home for FIRE?", Prompt: "Is it better to rent or buy a home when planning for FIRE?"},
		}

	case KindFollowUp:
		lastBot := ""
		for i := len(req.History) - 1; i >= 0; i-- {
			if req.History[i].Role == "assistant" {
				lastBot = req.History[i].Text
				break
			}
		}
		md = buildFollowUpConversationalReply(req.Question, lastBot, userBase, snap, curCity, retCity)
		actions = []CopilotAction{
			{Label: fmt.Sprintf("🎯 Check my %s → %s plan odds", curCity.Name, retCity.Name), Prompt: "Give me a plain-English checkup on my current retirement plan."},
			{Label: "🌍 Compare top geo-arbitrage cities", Prompt: fmt.Sprintf("Compare retiring in %s vs top domestic and global cities", curCity.Name)},
			{Label: "💬 Explain sequence-of-returns risk simply", Prompt: "Explain sequence-of-returns risk with a simple real-world example."},
		}

	case KindConceptualFIRE:
		rawMD, conceptActions := buildConceptConversationalReply(subtopic, userBase, snap, curCity, retCity)
		md = applyConceptPersonaHeader(rawMD, PersonaMode(strings.ToLower(req.Mode)))
		actions = conceptActions
	}

	return ChatResponse{
		OK:           true,
		EngineLabel:  "💬 FIRE Copilot · Conversational Advisor",
		UsedLLM:      false,
		Plan:         PlannerExecutionPlan{}, // Keep casual/conceptual turns clean without heavy trace chrome
		ToolTrace:    nil,
		UICommands:   nil,
		EvidencePack: evidence,
		Markdown:     md,
		Actions:      actions,
	}
}

func buildFollowUpConversationalReply(q, lastBot string, plan engine.PlanInput, snap engine.FullAnalysisResponse, curCity, retCity data.CityProfile) string {
	cc := plan.Country
	annSpend := snap.GeoArbitrage.EffectiveRetireMonthly * 12
	mult := snap.RequiredCorpus90 / math.Max(1, annSpend)
	qLow := strings.ToLower(q)

	if strings.Contains(qLow, "example") || strings.Contains(qLow, "simple") || strings.Contains(qLow, "simplify") {
		return strings.Join([]string{
			"Sure — here's a concrete, real-world example using your own numbers:",
			"",
			fmt.Sprintf("Imagine two people both retiring at **Age %d** in **%s** spending **%s/mo** (`%s/yr`):", plan.RetirementAge, retCity.Name, FormatCompact(snap.GeoArbitrage.EffectiveRetireMonthly, cc), FormatCompact(annSpend, cc)),
			"1. **Lucky Timing (Bull Start):** Markets rise `+15%/yr` in their first 4 years of retirement. Even after paying bills and taxes, their portfolio snowballs faster than they spend it.",
			fmt.Sprintf("2. **Unlucky Timing (1973 / 2000 / 2008 Start):** Markets drop `-30%%` right after they quit their job while inflation jumps. Because they still have to withdraw **%s/yr** to live on, they're forced to sell shares at depressed prices — permanently shrinking the engine before the recovery arrives.", FormatCompact(annSpend, cc)),
			"",
			fmt.Sprintf("That early-storm risk is why FireSage targets **%s** (`%.1fx` your annual spend) for **90%% confidence** instead of relying on a single average return. Want to see what happens to your plan in a `-35%%` Year-1 crash?", FormatCompact(snap.RequiredCorpus90, cc), mult),
		}, "\n")
	}

	if strings.Contains(strings.ToLower(lastBot), "what's on your mind") || strings.Contains(strings.ToLower(lastBot), "ways people like to chat") {
		return fmt.Sprintf(
			"Awesome — let's dive right in! Looking at your current setup (**%s → %s**, retiring at **Age %d**), your plan currently has a **%.1f%% survival rate** through Age %d (projected median nest egg **%s** vs. **%s** needed for 90%% confidence).\n\nWould you like to:\n1. Explore **how to push your odds above 90%%** (via SIP, retirement age, or guardrails),\n2. Compare **cheaper or higher-quality retirement cities**, or\n3. Walk through **any FIRE concept** (like Coast FIRE, SWR, or pre-Age-%d liquidity)?",
			curCity.Name, retCity.Name, plan.RetirementAge, snap.SuccessRate, plan.HorizonAge,
			FormatCompact(snap.CorpusAtFireRealP50, cc), FormatCompact(snap.RequiredCorpus90, cc), CountryUnlockAge(cc),
		)
	}

	return fmt.Sprintf(
		"Building on that for your **%s → %s** plan: retiring at **Age %d** currently gives you **%.1f%% survival** (`P50` corpus **%s** vs. 90%%-safe target **%s**, or an initial withdrawal rate of **`%.2f%%`**). Your earliest 90%%-safe retirement age on your current savings trajectory is **Age %.1f** (and **Coast FIRE** is **Age %.1f**).\n\nWant me to run a **Goal-Seek on your monthly SIP**, test a **5-year Barista/consulting bridge**, or **compare cities**?",
		curCity.Name, retCity.Name, plan.RetirementAge, snap.SuccessRate,
		FormatCompact(snap.CorpusAtFireRealP50, cc), FormatCompact(snap.RequiredCorpus90, cc),
		snap.ImpliedInitialSWR, snap.TargetFIREAge, snap.CoastFIREAge,
	)
}

func applyConceptPersonaHeader(body string, mode PersonaMode) string {
	switch mode {
	case PersonaELI5:
		return "### 👶 Plain-English Breakdown\n\n" + body
	case PersonaDebate:
		return "### ⚔️ Retirement Strategist Perspective\n\n" + body
	case PersonaDevil:
		return "### 🦈 Risk-First Reality Check\n\n" + body
	default:
		return body
	}
}

func buildConceptConversationalReply(
	topic string,
	plan engine.PlanInput,
	snap engine.FullAnalysisResponse,
	curCity, retCity data.CityProfile,
) (string, []CopilotAction) {
	cc := plan.Country
	unlockAge := CountryUnlockAge(cc)
	annSpend := math.Max(1, snap.GeoArbitrage.EffectiveRetireMonthly*12)
	dynMult := snap.RequiredCorpus90 / annSpend

	switch topic {
	case "fire_overview":
		return strings.Join([]string{
				"**FIRE (Financial Independence, Retire Early)** is the point where your investment portfolio generates enough sustainable, inflation-adjusted cash flow to cover your living expenses, taxes, and healthcare — making work **optional** decades before the traditional retirement age of 60–65.",
				"",
				"In practice, reaching FIRE comes down to **four moving pieces**:",
				"1. **Your Savings Velocity:** The gap between what you earn and what you spend during your working years.",
				"2. **Your Target Lifestyle & City:** Spending in retirement (which shifts dramatically if you relocate from a high-cost hub like Mumbai, San Jose, or Zurich to Goa, Raleigh, or the Algarve).",
				fmt.Sprintf("3. **The Two-Bucket Bridge:** Having enough **accessible liquid investments** to fund life before statutory retirement accounts (`EPF/NPS`, `401(k)`, `Super`, `KiwiSaver`) unlock penalty-free at **Age %d**.", unlockAge),
				"4. **Sequence-of-Returns Defense:** Making sure a market crash or inflation wave in your first 5 years of retirement doesn't permanently deplete your nest egg.",
				"",
				fmt.Sprintf("*On your live dashboard right now (%s → %s):* your target budget is **%s/mo**, which requires **%s** (`%.1fx` annual spend) for a **90%%-safe FIRE** — putting your 90%% target age at **Age %.1f**.",
					curCity.Name, retCity.Name, FormatCompact(snap.GeoArbitrage.EffectiveRetireMonthly, cc), FormatCompact(snap.RequiredCorpus90, cc), dynMult, snap.TargetFIREAge),
			}, "\n"), []CopilotAction{
				{Label: "💬 Compare Lean, Coast, Barista & Fat FIRE", Prompt: "What is the difference between Coast FIRE, Lean FIRE, Barista FIRE, and Fat FIRE?"},
				{Label: "💬 Why does 25x / 35x break down?", Prompt: "Why does the 4% rule or 35x multiplier break down for early retirees?"},
				{Label: "🎯 Audit my current plan in detail", Prompt: "Audit my retirement plan and rank my highest-impact action levers."},
			}

	case "fire_types":
		return strings.Join([]string{
				"Think of the different flavors of **FIRE** as milestones along the same freedom spectrum rather than rigid boxes:",
				"",
				"- **🌱 Coast FIRE:** You've already invested enough today that — even if you **never save another rupee/dollar** — compounding alone will grow your nest egg to fund a full traditional retirement at Age 60–65. You only need to earn enough to cover today's bills.",
				"- **☕ Barista / Consulting FIRE:** You leave high-stress full-time work early and cover **25%–40%** of your annual expenses through part-time consulting, teaching, or a passion project for 4–7 years, dramatically slashing early portfolio withdrawals.",
				"- **🪶 Lean FIRE:** Full early retirement on a minimalist, essentials-focused budget (roughly `70%–75%` of a standard middle/upper-middle lifestyle) — fastest to reach, but leaves less room for discretionary cuts during bear markets.",
				"- **🏡 Standard / Chubby FIRE:** Comfortable upper-middle lifestyle with travel, dining, quality healthcare buffers, and zero day-to-day frugality stress.",
				"- **👑 Fat FIRE:** High-spend, luxury-buffer independence (typically `2x–3x` standard metro budgets) with low withdrawal rates (`≤3.0% SWR`) and substantial generational legacy.",
				"",
				fmt.Sprintf("*Where your live plan sits:* FireSage calculates your **Coast FIRE Age at `%.1f`**, **Lean FIRE at `%.1f`**, **90%% Target FIRE at `%.1f`**, and **97%% Fortress FIRE at `%.1f`**.",
					snap.CoastFIREAge, snap.LeanFIREAge, snap.TargetFIREAge, snap.FortressFIREAge),
			}, "\n"), []CopilotAction{
				{Label: "📅 Test a 5-Yr Barista / Consulting Bridge", Prompt: fmt.Sprintf("What if I add a 5-year consulting bridge after retiring at Age %d?", plan.RetirementAge)},
				{Label: "🌍 See how Geo-Arbitrage shifts my FIRE ages", Prompt: fmt.Sprintf("Compare retiring in %s vs top domestic and global cities", curCity.Name)},
				{Label: "💬 How do Spending Guardrails help?", Prompt: "How do Guyton-Klinger spending guardrails work in retirement?"},
			}

	case "swr_multipliers":
		return strings.Join([]string{
				"Rules of thumb like the **4% Rule (`25x` annual expenses)** or **`33x–35x` (`~2.85%–3.0%` SWR)** are handy napkin math, **but they break down for real-world early retirees** for four big reasons:",
				"",
				"1. **Horizon Length:** William Bengen's original `4%` / `25x` rule only tested a **30-year** US retirement (Age 65 → 95). Retiring at 40 or 45 means funding **45–55 years** — giving inflation and crashes decades longer to compound.",
				"2. **Healthcare Inflates Faster Than General CPI:** Headline CPI might average `3%–6%`, while private medical costs compound at `6%–13%/yr` and spike in your 70s and 80s.",
				fmt.Sprintf("3. **Taxes & Statutory Lock-Ins:** A flat `35x` ignores capital-gains taxes, ACA/Medicare cliffs, and whether half your net worth is locked until **Age %d**.", unlockAge),
				"4. **Pensions & Guardrails Cut the Other Way:** Conversely, if you have flexible spending guardrails (`Guyton-Klinger`) or an Age-65/67 pension (`Social Security`, `NZ Super`, `AU Age Pension`), `35x` can actually **overstate** what you need!",
				"",
				fmt.Sprintf("*Your personalized multiplier right now:* For your **%d-year horizon** in **%s**, FireSage solves your 90%%-safe target at **%s** — which equals **`%.1fx`** your effective annual spend (`%.2f%%` initial SWR).",
					plan.HorizonAge-plan.RetirementAge, retCity.Name, FormatCompact(snap.RequiredCorpus90, cc), dynMult, snap.ImpliedInitialSWR),
			}, "\n"), []CopilotAction{
				{Label: "💬 Explain Sequence-of-Returns Risk", Prompt: "What is sequence-of-returns risk and how does a Bond Tent protect against it?"},
				{Label: "🧮 Solve exact SIP for my 90% target", Prompt: fmt.Sprintf("How much monthly SIP do I need for 90%% survival at Age %d?", plan.RetirementAge)},
			}

	case "sorr_bond_tent":
		return strings.Join([]string{
				"**Sequence-of-Returns Risk (SORR)** is the single biggest threat to an early retiree: **even if your 30-year average return is great, getting hit with a deep bear market or stagflation wave in Years 1–5 of retirement can permanently ruin a portfolio.**",
				"",
				"Why? While you're working and saving, market crashes help you buy cheap shares. Once you retire and start withdrawing living expenses, selling shares during a `-35%` drawdown locks in permanent losses so fewer shares remain for the eventual recovery.",
				"",
				"**The 3 Proven Institutional Defenses Against SORR:**",
				"- **⛺ ERN 10-Year Bond Tent:** Temporarily step up high-quality bonds/cash (`+15%–20%`) at retirement so Years 1–5 of spending come from stable fixed income rather than depressed equities, then glide back toward equities over 10 years to beat long-run inflation.",
				"- **🛡️ Dynamic Spending Guardrails (`Guyton-Klinger`):** Trim discretionary travel/luxury spending by `10%–15%` only during deep drawdowns — which often boosts survival odds by `+10% to +18%`.",
				"- **☕ Part-Time Bridge Income:** Even modest consulting income in Years 1–4 dramatically reduces early portfolio draw.",
			}, "\n"), []CopilotAction{
				{Label: "⚡ Stress-test my plan against a -35% Year-1 crash", Prompt: "Stress test my plan against a -35% Year-1 crash and +3.5% inflation spike"},
				{Label: "💬 How do Guyton-Klinger guardrails work?", Prompt: "Explain Guyton-Klinger guardrails vs Constant Real withdrawals."},
			}

	case "withdrawal_guardrails":
		return strings.Join([]string{
				"Most classic calculators assume **Constant Real Spending** — meaning even if stocks crash `-45%`, you robotically withdraw the exact same inflation-adjusted budget every year until you go broke. Real humans don't behave like that!",
				"",
				"FireSage models three institutional withdrawal rules:",
				"- **`Constant Real` (Rigid):** Zero spending cuts ever — great stress test for ultra-essential budgets, but requires the largest starting corpus.",
				"- **`Guyton-Klinger Guardrails` (Adaptive):** If a market crash pushes your current withdrawal rate `+20%` above your initial SWR, you trim discretionary spending by `10%` (down to your floor, e.g. `80%`); in booming markets, you give yourself a `10%` raise.",
				"- **`CAPE / Valuation Dynamic`:** Scales withdrawals smoothly based on market valuation multiples (`Shiller CAPE` / `Nifty P/E`) — drawing slightly less at market peaks and more after crashes.",
				"- **😊 Blanchett Spending Smile:** Captures empirical retiree behavior where real discretionary travel/leisure tapers slowly between Ages 65–80 (`Go-Go → Slow-Go`) while late-life healthcare rises (`No-Go`).",
			}, "\n"), []CopilotAction{
				{Label: "🛡️ Enable Guyton-Klinger + Bond Tent on my plan", Prompt: "Enable Guyton-Klinger guardrails and bond tent on my plan"},
				{Label: "💬 What asset allocation works best for FIRE?", Prompt: "What is a smart asset allocation between equity, bonds, gold, and cash for FIRE?"},
			}

	case "rent_vs_buy":
		return strings.Join([]string{
				"For FIRE planners, **Rent vs. Buy** comes down to **cash-flow certainty vs. capital compounding & geo-flexibility**:",
				"",
				"- **Why Owning a Paid-Off Home Helps in Retirement:** Eliminates housing inflation risk (`35%` of a typical city budget!), lowers your annual portfolio withdrawals — which also keeps you in lower capital-gains tax brackets (and below the US `400% FPL` ACA healthcare subsidy cliff, or exempted under Australia's Centrelink homeowner Assets Test!).",
				"- **Why Renting Often Wins Early On (Especially High-Price Metros):** In cities where rental yields are `2.2%–3.0%` (like Mumbai, Bengaluru, San Jose, Zurich, Sydney, or Auckland), locking `30%–50%` of your net worth into an illiquid house drags down liquid compounding and delays Coast/Target FIRE — especially if you plan to geo-arbitrage later.",
				"- **Popular Hybrid Playbook:** Rent & compound aggressively in your high-income Tier-1 work hub, then either **rent for 1–2 years in your target retirement city first** (to test climate, hospitals, and community) before buying or downsizing.",
				"",
				fmt.Sprintf("*Tip for FireSage:* Exclude your primary home's equity from `Current Corpus` if you plan to live in it forever, and only enter your **out-of-pocket monthly budget** (excluding paid-off rent/EMI, plus maintenance & property tax). In **%s**, a typical quality home runs around **%s**.",
					retCity.Name, FormatCompact(retCity.TypicalHomePrice, retCity.Country)),
			}, "\n"), []CopilotAction{
				{Label: "📅 Simulate downsizing / selling a home in retirement", Prompt: fmt.Sprintf("What if I downsize my home at Age %d?", plan.RetirementAge+5)},
				{Label: fmt.Sprintf("🌍 Compare housing & COL across top cities vs %s", curCity.Name), Prompt: fmt.Sprintf("Compare retiring in %s vs top domestic and global cities", curCity.Name)},
			}

	case "asset_allocation":
		return strings.Join([]string{
				"A resilient **FIRE Asset Allocation** balances **long-term inflation-beating growth (Equities)** against **multi-year crash protection (Bonds, Cash & Gold)**:",
				"",
				"- **Equities (`55%–75%`):** The primary engine that outpaces headline + medical inflation over 35–50 year horizons (broad domestic + global index funds).",
				"- **Quality Debt / Govt Bonds (`20%–35%`):** Dampens volatility and funds rebalancing into cheap equities during bear markets.",
				"- **Gold (`5%–10%`):** Historically shines during stagflation, currency weakness, and sovereign real-rate shocks (like the 1970s or 2000–2011).",
				"- **Cash / T-Bills (`2%–5%`):** Covers 12–24 months of immediate living expenses so you never panic-sell.",
				"",
				fmt.Sprintf("*Your current mix:* **`%.0f%%` Equity / `%.0f%%` Debt / `%.0f%%` Gold / `%.0f%%` Cash** (ERN Bond Tent: `%v`). Want to test how shifting to `65%%` or `75%%` equity changes your survival odds?",
					plan.Allocation.Equity*100, plan.Allocation.Debt*100, plan.Allocation.Gold*100, plan.Allocation.Cash*100, plan.Allocation.UseBondTent),
			}, "\n"), []CopilotAction{
				{Label: "🧪 Test 70% Equity + Bond Tent on my plan", Prompt: "What if I set 70% equity and enable the bond tent?"},
				{Label: "⚡ Stress-test my current allocation in crises", Prompt: "Stress test my plan against historical crises and stagflation"},
			}

	case "pension_unlock_strategies":
		return strings.Join([]string{
				"When you retire early (say Age 40–52), accounts like **India's `EPF/PPF/NPS`**, **US `401(k)/Traditional IRA`**, **Australia's `Superannuation`**, **Swiss `Pillar 2/3a`**, and **NZ `KiwiSaver`** are locked behind statutory age gates (**Age 60** in IN/US/AU/EU, **Age 65** in NZ).",
				"",
				"**How Early Retirees Bridge the Gap Penalty-Free:**",
				"- **1. Dedicated Liquid Bridge (`Taxable Brokerage + Cash/Debt + Roth Contributions`):** Size your liquid SIP so your accessible bucket carries you cleanly to Age 60/65 while your locked retirement accounts compound untouched in the background for post-60 life!",
				"- **2. US-Specific Early Bridges:** A **5-Year Roth Conversion Ladder** (converting Traditional IRA → Roth IRA in low-tax brackets below the 2026 `400% FPL` ACA subsidy cliff, withdrawable penalty-free 5 years later), **IRS Rule of 55** (leaving employer at 55+), or **72(t) SEPP**.",
				"- **3. India / AU / NZ Nuances:** In India, `PPF` matures in 15-year blocks and `EPF` becomes taxable on new interest if unemployed, while `Super` (AU at 60) and `KiwiSaver` (NZ at 65) unlock **100% tax-free** — making a clean liquid bridge far superior to paying early-access penalties.",
				"",
				fmt.Sprintf("*On your live plan:* your **Pre-Age-%d Liquidity Safe rate is `%.1f%%`** vs. overall survival of **`%.1f%%`**.", unlockAge, snap.LiquiditySafeRate, snap.SuccessRate),
			}, "\n"), []CopilotAction{
				{Label: fmt.Sprintf("🌉 Audit my exact Pre-Age-%d Liquid vs Locked split", unlockAge), Prompt: fmt.Sprintf("Audit my pre-Age-%d liquid vs locked bridge and lifetime taxes", unlockAge)},
				{Label: "🧮 Solve exact Liquid SIP needed", Prompt: fmt.Sprintf("How much monthly liquid SIP do I need for 90%% survival at Age %d?", plan.RetirementAge)},
			}

	case "healthcare_inflation":
		return strings.Join([]string{
				"**Healthcare inflation is the silent budget-killer in early retirement** because medical costs compound **1.5x to 2.2x faster than general CPI** AND utilization rises steeply after Age 65:",
				"",
				"- **🇮🇳 India:** Private hospital tariffs compound at **`11%–13%/yr`**. A smart setup pairs a base **₹10–15L Family Floater** with a **₹1 Cr Super Top-Up** plus a dedicated self-insured medical buffer (especially in Tier-2/hill towns like Dehradun or Kodaikanal where tertiary cardiac/oncology care may involve metro travel).",
				"- **🇺🇸 USA:** Before **Age 65 Medicare**, early retirees buy ACA Marketplace plans — where keeping MAGI below the **restored 2026 hard `400% FPL` cliff** saves `$12k–$22k/yr` in subsidies. After 65, Medicare Part B/D + Medigap + IRMAA brackets apply.",
				"- **🇦🇺 AU / 🇪🇺 EU / 🇳🇿 NZ:** Public universal systems (`Medicare`, `SSN/SNS/PUMA`, `Health NZ + ACC`) cap catastrophic ruin, though Swiss mandatory `LAMal` premiums and NZ age-rated private surgical cover (`Southern Cross`) still climb meaningfully with age.",
			}, "\n"), []CopilotAction{
				{Label: "🏛️ Audit my plan's lifetime medical & tax drag", Prompt: fmt.Sprintf("Break down my lifetime healthcare costs and taxes in %s", retCity.Name)},
				{Label: "⚡ Test a lump-sum medical emergency at Age 72", Prompt: "Stress test my plan with a medical emergency at age 72"},
			}

	case "geo_arbitrage_concept":
		return strings.Join([]string{
				"**Geo-Arbitrage** means earning and compounding capital in a high-compensation hub (like **San Jose, NYC, Bengaluru, Mumbai, Sydney, or Zurich**) and retiring in a high-livability, lower-cost or tax-advantaged destination (**Goa, Kochi, Dehradun, Kolkata, Raleigh, Tampa, Portugal's Algarve, Southern Italy, or Tauranga**).",
				"",
				"Why it's the ultimate FIRE cheat code:",
				"- Cutting equivalent monthly housing + living costs by `25%–45%` lowers **both** your annual withdrawals AND your 90%-safe required nest egg — often shaving **4 to 8 years** off your working career without sacrificing lifestyle quality.",
				"- In **Cross-Border Geo-Arbitrage**, destination tax regimes matter enormously: e.g., **Southern Italy (`7%` flat retiree tax for 10 yrs)**, **Greece (`7%` for 15 yrs)**, **New Zealand (`4-yr` Transitional Resident `0%` FIF holiday)**, or **India's `RNOR` window** for returning NRIs.",
				"",
				"*FireSage models both domestic and any-to-any cross-border moves across all **34 cities** automatically!*",
			}, "\n"), []CopilotAction{
				{Label: fmt.Sprintf("🌍 Compare %s vs Top Domestic & Global Cities", curCity.Name), Prompt: fmt.Sprintf("Compare retiring in %s vs top domestic and global geo-arbitrage cities", curCity.Name)},
				{Label: "💬 What about healthcare in Tier-2 cities?", Prompt: "How does healthcare inflation work in Tier-2 or hill-station retirement cities?"},
			}

	case "psychology_fire":
		return strings.Join([]string{
				"The math of FIRE is often easier than the **psychology of transitioning from saver to spender**:",
				"",
				"- **\"One-More-Year\" Syndrome:** Moving the goalpost from `25x → 30x → 35x → 40x` out of fear of market crashes. Objective survival milestones (`90% Target` vs. `97% Fortress`) help you see diminishing returns clearly.",
				"- **Retire *To* Something, Not Just *From* Burnout:** The happiest early retirees test-drive their post-FIRE routine *before* quitting — building fitness habits, creative/consulting projects, community, and slower travel.",
				"- **Permission to Spend:** Switching from decades of high savings rates to drawing down a portfolio feels unnatural at first; using explicit **Spending Guardrails** gives you mathematical permission to enjoy upside years guilt-free.",
			}, "\n"), []CopilotAction{
				{Label: "📅 Test a 2-Yr Pre-FIRE Sabbatical", Prompt: "What if I take a sabbatical before retiring?"},
				{Label: "🎯 Check my Coast, Target (90%) & Fortress (97%) Ages", Prompt: "Am I on track to retire early and what are my milestone FIRE ages?"},
			}

	case "family_planning":
		return strings.Join([]string{
				"Planning **FIRE with a family** (kids' higher education, weddings, or supporting aging parents) works best when you **separate lumpy one-time milestones from recurring monthly living expenses**:",
				"",
				"- **Don't Bake College Into Forever-Spend:** If university costs `₹40L` or `$120k` over 4 years in your 40s/50s, adding it to your monthly spend would trick a simple `35x` calculator into thinking you'll pay tuition at Age 85! Instead, model it as a **bounded Life Event** in specific ages.",
				"- **Parental Medical Buffer:** Aging parents often aren't covered smoothly by standard corporate floaters post-retirement — keeping a dedicated liquid medical reserve prevents forced equity sales in a down market.",
				"- **Partner Alignment:** Agreeing on your *floor* lifestyle vs. *upside* lifestyle makes dynamic guardrails realistic as a couple.",
			}, "\n"), []CopilotAction{
				{Label: "📅 Simulate Kids' Higher Education on my plan", Prompt: fmt.Sprintf("What if we add kids college expenses at age %d?", plan.CurrentAge+12)},
				{Label: "⚡ Test a family medical shock at Age 72", Prompt: "Stress test my plan with a medical emergency at age 72"},
			}

	case "savings_velocity":
		return strings.Join([]string{
				"During accumulation, **your savings rate and annual step-up matter far more than chasing an extra `1%` investment return** — because raising your savings rate works on both ends at once (it grows your corpus faster *and* proves you thrive on a leaner share of income):",
				"",
				"- **Beat Lifestyle Creep with a `5%–10%` Step-Up SIP:** Automatically routing **half of every annual raise/bonus** into investments lets you enjoy a better lifestyle today while accelerating FIRE by years.",
				"- **Respect Liquid vs. Locked Balance:** Employer matches (`EPF`, `401(k)`, `Super`, `KiwiSaver`) are unbeatable free returns, but ensure enough savings flow into **liquid taxable/Roth accounts** if you plan to retire before Age 60/65.",
			}, "\n"), []CopilotAction{
				{Label: "🧮 Solve exact Monthly SIP for 90% confidence", Prompt: fmt.Sprintf("How much monthly SIP do I need to reach 90%% survival at Age %d?", plan.RetirementAge)},
				{Label: "🎯 Rank my highest-impact action levers", Prompt: "Audit my plan and rank my highest-impact action levers"},
			}

	case "tax_and_inflation":
		return strings.Join([]string{
				"**Taxes and inflation are the two invisible headwinds in early retirement** — ignoring either one can make a spreadsheet look safe when real purchasing power is quietly eroding:",
				"",
				"- **1. You Only Pay Capital-Gains Tax on the *Gain* Portion (Basis Tracking):** When you sell ₹12L or `$60k` from a taxable brokerage fund, not every rupee/dollar is taxable — only the embedded profit above your cost basis is taxed (e.g., India's `12.5%` equity LTCG above `₹1.25L` exemption, US `0%/15%` LTCG brackets, Australia's `50%` CGT discount, or `0%` CGT in Switzerland/New Zealand).",
				"- **2. Personal Inflation ≠ Government CPI:** Lifestyle + private healthcare inflation often runs `1.5%–3.5%` hotter than official headline CPI, which is why FireSage models general CPI and medical inflation on separate compounding curves.",
				fmt.Sprintf("- **3. Tax-Free Post-%d Buckets:** Accounts like India `EPF/PPF`, US `Roth`, AU `Super` (at 60), and NZ `KiwiSaver` (at 65) dramatically slash late-life tax drag once unlocked.", unlockAge),
			}, "\n"), []CopilotAction{
				{Label: "🏛️ Audit my plan's exact lifetime tax & medical drag", Prompt: fmt.Sprintf("Audit my pre-Age-%d liquid vs locked bridge and lifetime taxes", unlockAge)},
				{Label: "⚡ Stress-test a +3.5% inflation spike", Prompt: "Stress test my plan against a +3.5% inflation wave"},
			}

	default:
		// simulation_methodology or general fallback
		return strings.Join([]string{
				"FireSage replaces normal-distribution toy calculators with **Politis & Romano (1994) Stationary Block Bootstrap** plus **Common Random Numbers (CRN)**:",
				"",
				"- **Why Block Bootstrap Beats Standard Monte Carlo:** Normal bell-curve simulations assume every year is an independent coin toss — missing multi-year stagflation waves (like `1973–1981`) and back-to-back crashes (`2000–2002`). Sampling random-length historical blocks preserves real fat tails, inflation waves, and stock/bond/gold correlations.",
				"- **Why CRN Makes Comparisons Exact:** When comparing two cities or solving your required SIP, FireSage tests both choices against the **exact same 3,000 market histories**, isolating the true impact of your decision with zero random noise.",
			}, "\n"), []CopilotAction{
				{Label: "🎯 Run a full checkup on my current plan", Prompt: "Audit my retirement plan and highlight top action levers."},
				{Label: "🌍 Compare top retirement cities", Prompt: fmt.Sprintf("Compare retiring in %s vs top domestic and global cities", curCity.Name)},
			}
	}
}
