package advstats

import (
	"math"
	"sort"
	"strings"
	"time"
)

type RequestLog struct {
	Timestamp       string `json:"timestamp"`
	ProviderID      string `json:"provider_id,omitempty"`
	ModelID         string `json:"model_id"`
	Path            string `json:"path,omitempty"`
	InputTokens     int    `json:"input_tokens"`
	OutputTokens    int    `json:"output_tokens"`
	ReasoningTokens int    `json:"reasoning_tokens"`
	CachedTokens    int    `json:"cached_tokens"`
}

type AggregateRequest struct {
	Start         string       `json:"start"`
	End           string       `json:"end"`
	BucketSeconds int          `json:"bucket_seconds"`
	WindowLogs    []RequestLog `json:"window_logs"`
	TodayLogs     []RequestLog `json:"today_logs"`
}

type Summary struct {
	Requests        int     `json:"requests"`
	InputTokens     int     `json:"input_tokens"`
	OutputTokens    int     `json:"output_tokens"`
	ReasoningTokens int     `json:"reasoning_tokens"`
	CachedTokens    int     `json:"cached_tokens"`
	StandardTokens  int     `json:"standard_tokens"`
	TotalTokens     int     `json:"total_tokens"`
	EstimatedUSD    float64 `json:"estimated_usd"`
}

type ModelUsage struct {
	ModelID         string  `json:"model_id"`
	Requests        int     `json:"requests"`
	InputTokens     int     `json:"input_tokens"`
	OutputTokens    int     `json:"output_tokens"`
	ReasoningTokens int     `json:"reasoning_tokens"`
	CachedTokens    int     `json:"cached_tokens"`
	StandardTokens  int     `json:"standard_tokens"`
	TotalTokens     int     `json:"total_tokens"`
	EstimatedUSD    float64 `json:"estimated_usd"`
	Share           float64 `json:"share"`
}

type ProviderUsage struct {
	ProviderID      string  `json:"provider_id"`
	Requests        int     `json:"requests"`
	InputTokens     int     `json:"input_tokens"`
	OutputTokens    int     `json:"output_tokens"`
	ReasoningTokens int     `json:"reasoning_tokens"`
	CachedTokens    int     `json:"cached_tokens"`
	StandardTokens  int     `json:"standard_tokens"`
	TotalTokens     int     `json:"total_tokens"`
	EstimatedUSD    float64 `json:"estimated_usd"`
	Share           float64 `json:"share"`
}

type APIUsage struct {
	BackendID       string  `json:"backend_id"`
	BackendURL      string  `json:"backend_url"`
	Requests        int     `json:"requests"`
	InputTokens     int     `json:"input_tokens"`
	OutputTokens    int     `json:"output_tokens"`
	ReasoningTokens int     `json:"reasoning_tokens"`
	CachedTokens    int     `json:"cached_tokens"`
	StandardTokens  int     `json:"standard_tokens"`
	TotalTokens     int     `json:"total_tokens"`
	EstimatedUSD    float64 `json:"estimated_usd"`
	Share           float64 `json:"share"`
}

type SpecialBackendUsage struct {
	ID            string          `json:"id"`
	Name          string          `json:"name"`
	BackendURL    string          `json:"backend_url"`
	Summary       Summary         `json:"summary"`
	ModelUsage    []ModelUsage    `json:"model_usage"`
	ProviderUsage []ProviderUsage `json:"provider_usage"`
}

type TrendPoint struct {
	BucketStart     string  `json:"bucket_start"`
	Requests        int     `json:"requests"`
	InputTokens     int     `json:"input_tokens"`
	OutputTokens    int     `json:"output_tokens"`
	ReasoningTokens int     `json:"reasoning_tokens"`
	CachedTokens    int     `json:"cached_tokens"`
	StandardTokens  int     `json:"standard_tokens"`
	TotalTokens     int     `json:"total_tokens"`
	EstimatedUSD    float64 `json:"estimated_usd"`
}

