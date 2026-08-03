import { mount } from '@vue/test-utils'
import { createI18n } from 'vue-i18n'
import { describe, expect, it } from 'vitest'
import AvailableChannelsTable from '../AvailableChannelsTable.vue'

const i18n = createI18n({
  legacy: false,
  locale: 'en',
  messages: {
    en: {
      availableChannels: {
        exclusive: 'Exclusive',
        exclusiveTooltip: 'Exclusive group',
        public: 'Public',
        publicTooltip: 'Public group',
      },
    },
  },
})

describe('AvailableChannelsTable', () => {
  it('renders channel groups once instead of repeating them for every pricing namespace', () => {
    const wrapper = mount(AvailableChannelsTable, {
      props: {
        columns: {
          name: 'Name',
          description: 'Description',
          platform: 'Pricing namespace',
          groups: 'Groups',
          supportedModels: 'Models',
        },
        rows: [
          {
            name: 'Primary',
            description: '',
            platforms: [
              {
                platform: 'anthropic',
                groups: [
                  {
                    id: 7,
                    name: 'Neutral Group',
                    icon: 'server',
                    color: '#0F766E',
                    upstream_platforms: ['custom'],
                    upstream_protocols: ['openai_chat_completions'],
                    available_ingress_protocols: ['openai_chat_completions'],
                    subscription_type: 'standard',
                    rate_multiplier: 1,
                    is_exclusive: false,
                  },
                ],
                supported_models: [],
              },
              {
                platform: 'openai',
                groups: [
                  {
                    id: 7,
                    name: 'Neutral Group',
                    icon: 'server',
                    color: '#0F766E',
                    upstream_platforms: ['custom'],
                    upstream_protocols: ['openai_chat_completions'],
                    available_ingress_protocols: ['openai_chat_completions'],
                    subscription_type: 'standard',
                    rate_multiplier: 1,
                    is_exclusive: false,
                  },
                ],
                supported_models: [],
              },
            ],
          },
        ],
        loading: false,
        pricingKeyPrefix: 'pricing',
        noPricingLabel: 'No pricing',
        noModelsLabel: 'No models',
        emptyLabel: 'Empty',
        userGroupRates: {},
      },
      global: {
        plugins: [i18n],
        stubs: {
          Icon: true,
          PlatformIcon: {
            props: ['platform'],
            template: '<span class="platform">{{ platform }}</span>',
          },
          GroupBadge: {
            props: ['name'],
            template: '<span class="group-badge">{{ name }}</span>',
          },
          GroupUpstreamBadges: {
            props: ['upstreamProtocols'],
            template: '<span class="protocols">{{ upstreamProtocols.join(",") }}</span>',
          },
          SupportedModelChip: true,
        },
      },
    })

    expect(wrapper.findAll('.group-badge')).toHaveLength(1)
    expect(wrapper.text()).toContain('anthropic')
    expect(wrapper.text()).toContain('openai')
    expect(wrapper.text()).toContain('openai_chat_completions')
    expect(wrapper.text()).not.toContain('anthropic_messages')
  })
})
