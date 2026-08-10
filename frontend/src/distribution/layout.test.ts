import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'

const root = resolve(dirname(fileURLToPath(import.meta.url)), '..')

describe('content distribution page layout', () => {
  it.each(['views/user/DistributionsView.vue', 'views/admin/DistributionsView.vue'])(
    'renders %s inside the shared application shell',
    (relativePath) => {
      const source = readFileSync(resolve(root, relativePath), 'utf8')

      expect(source).toContain("import AppLayout from '@/components/layout/AppLayout.vue'")
      expect(source).toMatch(/<template>\s*<AppLayout>/)
      expect(source).toMatch(/<\/AppLayout>\s*<\/template>/)
    }
  )
})
