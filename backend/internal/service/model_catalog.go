package service

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"
)

const (
	AccountExtraKeySupportedModels     = "supported_models"
	AccountExtraKeyRestrictToModelList = "restrict_to_model_list"
	AccountExtraKeyModelCatalogSync    = "model_catalog_sync"

	ModelCatalogSourceManual           = "manual"
	ModelCatalogSourceUpstream         = "upstream"
	ModelCatalogSourceMappingMigration = "mapping_migration"

	ModelCatalogSyncStatusOK    = "ok"
	ModelCatalogSyncStatusError = "error"
)

type AccountModelCatalogEntry struct {
	ID          int64
	AccountID   int64
	ModelID     string
	Platform    string
	Source      string
	DisplayName string
	UsageModes  []string
	LastSeenAt  time.Time
	SyncBatchID string
	SyncStatus  string
	SyncError   string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type AccountModelSyncState struct {
	AccountID    int64
	Source       string
	Status       string
	ModelCount   int
	SyncBatchID  string
	LastSyncedAt *time.Time
	ErrorMessage string
	UpdatedAt    time.Time
}

type ModelCatalogRepository interface {
	ReplaceAccountModels(ctx context.Context, accountID int64, platform, source string, modelIDs []string, usageModes []string) (*AccountModelSyncState, error)
	RecordAccountSyncFailure(ctx context.Context, accountID int64, source, message string) (*AccountModelSyncState, error)
	ListByAccount(ctx context.Context, accountID int64) ([]AccountModelCatalogEntry, *AccountModelSyncState, error)
	ListByAccounts(ctx context.Context, accountIDs []int64) ([]AccountModelCatalogEntry, map[int64]*AccountModelSyncState, error)
}

type ModelCatalogService struct {
	repo           ModelCatalogRepository
	accountRepo    AccountRepository
	groupRepo      GroupRepository
	channelService *ChannelService
	monitorService *ChannelMonitorService
	settingService *SettingService
}

func NewModelCatalogService(
	repo ModelCatalogRepository,
	accountRepo AccountRepository,
	groupRepo GroupRepository,
	channelService *ChannelService,
	monitorService *ChannelMonitorService,
	settingService *SettingService,
) *ModelCatalogService {
	return &ModelCatalogService{
		repo:           repo,
		accountRepo:    accountRepo,
		groupRepo:      groupRepo,
		channelService: channelService,
		monitorService: monitorService,
		settingService: settingService,
	}
}

func (s *ModelCatalogService) ReplaceAccountModelsFromSync(ctx context.Context, account *Account, models []string, source string) (*AccountModelSyncState, []string, error) {
	if s == nil || s.repo == nil || s.accountRepo == nil {
		return nil, normalizeModelIDs(models), nil
	}
	if account == nil {
		return nil, nil, ErrAccountNilInput
	}
	normalized := normalizeModelIDs(models)
	if source == "" {
		source = ModelCatalogSourceUpstream
	}
	state, err := s.repo.ReplaceAccountModels(ctx, account.ID, account.EffectivePlatform(), source, normalized, defaultUsageModesForAccount(account))
	if err != nil {
		return nil, nil, fmt.Errorf("replace account model catalog: %w", err)
	}
	extra := map[string]any{
		AccountExtraKeySupportedModels:  normalized,
		AccountExtraKeyModelCatalogSync: syncStateToExtra(state),
	}
	if err := s.accountRepo.UpdateExtra(ctx, account.ID, extra); err != nil {
		return nil, nil, fmt.Errorf("update account model catalog cache: %w", err)
	}
	return state, normalized, nil
}

func (s *ModelCatalogService) RecordSyncFailure(ctx context.Context, account *Account, source string, syncErr error) {
	if s == nil || s.repo == nil || s.accountRepo == nil || account == nil || syncErr == nil {
		return
	}
	if source == "" {
		source = ModelCatalogSourceUpstream
	}
	message := syncErr.Error()
	state, err := s.repo.RecordAccountSyncFailure(ctx, account.ID, source, message)
	if err != nil {
		return
	}
	_ = s.accountRepo.UpdateExtra(ctx, account.ID, map[string]any{
		AccountExtraKeyModelCatalogSync: syncStateToExtra(state),
	})
}

func (s *ModelCatalogService) ListAccountModels(ctx context.Context, account *Account) ([]string, *AccountModelSyncState, error) {
	if account == nil {
		return nil, nil, ErrAccountNilInput
	}
	if s == nil || s.repo == nil {
		return account.SupportedModelIDs(), account.ModelCatalogSyncState(), nil
	}
	entries, state, err := s.repo.ListByAccount(ctx, account.ID)
	if err != nil {
		return nil, nil, err
	}
	models := catalogEntriesToModelIDs(entries)
	if len(models) == 0 {
		models = account.SupportedModelIDs()
	}
	if state == nil {
		state = account.ModelCatalogSyncState()
	}
	return models, state, nil
}

func (s *ModelCatalogService) ListCatalog(ctx context.Context, groupID *int64, includeSources bool) (*ModelCatalogView, error) {
	accounts, err := s.catalogAccounts(ctx, groupID)
	if err != nil {
		return nil, err
	}
	accountIDs := make([]int64, 0, len(accounts))
	accountByID := make(map[int64]Account, len(accounts))
	for _, acc := range accounts {
		accountIDs = append(accountIDs, acc.ID)
		accountByID[acc.ID] = acc
	}

	entriesByAccount := make(map[int64][]AccountModelCatalogEntry, len(accounts))
	var states map[int64]*AccountModelSyncState
	if s != nil && s.repo != nil && len(accountIDs) > 0 {
		entries, syncStates, err := s.repo.ListByAccounts(ctx, accountIDs)
		if err != nil {
			return nil, err
		}
		states = syncStates
		for _, entry := range entries {
			entriesByAccount[entry.AccountID] = append(entriesByAccount[entry.AccountID], entry)
		}
	}

	groups := make(map[int64]ModelCatalogGroupRef)
	if s != nil && s.groupRepo != nil {
		active, err := s.groupRepo.ListActive(ctx)
		if err == nil {
			for _, g := range active {
				if groupID != nil && g.ID != *groupID {
					continue
				}
				groups[g.ID] = ModelCatalogGroupRef{
					ID:               g.ID,
					Name:             g.Name,
					Platform:         g.Platform,
					SubscriptionType: g.SubscriptionType,
					RateMultiplier:   g.RateMultiplier,
					IsExclusive:      g.IsExclusive,
				}
			}
		}
	}

	sourceChannels := s.modelPricingByAccount(ctx)
	merged := map[string]*ModelCatalogModel{}
	for _, account := range accounts {
		models := entriesByAccount[account.ID]
		if len(models) == 0 {
			for _, modelID := range account.SupportedModelIDs() {
				models = append(models, AccountModelCatalogEntry{
					AccountID:   account.ID,
					ModelID:     modelID,
					Platform:    account.EffectivePlatform(),
					Source:      ModelCatalogSourceManual,
					DisplayName: modelID,
					UsageModes:  defaultUsageModesForAccount(&account),
				})
			}
		}
		if len(models) == 0 {
			continue
		}
		for _, entry := range models {
			modelID := strings.TrimSpace(entry.ModelID)
			if modelID == "" {
				continue
			}
			key := strings.ToLower(modelID)
			model := merged[key]
			if model == nil {
				model = &ModelCatalogModel{
					ID:          modelID,
					DisplayName: firstNonEmptyCatalogString(entry.DisplayName, modelID),
					Platform:    firstNonEmptyCatalogString(entry.Platform, account.EffectivePlatform()),
					UsageModes:  normalizeStringSet(entry.UsageModes),
				}
				merged[key] = model
			}
			model.SourceCount++
			model.UsageModes = mergeStringSlices(model.UsageModes, entry.UsageModes)
			for _, gid := range account.GroupIDs {
				if groupID != nil && gid != *groupID {
					continue
				}
				if g, ok := groups[gid]; ok {
					model.Groups = appendUniqueGroup(model.Groups, g)
				}
			}
			if includeSources {
				src := ModelCatalogSourceRef{
					AccountID:   account.ID,
					AccountName: account.Name,
					Platform:    account.EffectivePlatform(),
					Source:      entry.Source,
					SyncStatus:  entry.SyncStatus,
					UpdatedAt:   entry.UpdatedAt,
				}
				if state, ok := states[account.ID]; ok && state != nil {
					src.SyncStatus = state.Status
					src.SyncError = state.ErrorMessage
				}
				if channels := sourceChannels[account.ID][strings.ToLower(modelID)]; len(channels) > 0 {
					src.Channels = channels
					for _, channel := range channels {
						if channel.Pricing != nil {
							src.Pricing = channel.Pricing
						}
						model.PriceRange = mergePriceRange(model.PriceRange, channel.Pricing)
					}
				}
				model.Sources = append(model.Sources, src)
			} else if channels := sourceChannels[account.ID][strings.ToLower(modelID)]; len(channels) > 0 {
				for _, channel := range channels {
					model.PriceRange = mergePriceRange(model.PriceRange, channel.Pricing)
				}
			}
		}
	}

	out := &ModelCatalogView{Models: make([]ModelCatalogModel, 0, len(merged))}
	for _, model := range merged {
		sort.SliceStable(model.Groups, func(i, j int) bool { return model.Groups[i].Name < model.Groups[j].Name })
		sort.Strings(model.UsageModes)
		out.Models = append(out.Models, *model)
	}
	sort.SliceStable(out.Models, func(i, j int) bool {
		return strings.ToLower(out.Models[i].ID) < strings.ToLower(out.Models[j].ID)
	})

	// 填充监控状态：按 primary_model 匹配 channel_monitors，批量查最新状态 + 7d 可用率
	s.enrichMonitorStatus(ctx, out.Models)

	return out, nil
}

func (s *ModelCatalogService) catalogAccounts(ctx context.Context, groupID *int64) ([]Account, error) {
	if s == nil || s.accountRepo == nil {
		return nil, nil
	}
	if groupID != nil {
		return s.accountRepo.ListByGroup(ctx, *groupID)
	}
	return s.accountRepo.ListActive(ctx)
}

func (s *ModelCatalogService) modelPricingByAccount(ctx context.Context) map[int64]map[string][]ModelCatalogChannelRef {
	result := make(map[int64]map[string][]ModelCatalogChannelRef)
	if s == nil || s.channelService == nil || s.accountRepo == nil {
		return result
	}
	channels, err := s.channelService.ListAvailable(ctx)
	if err != nil {
		return result
	}
	for _, ch := range channels {
		for _, group := range ch.Groups {
			accounts, err := s.accountRepo.ListByGroup(ctx, group.ID)
			if err != nil {
				continue
			}
			for _, acc := range accounts {
				if _, ok := result[acc.ID]; !ok {
					result[acc.ID] = make(map[string][]ModelCatalogChannelRef)
				}
				for _, m := range ch.SupportedModels {
					if m.Platform != "" && m.Platform != acc.EffectivePlatform() && !acc.IsAntigravity() {
						continue
					}
					modelLower := strings.ToLower(m.Name)
					channelRef := ModelCatalogChannelRef{
						ID:      ch.ID,
						Name:    ch.Name,
						Pricing: modelCatalogPricingFromChannel(m.Pricing),
					}
					if !hasCatalogChannelRef(result[acc.ID][modelLower], channelRef.ID) {
						result[acc.ID][modelLower] = append(result[acc.ID][modelLower], channelRef)
					}
				}
			}
		}
	}
	return result
}

type ModelCatalogView struct {
	Models []ModelCatalogModel `json:"models"`
}

type ModelCatalogModel struct {
	ID          string                  `json:"id"`
	DisplayName string                  `json:"display_name"`
	Platform    string                  `json:"platform"`
	UsageModes  []string                `json:"usage_modes"`
	SourceCount int                     `json:"source_count"`
	Groups      []ModelCatalogGroupRef  `json:"groups"`
	PriceRange  *ModelCatalogPriceRange `json:"price_range,omitempty"`
	Sources     []ModelCatalogSourceRef `json:"sources,omitempty"`

	// 监控状态（由 channel_monitors 按 primary_model 匹配聚合）
	MonitorID        *int64   `json:"monitor_id,omitempty"`
	MonitorStatus    string   `json:"monitor_status,omitempty"`
	MonitorLatencyMs *int     `json:"monitor_latency_ms,omitempty"`
	MonitorAvail7d   *float64 `json:"monitor_availability_7d,omitempty"`
}

type ModelCatalogGroupRef struct {
	ID               int64   `json:"id"`
	Name             string  `json:"name"`
	Platform         string  `json:"platform"`
	SubscriptionType string  `json:"subscription_type"`
	RateMultiplier   float64 `json:"rate_multiplier"`
	IsExclusive      bool    `json:"is_exclusive"`
}

type ModelCatalogSourceRef struct {
	AccountID   int64                    `json:"account_id,omitempty"`
	AccountName string                   `json:"account_name,omitempty"`
	Platform    string                   `json:"platform"`
	Source      string                   `json:"source"`
	SyncStatus  string                   `json:"sync_status,omitempty"`
	SyncError   string                   `json:"sync_error,omitempty"`
	Pricing     *ModelCatalogPricing     `json:"pricing,omitempty"`
	Channels    []ModelCatalogChannelRef `json:"channels,omitempty"`
	UpdatedAt   time.Time                `json:"updated_at,omitempty"`
}

type ModelCatalogChannelRef struct {
	ID      int64                `json:"id"`
	Name    string               `json:"name"`
	Pricing *ModelCatalogPricing `json:"pricing,omitempty"`
}

type ModelCatalogPricing struct {
	BillingMode      string   `json:"billing_mode"`
	InputPrice       *float64 `json:"input_price"`
	OutputPrice      *float64 `json:"output_price"`
	CacheWritePrice  *float64 `json:"cache_write_price"`
	CacheReadPrice   *float64 `json:"cache_read_price"`
	ImageOutputPrice *float64 `json:"image_output_price"`
	PerRequestPrice  *float64 `json:"per_request_price"`
}

type ModelCatalogPriceRange struct {
	BillingMode        string   `json:"billing_mode"`
	MinInputPrice      *float64 `json:"min_input_price"`
	MaxInputPrice      *float64 `json:"max_input_price"`
	MinOutputPrice     *float64 `json:"min_output_price"`
	MaxOutputPrice     *float64 `json:"max_output_price"`
	MinPerRequestPrice *float64 `json:"min_per_request_price"`
	MaxPerRequestPrice *float64 `json:"max_per_request_price"`
}

func normalizeModelIDs(models []string) []string {
	seen := make(map[string]struct{}, len(models))
	out := make([]string, 0, len(models))
	for _, raw := range models {
		model := strings.TrimSpace(raw)
		if model == "" {
			continue
		}
		key := strings.ToLower(model)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, model)
	}
	sort.Strings(out)
	return out
}

