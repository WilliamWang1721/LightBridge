<template>
  <span class="hidden" aria-hidden="true"></span>
</template>

<script setup lang="ts">
import { watch } from 'vue'
import { useAuthStore } from '@/stores/auth'
import { useUIPlatform } from '@/composables/useUIPlatform'

const authStore = useAuthStore()
const { hydrateAccountPreferences, clearAccountPreferences } = useUIPlatform()

watch(
  () => authStore.isAuthenticated,
  (authenticated) => {
    if (authenticated) {
      void hydrateAccountPreferences()
    } else {
      clearAccountPreferences()
    }
  },
  { immediate: true },
)
</script>
