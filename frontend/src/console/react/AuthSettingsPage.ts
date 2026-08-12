import { type ReactNode } from 'react'
import { createShadcnElement as h } from './ui/createElement'
import { Switch as ShadcnSwitch } from './ui'

export interface AuthSettingsForm {
  registration_enabled: boolean
  email_verify_enabled: boolean
  password_reset_enabled: boolean
  invitation_code_enabled: boolean
  oidc_connect_enabled: boolean
  oidc_connect_provider_name: string
  oidc_connect_client_id: string
  oidc_connect_issuer_url: string
  oidc_connect_redirect_url: string
  github_oauth_enabled: boolean
  github_oauth_client_id: string
  github_oauth_redirect_url: string
  google_oauth_enabled: boolean
  google_oauth_client_id: string
  google_oauth_redirect_url: string
  wechat_connect_enabled: boolean
  wechat_connect_app_id: string
  wechat_connect_redirect_url: string
  linuxdo_connect_enabled: boolean
  linuxdo_connect_client_id: string
  linuxdo_connect_redirect_url: string
  dingtalk_connect_enabled: boolean
  dingtalk_connect_client_id: string
  dingtalk_connect_redirect_url: string
}

interface ProviderCopy {
  title: string
  description: string
  enabled: string
  idLabel: string
  idPlaceholder: string
  providerNameLabel?: string
  providerNamePlaceholder?: string
  appIdLabel?: string
  appIdPlaceholder?: string
  issuerLabel?: string
  issuerPlaceholder?: string
  redirectLabel: string
  redirectPlaceholder: string
}

export interface AuthSettingsPageCopy {
  registration: {
    title: string
    description: string
    emailVerify: string
    emailVerifyHint: string
    passwordReset: string
    passwordResetHint: string
    invitationCode: string
    invitationCodeHint: string
  }
  provider: (id: keyof AuthSettingsForm) => ProviderCopy
  save: string
  saving: string
}

export interface AuthSettingsPageProps {
  form: AuthSettingsForm
  loading: boolean
  saving: boolean
  copy: AuthSettingsPageCopy
  onFieldChange: (key: keyof AuthSettingsForm, value: string | boolean) => void
  onToggle: (key: keyof AuthSettingsForm, value: boolean) => void
  onSave: () => void
}

function Switch({ checked, disabled, label, onChange }: { checked: boolean; disabled?: boolean; label: string; onChange: (value: boolean) => void }): ReactNode {
  return h(ShadcnSwitch, { checked, disabled, 'aria-label': label, onCheckedChange: onChange })
}

function Field({ label, value, placeholder, type = 'text', onChange }: { label: string; value: string; placeholder: string; type?: string; onChange: (value: string) => void }): ReactNode {
  return h('label', { className: 'block' }, [
    h('span', { key: 'label', className: 'mb-1 block text-xs font-medium text-[hsl(var(--muted-foreground))]' }, label),
    h('input', { key: 'input', type, value, placeholder, className: 'input', onChange: (event: { target: { value: string } }) => onChange(event.target.value) }),
  ])
}

function ProviderCard({
  enabledKey,
  copy,
  form,
  saving,
  saveLabel,
  savingLabel,
  idKey,
  redirectKey,
  providerNameKey,
  onFieldChange,
  onToggle,
  onSave,
}: {
  enabledKey: keyof AuthSettingsForm
  copy: ProviderCopy
  form: AuthSettingsForm
  saving: boolean
  saveLabel: string
  savingLabel: string
  idKey: keyof AuthSettingsForm
  redirectKey: keyof AuthSettingsForm
  providerNameKey?: keyof AuthSettingsForm
  onFieldChange: AuthSettingsPageProps['onFieldChange']
  onToggle: AuthSettingsPageProps['onToggle']
  onSave: () => void
}): ReactNode {
  const enabled = Boolean(form[enabledKey])
  const field = (key: keyof AuthSettingsForm, label: string, placeholder: string, type = 'text') => h(Field, { key, label, value: String(form[key] || ''), placeholder, type, onChange: (value) => onFieldChange(key, value) })
  return h('section', { className: 'card overflow-hidden' }, [
    h('header', { key: 'header', className: 'flex items-center justify-between border-b border-[hsl(var(--border))] px-5 py-4' }, [
      h('div', { key: 'title', className: 'min-w-0' }, [h('h2', { key: 'name', className: 'text-sm font-semibold text-[hsl(var(--foreground))]' }, copy.title), h('p', { key: 'description', className: 'mt-1 text-xs text-[hsl(var(--muted-foreground))]' }, copy.description)]),
      h(Switch, { key: 'switch', checked: enabled, disabled: saving, label: copy.enabled, onChange: (value) => onToggle(enabledKey, value) }),
    ]),
    enabled ? h('div', { key: 'body', className: 'space-y-3 p-5' }, [
      copy.appIdLabel ? field('wechat_connect_app_id', copy.appIdLabel, copy.appIdPlaceholder || '') : null,
      providerNameKey && copy.providerNameLabel ? field(providerNameKey, copy.providerNameLabel, copy.providerNamePlaceholder || '') : null,
      copy.issuerLabel ? field('oidc_connect_issuer_url', copy.issuerLabel, copy.issuerPlaceholder || '', 'url') : null,
      !copy.appIdLabel ? field(idKey, copy.idLabel, copy.idPlaceholder) : null,
      field(redirectKey, copy.redirectLabel, copy.redirectPlaceholder, 'url'),
      h('button', { key: 'save', type: 'button', className: 'btn btn-secondary w-full', disabled: saving, onClick: onSave }, saving ? savingLabel : saveLabel),
    ]) : null,
  ])
}