func catalogEntriesToModelIDs(entries []AccountModelCatalogEntry) []string {
	models := make([]string, 0, len(entries))
	for _, entry := range entries {
		models = append(models, entry.ModelID)
	}
	return normalizeModelIDs(models)
}

func defaultUsageModesForAccount(account *Account) []string {
	if account != nil && account.IsOpenAI() {
		return []string{"chat", "responses"}
	}
	return []string{"chat"}
}

func syncStateToExtra(state *AccountModelSyncState) map[string]any {
	if state == nil {
		return nil
	}
	out := map[string]any{
		"source":      state.Source,
		"status":      state.Status,
		"model_count": state.ModelCount,
		"updated_at":  state.UpdatedAt.Format(time.RFC3339),
	}
	if state.SyncBatchID != "" {
		out["sync_batch_id"] = state.SyncBatchID
	}
	if state.LastSyncedAt != nil {
		out["last_synced_at"] = state.LastSyncedAt.Format(time.RFC3339)
	}
	if state.ErrorMessage != "" {
		out["error_message"] = state.ErrorMessage
	}
	return out
}

func normalizeStringSet(values []string) []string {
	return mergeStringSlices(nil, values)
}

func mergeStringSlices(a, b []string) []string {
	seen := make(map[string]struct{}, len(a)+len(b))
	out := make([]string, 0, len(a)+len(b))
	for _, raw := range append(a, b...) {
		v := strings.TrimSpace(raw)
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}

func appendUniqueGroup(groups []ModelCatalogGroupRef, group ModelCatalogGroupRef) []ModelCatalogGroupRef {
	for _, g := range groups {
		if g.ID == group.ID {
			return groups
		}
	}
	return append(groups, group)
}

func firstNonEmptyCatalogString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func modelCatalogPricingFromChannel(p *ChannelModelPricing) *ModelCatalogPricing {
	if p == nil {
		return nil
	}
	mode := string(p.BillingMode)
	if mode == "" {
		mode = string(BillingModeToken)
	}
	return &ModelCatalogPricing{
		BillingMode:      mode,
		InputPrice:       p.InputPrice,
		OutputPrice:      p.OutputPrice,
		CacheWritePrice:  p.CacheWritePrice,
		CacheReadPrice:   p.CacheReadPrice,
		ImageOutputPrice: p.ImageOutputPrice,
		PerRequestPrice:  p.PerRequestPrice,
	}
}

func hasCatalogChannelRef(channels []ModelCatalogChannelRef, channelID int64) bool {
	for _, ch := range channels {
		if ch.ID == channelID {
			return true
		}
	}
	return false
}

func mergePriceRange(current *ModelCatalogPriceRange, pricing *ModelCatalogPricing) *ModelCatalogPriceRange {
	if pricing == nil {
		return current
	}
	if current == nil {
		current = &ModelCatalogPriceRange{BillingMode: pricing.BillingMode}
	}
	current.MinInputPrice, current.MaxInputPrice = mergeMinMax(current.MinInputPrice, current.MaxInputPrice, pricing.InputPrice)
	current.MinOutputPrice, current.MaxOutputPrice = mergeMinMax(current.MinOutputPrice, current.MaxOutputPrice, pricing.OutputPrice)
	current.MinPerRequestPrice, current.MaxPerRequestPrice = mergeMinMax(current.MinPerRequestPrice, current.MaxPerRequestPrice, pricing.PerRequestPrice)
	return current
}

func mergeMinMax(minPtr, maxPtr, value *float64) (*float64, *float64) {
	if value == nil {
		return minPtr, maxPtr
	}
	if minPtr == nil || *value < *minPtr {
		v := *value
		minPtr = &v
	}
	if maxPtr == nil || *value > *maxPtr {
		v := *value
		maxPtr = &v
	}
	return minPtr, maxPtr
}

// enrichMonitorStatus 批量填充模型目录的监控状态。
// 按 primary_model 匹配 enabled 的 channel_monitors，再批量查最新状态 + 7d 可用率。
// 失败时仅 log warning，不阻断目录渲染。
func (s *ModelCatalogService) enrichMonitorStatus(ctx context.Context, models []ModelCatalogModel) {
	if s == nil || s.monitorService == nil || s.monitorService.repo == nil || len(models) == 0 {
		return
	}
	if s.settingService != nil && !s.settingService.IsProgressiveFeatureEnabled(ctx, ProgressiveFeatureChannelMonitor) {
		return
	}

	// 1. 收集所有模型 ID
	modelIDs := make([]string, 0, len(models))
	for _, m := range models {
		modelIDs = append(modelIDs, m.ID)
	}

	// 2. 查所有 enabled 的 monitors
	allMonitors, err := s.monitorService.repo.ListEnabled(ctx)
	if err != nil {
		slog.Warn("model_catalog: failed to list enabled monitors for enrichment", "error", err)
		return
	}
	if len(allMonitors) == 0 {
		return
	}

	// 3. 按 primary_model 建索引（小写匹配）
	modelSet := make(map[string]struct{}, len(modelIDs))
	for _, id := range modelIDs {
		modelSet[strings.ToLower(id)] = struct{}{}
	}

	type monitorMatch struct {
		monitorID int64
		modelID   string
	}
	var matches []monitorMatch
	for _, mon := range allMonitors {
		if _, ok := modelSet[strings.ToLower(mon.PrimaryModel)]; ok {
			matches = append(matches, monitorMatch{monitorID: mon.ID, modelID: mon.PrimaryModel})
		}
	}
	if len(matches) == 0 {
		return
	}

	// 4. 收集 monitor IDs，批量查最新状态 + 7d 可用率
	monitorIDs := make([]int64, 0, len(matches))
	for _, m := range matches {
		monitorIDs = append(monitorIDs, m.monitorID)
	}

	latestMap, err := s.monitorService.repo.ListLatestForMonitorIDs(ctx, monitorIDs)
	if err != nil {
		slog.Warn("model_catalog: batch load monitor latest failed", "error", err)
		latestMap = map[int64][]*ChannelMonitorLatest{}
	}
	availMap, err := s.monitorService.repo.ComputeAvailabilityForMonitors(ctx, monitorIDs, 7)
	if err != nil {
		slog.Warn("model_catalog: batch compute monitor availability failed", "error", err)
		availMap = map[int64][]*ChannelMonitorAvailability{}
	}

	// 5. 按 model name 构建最新状态索引：modelID -> latest status
	type modelStatus struct {
		monitorID int64
		status    string
		latencyMs *int
		avail7d   *float64
	}
	statusByModel := make(map[string]modelStatus, len(matches))

	for _, m := range matches {
		// 找该 monitor 主模型的最新状态
		latests := latestMap[m.monitorID]
		for _, l := range latests {
			if strings.EqualFold(l.Model, m.modelID) {
				statusByModel[strings.ToLower(m.modelID)] = modelStatus{
					monitorID: m.monitorID,
					status:    l.Status,
					latencyMs: l.LatencyMs,
				}
				break
			}
		}
	}

	// 填充可用率
	for _, m := range matches {
		key := strings.ToLower(m.modelID)
		st, ok := statusByModel[key]
		if !ok {
			continue
		}
		avails := availMap[m.monitorID]
		for _, a := range avails {
			if strings.EqualFold(a.Model, m.modelID) {
				v := a.AvailabilityPct
				st.avail7d = &v
				statusByModel[key] = st
				break
			}
		}
	}

	// 6. 填充到 models
	for i := range models {
		st, ok := statusByModel[strings.ToLower(models[i].ID)]
		if !ok {
			continue
		}
		models[i].MonitorID = &st.monitorID
		models[i].MonitorStatus = st.status
		models[i].MonitorLatencyMs = st.latencyMs
		models[i].MonitorAvail7d = st.avail7d
	}
}
