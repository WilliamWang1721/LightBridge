package service

import (
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	infraerrors "github.com/WilliamWang1721/LightBridge/internal/pkg/errors"
)

// LocalBackupCompletionMarker is a deterministic, valid empty gzip member. It
// is appended only after pg_dump and the primary gzip member finish cleanly.
// Concatenated gzip members are standard, so the downloaded file remains a
// normal .sql.gz while browser clients can verify that streaming completed.
const LocalBackupCompletionMarker = "\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x01\x00\x00\xff\xff\x00\x00\x00\x00\x00\x00\x00\x00"

// LocalBackupDownload describes a database dump streamed directly to the
// requesting administrator. The body owns the backup-operation lock until the
// gzip producer finishes or the client closes the stream.
type LocalBackupDownload struct {
	FileName string
	Body     io.ReadCloser
}

// LocalBackupFileName returns a safe filename for a browser-downloaded full
// PostgreSQL logical backup. Local downloads never require S3 configuration.
func (s *BackupService) LocalBackupFileName() string {
	name := strings.TrimSpace(s.dbCfg.DBName)
	if name == "" {
		name = "lightbridge"
	}
	name = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			return r
		}
		return '_'
	}, name)
	return fmt.Sprintf("%s_%s.sql.gz", name, time.Now().Format("20060102_150405"))
}

// OpenLocalBackup starts a full pg_dump and returns a gzip stream without
// touching S3, creating a backup record, or writing a temporary server file.
// Dump startup errors are returned before HTTP download headers are committed.
func (s *BackupService) OpenLocalBackup(ctx context.Context) (*LocalBackupDownload, error) {
	if s.dumper == nil {
		return nil, infraerrors.ServiceUnavailable("LOCAL_BACKUP_UNAVAILABLE", "local database backup is unavailable")
	}

	s.opMu.Lock()
	if s.backingUp {
		s.opMu.Unlock()
		return nil, ErrBackupInProgress
	}
	s.backingUp = true
	s.opMu.Unlock()
	finish, err := s.beginTrackedOperation()
	if err != nil {
		s.opMu.Lock()
		s.backingUp = false
		s.opMu.Unlock()
		return nil, err
	}

	release := func() {
		s.opMu.Lock()
		s.backingUp = false
		s.opMu.Unlock()
		finish()
	}

	dumpReader, err := s.dumper.Dump(ctx)
	if err != nil {
		release()
		return nil, infraerrors.InternalServer(
			"LOCAL_BACKUP_DUMP_FAILED",
			"failed to start local database backup",
		).WithCause(err)
	}

	pipeReader, pipeWriter := io.Pipe()
	go func() {
		defer release()
		defer func() {
			if recovered := recover(); recovered != nil {
				_ = dumpReader.Close()
				_ = pipeWriter.CloseWithError(fmt.Errorf("local backup stream panic: %v", recovered))
			}
		}()

		gzipWriter := gzip.NewWriter(pipeWriter)
		_, copyErr := io.Copy(gzipWriter, dumpReader)
		dumpCloseErr := dumpReader.Close()
		if copyErr != nil || dumpCloseErr != nil {
			streamErr := errors.Join(
				wrapLocalBackupError("stream pg_dump", copyErr),
				wrapLocalBackupError("finish pg_dump stream", dumpCloseErr),
			)
			// Do not close the gzip writer on an upstream failure. Omitting the
			// gzip footer ensures a partial database dump cannot masquerade as a
			// complete, valid .sql.gz file.
			_ = pipeWriter.CloseWithError(streamErr)
			return
		}
		if err := gzipWriter.Close(); err != nil {
			_ = pipeWriter.CloseWithError(fmt.Errorf("finish local backup gzip: %w", err))
			return
		}
		if _, err := io.WriteString(pipeWriter, LocalBackupCompletionMarker); err != nil {
			_ = pipeWriter.CloseWithError(fmt.Errorf("write local backup completion marker: %w", err))
			return
		}
		_ = pipeWriter.Close()
	}()

	return &LocalBackupDownload{
		FileName: s.LocalBackupFileName(),
		Body:     pipeReader,
	}, nil
}

func wrapLocalBackupError(operation string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", operation, err)
}
