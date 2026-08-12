package handler

import (
	"html"
	"net/http"
	"strings"

	"github.com/WilliamWang1721/LightBridge/internal/handler/dto"
	"github.com/WilliamWang1721/LightBridge/internal/pkg/response"
	"github.com/WilliamWang1721/LightBridge/internal/service"

	"github.com/gin-gonic/gin"
)

// SettingHandler 公开设置处理器（无需认证）
type SettingHandler struct {
	settingService           *service.SettingService
	notificationEmailService *service.NotificationEmailService
	featureRuntime           *service.FeatureRuntimeManager
	version                  string
}

// NewSettingHandler 创建公开设置处理器
func NewSettingHandler(settingService *service.SettingService, version string) *SettingHandler {
	return &SettingHandler{
		settingService: settingService,
		version:        version,
	}
}

// SetNotificationEmailService attaches the public notification email service without
// changing the constructor signature used by existing tests.
func (h *SettingHandler) SetNotificationEmailService(notificationEmailService *service.NotificationEmailService) {
	h.notificationEmailService = notificationEmailService
}

// SetFeatureRuntime attaches administrator-only runtime diagnostics while the
// public manifest remains limited to stable feature availability data.
func (h *SettingHandler) SetFeatureRuntime(featureRuntime *service.FeatureRuntimeManager) {
	h.featureRuntime = featureRuntime
}

// GetFeatureManifest returns the public progressive feature catalog.
// GET /api/v1/settings/features
func (h *SettingHandler) GetFeatureManifest(c *gin.Context) {
	response.Success(c, h.settingService.ProgressiveFeatureManifest(c.Request.Context()))
}

// GetFeatureRuntimeStatus returns optional worker diagnostics. The route is
// registered only below the administrator authentication middleware.
// GET /api/v1/admin/features/runtime
func (h *SettingHandler) GetFeatureRuntimeStatus(c *gin.Context) {
	if h.featureRuntime == nil {
		response.Success(c, []service.FeatureRuntimeComponentStatus{})
		return
	}
	response.Success(c, h.featureRuntime.Status())
}

// GetProgressiveFeatureControls returns the unified administrator registration
// and runtime view. It is deliberately registered outside feature guards.
// GET /api/v1/admin/features
func (h *SettingHandler) GetProgressiveFeatureControls(c *gin.Context) {
	response.Success(c, h.progressiveFeatureControlOverview(c))
}

type updateProgressiveFeatureControlRequest struct {
	Enabled *bool `json:"enabled" binding:"required"`
}

// UpdateProgressiveFeatureControl persists an explicit per-feature override.
// PUT /api/v1/admin/features/:id
func (h *SettingHandler) UpdateProgressiveFeatureControl(c *gin.Context) {
	var req updateProgressiveFeatureControlRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Enabled == nil {
		response.BadRequest(c, "enabled must be a boolean")
		return
	}
	if err := h.settingService.SetProgressiveFeatureOverride(
		c.Request.Context(),
		service.ProgressiveFeature(strings.TrimSpace(c.Param("id"))),
		req.Enabled,
	); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, h.progressiveFeatureControlOverview(c))
}

// ResetProgressiveFeatureControl removes the database override and restores
// inherited profile/configuration behavior.
// DELETE /api/v1/admin/features/:id/override
func (h *SettingHandler) ResetProgressiveFeatureControl(c *gin.Context) {
	if err := h.settingService.SetProgressiveFeatureOverride(
		c.Request.Context(),
		service.ProgressiveFeature(strings.TrimSpace(c.Param("id"))),
		nil,
	); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, h.progressiveFeatureControlOverview(c))
}

func (h *SettingHandler) progressiveFeatureControlOverview(c *gin.Context) service.ProgressiveFeatureControlOverview {
	var runtime []service.FeatureRuntimeComponentStatus
	if h.featureRuntime != nil {
		runtime = h.featureRuntime.Status()
	}
	return h.settingService.ProgressiveFeatureControlOverview(c.Request.Context(), runtime)
}

