package handler

import (
	"mime"
	"net/http"
	"strconv"

	"github.com/WilliamWang1721/LightBridge/internal/pkg/pagination"
	"github.com/WilliamWang1721/LightBridge/internal/pkg/response"
	middleware2 "github.com/WilliamWang1721/LightBridge/internal/server/middleware"
	"github.com/WilliamWang1721/LightBridge/internal/service"
	"github.com/gin-gonic/gin"
)

type DistributionHandler struct {
	service *service.DistributionService
}

func NewDistributionHandler(distributionService *service.DistributionService) *DistributionHandler {
	return &DistributionHandler{service: distributionService}
}

func (h *DistributionHandler) List(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not found in context")
		return
	}
	page, pageSize := response.ParsePagination(c)
	items, paginationResult, err := h.service.ListForUser(
		c.Request.Context(),
		subject.UserID,
		pagination.PaginationParams{Page: page, PageSize: pageSize},
		parseBoolQuery(c.Query("unread_only")),
	)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Paginated(c, items, paginationResult.Total, paginationResult.Page, paginationResult.PageSize)
}

func (h *DistributionHandler) Get(c *gin.Context) {
	subject, id, ok := distributionSubjectAndID(c)
	if !ok {
		return
	}
	item, err := h.service.GetForUser(c.Request.Context(), id, subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, item)
}

func (h *DistributionHandler) MarkRead(c *gin.Context) {
	subject, id, ok := distributionSubjectAndID(c)
	if !ok {
		return
	}
	if err := h.service.MarkRead(c.Request.Context(), id, subject.UserID); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"message": "ok"})
}

func (h *DistributionHandler) Download(c *gin.Context) {
	subject, id, ok := distributionSubjectAndID(c)
	if !ok {
		return
	}
	data, attachment, err := h.service.DownloadForUser(c.Request.Context(), id, subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	contentType := attachment.ContentType
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	c.Header("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": attachment.FileName}))
	c.Header("X-Content-Type-Options", "nosniff")
	c.Data(http.StatusOK, contentType, data)
}

func distributionSubjectAndID(c *gin.Context) (middleware2.AuthSubject, int64, bool) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not found in context")
		return middleware2.AuthSubject{}, 0, false
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "Invalid distribution ID")
		return middleware2.AuthSubject{}, 0, false
	}
	return subject, id, true
}
