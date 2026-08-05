package service

import (
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	infraerrors "github.com/WilliamWang1721/LightBridge/internal/pkg/errors"
)

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

// StreamLocalBackup writes a full pg_dump through gzip directly to dst. It
// does not touch S3, create a backup record, or write a temporary server file.
func (s *BackupService) StreamLocalBackup(ctx context.Context, dst io.Writer) error {
	if s.shuttingDown.Load() {
		return infraerrors.ServiceUnavailable("SERVER_SHUTTING_DOWN", "server is shutting down")
	}

	s.opMu.Lock()
	if s.backingUp {
		s.opMu.Unlock()
		return ErrBackupInProgress
	}
	s.backingUp = true
	s.opMu.Unlock()
	defer func() {
		s.opMu.Lock()
		s.backingUp = false
		s.opMu.Unlock()
	}()

	dumpReader, err := s.dumper.Dump(ctx)
	if err != nil {
		return fmt.Errorf("pg_dump: %w", err)
	}
	defer dumpReader.Close()

	gz := gzip.NewWriter(dst)
	if _, err = io.Copy(gz, dumpReader); err != nil {
		_ = gz.Close()
		return fmt.Errorf("stream local backup: %w", err)
	}
	if err = gz.Close(); err != nil {
		return fmt.Errorf("finish local backup gzip: %w", err)
	}
	if err = dumpReader.Close(); err != nil {
		return fmt.Errorf("finish pg_dump stream: %w", err)
	}
	return nil
}
