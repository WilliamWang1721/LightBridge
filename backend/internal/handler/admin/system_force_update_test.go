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
	u := &systemHandlerUpdateServiceStub{updateInfo: &service.UpdateInfo{CurrentVersion: "0.1.132", LatestVersion: "0.1.133"}}
	b := &systemHandlerBackupServiceStub{createErr: errors.New("must not run")}
	f := &systemHandlerFeatureReaderStub{enabled: false}
	r := newMemoryIdempotencyRepoStub()
	router := newSystemHandlerTestRouterWithVersionBackup(t, u, r, b, f, 42)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/system/update", strings.NewReader(`{"version":"0.1.133","force_without_backup":true}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "force-update")
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, 1, u.performCall)
	require.Zero(t, b.createCalls)
	require.Zero(t, f.calls)
	var body struct { Data struct { Forced bool `json:"forced_without_backup"` } `json:"data"` }
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.True(t, body.Data.Forced)
}

func TestSystemHandlerForceUpdateFailureDoesNotRestore(t *testing.T) {
	u := &systemHandlerUpdateServiceStub{updateInfo: &service.UpdateInfo{CurrentVersion: "0.1.132", LatestVersion: "0.1.133"}, performErr: errors.New("replace failed")}
	b := &systemHandlerBackupServiceStub{}
	r := newMemoryIdempotencyRepoStub()
	router := newSystemHandlerTestRouterWithVersionBackup(t, u, r, b, &systemHandlerFeatureReaderStub{enabled: true}, 42)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/system/update", strings.NewReader(`{"force_without_backup":true}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusInternalServerError, rec.Code)
	require.Empty(t, b.restoreIDs)
	var body systemUpdateErrorEnvelope
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, "SYSTEM_UPDATE_FAILED_WITHOUT_BACKUP", body.Reason)
}
