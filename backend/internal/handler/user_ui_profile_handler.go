package handler

import (
	"encoding/json"
	"io"

	"github.com/WilliamWang1721/LightBridge/internal/pkg/response"
	middleware2 "github.com/WilliamWang1721/LightBridge/internal/server/middleware"

	"github.com/gin-gonic/gin"
)

// GetUIProfile returns the authenticated user's UI profile overrides.
func (h *UserHandler) GetUIProfile(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	profile, err := h.userService.GetUserUIProfile(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"profile": profile})
}

// UpdateUIProfile validates and saves the authenticated user's UI profile.
func (h *UserHandler) UpdateUIProfile(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	body, err := io.ReadAll(io.LimitReader(c.Request.Body, (16<<10)+1))
	if err != nil {
		response.BadRequest(c, "Invalid UI profile request")
		return
	}
	profile, err := h.userService.UpdateUserUIProfile(c.Request.Context(), subject.UserID, json.RawMessage(body))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"profile": profile})
}

// ResetUIProfile removes account-level overrides so lower-precedence defaults apply.
func (h *UserHandler) ResetUIProfile(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	if err := h.userService.ResetUserUIProfile(c.Request.Context(), subject.UserID); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"profile": gin.H{}})
}
