export default {
      title: '数据库备份',
      description: '全量数据库备份到 S3 兼容存储，支持定时备份与恢复',
      s3: {
        title: 'S3 存储配置',
        description: '配置 S3 兼容存储（支持 Cloudflare R2）',
        descriptionPrefix: '配置 S3 兼容存储（支持',
        descriptionSuffix: '）',
        enabled: '启用 S3 存储',
        endpoint: '端点地址',
        region: '区域',
        bucket: '存储桶',
        prefix: 'Key 前缀',
        accessKeyId: 'Access Key ID',
        secretAccessKey: 'Secret Access Key',
        secretConfigured: '已配置，留空保持不变',
        forcePathStyle: '强制路径风格',
        testConnection: '测试连接',
        testSuccess: 'S3 连接测试成功',
        testFailed: 'S3 连接测试失败',
        saved: 'S3 配置已保存'
      },
      schedule: {
        title: '定时备份',
        description: '配置自动定时备份',
        enabled: '启用定时备份',
        cronExpr: 'Cron 表达式',
        cronHint: '例如 "0 2 * * *" 表示每天凌晨 2 点',
        retainDays: '备份过期天数',
        retainDaysHint: '备份文件超过此天数后自动删除，0 = 永不过期',
        retainCount: '最大保留份数',
        retainCountHint: '最多保留的备份数量，0 = 不限制',
        saved: '定时备份配置已保存'
      },
      operations: {
        title: '备份记录',
        description: '创建手动备份和管理已有备份记录',
        createBackup: '创建备份',
        backing: '备份中...',
        backupCreated: '备份创建成功',
        expireDays: '过期天数',
        alreadyInProgress: '已有备份正在进行中',
        backupRunning: '备份进行中...',
        backupFailed: '备份失败',
        restoreRunning: '恢复进行中...',
        restoreFailed: '恢复失败',
      },
      columns: {
        status: '状态',
        fileName: '文件名',
        origin: '来源',
        size: '大小',
        expiresAt: '过期时间',
        triggeredBy: '触发方式',
        startedAt: '开始时间',
        actions: '操作'
      },
      status: {
        pending: '等待中',
        running: '执行中',
        completed: '已完成',
        failed: '失败'
      },
      progress: {
        pending: '准备中',
        dumping: '导出数据库',
        uploading: '上传中',
      },
      trigger: {
        manual: '手动',
        scheduled: '定时',
        versionManager: '版本管理器'
      },
      versionSnapshot: {
        label: '版本管理器快照',
        update: '更新前：{sourceVersion} → {targetVersion}',
        updateWithoutTarget: '从 {sourceVersion} 更新前',
        rollback: '从 {sourceVersion} 回滚前',
        operationId: '操作：{id}',
        initiatedBy: '发起者：{initiatedBy}'
      },
      neverExpire: '永不过期',
      empty: '暂无备份记录',
      actions: {
        download: '下载',
        restore: '恢复',
        restoreConfirm: '确定要从此备份恢复吗？这将覆盖当前数据库！',
        restorePasswordPrompt: '请输入管理员密码以确认恢复操作',
        restoreSuccess: '数据库恢复成功',
        deleteConfirm: '确定要删除此备份吗？',
        deleted: '备份已删除'
      },
      r2Guide: {
        title: 'Cloudflare R2 配置教程',
        intro: 'Cloudflare R2 提供 S3 兼容的对象存储，免费额度为 10GB 存储 + 每月 100 万次 A 类请求，非常适合数据库备份。',
        step1: {
          title: '创建 R2 存储桶',
          line1: '登录 Cloudflare Dashboard (dash.cloudflare.com)，左侧菜单选择「R2 对象存储」',
          line2: '点击「创建存储桶」，输入名称（如 LightBridge-backups），选择区域',
          line3: '点击创建完成'
        },
        step2: {
          title: '创建 API 令牌',
          line1: '在 R2 页面，点击右上角「管理 R2 API 令牌」',
          line2: '点击「创建 API 令牌」，权限选择「对象读和写」',
          line3: '建议指定存储桶范围（仅允许访问备份桶，更安全）',
          line4: '创建后会显示 Access Key ID 和 Secret Access Key',
          warning: 'Secret Access Key 只会显示一次，请立即复制保存！'
        },
        step3: {
          title: '获取 S3 端点地址',
          desc: '在 R2 概览页面找到你的账户 ID（在 URL 或右侧面板中），端点格式为：',
          accountId: '你的账户 ID'
        },
        step4: {
          title: '填写以下配置',
          checkEnabled: '勾选',
          bucketValue: '你创建的存储桶名称',
          fromStep2: '第 2 步获取的值',
          unchecked: '不勾选'
        },
        freeTier: 'R2 免费额度：10GB 存储 + 每月 100 万次 A 类请求 + 1000 万次 B 类请求，对数据库备份完全够用。'
      }
    }