type AggregateResponse struct {
	OK             bool    `json:"ok"`
	Start          string  `json:"start"`
	End            string  `json:"end"`
	Now            string  `json:"now"`
	BucketSeconds  int     `json:"bucket_seconds"`
	Today          Summary `json:"today"`
	Window         Summary `json:"window"`
	TokenBreakdown struct {
		StandardTokens  int `json:"standard_tokens"`
		ReasoningTokens int `json:"reasoning_tokens"`
		CachedTokens    int `json:"cached_tokens"`
		TotalTokens     int `json:"total_tokens"`
	} `json:"token_breakdown"`
	ModelUsage      []ModelUsage          `json:"model_usage"`
	ProviderUsage   []ProviderUsage       `json:"provider_usage"`
	APIUsage        []APIUsage            `json:"api_usage"`
	SpecialBackends []SpecialBackendUsage `json:"special_backends"`
	Trend           []TrendPoint          `json:"trend"`
}

type priceRule struct {
	Prefix       string
	InputUSD     float64
	OutputUSD    float64
	CachedUSD    float64
	ReasoningUSD float64
}

var priceRules = []priceRule{
	{Prefix: "gpt-5", InputUSD: 1.25, OutputUSD: 10.0, CachedUSD: 0.125, ReasoningUSD: 10.0},
	{Prefix: "gpt-4.1", InputUSD: 2.0, OutputUSD: 8.0, CachedUSD: 0.5, ReasoningUSD: 8.0},
	{Prefix: "gpt-4o-mini", InputUSD: 0.15, OutputUSD: 0.6, CachedUSD: 0.075, ReasoningUSD: 0.6},
	{Prefix: "gpt-4o", InputUSD: 5.0, OutputUSD: 15.0, CachedUSD: 2.5, ReasoningUSD: 15.0},
	{Prefix: "o1", InputUSD: 15.0, OutputUSD: 60.0, CachedUSD: 7.5, ReasoningUSD: 60.0},
	{Prefix: "o3", InputUSD: 2.0, OutputUSD: 8.0, CachedUSD: 0.5, ReasoningUSD: 8.0},
	{Prefix: "claude-3", InputUSD: 3.0, OutputUSD: 15.0, CachedUSD: 1.5, ReasoningUSD: 15.0},
	{Prefix: "gemini-1.5-pro", InputUSD: 3.5, OutputUSD: 10.5, CachedUSD: 0.9, ReasoningUSD: 10.5},
	{Prefix: "gemini-1.5-flash", InputUSD: 0.35, OutputUSD: 1.05, CachedUSD: 0.09, ReasoningUSD: 1.05},
}

var fallbackRule = priceRule{InputUSD: 0.3, OutputUSD: 1.2, CachedUSD: 0.08, ReasoningUSD: 1.2}

type specialBackendDef struct {
	ID      string
	Name    string
	Aliases []string
}

var specialBackendDefs = []specialBackendDef{
	{ID: "claude-code", Name: "Claude Code", Aliases: []string{"claude-code", "claude_code", "claudecode"}},
	{ID: "codex", Name: "Codex", Aliases: []string{"codex"}},
	{ID: "cherry-studio", Name: "Cherry Studio", Aliases: []string{"cherry-studio", "cherry_studio", "cherrystudio"}},
}

