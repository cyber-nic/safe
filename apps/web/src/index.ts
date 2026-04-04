import {
  buildDeleteItemMutation,
  buildPutItemMutation,
  createTotpItem,
  createVaultItemRecord,
  describeVaultItem,
  generateTotpCodeForItem,
  isTotpItem,
  parseAccountConfigRecord,
  parseCollectionHeadRecord,
  parseVaultEventRecords,
  parseVaultItemRecord,
  parseVaultItemRecords,
  replayCollectionAgainstHead,
  type AccountConfigRecord,
  type CollectionHeadRecord,
  type TotpCodeSnapshot,
  type VaultEventAction,
  type VaultEventRecord,
  type VaultItem,
  type VaultItemKind,
  type VaultItemRecord,
} from "../../../packages/ts-sdk/src/index.ts";
import {
  sampleAccountConfigRecord,
  sampleCollectionHeadRecord,
  sampleVaultEventRecords,
  sampleVaultItemRecords,
  sampleVaultSecretMaterial,
} from "../../../packages/test-vectors/src/index.ts";

export type VaultWorkspaceQuery = {
  text?: string;
  kind?: VaultItemKind | "all";
  tag?: string | "all";
  limit?: number;
};

export type VaultOverview = {
  accountId: string;
  defaultCollectionId: string;
  collectionCount: number;
  deviceCount: number;
  itemCount: number;
  itemCountByKind: Record<VaultItemKind, number>;
  latestSeq: number;
  latestEventId: string;
  lastUpdatedAt: string;
};

export type VaultSpotlightCard = {
  id: string;
  title: string;
  value: string;
  detail: string;
};

export type VaultListEntry = {
  id: string;
  kind: VaultItemKind;
  title: string;
  summary: string;
  tags: string[];
  matchedFields: string[];
};

export type AuthenticatorCard = {
  id: string;
  title: string;
  issuer: string;
  accountName: string;
  relatedLoginTitle: string | null;
  relatedLoginURL: string | null;
  summary: string;
  code: string | null;
  secondsRemaining: number | null;
  validFrom: string | null;
  validUntil: string | null;
  status: "locked" | "ready" | "error";
  statusDetail: string;
};

export type VaultSecretMaterial = Record<string, string>;

export type ActivityEntry = {
  eventId: string;
  sequence: number;
  occurredAt: string;
  action: VaultEventAction;
  itemId: string;
  itemTitle: string;
  itemKind: VaultItemKind | "deleted";
};

export type VaultInsight = {
  id: string;
  title: string;
  detail: string;
  severity: "info" | "warning";
  itemIds: string[];
};

export type VaultWorkspace = {
  accountConfig: AccountConfigRecord;
  head: CollectionHeadRecord | null;
  events: VaultEventRecord[];
  query: VaultWorkspaceQuery;
  overview: VaultOverview;
  spotlight: VaultSpotlightCard[];
  insights: VaultInsight[];
  items: VaultListEntry[];
  authenticators: AuthenticatorCard[];
  activity: ActivityEntry[];
  availableKinds: Array<VaultItemKind | "all">;
  availableTags: string[];
  itemRecords: VaultItemRecord[];
  starterRecords: VaultItemRecord[];
};

export type VaultWorkspaceUpdate = {
  workspace: VaultWorkspace;
  secretMaterial: VaultSecretMaterial;
  itemId: string;
};

export type VaultItemHistoryEntry = {
  eventId: string;
  sequence: number;
  occurredAt: string;
  action: VaultEventAction;
  title: string;
  kind: VaultItemKind | "deleted";
  summary: string;
};

export type VaultDeletedItem = {
  id: string;
  title: string;
  kind: VaultItemKind;
  summary: string;
  deletedAt: string;
  deletedEventId: string;
  lastActiveEventId: string | null;
};

export type VaultItemDetail = {
  id: string;
  status: "active" | "deleted" | "missing";
  title: string;
  kind: VaultItemKind | "deleted";
  summary: string;
  tags: string[];
  matchedFields: string[];
  history: VaultItemHistoryEntry[];
  canRestore: boolean;
};

export type VaultLoginCredentialEntry = {
  id: string;
  title: string;
  username: string;
  primaryURL: string | null;
  tags: string[];
  summary: string;
  passwordStatus: "ready" | "locked" | "missing";
  password: string | null;
  secretRef: string | null;
  relatedAuthenticatorId: string | null;
  relatedAuthenticatorTitle: string | null;
  relatedAuthenticatorStatus: AuthenticatorCard["status"] | null;
  relatedAuthenticatorCode: string | null;
};

export type VaultExportPayload = {
  accountId: string;
  collectionId: string;
  latestSeq: number;
  item?: VaultItemRecord;
  items?: VaultItemRecord[];
  secretMaterial?: VaultSecretMaterial;
};

export type VaultImportResult = {
  workspace: VaultWorkspace;
  secretMaterial: VaultSecretMaterial;
  importedItemIds: string[];
};

export type VaultWorkspaceStorage = Pick<
  Storage,
  "getItem" | "setItem" | "removeItem"
>;

export type PersistedVaultRuntimeSnapshot = {
  schemaVersion: 1;
  savedAt: string;
  accountConfig: AccountConfigRecord;
  head: CollectionHeadRecord | null;
  events: VaultEventRecord[];
};

export type PersistedVaultWorkspaceSnapshot = PersistedVaultRuntimeSnapshot;

function normalizeSearchValue(value: string): string {
  return value.trim().toLowerCase();
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null;
}

function countItemsByKind(items: VaultItem[]): Record<VaultItemKind, number> {
  return {
    login: items.filter((item) => item.kind === "login").length,
    note: items.filter((item) => item.kind === "note").length,
    apiKey: items.filter((item) => item.kind === "apiKey").length,
    sshKey: items.filter((item) => item.kind === "sshKey").length,
    totp: items.filter((item) => item.kind === "totp").length,
  };
}

function sortItems(items: VaultItem[]): VaultItem[] {
  return [...items].sort(
    (left, right) =>
      left.title.localeCompare(right.title) || left.id.localeCompare(right.id),
  );
}

function sortEvents(events: VaultEventRecord[]): VaultEventRecord[] {
  return [...events].sort(
    (left, right) =>
      right.sequence - left.sequence ||
      right.eventId.localeCompare(left.eventId),
  );
}

function matchesLogin(totpItem: VaultItem, candidate: VaultItem): boolean {
  if (candidate.kind !== "login" || !isTotpItem(totpItem)) {
    return false;
  }

  if (candidate.username === totpItem.accountName) {
    return true;
  }

  return (
    normalizeSearchValue(candidate.title) ===
    normalizeSearchValue(totpItem.title.replace(/\s+2fa$/i, ""))
  );
}

function findRelatedLogin(totpItem: VaultItem, items: VaultItem[]): VaultItem | null {
  return items.find((item) => matchesLogin(totpItem, item)) ?? null;
}

function findRelatedAuthenticator(
  loginItem: Extract<VaultItem, { kind: "login" }>,
  items: VaultItem[],
): Extract<VaultItem, { kind: "totp" }> | null {
  return (
    items.find(
      (item): item is Extract<VaultItem, { kind: "totp" }> =>
        item.kind === "totp" && matchesLogin(item, loginItem),
    ) ?? null
  );
}

