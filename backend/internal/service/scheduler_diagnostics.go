package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

const schedulerDiagnosticsMarker = "scheduler_diagnostics="

// SchedulerSelectionError separates the safe client-facing summary from
// request-time account diagnostics, which may contain internal account IDs,
// names and operational state. Diagnostics are attached to Ops context only.
type SchedulerSelectionError struct {
	message     string
	diagnostics string
}

func (e *SchedulerSelectionError) Error() string {
	if e == nil || strings.TrimSpace(e.message) == "" {
		return ErrNoAvailableAccounts.Error()
	}
	return e.message
}

func (e *SchedulerSelectionError) Unwrap() error {
	return ErrNoAvailableAccounts
}

func newSchedulerSelectionError(message, diagnostics string) error {
	return &SchedulerSelectionError{
		message:     strings.TrimSpace(message),
		diagnostics: strings.TrimSpace(diagnostics),
	}
}

// SchedulerDiagnosticsFromError returns an Ops-only payload. It must never
// be appended to a downstream/client-facing error response.
func SchedulerDiagnosticsFromError(err error) string {
	var selectionErr *SchedulerSelectionError
	if !errors.As(err, &selectionErr) || selectionErr == nil || selectionErr.diagnostics == "" {
		return ""
	}
	return schedulerDiagnosticsMarker + selectionErr.diagnostics
}

type schedulerAccountDiagnostic struct {
	AccountID      int64  `json:"account_id"`
	AccountName    string `json:"account_name,omitempty"`
	Platform       string `json:"platform,omitempty"`
	RelayMode      string `json:"relay_mode,omitempty"`
	TargetProtocol string `json:"target_protocol,omitempty"`
	Available      bool   `json:"available"`
	Stage          string `json:"stage,omitempty"`
	Reason         string `json:"reason,omitempty"`
	Detail         string `json:"detail,omitempty"`
}

type schedulerDiagnosticsEnvelope struct {
	Version         int                          `json:"version"`
	InboundProtocol string                       `json:"inbound_protocol,omitempty"`
	RequestedModel  string                       `json:"requested_model,omitempty"`
	GroupID         int64                        `json:"group_id,omitempty"`
	Platform        string                       `json:"platform,omitempty"`
	ReasonCounts    map[string]int               `json:"reason_counts,omitempty"`
	Accounts        []schedulerAccountDiagnostic `json:"accounts"`
}

func encodeSchedulerDiagnostics(envelope schedulerDiagnosticsEnvelope) string {
	if len(envelope.Accounts) == 0 {
		return ""
	}
	payload, err := json.Marshal(envelope)
	if err != nil {
		return ""
	}
	return base64.RawURLEncoding.EncodeToString(payload)
}

func filterAccountsByRequestProtocolForScheduling(ctx context.Context, groupID *int64, platform string, accounts []Account) ([]Account, error) {
	if len(accounts) == 0 {
		return accounts, nil
	}
	filtered := make([]Account, 0, len(accounts))
	rejections := make([]schedulerAccountDiagnostic, 0)
	reasonCounts := map[string]int{}
	for i := range accounts {
		account := &accounts[i]
		if accountMatchesRequestProtocol(ctx, account) {
			filtered = append(filtered, accounts[i])
			continue
		}
		diagnostic := protocolDiagnosticForAccount(ctx, account)
		rejections = append(rejections, diagnostic)
		reasonCounts[diagnostic.Reason]++
	}
	if len(filtered) > 0 || len(rejections) == 0 {
		return filtered, nil
	}
	envelope := schedulerDiagnosticsEnvelope{
		Version:         1,
		InboundProtocol: InboundProtocolFromContext(ctx),
		GroupID:         derefGroupID(groupID),
		Platform:        platform,
		ReasonCounts:    reasonCounts,
		Accounts:        rejections,
	}
	return nil, newSchedulerSelectionError(
		"no available accounts: all accounts rejected by protocol router",
		encodeSchedulerDiagnostics(envelope),
	)
}

