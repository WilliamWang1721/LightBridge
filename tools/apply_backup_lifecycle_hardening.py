#!/usr/bin/env python3
from pathlib import Path


def replace_once(path: Path, old: str, new: str) -> None:
    text = path.read_text()
    count = text.count(old)
    if count != 1:
        raise SystemExit(f"{path}: expected one match, found {count}\n--- old ---\n{old}")
    path.write_text(text.replace(old, new))


service_path = Path("backend/internal/service/backup_service.go")
replace_once(
    service_path,
    "\tDefaultPreVersionBackupRetentionDays = 14\n)",
    "\tDefaultPreVersionBackupRetentionDays = 14\n\n"
    "\tbackupShutdownGracePeriod  = 5 * time.Minute\n"
    "\tbackupShutdownCancelPeriod = 10 * time.Second\n"
    "\tbackupRecordFinalizeTimeout = 10 * time.Second\n)",
)
replace_once(
    service_path,
    "\twg           sync.WaitGroup     // 追踪活跃的备份/恢复 goroutine\n"
    "\tshuttingDown atomic.Bool        // 阻止新备份启动\n"
    "\tbgCtx        context.Context    // 所有后台操作的 parent context\n"
    "\tbgCancel     context.CancelFunc // 取消所有活跃后台操作\n",
    "\twg           sync.WaitGroup     // 追踪活跃的备份/恢复 goroutine\n"
    "\tshuttingDown atomic.Bool        // 阻止新备份启动\n"
    "\tbgCtx        context.Context    // 所有后台操作的 parent context\n"
    "\tbgCancel     context.CancelFunc // 取消所有活跃后台操作\n\n"
    "\tshutdownGracePeriod  time.Duration // cancellation-free drain window\n"
    "\tshutdownCancelPeriod time.Duration // bounded cleanup window after cancellation\n",
)
replace_once(
    service_path,
    "\t\tbgCtx:        bgCtx,\n\t\tbgCancel:     bgCancel,\n",
    "\t\tbgCtx:                bgCtx,\n"
    "\t\tbgCancel:             bgCancel,\n"
    "\t\tshutdownGracePeriod:  backupShutdownGracePeriod,\n"
    "\t\tshutdownCancelPeriod: backupShutdownCancelPeriod,\n",
)
replace_once(
    service_path,
    "// beginTrackedOperation admits work atomically with respect to Stop.\n"
    "func (s *BackupService) beginTrackedOperation() (func(), error) {\n"
    "\ts.lifecycleMu.Lock()\n"
    "\tdefer s.lifecycleMu.Unlock()\n"
    "\tif s.shuttingDown.Load() {\n"
    "\t\treturn nil, infraerrors.ServiceUnavailable(\"SERVER_SHUTTING_DOWN\", \"server is shutting down\")\n"
    "\t}\n"
    "\ts.wg.Add(1)\n"
    "\tvar once sync.Once\n"
    "\treturn func() { once.Do(s.wg.Done) }, nil\n"
    "}\n",
    "// beginTrackedOperation admits work atomically with respect to Stop and\n"
    "// returns a context cancelled by either the caller or service shutdown.\n"
    "func (s *BackupService) beginTrackedOperation(parent context.Context) (context.Context, func(), error) {\n"
    "\tif parent == nil {\n"
    "\t\treturn nil, nil, errors.New(\"backup operation context is required\")\n"
    "\t}\n\n"
    "\ts.lifecycleMu.Lock()\n"
    "\tif s.shuttingDown.Load() {\n"
    "\t\ts.lifecycleMu.Unlock()\n"
    "\t\treturn nil, nil, infraerrors.ServiceUnavailable(\"SERVER_SHUTTING_DOWN\", \"server is shutting down\")\n"
    "\t}\n"
    "\toperationCtx, cancelOperation := context.WithCancel(parent)\n"
    "\tstopShutdownLink := context.AfterFunc(s.bgCtx, cancelOperation)\n"
    "\ts.wg.Add(1)\n"
    "\ts.lifecycleMu.Unlock()\n\n"
    "\tvar once sync.Once\n"
    "\tfinish := func() {\n"
    "\t\tonce.Do(func() {\n"
    "\t\t\tstopShutdownLink()\n"
    "\t\t\tcancelOperation()\n"
    "\t\t\ts.wg.Done()\n"
    "\t\t})\n"
    "\t}\n"
    "\treturn operationCtx, finish, nil\n"
    "}\n",
)
replace_once(
    service_path,
    "// Stop 停止定时备份并等待活跃操作完成\n"
    "func (s *BackupService) Stop() {\n"
    "\ts.lifecycleMu.Lock()\n"
    "\ts.shuttingDown.Store(true)\n"
    "\ts.lifecycleMu.Unlock()\n\n"
    "\ts.cronMu.Lock()\n"
    "\tif s.cronSched != nil {\n"
    "\t\ts.cronSched.Stop()\n"
    "\t}\n"
    "\ts.cronMu.Unlock()\n\n"
    "\t// 等待活跃备份/恢复完成（最多 5 分钟）\n"
    "\tdone := make(chan struct{})\n"
    "\tgo func() {\n"
    "\t\ts.wg.Wait()\n"
    "\t\tclose(done)\n"
    "\t}()\n"
    "\tselect {\n"
    "\tcase <-done:\n"
    "\t\tlogger.LegacyPrintf(\"service.backup\", \"[Backup] all active operations finished\")\n"
    "\tcase <-time.After(5 * time.Minute):\n"
    "\t\tlogger.LegacyPrintf(\"service.backup\", \"[Backup] shutdown timeout after 5min, cancelling active operations\")\n"
    "\t\tif s.bgCancel != nil {\n"
    "\t\t\ts.bgCancel() // 取消所有后台操作\n"
    "\t\t}\n"
    "\t\t// 给 goroutine 时间响应取消并完成清理\n"
    "\t\tselect {\n"
    "\t\tcase <-done:\n"
    "\t\t\tlogger.LegacyPrintf(\"service.backup\", \"[Backup] active operations cancelled and cleaned up\")\n"
    "\t\tcase <-time.After(10 * time.Second):\n"
    "\t\t\tlogger.LegacyPrintf(\"service.backup\", \"[Backup] goroutine cleanup timed out\")\n"
    "\t\t}\n"
    "\t}\n"
    "}\n",
    "// Stop stops scheduling, allows a bounded graceful drain, then cancels every\n"
    "// admitted operation and waits for command/resource cleanup to complete.\n"
    "func (s *BackupService) Stop() {\n"
    "\ts.lifecycleMu.Lock()\n"
    "\ts.shuttingDown.Store(true)\n"
    "\ts.lifecycleMu.Unlock()\n\n"
    "\ts.cronMu.Lock()\n"
    "\tif s.cronSched != nil {\n"
    "\t\ts.cronSched.Stop()\n"
    "\t}\n"
    "\ts.cronMu.Unlock()\n\n"
    "\tdone := make(chan struct{})\n"
    "\tgo func() {\n"
    "\t\ts.wg.Wait()\n"
    "\t\tclose(done)\n"
    "\t}()\n\n"
    "\tgracePeriod := s.shutdownGracePeriod\n"
    "\tif gracePeriod <= 0 {\n"
    "\t\tgracePeriod = backupShutdownGracePeriod\n"
    "\t}\n"
    "\tcancelPeriod := s.shutdownCancelPeriod\n"
    "\tif cancelPeriod <= 0 {\n"
    "\t\tcancelPeriod = backupShutdownCancelPeriod\n"
    "\t}\n\n"
    "\tgraceTimer := time.NewTimer(gracePeriod)\n"
    "\tdefer graceTimer.Stop()\n"
    "\tselect {\n"
    "\tcase <-done:\n"
    "\t\tlogger.LegacyPrintf(\"service.backup\", \"[Backup] all active operations finished\")\n"
    "\tcase <-graceTimer.C:\n"
    "\t\tlogger.LegacyPrintf(\"service.backup\", \"[Backup] graceful shutdown deadline reached; cancelling active operations\")\n"
    "\t\tif s.bgCancel != nil {\n"
    "\t\t\ts.bgCancel()\n"
    "\t\t}\n"
    "\t\tcancelTimer := time.NewTimer(cancelPeriod)\n"
    "\t\tdefer cancelTimer.Stop()\n"
    "\t\tselect {\n"
    "\t\tcase <-done:\n"
    "\t\t\tlogger.LegacyPrintf(\"service.backup\", \"[Backup] active operations cancelled and cleaned up\")\n"
    "\t\tcase <-cancelTimer.C:\n"
    "\t\t\tlogger.LegacyPrintf(\"service.backup\", \"[Backup] active operations did not finish before the cancellation cleanup deadline\")\n"
    "\t\t}\n"
    "\t}\n\n"
    "\tif s.bgCancel != nil {\n"
    "\t\ts.bgCancel()\n"
    "\t}\n"
    "}\n",
)

