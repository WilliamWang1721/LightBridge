<template>
  <div class="space-y-6">
    <div class="card p-6">
      <div class="flex flex-wrap items-center justify-between gap-4">
        <div><h3 class="font-semibold">本地数据库备份</h3><p class="mt-1 text-sm text-gray-500">无需 S3，直接一键下载全量 PostgreSQL 数据库备份（.sql.gz）。</p></div>
        <button data-test="local-backup" class="btn btn-primary" :disabled="downloadingLocal" @click="downloadLocal">{{ downloadingLocal ? '正在生成…' : '一键本地备份' }}</button>
      </div>
    </div>
    <div class="card p-6">
      <h3 class="font-semibold">{{ t('admin.backup.s3.title') }}</h3>
      <div class="mt-4 grid gap-3 md:grid-cols-2">
        <input v-model="s3.endpoint" class="input" :placeholder="t('admin.backup.s3.endpoint')"/><input v-model="s3.region" class="input" :placeholder="t('admin.backup.s3.region')"/><input v-model="s3.bucket" class="input" :placeholder="t('admin.backup.s3.bucket')"/><input v-model="s3.prefix" class="input" :placeholder="t('admin.backup.s3.prefix')"/><input v-model="s3.access_key_id" class="input" placeholder="Access Key ID"/><input v-model="s3.secret_access_key" type="password" class="input" placeholder="Secret Access Key"/>
      </div>
      <div class="mt-4 flex gap-2"><button class="btn btn-secondary" @click="testS3">{{ t('admin.backup.s3.testConnection') }}</button><button class="btn btn-primary" @click="saveS3">{{ t('common.save') }}</button></div>
    </div>
    <div class="card p-6">
      <div class="flex flex-wrap items-center justify-between gap-3"><div><h3 class="font-semibold">{{ t('admin.backup.operations.title') }}</h3><p class="text-sm text-gray-500">{{ t('admin.backup.operations.description') }}</p></div><div class="flex gap-2"><button class="btn btn-primary" :disabled="creating" @click="createRemote">{{ creating ? t('admin.backup.operations.backing') : t('admin.backup.operations.createBackup') }}</button><button class="btn btn-secondary" @click="load">{{ t('common.refresh') }}</button></div></div>
      <div class="mt-4 overflow-x-auto"><table class="w-full text-sm"><thead><tr><th class="text-left">ID</th><th class="text-left">{{ t('admin.backup.columns.status') }}</th><th class="text-left">{{ t('admin.backup.columns.fileName') }}</th><th></th></tr></thead><tbody><tr v-for="r in records" :key="r.id"><td class="py-2 font-mono text-xs">{{ r.id }}</td><td>{{ r.status }}</td><td>{{ r.file_name }}</td><td class="text-right"><button v-if="r.status==='completed'" class="btn btn-secondary btn-xs" @click="downloadRemote(r.id)">{{ t('admin.backup.actions.download') }}</button></td></tr></tbody></table></div>
    </div>
  </div>
</template>
<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { adminAPI } from '@/api'
import { useAppStore } from '@/stores'
import type { BackupRecord, BackupS3Config } from '@/api/admin/backup'
const { t } = useI18n(); const app = useAppStore(); const downloadingLocal = ref(false); const creating = ref(false); const records = ref<BackupRecord[]>([])
const s3 = ref<BackupS3Config>({ endpoint: '', region: 'auto', bucket: '', access_key_id: '', secret_access_key: '', prefix: 'backups/', force_path_style: false })
async function downloadLocal() { downloadingLocal.value = true; try { const name = await adminAPI.backup.downloadLocalBackup(); app.showSuccess(`本地备份已下载：${name}`) } catch (e: any) { app.showError(e?.message || '本地备份失败') } finally { downloadingLocal.value = false } }
async function load() { try { records.value = (await adminAPI.backup.listBackups()).items || [] } catch (e: any) { app.showError(e?.message || t('errors.networkError')) } }
async function createRemote() { creating.value = true; try { await adminAPI.backup.createBackup({ expire_days: 14 }); app.showSuccess(t('admin.backup.operations.backupCreated')); await load() } catch (e: any) { app.showError(e?.message || t('admin.backup.operations.backupFailed')) } finally { creating.value = false } }
async function downloadRemote(id: string) { try { window.open((await adminAPI.backup.getDownloadURL(id)).url, '_blank') } catch (e: any) { app.showError(e?.message || t('errors.networkError')) } }
async function saveS3() { try { await adminAPI.backup.updateS3Config(s3.value); app.showSuccess(t('admin.backup.s3.saved')) } catch (e: any) { app.showError(e?.message || t('errors.networkError')) } }
async function testS3() { try { const r = await adminAPI.backup.testS3Connection(s3.value); r.ok ? app.showSuccess(r.message) : app.showError(r.message) } catch (e: any) { app.showError(e?.message || t('errors.networkError')) } }
onMounted(async () => { try { const cfg = await adminAPI.backup.getS3Config(); s3.value = { ...cfg, secret_access_key: '' } } catch {}; await load() })
</script>