func protocolDiagnosticForAccount(ctx context.Context, account *Account) schedulerAccountDiagnostic {
	diagnostic := schedulerAccountDiagnostic{Available: false, Stage: "protocol_router"}
	if account == nil {
		diagnostic.Reason = "account_nil"
		return diagnostic
	}
	diagnostic.AccountID = account.ID
	diagnostic.AccountName = account.Name
	diagnostic.Platform = account.EffectivePlatform()
	diagnostic.RelayMode = account.RelayMode()
	diagnostic.TargetProtocol = account.TargetProtocol()
	decision, ok := ProtocolRouteDecisionForAccount(ctx, account)
	if ok {
		diagnostic.Available = true
		diagnostic.Stage = "eligible"
		return diagnostic
	}
	diagnostic.Detail = decision.FailureReason
	switch {
	case account.IsCustom() && strings.TrimSpace(account.CustomProtocol()) == "":
		diagnostic.Reason = "custom_protocol_missing"
	case account.RelayMode() == RelayModePassthrough || account.RelayMode() == RelayModeFullPassthrough:
		diagnostic.Reason = "relay_mode_protocol_mismatch"
	case strings.Contains(strings.ToLower(decision.FailureReason), "not implemented"):
		diagnostic.Reason = "protocol_conversion_unavailable"
	default:
		diagnostic.Reason = "protocol_incompatible"
	}
	return diagnostic
}

func (s *GatewayService) encodeSchedulerSelectionDiagnostics(ctx context.Context, groupID *int64, accounts []Account, requestedModel string, platform string, excludedIDs map[int64]struct{}) string {
	if len(accounts) == 0 {
		return ""
	}
	const maxAccounts = 50
	limit := len(accounts)
	if limit > maxAccounts {
		limit = maxAccounts
	}
	diagnostics := make([]schedulerAccountDiagnostic, 0, limit)
	reasonCounts := make(map[string]int)
	for i := 0; i < limit; i++ {
		diagnostic := s.schedulerSelectionDiagnostic(ctx, groupID, &accounts[i], requestedModel, excludedIDs)
		diagnostics = append(diagnostics, diagnostic)
		if diagnostic.Reason != "" {
			reasonCounts[diagnostic.Reason]++
		}
	}
	return encodeSchedulerDiagnostics(schedulerDiagnosticsEnvelope{Version: 1, InboundProtocol: InboundProtocolFromContext(ctx), RequestedModel: requestedModel, GroupID: derefGroupID(groupID), Platform: platform, ReasonCounts: reasonCounts, Accounts: diagnostics})
}

func (s *GatewayService) schedulerNoAvailableError(
	ctx context.Context,
	groupID *int64,
	accounts []Account,
	requestedModel string,
	platform string,
	excludedIDs map[int64]struct{},
	summary string,
) error {
	diagnosticAccounts := accounts
	if len(diagnosticAccounts) == 0 && groupID != nil && s.accountRepo != nil {
		if allAccounts, err := s.accountRepo.ListByGroup(ctx, *groupID); err == nil {
			diagnosticAccounts = allAccounts
		}
	}
	encoded := s.encodeSchedulerSelectionDiagnostics(ctx, groupID, diagnosticAccounts, requestedModel, platform, excludedIDs)
	return newSchedulerSelectionError("no available accounts: "+strings.TrimSpace(summary), encoded)
}

func schedulerSessionLimitError(
	ctx context.Context,
	groupID *int64,
	accounts []*Account,
	requestedModel string,
	platform string,
) error {
	diagnostics := make([]schedulerAccountDiagnostic, 0, len(accounts))
	for _, account := range accounts {
		if account == nil {
			continue
		}
		diagnostics = append(diagnostics, schedulerAccountDiagnostic{
			AccountID:      account.ID,
			AccountName:    account.Name,
			Platform:       account.EffectivePlatform(),
			RelayMode:      account.RelayMode(),
			TargetProtocol: account.TargetProtocol(),
			Available:      false,
			Stage:          "session_limit",
			Reason:         "session_window_rejected",
			Detail:         "request-time session limit rejected this account",
		})
	}
	envelope := schedulerDiagnosticsEnvelope{
		Version:         1,
		InboundProtocol: InboundProtocolFromContext(ctx),
		RequestedModel:  requestedModel,
		GroupID:         derefGroupID(groupID),
		Platform:        platform,
		ReasonCounts:    map[string]int{"session_window_rejected": len(diagnostics)},
		Accounts:        diagnostics,
	}
	return newSchedulerSelectionError(
		"no available accounts: all eligible accounts rejected by session limits",
		encodeSchedulerDiagnostics(envelope),
	)
}

