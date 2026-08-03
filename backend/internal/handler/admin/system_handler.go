package admin

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	infraerrors "github.com/WilliamWang1721/LightBridge/internal/pkg/errors"
	"github.com/WilliamWang1721/LightBridge/internal/pkg/response"
	"github.com/WilliamWang1721/LightBridge/internal/pkg/sysutil"
	middleware2 "github.com/WilliamWang1721/LightBridge/internal/server/middleware"
	"github.com/WilliamWang1721/LightBridge/internal/service"

	"github.com/gin-gonic/gin"
)

// SystemHandler handles system-related operations
type SystemHandler struct {
	updateSvc     systemUpdateService
	backupSvc     systemBackupService
	featureReader progressiveFeatureReader
	lockSvc       *service.SystemOperationLockService
}

type systemUpdateService interface {
	CheckUpdate(ctx context.Context, force bool) (*service.UpdateInfo, error)
	ListVersionReleases(ctx context.Context, force bool) ([]service.VersionRelease, *service.UpdateInfo, error)
	PerformUpdate(ctx context.Context) error
	PerformUpdateToVersion(ctx context.Context, targetVersion string) error
	Rollback() error
}

type systemBackupService interface {
	CreateBackupWithMetadata(ctx context.Context, triggeredBy string, expireDays int, metadata service.BackupRecordMetadata) (*service.BackupRecord, error)
}

type progressiveFeatureReader interface {
	IsProgressiveFeatureEnabled(ctx context.Context, feature service.ProgressiveFeature) bool
}

const versionManagerBackupTriggeredBy = "version_manager"

// NewSystemHandler creates a new SystemHandler
func NewSystemHandler(updateSvc systemUpdateService, lockSvc *service.SystemOperationLockService) *SystemHandler {
	return &SystemHandler{
		updateSvc: updateSvc,
		lockSvc:   lockSvc,
	}
}

// SetVersionBackupDependencies configures optional pre-version snapshot
// support. The handler fails closed if a caller requests a snapshot and these
// dependencies are unavailable.
func (h *SystemHandler) SetVersionBackupDependencies(backupSvc systemBackupService, featureReader progressiveFeatureReader) {
	h.backupSvc = backupSvc
	h.featureReader = featureReader
}

// GetVersion returns the current version
// GET /api/v1/admin/system/version
func (h *SystemHandler) GetVersion(c *gin.Context) {
	info, _ := h.updateSvc.CheckUpdate(c.Request.Context(), false)
	response.Success(c, gin.H{
		"version": info.CurrentVersion,
	})
}

// CheckUpdates checks for available updates
// GET /api/v1/admin/system/check-updates
func (h *SystemHandler) CheckUpdates(c *gin.Context) {
	force := c.Query("force") == "true"
	info, err := h.updateSvc.CheckUpdate(c.Request.Context(), force)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, info)
}

// ListVersionReleases returns installable application releases for the version control page.
// GET /api/v1/admin/system/versions
func (h *SystemHandler) ListVersionReleases(c *gin.Context) {
	force := c.Query("force") == "true"
	releases, info, err := h.updateSvc.ListVersionReleases(c.Request.Context(), force)
	if err != nil {
		response.ErrorFrom(c, infraerrors.InternalServer("SYSTEM_VERSION_LIST_FAILED", "failed to list versions: "+err.Error()).WithCause(err))
		return
	}

	response.Success(c, gin.H{
		"current_version": info.CurrentVersion,
		"latest_version":  info.LatestVersion,
		"build_type":      info.BuildType,
		"capabilities":    info.Capabilities,
		"releases":        releases,
	})
}

