import { mount } from "@vue/test-utils";
import { describe, expect, it } from "vitest";
import { createI18n } from "vue-i18n";
import SubscriptionPlanCard from "../SubscriptionPlanCard.vue";

const i18n = createI18n({
  legacy: false,
  locale: "en",
  fallbackWarn: false,
  missingWarn: false,
  messages: {
    en: {
      payment: {
        days: "days",
        models: "Models",
        planCard: {
          quota: "Quota",
          rate: "Rate",
          unlimited: "Unlimited",
        },
        subscribeNow: "Subscribe now",
      },
      admin: {
        groups: {
          upstreamPlatforms: {
            custom: "Custom",
          },
          upstreamProtocols: {
            openai_responses: "Responses",
            openai_chat_completions: "Chat Completions",
            anthropic_messages: "Messages",
            gemini: "Gemini",
          },
        },
      },
    },
  },
});

const mountPlanCard = () =>
  mount(SubscriptionPlanCard, {
    props: {
      plan: {
        id: 1,
        group_id: 10,
        group_name: "Custom Group",
        group_icon: "server",
        group_color: "#0F766E",
        upstream_platforms: ["custom"],
        upstream_protocols: ["openai_chat_completions"],
        available_ingress_protocols: ["openai_chat_completions"],
        name: "Pro",
        price: 10,
        amount: 1000,
        features: [],
        rate_multiplier: 1,
        validity_days: 30,
        validity_unit: "day",
        is_active: true,
      },
    },
    global: { plugins: [i18n] },
  });

describe("SubscriptionPlanCard", () => {
  it("shows the bound group identity and only its actual upstream protocol", () => {
    const text = mountPlanCard().text();

    expect(text).toContain("Custom Group");
    expect(text).toContain("Chat Completions");
    expect(text).not.toContain("Responses");
    expect(text).not.toContain("Messages");
    expect(text).not.toContain("Gemini");
  });

  it("uses the group's configured color without requiring a platform", () => {
    const wrapper = mountPlanCard();

    expect(wrapper.attributes("style")).toContain("border-color");
    expect(wrapper.html()).toContain("rgb(15, 118, 110)");
  });
});
