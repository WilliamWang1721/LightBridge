//go:build unit

package admin

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	middleware2 "github.com/WilliamWang1721/LightBridge/internal/server/middleware"
	"github.com/WilliamWang1721/LightBridge/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type systemHandlerUpdateServiceStub struct {
	performErr   error
	rollbackErr  error
	updateInfo   *service.UpdateInfo
	checkErr     error
	checkForces  []bool
	performCall  int
	rollbackCall int
	events       *[]string
	targets      []string
}

func (s *systemHandlerUpdateServiceStub) CheckUpdate(_ context.Context, force bool) (*service.UpdateInfo, error) {
	s.checkForces = append(s.checkForces, force)
	return s.updateInfo, s.checkErr
}

func (s *systemHandlerUpdateServiceStub) ListVersionReleases(ctx context.Context, force bool) ([]service.VersionRelease, *service.UpdateInfo, error) {
	info, err := s.CheckUpdate(ctx, force)
	return nil, info, err
}

func (s *systemHandlerUpdateServiceStub) PerformUpdate(context.Context) error {
	s.performCall++
	if s.events != nil {
		*s.events = append(*s.events, "update")
	}
	return s.performErr
}

func (s *systemHandlerUpdateServiceStub) PerformUpdateToVersion(_ context.Context, targetVersion string) error {
	s.targets = append(s.targets, targetVersion)
	return s.PerformUpdate(context.Background())
}

func (s *systemHandlerUpdateServiceStub) Rollback() error {
	s.rollbackCall++
	if s.events != nil {
		*s.events = append(*s.events, "rollback")
	}
	return s.rollbackErr
}

type systemHandlerBackupServiceStub struct {
	createErr   error
	restoreErr  error
	listErr     error
	record      *service.BackupRecord
	records     []service.BackupRecord
	createCalls int
	restoreIDs  []string
	triggeredBy string
	expireDays  int
	metadata    []service.BackupRecordMetadata
	events      *[]string
}

func (s *systemHandlerBackupServiceStub) CreateBackupWithMetadata(
	_ context.Context,
	triggeredBy string,
	expireDays int,
	metadata service.BackupRecordMetadata,
) (*service.BackupRecord, error) {
	s.createCalls++
	s.triggeredBy = triggeredBy
	s.expireDays = expireDays
	s.metadata = append(s.metadata, metadata)
	if s.events != nil {
		*s.events = append(*s.events, "backup")
	}
	return s.record, s.createErr
}

type systemHandlerFeatureReaderStub struct {
	enabled bool
	calls   int
}

func (s *systemHandlerBackupServiceStub) RestoreBackup(_ context.Context, backupID string) error {
	s.restoreIDs = append(s.restoreIDs, backupID)
	if s.events != nil {
		*s.events = append(*s.events, "restore:"+backupID)
	}
	return s.restoreErr
}

func (s *systemHandlerBackupServiceStub) ListBackups(_ context.Context) ([]service.BackupRecord, error) {
	return append([]service.BackupRecord(nil), s.records...), s.listErr
}

func (s *systemHandlerFeatureReaderStub) IsProgressiveFeatureEnabled(_ context.Context, feature service.ProgressiveFeature) bool {
	s.calls++
	return feature == service.ProgressiveFeatureBackup && s.enabled
}

type systemUpdateResponseEnvelope struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		Message         string `json:"message"`
		AlreadyUpToDate bool   `json:"already_up_to_date"`
		CurrentVersion  string `json:"current_version"`
		LatestVersion   string `json:"latest_version"`
		OperationID     string `json:"operation_id"`
	} `json:"data"`
}

type systemVersionBackupResponseEnvelope struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		OperationID string                `json:"operation_id"`
		Backup      *service.BackupRecord `json:"backup"`
	} `json:"data"`
}

type systemUpdateErrorEnvelope struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Reason  string `json:"reason"`
}

