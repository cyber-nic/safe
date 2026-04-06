import {
  argon2Sync,
  createCipheriv,
  createDecipheriv,
  randomBytes,
} from "node:crypto";
import bip39 from "bip39";
import {
  access,
  mkdir,
  readdir,
  readFile,
  rm,
  writeFile,
} from "node:fs/promises";
import path from "node:path";

import {
  createVaultWorkspace,
  type VaultSecretMaterial,
  type VaultWorkspace,
} from "./index.ts";
import {
  parseAccountConfigRecord,
  parseCollectionHeadRecord,
  parseVaultEventRecord,
  parseVaultItemRecord,
  type AccountConfigRecord,
  type CollectionHeadRecord,
  type VaultEventRecord,
} from "../../../packages/ts-sdk/src/index.ts";

const localUnlockSchemaVersion = 1;
const localUnlockAADPrefix = "safe.local-unlock.v1/";
const localRecoverySchemaVersion = 1;
const localRecoveryAADPrefix = "safe.local-recovery.v1/";
const secretEnvelopeSchemaVersion = 1;
const secretEnvelopeAAD = "safe.secret-material.v1";
const argonMemoryKiB = 64 * 1024;
const argonTimeCost = 3;
const argonParallelism = 4;
const accountKeyBytes = 32;
const recoveryKeyBytes = 32;
const gcmNonceBytes = 12;
const gcmTagBytes = 16;
const defaultCollectionId = "vault-personal";

export type ClientIdentity = {
  accountId: string;
  deviceId: string;
  env: string;
};

export type LocalUnlockRecord = {
  schemaVersion: 1;
  accountId: string;
  kdf: {
    name: "argon2id";
    salt: string;
    memoryKiB: number;
    timeCost: number;
    parallelism: number;
    keyBytes: number;
  };
  wrappedKey: {
    algorithm: "aes-256-gcm";
    nonce: string;
    ciphertext: string;
  };
};

export type LocalRecoveryRecord = {
  schemaVersion: 1;
  accountId: string;
  wrappedKey: {
    algorithm: "aes-256-gcm";
    nonce: string;
    ciphertext: string;
  };
};

type SecretMaterialEnvelope = {
  schemaVersion: 1;
  algorithm: "aes-256-gcm";
  nonce: string;
  ciphertext: string;
};

export type UnlockedLocalVault = {
  workspace: VaultWorkspace;
  secretMaterial: VaultSecretMaterial;
  accountKey: Buffer;
};

export type LocalRuntimeStoreOptions = {
  baseDir: string;
};

export type PreparedFirstUse = {
  recoveryMnemonic: string;
  accountKey: Buffer;
  unlockRecord: LocalUnlockRecord;
  recoveryRecord: LocalRecoveryRecord;
};

export function createLocalRuntimeStore(options: LocalRuntimeStoreOptions) {
  return {
    hasUnlockRecord(accountId: string): Promise<boolean> {
      return fileExists(unlockRecordPath(options.baseDir, accountId));
    },

    prepareFirstUse(
      identity: ClientIdentity,
      password: string,
    ): PreparedFirstUse {
      return createFirstUseVault(identity, password);
    },

    finalizeFirstUse(
      identity: ClientIdentity,
      prepared: PreparedFirstUse,
    ): Promise<UnlockedLocalVault> {
      return persistFirstUseVault(options.baseDir, identity, prepared);
    },

    unlock(
      identity: ClientIdentity,
      password: string,
    ): Promise<{ firstUse: false } & UnlockedLocalVault> {
      return loadUnlockedVault(options.baseDir, identity, password).then((unlocked) => ({
        firstUse: false as const,
        ...unlocked,
      }));
    },

    hasRecoveryRecord(accountId: string): Promise<boolean> {
      return fileExists(recoveryRecordPath(options.baseDir, accountId));
    },

    loadUnlockedWithAccountKey(
      identity: ClientIdentity,
      accountKey: Buffer,
    ): Promise<UnlockedLocalVault> {
      return loadUnlockedVaultWithAccountKey(
        options.baseDir,
        identity,
        accountKey,
      );
    },

    persistUnlockedVault(
      identity: ClientIdentity,
      unlocked: UnlockedLocalVault,
    ): Promise<void> {
      return persistUnlockedVault(options.baseDir, identity, unlocked);
    },
  };
}

