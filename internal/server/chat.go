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
	vertexMu          sync.Mutex
	vertexToken       string
	vertexProjectID   string
	vertexTokenExpiry time.Time
	vertexUnavailable bool

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

	// Stage 1 (Strategic Planner + Page Driver) & Stage 2 (Parallel Go Stochastic Engine Tool Calls)
	resp := copilot.ExecutePipeline(req)

	// Stage 3: Attempt Gemini / Cloud Run Vertex AI Deep Reasoner synthesis over EvidencePack
	reply, usedModel, provider, err := tryGeminiSynthesis(req, resp)
	if err == nil && strings.TrimSpace(reply) != "" {
		cleanMD, followUps, llmCmds := parseLLMDirectives(reply)
		if cleanMD != "" {
			resp.Markdown = cleanMD
			resp.UsedLLM = true
			resp.EngineLabel = fmt.Sprintf("🧠 %s (%s · Planner + %d Tools)", usedModel, provider, len(resp.ToolTrace))
			if len(llmCmds) > 0 {
				resp.UICommands = mergeUICommands(resp.UICommands, llmCmds)
			}
			if len(followUps) > 0 {
				resp.Actions = mergeActionsWithFollowUps(resp.Actions, followUps)
			}
		}
	} else if err != nil && strings.TrimSpace(req.APIKey) != "" {
		resp.LLMNote = "Gemini API fallback to deterministic Go Quant Engine: " + err.Error()
	}

	writeJSON(w, http.StatusOK, resp)
}

func getGCPVertexAuth() (projectID, accessToken string) {
	if runtime.GOOS == "js" {
		return "", ""
	}
	vertexMu.Lock()
	defer vertexMu.Unlock()

	if vertexUnavailable {
		return "", ""
	}
	if vertexToken != "" && vertexProjectID != "" && time.Now().Before(vertexTokenExpiry) {
		return vertexProjectID, vertexToken
	}

	proj := strings.TrimSpace(os.Getenv("GOOGLE_CLOUD_PROJECT"))
	if proj == "" {
		proj = strings.TrimSpace(os.Getenv("GCP_PROJECT"))
	}

	client := &http.Client{Timeout: 900 * time.Millisecond}
	if proj == "" {
		reqP, _ := http.NewRequest(http.MethodGet, "http://metadata.google.internal/computeMetadata/v1/project/project-id", nil)
		reqP.Header.Set("Metadata-Flavor", "Google")
		resP, err := client.Do(reqP)
		if err != nil {
			vertexUnavailable = true
			return "", ""
		}
		defer resP.Body.Close()
		if resP.StatusCode != http.StatusOK {
			vertexUnavailable = true
			return "", ""
		}
		b, _ := io.ReadAll(io.LimitReader(resP.Body, 4096))
		proj = strings.TrimSpace(string(b))
	}

	reqT, _ := http.NewRequest(http.MethodGet, "http://metadata.google.internal/computeMetadata/v1/instance/service-accounts/default/token", nil)
	reqT.Header.Set("Metadata-Flavor", "Google")
	resT, err := client.Do(reqT)
	if err != nil {
		vertexUnavailable = true
		return "", ""
	}
	defer resT.Body.Close()
	if resT.StatusCode != http.StatusOK {
		vertexUnavailable = true
		return "", ""
	}

	var tokData struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.NewDecoder(resT.Body).Decode(&tokData); err != nil || tokData.AccessToken == "" {
		vertexUnavailable = true
		return "", ""
	}

	ttl := tokData.ExpiresIn - 60
	if ttl < 60 {
		ttl = 300
	}
	vertexProjectID = proj
	vertexToken = tokData.AccessToken
	vertexTokenExpiry = time.Now().Add(time.Duration(ttl) * time.Second)
	return vertexProjectID, vertexToken
}