func Aggregate(req AggregateRequest, now time.Time) AggregateResponse {
	if now.IsZero() {
		now = time.Now().UTC()
	} else {
		now = now.UTC()
	}
	start, ok := ParseRFC3339(req.Start)
	if !ok {
		start = now.Add(-7 * 24 * time.Hour)
	}
	end, ok := ParseRFC3339(req.End)
	if !ok || !end.After(start) {
		end = now
	}

	bucketSeconds := req.BucketSeconds
	if bucketSeconds <= 0 {
		bucketSeconds = 300
	}
	if bucketSeconds > 24*3600 {
		bucketSeconds = 24 * 3600
	}

	windowSummary := summarize(req.WindowLogs)
	todaySummary := summarize(req.TodayLogs)
	models := aggregateModels(req.WindowLogs)
	providers := aggregateProviders(req.WindowLogs)
	apis := aggregateAPIs(req.WindowLogs)
	specialBackends := buildSpecialBackends(req.WindowLogs)
	applyModelShare(models, windowSummary.TotalTokens)
	applyProviderShare(providers, windowSummary.TotalTokens)
	applyAPIShare(apis, windowSummary.TotalTokens)
	trend := buildTrend(req.WindowLogs, start, end, bucketSeconds)

	out := AggregateResponse{
		OK:              true,
		Start:           start.Format(time.RFC3339),
		End:             end.Format(time.RFC3339),
		Now:             now.Format(time.RFC3339),
		BucketSeconds:   bucketSeconds,
		Today:           todaySummary,
		Window:          windowSummary,
		ModelUsage:      models,
		ProviderUsage:   providers,
		APIUsage:        apis,
		SpecialBackends: specialBackends,
		Trend:           trend,
	}
	out.TokenBreakdown.StandardTokens = windowSummary.StandardTokens
	out.TokenBreakdown.ReasoningTokens = windowSummary.ReasoningTokens
	out.TokenBreakdown.CachedTokens = windowSummary.CachedTokens
	out.TokenBreakdown.TotalTokens = windowSummary.TotalTokens
	return out
}

func ParseRFC3339(raw string) (time.Time, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, false
	}
	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return time.Time{}, false
	}
	return t.UTC(), true
}

func summarize(logs []RequestLog) Summary {
	out := Summary{}
	for _, row := range logs {
		inTok := clampNonNegative(row.InputTokens)
		outTok := clampNonNegative(row.OutputTokens)
		reasoning := clampNonNegative(row.ReasoningTokens)
		cached := clampNonNegative(row.CachedTokens)
		if reasoning > outTok {
			reasoning = outTok
		}
		if cached > inTok {
			cached = inTok
		}

		out.Requests++
		out.InputTokens += inTok
		out.OutputTokens += outTok
		out.ReasoningTokens += reasoning
		out.CachedTokens += cached
		out.EstimatedUSD += estimateCostUSD(row.ModelID, inTok, outTok, reasoning, cached)
	}
	out.TotalTokens = out.InputTokens + out.OutputTokens
	out.StandardTokens = clampNonNegative(out.TotalTokens - out.ReasoningTokens - out.CachedTokens)
	out.EstimatedUSD = roundUSD(out.EstimatedUSD)
	return out
}

func aggregateModels(logs []RequestLog) []ModelUsage {
	byModel := make(map[string]*ModelUsage)
	for _, row := range logs {
		model := strings.TrimSpace(row.ModelID)
		if model == "" {
			model = "-"
		}
		item := byModel[model]
		if item == nil {
			item = &ModelUsage{ModelID: model}
			byModel[model] = item
		}
		inTok := clampNonNegative(row.InputTokens)
		outTok := clampNonNegative(row.OutputTokens)
		reasoning := clampNonNegative(row.ReasoningTokens)
		cached := clampNonNegative(row.CachedTokens)
		if reasoning > outTok {
			reasoning = outTok
		}
		if cached > inTok {
			cached = inTok
		}

		item.Requests++
		item.InputTokens += inTok
		item.OutputTokens += outTok
		item.ReasoningTokens += reasoning
		item.CachedTokens += cached
		item.EstimatedUSD += estimateCostUSD(model, inTok, outTok, reasoning, cached)
	}

	out := make([]ModelUsage, 0, len(byModel))
	for _, item := range byModel {
		item.TotalTokens = item.InputTokens + item.OutputTokens
		item.StandardTokens = clampNonNegative(item.TotalTokens - item.ReasoningTokens - item.CachedTokens)
		item.EstimatedUSD = roundUSD(item.EstimatedUSD)
		out = append(out, *item)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].TotalTokens != out[j].TotalTokens {
			return out[i].TotalTokens > out[j].TotalTokens
		}
		if out[i].Requests != out[j].Requests {
			return out[i].Requests > out[j].Requests
		}
		return out[i].ModelID < out[j].ModelID
	})
	return out
}