text = service_path.read_text()
text = text.replace(
    "\tfinish, err := s.beginTrackedOperation()\n\tif err != nil {\n\t\treturn\n\t}\n\tdefer finish()\n\n\tctx, cancel := context.WithTimeout(s.bgCtx, 30*time.Minute)",
    "\t_, finish, err := s.beginTrackedOperation(s.bgCtx)\n\tif err != nil {\n\t\treturn\n\t}\n\tdefer finish()\n\n\tctx, cancel := context.WithTimeout(s.bgCtx, 30*time.Minute)",
    1,
)
text = text.replace(
    ") (*BackupRecord, error) {\n\tfinish, err := s.beginTrackedOperation()\n\tif err != nil {\n\t\treturn nil, err\n\t}\n\tdefer finish()",
    ") (*BackupRecord, error) {\n\toperationCtx, finish, err := s.beginTrackedOperation(ctx)\n\tif err != nil {\n\t\treturn nil, err\n\t}\n\tdefer finish()\n\tctx = operationCtx",
    1,
)
text = text.replace(
    "\tfinish, err := s.beginTrackedOperation()\n\tif err != nil {\n\t\ts.opMu.Lock()\n\t\ts.backingUp = false",
    "\t_, finish, err := s.beginTrackedOperation(s.bgCtx)\n\tif err != nil {\n\t\ts.opMu.Lock()\n\t\ts.backingUp = false",
    1,
)
text = text.replace(
    "func (s *BackupService) RestoreBackup(ctx context.Context, backupID string) error {\n\tfinish, err := s.beginTrackedOperation()\n\tif err != nil {\n\t\treturn err\n\t}\n\tdefer finish()",
    "func (s *BackupService) RestoreBackup(ctx context.Context, backupID string) error {\n\toperationCtx, finish, err := s.beginTrackedOperation(ctx)\n\tif err != nil {\n\t\treturn err\n\t}\n\tdefer finish()\n\tctx = operationCtx",
    1,
)
text = text.replace(
    "\tfinish, err := s.beginTrackedOperation()\n\tif err != nil {\n\t\ts.opMu.Lock()\n\t\ts.restoring = false",
    "\t_, finish, err := s.beginTrackedOperation(s.bgCtx)\n\tif err != nil {\n\t\ts.opMu.Lock()\n\t\ts.restoring = false",
    1,
)
if "beginTrackedOperation()" in text:
    raise SystemExit("unconverted beginTrackedOperation call remains")