function createFirstUseVault(
  identity: ClientIdentity,
  password: string,
): PreparedFirstUse {
  if (password.trim() === "") {
    throw new Error("password is required");
  }

  const { record: unlockRecord, accountKey } = createLocalUnlockRecord(
    identity.accountId,
    password,
  );
  const recoveryKey = randomBytes(recoveryKeyBytes);

  return {
    recoveryMnemonic: bip39.entropyToMnemonic(recoveryKey.toString("hex")),
    accountKey,
    unlockRecord,
    recoveryRecord: createLocalRecoveryRecord(
      identity.accountId,
      recoveryKey,
      accountKey,
    ),
  };
}

async function persistFirstUseVault(
  baseDir: string,
  identity: ClientIdentity,
  prepared: PreparedFirstUse,
): Promise<UnlockedLocalVault> {
  const workspace = createVaultWorkspace({
    accountConfig: createDefaultAccountConfig(identity),
    head: null,
    events: [],
  });
  const unlocked = {
    workspace,
    secretMaterial: {},
    accountKey: prepared.accountKey,
  };

  await writeJSON(unlockRecordPath(baseDir, identity.accountId), prepared.unlockRecord);
  await writeJSON(
    recoveryRecordPath(baseDir, identity.accountId),
    prepared.recoveryRecord,
  );
  await persistUnlockedVault(baseDir, identity, unlocked);

  return unlocked;
}

async function loadUnlockedVault(
  baseDir: string,
  identity: ClientIdentity,
  password: string,
): Promise<UnlockedLocalVault> {
  const record = parseLocalUnlockRecord(
    JSON.parse(await readFile(unlockRecordPath(baseDir, identity.accountId), "utf8")),
  );
  const accountKey = openLocalUnlockRecord(record, password);
  return loadUnlockedVaultWithAccountKey(baseDir, identity, accountKey);
}

async function loadUnlockedVaultWithAccountKey(
  baseDir: string,
  identity: ClientIdentity,
  accountKey: Buffer,
): Promise<UnlockedLocalVault> {
  const accountConfig = parseAccountConfigRecord(
    JSON.parse(await readFile(accountConfigPath(baseDir, identity.accountId), "utf8")),
  );
  const head = await readOptionalJSON(
    collectionHeadPath(
      baseDir,
      identity.accountId,
      accountConfig.defaultCollectionId,
    ),
    parseCollectionHeadRecord,
  );
  const events = await readEvents(
    baseDir,
    identity.accountId,
    accountConfig.defaultCollectionId,
  );
  const workspace = createVaultWorkspace({
    accountConfig,
    head,
    events,
  });

  return {
    workspace,
    secretMaterial: await readSecretMaterial(
      baseDir,
      identity.accountId,
      accountConfig.defaultCollectionId,
      accountKey,
    ),
    accountKey,
  };
}

async function persistUnlockedVault(
  baseDir: string,
  identity: ClientIdentity,
  unlocked: UnlockedLocalVault,
): Promise<void> {
  const accountId = identity.accountId;
  const collectionId =
    unlocked.workspace.accountConfig.defaultCollectionId ?? defaultCollectionId;
  const collectionBaseDir = collectionDir(baseDir, accountId, collectionId);

  await writeJSON(accountConfigPath(baseDir, accountId), unlocked.workspace.accountConfig);
  await mkdir(collectionBaseDir, { recursive: true });

  if (unlocked.workspace.head === null) {
    await rm(collectionHeadPath(baseDir, accountId, collectionId), {
      force: true,
    });
  } else {
    await writeJSON(
      collectionHeadPath(baseDir, accountId, collectionId),
      unlocked.workspace.head,
    );
  }

  await rewriteDirectory(
    itemDir(baseDir, accountId, collectionId),
    unlocked.workspace.itemRecords.map((record) => ({
      name: `${record.item.id}.json`,
      payload: JSON.stringify(record, null, 2),
    })),
  );
  await rewriteDirectory(
    eventDir(baseDir, accountId, collectionId),
    unlocked.workspace.events.map((event) => ({
      name: `${event.eventId}.json`,
      payload: JSON.stringify(event, null, 2),
    })),
  );
  await rewriteDirectory(
    secretDir(baseDir, accountId, collectionId),
    await Promise.all(
      Object.entries(unlocked.secretMaterial).map(async ([secretRef, secret]) => ({
        name: `${encodeBase64Url(secretRef)}.txt`,
        payload: JSON.stringify(
          encryptSecretMaterial(unlocked.accountKey, Buffer.from(secret, "utf8")),
          null,
          2,
        ),
      })),
    ),
  );
}

