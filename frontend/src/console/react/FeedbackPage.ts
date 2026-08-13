import { type ReactNode } from 'react'
import { createShadcnElement as h } from './ui/createElement'
import { AppIcon } from './ui/app-icon'

type IconName = 'github' | 'users' | 'link' | 'help'

export interface FeedbackPageCopy {
  quickActions: string
  githubIssue: string
  githubIssueDesc: string
  linuxDoProfile: string
  referencedProjects: string
  referencedProjectsDesc: string
  authconvName: string
  authconvDesc: string
  contactInfo: string
  howToReport: string
  howToReportDesc: string
}

export interface FeedbackPageProps {
  copy: FeedbackPageCopy
  githubIssueUrl: string
  linuxDoProfileUrl: string
}

function Icon({ name, size = 'md' }: { name: IconName; size?: 'sm' | 'md' }) {
  return h(AppIcon, { name, className: size === 'sm' ? 'h-4 w-4' : 'h-6 w-6' })
}

function LinkCard({
  href,
  icon,
  iconClassName,
  title,
  description,
}: {
  href: string
  icon: IconName
  iconClassName: string
  title: string
  description: string
}) {
  return h('a', {
    href,
    target: '_blank',
    rel: 'noopener noreferrer',
    className: 'group flex items-center gap-4 rounded-lg border border-gray-200 p-4 transition-all hover:border-primary-300 hover:shadow-md dark:border-dark-600 dark:hover:border-primary-600',
  }, [
    h('div', { key: 'icon', className: iconClassName }, h(Icon, { name: icon })),
    h('div', { key: 'body' }, [
      h('h3', { key: 'title', className: 'font-medium text-gray-900 group-hover:text-primary-600 dark:text-white dark:group-hover:text-primary-400' }, title),
      h('p', { key: 'description', className: 'text-sm text-gray-500 dark:text-gray-400' }, description),
    ]),
  ])
}

function Section({ title, children }: { title: string; children?: ReactNode }) {
  return h('div', {
    className: 'rounded-xl border border-gray-200 bg-white p-6 shadow-sm dark:border-dark-700 dark:bg-dark-800',
  }, [
    h('h2', { key: 'title', className: 'mb-4 text-lg font-semibold text-gray-900 dark:text-white' }, title),
    h('div', { key: 'content' }, children),
  ])
}

export default function FeedbackPage({ copy, githubIssueUrl, linuxDoProfileUrl }: FeedbackPageProps) {
  return h('div', { className: 'space-y-6' }, [
    h(Section, { key: 'quick-actions', title: copy.quickActions }, h('div', { className: 'grid gap-4 sm:grid-cols-2 lg:grid-cols-3' }, [
      h(LinkCard, {
        key: 'github-issue',
        href: githubIssueUrl,
        icon: 'github',
        iconClassName: 'flex h-12 w-12 items-center justify-center rounded-lg bg-gray-900 text-white group-hover:bg-primary-600',
        title: copy.githubIssue,
        description: copy.githubIssueDesc,
      }),
      h(LinkCard, {
        key: 'linuxdo-profile',
        href: linuxDoProfileUrl,
        icon: 'users',
        iconClassName: 'flex h-12 w-12 items-center justify-center rounded-lg bg-green-100 text-green-600 group-hover:bg-green-600 group-hover:text-white dark:bg-green-900/30 dark:text-green-400',
        title: copy.linuxDoProfile,
        description: '@starfox',
      }),
    ])),
    h(Section, { key: 'referenced-projects', title: copy.referencedProjects }, [
      h('p', { key: 'description', className: 'mb-4 text-sm text-gray-600 dark:text-dark-300' }, copy.referencedProjectsDesc),
      h('div', { key: 'links', className: 'grid gap-4 sm:grid-cols-2 lg:grid-cols-3' }, h(LinkCard, {
        key: 'authconv',
        href: 'https://github.com/ltxgit/authconv',
        icon: 'link',
        iconClassName: 'flex h-12 w-12 items-center justify-center rounded-lg bg-blue-100 text-blue-600 group-hover:bg-blue-600 group-hover:text-white dark:bg-blue-900/30 dark:text-blue-400',
        title: copy.authconvName,
        description: copy.authconvDesc,
      })),
    ]),
    h(Section, { key: 'contact-information', title: copy.contactInfo }, h('div', { className: 'space-y-4' }, h('div', { className: 'flex items-start gap-3' }, [
      h('div', { key: 'icon', className: 'flex h-8 w-8 items-center justify-center rounded-lg bg-gray-100 dark:bg-dark-700' }, h(Icon, { name: 'help', size: 'sm' })),
    h('div', { key: 'body' }, [
        h('h3', { key: 'title', className: 'font-medium text-gray-900 dark:text-white' }, copy.howToReport),
        h('p', { key: 'description', className: 'mt-1 text-sm text-gray-500 dark:text-gray-400' }, copy.howToReportDesc),
      ]),
    ]))),
  ])
}
