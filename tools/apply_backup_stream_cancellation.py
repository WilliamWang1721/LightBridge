#!/usr/bin/env python3
from pathlib import Path


def replace_once(path: Path, old: str, new: str) -> None:
    text = path.read_text()
    count = text.count(old)
    if count != 1:
        raise SystemExit(f"{path}: expected one match, found {count}\n--- old ---\n{old}")
    path.write_text(text.replace(old, new))


local_service = Path("backend/internal/service/backup_local_service.go")
replace_once(
    local_service,
    "\tpipeReader, pipeWriter := io.Pipe()\n\tgo func() {\n\t\tdefer release()\n",
    "\tpipeReader, pipeWriter := io.Pipe()\n"
    "\tstopCancellation := context.AfterFunc(ctx, func() {\n"
    "\t\tcancelErr := context.Cause(ctx)\n"
    "\t\tif cancelErr == nil {\n"
    "\t\t\tcancelErr = context.Canceled\n"
    "\t\t}\n"
    "\t\t_ = dumpReader.Close()\n"
    "\t\t_ = pipeWriter.CloseWithError(cancelErr)\n"
    "\t})\n"
    "\tgo func() {\n"
    "\t\tdefer stopCancellation()\n"
    "\t\tdefer release()\n",
)

pg_dumper = Path("backend/internal/repository/backup_pg_dumper.go")
replace_once(pg_dumper, '"os/exec"\n', '"os/exec"\n\t"sync"\n')
replace_once(
    pg_dumper,
    "type cmdReadCloser struct {\n\tio.ReadCloser\n\tcmd *exec.Cmd\n}\n\n"
    "func (c *cmdReadCloser) Close() error {\n"
    "\t// Close the pipe first\n"
    "\t_ = c.ReadCloser.Close()\n"
    "\t// Wait for the process to exit\n"
    "\tif err := c.cmd.Wait(); err != nil {\n"
    "\t\treturn fmt.Errorf(\"pg_dump exited with error: %w\", err)\n"
    "\t}\n"
    "\treturn nil\n"
    "}\n",
    "type cmdReadCloser struct {\n"
    "\tio.ReadCloser\n"
    "\tcmd       *exec.Cmd\n"
    "\tcloseOnce sync.Once\n"
    "\tcloseErr  error\n"
    "}\n\n"
    "func (c *cmdReadCloser) Close() error {\n"
    "\tc.closeOnce.Do(func() {\n"
    "\t\t_ = c.ReadCloser.Close()\n"
    "\t\tif err := c.cmd.Wait(); err != nil {\n"
    "\t\t\tc.closeErr = fmt.Errorf(\"pg_dump exited with error: %w\", err)\n"
    "\t\t}\n"
    "\t})\n"
    "\treturn c.closeErr\n"
    "}\n",
)

local_test = Path("backend/internal/service/backup_local_service_test.go")
text = local_test.read_text()
marker = "func TestStopCancelsLocalStreamingBackupAfterGracePeriod"
if marker not in text:
    raise SystemExit("expected lifecycle test from first patch")
if "TestStopUnblocksLocalBackupWhenClientDoesNotRead" in text:
    raise SystemExit("blocked-client test already present")
text += r'''

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
'''
local_test.write_text(text)