func (s *GatewayService) schedulerSelectionDiagnostic(ctx context.Context, groupID *int64, account *Account, requestedModel string, excludedIDs map[int64]struct{}) schedulerAccountDiagnostic {
	if account == nil {
		return schedulerAccountDiagnostic{Available: false, Stage: "account_state", Reason: "account_nil"}
	}
	diagnostic := schedulerAccountDiagnostic{AccountID: account.ID, AccountName: account.Name, Platform: account.EffectivePlatform(), RelayMode: account.RelayMode(), TargetProtocol: account.TargetProtocol()}
	reject := func(stage, reason, detail string) schedulerAccountDiagnostic {
		diagnostic.Available = false
		diagnostic.Stage = stage
		diagnostic.Reason = reason
		diagnostic.Detail = detail
		return diagnostic
	}
	if _, excluded := excludedIDs[account.ID]; excluded {
		return reject("request_exclusion", "excluded", "account was excluded by a previous attempt")
	}
	if route, ok := ProtocolRouteDecisionForAccount(ctx, account); !ok {
		return protocolDiagnosticForAccount(ctx, account)
	} else {
		diagnostic.TargetProtocol = route.TargetProtocol
	}
	if !account.IsActive() {
		return reject("account_state", "status_inactive", "status="+account.Status)
	}
	if !account.Schedulable {
		return reject("account_state", "schedulable_disabled", "manual scheduling switch is off")
	}
	now := time.Now()
	if account.AutoPauseOnExpired && account.ExpiresAt != nil && !now.Before(*account.ExpiresAt) {
		return reject("account_state", "expired", account.ExpiresAt.Format(time.RFC3339))
	}
	if account.OverloadUntil != nil && now.Before(*account.OverloadUntil) {
		return reject("account_state", "overloaded", account.OverloadUntil.Format(time.RFC3339))
	}
	if account.RateLimitResetAt != nil && now.Before(*account.RateLimitResetAt) {
		return reject("account_state", "rate_limited", account.RateLimitResetAt.Format(time.RFC3339))
	}
	if account.TempUnschedulableUntil != nil && now.Before(*account.TempUnschedulableUntil) {
		detail := account.TempUnschedulableUntil.Format(time.RFC3339)
		if account.TempUnschedulableReason != "" {
			detail = account.TempUnschedulableReason + "; until=" + detail
		}
		return reject("account_state", "temp_unschedulable", detail)
	}
	if account.IsAPIKeyOrBedrock() && account.IsQuotaExceeded() {
		return reject("quota", "quota_exhausted", "account quota is exhausted")
	}
	if groupID != nil {
		if group := s.groupFromContext(ctx, *groupID); group != nil && group.RequirePrivacySet && !account.IsPrivacySet() {
			return reject("group_policy", "privacy_required", "group requires privacy to be configured")
		}
	}
	if requestedModel != "" && !s.isModelSupportedByAccountWithContext(ctx, account, requestedModel) {
		return reject("model_filter", "model_not_allowed", "model="+requestedModel)
	}
	if groupID != nil && s.needsUpstreamChannelRestrictionCheck(ctx, groupID) && s.isUpstreamModelRestrictedByChannel(ctx, *groupID, account, requestedModel) {
		return reject("channel_policy", "channel_restricted", "upstream model is restricted by channel pricing policy")
	}
	if !s.isAccountSchedulableForModelSelection(ctx, account, requestedModel) {
		remaining := account.GetRateLimitRemainingTimeWithContext(ctx, requestedModel).Truncate(time.Second)
		return reject("model_rate_limit", "model_rate_limited", "remaining="+remaining.String())
	}
	if !s.isAccountSchedulableForQuota(account) {
		return reject("quota", "quota_exhausted", "quota eligibility check rejected account")
	}
	if !s.isAccountSchedulableForWindowCost(ctx, account, false) {
		return reject("window_cost", "window_cost_exceeded", "rolling window cost limit reached")
	}
	if !s.isAccountSchedulableForRPM(ctx, account, false) {
		return reject("rpm", "rpm_limit", "account RPM limit reached")
	}
	diagnostic.Available = true
	diagnostic.Stage = "eligible"
	return diagnostic
}
