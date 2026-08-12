<template>
  <AppLayout>
    <ReactPageHost
      :load="loadAuthSettingsPage"
      :props="pageProps"
      :error-message="t('common.error')"
    >
      <template #fallback>
    <div class="mx-auto max-w-4xl space-y-6">
      <!-- Loading -->
      <div v-if="loading" class="flex items-center justify-center py-12">
        <div class="h-8 w-8 animate-spin rounded-full border-b-2 border-primary-600"></div>
      </div>

      <template v-else>
        <!-- Registration Settings -->
        <div class="card">
          <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
            <div class="flex items-center justify-between">
              <div>
                <h2 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('admin.authSettings.registration.title') }}</h2>
                <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ t('admin.authSettings.registration.description') }}</p>
              </div>
              <Toggle v-model="form.registration_enabled" @update:model-value="saveSettings" />
            </div>
          </div>
          <div v-if="form.registration_enabled" class="space-y-4 p-6">
            <div class="flex items-center justify-between">
              <div>
                <label class="text-sm font-medium text-gray-700 dark:text-gray-300">{{ t('admin.authSettings.registration.emailVerify') }}</label>
                <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.authSettings.registration.emailVerifyHint') }}</p>
              </div>
              <Toggle v-model="form.email_verify_enabled" @update:model-value="saveSettings" />
            </div>
            <div class="flex items-center justify-between">
              <div>
                <label class="text-sm font-medium text-gray-700 dark:text-gray-300">{{ t('admin.authSettings.registration.passwordReset') }}</label>
                <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.authSettings.registration.passwordResetHint') }}</p>
              </div>
              <Toggle v-model="form.password_reset_enabled" @update:model-value="saveSettings" />
            </div>
            <div class="flex items-center justify-between">
              <div>
                <label class="text-sm font-medium text-gray-700 dark:text-gray-300">{{ t('admin.authSettings.registration.invitationCode') }}</label>
                <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.authSettings.registration.invitationCodeHint') }}</p>
              </div>
              <Toggle v-model="form.invitation_code_enabled" @update:model-value="saveSettings" />
            </div>
          </div>
        </div>

        <!-- Login Providers Grid -->
        <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
          <!-- OIDC Login -->
          <div class="card">
            <div class="border-b border-gray-100 px-5 py-3 dark:border-dark-700">
              <div class="flex items-center justify-between">
                <div class="flex items-center gap-3">
                  <span class="flex h-8 w-8 items-center justify-center rounded-lg bg-indigo-50 dark:bg-indigo-900/30">
                    <Icon name="shield" size="sm" class="text-indigo-600 dark:text-indigo-400" />
                  </span>
                  <div>
                    <h3 class="text-sm font-semibold text-gray-900 dark:text-white">OIDC</h3>
                    <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.authSettings.oidc.description') }}</p>
                  </div>
                </div>
                <Toggle :model-value="form.oidc_connect_enabled" @update:model-value="toggleOidc" />
              </div>
            </div>
            <div v-if="form.oidc_connect_enabled" class="space-y-3 p-5">
              <div>
                <label class="text-xs font-medium text-gray-600 dark:text-gray-400">{{ t('admin.authSettings.oidc.providerName') }}</label>
                <input v-model="form.oidc_connect_provider_name" type="text" class="input mt-1" :placeholder="t('admin.authSettings.oidc.providerNamePlaceholder')" />
              </div>
              <div>
                <label class="text-xs font-medium text-gray-600 dark:text-gray-400">{{ t('admin.authSettings.oidc.clientId') }}</label>
                <input v-model="form.oidc_connect_client_id" type="text" class="input mt-1" :placeholder="t('admin.authSettings.oidc.clientIdPlaceholder')" />
              </div>
              <div>
                <label class="text-xs font-medium text-gray-600 dark:text-gray-400">{{ t('admin.authSettings.oidc.issuerUrl') }}</label>
                <input v-model="form.oidc_connect_issuer_url" type="url" class="input mt-1" :placeholder="t('admin.authSettings.oidc.issuerUrlPlaceholder')" />
              </div>
              <div>
                <label class="text-xs font-medium text-gray-600 dark:text-gray-400">{{ t('admin.authSettings.oidc.redirectUrl') }}</label>
                <input v-model="form.oidc_connect_redirect_url" type="url" class="input mt-1" :placeholder="t('admin.authSettings.oidc.redirectUrlPlaceholder')" />
              </div>
              <button type="button" class="btn btn-primary btn-sm w-full" :disabled="saving" @click="saveSettings">
                {{ saving ? t('common.saving') : t('common.save') }}
              </button>
            </div>
          </div>

          <!-- GitHub Login -->
          <div class="card">
            <div class="border-b border-gray-100 px-5 py-3 dark:border-dark-700">
              <div class="flex items-center justify-between">
                <div class="flex items-center gap-3">
                  <span class="flex h-8 w-8 items-center justify-center rounded-lg bg-gray-100 dark:bg-gray-800">
                    <Icon name="link" size="sm" class="text-gray-700 dark:text-gray-300" />
                  </span>
                  <div>
                    <h3 class="text-sm font-semibold text-gray-900 dark:text-white">GitHub</h3>
                    <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.authSettings.github.description') }}</p>
                  </div>
                </div>
                <Toggle :model-value="form.github_oauth_enabled" @update:model-value="toggleGithub" />
              </div>
            </div>
            <div v-if="form.github_oauth_enabled" class="space-y-3 p-5">
              <div>
                <label class="text-xs font-medium text-gray-600 dark:text-gray-400">{{ t('admin.authSettings.github.clientId') }}</label>
                <input v-model="form.github_oauth_client_id" type="text" class="input mt-1" :placeholder="t('admin.authSettings.github.clientIdPlaceholder')" />
              </div>
              <div>
                <label class="text-xs font-medium text-gray-600 dark:text-gray-400">{{ t('admin.authSettings.github.redirectUrl') }}</label>
                <input v-model="form.github_oauth_redirect_url" type="url" class="input mt-1" :placeholder="t('admin.authSettings.github.redirectUrlPlaceholder')" />
              </div>
              <button type="button" class="btn btn-primary btn-sm w-full" :disabled="saving" @click="saveSettings">
                {{ saving ? t('common.saving') : t('common.save') }}
              </button>
            </div>
          </div>

          <!-- Google Login -->
          <div class="card">
            <div class="border-b border-gray-100 px-5 py-3 dark:border-dark-700">
              <div class="flex items-center justify-between">
                <div class="flex items-center gap-3">
                  <span class="flex h-8 w-8 items-center justify-center rounded-lg bg-red-50 dark:bg-red-900/30">
                    <Icon name="globe" size="sm" class="text-red-600 dark:text-red-400" />
                  </span>
                  <div>
                    <h3 class="text-sm font-semibold text-gray-900 dark:text-white">Google</h3>
                    <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.authSettings.google.description') }}</p>
                  </div>
                </div>
                <Toggle :model-value="form.google_oauth_enabled" @update:model-value="toggleGoogle" />
              </div>
            </div>
            <div v-if="form.google_oauth_enabled" class="space-y-3 p-5">
              <div>
                <label class="text-xs font-medium text-gray-600 dark:text-gray-400">{{ t('admin.authSettings.google.clientId') }}</label>
                <input v-model="form.google_oauth_client_id" type="text" class="input mt-1" :placeholder="t('admin.authSettings.google.clientIdPlaceholder')" />
              </div>
              <div>
                <label class="text-xs font-medium text-gray-600 dark:text-gray-400">{{ t('admin.authSettings.google.redirectUrl') }}</label>
                <input v-model="form.google_oauth_redirect_url" type="url" class="input mt-1" :placeholder="t('admin.authSettings.google.redirectUrlPlaceholder')" />
              </div>
              <button type="button" class="btn btn-primary btn-sm w-full" :disabled="saving" @click="saveSettings">
                {{ saving ? t('common.saving') : t('common.save') }}
              </button>
            </div>
          </div>

          <!-- WeChat Login -->
          <div class="card">
            <div class="border-b border-gray-100 px-5 py-3 dark:border-dark-700">
              <div class="flex items-center justify-between">
                <div class="flex items-center gap-3">
                  <span class="flex h-8 w-8 items-center justify-center rounded-lg bg-green-50 dark:bg-green-900/30">
                    <Icon name="chat" size="sm" class="text-green-600 dark:text-green-400" />
                  </span>
                  <div>
                    <h3 class="text-sm font-semibold text-gray-900 dark:text-white">WeChat</h3>
                    <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.authSettings.wechat.description') }}</p>
                  </div>
                </div>
                <Toggle :model-value="form.wechat_connect_enabled" @update:model-value="toggleWechat" />
              </div>
            </div>
            <div v-if="form.wechat_connect_enabled" class="space-y-3 p-5">
              <div>
                <label class="text-xs font-medium text-gray-600 dark:text-gray-400">{{ t('admin.authSettings.wechat.appId') }}</label>
                <input v-model="form.wechat_connect_app_id" type="text" class="input mt-1" :placeholder="t('admin.authSettings.wechat.appIdPlaceholder')" />
              </div>
              <div>
                <label class="text-xs font-medium text-gray-600 dark:text-gray-400">{{ t('admin.authSettings.wechat.redirectUrl') }}</label>
                <input v-model="form.wechat_connect_redirect_url" type="url" class="input mt-1" :placeholder="t('admin.authSettings.wechat.redirectUrlPlaceholder')" />
              </div>
              <button type="button" class="btn btn-primary btn-sm w-full" :disabled="saving" @click="saveSettings">
                {{ saving ? t('common.saving') : t('common.save') }}
              </button>
            </div>
          </div>

          <!-- LinuxDO Login -->
          <div class="card">
            <div class="border-b border-gray-100 px-5 py-3 dark:border-dark-700">
              <div class="flex items-center justify-between">
                <div class="flex items-center gap-3">
                  <span class="flex h-8 w-8 items-center justify-center rounded-lg bg-orange-50 dark:bg-orange-900/30">
                    <Icon name="terminal" size="sm" class="text-orange-600 dark:text-orange-400" />
                  </span>
                  <div>
                    <h3 class="text-sm font-semibold text-gray-900 dark:text-white">LinuxDO</h3>
                    <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.authSettings.linuxdo.description') }}</p>
                  </div>
                </div>
                <Toggle :model-value="form.linuxdo_connect_enabled" @update:model-value="toggleLinuxdo" />
              </div>
            </div>
            <div v-if="form.linuxdo_connect_enabled" class="space-y-3 p-5">
              <div>
                <label class="text-xs font-medium text-gray-600 dark:text-gray-400">{{ t('admin.authSettings.linuxdo.clientId') }}</label>
                <input v-model="form.linuxdo_connect_client_id" type="text" class="input mt-1" :placeholder="t('admin.authSettings.linuxdo.clientIdPlaceholder')" />
              </div>
              <div>
                <label class="text-xs font-medium text-gray-600 dark:text-gray-400">{{ t('admin.authSettings.linuxdo.redirectUrl') }}</label>
                <input v-model="form.linuxdo_connect_redirect_url" type="url" class="input mt-1" :placeholder="t('admin.authSettings.linuxdo.redirectUrlPlaceholder')" />
              </div>
              <button type="button" class="btn btn-primary btn-sm w-full" :disabled="saving" @click="saveSettings">
                {{ saving ? t('common.saving') : t('common.save') }}
              </button>
            </div>
          </div>

          <!-- DingTalk Login -->
          <div class="card">
            <div class="border-b border-gray-100 px-5 py-3 dark:border-dark-700">
              <div class="flex items-center justify-between">
                <div class="flex items-center gap-3">
                  <span class="flex h-8 w-8 items-center justify-center rounded-lg bg-blue-50 dark:bg-blue-900/30">
                    <Icon name="server" size="sm" class="text-blue-600 dark:text-blue-400" />
                  </span>
                  <div>
                    <h3 class="text-sm font-semibold text-gray-900 dark:text-white">DingTalk</h3>
                    <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.authSettings.dingtalk.description') }}</p>
                  </div>
                </div>
                <Toggle :model-value="form.dingtalk_connect_enabled" @update:model-value="toggleDingtalk" />
              </div>
            </div>
            <div v-if="form.dingtalk_connect_enabled" class="space-y-3 p-5">
              <div>
                <label class="text-xs font-medium text-gray-600 dark:text-gray-400">{{ t('admin.authSettings.dingtalk.clientId') }}</label>
                <input v-model="form.dingtalk_connect_client_id" type="text" class="input mt-1" :placeholder="t('admin.authSettings.dingtalk.clientIdPlaceholder')" />
              </div>
              <div>
                <label class="text-xs font-medium text-gray-600 dark:text-gray-400">{{ t('admin.authSettings.dingtalk.redirectUrl') }}</label>
                <input v-model="form.dingtalk_connect_redirect_url" type="url" class="input mt-1" :placeholder="t('admin.authSettings.dingtalk.redirectUrlPlaceholder')" />
              </div>
              <button type="button" class="btn btn-primary btn-sm w-full" :disabled="saving" @click="saveSettings">
                {{ saving ? t('common.saving') : t('common.save') }}
              </button>
            </div>
          </div>
        </div>
      </template>
    </div>
      </template>
    </ReactPageHost>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, ref, reactive, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import AppLayout from '@/components/layout/AppLayout.vue'
