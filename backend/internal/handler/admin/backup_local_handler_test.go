//go:build unit

package admin

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/WilliamWang1721/LightBridge/internal/config"
	"github.com/WilliamWang1721/LightBridge/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type localBackupHandlerDumperStub struct {
	data string
	err  error
}

func (d localBackupHandlerDumperStub) Dump(context.Context) (io.ReadCloser, error) {
	if d.err != nil {
		return nil, d.err
	}
	return io.NopCloser(strings.NewReader(d.data)), nil
}

func (d localBackupHandlerDumperStub) Restore(context.Context, io.Reader) error {
	return nil
}

func newLocalBackupHandlerRouter(dumper service.DBDumper) *gin.Engine {
	gin.SetMode(gin.TestMode)
	backupService := service.NewBackupService(
		nil,
		&config.Config{Database: config.DatabaseConfig{DBName: "lightbridge"}},
		nil,
		nil,
		dumper,
	)
	handler := NewBackupHandler(backupService, nil)
	router := gin.New()
	router.POST("/api/v1/admin/backups", handler.CreateBackup)
	return router
}

func TestBackupHandlerStreamsLocalBackupWithoutS3(t *testing.T) {
	router := newLocalBackupHandlerRouter(localBackupHandlerDumperStub{data: "create table local_backup();"})
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/admin/backups",
		strings.NewReader(`{"destination":"local"}`),
	)
	request.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "application/gzip", recorder.Header().Get("Content-Type"))
	require.Contains(t, recorder.Header().Get("Content-Disposition"), "attachment")
	require.Contains(t, recorder.Header().Get("Content-Disposition"), ".sql.gz")
	require.Equal(t, "no-store", recorder.Header().Get("Cache-Control"))
	require.Equal(t, "nosniff", recorder.Header().Get("X-Content-Type-Options"))

	compressed := recorder.Body.Bytes()
	require.True(t, bytes.HasSuffix(compressed, []byte(service.LocalBackupCompletionMarker)))
	gzipReader, err := gzip.NewReader(bytes.NewReader(compressed))
	require.NoError(t, err)
	defer gzipReader.Close()
	decoded, err := io.ReadAll(gzipReader)
	require.NoError(t, err)
	require.Equal(t, "create table local_backup();", string(decoded))
}

func TestBackupHandlerReturnsJSONWhenLocalDumpCannotStart(t *testing.T) {
	router := newLocalBackupHandlerRouter(localBackupHandlerDumperStub{err: errors.New("pg_dump unavailable")})
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/admin/backups",
		strings.NewReader(`{"destination":"local"}`),
	)
	request.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusInternalServerError, recorder.Code)
	require.Contains(t, recorder.Header().Get("Content-Type"), "application/json")
	require.NotContains(t, recorder.Header().Get("Content-Disposition"), "attachment")
	require.Contains(t, recorder.Body.String(), "LOCAL_BACKUP_DUMP_FAILED")
}

func TestBackupHandlerRejectsMalformedCreateRequest(t *testing.T) {
	router := newLocalBackupHandlerRouter(localBackupHandlerDumperStub{data: "must not run"})
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/admin/backups",
		strings.NewReader(`{"destination":`),
	)
	request.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusBadRequest, recorder.Code)
	require.Contains(t, recorder.Body.String(), "Invalid request")
	require.Empty(t, recorder.Header().Get("Content-Disposition"))
}

func TestBackupHandlerRejectsUnknownDestination(t *testing.T) {
	router := newLocalBackupHandlerRouter(localBackupHandlerDumperStub{data: "must not run"})
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/admin/backups",
		strings.NewReader(`{"destination":"filesystem"}`),
	)
	request.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusBadRequest, recorder.Code)
	require.Contains(t, recorder.Body.String(), "destination must be s3 or local")
	require.Empty(t, recorder.Header().Get("Content-Disposition"))
}