start = text.index("func (s *BackupService) createBackup(")
end = text.index("func cloneBackupRecordMetadata", start)
create_block = text[start:end]
create_block = create_block.replace("_ = s.saveRecord(ctx, record)", "s.saveRecordAfterOperation(record)")
create_block = create_block.replace(
    "\tif err := s.saveRecord(ctx, record); err != nil {\n"
    "\t\tlogger.LegacyPrintf(\"service.backup\", \"[Backup] 保存备份记录失败: %v\", err)\n"
    "\t}\n",
    "\ts.saveRecordAfterOperation(record)\n",
)
helper = (
    "func (s *BackupService) saveRecordAfterOperation(record *BackupRecord) {\n"
    "\tctx, cancel := context.WithTimeout(context.Background(), backupRecordFinalizeTimeout)\n"
    "\tdefer cancel()\n"
    "\tif err := s.saveRecord(ctx, record); err != nil {\n"
    "\t\tlogger.LegacyPrintf(\"service.backup\", \"[Backup] failed to persist terminal backup state: %v\", err)\n"
    "\t}\n"
    "}\n\n"
)
text = text[:start] + create_block + helper + text[end:]
service_path.write_text(text)

local_path = Path("backend/internal/service/backup_local_service.go")
replace_once(
    local_path,
    "\tfinish, err := s.beginTrackedOperation()\n"
    "\tif err != nil {\n"
    "\t\ts.opMu.Lock()\n"
    "\t\ts.backingUp = false\n"
    "\t\ts.opMu.Unlock()\n"
    "\t\treturn nil, err\n"
    "\t}\n",
    "\toperationCtx, finish, err := s.beginTrackedOperation(ctx)\n"
    "\tif err != nil {\n"
    "\t\ts.opMu.Lock()\n"
    "\t\ts.backingUp = false\n"
    "\t\ts.opMu.Unlock()\n"
    "\t\treturn nil, err\n"
    "\t}\n"
    "\tctx = operationCtx\n",
)

