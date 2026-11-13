package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/ausanyal/firesage/internal/copilot"
)

var (
	vertexMu            sync.Mutex
	vertexToken         string
	vertexProjectID     string
	vertexTokenExpiry   time.Time
	vertexCooldownUntil time.Time

	reFollowUps  = regexp.MustCompile(`(?im)^FOLLOW_UPS:\s*(.+)$`)
	reUICmdsJSON = regexp.MustCompile(`(?im)^UI_COMMANDS_JSON:\s*(\[.*\])\s*$`)
)

func handleChat(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1 MB max body
	var req copilot.ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"ok":    false,
			"error": "invalid JSON request payload: " + err.Error(),
		})
		return
	}

	// Stage 1 (Intent/Planner + Page Driver) & Stage 2 (Conversational or Parallel Go Stochastic Tools)
	resp := copilot.ExecutePipeline(req)

	// Stage 3: Attempt Gemini / Cloud Run Vertex AI conversational synthesis
	reply, usedModel, provider, err := tryGeminiSynthesis(req, resp)
	if err == nil && strings.TrimSpace(reply) != "" {
		cleanMD, followUps, llmCmds := parseLLMDirectives(reply)
		if cleanMD != "" {
			resp.Markdown = cleanMD
			resp.UsedLLM = true
			if len(resp.ToolTrace) > 0 {
				resp.EngineLabel = fmt.Sprintf("🧠 %s (%s · Planner + %d Tools)", usedModel, provider, len(resp.ToolTrace))
			} else {
				resp.EngineLabel = fmt.Sprintf("🧠 %s (%s · Conversational Advisor)", usedModel, provider)
			}
			if len(llmCmds) > 0 {
				resp.UICommands = mergeUICommands(resp.UICommands, llmCmds)
			}
			if len(followUps) > 0 {
				resp.Actions = mergeActionsWithFollowUps(resp.Actions, followUps)
			}
		}
	} else if err != nil && strings.TrimSpace(req.APIKey) != "" {
		resp.LLMNote = "Gemini API key error (answered via built-in FIRE Advisor): " + err.Error()
	}

	writeJSON(w, http.StatusOK, resp)
}

func getGCPVertexAuth() (projectID, accessToken string) {
	if runtime.GOOS == "js" {
		return "", ""
	}

	vertexMu.Lock()
	defer vertexMu.Unlock()

	now := time.Now()
	if now.Before(vertexCooldownUntil) {
		return "", ""
	}
	if vertexToken != "" && vertexProjectID != "" && now.Before(vertexTokenExpiry) {
		return vertexProjectID, vertexToken
	}

	proj := strings.TrimSpace(os.Getenv("GOOGLE_CLOUD_PROJECT"))
	if proj == "" {
		proj = strings.TrimSpace(os.Getenv("GCLOUD_PROJECT"))
	}
	if proj == "" {
		proj = strings.TrimSpace(os.Getenv("GCP_PROJECT"))
	}

	// Fast-skip metadata lookup on local laptops where neither Cloud Run K_SERVICE nor GCP project env is set
	isCloudRun := os.Getenv("K_SERVICE") != "" || proj != ""
	timeout := 400 * time.Millisecond
	if isCloudRun {
		timeout = 1800 * time.Millisecond
	}

	client := &http.Client{Timeout: timeout}
	if proj == "" {
		reqP, _ := http.NewRequest(http.MethodGet, "http://metadata.google.internal/computeMetadata/v1/project/project-id", nil)
		reqP.Header.Set("Metadata-Flavor", "Google")
		resP, err := client.Do(reqP)
		if err != nil {
			if !isCloudRun {
				vertexCooldownUntil = now.Add(60 * time.Second)
			}
			return "", ""
		}
		defer resP.Body.Close()
		if resP.StatusCode != http.StatusOK {
			if !isCloudRun {
				vertexCooldownUntil = now.Add(60 * time.Second)
			}
			return "", ""
		}
		b, _ := io.ReadAll(io.LimitReader(resP.Body, 4096))
		proj = strings.TrimSpace(string(b))
	}

	reqT, _ := http.NewRequest(http.MethodGet, "http://metadata.google.internal/computeMetadata/v1/instance/service-accounts/default/token", nil)
	reqT.Header.Set("Metadata-Flavor", "Google")
	resT, err := client.Do(reqT)
	if err != nil {
		if !isCloudRun {
			vertexCooldownUntil = now.Add(60 * time.Second)
		}
		return "", ""
	}
	defer resT.Body.Close()
	if resT.StatusCode != http.StatusOK {
		return "", ""
	}

	var tokData struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.NewDecoder(resT.Body).Decode(&tokData); err != nil || tokData.AccessToken == "" {
		return "", ""
	}

	ttl := tokData.ExpiresIn - 60
	if ttl < 60 {
		ttl = 300
	}
	vertexProjectID = proj
	vertexToken = tokData.AccessToken
	vertexTokenExpiry = now.Add(time.Duration(ttl) * time.Second)
	return vertexProjectID, vertexToken
}