func applyModelShare(list []ModelUsage, totalTokens int) {
	if totalTokens <= 0 {
		for i := range list {
			list[i].Share = 0
		}
		return
	}
	den := float64(totalTokens)
	for i := range list {
		list[i].Share = math.Round((float64(list[i].TotalTokens)/den)*10000) / 100
	}
}

func aggregateProviders(logs []RequestLog) []ProviderUsage {
	byProvider := make(map[string]*ProviderUsage)
	for _, row := range logs {
		provider := strings.TrimSpace(row.ProviderID)
		if provider == "" {
			provider = "-"
		}
		item := byProvider[provider]
		if item == nil {
			item = &ProviderUsage{ProviderID: provider}
			byProvider[provider] = item
		}
		inTok := clampNonNegative(row.InputTokens)
		outTok := clampNonNegative(row.OutputTokens)
		reasoning := clampNonNegative(row.ReasoningTokens)
		cached := clampNonNegative(row.CachedTokens)
		if reasoning > outTok {
			reasoning = outTok
		}
		if cached > inTok {
			cached = inTok
		}

		item.Requests++
		item.InputTokens += inTok
		item.OutputTokens += outTok
		item.ReasoningTokens += reasoning
		item.CachedTokens += cached
		item.EstimatedUSD += estimateCostUSD(row.ModelID, inTok, outTok, reasoning, cached)
	}

	out := make([]ProviderUsage, 0, len(byProvider))
	for _, item := range byProvider {
		item.TotalTokens = item.InputTokens + item.OutputTokens
		item.StandardTokens = clampNonNegative(item.TotalTokens - item.ReasoningTokens - item.CachedTokens)
		item.EstimatedUSD = roundUSD(item.EstimatedUSD)
		out = append(out, *item)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].TotalTokens != out[j].TotalTokens {
			return out[i].TotalTokens > out[j].TotalTokens
		}
		if out[i].Requests != out[j].Requests {
			return out[i].Requests > out[j].Requests
		}
		return out[i].ProviderID < out[j].ProviderID
	})
	return out
}

func applyProviderShare(list []ProviderUsage, totalTokens int) {
	if totalTokens <= 0 {
		for i := range list {
			list[i].Share = 0
		}
		return
	}
	den := float64(totalTokens)
	for i := range list {
		list[i].Share = math.Round((float64(list[i].TotalTokens)/den)*10000) / 100
	}
}

func aggregateAPIs(logs []RequestLog) []APIUsage {
	byAPI := make(map[string]*APIUsage)
	for _, row := range logs {
		backendID, backendURL := canonicalBackendFromPath(row.Path)
		if backendID == "" {
			backendID = "unknown"
		}
		if backendURL == "" {
			backendURL = "-"
		}
		key := backendID + "\x00" + backendURL
		item := byAPI[key]
		if item == nil {
			item = &APIUsage{BackendID: backendID, BackendURL: backendURL}
			byAPI[key] = item
		}

		inTok := clampNonNegative(row.InputTokens)
		outTok := clampNonNegative(row.OutputTokens)
		reasoning := clampNonNegative(row.ReasoningTokens)
		cached := clampNonNegative(row.CachedTokens)
		if reasoning > outTok {
			reasoning = outTok
		}
		if cached > inTok {
			cached = inTok
		}

		item.Requests++
		item.InputTokens += inTok
		item.OutputTokens += outTok
		item.ReasoningTokens += reasoning
		item.CachedTokens += cached
		item.EstimatedUSD += estimateCostUSD(row.ModelID, inTok, outTok, reasoning, cached)
	}

	out := make([]APIUsage, 0, len(byAPI))
	for _, item := range byAPI {
		item.TotalTokens = item.InputTokens + item.OutputTokens
		item.StandardTokens = clampNonNegative(item.TotalTokens - item.ReasoningTokens - item.CachedTokens)
		item.EstimatedUSD = roundUSD(item.EstimatedUSD)
		out = append(out, *item)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].TotalTokens != out[j].TotalTokens {
			return out[i].TotalTokens > out[j].TotalTokens
		}
		if out[i].Requests != out[j].Requests {
			return out[i].Requests > out[j].Requests
		}
		if out[i].BackendID != out[j].BackendID {
			return out[i].BackendID < out[j].BackendID
		}
		return out[i].BackendURL < out[j].BackendURL
	})
	return out
}

