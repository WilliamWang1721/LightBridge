//go:build unit

package admin

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/WilliamWang1721/LightBridge/internal/service"
	"github.com/stretchr/testify/require"
)

func TestSystemHandlerForceUpdateSkipsBackup(t *testing.T) {
	updateSvc := &systemHandlerUpdateServiceStub{
		updateInfo: &service.UpdateInfo{CurrentVersion: "0.1.132", LatestVersion: "0.1.133"},
	}
	backupSvc := &systemHandlerBackupServiceStub{createErr: errors.New("must not run")}
	featureReader := &systemHandlerFeatureReaderStub{enabled: false}
	repo := newMemoryIdempotencyRepoStub()
	router := newSystemHandlerTestRouterWithVersionBackup(
		t,
		updateSvc,
		repo,
		backupSvc,
		featureReader,
		42,
	)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/admin/system/update",
		strings.NewReader(`{"version":"0.1.133","force_without_backup":true}`),
	)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "force-update")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, 1, updateSvc.performCall)
	require.Zero(t, backupSvc.createCalls)
	require.Zero(t, featureReader.calls)

	var body struct {
		Data struct {
			Forced bool `json:"forced_without_backup"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.True(t, body.Data.Forced)
	requireSystemLockStatus(t, repo, service.IdempotencyStatusSucceeded)
}

func TestSystemHandlerBackupCurrentFalseAloneRemainsFailClosed(t *testing.T) {
	updateSvc := &systemHandlerUpdateServiceStub{
		updateInfo: &service.UpdateInfo{CurrentVersion: "0.1.132", LatestVersion: "0.1.133"},
	}
	backupSvc := &systemHandlerBackupServiceStub{
		record: &service.BackupRecord{ID: "required-backup", Status: "completed"},
	}
	repo := newMemoryIdempotencyRepoStub()
	router := newSystemHandlerTestRouterWithVersionBackup(
		t,
		updateSvc,
		repo,
		backupSvc,
		&systemHandlerFeatureReaderStub{enabled: true},
		42,
	)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/admin/system/update",
		strings.NewReader(`{"version":"0.1.133","backup_current":false}`),
	)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, 1, updateSvc.performCall)
	require.Equal(t, 1, backupSvc.createCalls)
}

func TestSystemHandlerForceUpdateFailureDoesNotRestore(t *testing.T) {
	updateSvc := &systemHandlerUpdateServiceStub{
		updateInfo: &service.UpdateInfo{CurrentVersion: "0.1.132", LatestVersion: "0.1.133"},
		performErr: errors.New("replace failed"),
	}
	backupSvc := &systemHandlerBackupServiceStub{}
	repo := newMemoryIdempotencyRepoStub()
	router := newSystemHandlerTestRouterWithVersionBackup(
		t,
		updateSvc,
		repo,
		backupSvc,
		&systemHandlerFeatureReaderStub{enabled: true},
		42,
	)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/admin/system/update",
		strings.NewReader(`{"force_without_backup":true}`),
	)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusInternalServerError, rec.Code)
	require.Empty(t, backupSvc.restoreIDs)

	var body struct {
		Reason   string            `json:"reason"`
		Metadata map[string]string `json:"metadata"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, "SYSTEM_UPDATE_FAILED_WITHOUT_BACKUP", body.Reason)
	require.Equal(t, "true", body.Metadata["backup_bypassed"])
	require.Equal(t, "false", body.Metadata["database_recovery_attempted"])
	requireSystemLockStatus(t, repo, service.IdempotencyStatusFailedRetryable)
}
