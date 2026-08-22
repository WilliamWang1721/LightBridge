const style = `
.lbms { font-family: Inter, ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif; padding: 20px; color: #111827; }
.lbms h2 { margin: 0 0 8px; font-size: 20px; font-weight: 700; }
.lbms h3 { margin: 20px 0 8px; font-size: 15px; font-weight: 650; }
.lbms p { margin: 0 0 14px; color: #4b5563; font-size: 13px; line-height: 1.55; }
.lbms .grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(180px, 1fr)); gap: 12px; margin: 16px 0; }
.lbms .card { border: 1px solid #e5e7eb; border-radius: 10px; padding: 14px; background: #fff; }
.lbms .metric { color: #6b7280; font-size: 12px; }
.lbms .value { margin-top: 6px; font-size: 20px; font-weight: 700; }
.lbms label { display: block; margin: 12px 0 6px; color: #374151; font-size: 12px; font-weight: 600; }
.lbms input, .lbms select { box-sizing: border-box; width: 100%; border: 1px solid #d1d5db; border-radius: 8px; padding: 9px 10px; font-size: 13px; }
.lbms button { border: 1px solid #111827; border-radius: 8px; background: #111827; color: #fff; padding: 8px 12px; font-size: 13px; cursor: pointer; margin: 4px 6px 4px 0; }
.lbms button.secondary { background: #fff; color: #111827; }
.lbms button:disabled { opacity: 0.55; cursor: not-allowed; }
.lbms table { width: 100%; border-collapse: collapse; font-size: 13px; margin-top: 12px; }
.lbms th, .lbms td { text-align: left; border-bottom: 1px solid #e5e7eb; padding: 10px 8px; }
.lbms th { color: #374151; font-weight: 650; background: #f9fafb; }
.lbms .hint { color: #6b7280; font-size: 12px; }
.lbms .danger { color: #b91c1c; }
.lbms .ok { color: #047857; }
.lbms .pill { display: inline-block; border: 1px solid #d1d5db; border-radius: 999px; padding: 2px 8px; font-size: 12px; background: #f9fafb; }
`;

function ensureStyle() {
  if (document.getElementById("lbms-style")) return;
  const node = document.createElement("style");
  node.id = "lbms-style";
  node.textContent = style;
  document.head.appendChild(node);
}

function apiBase() {
  return localStorage.getItem("lbms.baseUrl") || "http://127.0.0.1:8091";
}

function apiKey() {
  return localStorage.getItem("lbms.apiKey") || "";
}

async function lbmsFetch(path, options = {}) {
  const headers = new Headers(options.headers || {});
  if (apiKey()) headers.set("Authorization", `Bearer ${apiKey()}`);
  if (!headers.has("Content-Type") && options.body) headers.set("Content-Type", "application/json");
  const res = await fetch(`${apiBase()}${path}`, { ...options, headers });
  const data = await res.json().catch(() => ({}));
  if (!res.ok) throw new Error(data?.error?.message || "LightBridge Mail Service request failed");
  return data;
}

function mailboxRows(payload) {
  return payload?.data?.mailboxes || [];
}

function formatTime(value) {
  if (!value) return "—";
  try {
    return new Date(value).toLocaleString();
  } catch {
    return value;
  }
}

const MailServiceHome = {
  name: "MailServiceHome",
  data() {
    return {
      health: null,
      mailboxes: [],
      loading: false,
      error: "",
    };
  },
  mounted() {
    ensureStyle();
    this.refresh();
  },
  computed: {
    bindingTotal() {
      return this.mailboxes.reduce((sum, item) => sum + Number(item.binding_count || 0), 0);
    },
  },
  methods: {
    async refresh() {
      this.loading = true;
      this.error = "";
      try {
        const [health, mailboxPayload] = await Promise.all([
          lbmsFetch("/mail/v1/health"),
          lbmsFetch("/mail/v1/mailboxes"),
        ]);
        this.health = health;
        this.mailboxes = mailboxRows(mailboxPayload);
      } catch (err) {
        this.health = null;
        this.mailboxes = [];
        this.error = err.message || "LightBridge Mail Service 暂不可用。";
      } finally {
        this.loading = false;
      }
    },
  },
  template: `
    <section class="lbms">
      <h2>LightBridge Mail Service</h2>
      <p>统一管理 OAuth 账户关联邮箱、邮箱池、验证码和验证链接。</p>
      <div class="grid">
        <div class="card"><div class="metric">服务状态</div><div class="value">{{ health ? '正常' : '未连接' }}</div></div>
        <div class="card"><div class="metric">Driver 状态</div><div class="value">{{ health?.data?.driver_status || '未知' }}</div></div>
        <div class="card"><div class="metric">邮箱总数</div><div class="value">{{ mailboxes.length }}</div></div>
        <div class="card"><div class="metric">已绑定 OAuth</div><div class="value">{{ bindingTotal }}</div></div>
      </div>
      <p v-if="error" class="danger">{{ error }}</p>
      <button type="button" @click="refresh" :disabled="loading">{{ loading ? '刷新中...' : '刷新状态' }}</button>
      <button type="button" class="secondary">新增邮箱</button>
      <button type="button" class="secondary">导入邮箱池</button>
      <button type="button" class="secondary">查看审计日志</button>
      <button type="button" class="secondary">打开设置</button>
    </section>
  `,
};

