#!/usr/bin/env python3
from pathlib import Path

path = Path("backend/internal/service/backup_shutdown_test.go")
text = path.read_text()
if '"context"' not in text:
    text = text.replace('import (\n', 'import (\n\t"context"\n', 1)
old = "finish, err := s.beginTrackedOperation()"
if text.count(old) != 2:
    raise SystemExit(f"expected two admitted-operation calls, found {text.count(old)}")
text = text.replace(old, "_, finish, err := s.beginTrackedOperation(context.Background())")
old_rejected = "_, err := s.beginTrackedOperation()"
if text.count(old_rejected) != 1:
    raise SystemExit(f"expected one rejected-operation call, found {text.count(old_rejected)}")
text = text.replace(old_rejected, "_, _, err := s.beginTrackedOperation(context.Background())")
path.write_text(text)