func buildStage3SystemPrompt(mode string, hasQuantTools bool) string {
	personaDirective := "PERSONA: Warm, articulate, conversational FIRE & Global Retirement Strategist. Speak naturally in first person like a trusted human advisor sitting across the table."
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "eli5":
		personaDirective = "PERSONA: Friendly Plain-English Retirement Coach (ELI5). Explain ideas conversationally using vivid everyday analogies (like a two-tank water reservoir for Liquid vs. Locked accounts and weather storms for early market crashes), keeping jargon out."
	case "debate":
		personaDirective = "PERSONA: Conversational Retirement Committee Debate. If analyzing a plan or trade-off, contrast '### 🟢 The Bull Case', '### 🔴 The Bear Case', and '### ⚖️ Verdict'; if just greeting or chatting casually, respond naturally without rigid headers."
	case "devil":
		personaDirective = "PERSONA: Candid Chief Risk Officer & Devil's Advocate. Be conversational and engaging, while stress-testing assumptions around sequence-of-returns risk, pre-unlock liquidity, medical inflation, and tax drag."
	}

	depthRule := "2. ADAPTIVE CONVERSATIONAL STYLE (CRITICAL): This turn is a conversational, greeting, capability, or general FIRE concept question. Answer directly, warmly, and naturally in conversational prose (80–220 words). Do NOT dump unsolicited KPI tables or robotic audit headers! Only weave in 1 brief personalized reference from [EXECUTED EVIDENCE / SNAPSHOT] if it genuinely illustrates the concept."
	if hasQuantTools {
		depthRule = "2. CONVERSATIONAL + DATA-BACKED SYNTHESIS: Open with a natural, conversational 1–2 sentence answer directly addressing what the user asked, then present crisp Markdown bullets or a compact table grounded strictly in [EXECUTED EVIDENCE / SNAPSHOT] (180–340 words max)."
	}

	return strings.Join([]string{
		"You are FIRE Copilot — FireSage's conversational AI retirement, tax, investing, and global geo-arbitrage strategist.",
		personaDirective,
		"CORE RULES:",
		"1. Answer ANY natural-language question naturally — including greetings ('can I ask you FIRE questions?'), general FIRE concepts (Coast/Lean/Barista/Fat FIRE, 4% rule vs 35x, sequence risk, rent vs buy, asset allocation, Roth ladders, EPF/NPS/Super/KiwiSaver, healthcare inflation, psychology of retirement), follow-up questions, AND live plan simulations.",
		depthRule,
		"3. NUMERICAL GROUNDING: Whenever citing numbers from the user's plan, use ONLY exact figures from [EXECUTED EVIDENCE / SNAPSHOT] in native currency (₹ Cr / ₹ L for IN, $ for US, A$ for AU, € for EU, NZ$ for NZ). Never invent numbers.",
		"4. At the very end of your response, append ONE line starting with 'FOLLOW_UPS: ' containing 3 natural, conversational follow-up questions separated by ' | '.",
		"5. Optional Page-Driver Navigation: ONLY if the user explicitly asked to switch tabs, open autopsy, toggle real/nominal, or focus a chart section and it wasn't already dispatched, you may append 'UI_COMMANDS_JSON: [{\"cmd\":\"OPEN_LAB_TAB\",\"tab\":\"crises\",\"reason\":\"...\"}]' on the final line.",
	}, "\n")
}

