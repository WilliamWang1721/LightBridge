package service

import (
	"context"
	"log/slog"

	"github.com/WilliamWang1721/LightBridge/internal/config"
	"github.com/WilliamWang1721/LightBridge/internal/pkg/ctxkey"
)

// checkClaudeCodeRestriction 检查分组的 Claude Code 客户端限制
// 如果分组启用了 claude_code_only 且请求不是来自 Claude Code 客户端：
//   - 有降级分组：返回降级分组的 ID
//   - 无降级分组：返回 ErrClaudeCodeOnly 错误
func (s *GatewayService) checkClaudeCodeRestriction(ctx context.Context, groupID *int64) (*Group, *int64, error) {
	if groupID == nil {
		return nil, groupID, nil
	}

	// 强制平台模式不检查 Claude Code 限制
	if forcePlatform, hasForcePlatform := ctx.Value(ctxkey.ForcePlatform).(string); hasForcePlatform && forcePlatform != "" {
		return nil, groupID, nil
	}

	group, resolvedID, err := s.resolveGatewayGroup(ctx, groupID)
	if err != nil {
		return nil, nil, err
	}

	return group, resolvedID, nil
}

func (s *GatewayService) resolvePlatform(ctx context.Context, groupID *int64, group *Group) (string, bool, error) {
	forcePlatform, hasForcePlatform := ctx.Value(ctxkey.ForcePlatform).(string)
	if hasForcePlatform && forcePlatform != "" {
		return forcePlatform, true, nil
	}
	// An unannotated gateway call is the legacy Anthropic Messages entrypoint.
	// Normal HTTP handlers always bind the actual inbound protocol before
	// scheduling. The group is only an account boundary, never a protocol hint.
	return PlatformForRequest(ctx, PlatformAnthropic), false, nil
}

func (s *GatewayService) listSchedulableAccounts(ctx context.Context, groupID *int64, platform string, hasForcePlatform bool) ([]Account, bool, error) {
	if s.schedulerSnapshot != nil {
		accounts, useMixed, err := s.schedulerSnapshot.ListSchedulableAccounts(ctx, groupID, platform, hasForcePlatform)
		if err == nil {
			// Group snapshots contain every schedulable upstream. The protocol router,
			// rather than group.platform, is the authority for message compatibility.
			accounts, err = filterAccountsByRequestProtocolForScheduling(ctx, groupID, platform, accounts)
			slog.Debug("account_scheduling_list_snapshot",
				"group_id", derefGroupID(groupID),
				"platform", platform,
				"use_mixed", useMixed,
				"count", len(accounts))
			for _, acc := range accounts {
				slog.Debug("account_scheduling_account_detail",
					"account_id", acc.ID,
					"name", acc.Name,
					"platform", acc.Platform,
					"type", acc.Type,
					"status", acc.Status,
					"tls_fingerprint", acc.IsTLSFingerprintEnabled())
			}
		}
		return accounts, useMixed, err
	}
	useMixed := (platform == PlatformAnthropic || platform == PlatformGemini) && !hasForcePlatform
	if groupID != nil {
		accounts, err := s.accountRepo.ListSchedulableByGroupID(ctx, *groupID)
		if err != nil {
			slog.Debug("account_scheduling_list_failed",
				"group_id", derefGroupID(groupID),
				"platform", platform,
				"error", err)
			return nil, useMixed, err
		}
		rawCount := len(accounts)
		accounts, err = filterAccountsByRequestProtocolForScheduling(ctx, groupID, platform, accounts)
		if err != nil {
			return nil, useMixed, err
		}
		slog.Debug("account_scheduling_list",
			"group_id", derefGroupID(groupID),
			"platform", platform,
			"use_mixed", useMixed,
			"raw_count", rawCount,
			"filtered_count", len(accounts))
		return accounts, useMixed, nil
	}
	// 解析需查询的 DB platform 列表（Antigravity 账号现位于 gemini 平台之下，
	// 故强制 antigravity / anthropic 混合等场景需把别名翻译为实际 platform）。
	queryPlatforms := schedulingQueryPlatformsForRequest(ctx, platform, useMixed)
	var accounts []Account
	var err error
	if useMixed {
		if groupID != nil {
			accounts, err = s.accountRepo.ListSchedulableByGroupIDAndPlatforms(ctx, *groupID, queryPlatforms)
		} else if s.cfg != nil && s.cfg.RunMode == config.RunModeSimple {
			accounts, err = s.accountRepo.ListSchedulableByPlatforms(ctx, queryPlatforms)
		} else {
			accounts, err = s.accountRepo.ListSchedulableUngroupedByPlatforms(ctx, queryPlatforms)
		}
	} else {
		if len(queryPlatforms) == 1 {
			qp := queryPlatforms[0]
			if s.cfg != nil && s.cfg.RunMode == config.RunModeSimple {
				accounts, err = s.accountRepo.ListSchedulableByPlatform(ctx, qp)
			} else if groupID != nil {
				accounts, err = s.accountRepo.ListSchedulableByGroupIDAndPlatform(ctx, *groupID, qp)
				// 分组内无账号则返回空列表，由上层处理错误，不再回退到全平台查询
			} else {
				accounts, err = s.accountRepo.ListSchedulableUngroupedByPlatform(ctx, qp)
			}
		} else if s.cfg != nil && s.cfg.RunMode == config.RunModeSimple {
			accounts, err = s.accountRepo.ListSchedulableByPlatforms(ctx, queryPlatforms)
		} else if groupID != nil {
			accounts, err = s.accountRepo.ListSchedulableByGroupIDAndPlatforms(ctx, *groupID, queryPlatforms)
			// 分组内无账号则返回空列表，由上层处理错误，不再回退到全平台查询
		} else {
			accounts, err = s.accountRepo.ListSchedulableUngroupedByPlatforms(ctx, queryPlatforms)
		}
	}
	if err != nil {
		slog.Debug("account_scheduling_list_failed",
			"group_id", derefGroupID(groupID),
			"platform", platform,
			"error", err)
		return nil, useMixed, err
	}
	// 按目标平台别名 + 是否混合调度过滤（统一处理 Gemini/Antigravity 合并后的成员归属）。
	filtered := make([]Account, 0, len(accounts))
	for i := range accounts {
		if accountServesRequestPlatform(ctx, &accounts[i], platform, useMixed) {
			filtered = append(filtered, accounts[i])
		}
	}
	// 请求级协议过滤（按当前入站 endpoint 推导）；如果全部账号在这里
	// 被淘汰，返回包含逐账号原因的错误，而不是无信息的空候选集。
	filtered, err = filterAccountsByRequestProtocolForScheduling(ctx, groupID, platform, filtered)
	if err != nil {
		return nil, useMixed, err
	}
	slog.Debug("account_scheduling_list",
		"group_id", derefGroupID(groupID),
		"platform", platform,
		"use_mixed", useMixed,
		"raw_count", len(accounts),
		"filtered_count", len(filtered))
	for i := range filtered {
		acc := &filtered[i]
		slog.Debug("account_scheduling_account_detail",
			"account_id", acc.ID,
			"name", acc.Name,
			"platform", acc.Platform,
			"sub_platform", acc.SubPlatform,
			"type", acc.Type,
			"status", acc.Status,
			"tls_fingerprint", acc.IsTLSFingerprintEnabled())
	}
	return filtered, useMixed, nil
}