func newSystemHandlerTestRouter(t *testing.T, updateSvc *systemHandlerUpdateServiceStub, repo *memoryIdempotencyRepoStub) *gin.Engine {
	if updateSvc.updateInfo == nil {
		updateSvc.updateInfo = &service.UpdateInfo{CurrentVersion: "0.1.133", LatestVersion: "0.1.134"}
	}
	backupSvc := &systemHandlerBackupServiceStub{
		record: &service.BackupRecord{ID: "safety-backup", Status: "completed"},
		records: []service.BackupRecord{{
			ID: "pre-upgrade-backup", Status: "completed", TriggeredBy: versionManagerBackupTriggeredBy,
			Metadata: &service.BackupRecordMetadata{SourceVersion: "0.1.132", TargetVersion: updateSvc.updateInfo.CurrentVersion, VersionAction: "update"},
		}},
	}
	return newSystemHandlerTestRouterWithVersionBackup(t, updateSvc, repo, backupSvc, &systemHandlerFeatureReaderStub{enabled: true}, 1)
}

func newSystemHandlerTestRouterWithVersionBackup(
	t *testing.T,
	updateSvc *systemHandlerUpdateServiceStub,
	repo *memoryIdempotencyRepoStub,
	backupSvc systemBackupService,
	featureReader progressiveFeatureReader,
	adminID int64,
) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	service.SetDefaultIdempotencyCoordinator(nil)
	t.Cleanup(func() {
		service.SetDefaultIdempotencyCoordinator(nil)
	})

	lockSvc := service.NewSystemOperationLockService(repo, service.IdempotencyConfig{
		ProcessingTimeout:  time.Second,
		SystemOperationTTL: time.Minute,
	})
	handler := NewSystemHandler(updateSvc, lockSvc)
	handler.SetVersionBackupDependencies(backupSvc, featureReader)

	router := gin.New()
	if adminID > 0 {
		router.Use(func(c *gin.Context) {
			c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: adminID})
			c.Next()
		})
	}
	router.POST("/api/v1/admin/system/update", handler.PerformUpdate)
	router.POST("/api/v1/admin/system/rollback", handler.Rollback)
	return router
}

func TestSystemHandlerUpdateWithPreVersionBackupRunsBackupFirst(t *testing.T) {
	events := make([]string, 0, 2)
	updateSvc := &systemHandlerUpdateServiceStub{
		updateInfo: &service.UpdateInfo{CurrentVersion: "0.1.132", LatestVersion: "0.1.133"},
		events:     &events,
	}
	backupSvc := &systemHandlerBackupServiceStub{
		record: &service.BackupRecord{ID: "backup-update", Status: "completed"},
		events: &events,
	}
	featureReader := &systemHandlerFeatureReaderStub{enabled: true}
	repo := newMemoryIdempotencyRepoStub()
	router := newSystemHandlerTestRouterWithVersionBackup(t, updateSvc, repo, backupSvc, featureReader, 42)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/system/update", strings.NewReader(`{"version":"0.1.133","backup_current":true}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "backup-before-update")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, []string{"backup", "update"}, events)
	require.Equal(t, 1, backupSvc.createCalls)
	require.Equal(t, versionManagerBackupTriggeredBy, backupSvc.triggeredBy)
	require.Equal(t, service.DefaultPreVersionBackupRetentionDays, backupSvc.expireDays)
	require.Len(t, backupSvc.metadata, 1)
	require.Equal(t, "0.1.132", backupSvc.metadata[0].SourceVersion)
	require.Equal(t, "update", backupSvc.metadata[0].VersionAction)
	require.Equal(t, "0.1.133", backupSvc.metadata[0].TargetVersion)
	require.Equal(t, int64(42), backupSvc.metadata[0].InitiatingAdminID)
	require.NotEmpty(t, backupSvc.metadata[0].SystemOperationID)
	require.Equal(t, 1, featureReader.calls)

	var body systemVersionBackupResponseEnvelope
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, 0, body.Code)
	require.NotNil(t, body.Data.Backup)
	require.Equal(t, "backup-update", body.Data.Backup.ID)
	require.Equal(t, backupSvc.metadata[0].SystemOperationID, body.Data.OperationID)
	requireSystemLockStatus(t, repo, service.IdempotencyStatusSucceeded)
}