function buildSearchableFields(item: VaultItem): Array<[field: string, value: string]> {
  const fields: Array<[field: string, value: string]> = [
    ["id", item.id],
    ["title", item.title],
    ["tag", item.tags.join(" ")],
  ];

  switch (item.kind) {
    case "login":
      fields.push(["username", item.username], ["url", item.urls.join(" ")]);
      break;
    case "note":
      fields.push(["preview", item.bodyPreview]);
      break;
    case "apiKey":
      fields.push(["service", item.service]);
      break;
    case "sshKey":
      fields.push(["username", item.username], ["host", item.host]);
      break;
    case "totp":
      fields.push(["issuer", item.issuer], ["account", item.accountName]);
      break;
  }

  return fields;
}

function filterItems(items: VaultItem[], query: VaultWorkspaceQuery): VaultItem[] {
  const text = normalizeSearchValue(query.text ?? "");
  const tag = normalizeSearchValue(query.tag ?? "all");

  return sortItems(items).filter((item) => {
    if (query.kind && query.kind !== "all" && item.kind !== query.kind) {
      return false;
    }

    if (
      tag !== "all" &&
      !item.tags.some((itemTag) => normalizeSearchValue(itemTag) === tag)
    ) {
      return false;
    }

    if (text === "") {
      return true;
    }

    return buildSearchableFields(item).some(([, value]) =>
      normalizeSearchValue(value).includes(text),
    );
  });
}

function matchedFieldsForItem(item: VaultItem, query: VaultWorkspaceQuery): string[] {
  const text = normalizeSearchValue(query.text ?? "");
  if (text === "") {
    return [];
  }

  return buildSearchableFields(item)
    .filter(([, value]) => normalizeSearchValue(value).includes(text))
    .map(([field]) => field);
}

function buildListEntries(items: VaultItem[], query: VaultWorkspaceQuery): VaultListEntry[] {
  const limitedItems =
    query.limit && query.limit > 0 ? items.slice(0, query.limit) : items;

  return limitedItems.map((item) => ({
    id: item.id,
    kind: item.kind,
    title: item.title,
    summary: describeVaultItem(item),
    tags: item.tags,
    matchedFields: matchedFieldsForItem(item, query),
  }));
}

function buildAuthenticatorCards(items: VaultItem[]): AuthenticatorCard[] {
  return sortItems(items)
    .filter(isTotpItem)
    .map((item) => {
      const relatedLogin = findRelatedLogin(item, items);

      return {
        id: item.id,
        title: item.title,
        issuer: item.issuer,
        accountName: item.accountName,
        relatedLoginTitle: relatedLogin?.title ?? null,
        relatedLoginURL:
          relatedLogin?.kind === "login" ? relatedLogin.urls[0] ?? null : null,
        summary: describeVaultItem(item),
        code: null,
        secondsRemaining: null,
        validFrom: null,
        validUntil: null,
        status: "locked",
        statusDetail: "Unlock to generate a local code",
      };
    });
}

function applyAuthenticatorSnapshot(
  card: AuthenticatorCard,
  snapshot: TotpCodeSnapshot,
): AuthenticatorCard {
  return {
    ...card,
    code: snapshot.code,
    secondsRemaining: snapshot.secondsRemaining,
    validFrom: snapshot.validFrom,
    validUntil: snapshot.validUntil,
    status: "ready",
    statusDetail: `${snapshot.secondsRemaining}s remaining in the current ${snapshot.periodSeconds}s window`,
  };
}

function buildActivityEntries(events: VaultEventRecord[]): ActivityEntry[] {
  const lastKnownItems = new Map<string, VaultItem>();
  const orderedAscending = [...events].sort(
    (left, right) => left.sequence - right.sequence,
  );

  for (const event of orderedAscending) {
    if (event.action === "put_item") {
      lastKnownItems.set(event.itemRecord.item.id, event.itemRecord.item);
    }
  }

  return sortEvents(events).map((event) => {
    if (event.action === "put_item") {
      return {
        eventId: event.eventId,
        sequence: event.sequence,
        occurredAt: event.occurredAt,
        action: event.action,
        itemId: event.itemRecord.item.id,
        itemTitle: event.itemRecord.item.title,
        itemKind: event.itemRecord.item.kind,
      };
    }

    const deletedItem = lastKnownItems.get(event.itemId);
    return {
      eventId: event.eventId,
      sequence: event.sequence,
      occurredAt: event.occurredAt,
      action: event.action,
      itemId: event.itemId,
      itemTitle: deletedItem?.title ?? event.itemId,
      itemKind: deletedItem?.kind ?? "deleted",
    };
  });
}

function buildAvailableTags(items: VaultItem[]): string[] {
  return [...new Set(items.flatMap((item) => item.tags))].sort((left, right) =>
    left.localeCompare(right),
  );
}

function buildVaultInsights(items: VaultItem[]): VaultInsight[] {
  const logins = sortItems(items).filter(
    (item): item is Extract<VaultItem, { kind: "login" }> => item.kind === "login",
  );
  const totpItems = sortItems(items).filter(
    (item): item is Extract<VaultItem, { kind: "totp" }> => item.kind === "totp",
  );
  const insights: VaultInsight[] = [];

  const loginsWithoutTotp = logins.filter(
    (login) => !totpItems.some((totpItem) => matchesLogin(totpItem, login)),
  );
  if (loginsWithoutTotp.length > 0) {
    insights.push({
      id: "logins-missing-totp",
      title: "Logins Missing 2FA Coverage",
      detail: `${loginsWithoutTotp.length} login${loginsWithoutTotp.length === 1 ? "" : "s"} do not have a linked built-in authenticator`,
      severity: "warning",
      itemIds: loginsWithoutTotp.map((item) => item.id),
    });
  }

  const orphanAuthenticators = totpItems.filter(
    (totpItem) => !logins.some((login) => matchesLogin(totpItem, login)),
  );
  if (orphanAuthenticators.length > 0) {
    insights.push({
      id: "orphan-authenticators",
      title: "Authenticators Without Linked Logins",
      detail: `${orphanAuthenticators.length} authenticator${orphanAuthenticators.length === 1 ? "" : "s"} cannot be matched back to an active login`,
      severity: "info",
      itemIds: orphanAuthenticators.map((item) => item.id),
    });
  }

  const loginGroups = new Map<string, string[]>();
  for (const login of logins) {
    const groupKey = normalizeSearchValue(
      `${login.username}|${login.urls[0] ?? login.title}`,
    );
    loginGroups.set(groupKey, [...(loginGroups.get(groupKey) ?? []), login.id]);
  }
  for (const itemIds of loginGroups.values()) {
    if (itemIds.length > 1) {
      insights.push({
        id: `duplicate-logins-${itemIds[0]}`,
        title: "Duplicate Login Candidates",
        detail: `${itemIds.length} login records appear to share the same username and primary URL`,
        severity: "info",
        itemIds,
      });
    }
  }

  const apiKeyGroups = new Map<string, string[]>();
  for (const item of items) {
    if (item.kind !== "apiKey") {
      continue;
    }

    const groupKey = normalizeSearchValue(item.service);
    apiKeyGroups.set(groupKey, [...(apiKeyGroups.get(groupKey) ?? []), item.id]);
  }
  for (const itemIds of apiKeyGroups.values()) {
    if (itemIds.length > 1) {
      insights.push({
        id: `duplicate-api-keys-${itemIds[0]}`,
        title: "Multiple API Keys For One Service",
        detail: `${itemIds.length} API-key records target the same service`,
        severity: "info",
        itemIds,
      });
    }
  }

  return insights;
}