import ReactPageHost from '@/console/ReactPageHost.vue'
import Icon from '@/components/icons/Icon.vue'
import Toggle from '@/components/common/Toggle.vue'
import { settingsAPI } from '@/api/admin/settings'
import { extractApiErrorMessage } from '@/utils/apiError'
import type { AuthSettingsForm, AuthSettingsPageProps } from '@/console/react/AuthSettingsPage'

const { t } = useI18n()
const appStore = useAppStore()

const loading = ref(true)
const saving = ref(false)

const form = reactive({
  // Registration
  registration_enabled: true,
  email_verify_enabled: false,
  password_reset_enabled: false,
  invitation_code_enabled: false,
  // OIDC
  oidc_connect_enabled: false,
  oidc_connect_provider_name: '',
  oidc_connect_client_id: '',
  oidc_connect_issuer_url: '',
  oidc_connect_redirect_url: '',
  // GitHub
  github_oauth_enabled: false,
  github_oauth_client_id: '',
  github_oauth_redirect_url: '',
  // Google
  google_oauth_enabled: false,
  google_oauth_client_id: '',
  google_oauth_redirect_url: '',
  // WeChat
  wechat_connect_enabled: false,
  wechat_connect_app_id: '',
  wechat_connect_redirect_url: '',
  // LinuxDO
  linuxdo_connect_enabled: false,
  linuxdo_connect_client_id: '',
  linuxdo_connect_redirect_url: '',
  // DingTalk
  dingtalk_connect_enabled: false,
  dingtalk_connect_client_id: '',
  dingtalk_connect_redirect_url: '',
})