func TestSystemHandlerRollbackWithPreVersionBackupRunsBackupFirst(t *testing.T) {
	events := make([]string, 0, 2)
	updateSvc := &systemHandlerUpdateServiceStub{
		updateInfo: &service.UpdateInfo{CurrentVersion: "0.1.133", LatestVersion: "0.1.134"},
		events:     &events,
	}
	backupSvc := &systemHandlerBackupServiceStub{
		record: &service.BackupRecord{ID: "backup-rollback", Status: "completed"},
		records: []service.BackupRecord{{
			ID: "pre-upgrade-0.1.133", Status: "completed", TriggeredBy: versionManagerBackupTriggeredBy,
			Metadata: &service.BackupRecordMetadata{SourceVersion: "0.1.132", TargetVersion: "0.1.133", VersionAction: "update"},
		}},
		events: &events,
	}
	featureReader := &systemHandlerFeatureReaderStub{enabled: true}
	repo := newMemoryIdempotencyRepoStub()
	router := newSystemHandlerTestRouterWithVersionBackup(t, updateSvc, repo, backupSvc, featureReader, 7)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/system/rollback", strings.NewReader(`{"backup_current":true}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "backup-before-rollback")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, []string{"backup", "restore:pre-upgrade-0.1.133", "rollback"}, events)
	require.Equal(t, 1, backupSvc.createCalls)
	require.Len(t, backupSvc.metadata, 1)
	require.Equal(t, "0.1.133", backupSvc.metadata[0].SourceVersion)
	require.Equal(t, "rollback", backupSvc.metadata[0].VersionAction)
	require.Empty(t, backupSvc.metadata[0].TargetVersion)
	require.Equal(t, int64(7), backupSvc.metadata[0].InitiatingAdminID)

	var body systemVersionBackupResponseEnvelope
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.NotNil(t, body.Data.Backup)
	require.Equal(t, "backup-rollback", body.Data.Backup.ID)
	require.Equal(t, []string{"pre-upgrade-0.1.133"}, backupSvc.restoreIDs)
	requireSystemLockStatus(t, repo, service.IdempotencyStatusSucceeded)
}

func TestSystemHandlerPreVersionBackupStreamFailurePreventsUpdate(t *testing.T) {
	updateSvc := &systemHandlerUpdateServiceStub{
		updateInfo: &service.UpdateInfo{CurrentVersion: "0.1.132", LatestVersion: "0.1.133"},
	}
	backupSvc := &systemHandlerBackupServiceStub{
		record:    &service.BackupRecord{ID: "failed-backup", Status: "failed"},
		createErr: errors.New("gzip/dump failed after upload: pg_dump stream close failed"),
	}
	featureReader := &systemHandlerFeatureReaderStub{enabled: true}
	repo := newMemoryIdempotencyRepoStub()
	router := newSystemHandlerTestRouterWithVersionBackup(t, updateSvc, repo, backupSvc, featureReader, 42)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/system/update", strings.NewReader(`{"backup_current":true}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "backup-failure-stops-update")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusServiceUnavailable, rec.Code)
	require.Equal(t, 1, backupSvc.createCalls)
	require.Zero(t, updateSvc.performCall)

	var body struct {
		Code     int               `json:"code"`
		Reason   string            `json:"reason"`
		Message  string            `json:"message"`
		Metadata map[string]string `json:"metadata"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, http.StatusServiceUnavailable, body.Code)
	require.Equal(t, "SYSTEM_VERSION_BACKUP_FAILED", body.Reason)
	require.Contains(t, body.Message, "retry")
	require.Equal(t, "failed-backup", body.Metadata["backup_id"])
	requireSystemLockStatus(t, repo, service.IdempotencyStatusFailedRetryable)
}

func TestSystemHandlerDisabledPreVersionBackupFailsClosed(t *testing.T) {
	updateSvc := &systemHandlerUpdateServiceStub{
		updateInfo: &service.UpdateInfo{CurrentVersion: "0.1.132", LatestVersion: "0.1.133"},
	}
	backupSvc := &systemHandlerBackupServiceStub{record: &service.BackupRecord{ID: "unexpected"}}
	featureReader := &systemHandlerFeatureReaderStub{enabled: false}
	repo := newMemoryIdempotencyRepoStub()
	router := newSystemHandlerTestRouterWithVersionBackup(t, updateSvc, repo, backupSvc, featureReader, 42)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/system/rollback", strings.NewReader(`{"backup_current":true}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "disabled-backup-fails-closed")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusConflict, rec.Code)
	require.Equal(t, 1, featureReader.calls)
	require.Zero(t, backupSvc.createCalls)
	require.Zero(t, updateSvc.rollbackCall)
	require.Empty(t, updateSvc.checkForces)

	var body systemUpdateErrorEnvelope
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, "SYSTEM_VERSION_BACKUP_FEATURE_DISABLED", body.Reason)
	requireSystemLockStatus(t, repo, service.IdempotencyStatusFailedRetryable)
}

