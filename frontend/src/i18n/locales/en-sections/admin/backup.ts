export default {
      title: 'Database Backup',
      description: 'Full database backup to S3-compatible storage with scheduled backup and restore',
      s3: {
        title: 'S3 Storage Configuration',
        description: 'Configure S3-compatible storage (supports Cloudflare R2)',
        descriptionPrefix: 'Configure S3-compatible storage (supports',
        descriptionSuffix: ')',
        enabled: 'Enable S3 Storage',
        endpoint: 'Endpoint',
        region: 'Region',
        bucket: 'Bucket',
        prefix: 'Key Prefix',
        accessKeyId: 'Access Key ID',
        secretAccessKey: 'Secret Access Key',
        secretConfigured: 'Already configured, leave empty to keep',
        forcePathStyle: 'Force Path Style',
        testConnection: 'Test Connection',
        testSuccess: 'S3 connection test successful',
        testFailed: 'S3 connection test failed',
        saved: 'S3 configuration saved'
      },
      schedule: {
        title: 'Scheduled Backup',
        description: 'Configure automatic scheduled backups',
        enabled: 'Enable Scheduled Backup',
        cronExpr: 'Cron Expression',
        cronHint: 'e.g. "0 2 * * *" means every day at 2:00 AM',
        retainDays: 'Backup Expire Days',
        retainDaysHint: 'Backup files auto-delete after this many days, 0 = never expire',
        retainCount: 'Max Retain Count',
        retainCountHint: 'Maximum number of backups to keep, 0 = unlimited',
        saved: 'Schedule configuration saved'
      },
      operations: {
        title: 'Backup Records',
        description: 'Create manual backups and manage existing backup records',
        createBackup: 'Create Backup',
        backing: 'Backing up...',
        backupCreated: 'Backup created successfully',
        expireDays: 'Expire Days',
        alreadyInProgress: 'A backup is already in progress',
        backupRunning: 'Backup in progress...',
        backupFailed: 'Backup failed',
        restoreRunning: 'Restore in progress...',
        restoreFailed: 'Restore failed',
      },
      columns: {
        status: 'Status',
        fileName: 'File Name',
        origin: 'Origin',
        size: 'Size',
        expiresAt: 'Expires At',
        triggeredBy: 'Triggered By',
        startedAt: 'Started At',
        actions: 'Actions'
      },
      status: {
        pending: 'Pending',
        running: 'Running',
        completed: 'Completed',
        failed: 'Failed'
      },
      progress: {
        pending: 'Preparing',
        dumping: 'Dumping database',
        uploading: 'Uploading',
      },
      trigger: {
        manual: 'Manual',
        scheduled: 'Scheduled',
        versionManager: 'Version Manager'
      },
      versionSnapshot: {
        label: 'Version Manager snapshot',
        update: 'Before update: {sourceVersion} → {targetVersion}',
        updateWithoutTarget: 'Before update from {sourceVersion}',
        rollback: 'Before rollback from {sourceVersion}',
        operationId: 'Operation: {id}',
        initiatedBy: 'Initiated by: {initiatedBy}'
      },
      neverExpire: 'Never',
      empty: 'No backup records',
      actions: {
        download: 'Download',
        restore: 'Restore',
        restoreConfirm: 'Are you sure you want to restore from this backup? This will overwrite the current database!',
        restorePasswordPrompt: 'Please enter your admin password to confirm the restore operation',
        restoreSuccess: 'Database restored successfully',
        deleteConfirm: 'Are you sure you want to delete this backup?',
        deleted: 'Backup deleted'
      },
      r2Guide: {
        title: 'Cloudflare R2 Setup Guide',
        intro: 'Cloudflare R2 provides S3-compatible object storage with a free tier of 10GB storage + 1M Class A requests/month, ideal for database backups.',
        step1: {
          title: 'Create an R2 Bucket',
          line1: 'Log in to the Cloudflare Dashboard (dash.cloudflare.com), select "R2 Object Storage" from the sidebar',
          line2: 'Click "Create bucket", enter a name (e.g. LightBridge-backups), choose a region',
          line3: 'Click create to finish'
        },
        step2: {
          title: 'Create an API Token',
          line1: 'On the R2 page, click "Manage R2 API Tokens" in the top right',
          line2: 'Click "Create API token", set permission to "Object Read & Write"',
          line3: 'Recommended: restrict to specific bucket for better security',
          line4: 'After creation, you will see the Access Key ID and Secret Access Key',
          warning: 'The Secret Access Key is only shown once — copy and save it immediately!'
        },
        step3: {
          title: 'Get the S3 Endpoint',
          desc: 'Find your Account ID on the R2 overview page (in the URL or the right panel). The endpoint format is:',
          accountId: 'your_account_id'
        },
        step4: {
          title: 'Fill in the Configuration',
          checkEnabled: 'Checked',
          bucketValue: 'Your bucket name',
          fromStep2: 'Value from Step 2',
          unchecked: 'Unchecked'
        },
        freeTier: 'R2 Free Tier: 10GB storage + 1M Class A requests + 10M Class B requests per month — more than enough for database backups.'
      }
    }
