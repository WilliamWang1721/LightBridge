//go:build unit

package service

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/WilliamWang1721/LightBridge/internal/config"
	"github.com/stretchr/testify/require"
)

type localBackupDumperStub struct{ data string; err error }
func (d localBackupDumperStub) Dump(context.Context) (io.ReadCloser, error) { if d.err != nil { return nil, d.err }; return io.NopCloser(strings.NewReader(d.data)), nil }
func (d localBackupDumperStub) Restore(context.Context, io.Reader) error { return nil }

func TestStreamLocalBackupDoesNotRequireS3(t *testing.T) {
	s := NewBackupService(nil, &config.Config{Database: config.DatabaseConfig{DBName: "lightbridge"}}, nil, nil, localBackupDumperStub{data: "create table test();"})
	var out bytes.Buffer
	require.NoError(t, s.StreamLocalBackup(context.Background(), &out))
	zr, err := gzip.NewReader(bytes.NewReader(out.Bytes())); require.NoError(t, err); defer zr.Close()
	decoded, err := io.ReadAll(zr); require.NoError(t, err); require.Equal(t, "create table test();", string(decoded))
	require.Contains(t, s.LocalBackupFileName(), "lightbridge_"); require.True(t, strings.HasSuffix(s.LocalBackupFileName(), ".sql.gz"))
}
func TestStreamLocalBackupPropagatesDumpFailure(t *testing.T) {
	s := NewBackupService(nil, &config.Config{}, nil, nil, localBackupDumperStub{err: errors.New("dump failed")})
	require.ErrorContains(t, s.StreamLocalBackup(context.Background(), io.Discard), "pg_dump")
}
func TestStreamLocalBackupRejectsConcurrentBackup(t *testing.T) {
	s := NewBackupService(nil, &config.Config{}, nil, nil, localBackupDumperStub{})
	s.backingUp = true
	require.ErrorIs(t, s.StreamLocalBackup(context.Background(), io.Discard), ErrBackupInProgress)
}