function eventTargetsItem(event: VaultEventRecord, itemId: string): boolean {
  return event.action === "put_item"
    ? event.itemRecord.item.id === itemId
    : event.itemId === itemId;
}

function findLatestItemRecord(
  events: VaultEventRecord[],
  itemId: string,
): VaultItemRecord | null {
  const ordered = [...events].sort((left, right) => right.sequence - left.sequence);

  for (const event of ordered) {
    if (event.action === "put_item" && event.itemRecord.item.id === itemId) {
      return event.itemRecord;
    }
  }

  return null;
}

function buildItemHistoryEntries(
  events: VaultEventRecord[],
  itemId: string,
): VaultItemHistoryEntry[] {
  const matches = events
    .filter((event) => eventTargetsItem(event, itemId))
    .sort((left, right) => right.sequence - left.sequence);

  return matches.map((event) => {
    if (event.action === "put_item") {
      return {
        eventId: event.eventId,
        sequence: event.sequence,
        occurredAt: event.occurredAt,
        action: event.action,
        title: event.itemRecord.item.title,
        kind: event.itemRecord.item.kind,
        summary: describeVaultItem(event.itemRecord.item),
      };
    }

    const latestRecord = findLatestItemRecord(events, itemId);
    return {
      eventId: event.eventId,
      sequence: event.sequence,
      occurredAt: event.occurredAt,
      action: event.action,
      title: latestRecord?.item.title ?? itemId,
      kind: latestRecord?.item.kind ?? "deleted",
      summary: latestRecord
        ? `${describeVaultItem(latestRecord.item)} deleted`
        : `${itemId} deleted`,
    };
  });
}

function normalizeSecretBase32(secretBase32: string): string {
  const normalized = secretBase32
    .toUpperCase()
    .replaceAll("=", "")
    .replace(/\s+/g, "");

  if (normalized === "") {
    throw new Error("invalid totp secret: empty secret");
  }

  if (!/^[A-Z2-7]+$/.test(normalized)) {
    throw new Error("invalid totp secret: invalid base32 input");
  }

  return normalized;
}

function slugify(value: string): string {
  const slug = value
    .trim()
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-+|-+$/g, "");

  if (slug === "") {
    throw new Error("invalid item slug: empty value");
  }

  return slug;
}

function parsePersistedVaultRuntimeSnapshot(
  value: unknown,
): PersistedVaultRuntimeSnapshot {
  if (!isRecord(value)) {
    throw new Error("invalid persisted vault runtime snapshot");
  }

  if (value.schemaVersion !== 1) {
    throw new Error("invalid persisted vault runtime snapshot field: schemaVersion");
  }

  if (typeof value.savedAt !== "string" || value.savedAt === "") {
    throw new Error("invalid persisted vault runtime snapshot field: savedAt");
  }

  const head =
    value.head === null || value.head === undefined
      ? null
      : parseCollectionHeadRecord(value.head);
  const events = parseVaultEventRecords(value.events);
  if (head === null && events.length > 0) {
    throw new Error("invalid persisted vault runtime snapshot: head required when events exist");
  }

  return {
    schemaVersion: 1,
    savedAt: value.savedAt,
    accountConfig: parseAccountConfigRecord(value.accountConfig),
    head,
    events,
  };
}

function parsePersistedVaultWorkspaceSnapshot(
  value: unknown,
): PersistedVaultWorkspaceSnapshot {
  return parsePersistedVaultRuntimeSnapshot(value);
}

function createEmptyVaultWorkspace(input: {
  accountConfig: AccountConfigRecord;
  query?: VaultWorkspaceQuery;
  starterRecords?: VaultItemRecord[];
}): VaultWorkspace {
  const query = input.query ?? {};

  return {
    accountConfig: input.accountConfig,
    head: null,
    events: [],
    query,
    overview: {
      accountId: input.accountConfig.accountId,
      defaultCollectionId: input.accountConfig.defaultCollectionId,
      collectionCount: input.accountConfig.collectionIds.length,
      deviceCount: input.accountConfig.deviceIds.length,
      itemCount: 0,
      itemCountByKind: {
        login: 0,
        note: 0,
        apiKey: 0,
        sshKey: 0,
        totp: 0,
      },
      latestSeq: 0,
      latestEventId: "",
      lastUpdatedAt: "",
    },
    spotlight: [
      {
        id: "vault-items",
        title: "Vault Items",
        value: "0",
        detail: "No vault items have been saved to this local runtime yet",
      },
      {
        id: "authenticator-ready",
        title: "Authenticator Ready",
        value: "0",
        detail: "No built-in authenticators configured yet",
      },
      {
        id: "connected-devices",
        title: "Connected Devices",
        value: String(input.accountConfig.deviceIds.length),
        detail: `${input.accountConfig.collectionIds.length} synced collection${input.accountConfig.collectionIds.length === 1 ? "" : "s"}`,
      },
    ],
    insights: [],
    items: [],
    authenticators: [],
    activity: [],
    availableKinds: ["all", "login", "note", "apiKey", "sshKey", "totp"],
    availableTags: [],
    itemRecords: [],
    starterRecords: input.starterRecords ?? [],
  };
}

export function createVaultWorkspace(input: {
  accountConfig: AccountConfigRecord;
  head: CollectionHeadRecord | null;
  events: VaultEventRecord[];
  starterRecords?: VaultItemRecord[];
  query?: VaultWorkspaceQuery;
}): VaultWorkspace {
  const query = input.query ?? {};
  if (input.head === null) {
    if (input.events.length > 0) {
      throw new Error("vault runtime head required when events are present");
    }

    return createEmptyVaultWorkspace({
      accountConfig: input.accountConfig,
      query,
      starterRecords: input.starterRecords,
    });
  }

  if (input.accountConfig.defaultCollectionId !== input.head.collectionId) {
    throw new Error(
      `default collection mismatch: expected ${input.accountConfig.defaultCollectionId} got ${input.head.collectionId}`,
    );
  }

  const projection = replayCollectionAgainstHead(input.events, input.head);
  const items = [...projection.items.values()].map((record) => record.item);
  const counts = countItemsByKind(items);
  const filteredItems = filterItems(items, query);
  const sortedEvents = sortEvents(input.events);

  return {
    accountConfig: input.accountConfig,
    head: input.head,
    events: input.events,
    query,
    overview: {
      accountId: input.accountConfig.accountId,
      defaultCollectionId: input.accountConfig.defaultCollectionId,
      collectionCount: input.accountConfig.collectionIds.length,
      deviceCount: input.accountConfig.deviceIds.length,
      itemCount: items.length,
      itemCountByKind: counts,
      latestSeq: projection.latestSeq,
      latestEventId: projection.latestEventId,
      lastUpdatedAt: sortedEvents[0]?.occurredAt ?? "",
    },
    spotlight: [
      {
        id: "vault-items",
        title: "Vault Items",
        value: String(items.length),
        detail: `${counts.login} logins, ${counts.totp} authenticators, ${items.length - counts.login - counts.totp} other items`,
      },
      {
        id: "authenticator-ready",
        title: "Authenticator Ready",
        value: String(counts.totp),
        detail:
          counts.totp === 0
            ? "No built-in authenticators configured yet"
            : `${counts.totp} local OTP/TOTP entries available for quick codes`,
      },
      {
        id: "connected-devices",
        title: "Connected Devices",
        value: String(input.accountConfig.deviceIds.length),
        detail: `${input.accountConfig.collectionIds.length} synced collection${input.accountConfig.collectionIds.length === 1 ? "" : "s"}`,
      },
    ],
    insights: buildVaultInsights(items),
    items: buildListEntries(filteredItems, query),
    authenticators: buildAuthenticatorCards(items),
    activity: buildActivityEntries(input.events),
    availableKinds: ["all", "login", "note", "apiKey", "sshKey", "totp"],
    availableTags: buildAvailableTags(items),
    itemRecords: [...projection.items.values()],
    starterRecords: input.starterRecords ?? [],
  };
}