const MailServiceSettings = {
  name: "MailServiceSettings",
  data() {
    return {
      baseUrl: apiBase(),
      key: apiKey(),
      enabled: true,
      status: "",
    };
  },
  mounted() { ensureStyle(); },
  methods: {
    save() {
      localStorage.setItem("lbms.baseUrl", this.baseUrl.trim());
      localStorage.setItem("lbms.apiKey", this.key.trim());
      this.status = "设置已保存到当前浏览器。后续应接入 LightBridge 模块 secret 存储。";
    },
    async test() {
      this.save();
      try {
        const health = await lbmsFetch("/mail/v1/health");
        this.status = `LightBridge Mail Service 已连接，Driver 状态：${health?.data?.driver_status || '未知'}，Store：${health?.data?.store?.type || '未知'}`;
      } catch (err) {
        this.status = err.message || "LightBridge Mail Service 暂不可用。";
      }
    },
  },
  template: `
    <section class="lbms">
      <h2>LightBridge Mail Service 设置</h2>
      <p>配置 sidecar 地址和访问密钥。底层邮件驱动只在服务端配置，不应暴露给普通用户。</p>
      <h3>基础信息</h3>
      <label>服务名称</label><input value="LightBridge Mail Service" disabled />
      <label>服务地址</label><input v-model="baseUrl" placeholder="http://127.0.0.1:8091" />
      <label>公开 API 前缀</label><input value="/mail/v1" disabled />
      <label><input type="checkbox" v-model="enabled" style="width:auto" /> 启用 LBMS 集成</label>
      <h3>安全策略</h3>
      <label>LBMS API Key</label><input v-model="key" type="password" autocomplete="off" />
      <p class="hint">Phase 1 使用 LBMS API Key。后续 Phase 4 会接入 LightBridge API Key 校验。</p>
      <button type="button" @click="test">测试连接</button>
      <button type="button" class="secondary" @click="save">保存设置</button>
      <p v-if="status">{{ status }}</p>
    </section>
  `,
};

