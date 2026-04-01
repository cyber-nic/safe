export const vaultItemKinds = [
  "login",
  "note",
  "apiKey",
  "sshKey",
  "totp",
] as const;

export type VaultItemKind = (typeof vaultItemKinds)[number];

type VaultItemBase = {
  id: string;
  kind: VaultItemKind;
  title: string;
  tags: string[];
};

export type LoginItem = VaultItemBase & {
  kind: "login";
  username: string;
  urls: string[];
};

export type NoteItem = VaultItemBase & {
  kind: "note";
  bodyPreview: string;
};

export type ApiKeyItem = VaultItemBase & {
  kind: "apiKey";
  service: string;
};

export type SshKeyItem = VaultItemBase & {
  kind: "sshKey";
  username: string;
  host: string;
};

export type TotpItem = VaultItemBase & {
  kind: "totp";
  issuer: string;
  accountName: string;
  digits: 6;
  periodSeconds: 30;
  algorithm: "SHA1";
  secretRef: string;
};

export type VaultItem =
  | LoginItem
  | NoteItem
  | ApiKeyItem
  | SshKeyItem
  | TotpItem;

export type VaultItemRecord = {
  schemaVersion: 1;
  item: VaultItem;
};

export type CollectionHeadRecord = {
  schemaVersion: 1;
  accountId: string;
  collectionId: string;
  latestEventId: string;
  latestSeq: number;
};

export type AccountConfigRecord = {
  schemaVersion: 1;
  accountId: string;
  defaultCollectionId: string;
  collectionIds: string[];
  deviceIds: string[];
};

export type VaultEventAction = "put_item" | "delete_item";

type VaultEventRecordBase = {
  schemaVersion: 1;
  eventId: string;
  accountId: string;
  deviceId: string;
  collectionId: string;
  sequence: number;
  occurredAt: string;
};

export type PutItemEventRecord = VaultEventRecordBase & {
  action: "put_item";
  itemRecord: VaultItemRecord;
};

export type DeleteItemEventRecord = VaultEventRecordBase & {
  action: "delete_item";
  itemId: string;
};

export type VaultEventRecord = PutItemEventRecord | DeleteItemEventRecord;

export type CollectionHeadValidationErrorField =
  | "schemaVersion"
  | "accountId"
  | "collectionId"
  | "latestEventId"
  | "latestSeq";

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null;
}

function expectString(value: unknown, field: string): string {
  if (typeof value !== "string") {
    throw new Error(`invalid vault item field: ${field}`);
  }

  return value;
}

function expectStringArray(value: unknown, field: string): string[] {
  if (!Array.isArray(value) || value.some((item) => typeof item !== "string")) {
    throw new Error(`invalid vault item field: ${field}`);
  }

  return value;
}

export function createTotpItem(input: {
  id: string;
  title: string;
  issuer: string;
  accountName: string;
  secretRef: string;
  tags?: string[];
}): TotpItem {
  return {
    id: input.id,
    kind: "totp",
    title: input.title,
    tags: input.tags ?? ["2fa", "authenticator"],
    issuer: input.issuer,
    accountName: input.accountName,
    digits: 6,
    periodSeconds: 30,
    algorithm: "SHA1",
    secretRef: input.secretRef,
  };
}

export function isTotpItem(item: VaultItem): item is TotpItem {
  return item.kind === "totp";
}

export function createVaultItemRecord(item: VaultItem): VaultItemRecord {
  return {
    schemaVersion: 1,
    item,
  };
}

export function createCollectionHeadRecord(input: {
  accountId: string;
  collectionId: string;
  latestEventId: string;
  latestSeq: number;
}): CollectionHeadRecord {
  return {
    schemaVersion: 1,
    accountId: input.accountId,
    collectionId: input.collectionId,
    latestEventId: input.latestEventId,
    latestSeq: input.latestSeq,
  };
}

