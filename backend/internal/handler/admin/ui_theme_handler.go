package admin

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/WilliamWang1721/LightBridge/internal/pkg/response"
	"github.com/WilliamWang1721/LightBridge/internal/service"
	"github.com/gin-gonic/gin"
)

type UIThemeHandler struct {
	service *service.UIThemeService
}

func NewUIThemeHandler(s *service.UIThemeService) *UIThemeHandler {
	return &UIThemeHandler{service: s}
}

func (h *UIThemeHandler) List(c *gin.Context) {
	items, err := h.service.List(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"themes": items})
}

func (h *UIThemeHandler) Upload(c *gin.Context) {
	replace := strings.EqualFold(c.Query("replace"), "true")
	file, err := c.FormFile("file")
	if err != nil {
		response.BadRequest(c, "file is required")
		return
	}
	if file.Size > 10<<20 {
		response.BadRequest(c, "theme zip exceeds 10MB")
		return
	}
	src, err := file.Open()
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to open uploaded file")
		return
	}
	defer func() { _ = src.Close() }()
	data, err := io.ReadAll(io.LimitReader(src, 10<<20+1))
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to read uploaded file")
		return
	}
	result, err := h.service.InstallZIP(c.Request.Context(), data, "upload:"+file.Filename, replace)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Created(c, result)
}

type importGitHubThemeRequest struct {
	URL     string `json:"url"`
	Replace bool   `json:"replace"`
}

func (h *UIThemeHandler) ImportGitHub(c *gin.Context) {
	var req importGitHubThemeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request: "+err.Error())
		return
	}
	result, err := h.service.ImportGitHub(c.Request.Context(), req.URL, req.Replace)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Created(c, result)
}

func (h *UIThemeHandler) Activate(c *gin.Context) {
	theme, err := h.service.Activate(c.Request.Context(), c.Param("id"))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, theme)
}

func (h *UIThemeHandler) Deactivate(c *gin.Context) {
	theme, err := h.service.Deactivate(c.Request.Context(), c.Param("id"))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, theme)
}

type updateThemeConfigRequest struct {
	Config json.RawMessage `json:"config"`
}

func (h *UIThemeHandler) UpdateConfig(c *gin.Context) {
	var req updateThemeConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request: "+err.Error())
		return
	}

	ctx := c.Request.Context()
	theme, err := h.service.Get(ctx, c.Param("id"))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	normalized, err := service.ValidateUIThemeConfig(theme, req.Config)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	updated, err := h.service.UpdateConfig(ctx, c.Param("id"), normalized)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, updated)
}

func (h *UIThemeHandler) Delete(c *gin.Context) {
	if err := h.service.Delete(c.Request.Context(), c.Param("id")); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"deleted": true})
}