export async function unlockVaultWorkspace(input: {
  workspace: VaultWorkspace;
  secretMaterial: VaultSecretMaterial;
  at?: Date;
}): Promise<VaultWorkspace> {
  const at = input.at ?? new Date();
  const authenticators = await Promise.all(
    input.workspace.authenticators.map(async (card) => {
      const record = input.workspace.itemRecords.find(
        (itemRecord) => itemRecord.item.id === card.id && itemRecord.item.kind === "totp",
      );
      const secretRef =
        record?.item.kind === "totp" ? record.item.secretRef : null;

      if (!secretRef) {
        return {
          ...card,
          status: "error" as const,
          statusDetail: "Authenticator record is missing a secret reference",
        };
      }

      const secret = input.secretMaterial[secretRef];
      if (!secret) {
        return card;
      }

      try {
        const item =
          record?.item.kind === "totp" ? record.item : null;
        if (!item) {
          return {
            ...card,
            status: "error" as const,
            statusDetail: "Authenticator record is unavailable in the current projection",
          };
        }

        return applyAuthenticatorSnapshot(
          card,
          await generateTotpCodeForItem(item, secret, at),
        );
      } catch (error) {
        return {
          ...card,
          status: "error" as const,
          statusDetail:
            error instanceof Error ? error.message : "Failed to generate local code",
        };
      }
    }),
  );

  return {
    ...input.workspace,
    authenticators,
  };
}

async function rebuildWorkspace(input: {
  workspace: VaultWorkspace;
  head: CollectionHeadRecord | null;
  events: VaultEventRecord[];
  secretMaterial: VaultSecretMaterial;
  at?: Date;
}): Promise<VaultWorkspace> {
  return createUnlockedVaultWorkspace({
    accountConfig: input.workspace.accountConfig,
    head: input.head,
    events: input.events,
    starterRecords: input.workspace.starterRecords,
    query: input.workspace.query,
    secretMaterial: input.secretMaterial,
    at: input.at,
  });
}

async function persistUpdatedItem(input: {
  workspace: VaultWorkspace;
  secretMaterial: VaultSecretMaterial;
  deviceId: string;
  itemRecord: VaultItemRecord;
  at?: Date;
}): Promise<VaultWorkspaceUpdate> {
  const mutation =
    input.workspace.head === null
      ? buildInitialPutItemMutation({
          accountId: input.workspace.accountConfig.accountId,
          collectionId: input.workspace.accountConfig.defaultCollectionId,
          deviceId: input.deviceId,
          itemRecord: input.itemRecord,
          occurredAt: (input.at ?? new Date()).toISOString(),
        })
      : buildPutItemMutation(
          input.workspace.head,
          input.deviceId,
          input.itemRecord,
          (input.at ?? new Date()).toISOString(),
        );
  const events = [...input.workspace.events, mutation.event];

  return {
    workspace: await rebuildWorkspace({
      workspace: input.workspace,
      head: mutation.newHead,
      events,
      secretMaterial: input.secretMaterial,
      at: input.at,
    }),
    secretMaterial: input.secretMaterial,
    itemId: input.itemRecord.item.id,
  };
}

function sortItemRecords(records: VaultItemRecord[]): VaultItemRecord[] {
  return [...records].sort((left, right) =>
    left.item.id.localeCompare(right.item.id),
  );
}

function getVaultItemSecretRef(item: VaultItem): string | null {
  if (item.kind === "totp") {
    return item.secretRef;
  }

  if (item.kind === "login" && item.secretRef) {
    return item.secretRef;
  }

  return null;
}

function collectExportSecretMaterial(
  records: VaultItemRecord[],
  secretMaterial: VaultSecretMaterial,
): VaultSecretMaterial {
  const exportedSecrets: VaultSecretMaterial = {};

  for (const record of records) {
    const secretRef = getVaultItemSecretRef(record.item);
    if (!secretRef) {
      continue;
    }

    const secret = secretMaterial[secretRef];
    if (secret) {
      exportedSecrets[secretRef] = secret;
    }
  }

  return exportedSecrets;
}

export function revealLoginPassword(input: {
  workspace: VaultWorkspace;
  secretMaterial: VaultSecretMaterial;
  itemId: string;
}): string {
  const record = input.workspace.itemRecords.find(
    (itemRecord) => itemRecord.item.id === input.itemId,
  );
  if (!record) {
    throw new Error(`vault item not found: ${input.itemId}`);
  }
  if (record.item.kind !== "login") {
    throw new Error(
      `vault login password reveal only supports login items: ${input.itemId}`,
    );
  }
  if (!record.item.secretRef) {
    throw new Error(`vault login password not configured: ${input.itemId}`);
  }

  const password = input.secretMaterial[record.item.secretRef];
  if (!password) {
    throw new Error(`vault login password not found: ${record.item.secretRef}`);
  }

  return password;
}

function getLoginPasswordState(input: {
  item: Extract<VaultItem, { kind: "login" }>;
  secretMaterial?: VaultSecretMaterial;
}): Pick<VaultLoginCredentialEntry, "passwordStatus" | "password" | "secretRef"> {
  if (!input.item.secretRef) {
    return {
      passwordStatus: "missing",
      password: null,
      secretRef: null,
    };
  }

  if (!input.secretMaterial) {
    return {
      passwordStatus: "locked",
      password: null,
      secretRef: input.item.secretRef,
    };
  }

  const password = input.secretMaterial[input.item.secretRef];
  if (!password) {
    return {
      passwordStatus: "locked",
      password: null,
      secretRef: input.item.secretRef,
    };
  }

  return {
    passwordStatus: "ready",
    password,
    secretRef: input.item.secretRef,
  };
}

