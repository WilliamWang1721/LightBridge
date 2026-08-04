<template>
  <section class="card overflow-hidden" data-testid="ui-platform-settings">
    <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
      <div class="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
            {{ copy.title }}
          </h2>
          <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
            {{ copy.description }}
          </p>
        </div>
        <button type="button" class="btn btn-secondary btn-sm" @click="resetPreferences">
          {{ copy.reset }}
        </button>
      </div>
    </div>

    <div class="space-y-6 p-6">
      <div class="grid gap-4 md:grid-cols-2">
        <label class="space-y-2">
          <span class="input-label">{{ copy.interfaceMode }}</span>
          <select class="input" :value="profile.mode" @change="setMode(($event.target as HTMLSelectElement).value)">
            <option value="legacy">{{ copy.legacy }}</option>
            <option value="modern">{{ copy.modern }}</option>
            <option value="package" :disabled="packages.length === 0">{{ copy.package }}</option>
          </select>
          <span class="input-hint">{{ copy.interfaceModeHint }}</span>
        </label>

        <label v-if="profile.mode === 'package'" class="space-y-2">
          <span class="input-label">{{ copy.activePackage }}</span>
          <select
            class="input"
            :value="profile.activePackageId || ''"
            @change="setPackage(($event.target as HTMLSelectElement).value)"
          >
            <option disabled value="">{{ copy.selectPackage }}</option>
            <option v-for="item in packages" :key="item.id" :value="item.id">
              {{ item.name }}
            </option>
          </select>
        </label>

        <div class="rounded-xl border border-gray-200 bg-gray-50 px-4 py-3 dark:border-dark-700 dark:bg-dark-900/40">
          <p class="text-sm font-medium text-gray-800 dark:text-gray-100">{{ copy.style }}</p>
          <p class="mt-1 text-sm text-gray-600 dark:text-gray-300">Luma</p>
          <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ copy.styleHint }}</p>
        </div>
      </div>

      <div class="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
        <label v-for="field in fields" :key="field.key" class="space-y-2">
          <span class="input-label">{{ field.label }}</span>
          <select
            class="input"
            :value="String(profile[field.key])"
            @change="setPreference(field.key, ($event.target as HTMLSelectElement).value)"
          >
            <option v-for="option in field.options" :key="option.id" :value="option.id">
              {{ option.label }}
            </option>
          </select>
        </label>
      </div>

      <div class="rounded-xl border border-primary-100 bg-primary-50/70 px-4 py-3 text-sm text-primary-800 dark:border-primary-900/40 dark:bg-primary-950/30 dark:text-primary-200">
        {{ copy.savedLocally }}
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useUIPlatform } from '@/composables/useUIPlatform'
import type { UIAxis, UIProfileOverrides } from '@/ui-platform/types'

interface PackageOption {
  id: string
  name: string
}

const props = withDefaults(defineProps<{
  packages?: PackageOption[]
}>(), {
  packages: () => [],
})

const { locale } = useI18n()
const { profile, options, updatePreferences, resetPreferences } = useUIPlatform()

const copy = computed(() => {
  const zh = locale.value.toLowerCase().startsWith('zh')
  return zh
    ? {
        title: 'UI 界面设置',
        description: '在经典界面、LightBridge Luma 界面和已安装的 UI 包之间切换，并独立调整各项视觉参数。',
        reset: '恢复默认',
        interfaceMode: '界面来源',
        interfaceModeHint: '经典界面继续保留；新版界面和自定义 UI 包使用统一的 UI 平台参数。',
        legacy: '经典界面',
        modern: 'LightBridge Luma',
        package: '自定义 UI 包',
        activePackage: 'UI 包',
        selectPackage: '选择已安装的 UI 包',
        style: '组件风格',
        styleHint: '风格固定为 Luma，不能被用户或 UI 包覆盖。',
        savedLocally: '更改会立即预览并保存在当前浏览器。账户级同步将在后续服务端偏好接口中接入。',
      }
    : {
        title: 'UI settings',
        description: 'Switch between the classic UI, the LightBridge Luma UI, and installed UI packages while configuring each visual axis independently.',
        reset: 'Restore defaults',
        interfaceMode: 'Interface source',
        interfaceModeHint: 'The classic interface remains available. Modern and package modes share the same UI platform preferences.',
        legacy: 'Classic UI',
        modern: 'LightBridge Luma',
        package: 'Custom UI package',
        activePackage: 'UI package',
        selectPackage: 'Select an installed UI package',
        style: 'Component style',
        styleHint: 'The component style is fixed to Luma and cannot be overridden by users or packages.',
        savedLocally: 'Changes are applied immediately and saved in this browser. Account synchronization will use the future server-side preference endpoint.',
      }
})

const labels = computed<Record<UIAxis, string>>(() => {
  const zh = locale.value.toLowerCase().startsWith('zh')
  return zh
    ? {
        layout: '页面布局',
        baseColor: '基础颜色',
        chartColor: '图表颜色',
        heading: '标题风格',
        font: '字体',
        iconLibrary: '图标库',
        radius: '圆角',
        density: '界面密度',
        menu: '菜单样式',
        menuSize: '菜单尺寸',
        motion: '动效',
        tableStyle: '表格密度',
      }
    : {
        layout: 'Page layout',
        baseColor: 'Base color',
        chartColor: 'Chart color',
        heading: 'Heading',
        font: 'Font',
        iconLibrary: 'Icon library',
        radius: 'Radius',
        density: 'Interface density',
        menu: 'Menu style',
        menuSize: 'Menu size',
        motion: 'Motion',
        tableStyle: 'Table density',
      }
})

const fields = computed(() => [
  { key: 'baseColor' as const, label: labels.value.baseColor, options: options.baseColors },
  { key: 'chartColor' as const, label: labels.value.chartColor, options: options.chartColors },
  { key: 'heading' as const, label: labels.value.heading, options: options.headings },
  { key: 'font' as const, label: labels.value.font, options: options.fonts },
  { key: 'iconLibrary' as const, label: labels.value.iconLibrary, options: options.iconLibraries },
  { key: 'radius' as const, label: labels.value.radius, options: options.radii },
  { key: 'layout' as const, label: labels.value.layout, options: options.layouts },
  { key: 'density' as const, label: labels.value.density, options: options.densities },
  { key: 'menu' as const, label: labels.value.menu, options: options.menus },
  { key: 'menuSize' as const, label: labels.value.menuSize, options: options.menuSizes },
  { key: 'motion' as const, label: labels.value.motion, options: options.motions },
  { key: 'tableStyle' as const, label: labels.value.tableStyle, options: options.tableStyles },
])

function setPreference(key: UIAxis, value: string) {
  updatePreferences({ [key]: value } as UIProfileOverrides)
}

function setMode(value: string) {
  if (value === 'legacy' || value === 'modern') {
    updatePreferences({ mode: value })
    return
  }

  if (value === 'package') {
    const selected = profile.value.activePackageId || props.packages[0]?.id
    if (selected) updatePreferences({ mode: 'package', activePackageId: selected })
  }
}

function setPackage(id: string) {
  if (!id) return
  updatePreferences({ mode: 'package', activePackageId: id })
}
</script>
