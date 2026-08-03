  // Onboarding Tour
export default {
    restartTour: '重新查看新手引导',
    dontShowAgain: '不再提示',
    dontShowAgainTitle: '永久关闭新手引导',
    confirmDontShow: '确定不再显示新手引导吗？\n\n您可以随时在右上角头像菜单中重新开启。',
    confirmExit: '确定要退出新手引导吗？您可以随时在右上角菜单重新开始。',
    interactiveHint: '按 Enter 或点击继续',
    navigation: {
      flipPage: '翻页',
      exit: '退出'
    },
    // Admin tour steps
    admin: {
      welcome: {
        title: '👋 欢迎使用 LightBridge',
        description:
          '<div style="line-height: 1.8;"><p style="margin-bottom: 16px;">LightBridge 是一个强大的 AI 服务中转平台，让您轻松管理和分发 AI 服务。</p><p style="margin-bottom: 12px;"><b>🎯 核心功能：</b></p><ul style="margin-left: 20px; margin-bottom: 16px;"><li>📦 <b>分组管理</b> - 创建不同的服务套餐（VIP、免费试用等）</li><li>🔗 <b>账号池</b> - 连接多个上游 AI 服务商账号</li><li>🔑 <b>密钥分发</b> - 为用户生成独立的 API Key</li><li>💰 <b>计费管理</b> - 灵活的费率和配额控制</li></ul><p style="color: #10b981; font-weight: 600;">接下来，我们将用 3 分钟带您完成首次配置 →</p></div>',
        nextBtn: '开始配置 🚀',
        prevBtn: '跳过'
      },
      groupManage: {
        title: '📦 第一步：分组管理',
        description:
          '<div style="line-height: 1.7;"><p style="margin-bottom: 12px;"><b>什么是分组？</b></p><p style="margin-bottom: 12px;">分组是 LightBridge 的核心概念，它就像一个"服务套餐"：</p><ul style="margin-left: 20px; margin-bottom: 12px; font-size: 13px;"><li>🎯 每个分组可以包含多个上游账号</li><li>💰 每个分组有独立的计费倍率</li><li>👥 可以设置为公开或专属分组</li></ul><p style="margin-top: 12px; padding: 8px 12px; background: #f0fdf4; border-left: 3px solid #10b981; border-radius: 4px; font-size: 13px;"><b>💡 示例：</b>您可以创建"VIP专线"（高倍率）和"免费试用"（低倍率）两个分组</p><p style="margin-top: 16px; color: #10b981; font-weight: 600;">👉 点击左侧的"分组管理"开始</p></div>'
      },
      createGroup: {
        title: '➕ 创建新分组',
        description:
          '<div style="line-height: 1.7;"><p style="margin-bottom: 12px;">现在让我们创建第一个分组。</p><p style="padding: 8px 12px; background: #eff6ff; border-left: 3px solid #3b82f6; border-radius: 4px; font-size: 13px; margin-bottom: 12px;"><b>📝 提示：</b>建议先创建一个测试分组，熟悉流程后再创建正式分组</p><p style="color: #10b981; font-weight: 600;">👉 点击"创建分组"按钮</p></div>'
      },
      groupName: {
        title: '✏️ 1. 分组名称',
        description:
          '<div style="line-height: 1.7;"><p style="margin-bottom: 12px;">为您的分组起一个易于识别的名称。</p><div style="padding: 8px 12px; background: #f0fdf4; border-left: 3px solid #10b981; border-radius: 4px; font-size: 13px; margin-bottom: 12px;"><b>💡 命名建议：</b><ul style="margin: 8px 0 0 16px;"><li>"测试分组" - 用于测试</li><li>"VIP专线" - 高质量服务</li><li>"免费试用" - 体验版</li></ul></div><p style="font-size: 13px; color: #6b7280;">填写完成后点击"下一步"继续</p></div>',
        nextBtn: '下一步'
      },
      groupMultiplier: {
        title: '💰 2. 费率倍数',
        description:
          '<div style="line-height: 1.7;"><p style="margin-bottom: 12px;">设置该分组的计费倍率，控制用户的实际扣费。</p><div style="padding: 8px 12px; background: #fef3c7; border-left: 3px solid #f59e0b; border-radius: 4px; font-size: 13px; margin-bottom: 12px;"><b>⚙️ 计费规则：</b><ul style="margin: 8px 0 0 16px;"><li><b>1.0</b> - 原价计费（成本价）</li><li><b>1.5</b> - 用户消耗 $1，扣除 $1.5</li><li><b>2.0</b> - 用户消耗 $1，扣除 $2</li><li><b>0.8</b> - 补贴模式（亏本运营）</li></ul></div><p style="font-size: 13px; color: #6b7280;">建议测试分组设置为 1.0</p></div>',
        nextBtn: '下一步'
      },
      groupExclusive: {
        title: '🔒 3. 专属分组（可选）',
        description:
          '<div style="line-height: 1.7;"><p style="margin-bottom: 12px;">控制分组的可见性和访问权限。</p><div style="padding: 8px 12px; background: #eff6ff; border-left: 3px solid #3b82f6; border-radius: 4px; font-size: 13px; margin-bottom: 12px;"><b>🔐 权限说明：</b><ul style="margin: 8px 0 0 16px;"><li><b>关闭</b> - 公开分组，所有用户可见</li><li><b>开启</b> - 专属分组，仅指定用户可见</li></ul></div><p style="padding: 8px 12px; background: #f0fdf4; border-left: 3px solid #10b981; border-radius: 4px; font-size: 13px;"><b>💡 使用场景：</b>VIP 用户专属、内部测试、特殊客户等</p></div>',
        nextBtn: '下一步'
      },
      groupSubmit: {
        title: '✅ 保存分组',
        description:
          '<div style="line-height: 1.7;"><p style="margin-bottom: 12px;">确认信息无误后，点击创建按钮保存分组。</p><p style="padding: 8px 12px; background: #eff6ff; border-left: 3px solid #3b82f6; border-radius: 4px; font-size: 13px; margin-bottom: 12px;"><b>📌 分组标识：</b>图标和强调色仅用于视觉区分，不会影响请求路由。</p><p style="padding: 8px 12px; background: #f0fdf4; border-left: 3px solid #10b981; border-radius: 4px; font-size: 13px;"><b>📌 下一步：</b>创建成功后，我们将添加上游账号。分组的实际上游与协议会根据绑定账号自动识别。</p><p style="margin-top: 12px; color: #10b981; font-weight: 600;">👉 点击"创建"按钮</p></div>'
      },
      accountManage: {
        title: '🔗 第二步：添加账号',
        description:
          '<div style="line-height: 1.7;"><p style="margin-bottom: 12px;"><b>太棒了！分组已创建成功 🎉</b></p><p style="margin-bottom: 12px;">现在需要添加上游 AI 服务商的账号，让分组能够实际提供服务。</p><div style="padding: 8px 12px; background: #eff6ff; border-left: 3px solid #3b82f6; border-radius: 4px; font-size: 13px; margin-bottom: 12px;"><b>🔑 账号的作用：</b><ul style="margin: 8px 0 0 16px;"><li>连接到上游 AI 服务（Claude、GPT、Gemini、Grok 等）</li><li>一个分组可以包含多个账号（负载均衡）</li><li>支持 OAuth 和 Session Key 等授权方式</li></ul></div><p style="margin-top: 16px; color: #10b981; font-weight: 600;">👉 点击左侧的"账号管理"</p></div>'
      },
      createAccount: {
        title: '➕ 添加新账号',
        description:
          '<div style="line-height: 1.7;"><p style="margin-bottom: 12px;">点击按钮开始添加您的第一个上游账号。</p><p style="padding: 8px 12px; background: #f0fdf4; border-left: 3px solid #10b981; border-radius: 4px; font-size: 13px;"><b>💡 提示：</b>建议使用 OAuth 方式，更安全且无需手动提取密钥</p><p style="margin-top: 12px; color: #10b981; font-weight: 600;">👉 点击"添加账号"按钮</p></div>'
      },
      accountName: {
        title: '✏️ 1. 账号名称',
        description:
          '<div style="line-height: 1.7;"><p style="margin-bottom: 12px;">为账号设置一个便于识别的名称。</p><p style="padding: 8px 12px; background: #f0fdf4; border-left: 3px solid #10b981; border-radius: 4px; font-size: 13px;"><b>💡 命名建议：</b>"Claude主账号"、"GPT备用1"、"测试账号" 等</p></div>',
        nextBtn: '下一步'
      },
      accountPlatform: {
        title: '🤖 2. 选择平台',
        description:
          '<div style="line-height: 1.7;"><p style="margin-bottom: 12px;">选择该账号实际连接的上游服务商。</p><p style="padding: 8px 12px; background: #eff6ff; border-left: 3px solid #3b82f6; border-radius: 4px; font-size: 13px;"><b>📌 路由说明：</b>分组不再拥有平台类型；账号服务商、Custom 协议与中继模式共同决定该账号可接收哪些入口协议。</p></div>',
        nextBtn: '下一步'
      },
      accountType: {
        title: '🔐 3. 授权方式',
        description:
          '<div style="line-height: 1.7;"><p style="margin-bottom: 12px;">选择账号的授权方式。</p><div style="padding: 8px 12px; background: #f0fdf4; border-left: 3px solid #10b981; border-radius: 4px; font-size: 13px; margin-bottom: 12px;"><b>✅ 推荐：OAuth 方式</b><ul style="margin: 8px 0 0 16px;"><li>无需手动提取密钥</li><li>更安全，支持自动刷新</li><li>适用于 Claude Code、ChatGPT OAuth、Grok OAuth</li></ul></div><div style="padding: 8px 12px; background: #eff6ff; border-left: 3px solid #3b82f6; border-radius: 4px; font-size: 13px;"><b>📌 Session Key 方式</b><ul style="margin: 8px 0 0 16px;"><li>需要手动从浏览器提取</li><li>可能需要定期更新</li><li>适用于不支持 OAuth 的平台</li></ul></div></div>',
        nextBtn: '下一步'
      },
      accountPriority: {
        title: '⚖️ 4. 优先级（可选）',
        description:
          '<div style="line-height: 1.7;"><p style="margin-bottom: 12px;">设置账号的调用优先级。</p><div style="padding: 8px 12px; background: #eff6ff; border-left: 3px solid #3b82f6; border-radius: 4px; font-size: 13px; margin-bottom: 12px;"><b>📊 优先级规则：</b><ul style="margin: 8px 0 0 16px;"><li>数字越小，优先级越高</li><li>系统优先使用低数值账号</li><li>相同优先级则随机选择</li></ul></div><p style="padding: 8px 12px; background: #f0fdf4; border-left: 3px solid #10b981; border-radius: 4px; font-size: 13px;"><b>💡 使用场景：</b>主账号设置低数值，备用账号设置高数值</p></div>',
        nextBtn: '下一步'
      },
      accountGroups: {
        title: '🎯 5. 分配分组',
        description:
          '<div style="line-height: 1.7;"><p style="margin-bottom: 12px;"><b>关键步骤！</b>将账号分配到刚才创建的分组。</p><div style="padding: 8px 12px; background: #fee2e2; border-left: 3px solid #ef4444; border-radius: 4px; font-size: 13px; margin-bottom: 12px;"><b>⚠️ 重要提醒：</b><ul style="margin: 8px 0 0 16px;"><li>必须勾选至少一个分组</li><li>未分配分组的账号无法使用</li><li>一个账号可以分配给多个分组</li></ul></div><p style="padding: 8px 12px; background: #f0fdf4; border-left: 3px solid #10b981; border-radius: 4px; font-size: 13px;"><b>💡 提示：</b>请勾选刚才创建的测试分组</p></div>',
        nextBtn: '下一步'
      },
      accountSubmit: {
        title: '✅ 保存账号',
        description:
          '<div style="line-height: 1.7;"><p style="margin-bottom: 12px;">确认信息无误后，点击保存按钮。</p><div style="padding: 8px 12px; background: #eff6ff; border-left: 3px solid #3b82f6; border-radius: 4px; font-size: 13px; margin-bottom: 12px;"><b>📌 OAuth 授权流程：</b><ul style="margin: 8px 0 0 16px;"><li>点击保存后会跳转到服务商页面</li><li>在服务商页面完成登录授权</li><li>授权成功后自动返回</li></ul></div><p style="padding: 8px 12px; background: #f0fdf4; border-left: 3px solid #10b981; border-radius: 4px; font-size: 13px;"><b>📌 下一步：</b>账号添加成功后，我们将创建 API 密钥</p><p style="margin-top: 12px; color: #10b981; font-weight: 600;">👉 点击"保存"按钮</p></div>'
      },
      keyManage: {
        title: '🔑 第三步：生成密钥',
        description:
          '<div style="line-height: 1.7;"><p style="margin-bottom: 12px;"><b>恭喜！账号配置完成 🎉</b></p><p style="margin-bottom: 12px;">最后一步，生成 API Key 来测试服务是否正常工作。</p><div style="padding: 8px 12px; background: #eff6ff; border-left: 3px solid #3b82f6; border-radius: 4px; font-size: 13px; margin-bottom: 12px;"><b>🔑 API Key 的作用：</b><ul style="margin: 8px 0 0 16px;"><li>用于调用 AI 服务的凭证</li><li>每个 Key 绑定一个分组</li><li>可以设置配额和有效期</li><li>支持独立的使用统计</li></ul></div><p style="margin-top: 16px; color: #10b981; font-weight: 600;">👉 点击左侧的"API 密钥"</p></div>'
      },
      createKey: {
        title: '➕ 创建密钥',
        description:
          '<div style="line-height: 1.7;"><p style="margin-bottom: 12px;">点击按钮创建您的第一个 API Key。</p><p style="padding: 8px 12px; background: #f0fdf4; border-left: 3px solid #10b981; border-radius: 4px; font-size: 13px;"><b>💡 提示：</b>创建后请立即复制保存，密钥只显示一次</p><p style="margin-top: 12px; color: #10b981; font-weight: 600;">👉 点击"创建密钥"按钮</p></div>'
      },
      keyName: {
        title: '✏️ 1. 密钥名称',
        description:
          '<div style="line-height: 1.7;"><p style="margin-bottom: 12px;">为密钥设置一个便于管理的名称。</p><p style="padding: 8px 12px; background: #f0fdf4; border-left: 3px solid #10b981; border-radius: 4px; font-size: 13px;"><b>💡 命名建议：</b>"测试密钥"、"生产环境"、"移动端" 等</p></div>',
        nextBtn: '下一步'
      },
      keyGroup: {
        title: '🎯 2. 选择分组',
        description:
          '<div style="line-height: 1.7;"><p style="margin-bottom: 12px;">选择刚才配置好的分组。</p><div style="padding: 8px 12px; background: #eff6ff; border-left: 3px solid #3b82f6; border-radius: 4px; font-size: 13px; margin-bottom: 12px;"><b>📌 分组决定：</b><ul style="margin: 8px 0 0 16px;"><li>该密钥可以使用哪些账号</li><li>计费倍率是多少</li><li>是否为专属密钥</li></ul></div><p style="padding: 8px 12px; background: #f0fdf4; border-left: 3px solid #10b981; border-radius: 4px; font-size: 13px;"><b>💡 提示：</b>选择刚才创建的测试分组</p></div>',
        nextBtn: '下一步'
      },
      keySubmit: {
        title: '🎉 生成并复制',
        description:
          '<div style="line-height: 1.7;"><p style="margin-bottom: 12px;">点击创建后，系统会生成完整的 API Key。</p><div style="padding: 8px 12px; background: #fee2e2; border-left: 3px solid #ef4444; border-radius: 4px; font-size: 13px; margin-bottom: 12px;"><b>⚠️ 重要提醒：</b><ul style="margin: 8px 0 0 16px;"><li>密钥只显示一次，请立即复制</li><li>丢失后需要重新生成</li><li>妥善保管，不要泄露给他人</li></ul></div><div style="padding: 8px 12px; background: #f0fdf4; border-left: 3px solid #10b981; border-radius: 4px; font-size: 13px; margin-bottom: 12px;"><b>🚀 下一步：</b><ul style="margin: 8px 0 0 16px;"><li>复制生成的 sk-xxx 密钥</li><li>在支持 OpenAI 接口的客户端中使用</li><li>开始体验 AI 服务！</li></ul></div><p style="margin-top: 12px; color: #10b981; font-weight: 600;">👉 点击"创建"按钮</p></div>'
      }
    },
    // User tour steps
    user: {
      welcome: {
        title: '👋 欢迎使用 LightBridge',
        description:
          '<div style="line-height: 1.8;"><p style="margin-bottom: 16px;">您好！欢迎来到 LightBridge AI 服务平台。</p><p style="margin-bottom: 12px;"><b>🎯 快速开始：</b></p><ul style="margin-left: 20px; margin-bottom: 16px;"><li>🔑 创建 API 密钥</li><li>📋 复制密钥到您的应用</li><li>🚀 开始使用 AI 服务</li></ul><p style="color: #10b981; font-weight: 600;">只需 1 分钟，让我们开始吧 →</p></div>',
        nextBtn: '开始 🚀',
        prevBtn: '跳过'
      },
      keyManage: {
        title: '🔑 API 密钥管理',
        description:
          '<div style="line-height: 1.7;"><p style="margin-bottom: 12px;">在这里管理您的所有 API 访问密钥。</p><p style="padding: 8px 12px; background: #eff6ff; border-left: 3px solid #3b82f6; border-radius: 4px; font-size: 13px;"><b>📌 什么是 API 密钥？</b><br/>API 密钥是您访问 AI 服务的凭证，就像一把钥匙，让您的应用能够调用 AI 能力。</p><p style="margin-top: 12px; color: #10b981; font-weight: 600;">👉 点击进入密钥页面</p></div>'
      },
      createKey: {
        title: '➕ 创建新密钥',
        description:
          '<div style="line-height: 1.7;"><p style="margin-bottom: 12px;">点击按钮创建您的第一个 API 密钥。</p><p style="padding: 8px 12px; background: #f0fdf4; border-left: 3px solid #10b981; border-radius: 4px; font-size: 13px;"><b>💡 提示：</b>创建后密钥只显示一次，请务必复制保存</p><p style="margin-top: 12px; color: #10b981; font-weight: 600;">👉 点击"创建密钥"</p></div>'
      },
      keyName: {
        title: '✏️ 密钥名称',
        description:
          '<div style="line-height: 1.7;"><p style="margin-bottom: 12px;">为密钥起一个便于识别的名称。</p><p style="padding: 8px 12px; background: #f0fdf4; border-left: 3px solid #10b981; border-radius: 4px; font-size: 13px;"><b>💡 示例：</b>"我的第一个密钥"、"测试用" 等</p></div>',
        nextBtn: '下一步'
      },
      keyGroup: {
        title: '🎯 选择分组',
        description:
          '<div style="line-height: 1.7;"><p style="margin-bottom: 12px;">选择管理员为您分配的服务分组。</p><p style="padding: 8px 12px; background: #eff6ff; border-left: 3px solid #3b82f6; border-radius: 4px; font-size: 13px;"><b>📌 分组说明：</b><br/>不同分组可能有不同的服务质量和计费标准，请根据需要选择。</p></div>',
        nextBtn: '下一步'
      },
      keySubmit: {
        title: '🎉 完成创建',
        description:
          '<div style="line-height: 1.7;"><p style="margin-bottom: 12px;">点击确认创建您的 API 密钥。</p><div style="padding: 8px 12px; background: #fee2e2; border-left: 3px solid #ef4444; border-radius: 4px; font-size: 13px; margin-bottom: 12px;"><b>⚠️ 重要：</b><ul style="margin: 8px 0 0 16px;"><li>创建后请立即复制密钥（sk-xxx）</li><li>密钥只显示一次，丢失需重新生成</li></ul></div><p style="padding: 8px 12px; background: #f0fdf4; border-left: 3px solid #10b981; border-radius: 4px; font-size: 13px;"><b>🚀 如何使用：</b><br/>将密钥配置到支持 OpenAI 接口的任何客户端（如 ChatBox、OpenCat 等），即可开始使用！</p><p style="margin-top: 12px; color: #10b981; font-weight: 600;">👉 点击"创建"按钮</p></div>'
      }
    },
    personal: {
      welcome: {
        title: '👋 欢迎使用 LightBridge 个人模式',
        description:
          '<div style="line-height: 1.8;"><p style="margin-bottom: 16px;">个人模式已为您精简界面，专注「接入账户 → 创建密钥」两步即可使用。</p><p style="margin-bottom: 12px;"><b>🎯 我们将带您：</b></p><ul style="margin-left: 20px; margin-bottom: 16px;"><li>🌐 添加一个 AI 账户</li><li>🔑 创建调用密钥</li><li>🚀 开始使用</li></ul><p style="color: #10b981; font-weight: 600;">跟着引导走一遍吧 →</p></div>',
        nextBtn: '开始 🚀',
        prevBtn: '跳过'
      },
      accountManage: {
        title: '🌐 账户管理',
        description: '<div style="line-height: 1.7;"><p>这里管理您接入的 AI 账户（如 Claude、OpenAI、Gemini、Grok）。</p><p style="margin-top: 12px; color: #10b981; font-weight: 600;">👉 点击进入账户页面</p></div>'
      },
      createAccount: {
        title: '➕ 添加账户',
        description: '<div style="line-height: 1.7;"><p>点击按钮添加您的第一个 AI 账户。</p><p style="margin-top: 12px; color: #10b981; font-weight: 600;">👉 点击"添加账户"</p></div>'
      },
      accountName: {
        title: '✏️ 账户名称',
        description: '<div style="line-height: 1.7;"><p>给账户起一个便于识别的名称，例如「我的 Claude」。</p></div>',
        nextBtn: '下一步'
      },
      accountPlatform: {
        title: '🧩 平台类型',
        description: '<div style="line-height: 1.7;"><p>选择该账户所属的平台（Anthropic / OpenAI / Gemini / Grok / Antigravity / 自定义）。</p></div>',
        nextBtn: '下一步'
      },
      accountType: {
        title: '🔌 添加方式',
        description: '<div style="line-height: 1.7;"><p>选择凭证接入方式（如 API Key 或 OAuth 授权）。个人模式下其余高级选项已折叠，保持默认即可。</p></div>',
        nextBtn: '下一步'
      },
      accountGroups: {
        title: '🎯 分组',
        description: '<div style="line-height: 1.7;"><p>把账户加入一个分组，密钥将通过分组调度到此账户。</p></div>',
        nextBtn: '下一步'
      },
      accountSubmit: {
        title: '✅ 保存账户',
        description: '<div style="line-height: 1.7;"><p>确认后即可保存账户。</p><p style="margin-top: 12px; color: #10b981; font-weight: 600;">👉 点击"保存"</p></div>'
      },
      keyManage: {
        title: '🔑 密钥管理',
        description: '<div style="line-height: 1.7;"><p>账户接入后，在这里创建调用密钥。</p><p style="margin-top: 12px; color: #10b981; font-weight: 600;">👉 点击进入密钥页面</p></div>'
      },
      createKey: {
        title: '➕ 创建密钥',
        description: '<div style="line-height: 1.7;"><p>点击创建一个 API 密钥。</p><p style="margin-top: 12px; color: #10b981; font-weight: 600;">👉 点击"创建密钥"</p></div>'
      },
      keySubmit: {
        title: '🎉 完成',
        description: '<div style="line-height: 1.7;"><p>确认创建密钥后，立即复制保存（只显示一次）。把它配置到任意兼容 OpenAI 接口的客户端即可开始使用。</p><p style="margin-top: 12px; color: #10b981; font-weight: 600;">👉 点击"创建"</p></div>'
      }
    }
  }