export function getVaultLoginCredentialDetail(input: {
  workspace: VaultWorkspace;
  itemId: string;
  secretMaterial?: VaultSecretMaterial;
}): VaultLoginCredentialEntry {
  const record = input.workspace.itemRecords.find(
    (itemRecord) => itemRecord.item.id === input.itemId,
  );
  if (!record) {
    throw new Error(`vault item not found: ${input.itemId}`);
  }
  if (record.item.kind !== "login") {
    throw new Error(
      `vault login credential detail only supports login items: ${input.itemId}`,
    );
  }

  const passwordState = getLoginPasswordState({
    item: record.item,
    secretMaterial: input.secretMaterial,
  });
  const items = input.workspace.itemRecords.map((itemRecord) => itemRecord.item);
  const relatedAuthenticator = findRelatedAuthenticator(record.item, items);
  const relatedAuthenticatorCard = relatedAuthenticator
    ? input.workspace.authenticators.find((card) => card.id === relatedAuthenticator.id) ?? null
    : null;

  return {
    id: record.item.id,
    title: record.item.title,
    username: record.item.username,
    primaryURL: record.item.urls[0] ?? null,
    tags: record.item.tags,
    summary: describeVaultItem(record.item),
    passwordStatus: passwordState.passwordStatus,
    password: passwordState.password,
    secretRef: passwordState.secretRef,
    relatedAuthenticatorId: relatedAuthenticatorCard?.id ?? null,
    relatedAuthenticatorTitle: relatedAuthenticatorCard?.title ?? null,
    relatedAuthenticatorStatus: relatedAuthenticatorCard?.status ?? null,
    relatedAuthenticatorCode: relatedAuthenticatorCard?.code ?? null,
  };
}

export function listVaultLoginCredentials(input: {
  workspace: VaultWorkspace;
  secretMaterial?: VaultSecretMaterial;
}): VaultLoginCredentialEntry[] {
  return input.workspace.itemRecords
    .map((itemRecord) => itemRecord.item)
    .filter(
      (item): item is Extract<VaultItem, { kind: "login" }> => item.kind === "login",
    )
    .sort(
      (left, right) =>
        left.title.localeCompare(right.title) || left.id.localeCompare(right.id),
    )
    .map((item) =>
      getVaultLoginCredentialDetail({
        workspace: input.workspace,
        itemId: item.id,
        secretMaterial: input.secretMaterial,
      }),
    );
}

function parseVaultImportRecords(payload: unknown): {
  records: VaultItemRecord[];
  secretMaterial: VaultSecretMaterial;
} {
  if (typeof payload === "string") {
    return parseVaultImportRecords(JSON.parse(payload));
  }

  if (Array.isArray(payload)) {
    return {
      records: payload.map((value) => parseVaultItemRecord(value)),
      secretMaterial: {},
    };
  }

  if (!isRecord(payload)) {
    throw new Error(
      "vault import payload must be a vault item record or vault export JSON",
    );
  }

  const secretMaterial =
    isRecord(payload.secretMaterial)
      ? Object.fromEntries(
          Object.entries(payload.secretMaterial).filter(
            ([, value]) => typeof value === "string",
          ),
        )
      : {};

  if (Array.isArray(payload.items) && payload.items.length > 0) {
    return {
      records: payload.items.map((value) => parseVaultItemRecord(value)),
      secretMaterial,
    };
  }

  if ("item" in payload) {
    return {
      records: [parseVaultItemRecord(payload.item)],
      secretMaterial,
    };
  }

  return {
    records: [parseVaultItemRecord(payload)],
    secretMaterial,
  };
}

export function getVaultItemDetail(
  workspace: VaultWorkspace,
  itemId: string,
): VaultItemDetail {
  const activeRecord = workspace.itemRecords.find(
    (itemRecord) => itemRecord.item.id === itemId,
  );
  const history = buildItemHistoryEntries(workspace.events, itemId);

  if (activeRecord) {
    return {
      id: itemId,
      status: "active",
      title: activeRecord.item.title,
      kind: activeRecord.item.kind,
      summary: describeVaultItem(activeRecord.item),
      tags: activeRecord.item.tags,
      matchedFields: matchedFieldsForItem(activeRecord.item, workspace.query),
      history,
      canRestore: false,
    };
  }

  const latestRecord = findLatestItemRecord(workspace.events, itemId);
  if (latestRecord) {
    return {
      id: itemId,
      status: "deleted",
      title: latestRecord.item.title,
      kind: latestRecord.item.kind,
      summary: describeVaultItem(latestRecord.item),
      tags: latestRecord.item.tags,
      matchedFields: matchedFieldsForItem(latestRecord.item, workspace.query),
      history,
      canRestore: true,
    };
  }

  return {
    id: itemId,
    status: "missing",
    title: itemId,
    kind: "deleted",
    summary: "Vault item not found",
    tags: [],
    matchedFields: [],
    history: [],
    canRestore: false,
  };
}

export function listDeletedVaultItems(
  workspace: VaultWorkspace,
): VaultDeletedItem[] {
  const activeIds = new Set(workspace.itemRecords.map((record) => record.item.id));
  const deletedById = new Map<string, VaultDeletedItem>();

  for (const event of sortEvents(workspace.events)) {
    if (event.action !== "delete_item" || activeIds.has(event.itemId)) {
      continue;
    }

    const latestRecord = findLatestItemRecord(workspace.events, event.itemId);
    if (!latestRecord) {
      continue;
    }

    deletedById.set(event.itemId, {
      id: event.itemId,
      title: latestRecord.item.title,
      kind: latestRecord.item.kind,
      summary: describeVaultItem(latestRecord.item),
      deletedAt: event.occurredAt,
      deletedEventId: event.eventId,
      lastActiveEventId:
        buildItemHistoryEntries(workspace.events, event.itemId).find(
          (entry) => entry.action === "put_item",
        )?.eventId ?? null,
    });
  }

  return [...deletedById.values()].sort((left, right) =>
    right.deletedAt.localeCompare(left.deletedAt),
  );
}

export function exportVaultWorkspace(
  workspace: VaultWorkspace,
  secretMaterial: VaultSecretMaterial,
  itemId?: string,
): VaultExportPayload {
  if (itemId) {
    const record = workspace.itemRecords.find(
      (itemRecord) => itemRecord.item.id === itemId,
    );
    if (!record) {
      throw new Error(`vault item not found: ${itemId}`);
    }

    const exportedSecrets = collectExportSecretMaterial([record], secretMaterial);
    return {
      accountId: workspace.accountConfig.accountId,
      collectionId: workspace.head?.collectionId ?? workspace.accountConfig.defaultCollectionId,
      latestSeq: workspace.head?.latestSeq ?? 0,
      item: record,
      ...(Object.keys(exportedSecrets).length > 0
        ? { secretMaterial: exportedSecrets }
        : {}),
    };
  }

  const records = sortItemRecords(workspace.itemRecords);
  const exportedSecrets = collectExportSecretMaterial(records, secretMaterial);
  return {
    accountId: workspace.accountConfig.accountId,
    collectionId: workspace.head?.collectionId ?? workspace.accountConfig.defaultCollectionId,
    latestSeq: workspace.head?.latestSeq ?? 0,
    items: records,
    ...(Object.keys(exportedSecrets).length > 0
      ? { secretMaterial: exportedSecrets }
      : {}),
  };
}

export function serializeVaultExportPayload(payload: VaultExportPayload): string {
  return JSON.stringify(payload, null, 2);
}