const loadAuthSettingsPage = () => import('@/console/react/AuthSettingsPage')

const onFieldChange: AuthSettingsPageProps['onFieldChange'] = (key, value) => {
  Object.assign(form, { [key]: value })
}

const onToggle: AuthSettingsPageProps['onToggle'] = (key, value) => {
  Object.assign(form, { [key]: value })
  void saveSettings()
}

const pageProps = computed<AuthSettingsPageProps>(() => ({
  form: { ...form } as AuthSettingsForm,
  loading: loading.value,
  saving: saving.value,
  copy: {
    registration: {
      title: t('admin.authSettings.registration.title'),
      description: t('admin.authSettings.registration.description'),
      emailVerify: t('admin.authSettings.registration.emailVerify'),
      emailVerifyHint: t('admin.authSettings.registration.emailVerifyHint'),
      passwordReset: t('admin.authSettings.registration.passwordReset'),
      passwordResetHint: t('admin.authSettings.registration.passwordResetHint'),
      invitationCode: t('admin.authSettings.registration.invitationCode'),
      invitationCodeHint: t('admin.authSettings.registration.invitationCodeHint'),
    },
    provider: (id) => {
      const prefix = id === 'oidc_connect_enabled' ? 'oidc' : id === 'github_oauth_enabled' ? 'github' : id === 'google_oauth_enabled' ? 'google' : id === 'wechat_connect_enabled' ? 'wechat' : id === 'linuxdo_connect_enabled' ? 'linuxdo' : 'dingtalk'
      return {
        title: t(`admin.authSettings.${prefix}.title`),
        description: t(`admin.authSettings.${prefix}.description`),
        enabled: t(`admin.authSettings.${prefix}.title`),
        idLabel: t(`admin.authSettings.${prefix}.${prefix === 'wechat' ? 'appId' : 'clientId'}`),
        idPlaceholder: t(`admin.authSettings.${prefix}.${prefix === 'wechat' ? 'appIdPlaceholder' : 'clientIdPlaceholder'}`),
        providerNameLabel: prefix === 'oidc' ? t('admin.authSettings.oidc.providerName') : undefined,
        providerNamePlaceholder: prefix === 'oidc' ? t('admin.authSettings.oidc.providerNamePlaceholder') : undefined,
        appIdLabel: prefix === 'wechat' ? t('admin.authSettings.wechat.appId') : undefined,
        appIdPlaceholder: prefix === 'wechat' ? t('admin.authSettings.wechat.appIdPlaceholder') : undefined,
        issuerLabel: prefix === 'oidc' ? t('admin.authSettings.oidc.issuerUrl') : undefined,
        issuerPlaceholder: prefix === 'oidc' ? t('admin.authSettings.oidc.issuerUrlPlaceholder') : undefined,
        redirectLabel: t(`admin.authSettings.${prefix}.redirectUrl`),
        redirectPlaceholder: t(`admin.authSettings.${prefix}.redirectUrlPlaceholder`),
      }
    },
    save: t('common.save'),
    saving: t('common.saving'),
  },
  onFieldChange,
  onToggle,
  onSave: () => { void saveSettings() },
}))