function createDefaultAccountConfig(identity: ClientIdentity): AccountConfigRecord {
  return {
    schemaVersion: 1,
    accountId: identity.accountId,
    defaultCollectionId,
    collectionIds: [defaultCollectionId],
    deviceIds: [identity.deviceId],
  };
}

function createLocalUnlockRecord(
  accountId: string,
  password: string,
): {
  record: LocalUnlockRecord;
  accountKey: Buffer;
} {
  if (accountId.trim() === "") {
    throw new Error("accountId is required");
  }
  if (password.trim() === "") {
    throw new Error("password is required");
  }

  const salt = randomBytes(16);
  const accountKey = randomBytes(accountKeyBytes);
  const kdf = {
    name: "argon2id" as const,
    salt: encodeBase64Url(salt),
    memoryKiB: argonMemoryKiB,
    timeCost: argonTimeCost,
    parallelism: argonParallelism,
    keyBytes: accountKeyBytes,
  };
  const kek = deriveKey(password, kdf);
  const wrappedKey = encryptBytes(
    kek,
    accountKey,
    Buffer.from(localUnlockAAD(accountId), "utf8"),
  );

  return {
    record: {
      schemaVersion: localUnlockSchemaVersion,
      accountId,
      kdf,
      wrappedKey: {
        algorithm: "aes-256-gcm",
        nonce: encodeBase64Url(wrappedKey.nonce),
        ciphertext: encodeBase64Url(wrappedKey.ciphertext),
      },
    },
    accountKey,
  };
}

function openLocalUnlockRecord(
  record: LocalUnlockRecord,
  password: string,
): Buffer {
  if (password.trim() === "") {
    throw new Error("password is required");
  }

  const kek = deriveKey(password, record.kdf);

  try {
    return decryptBytes(
      kek,
      decodeBase64Url(record.wrappedKey.nonce),
      decodeBase64Url(record.wrappedKey.ciphertext),
      Buffer.from(localUnlockAAD(record.accountId), "utf8"),
    );
  } catch {
    throw new Error("local unlock authentication failed");
  }
}

function createLocalRecoveryRecord(
  accountId: string,
  recoveryKey: Buffer,
  accountKey: Buffer,
): LocalRecoveryRecord {
  if (accountId.trim() === "") {
    throw new Error("accountId is required");
  }
  if (recoveryKey.length !== recoveryKeyBytes) {
    throw new Error("recovery key is invalid");
  }

  const wrappedKey = encryptBytes(
    recoveryKey,
    accountKey,
    Buffer.from(localRecoveryAAD(accountId), "utf8"),
  );

  return {
    schemaVersion: localRecoverySchemaVersion,
    accountId,
    wrappedKey: {
      algorithm: "aes-256-gcm",
      nonce: encodeBase64Url(wrappedKey.nonce),
      ciphertext: encodeBase64Url(wrappedKey.ciphertext),
    },
  };
}

function encryptSecretMaterial(
  accountKey: Buffer,
  plaintext: Buffer,
): SecretMaterialEnvelope {
  const sealed = encryptBytes(
    accountKey,
    plaintext,
    Buffer.from(secretEnvelopeAAD, "utf8"),
  );

  return {
    schemaVersion: secretEnvelopeSchemaVersion,
    algorithm: "aes-256-gcm",
    nonce: encodeBase64Url(sealed.nonce),
    ciphertext: encodeBase64Url(sealed.ciphertext),
  };
}

function decryptSecretMaterial(
  accountKey: Buffer,
  envelope: SecretMaterialEnvelope,
): Buffer {
  try {
    return decryptBytes(
      accountKey,
      decodeBase64Url(envelope.nonce),
      decodeBase64Url(envelope.ciphertext),
      Buffer.from(secretEnvelopeAAD, "utf8"),
    );
  } catch {
    throw new Error("secret material authentication failed");
  }
}

function deriveKey(
  password: string,
  kdf: LocalUnlockRecord["kdf"],
): Buffer {
  return argon2Sync("argon2id", {
    message: Buffer.from(password, "utf8"),
    nonce: decodeBase64Url(kdf.salt),
    parallelism: kdf.parallelism,
    tagLength: kdf.keyBytes,
    memory: kdf.memoryKiB,
    passes: kdf.timeCost,
  });
}

