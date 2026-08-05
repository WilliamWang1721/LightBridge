package admin

import (
	"context"
	"errors"
	"io"
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

type SystemHandler struct {
	updateSvc     systemUpdateService
	backupSvc     systemBackupService
	featureReader progressiveFeatureReader
	lockSvc       *service.SystemOperationLockService
}

type systemUpdateService interface {
	CheckUpdate(context.Context, bool) (*service.UpdateInfo, error)
	ListVersionReleases(context.Context, bool) ([]service.VersionRelease, *service.UpdateInfo, error)
	PerformUpdate(context.Context) error
	PerformUpdateToVersion(context.Context, string) error
	Rollback() error
}

type systemBackupService interface {
	CreateBackupWithMetadata(context.Context, string, int, service.BackupRecordMetadata) (*service.BackupRecord, error)
	RestoreBackup(context.Context, string) error
	ListBackups(context.Context) ([]service.BackupRecord, error)
}

type progressiveFeatureReader interface {
	IsProgressiveFeatureEnabled(context.Context, service.ProgressiveFeature) bool
}

const versionManagerBackupTriggeredBy = "version_manager"

func NewSystemHandler(updateSvc systemUpdateService, lockSvc *service.SystemOperationLockService) *SystemHandler {
	return &SystemHandler{updateSvc: updateSvc, lockSvc: lockSvc}
}
func (h *SystemHandler) SetVersionBackupDependencies(backupSvc systemBackupService, featureReader progressiveFeatureReader) {
	h.backupSvc = backupSvc
	h.featureReader = featureReader
}
func (h *SystemHandler) GetVersion(c *gin.Context) {
	info, _ := h.updateSvc.CheckUpdate(c.Request.Context(), false)
	response.Success(c, gin.H{"version": info.CurrentVersion})
}
func (h *SystemHandler) CheckUpdates(c *gin.Context) {
	info, err := h.updateSvc.CheckUpdate(c.Request.Context(), c.Query("force") == "true")
	if err != nil { response.Error(c, http.StatusInternalServerError, err.Error()); return }
	response.Success(c, info)
}
func (h *SystemHandler) ListVersionReleases(c *gin.Context) {
	releases, info, err := h.updateSvc.ListVersionReleases(c.Request.Context(), c.Query("force") == "true")
	if err != nil { response.ErrorFrom(c, infraerrors.InternalServer("SYSTEM_VERSION_LIST_FAILED", "failed to list versions: "+err.Error()).WithCause(err)); return }
	response.Success(c, gin.H{"current_version": info.CurrentVersion, "latest_version": info.LatestVersion, "build_type": info.BuildType, "capabilities": info.Capabilities, "releases": releases})
}

func (h *SystemHandler) PerformUpdate(c *gin.Context) {
	operationID := buildSystemOperationID(c, "update")
	var req struct {
		Version            string `json:"version"`
		BackupCurrent      *bool  `json:"backup_current"`
		ForceWithoutBackup bool   `json:"force_without_backup"`
	}
	if err := bindOptionalVersionActionJSON(c, &req); err != nil { response.ErrorFrom(c, err); return }
	req.Version = normalizeVersionManagerTargetVersion(req.Version)
	backupCurrent := true
	if req.BackupCurrent != nil { backupCurrent = *req.BackupCurrent }
	if !backupCurrent { req.ForceWithoutBackup = true }
	payload := gin.H{"operation_id": operationID, "version": req.Version, "backup_current": !req.ForceWithoutBackup, "force_without_backup": req.ForceWithoutBackup}
	executeAdminIdempotentJSON(c, "admin.system.update", payload, service.DefaultSystemOperationIdempotencyTTL(), func(ctx context.Context) (any, error) {
		lock, release, err := h.acquireSystemLock(ctx, operationID)
		if err != nil { return nil, err }
		var releaseReason string
		succeeded := false
		defer func() { release(releaseReason, succeeded) }()
		var backup *service.BackupRecord
		if !req.ForceWithoutBackup {
			backup, err = h.createPreVersionBackup(ctx, c, lock.OperationID(), "update", req.Version)
			if err != nil { releaseReason = infraerrors.Reason(err); return nil, err }
		}
		if err = h.updateSvc.PerformUpdateToVersion(ctx, req.Version); err != nil {
			if errors.Is(err, service.ErrInPlaceUpdateUnsupported) { releaseReason = "SYSTEM_UPDATE_UNSUPPORTED"; return nil, err }
			if errors.Is(err, service.ErrNoUpdateAvailable) {
				succeeded = true
				result := gin.H{"message": "Already up to date", "already_up_to_date": true, "operation_id": lock.OperationID(), "forced_without_backup": req.ForceWithoutBackup}
				if backup != nil {
					result["backup"] = backup
					if backup.Metadata != nil { result["current_version"] = backup.Metadata.SourceVersion; result["latest_version"] = backup.Metadata.TargetVersion }
				}
				return result, nil
			}
			if backup == nil {
				releaseReason = "SYSTEM_UPDATE_FAILED_WITHOUT_BACKUP"
				return nil, infraerrors.InternalServer("SYSTEM_UPDATE_FAILED_WITHOUT_BACKUP", "update failed after backup was explicitly bypassed; automatic database recovery was not attempted: "+err.Error()).WithCause(err)
			}
			if restoreErr := h.backupSvc.RestoreBackup(ctx, backup.ID); restoreErr != nil {
				releaseReason = "SYSTEM_UPDATE_DATABASE_RECOVERY_FAILED"
				return nil, infraerrors.InternalServer("SYSTEM_UPDATE_DATABASE_RECOVERY_FAILED", "update failed and the pre-update database snapshot could not be restored: "+restoreErr.Error()).WithCause(errors.Join(err, restoreErr)).WithMetadata(map[string]string{"backup_id": backup.ID})
			}
			releaseReason = "SYSTEM_UPDATE_FAILED"
			return nil, infraerrors.InternalServer("SYSTEM_UPDATE_FAILED", "update failed: "+err.Error()).WithCause(err)
		}
		succeeded = true
		result := gin.H{"message": "Update completed. Please restart the service.", "need_restart": true, "operation_id": lock.OperationID(), "forced_without_backup": req.ForceWithoutBackup}
		if backup != nil { result["backup"] = backup }
		return result, nil
	})
}

func (h *SystemHandler) Rollback(c *gin.Context) {
	operationID := buildSystemOperationID(c, "rollback")
	var req struct { BackupCurrent bool `json:"backup_current"` }
	if err := bindOptionalVersionActionJSON(c, &req); err != nil { response.ErrorFrom(c, err); return }
	payload := gin.H{"operation_id": operationID, "backup_current": true}
	executeAdminIdempotentJSON(c, "admin.system.rollback", payload, service.DefaultSystemOperationIdempotencyTTL(), func(ctx context.Context) (any, error) {
		lock, release, err := h.acquireSystemLock(ctx, operationID)
		if err != nil { return nil, err }
		var releaseReason string
		succeeded := false
		defer func() { release(releaseReason, succeeded) }()
		safetyBackup, err := h.createPreVersionBackup(ctx, c, lock.OperationID(), "rollback", "")
		if err != nil { releaseReason = infraerrors.Reason(err); return nil, err }
		currentVersion := ""
		if safetyBackup.Metadata != nil { currentVersion = normalizeVersionManagerTargetVersion(safetyBackup.Metadata.SourceVersion) }
		records, err := h.backupSvc.ListBackups(ctx)
		if err != nil { releaseReason = "SYSTEM_ROLLBACK_BACKUP_LIST_FAILED"; return nil, infraerrors.ServiceUnavailable("SYSTEM_ROLLBACK_BACKUP_LIST_FAILED", "could not list database snapshots required for a full rollback").WithCause(err) }
		preUpgradeBackup := findPreUpgradeBackupForVersion(records, currentVersion)
		if preUpgradeBackup == nil { releaseReason = "SYSTEM_ROLLBACK_DATABASE_SNAPSHOT_NOT_FOUND"; return nil, infraerrors.Conflict("SYSTEM_ROLLBACK_DATABASE_SNAPSHOT_NOT_FOUND", "no completed pre-upgrade database snapshot matches the current application version").WithMetadata(map[string]string{"current_version": currentVersion}) }
		if err = h.backupSvc.RestoreBackup(ctx, preUpgradeBackup.ID); err != nil { releaseReason = "SYSTEM_ROLLBACK_DATABASE_RESTORE_FAILED"; return nil, infraerrors.InternalServer("SYSTEM_ROLLBACK_DATABASE_RESTORE_FAILED", "could not restore the database snapshot required for rollback: "+err.Error()).WithCause(err).WithMetadata(map[string]string{"backup_id": preUpgradeBackup.ID}) }
		if err = h.updateSvc.Rollback(); err != nil {
			recoveryErr := h.backupSvc.RestoreBackup(ctx, safetyBackup.ID)
			if recoveryErr != nil { releaseReason = "SYSTEM_ROLLBACK_DATABASE_RECOVERY_FAILED"; return nil, infraerrors.InternalServer("SYSTEM_ROLLBACK_DATABASE_RECOVERY_FAILED", "binary rollback failed and the current database snapshot could not be restored: "+recoveryErr.Error()).WithCause(errors.Join(err, recoveryErr)).WithMetadata(map[string]string{"backup_id": safetyBackup.ID}) }
			if errors.Is(err, service.ErrInPlaceUpdateUnsupported) { releaseReason = "SYSTEM_ROLLBACK_UNSUPPORTED"; return nil, err }
			releaseReason = "SYSTEM_ROLLBACK_FAILED"
			return nil, infraerrors.InternalServer("SYSTEM_ROLLBACK_FAILED", "rollback failed: "+err.Error()).WithCause(err)
		}
		succeeded = true
		return gin.H{"message": "Rollback completed. Please restart the service.", "need_restart": true, "operation_id": lock.OperationID(), "backup": safetyBackup, "restored_backup": preUpgradeBackup}, nil
	})
}

func bindOptionalVersionActionJSON(c *gin.Context, dst any) error {
	err := c.ShouldBindJSON(dst)
	if err == nil || errors.Is(err, io.EOF) { return nil }
	return infraerrors.BadRequest("INVALID_REQUEST_BODY", "invalid JSON request body").WithCause(err)
}
func findPreUpgradeBackupForVersion(records []service.BackupRecord, currentVersion string) *service.BackupRecord {
	currentVersion = normalizeVersionManagerTargetVersion(currentVersion)
	if currentVersion == "" { return nil }
	for i := range records {
		r := &records[i]
		if r.ID == "" || r.Status != "completed" || r.TriggeredBy != versionManagerBackupTriggeredBy || r.Metadata == nil || r.Metadata.VersionAction != "update" { continue }
		if normalizeVersionManagerTargetVersion(r.Metadata.TargetVersion) == currentVersion { return r }
	}
	return nil
}
func (h *SystemHandler) createPreVersionBackup(ctx context.Context, c *gin.Context, operationID, action, targetVersion string) (*service.BackupRecord, error) {
	if h.featureReader == nil { return nil, infraerrors.ServiceUnavailable("SYSTEM_VERSION_BACKUP_FEATURE_STATE_UNAVAILABLE", "pre-version backup feature state is unavailable; retry after the service is fully initialized") }
	if !h.featureReader.IsProgressiveFeatureEnabled(ctx, service.ProgressiveFeatureBackup) { return nil, infraerrors.Conflict("SYSTEM_VERSION_BACKUP_FEATURE_DISABLED", "pre-version backup is unavailable because the backup feature is disabled; enable it and retry") }
	if h.backupSvc == nil { return nil, infraerrors.ServiceUnavailable("SYSTEM_VERSION_BACKUP_UNAVAILABLE", "pre-version backup service is unavailable; verify backup configuration and retry") }
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok || subject.UserID <= 0 { return nil, infraerrors.Unauthorized("SYSTEM_VERSION_BACKUP_ADMIN_IDENTITY_REQUIRED", "an authenticated admin identity is required to create a pre-version backup") }
	info, err := h.updateSvc.CheckUpdate(ctx, false)
	if err != nil || info == nil || strings.TrimSpace(info.CurrentVersion) == "" {
		if err == nil { err = errors.New("current version is unavailable") }
		return nil, infraerrors.ServiceUnavailable("SYSTEM_VERSION_BACKUP_VERSION_CONTEXT_UNAVAILABLE", "could not determine the current version for the pre-version backup; retry the operation").WithCause(err)
	}
	if action == "update" && strings.TrimSpace(targetVersion) == "" { targetVersion = info.LatestVersion }
	metadata := service.BackupRecordMetadata{SourceVersion: strings.TrimSpace(info.CurrentVersion), VersionAction: action, TargetVersion: strings.TrimSpace(targetVersion), SystemOperationID: operationID, InitiatingAdminID: subject.UserID}
	record, err := h.backupSvc.CreateBackupWithMetadata(ctx, versionManagerBackupTriggeredBy, service.DefaultPreVersionBackupRetentionDays, metadata)
	if err == nil && record != nil && record.Status == "completed" { if record.Metadata == nil { m := metadata; record.Metadata = &m }; return record, nil }
	if err == nil { err = errors.New("backup completed without a completed backup record") }
	md := map[string]string{"force_without_backup_allowed": "true", "version_action_started": "false"}
	if record != nil && record.ID != "" { md["backup_id"] = record.ID }
	return record, infraerrors.ServiceUnavailable("SYSTEM_VERSION_BACKUP_FAILED", "pre-version PostgreSQL backup failed; the version action was not started. Fix backup storage or explicitly force the update without a backup.").WithCause(err).WithMetadata(md)
}
func normalizeVersionManagerTargetVersion(v string) string { return strings.TrimPrefix(strings.TrimSpace(v), "v") }
func (h *SystemHandler) RestartService(c *gin.Context) {
	operationID := buildSystemOperationID(c, "restart")
	executeAdminIdempotentJSON(c, "admin.system.restart", gin.H{"operation_id": operationID}, service.DefaultSystemOperationIdempotencyTTL(), func(ctx context.Context) (any, error) {
		lock, release, err := h.acquireSystemLock(ctx, operationID); if err != nil { return nil, err }
		succeeded := false; defer func() { release("", succeeded) }()
		go func() { time.Sleep(500 * time.Millisecond); sysutil.RestartServiceAsync() }()
		succeeded = true
		return gin.H{"message": "Service restart initiated", "operation_id": lock.OperationID()}, nil
	})
}
func (h *SystemHandler) acquireSystemLock(ctx context.Context, operationID string) (*service.SystemOperationLock, func(string, bool), error) {
	if h.lockSvc == nil { return nil, nil, service.ErrIdempotencyStoreUnavail }
	lock, err := h.lockSvc.Acquire(ctx, operationID); if err != nil { return nil, nil, err }
	release := func(reason string, succeeded bool) { releaseCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second); defer cancel(); _ = h.lockSvc.Release(releaseCtx, lock, succeeded, reason) }
	return lock, release, nil
}
func buildSystemOperationID(c *gin.Context, operation string) string {
	key := strings.TrimSpace(c.GetHeader("Idempotency-Key")); if key == "" { return "sysop-" + operation + "-" + strconv.FormatInt(time.Now().UnixNano(), 36) }
	actorScope := "admin:0"; if subject, ok := middleware2.GetAuthSubjectFromContext(c); ok { actorScope = "admin:" + strconv.FormatInt(subject.UserID, 10) }
	seed := operation + "|" + actorScope + "|" + c.FullPath() + "|" + key
	hash := service.HashIdempotencyKey(seed); if len(hash) > 24 { hash = hash[:24] }
	return "sysop-" + hash
}