export async function addLoginToVaultWorkspace(input: {
  workspace: VaultWorkspace;
  secretMaterial: VaultSecretMaterial;
  deviceId: string;
  title: string;
  username: string;
  url: string;
  password?: string;
  tags?: string[];
  at?: Date;
}): Promise<VaultWorkspaceUpdate> {
  const title = input.title.trim();
  const username = input.username.trim();
  const url = input.url.trim();
  if (title === "") {
    throw new Error("invalid login item: title");
  }
  if (username === "") {
    throw new Error("invalid login item: username");
  }
  if (url === "") {
    throw new Error("invalid login item: url");
  }

  const itemId = `login-${slugify(title)}-primary`;
  const secretRef = input.password
    ? `vault-secret://login/${slugify(title)}-primary`
    : undefined;
  const secretMaterial =
    input.password && secretRef
      ? {
          ...input.secretMaterial,
          [secretRef]: input.password,
        }
      : input.secretMaterial;
  return persistUpdatedItem({
    workspace: input.workspace,
    secretMaterial,
    deviceId: input.deviceId,
    itemRecord: createVaultItemRecord({
      id: itemId,
      kind: "login",
      title,
      tags: input.tags ?? ["manual"],
      username,
      urls: [url],
      ...(secretRef ? { secretRef } : {}),
    }),
    at: input.at,
  });
}

export async function addTotpToVaultWorkspace(input: {
  workspace: VaultWorkspace;
  secretMaterial: VaultSecretMaterial;
  deviceId: string;
  title: string;
  issuer: string;
  accountName: string;
  secretBase32: string;
  tags?: string[];
  at?: Date;
}): Promise<VaultWorkspaceUpdate> {
  const title = input.title.trim();
  const issuer = input.issuer.trim();
  const accountName = input.accountName.trim();
  if (title === "") {
    throw new Error("invalid totp item: title");
  }
  if (issuer === "") {
    throw new Error("invalid totp item: issuer");
  }
  if (accountName === "") {
    throw new Error("invalid totp item: accountName");
  }

  const slug = slugify(issuer);
  const itemId = `totp-${slug}-primary`;
  const secretRef = `vault-secret://totp/${slug}-primary`;
  const secretMaterial = {
    ...input.secretMaterial,
    [secretRef]: normalizeSecretBase32(input.secretBase32),
  };
  return persistUpdatedItem({
    workspace: input.workspace,
    secretMaterial,
    deviceId: input.deviceId,
    itemRecord: createVaultItemRecord(
      createTotpItem({
        id: itemId,
        title,
        issuer,
        accountName,
        secretRef,
        tags: input.tags ?? ["2fa", "authenticator"],
      }),
    ),
    at: input.at,
  });
}

export async function addNoteToVaultWorkspace(input: {
  workspace: VaultWorkspace;
  secretMaterial: VaultSecretMaterial;
  deviceId: string;
  title: string;
  bodyPreview: string;
  tags?: string[];
  at?: Date;
}): Promise<VaultWorkspaceUpdate> {
  const title = input.title.trim();
  const bodyPreview = input.bodyPreview.trim();
  if (title === "") {
    throw new Error("invalid note item: title");
  }
  if (bodyPreview === "") {
    throw new Error("invalid note item: bodyPreview");
  }

  const itemId = `note-${slugify(title)}-primary`;
  return persistUpdatedItem({
    workspace: input.workspace,
    secretMaterial: input.secretMaterial,
    deviceId: input.deviceId,
    itemRecord: createVaultItemRecord({
      id: itemId,
      kind: "note",
      title,
      tags: input.tags ?? ["note"],
      bodyPreview,
    }),
    at: input.at,
  });
}

export async function addApiKeyToVaultWorkspace(input: {
  workspace: VaultWorkspace;
  secretMaterial: VaultSecretMaterial;
  deviceId: string;
  title: string;
  service: string;
  tags?: string[];
  at?: Date;
}): Promise<VaultWorkspaceUpdate> {
  const title = input.title.trim();
  const service = input.service.trim();
  if (title === "") {
    throw new Error("invalid api key item: title");
  }
  if (service === "") {
    throw new Error("invalid api key item: service");
  }

  const itemId = `api-key-${slugify(title)}-primary`;
  return persistUpdatedItem({
    workspace: input.workspace,
    secretMaterial: input.secretMaterial,
    deviceId: input.deviceId,
    itemRecord: createVaultItemRecord({
      id: itemId,
      kind: "apiKey",
      title,
      tags: input.tags ?? ["api", "key"],
      service,
    }),
    at: input.at,
  });
}

export async function addSshKeyToVaultWorkspace(input: {
  workspace: VaultWorkspace;
  secretMaterial: VaultSecretMaterial;
  deviceId: string;
  title: string;
  username: string;
  host: string;
  tags?: string[];
  at?: Date;
}): Promise<VaultWorkspaceUpdate> {
  const title = input.title.trim();
  const username = input.username.trim();
  const host = input.host.trim();
  if (title === "") {
    throw new Error("invalid ssh key item: title");
  }
  if (username === "") {
    throw new Error("invalid ssh key item: username");
  }
  if (host === "") {
    throw new Error("invalid ssh key item: host");
  }

  const itemId = `ssh-key-${slugify(title)}-primary`;
  return persistUpdatedItem({
    workspace: input.workspace,
    secretMaterial: input.secretMaterial,
    deviceId: input.deviceId,
    itemRecord: createVaultItemRecord({
      id: itemId,
      kind: "sshKey",
      title,
      tags: input.tags ?? ["ssh"],
      username,
      host,
    }),
    at: input.at,
  });
}

export async function deleteItemFromVaultWorkspace(input: {
  workspace: VaultWorkspace;
  secretMaterial: VaultSecretMaterial;
  deviceId: string;
  itemId: string;
  at?: Date;
}): Promise<VaultWorkspaceUpdate> {
  const record = input.workspace.itemRecords.find(
    (itemRecord) => itemRecord.item.id === input.itemId,
  );
  if (!record) {
    throw new Error(`vault item not found: ${input.itemId}`);
  }

  const mutation = buildDeleteItemMutation(
    input.workspace.head!,
    input.deviceId,
    input.itemId,
    (input.at ?? new Date()).toISOString(),
  );
  const events = [...input.workspace.events, mutation.event];
  const secretRef = getVaultItemSecretRef(record.item);
  const secretMaterial = secretRef
    ? Object.fromEntries(
        Object.entries(input.secretMaterial).filter(
          ([candidateSecretRef]) => candidateSecretRef !== secretRef,
        ),
      )
    : input.secretMaterial;

  return {
    workspace: await rebuildWorkspace({
      workspace: input.workspace,
      head: mutation.newHead,
      events,
      secretMaterial,
      at: input.at,
    }),
    secretMaterial,
    itemId: input.itemId,
  };
}

export async function updateLoginInVaultWorkspace(input: {
  workspace: VaultWorkspace;
  secretMaterial: VaultSecretMaterial;
  deviceId: string;
  itemId: string;
  title: string;
  username: string;
  url?: string;
  password?: string;
  tags?: string[];
  at?: Date;
}): Promise<VaultWorkspaceUpdate> {
  const record = input.workspace.itemRecords.find(
    (itemRecord) => itemRecord.item.id === input.itemId,
  );
  if (!record) {
    throw new Error(`vault item not found: ${input.itemId}`);
  }
  if (record.item.kind !== "login") {
    throw new Error(`vault login update only supports login items: ${input.itemId}`);
  }

  const title = input.title.trim();
  const username = input.username.trim();
  const url = (input.url ?? record.item.urls[0] ?? "").trim();
  if (title === "") {
    throw new Error("invalid login item: title");
  }
  if (username === "") {
    throw new Error("invalid login item: username");
  }
  if (url === "") {
    throw new Error("invalid login item: url");
  }

  const secretMaterial =
    input.password === undefined
      ? input.secretMaterial
      : {
          ...input.secretMaterial,
          [(record.item.secretRef ??
            `vault-secret://login/${slugify(record.item.title)}-primary`)]: input.password,
        };
  const secretRef =
    input.password === undefined
      ? record.item.secretRef
      : (record.item.secretRef ??
          `vault-secret://login/${slugify(record.item.title)}-primary`);

  return persistUpdatedItem({
    workspace: input.workspace,
    secretMaterial,
    deviceId: input.deviceId,
    itemRecord: createVaultItemRecord({
      ...record.item,
      title,
      username,
      urls: [url],
      tags: input.tags ?? record.item.tags,
      ...(secretRef ? { secretRef } : {}),
    }),
    at: input.at,
  });
}