function encryptBytes(
  key: Buffer,
  plaintext: Buffer,
  aad: Buffer,
): {
  nonce: Buffer;
  ciphertext: Buffer;
} {
  const nonce = randomBytes(gcmNonceBytes);
  const cipher = createCipheriv("aes-256-gcm", key, nonce);
  cipher.setAAD(aad);
  const ciphertext = Buffer.concat([
    cipher.update(plaintext),
    cipher.final(),
    cipher.getAuthTag(),
  ]);

  return {
    nonce,
    ciphertext,
  };
}

function decryptBytes(
  key: Buffer,
  nonce: Buffer,
  ciphertext: Buffer,
  aad: Buffer,
): Buffer {
  if (ciphertext.length <= gcmTagBytes) {
    throw new Error("ciphertext is truncated");
  }

  const tag = ciphertext.subarray(ciphertext.length - gcmTagBytes);
  const sealed = ciphertext.subarray(0, ciphertext.length - gcmTagBytes);
  const decipher = createDecipheriv("aes-256-gcm", key, nonce);
  decipher.setAAD(aad);
  decipher.setAuthTag(tag);
  return Buffer.concat([decipher.update(sealed), decipher.final()]);
}

function parseLocalUnlockRecord(value: unknown): LocalUnlockRecord {
  if (!isRecord(value)) {
    throw new Error("invalid local unlock record");
  }
  if (value.schemaVersion !== 1) {
    throw new Error("invalid local unlock record field: schemaVersion");
  }
  if (typeof value.accountId !== "string" || value.accountId === "") {
    throw new Error("invalid local unlock record field: accountId");
  }
  if (!isRecord(value.kdf)) {
    throw new Error("invalid local unlock record field: kdf");
  }
  if (
    value.kdf.name !== "argon2id" ||
    typeof value.kdf.salt !== "string" ||
    typeof value.kdf.memoryKiB !== "number" ||
    typeof value.kdf.timeCost !== "number" ||
    typeof value.kdf.parallelism !== "number" ||
    typeof value.kdf.keyBytes !== "number"
  ) {
    throw new Error("invalid local unlock record field: kdf");
  }
  if (!isRecord(value.wrappedKey)) {
    throw new Error("invalid local unlock record field: wrappedKey");
  }
  if (
    value.wrappedKey.algorithm !== "aes-256-gcm" ||
    typeof value.wrappedKey.nonce !== "string" ||
    typeof value.wrappedKey.ciphertext !== "string"
  ) {
    throw new Error("invalid local unlock record field: wrappedKey");
  }

  return {
    schemaVersion: 1,
    accountId: value.accountId,
    kdf: {
      name: "argon2id",
      salt: value.kdf.salt,
      memoryKiB: value.kdf.memoryKiB,
      timeCost: value.kdf.timeCost,
      parallelism: value.kdf.parallelism,
      keyBytes: value.kdf.keyBytes,
    },
    wrappedKey: {
      algorithm: "aes-256-gcm",
      nonce: value.wrappedKey.nonce,
      ciphertext: value.wrappedKey.ciphertext,
    },
  };
}

function parseLocalRecoveryRecord(value: unknown): LocalRecoveryRecord {
  if (!isRecord(value)) {
    throw new Error("invalid local recovery record");
  }
  if (value.schemaVersion !== 1) {
    throw new Error("invalid local recovery record field: schemaVersion");
  }
  if (typeof value.accountId !== "string" || value.accountId === "") {
    throw new Error("invalid local recovery record field: accountId");
  }
  if (!isRecord(value.wrappedKey)) {
    throw new Error("invalid local recovery record field: wrappedKey");
  }
  if (
    value.wrappedKey.algorithm !== "aes-256-gcm" ||
    typeof value.wrappedKey.nonce !== "string" ||
    typeof value.wrappedKey.ciphertext !== "string"
  ) {
    throw new Error("invalid local recovery record field: wrappedKey");
  }

  return {
    schemaVersion: 1,
    accountId: value.accountId,
    wrappedKey: {
      algorithm: "aes-256-gcm",
      nonce: value.wrappedKey.nonce,
      ciphertext: value.wrappedKey.ciphertext,
    },
  };
}