func TestSystemHandlerVersionActionsForceBackupForLegacyClients(t *testing.T) {
	updateSvc := &systemHandlerUpdateServiceStub{updateInfo: &service.UpdateInfo{CurrentVersion: "0.1.133", LatestVersion: "0.1.134"}}
	backupSvc := &systemHandlerBackupServiceStub{
		record: &service.BackupRecord{ID: "forced-backup", Status: "completed"},
		records: []service.BackupRecord{{
			ID: "pre-upgrade-0.1.133", Status: "completed", TriggeredBy: versionManagerBackupTriggeredBy,
			Metadata: &service.BackupRecordMetadata{SourceVersion: "0.1.132", TargetVersion: "0.1.133", VersionAction: "update"},
		}},
	}
	featureReader := &systemHandlerFeatureReaderStub{enabled: true}
	repo := newMemoryIdempotencyRepoStub()
	router := newSystemHandlerTestRouterWithVersionBackup(t, updateSvc, repo, backupSvc, featureReader, 42)

	updateRec := httptest.NewRecorder()
	updateReq := httptest.NewRequest(http.MethodPost, "/api/v1/admin/system/update", strings.NewReader(`{"version":"0.1.134"}`))
	updateReq.Header.Set("Content-Type", "application/json")
	updateReq.Header.Set("Idempotency-Key", "legacy-update-without-backup")
	router.ServeHTTP(updateRec, updateReq)

	rollbackRec := httptest.NewRecorder()
	rollbackReq := httptest.NewRequest(http.MethodPost, "/api/v1/admin/system/rollback", strings.NewReader(`{}`))
	rollbackReq.Header.Set("Content-Type", "application/json")
	rollbackReq.Header.Set("Idempotency-Key", "legacy-rollback-without-backup")
	router.ServeHTTP(rollbackRec, rollbackReq)

	require.Equal(t, http.StatusOK, updateRec.Code)
	require.Equal(t, http.StatusOK, rollbackRec.Code)
	require.Equal(t, 1, updateSvc.performCall)
	require.Equal(t, 1, updateSvc.rollbackCall)
	require.Equal(t, 2, backupSvc.createCalls)
	require.Equal(t, 2, featureReader.calls)
	require.Equal(t, []string{"pre-upgrade-0.1.133"}, backupSvc.restoreIDs)
}

