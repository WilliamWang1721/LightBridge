<template>
  <AppLayout>
    <ReactPageHost
      :load="loadFeedbackPage"
      :props="pageProps"
      :fallback="FeedbackPageLegacy"
      :error-message="t('common.error')"
    />
  </AppLayout>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import ReactPageHost from '@/console/ReactPageHost.vue'
import FeedbackPageLegacy from '@/console/vue/FeedbackPageLegacy.vue'
import type { FeedbackPageProps } from '@/console/react/FeedbackPage'

const { t } = useI18n()

const githubRepoUrl = 'https://github.com/WilliamWang1721/LightBridge'
const githubIssueUrl = `${githubRepoUrl}/issues/new`
const linuxDoProfileUrl = 'https://linux.do/u/starfox'

const loadFeedbackPage = () => import('@/console/react/FeedbackPage')

const pageProps = computed<FeedbackPageProps>(() => ({
  githubIssueUrl,
  linuxDoProfileUrl,
  copy: {
    quickActions: t('admin.feedback.quickActions'),
    githubIssue: t('admin.feedback.githubIssue'),
    githubIssueDesc: t('admin.feedback.githubIssueDesc'),
    linuxDoProfile: t('admin.feedback.linuxDoProfile'),
    referencedProjects: t('admin.feedback.referencedProjects'),
    referencedProjectsDesc: t('admin.feedback.referencedProjectsDesc'),
    authconvName: t('admin.feedback.authconvName'),
    authconvDesc: t('admin.feedback.authconvDesc'),
    contactInfo: t('admin.feedback.contactInfo'),
    howToReport: t('admin.feedback.howToReport'),
    howToReportDesc: t('admin.feedback.howToReportDesc'),
  },
}))
</script>
