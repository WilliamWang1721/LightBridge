import { type ReactNode } from 'react'
import { createShadcnElement as h } from './ui/createElement'

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

const iconPaths: Record<IconName, ReactNode> = {
  github: h('path', {
    fillRule: 'evenodd',
    clipRule: 'evenodd',
    d: 'M12 2C6.477 2 2 6.484 2 12.017c0 4.425 2.865 8.18 6.839 9.504.5.092.682-.217.682-.483 0-.237-.008-.868-.013-1.703-2.782.605-3.369-1.343-3.369-1.343-.454-1.158-1.11-1.466-1.11-1.466-.908-.62.069-.608.069-.608 1.003.07 1.531 1.032 1.531 1.032.892 1.53 2.341 1.088 2.91.832.092-.647.35-1.088.636-1.338-2.22-.253-4.555-1.113-4.555-4.951 0-1.093.39-1.988 1.029-2.688-.103-.253-.446-1.272.098-2.65 0 0 .84-.27 2.75 1.026A9.564 9.564 0 0112 6.844c.85.004 1.705.115 2.504.337 1.909-1.296 2.747-1.027 2.747-1.027.546 1.379.202 2.398.1 2.651.64.7 1.028 1.595 1.028 2.688 0 3.848-2.339 4.695-4.566 4.943.359.309.678.92.678 1.855 0 1.338-.012 2.419-.012 2.747 0 .268.18.58.688.482A10.019 10.019 0 0022 12.017C22 6.484 17.522 2 12 2z',
  }),
  users: h('path', {
    fill: 'none',
    stroke: 'currentColor',
    strokeWidth: 1.5,
    strokeLinecap: 'round',
    strokeLinejoin: 'round',
    d: 'M18 18.72a9.094 9.094 0 003.741-.479 3 3 0 00-4.682-2.72m.94 3.198l.001.031c0 .225-.012.447-.037.666A11.944 11.944 0 0112 21c-2.17 0-4.207-.576-5.963-1.584A6.062 6.062 0 016 18.719m12 0a5.971 5.971 0 00-.941-3.197m0 0A5.995 5.995 0 0012 12.75a5.995 5.995 0 00-5.058 2.772m0 0a3 3 0 00-4.681 2.72 8.986 8.986 0 003.74.477m.94-3.197a5.971 5.971 0 00-.94 3.197M15 6.75a3 3 0 11-6 0 3 3 0 016 0zm6 3a2.25 2.25 0 11-4.5 0 2.25 2.25 0 014.5 0zm-13.5 0a2.25 2.25 0 11-4.5 0 2.25 2.25 0 014.5 0z',
  }),
  link: h('path', {
    fill: 'none',
    stroke: 'currentColor',
    strokeWidth: 1.5,
    strokeLinecap: 'round',
    strokeLinejoin: 'round',
    d: 'M7.5 21L3 16.5m0 0L7.5 12M3 16.5h13.5m-2.25-13.5L21 7.5m0 0L16.5 12M21 7.5H7.5',
  }),
  help: h('path', {
    fill: 'none',
    stroke: 'currentColor',
    strokeWidth: 1.5,
    strokeLinecap: 'round',
    strokeLinejoin: 'round',
    d: 'M9.879 7.519c1.171-1.025 3.071-1.025 4.242 0 1.172 1.172 1.172 2.687 0 3.712-.203.179-.43.326-.67.442-.745.361-1.45.999-1.45 1.827v.75M21 12a9 9 0 11-18 0 9 9 0 0118 0zm-9 5.25h.008v.008H12v-.008z',
  }),
}

function Icon({ name, size = 'md' }: { name: IconName; size?: 'sm' | 'md' }) {
  return h('svg', {
    className: size === 'sm' ? 'h-4 w-4' : 'h-6 w-6',
    fill: name === 'github' ? 'currentColor' : 'none',
    viewBox: '0 0 24 24',
    'aria-hidden': 'true',
  }, iconPaths[name])
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
