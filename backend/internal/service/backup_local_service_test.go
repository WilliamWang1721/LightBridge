//go:build unit

package service

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/WilliamWang1721/LightBridge/internal/config"
	infraerrors "github.com/WilliamWang1721/LightBridge/internal/pkg/errors"
	"github.com/stretchr/testify/require"
)

type localBackupDumperStub struct {
	data     string
	err      error
	closeErr error
}

type localBackupReadCloserStub struct {
	io.Reader
	closeErr error
}

func (r *localBackupReadCloserStub) Close() error {
	return r.closeErr
}

func (d localBackupDumperStub) Dump(context.Context) (io.ReadCloser, error) {
	if d.err != nil {
		return nil, d.err
	}
	return &localBackupReadCloserStub{
		Reader:   strings.NewReader(d.data),
		closeErr: d.closeErr,
	}, nil
}

func (d localBackupDumperStub) Restore(context.Context, io.Reader) error {
	return nil
}

func TestOpenLocalBackupDoesNotRequireS3(t *testing.T) {
	svc := NewBackupService(
		nil,
		&config.Config{Database: config.DatabaseConfig{DBName: "lightbridge"}},
		nil,
		nil,
		localBackupDumperStub{data: "create table test();"},
	)

	download, err := svc.OpenLocalBackup(context.Background())
	require.NoError(t, err)
	require.NotNil(t, download)
	require.Contains(t, download.FileName, "lightbridge_")
	require.True(t, strings.HasSuffix(download.FileName, ".sql.gz"))
	defer download.Body.Close()

	compressed, err := io.ReadAll(download.Body)
	require.NoError(t, err)
	require.True(t, bytes.HasSuffix(compressed, []byte(LocalBackupCompletionMarker)))

	gzipReader, err := gzip.NewReader(bytes.NewReader(compressed))
	require.NoError(t, err)
	defer gzipReader.Close()

	decoded, err := io.ReadAll(gzipReader)
	require.NoError(t, err)
	require.Equal(t, "create table test();", string(decoded))

	svc.wg.Wait()
	svc.opMu.Lock()
	backingUp := svc.backingUp
	svc.opMu.Unlock()
	require.False(t, backingUp)
}

func TestOpenLocalBackupReturnsDumpFailureBeforeDownload(t *testing.T) {
	svc := NewBackupService(
		nil,
		&config.Config{},
		nil,
		nil,
		localBackupDumperStub{err: errors.New("dump failed")},
	)

	download, err := svc.OpenLocalBackup(context.Background())
	require.Nil(t, download)
	require.Error(t, err)
	require.Equal(t, "LOCAL_BACKUP_DUMP_FAILED", infraerrors.Reason(err))

	svc.opMu.Lock()
	backingUp := svc.backingUp
	svc.opMu.Unlock()
	require.False(t, backingUp)
}

func TestOpenLocalBackupRejectsConcurrentBackup(t *testing.T) {
	svc := NewBackupService(nil, &config.Config{}, nil, nil, localBackupDumperStub{})
	svc.opMu.Lock()
	svc.backingUp = true
	svc.opMu.Unlock()

	download, err := svc.OpenLocalBackup(context.Background())
	require.Nil(t, download)
	require.ErrorIs(t, err, ErrBackupInProgress)
}

func TestOpenLocalBackupDoesNotFinalizeGzipWhenPgDumpFails(t *testing.T) {
	svc := NewBackupService(
		nil,
		&config.Config{},
		nil,
		nil,
		localBackupDumperStub{
			data:     "partial database dump",
			closeErr: errors.New("pg_dump exited with error"),
		},
	)

	download, err := svc.OpenLocalBackup(context.Background())
	require.NoError(t, err)
	compressed, streamErr := io.ReadAll(download.Body)
	require.Error(t, streamErr)
	require.Contains(t, streamErr.Error(), "finish pg_dump stream")
	require.False(t, bytes.HasSuffix(compressed, []byte(LocalBackupCompletionMarker)))

	gzipReader, gzipErr := gzip.NewReader(bytes.NewReader(compressed))
	if gzipErr == nil {
		_, readErr := io.Copy(io.Discard, gzipReader)
		closeErr := gzipReader.Close()
		require.Error(t, errors.Join(readErr, closeErr))
	}
}

func TestOpenLocalBackupReleasesActivityWhenClientCloses(t *testing.T) {
	svc := NewBackupService(
		nil,
		&config.Config{},
		nil,
		nil,
		localBackupDumperStub{data: strings.Repeat("database-row\n", 10000)},
	)

	download, err := svc.OpenLocalBackup(context.Background())
	require.NoError(t, err)
	require.NoError(t, download.Body.Close())

	done := make(chan struct{})
	go func() {
		svc.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("local backup remained active after the client closed the download")
	}

	svc.opMu.Lock()
	backingUp := svc.backingUp
	svc.opMu.Unlock()
	require.False(t, backingUp)
}

type shutdownBlockingLocalDumper struct {
	readStarted chan struct{}
}

type shutdownBlockingLocalReader struct {
	ctx     context.Context
	started chan struct{}
	once    sync.Once
}

func (r *shutdownBlockingLocalReader) Read(_ []byte) (int, error) {
	r.once.Do(func() { close(r.started) })
	<-r.ctx.Done()
	return 0, r.ctx.Err()
}

func (r *shutdownBlockingLocalReader) Close() error { return nil }

func (d shutdownBlockingLocalDumper) Dump(ctx context.Context) (io.ReadCloser, error) {
	return &shutdownBlockingLocalReader{ctx: ctx, started: d.readStarted}, nil
}

func (d shutdownBlockingLocalDumper) Restore(context.Context, io.Reader) error { return nil }

func TestStopCancelsLocalStreamingBackupAfterGracePeriod(t *testing.T) {
	dumper := shutdownBlockingLocalDumper{readStarted: make(chan struct{})}
	svc := NewBackupService(nil, &config.Config{}, nil, nil, dumper)
	configureFastBackupShutdown(svc)

	download, err := svc.OpenLocalBackup(context.Background())
	require.NoError(t, err)
	readErr := make(chan error, 1)
	go func() {
		_, err := io.Copy(io.Discard, download.Body)
		readErr <- err
	}()
	select {
	case <-dumper.readStarted:
	case <-time.After(time.Second):
		t.Fatal("local backup stream did not start reading pg_dump output")
	}

	waitForStop(t, svc)
	require.ErrorIs(t, <-readErr, context.Canceled)
	_ = download.Body.Close()
}

func TestStopUnblocksLocalBackupWhenClientDoesNotRead(t *testing.T) {
	svc := NewBackupService(
		nil,
		&config.Config{},
		nil,
		nil,
		localBackupDumperStub{data: strings.Repeat("blocked-output", 1<<20)},
	)
	configureFastBackupShutdown(svc)

	download, err := svc.OpenLocalBackup(context.Background())
	require.NoError(t, err)
	defer download.Body.Close()

	waitForStop(t, svc)
	svc.opMu.Lock()
	backingUp := svc.backingUp
	svc.opMu.Unlock()
	require.False(t, backingUp)
}