async function loadSettings() {
  loading.value = true
  try {
    const settings = await settingsAPI.getSettings()
    // Registration
    form.registration_enabled = settings.registration_enabled ?? true
    form.email_verify_enabled = settings.email_verify_enabled ?? false
    form.password_reset_enabled = settings.password_reset_enabled ?? false
    form.invitation_code_enabled = settings.invitation_code_enabled ?? false
    // OIDC
    form.oidc_connect_enabled = settings.oidc_connect_enabled || false
    form.oidc_connect_provider_name = settings.oidc_connect_provider_name || ''
    form.oidc_connect_client_id = settings.oidc_connect_client_id || ''
    form.oidc_connect_issuer_url = settings.oidc_connect_issuer_url || ''
    form.oidc_connect_redirect_url = settings.oidc_connect_redirect_url || ''
    // GitHub
    form.github_oauth_enabled = settings.github_oauth_enabled || false
    form.github_oauth_client_id = settings.github_oauth_client_id || ''
    form.github_oauth_redirect_url = settings.github_oauth_redirect_url || ''
    // Google
    form.google_oauth_enabled = settings.google_oauth_enabled || false
    form.google_oauth_client_id = settings.google_oauth_client_id || ''
    form.google_oauth_redirect_url = settings.google_oauth_redirect_url || ''
    // WeChat
    form.wechat_connect_enabled = settings.wechat_connect_enabled || false
    form.wechat_connect_app_id = settings.wechat_connect_app_id || ''
    form.wechat_connect_redirect_url = settings.wechat_connect_redirect_url || ''
    // LinuxDO
    form.linuxdo_connect_enabled = settings.linuxdo_connect_enabled || false
    form.linuxdo_connect_client_id = settings.linuxdo_connect_client_id || ''
    form.linuxdo_connect_redirect_url = settings.linuxdo_connect_redirect_url || ''
    // DingTalk
    form.dingtalk_connect_enabled = settings.dingtalk_connect_enabled || false
    form.dingtalk_connect_client_id = settings.dingtalk_connect_client_id || ''
    form.dingtalk_connect_redirect_url = settings.dingtalk_connect_redirect_url || ''
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('common.error')))
  } finally {
    loading.value = false
  }
}