async function readEvents(
  baseDir: string,
  accountId: string,
  collectionId: string,
): Promise<VaultEventRecord[]> {
  const dir = eventDir(baseDir, accountId, collectionId);
  if (!(await fileExists(dir))) {
    return [];
  }

  const entries = await readdir(dir, { withFileTypes: true });
  const events = await Promise.all(
    entries
      .filter((entry) => entry.isFile() && entry.name.endsWith(".json"))
      .map(async (entry) =>
        parseVaultEventRecord(
          JSON.parse(await readFile(path.join(dir, entry.name), "utf8")),
        ),
      ),
  );

  return events.sort(
    (left, right) =>
      left.sequence - right.sequence ||
      left.eventId.localeCompare(right.eventId),
  );
}

async function readSecretMaterial(
  baseDir: string,
  accountId: string,
  collectionId: string,
  accountKey: Buffer,
): Promise<VaultSecretMaterial> {
  const dir = secretDir(baseDir, accountId, collectionId);
  if (!(await fileExists(dir))) {
    return {};
  }

  const entries = await readdir(dir, { withFileTypes: true });
  const secretPairs = await Promise.all(
    entries
      .filter((entry) => entry.isFile() && entry.name.endsWith(".txt"))
      .map(async (entry) => {
        const secretRef = decodeBase64Url(entry.name.slice(0, -4)).toString("utf8");
        const envelope = JSON.parse(await readFile(path.join(dir, entry.name), "utf8"));
        const plaintext = decryptSecretMaterial(accountKey, envelope);
        return [secretRef, plaintext.toString("utf8")] as const;
      }),
  );

  return Object.fromEntries(secretPairs);
}

async function rewriteDirectory(
  dir: string,
  files: Array<{
    name: string;
    payload: string;
  }>,
): Promise<void> {
  await rm(dir, { recursive: true, force: true });
  await mkdir(dir, { recursive: true });

  await Promise.all(
    files.map((file) =>
      writeFile(path.join(dir, file.name), file.payload, "utf8"),
    ),
  );
}

async function writeJSON(filePath: string, value: unknown): Promise<void> {
  await mkdir(path.dirname(filePath), { recursive: true });
  await writeFile(filePath, JSON.stringify(value, null, 2), "utf8");
}

async function readOptionalJSON<T>(
  filePath: string,
  parse: (value: unknown) => T,
): Promise<T | null> {
  if (!(await fileExists(filePath))) {
    return null;
  }

  return parse(JSON.parse(await readFile(filePath, "utf8")));
}

async function fileExists(filePath: string): Promise<boolean> {
  try {
    await access(filePath);
    return true;
  } catch {
    return false;
  }
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null;
}

function encodeBase64Url(value: Buffer | string): string {
  return Buffer.from(value).toString("base64url");
}

function decodeBase64Url(value: string): Buffer {
  return Buffer.from(value, "base64url");
}

function localUnlockAAD(accountId: string): string {
  return `${localUnlockAADPrefix}${accountId}`;
}

function localRecoveryAAD(accountId: string): string {
  return `${localRecoveryAADPrefix}${accountId}`;
}

function accountBaseDir(baseDir: string, accountId: string): string {
  return path.join(baseDir, "accounts", accountId);
}

function collectionDir(
  baseDir: string,
  accountId: string,
  collectionId: string,
): string {
  return path.join(accountBaseDir(baseDir, accountId), "collections", collectionId);
}

function accountConfigPath(baseDir: string, accountId: string): string {
  return path.join(accountBaseDir(baseDir, accountId), "account.json");
}

function unlockRecordPath(baseDir: string, accountId: string): string {
  return path.join(accountBaseDir(baseDir, accountId), "unlock.json");
}

function recoveryRecordPath(baseDir: string, accountId: string): string {
  return path.join(accountBaseDir(baseDir, accountId), "recovery.json");
}

function collectionHeadPath(
  baseDir: string,
  accountId: string,
  collectionId: string,
): string {
  return path.join(collectionDir(baseDir, accountId, collectionId), "head.json");
}

function itemDir(
  baseDir: string,
  accountId: string,
  collectionId: string,
): string {
  return path.join(collectionDir(baseDir, accountId, collectionId), "items");
}

function eventDir(
  baseDir: string,
  accountId: string,
  collectionId: string,
): string {
  return path.join(collectionDir(baseDir, accountId, collectionId), "events");
}

function secretDir(
  baseDir: string,
  accountId: string,
  collectionId: string,
): string {
  return path.join(collectionDir(baseDir, accountId, collectionId), "secrets");
}