func TestSystemHandlerVersionActionIdempotencyPayloadCanonicalizesChoices(t *testing.T) {
	updateSvc := &systemHandlerUpdateServiceStub{updateInfo: &service.UpdateInfo{CurrentVersion: "0.1.133", LatestVersion: "0.1.134"}}
	backupSvc := &systemHandlerBackupServiceStub{
		record: &service.BackupRecord{ID: "unexpected", Status: "completed"},
		records: []service.BackupRecord{{
			ID: "pre-upgrade-0.1.133", Status: "completed", TriggeredBy: versionManagerBackupTriggeredBy,
			Metadata: &service.BackupRecordMetadata{SourceVersion: "0.1.132", TargetVersion: "0.1.133", VersionAction: "update"},
		}},
	}
	featureReader := &systemHandlerFeatureReaderStub{enabled: true}
	repo := newMemoryIdempotencyRepoStub()
	router := newSystemHandlerTestRouterWithVersionBackup(t, updateSvc, repo, backupSvc, featureReader, 42)

	config := service.DefaultIdempotencyConfig()
	service.SetDefaultIdempotencyCoordinator(service.NewIdempotencyCoordinator(repo, config))
	t.Cleanup(func() {
		service.SetDefaultIdempotencyCoordinator(nil)
	})

	call := func(path, body, key string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", key)
		router.ServeHTTP(rec, req)
		return rec
	}

	firstUpdate := call("/api/v1/admin/system/update", `{"version":" v0.1.133 "}`, "canonical-version")
	require.Equal(t, http.StatusOK, firstUpdate.Code)
	require.Equal(t, []string{"0.1.133"}, updateSvc.targets)

	replayedUpdate := call("/api/v1/admin/system/update", `{"version":"0.1.133","backup_current":false}`, "canonical-version")
	require.Equal(t, http.StatusOK, replayedUpdate.Code)
	require.Equal(t, "true", replayedUpdate.Header().Get("X-Idempotency-Replayed"))
	require.Equal(t, 1, updateSvc.performCall)

	targetChoiceConflict := call("/api/v1/admin/system/update", `{"version":"0.1.134","backup_current":false}`, "canonical-version")
	require.Equal(t, http.StatusConflict, targetChoiceConflict.Code)
	var targetConflict systemUpdateErrorEnvelope
	require.NoError(t, json.Unmarshal(targetChoiceConflict.Body.Bytes(), &targetConflict))
	require.Equal(t, "IDEMPOTENCY_KEY_CONFLICT", targetConflict.Reason)
	require.Equal(t, 1, updateSvc.performCall)

	backupChoiceReplay := call("/api/v1/admin/system/update", `{"version":"0.1.133","backup_current":true}`, "canonical-version")
	require.Equal(t, http.StatusOK, backupChoiceReplay.Code)
	require.Equal(t, "true", backupChoiceReplay.Header().Get("X-Idempotency-Replayed"))
	require.Equal(t, 1, updateSvc.performCall)
	require.Equal(t, 1, backupSvc.createCalls)

	firstRollback := call("/api/v1/admin/system/rollback", `{}`, "rollback-backup-choice")
	require.Equal(t, http.StatusOK, firstRollback.Code)
	rollbackChoiceReplay := call("/api/v1/admin/system/rollback", `{"backup_current":true}`, "rollback-backup-choice")
	require.Equal(t, http.StatusOK, rollbackChoiceReplay.Code)
	require.Equal(t, "true", rollbackChoiceReplay.Header().Get("X-Idempotency-Replayed"))
	require.Equal(t, 1, updateSvc.rollbackCall)
	require.Equal(t, 2, backupSvc.createCalls)
}

func requireSystemLockStatus(t *testing.T, repo *memoryIdempotencyRepoStub, wantStatus string) {
	t.Helper()
	repo.mu.Lock()
	defer repo.mu.Unlock()

	for _, record := range repo.data {
		if record.Status == wantStatus {
			return
		}
	}
	t.Fatalf("system lock status %q not found in records: %#v", wantStatus, repo.data)
}