service_test_path = Path("backend/internal/service/backup_service_test.go")
replace_once(service_test_path, 'import (\n\t"bytes"\n', 'import (\n\t"bytes"\n\t"compress/gzip"\n')
service_tests = r'''

type shutdownCancellationDumper struct {
	dumpStarted    chan struct{}
	restoreStarted chan struct{}
}

func (d *shutdownCancellationDumper) Dump(ctx context.Context) (io.ReadCloser, error) {
	close(d.dumpStarted)
	<-ctx.Done()
	return nil, ctx.Err()
}

func (d *shutdownCancellationDumper) Restore(ctx context.Context, _ io.Reader) error {
	close(d.restoreStarted)
	<-ctx.Done()
	return ctx.Err()
}

func configureFastBackupShutdown(svc *BackupService) {
	svc.shutdownGracePeriod = 20 * time.Millisecond
	svc.shutdownCancelPeriod = time.Second
}

func waitForStop(t *testing.T, svc *BackupService) {
	t.Helper()
	done := make(chan struct{})
	go func() {
		svc.Stop()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Stop did not finish after cancelling the active operation")
	}
}

func TestStopCancelsSynchronousBackupAfterGracePeriod(t *testing.T) {
	repo := newMockSettingRepo()
	seedS3Config(t, repo)
	dumper := &shutdownCancellationDumper{dumpStarted: make(chan struct{})}
	svc := newTestBackupService(repo, dumper, newMockObjectStore())
	configureFastBackupShutdown(svc)

	errCh := make(chan error, 1)
	go func() {
		_, err := svc.CreateBackup(context.Background(), "manual", 14)
		errCh <- err
	}()
	select {
	case <-dumper.dumpStarted:
	case <-time.After(time.Second):
		t.Fatal("synchronous backup did not reach pg_dump")
	}

	waitForStop(t, svc)
	require.ErrorIs(t, <-errCh, context.Canceled)
}

func TestStopCancelsSynchronousRestoreAfterGracePeriod(t *testing.T) {
	repo := newMockSettingRepo()
	seedS3Config(t, repo)
	store := newMockObjectStore()
	var compressed bytes.Buffer
	writer := gzip.NewWriter(&compressed)
	_, err := writer.Write([]byte("restore payload"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())
	store.objects["backups/restore.sql.gz"] = compressed.Bytes()

	dumper := &shutdownCancellationDumper{restoreStarted: make(chan struct{})}
	svc := newTestBackupService(repo, dumper, store)
	configureFastBackupShutdown(svc)
	require.NoError(t, svc.saveRecord(context.Background(), &BackupRecord{
		ID: "restore-on-shutdown", Status: "completed", S3Key: "backups/restore.sql.gz",
	}))

	errCh := make(chan error, 1)
	go func() {
		errCh <- svc.RestoreBackup(context.Background(), "restore-on-shutdown")
	}()
	select {
	case <-dumper.restoreStarted:
	case <-time.After(time.Second):
		t.Fatal("synchronous restore did not reach pg restore")
	}

	waitForStop(t, svc)
	require.ErrorIs(t, <-errCh, context.Canceled)
}
'''
service_test_text = service_test_path.read_text()
if "TestStopCancelsSynchronousBackupAfterGracePeriod" in service_test_text:
    raise SystemExit("backup lifecycle tests already present")
service_test_path.write_text(service_test_text + service_tests)

local_test_path = Path("backend/internal/service/backup_local_service_test.go")
local_tests = r'''

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
'''
local_test_text = local_test_path.read_text()
if "TestStopCancelsLocalStreamingBackupAfterGracePeriod" in local_test_text:
    raise SystemExit("local lifecycle test already present")
local_test_text = local_test_text.replace('"strings"\n\t"testing"', '"strings"\n\t"sync"\n\t"testing"')
local_test_path.write_text(local_test_text + local_tests)