export async function updateTotpInVaultWorkspace(input: {
  workspace: VaultWorkspace;
  secretMaterial: VaultSecretMaterial;
  deviceId: string;
  itemId: string;
  title: string;
  issuer: string;
  accountName: string;
  secretBase32?: string;
  tags?: string[];
  at?: Date;
}): Promise<VaultWorkspaceUpdate> {
  const record = input.workspace.itemRecords.find(
    (itemRecord) => itemRecord.item.id === input.itemId,
  );
  if (!record) {
    throw new Error(`vault item not found: ${input.itemId}`);
  }
  if (record.item.kind !== "totp") {
    throw new Error(`vault totp update only supports totp items: ${input.itemId}`);
  }

  const title = input.title.trim();
  const issuer = input.issuer.trim();
  const accountName = input.accountName.trim();
  if (title === "") {
    throw new Error("invalid totp item: title");
  }
  if (issuer === "") {
    throw new Error("invalid totp item: issuer");
  }
  if (accountName === "") {
    throw new Error("invalid totp item: accountName");
  }

  const secretMaterial =
    input.secretBase32 === undefined
      ? input.secretMaterial
      : {
          ...input.secretMaterial,
          [record.item.secretRef]: normalizeSecretBase32(input.secretBase32),
        };

  return persistUpdatedItem({
    workspace: input.workspace,
    secretMaterial,
    deviceId: input.deviceId,
    itemRecord: createVaultItemRecord({
      ...record.item,
      title,
      issuer,
      accountName,
      tags: input.tags ?? record.item.tags,
    }),
    at: input.at,
  });
}

export async function updateNoteInVaultWorkspace(input: {
  workspace: VaultWorkspace;
  secretMaterial: VaultSecretMaterial;
  deviceId: string;
  itemId: string;
  title: string;
  bodyPreview: string;
  tags?: string[];
  at?: Date;
}): Promise<VaultWorkspaceUpdate> {
  const record = input.workspace.itemRecords.find(
    (itemRecord) => itemRecord.item.id === input.itemId,
  );
  if (!record) {
    throw new Error(`vault item not found: ${input.itemId}`);
  }
  if (record.item.kind !== "note") {
    throw new Error(`vault note update only supports note items: ${input.itemId}`);
  }

  const title = input.title.trim();
  const bodyPreview = input.bodyPreview.trim();
  if (title === "") {
    throw new Error("invalid note item: title");
  }
  if (bodyPreview === "") {
    throw new Error("invalid note item: bodyPreview");
  }

  return persistUpdatedItem({
    workspace: input.workspace,
    secretMaterial: input.secretMaterial,
    deviceId: input.deviceId,
    itemRecord: createVaultItemRecord({
      ...record.item,
      title,
      bodyPreview,
      tags: input.tags ?? record.item.tags,
    }),
    at: input.at,
  });
}

export async function updateApiKeyInVaultWorkspace(input: {
  workspace: VaultWorkspace;
  secretMaterial: VaultSecretMaterial;
  deviceId: string;
  itemId: string;
  title: string;
  service: string;
  tags?: string[];
  at?: Date;
}): Promise<VaultWorkspaceUpdate> {
  const record = input.workspace.itemRecords.find(
    (itemRecord) => itemRecord.item.id === input.itemId,
  );
  if (!record) {
    throw new Error(`vault item not found: ${input.itemId}`);
  }
  if (record.item.kind !== "apiKey") {
    throw new Error(`vault api key update only supports api key items: ${input.itemId}`);
  }

  const title = input.title.trim();
  const service = input.service.trim();
  if (title === "") {
    throw new Error("invalid api key item: title");
  }
  if (service === "") {
    throw new Error("invalid api key item: service");
  }

  return persistUpdatedItem({
    workspace: input.workspace,
    secretMaterial: input.secretMaterial,
    deviceId: input.deviceId,
    itemRecord: createVaultItemRecord({
      ...record.item,
      title,
      service,
      tags: input.tags ?? record.item.tags,
    }),
    at: input.at,
  });
}

export async function updateSshKeyInVaultWorkspace(input: {
  workspace: VaultWorkspace;
  secretMaterial: VaultSecretMaterial;
  deviceId: string;
  itemId: string;
  title: string;
  username: string;
  host: string;
  tags?: string[];
  at?: Date;
}): Promise<VaultWorkspaceUpdate> {
  const record = input.workspace.itemRecords.find(
    (itemRecord) => itemRecord.item.id === input.itemId,
  );
  if (!record) {
    throw new Error(`vault item not found: ${input.itemId}`);
  }
  if (record.item.kind !== "sshKey") {
    throw new Error(`vault ssh key update only supports ssh key items: ${input.itemId}`);
  }

  const title = input.title.trim();
  const username = input.username.trim();
  const host = input.host.trim();
  if (title === "") {
    throw new Error("invalid ssh key item: title");
  }
  if (username === "") {
    throw new Error("invalid ssh key item: username");
  }
  if (host === "") {
    throw new Error("invalid ssh key item: host");
  }

  return persistUpdatedItem({
    workspace: input.workspace,
    secretMaterial: input.secretMaterial,
    deviceId: input.deviceId,
    itemRecord: createVaultItemRecord({
      ...record.item,
      title,
      username,
      host,
      tags: input.tags ?? record.item.tags,
    }),
    at: input.at,
  });
}

export async function restoreItemToVaultWorkspace(input: {
  workspace: VaultWorkspace;
  secretMaterial: VaultSecretMaterial;
  deviceId: string;
  itemId: string;
  at?: Date;
}): Promise<VaultWorkspaceUpdate> {
  const activeRecord = input.workspace.itemRecords.find(
    (itemRecord) => itemRecord.item.id === input.itemId,
  );
  if (activeRecord) {
    throw new Error(`vault item already active: ${input.itemId}`);
  }

  const latestRecord = findLatestItemRecord(input.workspace.events, input.itemId);
  if (!latestRecord) {
    throw new Error(`vault item version not found: ${input.itemId}`);
  }

  const mutation = buildPutItemMutation(
    input.workspace.head!,
    input.deviceId,
    latestRecord,
    (input.at ?? new Date()).toISOString(),
  );
  const events = [...input.workspace.events, mutation.event];

  return {
    workspace: await rebuildWorkspace({
      workspace: input.workspace,
      head: mutation.newHead,
      events,
      secretMaterial: input.secretMaterial,
      at: input.at,
    }),
    secretMaterial: input.secretMaterial,
    itemId: input.itemId,
  };
}