// IsSingleAntigravityAccountGroup 检查指定分组是否只有一个 antigravity 平台的可调度账号。
// 用于 Handler 层在首次请求时提前设置 SingleAccountRetry context，
// 避免单账号分组收到 503 时错误地设置模型限流标记导致后续请求连续快速失败。
func (s *GatewayService) IsSingleAntigravityAccountGroup(ctx context.Context, groupID *int64) bool {
	accounts, _, err := s.listSchedulableAccounts(ctx, groupID, PlatformAntigravity, true)
	if err != nil {
		return false
	}
	return len(accounts) == 1
}

func (s *GatewayService) isAccountAllowedForRequest(ctx context.Context, account *Account, groupID *int64, platform string, useMixed bool) bool {
	if account == nil || !accountMatchesRequestProtocol(ctx, account) {
		return false
	}
	// 分组只限定候选账号范围，不能绕过请求级平台约束。消息协议的 Router
	// 跨平台能力、强制平台以及 mixed_scheduling 均由该统一判定处理。
	return accountServesRequestPlatform(ctx, account, platform, useMixed)
}

func (s *GatewayService) isAccountSchedulableForSelection(account *Account) bool {
	if account == nil {
		return false
	}
	return account.IsSchedulable()
}

func (s *GatewayService) isAccountSchedulableForModelSelection(ctx context.Context, account *Account, requestedModel string) bool {
	if account == nil {
		return false
	}
	return account.IsSchedulableForModelWithContext(ctx, requestedModel)
}

// isAccountInGroup checks if the account belongs to the specified group.
// When groupID is nil, returns true only for ungrouped accounts (no group assignments).
func (s *GatewayService) isAccountInGroup(account *Account, groupID *int64) bool {
	if account == nil {
		return false
	}
	if groupID == nil {
		// 无分组的 API Key 只能使用未分组的账号
		return len(account.AccountGroups) == 0
	}
	for _, ag := range account.AccountGroups {
		if ag.GroupID == *groupID {
			return true
		}
	}
	return false
}

func (s *GatewayService) tryAcquireAccountSlot(ctx context.Context, accountID int64, maxConcurrency int) (*AcquireResult, error) {
	if s.concurrencyService == nil {
		return &AcquireResult{Acquired: true, ReleaseFunc: func() {}}, nil
	}
	return s.concurrencyService.AcquireAccountSlot(ctx, accountID, maxConcurrency)
}
