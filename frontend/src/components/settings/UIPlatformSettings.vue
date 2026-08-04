<template>
  <section
    class="overflow-hidden rounded-[calc(var(--ui-radius)+0.25rem)] border border-[hsl(var(--border))] bg-[hsl(var(--card))] text-[hsl(var(--card-foreground))] shadow-sm"
    data-testid="ui-platform-settings"
  >
    <div class="border-b border-[hsl(var(--border))] px-6 py-4">
      <div class="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h2 class="text-lg font-semibold">
            {{ copy.title }}
          </h2>
          <p class="mt-1 text-sm text-[hsl(var(--muted-foreground))]">
            {{ copy.description }}
          </p>
        </div>
        <Button variant="outline" size="sm" @click="resetPreferences">
          {{ copy.reset }}
        </Button>
      </div>
    </div>

    <div class="space-y-6 p-6">
      <div class="grid gap-4 md:grid-cols-2">
        <div class="space-y-2">
          <Label>{{ copy.interfaceMode }}</Label>
          <Select
            :model-value="profile.mode"
            @update:model-value="setMode(String($event))"
          >
            <SelectItem value="legacy">{{ copy.legacy }}</SelectItem>
            <SelectItem value="modern">{{ copy.modern }}</SelectItem>
            <SelectItem value="package" :disabled="availablePackages.length === 0">
              {{ copy.package }}
            </SelectItem>
          </Select>
          <p class="text-xs text-[hsl(var(--muted-foreground))]">{{ copy.interfaceModeHint }}</p>
        </div>

        <div v-if="profile.mode === 'package'" class="space-y-2">
          <Label>{{ copy.activePackage }}</Label>
          <Select
            :model-value="profile.activePackageId || undefined"
            :placeholder="copy.selectPackage"
            @update:model-value="setPackage(String($event))"
          >
            <SelectItem v-for="item in availablePackages" :key="item.id" :value="item.id">
              {{ item.name }}
            </SelectItem>
          </Select>
        </div>

        <div class="rounded-[var(--ui-radius)] border border-[hsl(var(--border))] bg-[hsl(var(--muted))] px-4 py-3">
          <p class="text-sm font-medium">{{ copy.style }}</p>
          <p class="mt-1 text-sm">Luma</p>
          <p class="mt-1 text-xs text-[hsl(var(--muted-foreground))]">{{ copy.styleHint }}</p>
        </div>
      </div>

      <div class="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
        <div v-for="field in fields" :key="field.key" class="space-y-2">
          <Label>{{ field.label }}</Label>
          <Select
            :model-value="String(profile[field.key])"
            @update:model-value="setPreference(field.key, String($event))"
          >
            <SelectItem v-for="option in field.options" :key="option.id" :value="option.id">
              {{ option.label }}
            </SelectItem>
          </Select>
        </div>
      </div>

      <div class="rounded-[var(--ui-radius)] border border-[hsl(var(--primary)/0.18)] bg-[hsl(var(--primary)/0.08)] px-4 py-3 text-sm text-[hsl(var(--foreground))]">
        {{ copy.savedLocally }}
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { Button } from '@/components/ui/button'
import { Label } from '@/components/ui/label'
import { Select, SelectItem } from '@/components/ui/select'
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

const injectedPackageId = typeof document === 'undefined'
  ? ''
  : document.querySelector<HTMLLinkElement>('link[data-lightbridge-ui-theme]')?.dataset.lightbridgeUiTheme || ''

const availablePackages = computed(() => {
  const packages = new Map(props.packages.map((item) => [item.id, item]))
  if (injectedPackageId && !packages.has(injectedPackageId)) {
    packages.set(injectedPackageId, { id: injectedPackageId, name: injectedPackageId })
  }
  return Array.from(packages.values())
})

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
    const selected = profile.value.activePackageId || availablePackages.value[0]?.id
    if (selected) updatePreferences({ mode: 'package', activePackageId: selected })
  }
}

function setPackage(id: string) {
  if (!id) return
  updatePreferences({ mode: 'package', activePackageId: id })
}
</script>