export function createAccountConfigRecord(input: {
  accountId: string;
  defaultCollectionId: string;
  collectionIds: string[];
  deviceIds: string[];
}): AccountConfigRecord {
  return {
    schemaVersion: 1,
    accountId: input.accountId,
    defaultCollectionId: input.defaultCollectionId,
    collectionIds: input.collectionIds,
    deviceIds: input.deviceIds,
  };
}

export function createPutItemEventRecord(input: {
  eventId: string;
  accountId: string;
  deviceId: string;
  collectionId: string;
  sequence: number;
  occurredAt: string;
  itemRecord: VaultItemRecord;
}): VaultEventRecord {
  return {
    schemaVersion: 1,
    eventId: input.eventId,
    accountId: input.accountId,
    deviceId: input.deviceId,
    collectionId: input.collectionId,
    sequence: input.sequence,
    occurredAt: input.occurredAt,
    action: "put_item",
    itemRecord: input.itemRecord,
  };
}

export function createDeleteItemEventRecord(input: {
  eventId: string;
  accountId: string;
  deviceId: string;
  collectionId: string;
  sequence: number;
  occurredAt: string;
  itemId: string;
}): DeleteItemEventRecord {
  return {
    schemaVersion: 1,
    eventId: input.eventId,
    accountId: input.accountId,
    deviceId: input.deviceId,
    collectionId: input.collectionId,
    sequence: input.sequence,
    occurredAt: input.occurredAt,
    action: "delete_item",
    itemId: input.itemId,
  };
}

export function parseVaultItem(value: unknown): VaultItem {
  if (!isRecord(value)) {
    throw new Error("invalid vault item");
  }

  const id = expectString(value.id, "id");
  const title = expectString(value.title, "title");
  const tags = expectStringArray(value.tags, "tags");
  const kind = expectString(value.kind, "kind");

  switch (kind) {
    case "login":
      return {
        id,
        kind,
        title,
        tags,
        username: expectString(value.username, "username"),
        urls: expectStringArray(value.urls, "urls"),
      };
    case "note":
      return {
        id,
        kind,
        title,
        tags,
        bodyPreview: expectString(value.bodyPreview, "bodyPreview"),
      };
    case "apiKey":
      return {
        id,
        kind,
        title,
        tags,
        service: expectString(value.service, "service"),
      };
    case "sshKey":
      return {
        id,
        kind,
        title,
        tags,
        username: expectString(value.username, "username"),
        host: expectString(value.host, "host"),
      };
    case "totp":
      if (value.digits !== 6) {
        throw new Error("invalid vault item field: digits");
      }

      if (value.periodSeconds !== 30) {
        throw new Error("invalid vault item field: periodSeconds");
      }

      if (value.algorithm !== "SHA1") {
        throw new Error("invalid vault item field: algorithm");
      }

      return {
        id,
        kind,
        title,
        tags,
        issuer: expectString(value.issuer, "issuer"),
        accountName: expectString(value.accountName, "accountName"),
        digits: 6,
        periodSeconds: 30,
        algorithm: "SHA1",
        secretRef: expectString(value.secretRef, "secretRef"),
      };
    default:
      throw new Error(`unsupported vault item kind: ${kind}`);
  }
}

export function parseVaultItems(values: unknown): VaultItem[] {
  if (!Array.isArray(values)) {
    throw new Error("invalid vault item list");
  }

  return values.map(parseVaultItem);
}

export function parseVaultItemRecord(value: unknown): VaultItemRecord {
  if (!isRecord(value)) {
    throw new Error("invalid vault item record");
  }

  if (value.schemaVersion !== 1) {
    throw new Error("invalid vault item record field: schemaVersion");
  }

  return {
    schemaVersion: 1,
    item: parseVaultItem(value.item),
  };
}

export function parseVaultItemRecords(values: unknown): VaultItemRecord[] {
  if (!Array.isArray(values)) {
    throw new Error("invalid vault item record list");
  }

  return values.map(parseVaultItemRecord);
}

