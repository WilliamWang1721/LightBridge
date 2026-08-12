import { act, createElement } from 'react'
import { createRoot } from 'react-dom/client'
import { afterEach, describe, expect, it } from 'vitest'
import FeedbackPage, { type FeedbackPageProps } from './FeedbackPage'

;(globalThis as typeof globalThis & { IS_REACT_ACT_ENVIRONMENT?: boolean }).IS_REACT_ACT_ENVIRONMENT = true

const props: FeedbackPageProps = {
  githubIssueUrl: 'https://example.com/issues',
  linuxDoProfileUrl: 'https://example.com/profile',
  copy: {
    quickActions: 'Quick actions',
    githubIssue: 'GitHub issue',
    githubIssueDesc: 'Open an issue',
    linuxDoProfile: 'LinuxDo profile',
    referencedProjects: 'Referenced projects',
    referencedProjectsDesc: 'Projects used by LightBridge',
    authconvName: 'authconv',
    authconvDesc: 'Authentication conversion',
    contactInfo: 'Contact information',
    howToReport: 'How to report',
    howToReportDesc: 'Tell us what happened',
  },
}

const roots: Array<{ container: HTMLDivElement; unmount: () => Promise<void> }> = []

afterEach(async () => {
  await act(async () => {
    await Promise.all(roots.splice(0).map(async ({ container, unmount }) => {
      await unmount()
      container.remove()
    }))
  })
})

describe('FeedbackPage', () => {
  it('keeps the existing console sections and links in React', async () => {
    const container = document.createElement('div')
    document.body.appendChild(container)
    const root = createRoot(container)
    roots.push({ container, unmount: async () => { root.unmount() } })

    await act(async () => {
      root.render(createElement(FeedbackPage, props))
    })

    expect(container.textContent).toContain('Quick actions')
    expect(container.textContent).toContain('Referenced projects')
    expect(container.querySelector('a[href="https://example.com/issues"]')).not.toBeNull()
    expect(container.querySelectorAll('section')).toHaveLength(0)
  })
})