func tryGeminiSynthesis(req copilot.ChatRequest, base copilot.ChatResponse) (reply, usedModel, provider string, err error) {
	apiKey := strings.TrimSpace(req.APIKey)
	if apiKey == "" {
		apiKey = strings.TrimSpace(os.Getenv("GEMINI_API_KEY"))
	}

	pref := strings.TrimSpace(req.Model)
	if pref == "" {
		pref = "gemini-2.5-flash"
	}
	models := []string{pref}
	for _, fallback := range []string{"gemini-2.5-flash", "gemini-2.5-pro", "gemini-2.0-flash-001", "gemini-2.0-flash"} {
		if fallback != pref {
			models = append(models, fallback)
		}
	}

	var histBuf strings.Builder
	if len(req.History) > 0 {
		start := 0
		if len(req.History) > 8 {
			start = len(req.History) - 8
		}
		histBuf.WriteString("=== RECENT CONVERSATION HISTORY ===\n")
		for _, h := range req.History[start:] {
			role := "USER"
			if h.Role == "assistant" || h.Role == "bot" {
				role = "FIRE COPILOT"
			}
			txt := strings.TrimSpace(h.Text)
			if len(txt) > 500 {
				txt = txt[:500] + "..."
			}
			histBuf.WriteString(fmt.Sprintf("%s: %s\n", role, txt))
		}
		histBuf.WriteString("\n")
	}

	hasTools := len(base.ToolTrace) > 0
	sysPrompt := buildStage3SystemPrompt(req.Mode, hasTools)

	hypBlock := ""
	if len(base.Plan.Hypotheses) > 0 {
		hypBlock = "=== STAGE 1 PLANNER HYPOTHESES ===\n- " + strings.Join(base.Plan.Hypotheses, "\n- ") + "\n\n"
	}

	// Embed system prompt directly inside contents[0] ("role": "user") — matching StockSage server.py
	// for 100% compatibility across both Vertex AI v1 and Gemini Developer API v1beta.
	combinedPrompt := fmt.Sprintf(
		"%s\n\n%s%s=== EXECUTED EVIDENCE / SNAPSHOT ===\n%s\n\n=== ALREADY DISPATCHED UI PAGE-DRIVER COMMANDS ===\n%d\n\n=== USER MESSAGE ===\n%s",
		sysPrompt,
		histBuf.String(),
		hypBlock,
		base.EvidencePack,
		len(base.UICommands),
		req.Question,
	)

	payload := map[string]any{
		"contents": []map[string]any{
			{
				"role":  "user",
				"parts": []map[string]string{{"text": combinedPrompt}},
			},
		},
		"generationConfig": map[string]any{
			"temperature":     0.35,
			"maxOutputTokens": 2048,
		},
	}
	bodyBytes, _ := json.Marshal(payload)
	httpClient := &http.Client{Timeout: 28 * time.Second}

	// Route A: Explicit or Server-Env Gemini Developer API Key
	if apiKey != "" {
		var lastErr error
		for _, m := range models {
			endpoint := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", m, apiKey)
			text, status, errBody, callErr := invokeGeminiEndpoint(httpClient, endpoint, "", bodyBytes)
			if callErr == nil && text != "" {
				return text, m, "Gemini API", nil
			}
			lastErr = callErr
			if (status == 400 || status == 401 || status == 403) && strings.Contains(errBody, "API_KEY_INVALID") {
				break
			}
		}
		if lastErr != nil {
			err = lastErr
		}
	}

	// Route B: Zero-Key Google Cloud Run Vertex AI via Service Account Metadata Token
	projID, accessToken := getGCPVertexAuth()
	if projID != "" && accessToken != "" {
		locations := []string{"global", "us-central1", "asia-south1"}
		abortVertex := false
		for _, loc := range locations {
			if abortVertex {
				break
			}
			host := fmt.Sprintf("%s-aiplatform.googleapis.com", loc)
			if loc == "global" {
				host = "aiplatform.googleapis.com"
			}
			for _, m := range models {
				endpoint := fmt.Sprintf("https://%s/v1/projects/%s/locations/%s/publishers/google/models/%s:generateContent", host, projID, loc, m)
				text, status, errBody, callErr := invokeGeminiEndpoint(httpClient, endpoint, accessToken, bodyBytes)
				if callErr == nil && text != "" {
					return text, m, fmt.Sprintf("Vertex AI · %s", loc), nil
				}
				if strings.Contains(errBody, "SERVICE_DISABLED") || strings.Contains(errBody, "has not been used in project") || (status == 403 && strings.Contains(errBody, "PERMISSION_DENIED")) {
					vertexMu.Lock()
					vertexCooldownUntil = time.Now().Add(5 * time.Minute)
					vertexMu.Unlock()
					abortVertex = true
					break
				}
			}
		}
	}

	return "", "", "", err
}

