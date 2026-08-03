import type { DefaultSubscriptionSetting } from "@/api/admin/settings";
import type { AdminGroup, LoginAgreementDocument } from "@/types";

export type SettingsTab =
  | "general"
  | "agreement"
  | "themes"
  | "security"
  | "users"
  | "gateway"
  | "payment"
  | "email"
  | "backup";

export const SETTINGS_TABS = [
  { key: "general", icon: "home" },
  { key: "agreement", icon: "document" },
  { key: "themes", icon: "sparkles" },
  { key: "security", icon: "shield" },
  { key: "users", icon: "user" },
  { key: "gateway", icon: "server" },
  { key: "payment", icon: "creditCard" },
  { key: "email", icon: "mail" },
  { key: "backup", icon: "database" },
] as const satisfies ReadonlyArray<{ key: SettingsTab; icon: string }>;

const SETTINGS_TAB_KEYBOARD_ACTIONS = {
  ArrowLeft: -1,
  ArrowUp: -1,
  ArrowRight: 1,
  ArrowDown: 1,
  Home: "first",
  End: "last",
} as const;

export function resolveNextSettingsTab(
  currentTab: SettingsTab,
  key: string,
  tabs: ReadonlyArray<{ key: SettingsTab }> = SETTINGS_TABS,
): SettingsTab | null {
  const action =
    SETTINGS_TAB_KEYBOARD_ACTIONS[
      key as keyof typeof SETTINGS_TAB_KEYBOARD_ACTIONS
    ];
  if (action === undefined) {
    return null;
  }

  const currentIndex = tabs.findIndex(
    (item) => item.key === currentTab,
  );
  let nextIndex = currentIndex < 0 ? 0 : currentIndex;

  if (action === "first") {
    nextIndex = 0;
  } else if (action === "last") {
    nextIndex = tabs.length - 1;
  } else {
    nextIndex =
      (nextIndex + action + tabs.length) % tabs.length;
  }

  return tabs[nextIndex]?.key ?? null;
}

export const TABLE_PAGE_SIZE_MIN = 5;
export const TABLE_PAGE_SIZE_MAX = 1000;
export const TABLE_PAGE_SIZE_DEFAULT = 20;

export function formatTablePageSizeOptions(options: number[]): string {
  return options.join(", ");
}

export function parseTablePageSizeOptionsInput(raw: string): number[] | null {
  const tokens = raw
    .split(",")
    .map((token) => token.trim())
    .filter((token) => token.length > 0);

  if (tokens.length === 0) {
    return null;
  }

  const parsed = tokens.map((token) => Number(token));
  if (parsed.some((value) => !Number.isInteger(value))) {
    return null;
  }

  const deduped = Array.from(new Set(parsed)).sort((a, b) => a - b);
  if (
    deduped.some(
      (value) => value < TABLE_PAGE_SIZE_MIN || value > TABLE_PAGE_SIZE_MAX,
    )
  ) {
    return null;
  }

  return deduped;
}

export function defaultLoginAgreementDocuments(): LoginAgreementDocument[] {
  return [
    { id: "terms", title: "服务条款", content_md: "" },
    { id: "usage-policy", title: "使用政策", content_md: "" },
    { id: "supported-regions", title: "支持的国家和地区", content_md: "" },
    {
      id: "service-specific-terms",
      title: "服务特定条款",
      content_md: "",
    },
  ];
}

export function normalizeLoginAgreementDocumentId(raw: string): string {
  return raw
    .trim()
    .toLowerCase()
    .replace(/[^a-z0-9_-]+/g, "-")
    .replace(/[-_]{2,}/g, "-")
    .replace(/^[-_]+|[-_]+$/g, "");
}

export function loginAgreementRoutePath(
  doc: LoginAgreementDocument,
  index: number,
): string {
  const id =
    normalizeLoginAgreementDocumentId(doc.id || doc.title) || `doc-${index + 1}`;
  return `/legal/${id}`;
}

export function findDuplicateLoginAgreementDocumentId(
  documents: LoginAgreementDocument[],
): string | null {
  const seen = new Set<string>();
  for (const doc of documents) {
    if (seen.has(doc.id)) {
      return doc.id;
    }
    seen.add(doc.id);
  }
  return null;
}

export interface DefaultSubscriptionGroupOption {
  value: number;
  label: string;
  description: string | null;
  icon: AdminGroup["icon"];
  color: AdminGroup["color"];
  upstreamPlatforms: AdminGroup["upstream_platforms"];
  upstreamProtocols: AdminGroup["upstream_protocols"];
  availableIngressProtocols: AdminGroup["available_ingress_protocols"];
  subscriptionType: AdminGroup["subscription_type"];
  rate: number;
  [key: string]: unknown;
}

export function findNextAvailableSubscriptionGroup(
  groups: AdminGroup[],
  existingGroupIDs: number[],
): AdminGroup | undefined {
  const existing = new Set(existingGroupIDs);
  return groups.find((group) => !existing.has(group.id));
}

export function findDuplicateDefaultSubscription(
  subscriptions: DefaultSubscriptionSetting[],
): DefaultSubscriptionSetting | undefined {
  const seenGroupIDs = new Set<number>();
  return subscriptions.find((item) => {
    if (seenGroupIDs.has(item.group_id)) {
      return true;
    }
    seenGroupIDs.add(item.group_id);
    return false;
  });
}