func buildStage3SystemPrompt(mode string) string {
	personaDirective := "PERSONA: Institutional Actuarial Strategist & Global Tax Architect. Lead with a crisp verdict, present a clean Markdown KPI table where helpful, cite exact percentiles (P10/P50/P90), pre-unlock liquidity safety, and statutory tax/medical mechanics, and end with concrete next levers."
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "eli5":
		personaDirective = "PERSONA: Plain-English Retirement Coach (ELI5). Explain every financial metric using vivid everyday metaphors (like a two-tank water reservoir for Liquid vs. Locked accounts and weather storms for sequence-of-returns crashes) while still quoting the user's exact numbers accurately."
	case "debate":
		personaDirective = "PERSONA: Institutional Retirement Committee Debate. Structure the response explicitly into '### 🟢 The Bull Case', '### 🔴 The Bear Case', and '### ⚖️ Committee Verdict', grounding both sides strictly in the executed quantitative tool outputs."
	case "devil":
		personaDirective = "PERSONA: Adversarial Chief Risk Officer (Pre-Mortem). Stress-test and challenge the user's retirement assumptions — highlight 10th-percentile sequence-of-returns risk, pre-unlock liquidity crunches, healthcare inflation compounding, and tax cliffs, then state what fortification is required."
	}

	return strings.Join([]string{
		"You are FIRE Copilot — FireSage's Stage 3 Deliberative Quantitative Retirement & Global Geo-Arbitrage Strategist.",
		personaDirective,
		"GROUNDING RULES (MANDATORY):",
		"1. Every single currency figure, survival percentage, SWR, tax rule, city comparison, and FIRE age MUST come strictly from the [EXECUTED GO QUANTITATIVE TOOL EVIDENCE PACK] provided in the prompt. Never invent or approximate numbers.",
		"2. Format currency naturally for the active work jurisdiction (₹ Cr / ₹ L for IN, $ for US, A$ for AU, € for EU, NZ$ for NZ).",
		"3. Keep formatting crisp and scannable (220–380 words max) using GitHub-flavored Markdown headings (### / ####), bullet points, bold callouts, and compact Markdown tables.",
		"4. At the very end of your response, append ONE line starting with 'FOLLOW_UPS: ' containing 3 short, high-value follow-up questions separated by ' | '.",
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
	for _, fallback := range []string{"gemini-2.5-flash", "gemini-2.5-pro", "gemini-2.0-flash"} {
		if fallback != pref {
			models = append(models, fallback)
		}
	}

	var histBuf strings.Builder
	if len(req.History) > 0 {
		start := 0
		if len(req.History) > 6 {
			start = len(req.History) - 6
		}
		histBuf.WriteString("[RECENT CONVERSATION HISTORY]\n")
		for _, h := range req.History[start:] {
			role := "User"
			if h.Role == "assistant" || h.Role == "bot" {
				role = "FIRE Copilot"
			}
			txt := h.Text
			if len(txt) > 500 {
				txt = txt[:500] + "..."
			}
			histBuf.WriteString(fmt.Sprintf("%s: %s\n", role, txt))
		}
		histBuf.WriteString("\n")
	}

	userPrompt := fmt.Sprintf(
		"%s[STAGE 1 PLANNER HYPOTHESES]\n- %s\n\n[EXECUTED GO QUANTITATIVE TOOL EVIDENCE PACK]\n%s\n\n[ALREADY DISPATCHED UI PAGE-DRIVER COMMANDS]: %d\n\n[USER QUESTION]: %s",
		histBuf.String(),
		strings.Join(base.Plan.Hypotheses, "\n- "),
		base.EvidencePack,
		len(base.UICommands),
		req.Question,
	)

	payload := map[string]any{
		"systemInstruction": map[string]any{
			"parts": []map[string]string{{"text": buildStage3SystemPrompt(req.Mode)}},
		},
		"contents": []map[string]any{
			{
				"role":  "user",
				"parts": []map[string]string{{"text": userPrompt}},
			},
		},
		"generationConfig": map[string]any{
			"temperature":     0.25,
			"maxOutputTokens": 2048,
		},
	}
	bodyBytes, _ := json.Marshal(payload)
	httpClient := &http.Client{Timeout: 24 * time.Second}

	// Route A: Explicit or Server-Env Gemini Developer API Key
	if apiKey != "" {
		var lastErr error
		for _, m := range models {
			endpoint := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", m, apiKey)
			text, callErr := invokeGeminiEndpoint(httpClient, endpoint, "", bodyBytes)
			if callErr == nil && text != "" {
				return text, m, "Gemini API", nil
			}
			lastErr = callErr
		}
		if lastErr != nil {
			err = lastErr
		}
	}

	// Route B: Zero-Key Google Cloud Run Vertex AI via Service Account Metadata Token
	projID, accessToken := getGCPVertexAuth()
	if projID != "" && accessToken != "" {
		locations := []string{"global", "us-central1", "asia-south1"}
		for _, m := range models {
			for _, loc := range locations {
				host := fmt.Sprintf("%s-aiplatform.googleapis.com", loc)
				if loc == "global" {
					host = "aiplatform.googleapis.com"
				}
				endpoint := fmt.Sprintf("https://%s/v1/projects/%s/locations/%s/publishers/google/models/%s:generateContent", host, projID, loc, m)
				text, callErr := invokeGeminiEndpoint(httpClient, endpoint, accessToken, bodyBytes)
				if callErr == nil && text != "" {
					return text, m, "Vertex AI", nil
				}
			}
		}
	}

	return "", "", "", err
}

func invokeGeminiEndpoint(client *http.Client, url, bearerToken string, bodyBytes []byte) (string, error) {
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	if bearerToken != "" {
		req.Header.Set("Authorization", "Bearer "+bearerToken)
	}

	res, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	respBytes, _ := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return "", fmt.Errorf("HTTP %d", res.StatusCode)
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
		return "", err
	}
	if len(parsed.Candidates) == 0 {
		return "", fmt.Errorf("empty candidates")
	}

	var out strings.Builder
	for _, p := range parsed.Candidates[0].Content.Parts {
		if !p.Thought && p.Text != "" {
			out.WriteString(p.Text)
		}
	}
	return strings.TrimSpace(out.String()), nil
}

func parseLLMDirectives(raw string) (cleanMD string, followUps []string, uiCmds []copilot.UICommand) {
	clean := raw
	if m := reFollowUps.FindStringSubmatch(clean); len(m) >= 2 {
		parts := strings.Split(m[1], "|")
		for _, p := range parts {
			trimmed := strings.Trim(strings.TrimSpace(p), `"'*- `)
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
