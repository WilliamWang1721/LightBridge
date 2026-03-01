# Chatbox Persistent Module (LightBridge)

`chatbox-persistent` 是一个可独立安装的持久化 Chatbox Tools 模块：

- 提供会话持久化 API（存储于 `LIGHTBRIDGE_DATA_DIR/chatbox_store.json`）
- 提供 OpenAI 兼容聊天代理端点（`/v1/chat/completions`）
- 提供健康检查端点（`/health`）

## 端点

- `GET /health`
- `GET /chatbox/conversations`
- `POST /chatbox/conversations`
- `GET /chatbox/conversations/{id}`
- `DELETE /chatbox/conversations/{id}`
- `POST /chatbox/conversations/{id}/messages`
- `POST /v1/chat/completions`（透传到上游）
- `GET /v1/models`

## 主要特性

- 所有会话与消息长期存储（模块重启后仍可读取）
- 会话列表按更新时间排序，包含最新消息摘要与消息数
- 消息发送自动拼接上下文（system prompt + 历史 user/assistant）
- `assistant` 返回中的 reasoning 字段会尽力提取并保存
- 支持 Markdown / LaTeX 内容的存储（渲染由前端负责）

## 配置

由 `manifest.json` 的 `config_schema` 定义，关键项：

- `base_url`：上游 OpenAI 兼容地址，默认 `https://api.openai.com/v1`
- `api_key`：上游 API Key（可留空并使用环境变量 `OPENAI_API_KEY`）
- `models`：可选模型列表
- `default_model`：默认模型
- `request_timeout_sec`：请求超时
- `extra_headers`：额外上游请求头

## 打包

```bash
bash modules/chatbox-persistent/package.sh
```

产物在：

- `modules/chatbox-persistent/dist/chatbox-persistent_0.1.0_universal.zip`
- `modules/chatbox-persistent/dist/index.json`

## 安装（本地 Marketplace）

```bash
cp modules/chatbox-persistent/dist/chatbox-persistent_0.1.0_universal.zip market/MODULES/chatbox-persistent.zip
```

在 Admin Marketplace 中将 source 设为 `local` 后安装并启动。