export async function importVaultWorkspace(input: {
  workspace: VaultWorkspace;
  secretMaterial: VaultSecretMaterial;
  deviceId: string;
  payload: string | unknown;
  at?: Date;
}): Promise<VaultImportResult> {
  const { records, secretMaterial: importedSecretMaterial } =
    parseVaultImportRecords(input.payload);
  if (records.length === 0) {
    throw new Error("vault import payload is empty");
  }

  let head = input.workspace.head;
  const events = [...input.workspace.events];
  const importedItemIds: string[] = [];

  for (const [index, record] of records.entries()) {
    const occurredAt = new Date(
      (input.at ?? new Date()).getTime() + index,
    ).toISOString();
    const mutation =
      head === null
        ? buildInitialPutItemMutation({
            accountId: input.workspace.accountConfig.accountId,
            collectionId: input.workspace.accountConfig.defaultCollectionId,
            deviceId: input.deviceId,
            itemRecord: record,
            occurredAt,
          })
        : buildPutItemMutation(
            head,
            input.deviceId,
            record,
            occurredAt,
          );

    head = mutation.newHead;
    events.push(mutation.event);
    importedItemIds.push(record.item.id);
  }

  const secretMaterial = {
    ...input.secretMaterial,
    ...importedSecretMaterial,
  };

  return {
    workspace: await rebuildWorkspace({
      workspace: input.workspace,
      head,
      events,
      secretMaterial,
      at: input.at,
    }),
    secretMaterial,
    importedItemIds,
  };
}

export async function createUnlockedVaultWorkspace(input: {
  accountConfig: AccountConfigRecord;
  head: CollectionHeadRecord | null;
  events: VaultEventRecord[];
  starterRecords?: VaultItemRecord[];
  query?: VaultWorkspaceQuery;
  secretMaterial: VaultSecretMaterial;
  at?: Date;
}): Promise<VaultWorkspace> {
  return unlockVaultWorkspace({
    workspace: createVaultWorkspace(input),
    secretMaterial: input.secretMaterial,
    at: input.at,
  });
}

export function createPersistedVaultRuntimeSnapshot(input: {
  workspace: VaultWorkspace;
  savedAt?: Date;
}): PersistedVaultRuntimeSnapshot {
  return {
    schemaVersion: 1,
    savedAt: (input.savedAt ?? new Date()).toISOString(),
    accountConfig: input.workspace.accountConfig,
    head: input.workspace.head,
    events: input.workspace.events,
  };
}

export function createPersistedVaultWorkspaceSnapshot(input: {
  workspace: VaultWorkspace;
  savedAt?: Date;
}): PersistedVaultWorkspaceSnapshot {
  return createPersistedVaultRuntimeSnapshot(input);
}

export function serializePersistedVaultRuntimeSnapshot(input: {
  workspace: VaultWorkspace;
  savedAt?: Date;
}): string {
  return JSON.stringify(
    createPersistedVaultRuntimeSnapshot(input),
    null,
    2,
  );
}

export function serializePersistedVaultWorkspaceSnapshot(input: {
  workspace: VaultWorkspace;
  savedAt?: Date;
}): string {
  return serializePersistedVaultRuntimeSnapshot(input);
}

export function persistVaultRuntimeSnapshot(input: {
  storage: VaultWorkspaceStorage;
  storageKey: string;
  workspace: VaultWorkspace;
  savedAt?: Date;
}): PersistedVaultRuntimeSnapshot {
  const snapshot = createPersistedVaultRuntimeSnapshot({
    workspace: input.workspace,
    savedAt: input.savedAt,
  });
  input.storage.setItem(input.storageKey, JSON.stringify(snapshot));
  return snapshot;
}

export function persistVaultWorkspaceSnapshot(input: {
  storage: VaultWorkspaceStorage;
  storageKey: string;
  workspace: VaultWorkspace;
  savedAt?: Date;
}): PersistedVaultWorkspaceSnapshot {
  return persistVaultRuntimeSnapshot(input);
}

export function loadPersistedVaultRuntime(input: {
  storage: VaultWorkspaceStorage;
  storageKey: string;
  query?: VaultWorkspaceQuery;
}): VaultWorkspace | null {
  const rawSnapshot = input.storage.getItem(input.storageKey);
  if (rawSnapshot === null) {
    return null;
  }

  const snapshot = parsePersistedVaultRuntimeSnapshot(JSON.parse(rawSnapshot));
  return createVaultWorkspace({
    accountConfig: snapshot.accountConfig,
    head: snapshot.head,
    events: snapshot.events,
    query: input.query,
  });
}

export async function loadUnlockedPersistedVaultRuntime(input: {
  storage: VaultWorkspaceStorage;
  storageKey: string;
  secretMaterial: VaultSecretMaterial;
  query?: VaultWorkspaceQuery;
  at?: Date;
}): Promise<VaultWorkspace | null> {
  const workspace = loadPersistedVaultRuntime({
    storage: input.storage,
    storageKey: input.storageKey,
    query: input.query,
  });
  if (workspace === null) {
    return null;
  }

  return unlockVaultWorkspace({
    workspace,
    secretMaterial: input.secretMaterial,
    at: input.at,
  });
}

export function loadPersistedVaultWorkspace(input: {
  storage: VaultWorkspaceStorage;
  storageKey: string;
  query?: VaultWorkspaceQuery;
}): VaultWorkspace | null {
  return loadPersistedVaultRuntime(input);
}

export function clearPersistedVaultRuntime(input: {
  storage: VaultWorkspaceStorage;
  storageKey: string;
}): void {
  input.storage.removeItem(input.storageKey);
}

export function clearPersistedVaultWorkspace(input: {
  storage: VaultWorkspaceStorage;
  storageKey: string;
}): void {
  clearPersistedVaultRuntime(input);
}

export const webBootstrap = createVaultWorkspace({
  accountConfig: sampleAccountConfigRecord,
  head: sampleCollectionHeadRecord,
  events: sampleVaultEventRecords,
  starterRecords: sampleVaultItemRecords,
});

export function createUnlockedWebBootstrap(at: Date = new Date()): Promise<VaultWorkspace> {
  return createUnlockedVaultWorkspace({
    accountConfig: sampleAccountConfigRecord,
    head: sampleCollectionHeadRecord,
    events: sampleVaultEventRecords,
    starterRecords: sampleVaultItemRecords,
    secretMaterial: sampleVaultSecretMaterial,
    at,
  });
}

function buildInitialPutItemMutation(input: {
  accountId: string;
  collectionId: string;
  deviceId: string;
  itemRecord: VaultItemRecord;
  occurredAt: string;
}) {
  const eventId = `evt-${input.itemRecord.item.id}-v1`;

  return {
    event: {
      schemaVersion: 1 as const,
      eventId,
      accountId: input.accountId,
      deviceId: input.deviceId,
      collectionId: input.collectionId,
      sequence: 1,
      occurredAt: input.occurredAt,
      action: "put_item" as const,
      itemRecord: input.itemRecord,
    },
    newHead: {
      schemaVersion: 1 as const,
      accountId: input.accountId,
      collectionId: input.collectionId,
      latestEventId: eventId,
      latestSeq: 1,
    },
  };
}
