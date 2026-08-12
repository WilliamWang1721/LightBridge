package admin

import (
	"github.com/WilliamWang1721/LightBridge/internal/pkg/response"
	"github.com/gin-gonic/gin"
)

// GetRuntimeAppearance returns the installation-level runtime appearance.
// GET /api/v1/admin/settings/appearance
func (h *SettingHandler) GetRuntimeAppearance(c *gin.Context) {
	appearance, err := h.settingService.GetRuntimeAppearance(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"appearance": appearance})
}

type UpdateRuntimeAppearanceRequest struct {
	PresetCode string `json:"preset_code"`
}

// UpdateRuntimeAppearance saves the installation-level runtime appearance.
// PUT /api/v1/admin/settings/appearance
func (h *SettingHandler) UpdateRuntimeAppearance(c *gin.Context) {
	var req UpdateRuntimeAppearanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	appearance, err := h.settingService.SetRuntimeAppearance(c.Request.Context(), req.PresetCode)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"appearance": appearance})
}

// ResetRuntimeAppearance removes the installation-level override.
// DELETE /api/v1/admin/settings/appearance
func (h *SettingHandler) ResetRuntimeAppearance(c *gin.Context) {
	if err := h.settingService.ResetRuntimeAppearance(c.Request.Context()); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"appearance": nil})
}
