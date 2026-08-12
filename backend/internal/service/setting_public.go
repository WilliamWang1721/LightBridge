package service

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

// GetAllSettings 获取所有系统设置
func (s *SettingService) GetAllSettings(ctx context.Context) (*SystemSettings, error) {
	settings, err := s.settingRepo.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("get all settings: %w", err)
	}

	return s.parseSettings(settings), nil
}

// GetFrontendURL 获取前端基础URL（数据库优先，fallback 到配置文件）
func (s *SettingService) GetFrontendURL(ctx context.Context) string {
	val, err := s.settingRepo.GetValue(ctx, SettingKeyFrontendURL)
	if err == nil && strings.TrimSpace(val) != "" {
		return strings.TrimSpace(val)
	}
	return s.cfg.Server.FrontendURL
}

// GetPublicSettings 获取公开设置（无需登录）
func (s *SettingService) GetPublicSettings(ctx context.Context) (*PublicSettings, error) {
	keys := []string{
		SettingKeyRegistrationEnabled,
		SettingKeyEmailVerifyEnabled,
		SettingKeyForceEmailOnThirdPartySignup,
		SettingKeyRegistrationEmailSuffixWhitelist,
		SettingKeyPromoCodeEnabled,
		SettingKeyPasswordResetEnabled,
		SettingKeyInvitationCodeEnabled,
		SettingKeyTotpEnabled,
		SettingKeyLoginAgreementEnabled,
		SettingKeyLoginAgreementMode,
		SettingKeyLoginAgreementUpdatedAt,
		SettingKeyLoginAgreementDocuments,
		SettingKeyTurnstileEnabled,
		SettingKeyTurnstileSiteKey,
		SettingKeyAPIKeyACLTrustForwardedIP,
		SettingKeySiteName,
		SettingKeySiteLogo,
		SettingKeySiteSubtitle,
		SettingKeyAPIBaseURL,
		SettingKeyContactInfo,
		SettingKeyDocURL,
		SettingKeyHomeContent,
		SettingKeyHideCcsImportButton,
		SettingKeyPurchaseSubscriptionEnabled,
		SettingKeyPurchaseSubscriptionURL,
		SettingKeyTableDefaultPageSize,
		SettingKeyTablePageSizeOptions,
		SettingKeyCustomMenuItems,
		SettingKeyCustomEndpoints,
		SettingKeyUIAppearance,
		SettingKeyLinuxDoConnectEnabled,
		SettingKeyDingTalkConnectEnabled,
		SettingKeyWeChatConnectEnabled,
		SettingKeyWeChatConnectAppID,
		SettingKeyWeChatConnectAppSecret,
		SettingKeyWeChatConnectOpenAppID,
		SettingKeyWeChatConnectOpenAppSecret,
		SettingKeyWeChatConnectMPAppID,
		SettingKeyWeChatConnectMPAppSecret,
		SettingKeyWeChatConnectMobileAppID,
		SettingKeyWeChatConnectMobileAppSecret,
		SettingKeyWeChatConnectOpenEnabled,
		SettingKeyWeChatConnectMPEnabled,
		SettingKeyWeChatConnectMobileEnabled,
		SettingKeyWeChatConnectMode,
		SettingKeyWeChatConnectScopes,
		SettingKeyWeChatConnectRedirectURL,
		SettingKeyWeChatConnectFrontendRedirectURL,
		SettingKeyBackendModeEnabled,
		SettingPaymentEnabled,
		SettingKeyOIDCConnectEnabled,
		SettingKeyOIDCConnectProviderName,
		SettingKeyGitHubOAuthEnabled,
		SettingKeyGitHubOAuthClientID,
		SettingKeyGitHubOAuthClientSecret,
		SettingKeyGoogleOAuthEnabled,
		SettingKeyGoogleOAuthClientID,
		SettingKeyGoogleOAuthClientSecret,
		SettingKeyBalanceLowNotifyEnabled,
		SettingKeyBalanceLowNotifyThreshold,
		SettingKeyBalanceLowNotifyRechargeURL,
		SettingKeyAccountQuotaNotifyEnabled,
		SettingKeyChannelMonitorEnabled,
		SettingKeyChannelMonitorDefaultIntervalSeconds,
		SettingKeyAvailableChannelsEnabled,
		SettingKeyAffiliateEnabled,
		SettingKeyRiskControlEnabled,
		SettingKeyPrivacyFilterEnabled,
		SettingKeyDeploymentMode,
	}

	settings, err := s.settingRepo.GetMultiple(ctx, keys)
	if err != nil {
		return nil, fmt.Errorf("get public settings: %w", err)
	}

	linuxDoEnabled := false
	if raw, ok := settings[SettingKeyLinuxDoConnectEnabled]; ok {
		linuxDoEnabled = raw == "true"
	} else {
		linuxDoEnabled = s.cfg != nil && s.cfg.LinuxDo.Enabled
	}
	dingTalkEnabled := false
	if raw, ok := settings[SettingKeyDingTalkConnectEnabled]; ok {
		dingTalkEnabled = raw == "true"
	} else {
		dingTalkEnabled = s.cfg != nil && s.cfg.DingTalk.Enabled
	}
	oidcEnabled := false
	if raw, ok := settings[SettingKeyOIDCConnectEnabled]; ok {
		oidcEnabled = raw == "true"
	} else {
		oidcEnabled = s.cfg != nil && s.cfg.OIDC.Enabled
	}
	oidcProviderName := strings.TrimSpace(settings[SettingKeyOIDCConnectProviderName])
	if oidcProviderName == "" && s.cfg != nil {
		oidcProviderName = strings.TrimSpace(s.cfg.OIDC.ProviderName)
	}
	if oidcProviderName == "" {
		oidcProviderName = "OIDC"
	}
	gitHubEnabled := s.emailOAuthPublicEnabled(settings, "github")
	googleEnabled := s.emailOAuthPublicEnabled(settings, "google")
	weChatEnabled, weChatOpenEnabled, weChatMPEnabled, weChatMobileEnabled := s.weChatOAuthCapabilitiesFromSettings(settings)

	// Password reset requires email verification to be enabled
	emailVerifyEnabled := settings[SettingKeyEmailVerifyEnabled] == "true"
	passwordResetEnabled := emailVerifyEnabled && settings[SettingKeyPasswordResetEnabled] == "true"
	registrationEmailSuffixWhitelist := ParseRegistrationEmailSuffixWhitelist(
		settings[SettingKeyRegistrationEmailSuffixWhitelist],
	)
	tableDefaultPageSize, tablePageSizeOptions := parseTablePreferences(
		settings[SettingKeyTableDefaultPageSize],
		settings[SettingKeyTablePageSizeOptions],
	)
	loginAgreementDocuments := parseLoginAgreementDocuments(settings[SettingKeyLoginAgreementDocuments])
	loginAgreementUpdatedAt := strings.TrimSpace(settings[SettingKeyLoginAgreementUpdatedAt])
	if loginAgreementUpdatedAt == "" {
		loginAgreementUpdatedAt = defaultLoginAgreementDate
	}

	var balanceLowNotifyThreshold float64
	if v, err := strconv.ParseFloat(settings[SettingKeyBalanceLowNotifyThreshold], 64); err == nil && v >= 0 {
		balanceLowNotifyThreshold = v
	}

	var uiAppearance *RuntimeAppearanceSettings
	if raw := settings[SettingKeyUIAppearance]; raw != "" {
		// ponytail: malformed appearance data fails closed here so public boot stays available.
		uiAppearance, _ = ParseRuntimeAppearance(raw)
	}

	return &PublicSettings{
		RegistrationEnabled:              settings[SettingKeyRegistrationEnabled] == "true",
		EmailVerifyEnabled:               emailVerifyEnabled,
		ForceEmailOnThirdPartySignup:     settings[SettingKeyForceEmailOnThirdPartySignup] == "true",
		RegistrationEmailSuffixWhitelist: registrationEmailSuffixWhitelist,
		PromoCodeEnabled:                 s.IsProgressiveFeatureEnabled(ctx, ProgressiveFeaturePromo),
		PasswordResetEnabled:             passwordResetEnabled,
		InvitationCodeEnabled:            settings[SettingKeyInvitationCodeEnabled] == "true",
		TotpEnabled:                      settings[SettingKeyTotpEnabled] == "true",
		LoginAgreementEnabled:            settings[SettingKeyLoginAgreementEnabled] == "true" && len(loginAgreementDocuments) > 0,
		LoginAgreementMode:               normalizeLoginAgreementMode(settings[SettingKeyLoginAgreementMode]),
		LoginAgreementUpdatedAt:          loginAgreementUpdatedAt,
		LoginAgreementRevision:           buildLoginAgreementRevision(loginAgreementUpdatedAt, loginAgreementDocuments),
		LoginAgreementDocuments:          loginAgreementDocuments,
		TurnstileEnabled:                 settings[SettingKeyTurnstileEnabled] == "true",
		TurnstileSiteKey:                 settings[SettingKeyTurnstileSiteKey],
		SiteName:                         s.getStringOrDefault(settings, SettingKeySiteName, "LightBridge"),
		SiteLogo:                         settings[SettingKeySiteLogo],
		SiteSubtitle:                     s.getStringOrDefault(settings, SettingKeySiteSubtitle, "Subscription to API Conversion Platform"),
		APIBaseURL:                       settings[SettingKeyAPIBaseURL],
		ContactInfo:                      settings[SettingKeyContactInfo],
		DocURL:                           settings[SettingKeyDocURL],
		HomeContent:                      settings[SettingKeyHomeContent],
		HideCcsImportButton:              settings[SettingKeyHideCcsImportButton] == "true",
		PurchaseSubscriptionEnabled:      settings[SettingKeyPurchaseSubscriptionEnabled] == "true",
		PurchaseSubscriptionURL:          strings.TrimSpace(settings[SettingKeyPurchaseSubscriptionURL]),
		TableDefaultPageSize:             tableDefaultPageSize,
		TablePageSizeOptions:             tablePageSizeOptions,
		CustomMenuItems:                  settings[SettingKeyCustomMenuItems],
		CustomEndpoints:                  settings[SettingKeyCustomEndpoints],
		UIAppearance:                     uiAppearance,
		LinuxDoOAuthEnabled:              linuxDoEnabled,
		DingTalkOAuthEnabled:             dingTalkEnabled,
		WeChatOAuthEnabled:               weChatEnabled,
		WeChatOAuthOpenEnabled:           weChatOpenEnabled,
		WeChatOAuthMPEnabled:             weChatMPEnabled,
		WeChatOAuthMobileEnabled:         weChatMobileEnabled,
		BackendModeEnabled:               settings[SettingKeyBackendModeEnabled] == "true",
		PaymentEnabled:                   s.IsProgressiveFeatureEnabled(ctx, ProgressiveFeaturePayment),
		OIDCOAuthEnabled:                 oidcEnabled,
		OIDCOAuthProviderName:            oidcProviderName,
		GitHubOAuthEnabled:               gitHubEnabled,
		GoogleOAuthEnabled:               googleEnabled,
		BalanceLowNotifyEnabled:          settings[SettingKeyBalanceLowNotifyEnabled] == "true",
		AccountQuotaNotifyEnabled:        settings[SettingKeyAccountQuotaNotifyEnabled] == "true",
		BalanceLowNotifyThreshold:        balanceLowNotifyThreshold,
		BalanceLowNotifyRechargeURL:      settings[SettingKeyBalanceLowNotifyRechargeURL],

		ChannelMonitorEnabled:                s.IsProgressiveFeatureEnabled(ctx, ProgressiveFeatureChannelMonitor),
		ChannelMonitorDefaultIntervalSeconds: parseChannelMonitorInterval(settings[SettingKeyChannelMonitorDefaultIntervalSeconds]),

		AvailableChannelsEnabled: s.IsProgressiveFeatureEnabled(ctx, ProgressiveFeatureAvailableChannels),

		AffiliateEnabled: s.IsProgressiveFeatureEnabled(ctx, ProgressiveFeatureAffiliate),

		RiskControlEnabled:   s.IsProgressiveFeatureEnabled(ctx, ProgressiveFeatureRiskControl),
		PrivacyFilterEnabled: s.IsProgressiveFeatureEnabled(ctx, ProgressiveFeaturePrivacyFilter),

		AnnouncementsEnabled:  s.IsProgressiveFeatureEnabled(ctx, ProgressiveFeatureAnnouncements),
		RedeemEnabled:         s.IsProgressiveFeatureEnabled(ctx, ProgressiveFeatureRedeem),
		PromoEnabled:          s.IsProgressiveFeatureEnabled(ctx, ProgressiveFeaturePromo),
		ProxiesEnabled:        s.IsProgressiveFeatureEnabled(ctx, ProgressiveFeatureProxies),
		ChannelPricingEnabled: s.IsProgressiveFeatureEnabled(ctx, ProgressiveFeatureChannelPricing),

		DeploymentMode: NormalizeDeploymentMode(settings[SettingKeyDeploymentMode]),
	}, nil
}
