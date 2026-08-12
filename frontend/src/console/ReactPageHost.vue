<template>
  <div ref="mountNode" class="contents" data-console-runtime="react"></div>
  <component v-if="error && fallback" :is="fallback" />
  <div
    v-else-if="error"
    class="rounded-lg border border-red-200 bg-red-50 p-4 text-sm text-red-700 dark:border-red-900/60 dark:bg-red-950/30 dark:text-red-300"
    role="alert"
  >
    {{ error }}
  </div>
</template>

<script setup lang="ts">
import { createElement, type ComponentType } from 'react'
import { createRoot, type Root } from 'react-dom/client'
import { onBeforeUnmount, onMounted, ref, watch, type Component } from 'vue'

type ReactPageProps = Record<string, unknown>
type ReactPageModule = { default?: unknown }

const props = withDefaults(defineProps<{
  load: () => Promise<ReactPageModule>
  props?: ReactPageProps
  errorMessage?: string
  fallback?: Component
}>(), {
  props: () => ({}),
  errorMessage: 'Failed to load console page',
  fallback: undefined,
})

const mountNode = ref<HTMLElement | null>(null)
const error = ref('')
let root: Root | null = null
let component: ComponentType<ReactPageProps> | null = null
let mounted = true
let generation = 0

function render() {
  if (!root || !component) return
  root.render(createElement(component, props.props))
}

async function loadPage() {
  const currentGeneration = ++generation
  error.value = ''

  try {
    const module = await props.load()
    if (!mounted || currentGeneration !== generation || !mountNode.value) return

    if (typeof module.default !== 'function' && typeof module.default !== 'object') {
      throw new Error('React console page export is invalid')
    }

    component = module.default as ComponentType<ReactPageProps>
    root = createRoot(mountNode.value)
    render()
  } catch {
    if (mounted && currentGeneration === generation) error.value = props.errorMessage
  }
}

watch(() => props.props, render, { deep: true })

onMounted(() => { void loadPage() })

onBeforeUnmount(() => {
  mounted = false
  ++generation
  root?.unmount()
  root = null
  component = null
})
</script>
