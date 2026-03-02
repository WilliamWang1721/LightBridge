package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type normalizedQuotaItem struct {
	ResourceType      string  `json:"resource_type,omitempty"`
	DisplayName       string  `json:"display_name,omitempty"`
	DisplayNamePlural string  `json:"display_name_plural,omitempty"`
	Unit              string  `json:"unit,omitempty"`
	Currency          string  `json:"currency,omitempty"`
	Used              float64 `json:"used"`
	Limit             float64 `json:"limit"`
	Remaining         float64 `json:"remaining"`
	UsedPercent       float64 `json:"used_percent"`
	RemainingPercent  float64 `json:"remaining_percent"`
	ResetAt           string  `json:"reset_at,omitempty"`
}

type normalizedQuota struct {
	Items            []normalizedQuotaItem `json:"items"`
	UsedPercent      float64               `json:"used_percent"`
	RemainingPercent float64               `json:"remaining_percent"`
	ResetAt          string                `json:"reset_at,omitempty"`
}

func (s *server) handleUsageLimits(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	accountID := strings.TrimSpace(r.URL.Query().Get("account_id"))
	if accountID == "" {
		accountID = s.store.activeAccountID()
	}
	if accountID == "" {
		acc, reason := s.store.selectAccount(s.cfg.SelectionStrategy)
		if acc == nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": nonEmpty(reason, "no account available")})
			return
		}
		accountID = acc.ID
	}
	acc, err := s.ensureAccountAccessToken(r.Context(), accountID, false)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": err.Error()})
		return
	}
	raw, err := s.fetchUsageLimits(r.Context(), acc)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": err.Error()})
		return
	}
	quota := normalizeUsageLimits(raw)
	_ = s.store.setQuotaCache(acc.ID, &quotaCache{
		UsedPercent:      quota.UsedPercent,
		RemainingPercent: quota.RemainingPercent,
		ResetAt:          quota.ResetAt,
		FetchedAt:        time.Now().UTC().Format(time.RFC3339),
	})
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":         true,
		"account_id": acc.ID,
		"raw":        raw,
		"quota":      quota,
	})
}

func generateMachineID(acc *account) string {
	if acc == nil {
		return ""
	}
	key := nonEmpty(strings.TrimSpace(acc.ID), nonEmpty(strings.TrimSpace(acc.ProfileARN), strings.TrimSpace(acc.ClientID)))
	if key == "" {
		key = "KIRO_DEFAULT_MACHINE"
	}
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:])
}

