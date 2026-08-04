from pathlib import Path

TEMPLATE = Path('frontend/src/components/account/templates/CreateAccountModal.template.html')
text = TEMPLATE.read_text(encoding='utf-8')

start_marker = '      <!-- Custom Provider Configuration -->\n'
relay_marker = '      <!-- Custom relay mode -->\n'

if 'data-testid="custom-connection-settings"' not in text:
    if text.count(start_marker) != 1 or text.count(relay_marker) != 1:
        raise SystemExit('Custom provider section markers are missing or duplicated')

    connection_open = '''      <!-- Custom Provider Configuration -->
      <section
        v-if="form.platform === 'custom'"
        class="space-y-4 rounded-xl border border-gray-200 bg-gray-50/70 p-4 dark:border-dark-600 dark:bg-dark-800/40"
        data-testid="custom-connection-settings"
      >
        <div>
          <h3 class="text-sm font-semibold text-gray-900 dark:text-white">
            {{ t('admin.accounts.custom.title') }}
          </h3>
          <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
            {{ t('admin.accounts.custom.presetHint') }}
          </p>
        </div>
'''
    text = text.replace(start_marker, connection_open, 1)
    text = text.replace(relay_marker, '      </section>\n\n' + relay_marker, 1)

custom_label_old = "        <label class=\"input-label\">{{ t('admin.accounts.apiKey') }}</label>\n        <input\n          v-model=\"form.customApiKey\""
custom_label_new = "        <label class=\"input-label\">{{ t('admin.accounts.apiKeyRequired') }}</label>\n        <input\n          v-model=\"form.customApiKey\""
if custom_label_old in text:
    text = text.replace(custom_label_old, custom_label_new, 1)
elif custom_label_new not in text:
    raise SystemExit('Custom API key label block not found')

custom_input_old = '''          required
          class="input"
          :placeholder="t('admin.accounts.custom.apiKeyPlaceholder')"
        />'''
custom_input_new = '''          required
          class="input font-mono"
          :placeholder="t('admin.accounts.custom.apiKeyPlaceholder')"
          data-testid="custom-api-key"
        />'''
if custom_input_old in text:
    text = text.replace(custom_input_old, custom_input_new, 1)
elif 'data-testid="custom-api-key"' not in text:
    raise SystemExit('Custom API key input block not found')

generic_key_old = '''        <template v-else>
        <div>
          <label class="input-label">{{ t('admin.accounts.apiKeyRequired') }}</label>'''
generic_key_new = '''        <template v-else-if="form.platform !== 'custom'">
        <div>
          <label class="input-label">{{ t('admin.accounts.apiKeyRequired') }}</label>'''
if generic_key_old in text:
    text = text.replace(generic_key_old, generic_key_new, 1)
elif generic_key_new not in text:
    raise SystemExit('Generic API key branch not found')

relay_open_old = '''      <div
        v-if="form.platform === 'custom'"
        class="border-t border-gray-200 pt-4 dark:border-dark-600"
      >'''
relay_open_new = '''      <section
        v-if="form.platform === 'custom'"
        class="rounded-xl border border-gray-200 bg-white p-4 dark:border-dark-600 dark:bg-dark-800/30"
        data-testid="custom-routing-settings"
      >'''
if relay_open_old in text:
    text = text.replace(relay_open_old, relay_open_new, 1)
elif relay_open_new not in text:
    raise SystemExit('Custom routing wrapper not found')

relay_close_old = '''        </div>
      </div>

      <!-- LightBridge Connect Configuration (for New API and compatible services) -->'''
relay_close_new = '''        </div>
      </section>

      <!-- LightBridge Connect Configuration (for New API and compatible services) -->'''
if relay_close_old in text:
    text = text.replace(relay_close_old, relay_close_new, 1)
elif relay_close_new not in text:
    raise SystemExit('Custom routing closing tag not found')

if text.count('v-model="form.customApiKey"') != 1:
    raise SystemExit('Custom API key field is not unique')
if text.count('v-model="apiKeyValue"') != 1:
    raise SystemExit('Generic API key field count changed unexpectedly')
if "v-else-if=\"form.platform !== 'custom'\"" not in text:
    raise SystemExit('Custom provider is not excluded from generic API key rendering')

TEMPLATE.write_text(text, encoding='utf-8')