const MailServiceMailboxes = {
  name: "MailServiceMailboxes",
  data() {
    return {
      keyword: "",
      status: "all",
      rows: [],
      loading: false,
      error: "",
    };
  },
  mounted() {
    ensureStyle();
    this.load();
  },
  computed: {
    filteredRows() {
      const keyword = this.keyword.trim().toLowerCase();
      return this.rows.filter((row) => {
        const statusMatched = this.status === "all" || row.status === this.status;
        const keywordMatched = !keyword || String(row.email_address || "").toLowerCase().includes(keyword) || String(row.id || "").toLowerCase().includes(keyword);
        return statusMatched && keywordMatched;
      });
    },
  },
  methods: {
    formatTime,
    async load() {
      this.loading = true;
      this.error = "";
      try {
        const payload = await lbmsFetch("/mail/v1/mailboxes");
        this.rows = mailboxRows(payload);
      } catch (err) {
        this.rows = [];
        this.error = err.message || "无法读取邮箱池。";
      } finally {
        this.loading = false;
      }
    },
    reset() {
      this.keyword = "";
      this.status = "all";
      this.load();
    },
  },
  template: `
    <section class="lbms">
      <h2>邮箱池</h2>
      <p>查看邮箱、绑定 OAuth 数量和操作入口。邮箱详情保存在 LightBridge Mail Service；LightBridge 主账号 Extra 只保留 lbms_link。</p>
      <div class="grid">
        <div><label>关键词</label><input v-model="keyword" placeholder="邮箱地址 / mailbox id" /></div>
        <div><label>状态</label><select v-model="status"><option value="all">全部</option><option value="active">active</option><option value="available">available</option><option value="disabled">disabled</option><option value="error">error</option></select></div>
        <div><label>平台</label><select disabled><option>后续按绑定平台过滤</option></select></div>
      </div>
      <button type="button" @click="load" :disabled="loading">{{ loading ? '读取中...' : '刷新' }}</button><button type="button" class="secondary" @click="reset">重置</button><button type="button" class="secondary">批量导入</button>
      <p v-if="error" class="danger">{{ error }}</p>
      <table>
        <thead><tr><th>邮箱地址</th><th>状态</th><th>绑定 OAuth 数</th><th>创建时间</th><th>更新时间</th><th>Mailbox ID</th><th>操作</th></tr></thead>
        <tbody>
          <tr v-if="filteredRows.length === 0"><td colspan="7" class="hint">暂无邮箱，或当前筛选条件没有结果。</td></tr>
          <tr v-for="row in filteredRows" :key="row.id">
            <td>{{ row.email_address }}</td>
            <td><span class="pill">{{ row.status }}</span></td>
            <td>{{ row.binding_count }}</td>
            <td>{{ formatTime(row.created_at) }}</td>
            <td>{{ formatTime(row.updated_at) }}</td>
            <td class="hint">{{ row.id }}</td>
            <td><button type="button" class="secondary">查看绑定</button><button type="button" class="secondary">获取验证码</button></td>
          </tr>
        </tbody>
      </table>
    </section>
  `,
};

const OAuthMailServicePanel = {
  name: "OAuthMailServicePanel",
  props: {
    modelValue: { type: Object, default: () => ({}) },
  },
  emits: ["update:modelValue"],
  data() {
    return {
      enabled: Boolean(this.modelValue?.extra?.lbms_link),
      emailAddress: "",
      bindMode: "link_or_create",
      syncPolicy: "create_binding_after_account_save",
      status: "",
    };
  },
  mounted() { ensureStyle(); },
  methods: {
    updateExtra(key, value) {
      const current = this.modelValue || {};
      this.$emit("update:modelValue", {
        ...current,
        extra: {
          ...(current.extra || {}),
          [key]: value,
        },
      });
    },
    setLink(link) {
      this.updateExtra("lbms_link", link);
    },
    markPending() {
      this.status = "保存 OAuth 账户后，将由 LightBridge Mail Service 建立双向绑定，并只把 lbms_link 写入账号 Extra。";
    },
  },
  template: `
    <section class="lbms">
      <h2>LightBridge Mail Service</h2>
      <p>为当前 OAuth 账户关联邮箱。一个邮箱可以绑定多个 OAuth 账户；当前 OAuth 账户只保存一个 lbms_link。</p>
      <label><input type="checkbox" v-model="enabled" style="width:auto" @change="markPending" /> 关联邮箱服务</label>
      <template v-if="enabled">
        <label>邮箱地址</label><input v-model="emailAddress" placeholder="aa@qq.com" @input="markPending" />
        <label>绑定方式</label>
        <select v-model="bindMode" @change="markPending">
          <option value="link_or_create">查找或创建邮箱</option>
          <option value="find_only">只查找已有邮箱</option>
          <option value="claim_from_pool">从邮箱池领取一个邮箱</option>
          <option value="none">暂不绑定，仅保存 OAuth 账户</option>
        </select>
        <label>同步策略</label>
        <select v-model="syncPolicy" @change="markPending">
          <option value="create_binding_after_account_save">创建 OAuth 账户后建立双向绑定</option>
          <option value="manual_retry">绑定失败后手动重试</option>
        </select>
        <button type="button" class="secondary">测试读取</button>
        <button type="button" class="secondary">选择已有邮箱</button>
        <p class="hint">当前表单不会写入邮箱详情；最终只应写入 extra.lbms_link。</p>
      </template>
      <p v-if="status">{{ status }}</p>
    </section>
  `,
};

export { MailServiceHome, MailServiceSettings, MailServiceMailboxes, OAuthMailServicePanel };
