# Advanced Statistics

该模块为 LightBridge Admin 的“统计（Statistics）”页面提供高级统计聚合能力。

## 运行方式

该模块由 LightBridge Core 以子进程启动，通过环境变量注入运行参数：

- `LIGHTBRIDGE_HTTP_PORT`：模块监听端口（仅绑定 `127.0.0.1`）
- `LIGHTBRIDGE_DATA_DIR`：模块私有数据目录
- `LIGHTBRIDGE_CONFIG_PATH`：模块配置文件路径

## 端点

- `GET /health`：健康检查
- `POST /stats/aggregate`：统计聚合接口（供 Core `proxyModuleHTTP` 调用）

## `/stats/aggregate` 输入概要

```json
{
  "start": "2026-03-01T00:00:00Z",
  "end": "2026-03-01T23:59:59Z",
  "bucket_seconds": 300,
  "window_logs": [
    {
      "timestamp": "2026-03-01T09:18:12Z",
      "model_id": "gpt-4o-mini",
      "input_tokens": 120,
      "output_tokens": 80,
      "reasoning_tokens": 10,
      "cached_tokens": 15
    }
  ],
  "today_logs": []
}
```

模块会返回：

- 今日与时间范围统计汇总（请求数、Token 分类、估算费用）
- 按模型聚合统计（含占比）
- 趋势序列（按 bucket 聚合，供折线图）

> 说明：费用为估算值，按内置模型费率规则折算，未知模型会使用默认估算费率。