export function parseVaultEventRecord(value: unknown): VaultEventRecord {
  if (!isRecord(value)) {
    throw new Error("invalid vault event record");
  }

  if (value.schemaVersion !== 1) {
    throw new Error("invalid vault event record field: schemaVersion");
  }

  const eventId = expectString(value.eventId, "eventId");
  const accountId = expectString(value.accountId, "accountId");
  const deviceId = expectString(value.deviceId, "deviceId");
  const collectionId = expectString(value.collectionId, "collectionId");
  const occurredAt = expectString(value.occurredAt, "occurredAt");

  if (typeof value.sequence !== "number" || !Number.isInteger(value.sequence) || value.sequence < 1) {
    throw new Error("invalid vault event record field: sequence");
  }

  if (value.action === "put_item") {
    return {
      schemaVersion: 1,
      eventId,
      accountId,
      deviceId,
      collectionId,
      sequence: value.sequence,
      occurredAt,
      action: "put_item",
      itemRecord: parseVaultItemRecord(value.itemRecord),
    };
  }

  if (value.action === "delete_item") {
    return {
      schemaVersion: 1,
      eventId,
      accountId,
      deviceId,
      collectionId,
      sequence: value.sequence,
      occurredAt,
      action: "delete_item",
      itemId: expectString(value.itemId, "itemId"),
    };
  }

  throw new Error("invalid vault event record field: action");
}

export function parseVaultEventRecords(values: unknown): VaultEventRecord[] {
  if (!Array.isArray(values)) {
    throw new Error("invalid vault event record list");
  }

  return values.map(parseVaultEventRecord);
}

export function parseCollectionHeadRecord(value: unknown): CollectionHeadRecord {
  if (!isRecord(value)) {
    throw new Error("invalid collection head record");
  }

  if (value.schemaVersion !== 1) {
    throw new Error("invalid collection head record field: schemaVersion");
  }

  const latestSeq = value.latestSeq;
  if (
    typeof latestSeq !== "number" ||
    !Number.isInteger(latestSeq) ||
    latestSeq < 1
  ) {
    throw new Error("invalid collection head record field: latestSeq");
  }

  return {
    schemaVersion: 1,
    accountId: expectString(value.accountId, "accountId"),
    collectionId: expectString(value.collectionId, "collectionId"),
    latestEventId: expectString(value.latestEventId, "latestEventId"),
    latestSeq,
  };
}

export function parseAccountConfigRecord(value: unknown): AccountConfigRecord {
  if (!isRecord(value)) {
    throw new Error("invalid account config record");
  }

  if (value.schemaVersion !== 1) {
    throw new Error("invalid account config record field: schemaVersion");
  }

  const collectionIds = expectStringArray(value.collectionIds, "collectionIds");
  const deviceIds = expectStringArray(value.deviceIds, "deviceIds");
  if (collectionIds.length === 0) {
    throw new Error("invalid account config record field: collectionIds");
  }
  if (deviceIds.length === 0) {
    throw new Error("invalid account config record field: deviceIds");
  }

  return {
    schemaVersion: 1,
    accountId: expectString(value.accountId, "accountId"),
    defaultCollectionId: expectString(
      value.defaultCollectionId,
      "defaultCollectionId",
    ),
    collectionIds,
    deviceIds,
  };
}