async function saveSettings() {
  if (saving.value) return
  saving.value = true
  try {
    await settingsAPI.updateSettings({
      registration_enabled: form.registration_enabled,
      email_verify_enabled: form.email_verify_enabled,
      password_reset_enabled: form.password_reset_enabled,
      invitation_code_enabled: form.invitation_code_enabled,
      oidc_connect_enabled: form.oidc_connect_enabled,
      oidc_connect_provider_name: form.oidc_connect_provider_name,
      oidc_connect_client_id: form.oidc_connect_client_id,
      oidc_connect_issuer_url: form.oidc_connect_issuer_url,
      oidc_connect_redirect_url: form.oidc_connect_redirect_url,
      github_oauth_enabled: form.github_oauth_enabled,
      github_oauth_client_id: form.github_oauth_client_id,
      github_oauth_redirect_url: form.github_oauth_redirect_url,
      google_oauth_enabled: form.google_oauth_enabled,
      google_oauth_client_id: form.google_oauth_client_id,
      google_oauth_redirect_url: form.google_oauth_redirect_url,
      wechat_connect_enabled: form.wechat_connect_enabled,
      wechat_connect_app_id: form.wechat_connect_app_id,
      wechat_connect_redirect_url: form.wechat_connect_redirect_url,
      linuxdo_connect_enabled: form.linuxdo_connect_enabled,
      linuxdo_connect_client_id: form.linuxdo_connect_client_id,
      linuxdo_connect_redirect_url: form.linuxdo_connect_redirect_url,
      dingtalk_connect_enabled: form.dingtalk_connect_enabled,
      dingtalk_connect_client_id: form.dingtalk_connect_client_id,
      dingtalk_connect_redirect_url: form.dingtalk_connect_redirect_url,
    } as Record<string, unknown>)
    await appStore.fetchPublicSettings(true)
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('common.error')))
  } finally {
    saving.value = false
  }
}

function toggleOidc(v: boolean) { form.oidc_connect_enabled = v; saveSettings() }
function toggleGithub(v: boolean) { form.github_oauth_enabled = v; saveSettings() }
function toggleGoogle(v: boolean) { form.google_oauth_enabled = v; saveSettings() }
function toggleWechat(v: boolean) { form.wechat_connect_enabled = v; saveSettings() }
function toggleLinuxdo(v: boolean) { form.linuxdo_connect_enabled = v; saveSettings() }
function toggleDingtalk(v: boolean) { form.dingtalk_connect_enabled = v; saveSettings() }

onMounted(loadSettings)
</script>