func TestSystemHandlerPerformUpdateAlreadyUpToDateReturnsOK(t *testing.T) {
	updateSvc := &systemHandlerUpdateServiceStub{
		performErr: service.ErrNoUpdateAvailable,
		updateInfo: &service.UpdateInfo{
			CurrentVersion: "0.1.132",
			LatestVersion:  "0.1.132",
			HasUpdate:      false,
		},
	}
	repo := newMemoryIdempotencyRepoStub()
	router := newSystemHandlerTestRouter(t, updateSvc, repo)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/system/update", nil)
	req.Header.Set("Idempotency-Key", "already-up-to-date")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, 1, updateSvc.performCall)
	require.Equal(t, []bool{false}, updateSvc.checkForces)
	requireSystemLockStatus(t, repo, service.IdempotencyStatusSucceeded)

	var body systemUpdateResponseEnvelope
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, 0, body.Code)
	require.Equal(t, "success", body.Message)
	require.Equal(t, "Already up to date", body.Data.Message)
	require.True(t, body.Data.AlreadyUpToDate)
	require.Equal(t, "0.1.132", body.Data.CurrentVersion)
	require.Equal(t, "0.1.132", body.Data.LatestVersion)
	require.NotEmpty(t, body.Data.OperationID)
}

func TestSystemHandlerPerformUpdateFailureReturnsActionableError(t *testing.T) {
	updateSvc := &systemHandlerUpdateServiceStub{
		performErr: errors.New("download failed"),
	}
	repo := newMemoryIdempotencyRepoStub()
	router := newSystemHandlerTestRouter(t, updateSvc, repo)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/system/update", nil)
	req.Header.Set("Idempotency-Key", "real-failure")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusInternalServerError, rec.Code)
	require.Equal(t, 1, updateSvc.performCall)
	require.Equal(t, []bool{false}, updateSvc.checkForces)
	requireSystemLockStatus(t, repo, service.IdempotencyStatusFailedRetryable)

	var body systemUpdateErrorEnvelope
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, http.StatusInternalServerError, body.Code)
	require.Equal(t, "update failed: download failed", body.Message)
}

func TestSystemHandlerPerformUpdateContainerReturnsStableConflict(t *testing.T) {
	updateSvc := &systemHandlerUpdateServiceStub{
		performErr: service.ErrInPlaceUpdateUnsupported,
	}
	repo := newMemoryIdempotencyRepoStub()
	router := newSystemHandlerTestRouter(t, updateSvc, repo)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/system/update", nil)
	req.Header.Set("Idempotency-Key", "container-update")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusConflict, rec.Code)
	require.Equal(t, 1, updateSvc.performCall)
	requireSystemLockStatus(t, repo, service.IdempotencyStatusFailedRetryable)

	var body systemUpdateErrorEnvelope
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, http.StatusConflict, body.Code)
	require.Equal(t, "IN_PLACE_UPDATE_UNSUPPORTED", body.Reason)
	require.Contains(t, body.Message, "docker compose pull")
	require.Contains(t, body.Message, "docker compose up -d")
}

func TestSystemHandlerRollbackContainerReturnsStableConflict(t *testing.T) {
	updateSvc := &systemHandlerUpdateServiceStub{
		rollbackErr: service.ErrInPlaceUpdateUnsupported,
	}
	repo := newMemoryIdempotencyRepoStub()
	router := newSystemHandlerTestRouter(t, updateSvc, repo)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/system/rollback", nil)
	req.Header.Set("Idempotency-Key", "container-rollback")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusConflict, rec.Code)
	require.Equal(t, 1, updateSvc.rollbackCall)
	requireSystemLockStatus(t, repo, service.IdempotencyStatusFailedRetryable)

	var body systemUpdateErrorEnvelope
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, http.StatusConflict, body.Code)
	require.Equal(t, "IN_PLACE_UPDATE_UNSUPPORTED", body.Reason)
	require.Contains(t, body.Message, "docker compose pull")
}