func (s *server) fetchUsageLimits(ctx context.Context, acc *account) (map[string]any, error) {
	if acc == nil {
		return nil, errors.New("nil account")
	}
	region := nonEmpty(acc.Region, s.cfg.Region)
	base := renderRegionTemplate(s.cfg.BaseURL, region)
	usageURL := strings.Replace(base, "generateAssistantResponse", "getUsageLimits", 1)

	params := url.Values{}
	params.Set("isEmailRequired", "true")
	params.Set("origin", "AI_EDITOR")
	params.Set("resourceType", "AGENTIC_REQUEST")
	if strings.EqualFold(strings.TrimSpace(acc.AuthMethod), authMethodSocial) && strings.TrimSpace(acc.ProfileARN) != "" {
		params.Set("profileArn", strings.TrimSpace(acc.ProfileARN))
	}
	fullURL := usageURL + "?" + params.Encode()

	machineID := generateMachineID(acc)
	invocationID := newUUID()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
	req.Header.Set("authorization", "Bearer "+strings.TrimSpace(acc.AccessToken))
	req.Header.Set("x-amz-user-agent", "aws-sdk-js/1.0.0 KiroIDE-0.8.140-"+machineID)
	req.Header.Set("user-agent", "aws-sdk-js/1.0.0 ua/2.1 api/codewhispererruntime#1.0.0 KiroIDE-0.8.140-"+machineID)
	req.Header.Set("amz-sdk-invocation-id", invocationID)
	req.Header.Set("amz-sdk-request", "attempt=1; max=1")
	req.Header.Set("connection", "close")

	resp, err := s.httpc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if resp.StatusCode == http.StatusUnauthorized {
		if _, ferr := s.refreshAccountTokens(ctx, acc.ID); ferr == nil {
			newAcc, ok := s.store.getAccount(acc.ID)
			if ok {
				return s.fetchUsageLimits(ctx, newAcc)
			}
		}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, errors.New(summarizeHTTPError(resp.StatusCode, body))
	}
	var out map[string]any
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func parseNumber(v any) float64 {
	switch t := v.(type) {
	case float64:
		return t
	case float32:
		return float64(t)
	case int:
		return float64(t)
	case int32:
		return float64(t)
	case int64:
		return float64(t)
	case json.Number:
		f, _ := t.Float64()
		return f
	case string:
		return parseUnitNumber(t)
	default:
		return 0
	}
}

func parseUnitNumber(raw string) float64 {
	s := strings.TrimSpace(strings.ToLower(raw))
	if s == "" {
		return 0
	}
	s = strings.ReplaceAll(s, ",", "")
	fields := strings.Fields(s)
	if len(fields) == 0 {
		return 0
	}
	numPart := fields[0]
	suffix := ""
	if len(fields) > 1 {
		suffix = fields[1]
	}
	mult := 1.0
	if strings.HasSuffix(numPart, "k") {
		mult = 1e3
		numPart = strings.TrimSuffix(numPart, "k")
	} else if strings.HasSuffix(numPart, "m") {
		mult = 1e6
		numPart = strings.TrimSuffix(numPart, "m")
	} else if strings.HasSuffix(numPart, "g") {
		mult = 1e9
		numPart = strings.TrimSuffix(numPart, "g")
	}
	switch suffix {
	case "k", "kilo", "kilobyte", "kilotoken", "kilo-token":
		mult = 1e3
	case "m", "mega", "megabyte", "megatoken", "mega-token":
		mult = 1e6
	case "g", "giga", "gigabyte", "gigatoken", "giga-token":
		mult = 1e9
	}
	f, err := strconv.ParseFloat(strings.TrimSpace(numPart), 64)
	if err != nil {
		return 0
	}
	return f * mult
}

func pickFloatMap(m map[string]any, keys ...string) float64 {
	for _, k := range keys {
		if v, ok := m[k]; ok {
			f := parseNumber(v)
			if f != 0 {
				return f
			}
		}
	}
	return 0
}

func pickResetAt(m map[string]any, keys ...string) string {
	for _, k := range keys {
		v := parseNumber(m[k])
		if v <= 0 {
			continue
		}
		return time.Unix(int64(v), 0).UTC().Format(time.RFC3339)
	}
	return ""
}

func normalizeUsageLimits(raw map[string]any) normalizedQuota {
	result := normalizedQuota{Items: []normalizedQuotaItem{}}
	if raw == nil {
		return result
	}
	if v := parseNumber(raw["nextDateReset"]); v > 0 {
		result.ResetAt = time.Unix(int64(v), 0).UTC().Format(time.RFC3339)
	}

	list, _ := raw["usageBreakdownList"].([]any)
	for _, item := range list {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		used := pickFloatMap(m, "currentUsageWithPrecision", "currentUsage")
		limit := pickFloatMap(m, "usageLimitWithPrecision", "usageLimit")
		remaining := limit - used
		if remaining < 0 {
			remaining = 0
		}
		usedPercent := 0.0
		if limit > 0 {
			usedPercent = clamp((used/limit)*100, 0, 100)
		}
		remainingPercent := 100 - usedPercent
		row := normalizedQuotaItem{
			ResourceType:      strings.TrimSpace(fmt.Sprint(m["resourceType"])),
			DisplayName:       strings.TrimSpace(fmt.Sprint(m["displayName"])),
			DisplayNamePlural: strings.TrimSpace(fmt.Sprint(m["displayNamePlural"])),
			Unit:              strings.TrimSpace(fmt.Sprint(m["unit"])),
			Currency:          strings.TrimSpace(fmt.Sprint(m["currency"])),
			Used:              round2(used),
			Limit:             round2(limit),
			Remaining:         round2(remaining),
			UsedPercent:       round2(usedPercent),
			RemainingPercent:  round2(remainingPercent),
			ResetAt:           pickResetAt(m, "nextDateReset"),
		}
		if row.ResetAt == "" {
			row.ResetAt = result.ResetAt
		}
		result.Items = append(result.Items, row)
	}

	if len(result.Items) > 0 {
		best := result.Items[0]
		for _, it := range result.Items[1:] {
			if it.Limit > best.Limit {
				best = it
			}
		}
		result.UsedPercent = best.UsedPercent
		result.RemainingPercent = best.RemainingPercent
		if result.ResetAt == "" {
			result.ResetAt = best.ResetAt
		}
	}
	return result
}

func countTextTokens(text string) int {
	runes := len([]rune(text))
	if runes <= 0 {
		return 0
	}
	return int(math.Ceil(float64(runes) / 4.0))
}

func estimatePromptTokens(req chatCompletionRequest) int {
	total := 0
	total += countTextTokens(contentToText(req.System))
	for _, msg := range req.Messages {
		if arr, ok := msg.Content.([]any); ok {
			for _, block := range arr {
				m, ok := block.(map[string]any)
				if !ok {
					continue
				}
				typ := strings.ToLower(strings.TrimSpace(fmt.Sprint(m["type"])))
				switch typ {
				case "image":
					total += 1600
				case "document":
					source, _ := m["source"].(map[string]any)
					data := strings.TrimSpace(fmt.Sprint(source["data"]))
					if data != "" {
						total += int(math.Ceil((float64(len(data)) * 0.75) / 4.0))
					}
				default:
					total += countTextTokens(contentToText([]any{m}))
				}
			}
		} else {
			total += countTextTokens(contentToText(msg.Content))
		}
		for _, tc := range msg.ToolCalls {
			if tc == nil {
				continue
			}
			fn, _ := tc["function"].(map[string]any)
			total += countTextTokens(fmt.Sprint(fn["name"]))
			total += countTextTokens(fmt.Sprint(fn["arguments"]))
		}
	}
	if len(req.Tools) > 0 {
		b, _ := json.Marshal(req.Tools)
		total += countTextTokens(string(b))
	}
	return total
}

func estimateCompletionTokens(text string, toolCalls []kiroToolCall) int {
	total := countTextTokens(text)
	for _, tc := range toolCalls {
		total += countTextTokens(tc.Name)
		total += countTextTokens(tc.Args)
	}
	if total < 0 {
		return 0
	}
	return total
}

func withUsageObject(promptTokens, completionTokens int) map[string]any {
	if promptTokens < 0 {
		promptTokens = 0
	}
	if completionTokens < 0 {
		completionTokens = 0
	}
	return map[string]any{
		"prompt_tokens":     promptTokens,
		"completion_tokens": completionTokens,
		"total_tokens":      promptTokens + completionTokens,
		"estimated":         true,
	}
}

func injectQuotaError(w http.ResponseWriter, status int, message string) {
	if strings.TrimSpace(message) == "" {
		message = "Kiro quota exhausted"
	}
	writeOpenAIError(w, status, message, "insufficient_quota", "insufficient_quota")
}

func parsePotentialJSONBody(body []byte) map[string]any {
	var m map[string]any
	if len(bytes.TrimSpace(body)) == 0 {
		return nil
	}
	if json.Unmarshal(body, &m) != nil {
		return nil
	}
	return m
}
