package admin

import (
	"encoding/json"
	"io"
	"mime"
	"net/http"
	"strconv"
	"strings"

	"github.com/WilliamWang1721/LightBridge/internal/pkg/pagination"
	"github.com/WilliamWang1721/LightBridge/internal/pkg/response"
	"github.com/WilliamWang1721/LightBridge/internal/service"
	"github.com/gin-gonic/gin"
)

const (
	// Base64 expands binary data to four bytes for every three bytes. Keep enough
	// JSON headroom so an attachment at the advertised 10 MiB limit is accepted.
	maxDistributionJSONBodyBytes int64 = ((service.MaxDistributionAttachmentBytes + 2) / 3 * 4) + (2 << 20)
	distributionAudienceModeError       = "audience must use exactly one non-empty mode: explicit recipients, filters, lines, or all"
)

type DistributionHandler struct {
	service *service.DistributionService
}

func NewDistributionHandler(distributionService *service.DistributionService) *DistributionHandler {
	return &DistributionHandler{service: distributionService}
}

func (h *DistributionHandler) Create(c *gin.Context) {
	input, err := parseCreateDistributionInput(c)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	if !validDistributionAudience(input.Audience) {
		response.BadRequest(c, distributionAudienceModeError)
		return
	}
	actorID := getAdminIDFromContext(c)
	if actorID > 0 {
		input.ActorID = &actorID
	}
	created, err := h.service.Create(c.Request.Context(), input)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, created)
}

func (h *DistributionHandler) BatchCreate(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 20<<20)
	var request struct {
		Items []service.CreateDistributionInput `json:"items"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	if len(request.Items) == 0 || len(request.Items) > service.MaxDistributionBatchItems {
		response.BadRequest(c, "items must contain between 1 and 100 entries")
		return
	}
	for i := range request.Items {
		if !validDistributionAudience(request.Items[i].Audience) {
			response.BadRequest(c, distributionAudienceModeError)
			return
		}
	}
	actorID := getAdminIDFromContext(c)
	var actorIDPtr *int64
	if actorID > 0 {
		actorIDPtr = &actorID
	}
	result := h.service.BatchCreate(c.Request.Context(), request.Items, actorIDPtr)
	response.Success(c, result)
}

func (h *DistributionHandler) PreviewAudience(c *gin.Context) {
	var audience service.DistributionAudienceInput
	if err := c.ShouldBindJSON(&audience); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	if !validDistributionAudience(audience) {
		response.BadRequest(c, distributionAudienceModeError)
		return
	}
	users, count, err := h.service.PreviewAudience(c.Request.Context(), audience)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"count": count, "preview": users})
}

func (h *DistributionHandler) List(c *gin.Context) {
	page, pageSize := response.ParsePagination(c)
	items, paginationResult, err := h.service.ListAdmin(c.Request.Context(), pagination.PaginationParams{Page: page, PageSize: pageSize})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Paginated(c, items, paginationResult.Total, paginationResult.Page, paginationResult.PageSize)
}

func (h *DistributionHandler) Get(c *gin.Context) {
	id, ok := parseDistributionID(c)
	if !ok {
		return
	}
	item, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, item)
}

func (h *DistributionHandler) Download(c *gin.Context) {
	id, ok := parseDistributionID(c)
	if !ok {
		return
	}
	data, attachment, err := h.service.DownloadAdmin(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	writeDistributionAttachment(c, data, attachment)
}

func (h *DistributionHandler) Delete(c *gin.Context) {
	id, ok := parseDistributionID(c)
	if !ok {
		return
	}
	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"message": "ok"})
}

func parseCreateDistributionInput(c *gin.Context) (service.CreateDistributionInput, error) {
	contentType := c.GetHeader("Content-Type")
	if !strings.HasPrefix(strings.ToLower(contentType), "multipart/form-data") {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxDistributionJSONBodyBytes)
		var input service.CreateDistributionInput
		if err := c.ShouldBindJSON(&input); err != nil {
			return input, err
		}
		return input, nil
	}

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, service.MaxDistributionAttachmentBytes+(2<<20))
	if err := c.Request.ParseMultipartForm(service.MaxDistributionAttachmentBytes + (1 << 20)); err != nil {
		return service.CreateDistributionInput{}, err
	}
	input := service.CreateDistributionInput{
		Title:       c.PostForm("title"),
		Kind:        c.PostForm("kind"),
		Content:     c.PostForm("content"),
		FileName:    c.PostForm("file_name"),
		ContentType: c.PostForm("content_type"),
	}
	if raw := strings.TrimSpace(c.PostForm("audience")); raw != "" {
		if err := json.Unmarshal([]byte(raw), &input.Audience); err != nil {
			return input, err
		}
	}
	if raw := strings.TrimSpace(c.PostForm("metadata")); raw != "" {
		if err := json.Unmarshal([]byte(raw), &input.Metadata); err != nil {
			return input, err
		}
	}
	file, header, err := c.Request.FormFile("file")
	if err == nil {
		defer func() { _ = file.Close() }()
		data, readErr := io.ReadAll(io.LimitReader(file, service.MaxDistributionAttachmentBytes+1))
		if readErr != nil {
			return input, readErr
		}
		if len(data) > service.MaxDistributionAttachmentBytes {
			return input, service.ErrDistributionTooLarge
		}
		input.FileData = data
		if input.FileName == "" {
			input.FileName = header.Filename
		}
		if input.ContentType == "" {
			input.ContentType = header.Header.Get("Content-Type")
		}
	} else if err != http.ErrMissingFile {
		return input, err
	}
	return input, nil
}

func validDistributionAudience(audience service.DistributionAudienceInput) bool {
	modes := 0
	if audience.All {
		modes++
	}
	if len(audience.UserIDs) > 0 || len(audience.Emails) > 0 {
		modes++
	}
	if strings.TrimSpace(audience.Lines) != "" {
		modes++
	}
	if hasNonEmptyDistributionFilters(audience.Filters) {
		modes++
	} else if audience.Filters != nil {
		return false
	}
	return modes == 1
}

func hasNonEmptyDistributionFilters(filters *service.DistributionUserFilters) bool {
	if filters == nil {
		return false
	}
	return strings.TrimSpace(filters.Status) != "" ||
		strings.TrimSpace(filters.Role) != "" ||
		strings.TrimSpace(filters.Search) != "" ||
		strings.TrimSpace(filters.GroupName) != "" ||
		strings.TrimSpace(filters.Activity) != "" ||
		len(filters.Attributes) > 0
}

func parseDistributionID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "Invalid distribution ID")
		return 0, false
	}
	return id, true
}

func writeDistributionAttachment(c *gin.Context, data []byte, attachment *service.DistributionAttachment) {
	contentType := attachment.ContentType
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	disposition := mime.FormatMediaType("attachment", map[string]string{"filename": attachment.FileName})
	c.Header("Content-Disposition", disposition)
	c.Header("X-Content-Type-Options", "nosniff")
	c.Data(http.StatusOK, contentType, data)
}
