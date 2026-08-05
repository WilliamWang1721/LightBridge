.PHONY: build build-backend build-frontend test test-backend test-frontend test-frontend-critical audit-ui test-model-sync-smoke model-sync-smoke secret-scan audit-codebase check-runtime-contract runtime-smoke

FRONTEND_CRITICAL_VITEST := \
	src/views/auth/__tests__/LinuxDoCallbackView.spec.ts \
	src/views/auth/__tests__/WechatCallbackView.spec.ts \
	src/views/user/__tests__/PaymentView.spec.ts \
	src/views/user/__tests__/PaymentResultView.spec.ts \
	src/components/user/profile/__tests__/ProfileInfoCard.spec.ts \
	src/views/admin/__tests__/SettingsView.spec.ts \
	src/views/admin/__tests__/VersionControlView.spec.ts \
	src/api/__tests__/admin.features.spec.ts \
	src/api/admin/__tests__/backup.spec.ts \
	src/views/admin/__tests__/FeatureRegistryView.spec.ts \
	src/router/__tests__/progressive-routes.spec.ts \
	src/components/ui/__tests__/primitives.spec.ts \
	src/ui-platform/chartTheme.test.ts \
	src/ui-platform/icons.test.ts \
	src/ui-platform/inputBridge.test.ts \
	src/ui-platform/resolver.test.ts \
	src/ui-platform/routeSurface.test.ts \
	src/ui-platform/themePackages.test.ts

# 一键编译前后端
build: build-backend build-frontend

# 编译后端（复用 backend/Makefile）
build-backend:
	@$(MAKE) -C backend build

# 编译前端（需要已安装依赖）
build-frontend:
	@pnpm --dir frontend run build

# 运行测试（后端 + 前端）
test: test-backend test-frontend

test-backend:
	@$(MAKE) -C backend test

test-frontend:
	@$(MAKE) audit-ui
	@pnpm --dir frontend run lint:check
	@pnpm --dir frontend run typecheck
	@$(MAKE) test-frontend-critical

test-frontend-critical:
	@pnpm --dir frontend exec vitest run $(FRONTEND_CRITICAL_VITEST)

audit-ui:
	@python3 tools/ui_migration_audit.py

test-model-sync-smoke:
	@cd backend && go test ./cmd/model-sync-smoke

model-sync-smoke:
	@cd backend && go run ./cmd/model-sync-smoke $(ARGS)

secret-scan:
	@python3 tools/secret_scan.py

# Verify that the checked-in line-level repository inventory is current.
audit-codebase:
	@python3 tools/codebase_inventory.py --check

# Validate the checked-in toolchain, lockfiles, container, and Compose contract.
check-runtime-contract:
	@python3 tools/ci/check_environment_contract.py

# Build lightbridge:ci first, then run the same full-stack smoke checks used by GitHub Actions.
runtime-smoke:
	@bash tools/ci/full_stack_smoke.sh