// PerformUpdate downloads and applies the update
// POST /api/v1/admin/system/update
func (h *SystemHandler) PerformUpdate(c *gin.Context) {
	operationID := buildSystemOperationID(c, "update")
	var req struct {
		Version       string `json:"version"`
		BackupCurrent bool   `json:"backup_current"`
	}
	_ = c.ShouldBindJSON(&req)
	req.Version = normalizeVersionManagerTargetVersion(req.Version)
	payload := gin.H{
		"operation_id":   operationID,
		"version":        req.Version,
		"backup_current": req.BackupCurrent,
	}
	executeAdminIdempotentJSON(c, "admin.system.update", payload, service.DefaultSystemOperationIdempotencyTTL(), func(ctx context.Context) (any, error) {
		lock, release, err := h.acquireSystemLock(ctx, operationID)
		if err != nil {
			return nil, err
		}
		var releaseReason string
		succeeded := false
		defer func() {
			release(releaseReason, succeeded)
		}()

		var backup *service.BackupRecord
		if req.BackupCurrent {
			backup, err = h.createPreVersionBackup(ctx, c, lock.OperationID(), "update", req.Version)
			if err != nil {
				releaseReason = infraerrors.Reason(err)
				return nil, err
			}
		}

		if err := h.updateSvc.PerformUpdateToVersion(ctx, req.Version); err != nil {
			if errors.Is(err, service.ErrInPlaceUpdateUnsupported) {
				releaseReason = "SYSTEM_UPDATE_UNSUPPORTED"
				return nil, err
			}
			if errors.Is(err, service.ErrNoUpdateAvailable) {
				info, checkErr := h.updateSvc.CheckUpdate(ctx, false)
				if checkErr != nil {
					releaseReason = "SYSTEM_UPDATE_FAILED"
					return nil, checkErr
				}
				succeeded = true
				result := gin.H{
					"message":            "Already up to date",
					"already_up_to_date": true,
					"current_version":    info.CurrentVersion,
					"latest_version":     info.LatestVersion,
					"operation_id":       lock.OperationID(),
				}
				if backup != nil {
					result["backup"] = backup
				}
				return result, nil
			}
			releaseReason = "SYSTEM_UPDATE_FAILED"
			return nil, infraerrors.InternalServer("SYSTEM_UPDATE_FAILED", "update failed: "+err.Error()).WithCause(err)
		}
		succeeded = true

		result := gin.H{
			"message":      "Update completed. Please restart the service.",
			"need_restart": true,
			"operation_id": lock.OperationID(),
		}
		if backup != nil {
			result["backup"] = backup
		}
		return result, nil
	})
}

// Rollback restores the previous version
// POST /api/v1/admin/system/rollback
func (h *SystemHandler) Rollback(c *gin.Context) {
	operationID := buildSystemOperationID(c, "rollback")
	var req struct {
		BackupCurrent bool `json:"backup_current"`
	}
	_ = c.ShouldBindJSON(&req)
	payload := gin.H{
		"operation_id":   operationID,
		"backup_current": req.BackupCurrent,
	}
	executeAdminIdempotentJSON(c, "admin.system.rollback", payload, service.DefaultSystemOperationIdempotencyTTL(), func(ctx context.Context) (any, error) {
		lock, release, err := h.acquireSystemLock(ctx, operationID)
		if err != nil {
			return nil, err
		}
		var releaseReason string
		succeeded := false
		defer func() {
			release(releaseReason, succeeded)
		}()

		var backup *service.BackupRecord
		if req.BackupCurrent {
			backup, err = h.createPreVersionBackup(ctx, c, lock.OperationID(), "rollback", "")
			if err != nil {
				releaseReason = infraerrors.Reason(err)
				return nil, err
			}
		}

		if err := h.updateSvc.Rollback(); err != nil {
			if errors.Is(err, service.ErrInPlaceUpdateUnsupported) {
				releaseReason = "SYSTEM_ROLLBACK_UNSUPPORTED"
				return nil, err
			}
			releaseReason = "SYSTEM_ROLLBACK_FAILED"
			return nil, infraerrors.InternalServer("SYSTEM_ROLLBACK_FAILED", "rollback failed: "+err.Error()).WithCause(err)
		}
		succeeded = true

		result := gin.H{
			"message":      "Rollback completed. Please restart the service.",
			"need_restart": true,
			"operation_id": lock.OperationID(),
		}
		if backup != nil {
			result["backup"] = backup
		}
		return result, nil
	})
}