func invokeGeminiEndpoint(client *http.Client, url, bearerToken string, bodyBytes []byte) (text string, statusCode int, errSnippet string, err error) {
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", 0, "", err
	}
	req.Header.Set("Content-Type", "application/json")
	if bearerToken != "" {
		req.Header.Set("Authorization", "Bearer "+bearerToken)
	}

	res, err := client.Do(req)
	if err != nil {
		return "", 0, "", err
	}
	defer res.Body.Close()

	respBytes, _ := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		snip := string(respBytes)
		if len(snip) > 260 {
			snip = snip[:260]
		}
		return "", res.StatusCode, snip, fmt.Errorf("HTTP %d", res.StatusCode)
	}

	var parsed struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text    string `json:"text"`
					Thought bool   `json:"thought"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	if err := json.Unmarshal(respBytes, &parsed); err != nil {
		return "", res.StatusCode, "", err
	}
	if len(parsed.Candidates) == 0 {
		return "", res.StatusCode, "", fmt.Errorf("empty candidates")
	}

	var out strings.Builder
	for _, p := range parsed.Candidates[0].Content.Parts {
		if !p.Thought && p.Text != "" {
			out.WriteString(p.Text)
		}
	}
	if out.Len() == 0 {
		for _, p := range parsed.Candidates[0].Content.Parts {
			if p.Text != "" {
				out.WriteString(p.Text)
			}
		}
	}
	return strings.TrimSpace(out.String()), res.StatusCode, "", nil
}

func parseLLMDirectives(raw string) (cleanMD string, followUps []string, uiCmds []copilot.UICommand) {
	clean := raw
	if m := reFollowUps.FindStringSubmatch(clean); len(m) >= 2 {
		parts := strings.Split(m[1], "|")
		for _, p := range parts {
			trimmed := strings.Trim(strings.TrimSpace(p), "`\"'*-• ")
			if trimmed != "" && len(followUps) < 3 {
				followUps = append(followUps, trimmed)
			}
		}
		clean = reFollowUps.ReplaceAllString(clean, "")
	}

	if m := reUICmdsJSON.FindStringSubmatch(clean); len(m) >= 2 {
		var parsed []copilot.UICommand
		if err := json.Unmarshal([]byte(m[1]), &parsed); err == nil {
			for _, c := range parsed {
				switch c.Cmd {
				case "SET_VALUE_BASIS", "OPEN_AUTOPSY", "OPEN_LAB_TAB", "SCROLL_TO_SECTION", "OPEN_METHODOLOGY":
					uiCmds = append(uiCmds, c)
				}
			}
		}
		clean = reUICmdsJSON.ReplaceAllString(clean, "")
	}

	return strings.TrimSpace(clean), followUps, uiCmds
}

func mergeUICommands(existing, extra []copilot.UICommand) []copilot.UICommand {
	out := append([]copilot.UICommand{}, existing...)
	seen := map[string]bool{}
	for _, c := range existing {
		seen[c.Cmd+":"+c.Tab+":"+c.Section+":"+c.Basis] = true
	}
	for _, c := range extra {
		key := c.Cmd + ":" + c.Tab + ":" + c.Section + ":" + c.Basis
		if !seen[key] {
			seen[key] = true
			out = append(out, c)
		}
	}
	return out
}

func mergeActionsWithFollowUps(existing []copilot.CopilotAction, followUps []string) []copilot.CopilotAction {
	var uiMutations []copilot.CopilotAction
	for _, a := range existing {
		if a.UICommand != nil {
			uiMutations = append(uiMutations, a)
		}
	}
	out := append([]copilot.CopilotAction{}, uiMutations...)
	for _, q := range followUps {
		if len(out) >= 5 {
			break
		}
		out = append(out, copilot.CopilotAction{
			Label:  "💬 " + q,
			Prompt: q,
		})
	}
	return out
}