func applyAPIShare(list []APIUsage, totalTokens int) {
	if totalTokens <= 0 {
		for i := range list {
			list[i].Share = 0
		}
		return
	}
	den := float64(totalTokens)
	for i := range list {
		list[i].Share = math.Round((float64(list[i].TotalTokens)/den)*10000) / 100
	}
}

func buildSpecialBackends(logs []RequestLog) []SpecialBackendUsage {
	out := make([]SpecialBackendUsage, 0, len(specialBackendDefs))
	for _, def := range specialBackendDefs {
		filtered := make([]RequestLog, 0, len(logs)/3+1)
		for _, row := range logs {
			if matchSpecialBackend(row, def) {
				filtered = append(filtered, row)
			}
		}
		summary := summarize(filtered)
		models := aggregateModels(filtered)
		providers := aggregateProviders(filtered)
		applyModelShare(models, summary.TotalTokens)
		applyProviderShare(providers, summary.TotalTokens)
		out = append(out, SpecialBackendUsage{
			ID:            def.ID,
			Name:          def.Name,
			BackendURL:    "/api/" + def.ID + "/v1/*",
			Summary:       summary,
			ModelUsage:    models,
			ProviderUsage: providers,
		})
	}
	return out
}

func matchSpecialBackend(row RequestLog, def specialBackendDef) bool {
	if extractAppIDFromPath(row.Path) == def.ID {
		return true
	}
	target := strings.ToLower(strings.Join([]string{row.ProviderID, row.Path, row.ModelID}, " "))
	for _, alias := range def.Aliases {
		if strings.Contains(target, alias) {
			return true
		}
	}
	return false
}

func canonicalBackendFromPath(path string) (string, string) {
	normalized := normalizeRoutePath(path)
	if appID := extractAppIDFromPath(normalized); appID != "" {
		return appID, "/api/" + appID + "/v1/*"
	}
	switch {
	case strings.HasPrefix(normalized, "/openai/v1"):
		return "openai", "/openai/v1/*"
	case strings.HasPrefix(normalized, "/v1"):
		return "default-v1", "/v1/*"
	}

	trimmed := strings.Trim(normalized, "/")
	if trimmed == "" {
		return "unknown", "-"
	}
	head := strings.SplitN(trimmed, "/", 2)[0]
	if head == "" {
		return "unknown", "-"
	}
	return head, "/" + head + "/*"
}

func extractAppIDFromPath(path string) string {
	normalized := normalizeRoutePath(path)
	trimmed := strings.Trim(normalized, "/")
	if trimmed == "" {
		return ""
	}
	parts := strings.Split(trimmed, "/")
	if len(parts) >= 3 && parts[0] == "api" && parts[2] == "v1" && parts[1] != "" {
		return parts[1]
	}
	return ""
}

func normalizeRoutePath(path string) string {
	out := strings.ToLower(strings.TrimSpace(path))
	if out == "" {
		return ""
	}
	if i := strings.Index(out, "?"); i >= 0 {
		out = out[:i]
	}
	out = strings.ReplaceAll(out, "\\", "/")
	if !strings.HasPrefix(out, "/") {
		out = "/" + out
	}
	for strings.Contains(out, "//") {
		out = strings.ReplaceAll(out, "//", "/")
	}
	return out
}

