import {
  buildDeleteItemMutation,
  buildPutItemMutation,
  createTotpItem,
  createVaultItemRecord,
  describeVaultItem,
  generateTotpCodeForItem,
  isTotpItem,
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

export type VaultWorkspace = {
  accountConfig: AccountConfigRecord;
  head: CollectionHeadRecord;
  events: VaultEventRecord[];
  query: VaultWorkspaceQuery;
  overview: VaultOverview;
  spotlight: VaultSpotlightCard[];
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

function normalizeSearchValue(value: string): string {
  return value.trim().toLowerCase();
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

export function createVaultWorkspace(input: {
  accountConfig: AccountConfigRecord;
  head: CollectionHeadRecord;
  events: VaultEventRecord[];
  starterRecords?: VaultItemRecord[];
  query?: VaultWorkspaceQuery;
}): VaultWorkspace {
  const query = input.query ?? {};
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
  head: CollectionHeadRecord;
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
  const mutation = buildPutItemMutation(
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

export async function addLoginToVaultWorkspace(input: {
  workspace: VaultWorkspace;
  secretMaterial: VaultSecretMaterial;
  deviceId: string;
  title: string;
  username: string;
  url: string;
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
  return persistUpdatedItem({
    workspace: input.workspace,
    secretMaterial: input.secretMaterial,
    deviceId: input.deviceId,
    itemRecord: createVaultItemRecord({
      id: itemId,
      kind: "login",
      title,
      tags: input.tags ?? ["manual"],
      username,
      urls: [url],
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
    input.workspace.head,
    input.deviceId,
    input.itemId,
    (input.at ?? new Date()).toISOString(),
  );
  const events = [...input.workspace.events, mutation.event];
  const secretMaterial =
    record.item.kind === "totp"
      ? Object.fromEntries(
          Object.entries(input.secretMaterial).filter(
            ([secretRef]) => secretRef !== record.item.secretRef,
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

  return persistUpdatedItem({
    workspace: input.workspace,
    secretMaterial: input.secretMaterial,
    deviceId: input.deviceId,
    itemRecord: createVaultItemRecord({
      ...record.item,
      title,
      username,
      urls: [url],
      tags: input.tags ?? record.item.tags,
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
    input.workspace.head,
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

export async function createUnlockedVaultWorkspace(input: {
  accountConfig: AccountConfigRecord;
  head: CollectionHeadRecord;
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
