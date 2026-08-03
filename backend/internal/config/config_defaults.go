package config

import (
	"time"

	"github.com/spf13/viper"
)

func setDefaults() {
	viper.SetDefault("run_mode", RunModeStandard)

	// Server
	viper.SetDefault("server.host", "0.0.0.0")
	viper.SetDefault("server.port", 8080)
	viper.SetDefault("server.mode", "release")
	viper.SetDefault("server.frontend_url", "")
	viper.SetDefault("server.read_header_timeout", 30) // 30秒读取请求头
	viper.SetDefault("server.idle_timeout", 120)       // 120秒空闲超时
	viper.SetDefault("server.trusted_proxies", []string{})
	viper.SetDefault("server.max_request_body_size", int64(256*1024*1024))
	// H2C 默认配置
	viper.SetDefault("server.h2c.enabled", false)
	viper.SetDefault("server.h2c.max_concurrent_streams", uint32(50))      // 50 个并发流
	viper.SetDefault("server.h2c.idle_timeout", 75)                        // 75 秒
	viper.SetDefault("server.h2c.max_read_frame_size", 1<<20)              // 1MB（够用）
	viper.SetDefault("server.h2c.max_upload_buffer_per_connection", 2<<20) // 2MB
	viper.SetDefault("server.h2c.max_upload_buffer_per_stream", 512<<10)   // 512KB

	// Log
	viper.SetDefault("log.level", "info")
	viper.SetDefault("log.format", "console")
	viper.SetDefault("log.service_name", "LightBridge")
	viper.SetDefault("log.env", "production")
	viper.SetDefault("log.caller", true)
	viper.SetDefault("log.stacktrace_level", "error")
	viper.SetDefault("log.output.to_stdout", true)
	viper.SetDefault("log.output.to_file", true)
	viper.SetDefault("log.output.file_path", "")
	viper.SetDefault("log.rotation.max_size_mb", 100)
	viper.SetDefault("log.rotation.max_backups", 10)
	viper.SetDefault("log.rotation.max_age_days", 7)
	viper.SetDefault("log.rotation.compress", true)
	viper.SetDefault("log.rotation.local_time", true)
	viper.SetDefault("log.sampling.enabled", false)
	viper.SetDefault("log.sampling.initial", 100)
	viper.SetDefault("log.sampling.thereafter", 100)

	// CORS
	viper.SetDefault("cors.allowed_origins", []string{})
	viper.SetDefault("cors.allow_credentials", true)

	// Security
	viper.SetDefault("security.url_allowlist.enabled", false)
	viper.SetDefault("security.url_allowlist.upstream_hosts", []string{
		"api.openai.com",
		"api.anthropic.com",
		"api.kimi.com",
		"open.bigmodel.cn",
		"api.minimaxi.com",
		"generativelanguage.googleapis.com",
		"cloudcode-pa.googleapis.com",
		"*.openai.azure.com",
	})
	viper.SetDefault("security.url_allowlist.pricing_hosts", []string{
		"raw.githubusercontent.com",
	})
	viper.SetDefault("security.url_allowlist.crs_hosts", []string{})
	viper.SetDefault("security.url_allowlist.allow_private_hosts", true)
	viper.SetDefault("security.url_allowlist.allow_insecure_http", true)
	viper.SetDefault("security.response_headers.enabled", true)
	viper.SetDefault("security.response_headers.additional_allowed", []string{})
	viper.SetDefault("security.response_headers.force_remove", []string{})
	viper.SetDefault("security.csp.enabled", true)
	viper.SetDefault("security.csp.policy", DefaultCSPPolicy)
	viper.SetDefault("security.proxy_probe.insecure_skip_verify", false)
	viper.SetDefault("security.trust_forwarded_ip_for_api_key_acl", false)

	// Security - disable direct fallback on proxy error
	viper.SetDefault("security.proxy_fallback.allow_direct_on_error", false)

	// Billing
	viper.SetDefault("billing.circuit_breaker.enabled", true)
	viper.SetDefault("billing.circuit_breaker.failure_threshold", 5)
	viper.SetDefault("billing.circuit_breaker.reset_timeout_seconds", 30)
	viper.SetDefault("billing.circuit_breaker.half_open_requests", 3)
	viper.SetDefault("billing.user_platform_quota_cache_ttl_seconds", 86400)
	viper.SetDefault("billing.user_platform_quota_sentinel_ttl_seconds", 3600)

	// Progressive features. Full preserves the historical all-in-one runtime;
	// standard and minimal are explicit resource-saving deployment choices.
	viper.SetDefault("features.profile", string(FeatureProfileFull))
	viper.SetDefault("features.overrides", map[string]bool{})

	// Modules
	viper.SetDefault("modules.data_dir", "data")
	viper.SetDefault("modules.signature_public_key_path", "")
	viper.SetDefault("modules.marketplace_registry_path", "")
	viper.SetDefault("modules.marketplace_registry_url", DefaultManagedProviderRegistryURL)
	viper.SetDefault("modules.marketplace_timeout_seconds", 60)
	viper.SetDefault("modules.proxy.mihomo_binary_path", "")
	viper.SetDefault("modules.proxy.runtime_dir", "")

	// Turnstile
	viper.SetDefault("turnstile.required", false)

	// LinuxDo Connect OAuth 登录
	viper.SetDefault("linuxdo_connect.enabled", false)
	viper.SetDefault("linuxdo_connect.client_id", "")
	viper.SetDefault("linuxdo_connect.client_secret", "")
	viper.SetDefault("linuxdo_connect.authorize_url", "https://connect.linux.do/oauth2/authorize")
	viper.SetDefault("linuxdo_connect.token_url", "https://connect.linux.do/oauth2/token")
	viper.SetDefault("linuxdo_connect.userinfo_url", "https://connect.linux.do/api/user")
	viper.SetDefault("linuxdo_connect.scopes", "user")
	viper.SetDefault("linuxdo_connect.redirect_url", "")
	viper.SetDefault("linuxdo_connect.frontend_redirect_url", "/auth/linuxdo/callback")
	viper.SetDefault("linuxdo_connect.token_auth_method", "client_secret_post")
	viper.SetDefault("linuxdo_connect.use_pkce", false)
	viper.SetDefault("linuxdo_connect.userinfo_email_path", "")
	viper.SetDefault("linuxdo_connect.userinfo_id_path", "")
	viper.SetDefault("linuxdo_connect.userinfo_username_path", "")

	// WeChat Connect OAuth 登录
	viper.SetDefault("wechat_connect.enabled", false)
	viper.SetDefault("wechat_connect.app_id", "")
	viper.SetDefault("wechat_connect.app_secret", "")
	viper.SetDefault("wechat_connect.open_app_id", "")
	viper.SetDefault("wechat_connect.open_app_secret", "")
	viper.SetDefault("wechat_connect.mp_app_id", "")
	viper.SetDefault("wechat_connect.mp_app_secret", "")
	viper.SetDefault("wechat_connect.mobile_app_id", "")
	viper.SetDefault("wechat_connect.mobile_app_secret", "")
	viper.SetDefault("wechat_connect.open_enabled", false)
	viper.SetDefault("wechat_connect.mp_enabled", false)
	viper.SetDefault("wechat_connect.mobile_enabled", false)
	viper.SetDefault("wechat_connect.mode", defaultWeChatConnectMode)
	viper.SetDefault("wechat_connect.scopes", defaultWeChatConnectScopes)
	viper.SetDefault("wechat_connect.redirect_url", "")
	viper.SetDefault("wechat_connect.frontend_redirect_url", defaultWeChatConnectFrontendRedirect)

	// Generic OIDC OAuth 登录
	viper.SetDefault("oidc_connect.enabled", false)
	viper.SetDefault("oidc_connect.provider_name", "OIDC")
	viper.SetDefault("oidc_connect.client_id", "")
	viper.SetDefault("oidc_connect.client_secret", "")
	viper.SetDefault("oidc_connect.issuer_url", "")
	viper.SetDefault("oidc_connect.discovery_url", "")
	viper.SetDefault("oidc_connect.authorize_url", "")
	viper.SetDefault("oidc_connect.token_url", "")
	viper.SetDefault("oidc_connect.userinfo_url", "")
	viper.SetDefault("oidc_connect.jwks_url", "")
	viper.SetDefault("oidc_connect.scopes", "openid email profile")
	viper.SetDefault("oidc_connect.redirect_url", "")
	viper.SetDefault("oidc_connect.frontend_redirect_url", "/auth/oidc/callback")
	viper.SetDefault("oidc_connect.token_auth_method", "client_secret_post")
	viper.SetDefault("oidc_connect.use_pkce", true)
	viper.SetDefault("oidc_connect.validate_id_token", true)
	viper.SetDefault("oidc_connect.allowed_signing_algs", "RS256,ES256,PS256")
	viper.SetDefault("oidc_connect.clock_skew_seconds", 120)
	viper.SetDefault("oidc_connect.require_email_verified", false)
	viper.SetDefault("oidc_connect.userinfo_email_path", "")
	viper.SetDefault("oidc_connect.userinfo_id_path", "")
	viper.SetDefault("oidc_connect.userinfo_username_path", "")

	// DingTalk Connect OAuth 登录
	viper.SetDefault("dingtalk_connect.enabled", false)
	viper.SetDefault("dingtalk_connect.authorize_url", "https://login.dingtalk.com/oauth2/auth")
	viper.SetDefault("dingtalk_connect.token_url", "https://api.dingtalk.com/v1.0/oauth2/userAccessToken")
	viper.SetDefault("dingtalk_connect.userinfo_url", "https://api.dingtalk.com/v1.0/contact/users/me")
	viper.SetDefault("dingtalk_connect.scopes", "openid")
	viper.SetDefault("dingtalk_connect.frontend_redirect_url", "/auth/dingtalk/callback")
	viper.SetDefault("dingtalk_connect.dingtalk_app_kind", "internal_app")
	viper.SetDefault("dingtalk_connect.app_type", "public")
	viper.SetDefault("dingtalk_connect.corp_restriction_policy", "none")
	viper.SetDefault("dingtalk_connect.require_email", true)
	viper.SetDefault("dingtalk_connect.username_overwrite_policy", "if_empty")

	// Database
	viper.SetDefault("database.host", "localhost")
	viper.SetDefault("database.port", 5432)
	viper.SetDefault("database.user", "postgres")
	viper.SetDefault("database.password", "postgres")
	viper.SetDefault("database.dbname", "LightBridge")
	viper.SetDefault("database.sslmode", "prefer")
	viper.SetDefault("database.max_open_conns", 256)
	viper.SetDefault("database.max_idle_conns", 128)
	viper.SetDefault("database.conn_max_lifetime_minutes", 30)
	viper.SetDefault("database.conn_max_idle_time_minutes", 5)
	viper.SetDefault("database.user_platform_quota_flusher_enabled", false)
	viper.SetDefault("database.user_platform_quota_flush_interval_ms", 2000)
	viper.SetDefault("database.user_platform_quota_flush_batch_size", 1000)

	// Redis
	viper.SetDefault("redis.host", "localhost")
	viper.SetDefault("redis.port", 6379)
	viper.SetDefault("redis.password", "")
	viper.SetDefault("redis.db", 0)
	viper.SetDefault("redis.dial_timeout_seconds", 5)
	viper.SetDefault("redis.read_timeout_seconds", 3)
	viper.SetDefault("redis.write_timeout_seconds", 3)
	viper.SetDefault("redis.pool_size", 1024)
	viper.SetDefault("redis.min_idle_conns", 128)
	viper.SetDefault("redis.enable_tls", false)

	// Ops (vNext)
	viper.SetDefault("ops.enabled", true)
	viper.SetDefault("ops.use_preaggregated_tables", true)
	viper.SetDefault("ops.cleanup.enabled", true)
	viper.SetDefault("ops.cleanup.schedule", "0 2 * * *")
	// Retention days: vNext defaults to 30 days across ops datasets.
	viper.SetDefault("ops.cleanup.error_log_retention_days", 30)
	viper.SetDefault("ops.cleanup.minute_metrics_retention_days", 30)
	viper.SetDefault("ops.cleanup.hourly_metrics_retention_days", 30)
	viper.SetDefault("ops.aggregation.enabled", true)
	viper.SetDefault("ops.metrics_collector_cache.enabled", true)
	// TTL should be slightly larger than collection interval (1m) to maximize cross-replica cache hits.
	viper.SetDefault("ops.metrics_collector_cache.ttl", 65*time.Second)

	// JWT
	viper.SetDefault("jwt.secret", "")
	viper.SetDefault("jwt.expire_hour", 24)
	viper.SetDefault("jwt.access_token_expire_minutes", 0) // 0 表示回退到 expire_hour
	viper.SetDefault("jwt.refresh_token_expire_days", 30)  // 30天Refresh Token有效期
	viper.SetDefault("jwt.refresh_window_minutes", 2)      // 过期前2分钟开始允许刷新

	// TOTP
	viper.SetDefault("totp.encryption_key", "")

	// Default
	// Admin credentials are created via the setup flow (web wizard / CLI / AUTO_SETUP).
	// Do not ship fixed defaults here to avoid insecure "known credentials" in production.
	viper.SetDefault("default.admin_email", "")
	viper.SetDefault("default.admin_password", "")
	viper.SetDefault("default.user_concurrency", 5)
	viper.SetDefault("default.user_balance", 0)
	viper.SetDefault("default.api_key_prefix", "sk-")
	viper.SetDefault("default.rate_multiplier", 1.0)

	// RateLimit
	viper.SetDefault("rate_limit.overload_cooldown_minutes", 10)
	viper.SetDefault("rate_limit.oauth_401_cooldown_minutes", 10)

	// Pricing - 从 model-price-repo 同步模型定价和上下文窗口数据（固定到 commit，避免分支漂移）
	viper.SetDefault("pricing.remote_url", "https://raw.githubusercontent.com/Wei-Shaw/model-price-repo/main/model_prices_and_context_window.json")
	viper.SetDefault("pricing.hash_url", "https://raw.githubusercontent.com/Wei-Shaw/model-price-repo/main/model_prices_and_context_window.sha256")
	viper.SetDefault("pricing.data_dir", "./data")
	viper.SetDefault("pricing.fallback_file", "./resources/model-pricing/model_prices_and_context_window.json")
	viper.SetDefault("pricing.update_interval_hours", 24)
	viper.SetDefault("pricing.hash_check_interval_minutes", 10)

	// Timezone (default to Asia/Shanghai for Chinese users)
	viper.SetDefault("timezone", "Asia/Shanghai")

	// Language for API error messages returned to clients (en|zh). Per-request
	// Accept-Language overrides this default.
	viper.SetDefault("language", "en")

	// API Key auth cache
	viper.SetDefault("api_key_auth_cache.l1_size", 65535)
	viper.SetDefault("api_key_auth_cache.l1_ttl_seconds", 15)
	viper.SetDefault("api_key_auth_cache.l2_ttl_seconds", 300)
	viper.SetDefault("api_key_auth_cache.negative_ttl_seconds", 30)
	viper.SetDefault("api_key_auth_cache.jitter_percent", 10)
	viper.SetDefault("api_key_auth_cache.singleflight", true)

	// Subscription auth L1 cache
	viper.SetDefault("subscription_cache.l1_size", 16384)
	viper.SetDefault("subscription_cache.l1_ttl_seconds", 10)
	viper.SetDefault("subscription_cache.jitter_percent", 10)

	// Dashboard cache
	viper.SetDefault("dashboard_cache.enabled", true)
	viper.SetDefault("dashboard_cache.key_prefix", "LightBridge:")
	viper.SetDefault("dashboard_cache.stats_fresh_ttl_seconds", 15)
	viper.SetDefault("dashboard_cache.stats_ttl_seconds", 30)
	viper.SetDefault("dashboard_cache.stats_refresh_timeout_seconds", 30)

	// Dashboard aggregation
	viper.SetDefault("dashboard_aggregation.enabled", true)
	viper.SetDefault("dashboard_aggregation.interval_seconds", 60)
	viper.SetDefault("dashboard_aggregation.lookback_seconds", 120)
	viper.SetDefault("dashboard_aggregation.backfill_enabled", false)
	viper.SetDefault("dashboard_aggregation.backfill_max_days", 31)
	viper.SetDefault("dashboard_aggregation.retention.usage_logs_days", 90)
	viper.SetDefault("dashboard_aggregation.retention.usage_billing_dedup_days", 365)
	viper.SetDefault("dashboard_aggregation.retention.hourly_days", 180)
	viper.SetDefault("dashboard_aggregation.retention.daily_days", 730)
	viper.SetDefault("dashboard_aggregation.recompute_days", 2)

	// Usage cleanup task
	viper.SetDefault("usage_cleanup.enabled", true)
	viper.SetDefault("usage_cleanup.max_range_days", 31)
	viper.SetDefault("usage_cleanup.batch_size", 5000)
	viper.SetDefault("usage_cleanup.worker_interval_seconds", 10)
	viper.SetDefault("usage_cleanup.task_timeout_seconds", 1800)

	// Idempotency
	viper.SetDefault("idempotency.observe_only", true)
	viper.SetDefault("idempotency.default_ttl_seconds", 86400)
	viper.SetDefault("idempotency.system_operation_ttl_seconds", 3600)
	viper.SetDefault("idempotency.processing_timeout_seconds", 30)
	viper.SetDefault("idempotency.failed_retry_backoff_seconds", 5)
	viper.SetDefault("idempotency.max_stored_response_len", 64*1024)
	viper.SetDefault("idempotency.cleanup_interval_seconds", 60)
	viper.SetDefault("idempotency.cleanup_batch_size", 500)

	// Gateway
	viper.SetDefault("gateway.response_header_timeout", 600) // 600秒(10分钟)等待上游响应头，LLM高负载时可能排队较久
	viper.SetDefault("gateway.openai_response_header_timeout", 0)
	viper.SetDefault("gateway.log_upstream_error_body", true)
	viper.SetDefault("gateway.log_upstream_error_body_max_bytes", 2048)
	viper.SetDefault("gateway.inject_beta_for_apikey", false)
	viper.SetDefault("gateway.failover_on_400", false)
	viper.SetDefault("gateway.max_account_switches", 10)
	viper.SetDefault("gateway.max_account_switches_gemini", 3)
	viper.SetDefault("gateway.force_codex_cli", false)
	viper.SetDefault("gateway.codex_image_generation_bridge_enabled", false)
	viper.SetDefault("gateway.openai_passthrough_allow_timeout_headers", false)
	// OpenAI Responses WebSocket（默认开启；可通过 force_http 紧急回滚）
	viper.SetDefault("gateway.openai_ws.enabled", true)
	viper.SetDefault("gateway.openai_ws.mode_router_v2_enabled", false)
	viper.SetDefault("gateway.openai_ws.ingress_mode_default", "ctx_pool")
	viper.SetDefault("gateway.openai_ws.oauth_enabled", true)
	viper.SetDefault("gateway.openai_ws.apikey_enabled", true)
	viper.SetDefault("gateway.openai_ws.force_http", false)
	viper.SetDefault("gateway.openai_ws.allow_store_recovery", false)
	viper.SetDefault("gateway.openai_ws.ingress_previous_response_recovery_enabled", true)
	viper.SetDefault("gateway.openai_ws.store_disabled_conn_mode", "strict")
	viper.SetDefault("gateway.openai_ws.store_disabled_force_new_conn", true)
	viper.SetDefault("gateway.openai_ws.prewarm_generate_enabled", false)
	viper.SetDefault("gateway.openai_ws.client_read_limit_bytes", 64*1024*1024)
	viper.SetDefault("gateway.openai_ws.http_bridge_enabled", true)
	viper.SetDefault("gateway.openai_ws.http_bridge_threshold_bytes", 15*1024*1024)
	viper.SetDefault("gateway.openai_ws.responses_websockets", false)
	viper.SetDefault("gateway.openai_ws.responses_websockets_v2", true)
	viper.SetDefault("gateway.openai_ws.max_conns_per_account", 128)
	viper.SetDefault("gateway.openai_ws.min_idle_per_account", 4)
	viper.SetDefault("gateway.openai_ws.max_idle_per_account", 12)
	viper.SetDefault("gateway.openai_ws.dynamic_max_conns_by_account_concurrency_enabled", true)
	viper.SetDefault("gateway.openai_ws.oauth_max_conns_factor", 1.0)
	viper.SetDefault("gateway.openai_ws.apikey_max_conns_factor", 1.0)
	viper.SetDefault("gateway.openai_ws.dial_timeout_seconds", 10)
	viper.SetDefault("gateway.openai_ws.read_timeout_seconds", 900)
	viper.SetDefault("gateway.openai_ws.write_timeout_seconds", 120)
	viper.SetDefault("gateway.openai_ws.pool_target_utilization", 0.7)
	viper.SetDefault("gateway.openai_ws.queue_limit_per_conn", 64)
	viper.SetDefault("gateway.openai_ws.event_flush_batch_size", 1)
	viper.SetDefault("gateway.openai_ws.event_flush_interval_ms", 10)
	viper.SetDefault("gateway.openai_ws.prewarm_cooldown_ms", 300)
	viper.SetDefault("gateway.openai_ws.fallback_cooldown_seconds", 30)
	viper.SetDefault("gateway.openai_ws.retry_backoff_initial_ms", 120)
	viper.SetDefault("gateway.openai_ws.retry_backoff_max_ms", 2000)
	viper.SetDefault("gateway.openai_ws.retry_jitter_ratio", 0.2)
	viper.SetDefault("gateway.openai_ws.retry_total_budget_ms", 5000)
	viper.SetDefault("gateway.openai_ws.payload_log_sample_rate", 0.2)
	viper.SetDefault("gateway.openai_ws.lb_top_k", 7)
	viper.SetDefault("gateway.openai_ws.sticky_session_ttl_seconds", 3600)
	viper.SetDefault("gateway.openai_ws.session_hash_read_old_fallback", true)
	viper.SetDefault("gateway.openai_ws.session_hash_dual_write_old", true)
	viper.SetDefault("gateway.openai_ws.metadata_bridge_enabled", true)
	viper.SetDefault("gateway.openai_ws.sticky_response_id_ttl_seconds", 3600)
	viper.SetDefault("gateway.openai_ws.sticky_previous_response_ttl_seconds", 3600)
	viper.SetDefault("gateway.openai_ws.scheduler_score_weights.priority", 1.0)
	viper.SetDefault("gateway.openai_ws.scheduler_score_weights.load", 1.0)
	viper.SetDefault("gateway.openai_ws.scheduler_score_weights.queue", 0.7)
	viper.SetDefault("gateway.openai_ws.scheduler_score_weights.error_rate", 0.8)
	viper.SetDefault("gateway.openai_ws.scheduler_score_weights.ttft", 0.5)
	// OpenAI HTTP upstream protocol strategy
	viper.SetDefault("gateway.openai_http2.enabled", true)
	viper.SetDefault("gateway.openai_http2.allow_proxy_fallback_to_http1", true)
	viper.SetDefault("gateway.openai_http2.fallback_error_threshold", 2)
	viper.SetDefault("gateway.openai_http2.fallback_window_seconds", 60)
	viper.SetDefault("gateway.openai_http2.fallback_ttl_seconds", 600)
	viper.SetDefault("gateway.image_concurrency.enabled", false)
	viper.SetDefault("gateway.image_concurrency.max_concurrent_requests", 0)
	viper.SetDefault("gateway.image_concurrency.overflow_mode", ImageConcurrencyOverflowModeReject)
	viper.SetDefault("gateway.image_concurrency.wait_timeout_seconds", 30)
	viper.SetDefault("gateway.image_concurrency.max_waiting_requests", 100)
	viper.SetDefault("gateway.antigravity_fallback_cooldown_minutes", 1)
	viper.SetDefault("gateway.antigravity_extra_retries", 10)
	viper.SetDefault("gateway.max_body_size", int64(256*1024*1024))
	viper.SetDefault("gateway.upstream_response_read_max_bytes", DefaultUpstreamResponseReadMaxBytes)
	viper.SetDefault("gateway.proxy_probe_response_read_max_bytes", int64(1024*1024))
	viper.SetDefault("gateway.gemini_debug_response_headers", false)
	viper.SetDefault("gateway.connection_pool_isolation", ConnectionPoolIsolationAccountProxy)
	// HTTP 上游连接池配置（针对 5000+ 并发用户优化）
	viper.SetDefault("gateway.max_idle_conns", 2560)          // 最大空闲连接总数（高并发场景可调大）
	viper.SetDefault("gateway.max_idle_conns_per_host", 120)  // 每主机最大空闲连接（HTTP/2 场景默认）
	viper.SetDefault("gateway.max_conns_per_host", 1024)      // 每主机最大连接数（含活跃；流式/HTTP1.1 场景可调大，如 2400+）
	viper.SetDefault("gateway.idle_conn_timeout_seconds", 90) // 空闲连接超时（秒）
	viper.SetDefault("gateway.max_upstream_clients", 5000)
	viper.SetDefault("gateway.client_idle_ttl_seconds", 900)
	viper.SetDefault("gateway.concurrency_slot_ttl_minutes", 30) // 并发槽位过期时间（支持超长请求）
	viper.SetDefault("gateway.stream_data_interval_timeout", 180)
	viper.SetDefault("gateway.stream_keepalive_interval", 10)
	viper.SetDefault("gateway.image_stream_data_interval_timeout", 900)
	viper.SetDefault("gateway.image_stream_keepalive_interval", 10)
	viper.SetDefault("gateway.max_line_size", 500*1024*1024)
	viper.SetDefault("gateway.scheduling.sticky_session_max_waiting", 3)
	viper.SetDefault("gateway.scheduling.sticky_session_wait_timeout", 120*time.Second)
	viper.SetDefault("gateway.scheduling.fallback_wait_timeout", 30*time.Second)
	viper.SetDefault("gateway.scheduling.fallback_max_waiting", 100)
	viper.SetDefault("gateway.scheduling.fallback_selection_mode", "last_used")
	viper.SetDefault("gateway.scheduling.load_batch_enabled", true)
	viper.SetDefault("gateway.scheduling.load_batch_cache_ttl_ms", 200)
	viper.SetDefault("gateway.scheduling.snapshot_mget_chunk_size", 128)
	viper.SetDefault("gateway.scheduling.snapshot_write_chunk_size", 256)
	viper.SetDefault("gateway.scheduling.slot_cleanup_interval", 30*time.Second)
	viper.SetDefault("gateway.scheduling.db_fallback_enabled", true)
	viper.SetDefault("gateway.scheduling.db_fallback_timeout_seconds", 0)
	viper.SetDefault("gateway.scheduling.db_fallback_max_qps", 0)
	viper.SetDefault("gateway.scheduling.outbox_poll_interval_seconds", 1)
	viper.SetDefault("gateway.scheduling.outbox_lag_warn_seconds", 5)
	viper.SetDefault("gateway.scheduling.outbox_lag_rebuild_seconds", 10)
	viper.SetDefault("gateway.scheduling.outbox_lag_rebuild_failures", 3)
	viper.SetDefault("gateway.scheduling.outbox_backlog_rebuild_rows", 10000)
	viper.SetDefault("gateway.scheduling.full_rebuild_interval_seconds", 300)
	viper.SetDefault("gateway.usage_record.worker_count", 128)
	viper.SetDefault("gateway.usage_record.queue_size", 16384)
	viper.SetDefault("gateway.usage_record.task_timeout_seconds", 30)
	viper.SetDefault("gateway.usage_record.overflow_policy", UsageRecordOverflowPolicySync)
	viper.SetDefault("gateway.usage_record.overflow_sample_percent", 10)
	viper.SetDefault("gateway.usage_record.auto_scale_enabled", true)
	viper.SetDefault("gateway.usage_record.auto_scale_min_workers", 128)
	viper.SetDefault("gateway.usage_record.auto_scale_max_workers", 512)
	viper.SetDefault("gateway.usage_record.auto_scale_up_queue_percent", 70)
	viper.SetDefault("gateway.usage_record.auto_scale_down_queue_percent", 15)
	viper.SetDefault("gateway.usage_record.auto_scale_up_step", 32)
	viper.SetDefault("gateway.usage_record.auto_scale_down_step", 16)
	viper.SetDefault("gateway.usage_record.auto_scale_check_interval_seconds", 3)
	viper.SetDefault("gateway.usage_record.auto_scale_cooldown_seconds", 10)
	viper.SetDefault("gateway.user_group_rate_cache_ttl_seconds", 30)
	viper.SetDefault("gateway.models_list_cache_ttl_seconds", 15)
	// TLS指纹伪装配置（默认关闭，需要账号级别单独启用）
	// 用户消息串行队列默认值
	viper.SetDefault("gateway.user_message_queue.enabled", false)
	viper.SetDefault("gateway.user_message_queue.lock_ttl_ms", 120000)
	viper.SetDefault("gateway.user_message_queue.wait_timeout_ms", 30000)
	viper.SetDefault("gateway.user_message_queue.min_delay_ms", 200)
	viper.SetDefault("gateway.user_message_queue.max_delay_ms", 2000)
	viper.SetDefault("gateway.user_message_queue.cleanup_interval_seconds", 60)

	viper.SetDefault("gateway.tls_fingerprint.enabled", true)
	viper.SetDefault("concurrency.ping_interval", 10)

	// TokenRefresh
	viper.SetDefault("token_refresh.enabled", true)
	viper.SetDefault("token_refresh.check_interval_minutes", 5)        // 每5分钟检查一次
	viper.SetDefault("token_refresh.refresh_before_expiry_hours", 0.5) // 提前30分钟刷新（适配Google 1小时token）
	viper.SetDefault("token_refresh.max_retries", 3)                   // 最多重试3次
	viper.SetDefault("token_refresh.retry_backoff_seconds", 2)         // 重试退避基础2秒

	// Gemini OAuth - configure via environment variables or config file
	// GEMINI_OAUTH_CLIENT_ID and GEMINI_OAUTH_CLIENT_SECRET
	// Default: uses Gemini CLI public credentials (set via environment)
	viper.SetDefault("gemini.oauth.client_id", "")
	viper.SetDefault("gemini.oauth.client_secret", "")
	viper.SetDefault("gemini.oauth.scopes", "")
	viper.SetDefault("gemini.quota.policy", "")

	// Subscription Maintenance (bounded queue + worker pool)
	viper.SetDefault("subscription_maintenance.worker_count", 2)
	viper.SetDefault("subscription_maintenance.queue_size", 1024)

}