// GetPublicSettings 获取公开设置
// GET /api/v1/settings/public
func (h *SettingHandler) GetPublicSettings(c *gin.Context) {
	settings, err := h.settingService.GetPublicSettings(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, dto.PublicSettings{
		RegistrationEnabled:              settings.RegistrationEnabled,
		EmailVerifyEnabled:               settings.EmailVerifyEnabled,
		ForceEmailOnThirdPartySignup:     settings.ForceEmailOnThirdPartySignup,
		RegistrationEmailSuffixWhitelist: settings.RegistrationEmailSuffixWhitelist,
		PromoCodeEnabled:                 settings.PromoCodeEnabled,
		PasswordResetEnabled:             settings.PasswordResetEnabled,
		InvitationCodeEnabled:            settings.InvitationCodeEnabled,
		TotpEnabled:                      settings.TotpEnabled,
		LoginAgreementEnabled:            settings.LoginAgreementEnabled,
		LoginAgreementMode:               settings.LoginAgreementMode,
		LoginAgreementUpdatedAt:          settings.LoginAgreementUpdatedAt,
		LoginAgreementRevision:           settings.LoginAgreementRevision,
		LoginAgreementDocuments:          publicLoginAgreementDocumentsToDTO(settings.LoginAgreementDocuments),
		TurnstileEnabled:                 settings.TurnstileEnabled,
		TurnstileSiteKey:                 settings.TurnstileSiteKey,
		SiteName:                         settings.SiteName,
		SiteLogo:                         settings.SiteLogo,
		SiteSubtitle:                     settings.SiteSubtitle,
		APIBaseURL:                       settings.APIBaseURL,
		ContactInfo:                      settings.ContactInfo,
		DocURL:                           settings.DocURL,
		HomeContent:                      settings.HomeContent,
		HideCcsImportButton:              settings.HideCcsImportButton,
		PurchaseSubscriptionEnabled:      settings.PurchaseSubscriptionEnabled,
		PurchaseSubscriptionURL:          settings.PurchaseSubscriptionURL,
		TableDefaultPageSize:             settings.TableDefaultPageSize,
		TablePageSizeOptions:             settings.TablePageSizeOptions,
		CustomMenuItems:                  dto.ParseUserVisibleMenuItems(settings.CustomMenuItems),
		CustomEndpoints:                  dto.ParseCustomEndpoints(settings.CustomEndpoints),
		UIAppearance:                     settings.UIAppearance,
		DingTalkOAuthEnabled:             settings.DingTalkOAuthEnabled,
		LinuxDoOAuthEnabled:              settings.LinuxDoOAuthEnabled,
		WeChatOAuthEnabled:               settings.WeChatOAuthEnabled,
		WeChatOAuthOpenEnabled:           settings.WeChatOAuthOpenEnabled,
		WeChatOAuthMPEnabled:             settings.WeChatOAuthMPEnabled,
		WeChatOAuthMobileEnabled:         settings.WeChatOAuthMobileEnabled,
		OIDCOAuthEnabled:                 settings.OIDCOAuthEnabled,
		OIDCOAuthProviderName:            settings.OIDCOAuthProviderName,
		GitHubOAuthEnabled:               settings.GitHubOAuthEnabled,
		GoogleOAuthEnabled:               settings.GoogleOAuthEnabled,
		BackendModeEnabled:               settings.BackendModeEnabled,
		PaymentEnabled:                   settings.PaymentEnabled,
		Version:                          h.version,
		BalanceLowNotifyEnabled:          settings.BalanceLowNotifyEnabled,
		AccountQuotaNotifyEnabled:        settings.AccountQuotaNotifyEnabled,
		BalanceLowNotifyThreshold:        settings.BalanceLowNotifyThreshold,
		BalanceLowNotifyRechargeURL:      settings.BalanceLowNotifyRechargeURL,

		ChannelMonitorEnabled:                settings.ChannelMonitorEnabled,
		ChannelMonitorDefaultIntervalSeconds: settings.ChannelMonitorDefaultIntervalSeconds,

		AvailableChannelsEnabled: settings.AvailableChannelsEnabled,

		AffiliateEnabled: settings.AffiliateEnabled,

		RiskControlEnabled: settings.RiskControlEnabled,

		PrivacyFilterEnabled: settings.PrivacyFilterEnabled,

		DeploymentMode: settings.DeploymentMode,
	})
}

// UnsubscribeNotificationEmail handles optional notification email opt-outs.
// GET /api/v1/settings/email-unsubscribe?token=...
func (h *SettingHandler) UnsubscribeNotificationEmail(c *gin.Context) {
	if h.notificationEmailService == nil {
		response.InternalError(c, "notification email service is not configured")
		return
	}
	token := strings.TrimSpace(c.Query("token"))
	if token == "" {
		response.BadRequest(c, "token is required")
		return
	}
	result, err := h.notificationEmailService.Unsubscribe(c.Request.Context(), token)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	body := "<!doctype html><html><head><meta charset=\"utf-8\"><title>Unsubscribed</title></head><body style=\"font-family:-apple-system,BlinkMacSystemFont,Segoe UI,sans-serif;padding:32px;\"><h1>Unsubscribed</h1><p>You have unsubscribed <strong>" + html.EscapeString(result.Email) + "</strong> from <strong>" + html.EscapeString(result.Event) + "</strong> emails.</p></body></html>"
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(body))
}

func publicLoginAgreementDocumentsToDTO(items []service.LoginAgreementDocument) []dto.LoginAgreementDocument {
	result := make([]dto.LoginAgreementDocument, 0, len(items))
	for _, item := range items {
		result = append(result, dto.LoginAgreementDocument{
			ID:        item.ID,
			Title:     item.Title,
			ContentMD: item.ContentMD,
		})
	}
	return result
}
