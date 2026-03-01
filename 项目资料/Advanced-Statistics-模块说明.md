# Advanced Statistics 模块说明

## 模块信息

- 模块 ID：`advanced-statistics`
- 模块名：`Advanced Statistics`
- 版本：`0.1.0`
- 协议：`http_rpc`
- 健康检查：`GET /health`

## 功能目标

安装并启用模块后，LightBridge 管理台的“统计（Statistics）”页面可通过模块聚合接口获得更详细的数据：

1. 当日请求数量
2. 今日 Token 与金额估算
3. 标准 / 推理 / 缓存读取 Token 分类
4. 按模型分布（饼图）
5. 按模型详细统计表（请求数、Token、费用、占比）
6. 指定时间范围（分钟/秒级）折线图

## 接口

### `POST /stats/aggregate`

Core 会将筛选后的日志数据转发给模块，模块返回聚合结果。

请求体关键字段：

- `start` / `end`：RFC3339 时间范围
- `bucket_seconds`：趋势聚合粒度（秒）
- `window_logs[]`：时间范围日志
- `today_logs[]`：今日日志

响应关键字段：

- `today`
- `window`
- `token_breakdown`
- `model_usage[]`
- `trend[]`

## 打包

```bash
bash modules/advanced-statistics/package.sh
```

产物目录：`modules/advanced-statistics/dist/`

- `advanced-statistics_0.1.0_universal.zip`
- `bin/darwin/arm64/advanced-statistics`
- `bin/darwin/amd64/advanced-statistics`
- `bin/linux/arm64/advanced-statistics`
- `bin/linux/amd64/advanced-statistics`

## 发布（参考《模块开发与发布指南》）

1. 生成 zip 并计算 SHA256
2. 打 tag：`module-advanced-statistics-v0.1.0`
3. 创建 GitHub Release 并上传 zip
4. 更新 `market/MODULES/index.json`