func buildTrend(logs []RequestLog, start, end time.Time, bucketSeconds int) []TrendPoint {
	if bucketSeconds <= 0 {
		bucketSeconds = 300
	}
	bucketSize := int64(bucketSeconds)

	type agg struct {
		requests  int
		input     int
		output    int
		reasoning int
		cached    int
		estimated float64
	}
	byBucket := make(map[int64]*agg)

	for _, row := range logs {
		ts, ok := ParseRFC3339(row.Timestamp)
		if !ok {
			continue
		}
		unix := ts.Unix()
		bucket := (unix / bucketSize) * bucketSize
		item := byBucket[bucket]
		if item == nil {
			item = &agg{}
			byBucket[bucket] = item
		}
		inTok := clampNonNegative(row.InputTokens)
		outTok := clampNonNegative(row.OutputTokens)
		reasoning := clampNonNegative(row.ReasoningTokens)
		cached := clampNonNegative(row.CachedTokens)
		if reasoning > outTok {
			reasoning = outTok
		}
		if cached > inTok {
			cached = inTok
		}
		item.requests++
		item.input += inTok
		item.output += outTok
		item.reasoning += reasoning
		item.cached += cached
		item.estimated += estimateCostUSD(row.ModelID, inTok, outTok, reasoning, cached)
	}

	startBucket := (start.Unix() / bucketSize) * bucketSize
	endBucket := (end.Unix() / bucketSize) * bucketSize
	if endBucket < startBucket {
		endBucket = startBucket
	}
	maxPoints := 600
	if int((endBucket-startBucket)/bucketSize)+1 > maxPoints {
		startBucket = endBucket - int64(maxPoints-1)*bucketSize
	}

	out := make([]TrendPoint, 0, int((endBucket-startBucket)/bucketSize)+1)
	for bucket := startBucket; bucket <= endBucket; bucket += bucketSize {
		item := byBucket[bucket]
		if item == nil {
			item = &agg{}
		}
		total := item.input + item.output
		standard := clampNonNegative(total - item.reasoning - item.cached)
		out = append(out, TrendPoint{
			BucketStart:     time.Unix(bucket, 0).UTC().Format(time.RFC3339),
			Requests:        item.requests,
			InputTokens:     item.input,
			OutputTokens:    item.output,
			ReasoningTokens: item.reasoning,
			CachedTokens:    item.cached,
			StandardTokens:  standard,
			TotalTokens:     total,
			EstimatedUSD:    roundUSD(item.estimated),
		})
	}
	return out
}

func estimateCostUSD(modelID string, inputTokens, outputTokens, reasoningTokens, cachedTokens int) float64 {
	rule := matchPriceRule(modelID)
	if cachedTokens > inputTokens {
		cachedTokens = inputTokens
	}
	if reasoningTokens > outputTokens {
		reasoningTokens = outputTokens
	}
	inputNormal := clampNonNegative(inputTokens - cachedTokens)
	outputNormal := clampNonNegative(outputTokens - reasoningTokens)

	cost := (float64(inputNormal) / 1_000_000.0 * rule.InputUSD) +
		(float64(cachedTokens) / 1_000_000.0 * rule.CachedUSD) +
		(float64(outputNormal) / 1_000_000.0 * rule.OutputUSD) +
		(float64(reasoningTokens) / 1_000_000.0 * rule.ReasoningUSD)
	if cost < 0 {
		return 0
	}
	return cost
}

func matchPriceRule(modelID string) priceRule {
	id := strings.ToLower(strings.TrimSpace(modelID))
	for _, rule := range priceRules {
		if strings.Contains(id, strings.ToLower(rule.Prefix)) {
			return rule
		}
	}
	return fallbackRule
}

func clampNonNegative(v int) int {
	if v < 0 {
		return 0
	}
	return v
}

func roundUSD(v float64) float64 {
	if v <= 0 {
		return 0
	}
	return math.Round(v*1_000_000) / 1_000_000
}
