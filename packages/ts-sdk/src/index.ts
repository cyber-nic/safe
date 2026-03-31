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