function canonicalVaultItemShape(item: VaultItem): Record<string, unknown> {
  switch (item.kind) {
    case "login":
      return {
        id: item.id,
        kind: item.kind,
        title: item.title,
        tags: item.tags,
        username: item.username,
        urls: item.urls,
      };
    case "note":
      return {
        id: item.id,
        kind: item.kind,
        title: item.title,
        tags: item.tags,
        bodyPreview: item.bodyPreview,
      };
    case "apiKey":
      return {
        id: item.id,
        kind: item.kind,
        title: item.title,
        tags: item.tags,
        service: item.service,
      };
    case "sshKey":
      return {
        id: item.id,
        kind: item.kind,
        title: item.title,
        tags: item.tags,
        username: item.username,
        host: item.host,
      };
    case "totp":
      return {
        id: item.id,
        kind: item.kind,
        title: item.title,
        tags: item.tags,
        issuer: item.issuer,
        accountName: item.accountName,
        digits: item.digits,
        periodSeconds: item.periodSeconds,
        algorithm: item.algorithm,
        secretRef: item.secretRef,
      };
  }
}

export function serializeCanonicalVaultItem(item: VaultItem): string {
  return JSON.stringify(canonicalVaultItemShape(item));
}

export function serializeCanonicalVaultItemRecord(record: VaultItemRecord): string {
  return JSON.stringify({
    schemaVersion: record.schemaVersion,
    item: canonicalVaultItemShape(record.item),
  });
}

export function serializeCanonicalVaultEventRecord(record: VaultEventRecord): string {
  if (record.action === "put_item") {
    return JSON.stringify({
      schemaVersion: record.schemaVersion,
      eventId: record.eventId,
      accountId: record.accountId,
      deviceId: record.deviceId,
      collectionId: record.collectionId,
      sequence: record.sequence,
      occurredAt: record.occurredAt,
      action: record.action,
      itemRecord: {
        schemaVersion: record.itemRecord.schemaVersion,
        item: canonicalVaultItemShape(record.itemRecord.item),
      },
    });
  }

  return JSON.stringify({
    schemaVersion: record.schemaVersion,
    eventId: record.eventId,
    accountId: record.accountId,
    deviceId: record.deviceId,
    collectionId: record.collectionId,
    sequence: record.sequence,
    occurredAt: record.occurredAt,
    action: record.action,
    itemId: record.itemId,
  });
}

export function serializeCanonicalCollectionHeadRecord(
  record: CollectionHeadRecord,
): string {
  return JSON.stringify({
    schemaVersion: record.schemaVersion,
    accountId: record.accountId,
    collectionId: record.collectionId,
    latestEventId: record.latestEventId,
    latestSeq: record.latestSeq,
  });
}

export function serializeCanonicalAccountConfigRecord(
  record: AccountConfigRecord,
): string {
  return JSON.stringify({
    schemaVersion: record.schemaVersion,
    accountId: record.accountId,
    defaultCollectionId: record.defaultCollectionId,
    collectionIds: record.collectionIds,
    deviceIds: record.deviceIds,
  });
}

export function ensureMonotonicHead(
  trusted: CollectionHeadRecord,
  candidate: CollectionHeadRecord,
): void {
  if (trusted.accountId !== candidate.accountId) {
    throw new Error("sync replay invariant violated: trustedHead.accountId");
  }

  if (trusted.collectionId !== candidate.collectionId) {
    throw new Error("sync replay invariant violated: trustedHead.collectionId");
  }

  if (candidate.latestSeq < trusted.latestSeq) {
    throw new Error(
      `sync stale head rejected: trusted ${trusted.latestSeq} candidate ${candidate.latestSeq}`,
    );
  }

  if (
    candidate.latestSeq === trusted.latestSeq &&
    candidate.latestEventId !== trusted.latestEventId
  ) {
    throw new Error(
      `sync head mismatch: latestEventId expected ${trusted.latestEventId} got ${candidate.latestEventId}`,
    );
  }
}

export function describeVaultItem(item: VaultItem): string {
  switch (item.kind) {
    case "login":
      return `${item.title} login for ${item.username}`;
    case "note":
      return `${item.title} secure note`;
    case "apiKey":
      return `${item.title} API key for ${item.service}`;
    case "sshKey":
      return `${item.title} SSH key for ${item.username}@${item.host}`;
    case "totp":
      return `${item.title} authenticator for ${item.issuer} (${item.accountName})`;
  }
}