// createPreVersionBackup creates a synchronous PostgreSQL logical snapshot
// while the system operation lock is held. It deliberately fails closed: a
// requested snapshot must succeed before an update or rollback can run.
func (h *SystemHandler) createPreVersionBackup(
	ctx context.Context,
	c *gin.Context,
	operationID string,
	action string,
	targetVersion string,
) (*service.BackupRecord, error) {
	if h.featureReader == nil {
		return nil, infraerrors.ServiceUnavailable(
			"SYSTEM_VERSION_BACKUP_FEATURE_STATE_UNAVAILABLE",
			"pre-version backup feature state is unavailable; retry after the service is fully initialized",
		)
	}
	if !h.featureReader.IsProgressiveFeatureEnabled(ctx, service.ProgressiveFeatureBackup) {
		return nil, infraerrors.Conflict(
			"SYSTEM_VERSION_BACKUP_FEATURE_DISABLED",
			"pre-version backup is unavailable because the backup feature is disabled; enable it and retry",
		)
	}
	if h.backupSvc == nil {
		return nil, infraerrors.ServiceUnavailable(
			"SYSTEM_VERSION_BACKUP_UNAVAILABLE",
			"pre-version backup service is unavailable; verify backup configuration and retry",
		)
	}

	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok || subject.UserID <= 0 {
		return nil, infraerrors.Unauthorized(
			"SYSTEM_VERSION_BACKUP_ADMIN_IDENTITY_REQUIRED",
			"an authenticated admin identity is required to create a pre-version backup",
		)
	}

	info, err := h.updateSvc.CheckUpdate(ctx, false)
	if err != nil || info == nil || strings.TrimSpace(info.CurrentVersion) == "" {
		versionContextErr := err
		if versionContextErr == nil {
			versionContextErr = errors.New("current version is unavailable")
		}
		return nil, infraerrors.ServiceUnavailable(
			"SYSTEM_VERSION_BACKUP_VERSION_CONTEXT_UNAVAILABLE",
			"could not determine the current version for the pre-version backup; retry the operation",
		).WithCause(versionContextErr)
	}

	if action == "update" && strings.TrimSpace(targetVersion) == "" {
		targetVersion = info.LatestVersion
	}
	metadata := service.BackupRecordMetadata{
		SourceVersion:     strings.TrimSpace(info.CurrentVersion),
		VersionAction:     action,
		TargetVersion:     strings.TrimSpace(targetVersion),
		SystemOperationID: operationID,
		InitiatingAdminID: subject.UserID,
	}
	record, err := h.backupSvc.CreateBackupWithMetadata(
		ctx,
		versionManagerBackupTriggeredBy,
		service.DefaultPreVersionBackupRetentionDays,
		metadata,
	)
	if err == nil && record != nil && record.Status == "completed" {
		return record, nil
	}
	if err == nil {
		err = errors.New("backup completed without a completed backup record")
	}

	metadataForResponse := map[string]string{}
	if record != nil && record.ID != "" {
		metadataForResponse["backup_id"] = record.ID
	}
	return record, infraerrors.ServiceUnavailable(
		"SYSTEM_VERSION_BACKUP_FAILED",
		"pre-version PostgreSQL backup failed; verify S3-compatible backup storage and retry",
	).WithCause(err).WithMetadata(metadataForResponse)
}

func normalizeVersionManagerTargetVersion(version string) string {
	return strings.TrimPrefix(strings.TrimSpace(version), "v")
}

// RestartService restarts the systemd service
// POST /api/v1/admin/system/restart
func (h *SystemHandler) RestartService(c *gin.Context) {
	operationID := buildSystemOperationID(c, "restart")
	payload := gin.H{"operation_id": operationID}
	executeAdminIdempotentJSON(c, "admin.system.restart", payload, service.DefaultSystemOperationIdempotencyTTL(), func(ctx context.Context) (any, error) {
		lock, release, err := h.acquireSystemLock(ctx, operationID)
		if err != nil {
			return nil, err
		}
		succeeded := false
		defer func() {
			release("", succeeded)
		}()

		// Schedule service restart in background after sending response
		// This ensures the client receives the success response before the service restarts
		go func() {
			// Wait a moment to ensure the response is sent
			time.Sleep(500 * time.Millisecond)
			sysutil.RestartServiceAsync()
		}()
		succeeded = true
		return gin.H{
			"message":      "Service restart initiated",
			"operation_id": lock.OperationID(),
		}, nil
	})
}

func (h *SystemHandler) acquireSystemLock(
	ctx context.Context,
	operationID string,
) (*service.SystemOperationLock, func(string, bool), error) {
	if h.lockSvc == nil {
		return nil, nil, service.ErrIdempotencyStoreUnavail
	}
	lock, err := h.lockSvc.Acquire(ctx, operationID)
	if err != nil {
		return nil, nil, err
	}
	release := func(reason string, succeeded bool) {
		releaseCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = h.lockSvc.Release(releaseCtx, lock, succeeded, reason)
	}
	return lock, release, nil
}

func buildSystemOperationID(c *gin.Context, operation string) string {
	key := strings.TrimSpace(c.GetHeader("Idempotency-Key"))
	if key == "" {
		return "sysop-" + operation + "-" + strconv.FormatInt(time.Now().UnixNano(), 36)
	}
	actorScope := "admin:0"
	if subject, ok := middleware2.GetAuthSubjectFromContext(c); ok {
		actorScope = "admin:" + strconv.FormatInt(subject.UserID, 10)
	}
	seed := operation + "|" + actorScope + "|" + c.FullPath() + "|" + key
	hash := service.HashIdempotencyKey(seed)
	if len(hash) > 24 {
		hash = hash[:24]
	}
	return "sysop-" + hash
}