function RegistrationCard({ form, copy, saving, onToggle }: { form: AuthSettingsForm; copy: AuthSettingsPageCopy['registration']; saving: boolean; onToggle: AuthSettingsPageProps['onToggle'] }): ReactNode {
  const item = (key: keyof AuthSettingsForm, label: string, hint: string) => h('div', { key, className: 'flex items-center justify-between gap-4' }, [h('div', { key: 'text' }, [h('p', { key: 'label', className: 'text-sm font-medium text-[hsl(var(--foreground))]' }, label), h('p', { key: 'hint', className: 'text-xs text-[hsl(var(--muted-foreground))]' }, hint)]), h(Switch, { key: 'switch', checked: Boolean(form[key]), disabled: saving, label, onChange: (value) => onToggle(key, value) })])
  return h('section', { className: 'card' }, [
    h('header', { key: 'header', className: 'flex items-center justify-between border-b border-[hsl(var(--border))] px-6 py-4' }, [h('div', { key: 'text' }, [h('h2', { key: 'title', className: 'text-base font-semibold text-[hsl(var(--foreground))]' }, copy.title), h('p', { key: 'description', className: 'mt-1 text-sm text-[hsl(var(--muted-foreground))]' }, copy.description)]), h(Switch, { key: 'switch', checked: form.registration_enabled, disabled: saving, label: copy.title, onChange: (value) => onToggle('registration_enabled', value) })]),
    form.registration_enabled ? h('div', { key: 'body', className: 'space-y-4 p-6' }, [item('email_verify_enabled', copy.emailVerify, copy.emailVerifyHint), item('password_reset_enabled', copy.passwordReset, copy.passwordResetHint), item('invitation_code_enabled', copy.invitationCode, copy.invitationCodeHint)]) : null,
  ])
}

export default function AuthSettingsPage({ form, loading, saving, copy, onFieldChange, onToggle, onSave }: AuthSettingsPageProps) {
  const providers: Array<{ enabledKey: keyof AuthSettingsForm; copyKey: keyof AuthSettingsForm; idKey: keyof AuthSettingsForm; redirectKey: keyof AuthSettingsForm; providerNameKey?: keyof AuthSettingsForm }> = [
    { enabledKey: 'oidc_connect_enabled', copyKey: 'oidc_connect_enabled', idKey: 'oidc_connect_client_id', redirectKey: 'oidc_connect_redirect_url', providerNameKey: 'oidc_connect_provider_name' },
    { enabledKey: 'github_oauth_enabled', copyKey: 'github_oauth_enabled', idKey: 'github_oauth_client_id', redirectKey: 'github_oauth_redirect_url' },
    { enabledKey: 'google_oauth_enabled', copyKey: 'google_oauth_enabled', idKey: 'google_oauth_client_id', redirectKey: 'google_oauth_redirect_url' },
    { enabledKey: 'wechat_connect_enabled', copyKey: 'wechat_connect_enabled', idKey: 'wechat_connect_app_id', redirectKey: 'wechat_connect_redirect_url' },
    { enabledKey: 'linuxdo_connect_enabled', copyKey: 'linuxdo_connect_enabled', idKey: 'linuxdo_connect_client_id', redirectKey: 'linuxdo_connect_redirect_url' },
    { enabledKey: 'dingtalk_connect_enabled', copyKey: 'dingtalk_connect_enabled', idKey: 'dingtalk_connect_client_id', redirectKey: 'dingtalk_connect_redirect_url' },
  ]
  if (loading) return h('div', { className: 'flex items-center justify-center py-12' }, h('span', { className: 'h-8 w-8 animate-spin rounded-full border-2 border-[hsl(var(--border))] border-t-[hsl(var(--foreground))]' }))
  return h('div', { className: 'mx-auto max-w-4xl space-y-6' }, [
    h(RegistrationCard, { key: 'registration', form, copy: copy.registration, saving, onToggle }),
    h('div', { key: 'providers', className: 'grid grid-cols-1 gap-4 md:grid-cols-2' }, providers.map(({ enabledKey, copyKey, idKey, redirectKey, providerNameKey }) => h(ProviderCard, { key: enabledKey, enabledKey, copy: copy.provider(copyKey), form, saving, saveLabel: copy.save, savingLabel: copy.saving, idKey, redirectKey, providerNameKey, onFieldChange, onToggle, onSave }))),
  ])
}
