package admin

import (
	"strings"

	"github.com/WilliamWang1721/LightBridge/internal/pkg/response"
	"github.com/WilliamWang1721/LightBridge/internal/server/middleware"
	"github.com/WilliamWang1721/LightBridge/internal/service"
	"github.com/gin-gonic/gin"
)

type BackupHandler struct {
	backupService *service.BackupService
	userService   *service.UserService
}

func NewBackupHandler(backupService *service.BackupService, userService *service.UserService) *BackupHandler {
	return &BackupHandler{backupService: backupService, userService: userService}
}
func (h *BackupHandler) GetS3Config(c *gin.Context) {
	cfg, err := h.backupService.GetS3Config(c.Request.Context())
	if err != nil { response.ErrorFrom(c, err); return }
	response.Success(c, cfg)
}
func (h *BackupHandler) UpdateS3Config(c *gin.Context) {
	var req service.BackupS3Config
	if err := c.ShouldBindJSON(&req); err != nil { response.BadRequest(c, "Invalid request: "+err.Error()); return }
	cfg, err := h.backupService.UpdateS3Config(c.Request.Context(), req)
	if err != nil { response.ErrorFrom(c, err); return }
	response.Success(c, cfg)
}
func (h *BackupHandler) TestS3Connection(c *gin.Context) {
	var req service.BackupS3Config
	if err := c.ShouldBindJSON(&req); err != nil { response.BadRequest(c, "Invalid request: "+err.Error()); return }
	if err := h.backupService.TestS3Connection(c.Request.Context(), req); err != nil { response.Success(c, gin.H{"ok": false, "message": err.Error()}); return }
	response.Success(c, gin.H{"ok": true, "message": "connection successful"})
}
func (h *BackupHandler) GetSchedule(c *gin.Context) {
	cfg, err := h.backupService.GetSchedule(c.Request.Context())
	if err != nil { response.ErrorFrom(c, err); return }
	response.Success(c, cfg)
}
func (h *BackupHandler) UpdateSchedule(c *gin.Context) {
	var req service.BackupScheduleConfig
	if err := c.ShouldBindJSON(&req); err != nil { response.BadRequest(c, "Invalid request: "+err.Error()); return }
	cfg, err := h.backupService.UpdateSchedule(c.Request.Context(), req)
	if err != nil { response.ErrorFrom(c, err); return }
	response.Success(c, cfg)
}

type CreateBackupRequest struct {
	ExpireDays  *int   `json:"expire_days"`
	Destination string `json:"destination"`
}

func (h *BackupHandler) CreateBackup(c *gin.Context) {
	var req CreateBackupRequest
	_ = c.ShouldBindJSON(&req)
	destination := strings.ToLower(strings.TrimSpace(req.Destination))
	if destination == "local" {
		fileName := h.backupService.LocalBackupFileName()
		c.Header("Content-Type", "application/gzip")
		c.Header("Content-Disposition", `attachment; filename="`+fileName+`"`)
		c.Header("Cache-Control", "no-store")
		c.Header("X-Content-Type-Options", "nosniff")
		if err := h.backupService.StreamLocalBackup(c.Request.Context(), c.Writer); err != nil {
			if !c.Writer.Written() { response.ErrorFrom(c, err) }
			return
		}
		return
	}
	if destination != "" && destination != "s3" { response.BadRequest(c, "destination must be s3 or local"); return }
	expireDays := 14
	if req.ExpireDays != nil { expireDays = *req.ExpireDays }
	record, err := h.backupService.StartBackup(c.Request.Context(), "manual", expireDays)
	if err != nil { response.ErrorFrom(c, err); return }
	response.Accepted(c, record)
}
func (h *BackupHandler) ListBackups(c *gin.Context) {
	records, err := h.backupService.ListBackups(c.Request.Context())
	if err != nil { response.ErrorFrom(c, err); return }
	if records == nil { records = []service.BackupRecord{} }
	response.Success(c, gin.H{"items": records})
}
func (h *BackupHandler) GetBackup(c *gin.Context) {
	id := c.Param("id"); if id == "" { response.BadRequest(c, "backup ID is required"); return }
	record, err := h.backupService.GetBackupRecord(c.Request.Context(), id)
	if err != nil { response.ErrorFrom(c, err); return }
	response.Success(c, record)
}
func (h *BackupHandler) DeleteBackup(c *gin.Context) {
	id := c.Param("id"); if id == "" { response.BadRequest(c, "backup ID is required"); return }
	if err := h.backupService.DeleteBackup(c.Request.Context(), id); err != nil { response.ErrorFrom(c, err); return }
	response.Success(c, gin.H{"deleted": true})
}
func (h *BackupHandler) GetDownloadURL(c *gin.Context) {
	id := c.Param("id"); if id == "" { response.BadRequest(c, "backup ID is required"); return }
	url, err := h.backupService.GetBackupDownloadURL(c.Request.Context(), id)
	if err != nil { response.ErrorFrom(c, err); return }
	response.Success(c, gin.H{"url": url})
}
type RestoreBackupRequest struct { Password string `json:"password" binding:"required"` }
func (h *BackupHandler) RestoreBackup(c *gin.Context) {
	id := c.Param("id"); if id == "" { response.BadRequest(c, "backup ID is required"); return }
	var req RestoreBackupRequest
	if err := c.ShouldBindJSON(&req); err != nil { response.BadRequest(c, "password is required for restore operation"); return }
	sub, ok := middleware.GetAuthSubjectFromContext(c); if !ok { response.Unauthorized(c, "unauthorized"); return }
	user, err := h.userService.GetByID(c.Request.Context(), sub.UserID)
	if err != nil { response.ErrorFrom(c, err); return }
	if !user.CheckPassword(req.Password) { response.BadRequest(c, "incorrect admin password"); return }
	record, err := h.backupService.StartRestore(c.Request.Context(), id)
	if err != nil { response.ErrorFrom(c, err); return }
	response.Accepted(c, record)
}
