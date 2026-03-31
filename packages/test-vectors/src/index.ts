import rawVaultItems from "./vault-items.json";
import rawEventRecords from "./event-records.json";
import rawVaultItemRecords from "./vault-item-records.json";

import {
  parseVaultEventRecords,
  parseVaultItemRecords,
  parseVaultItems,
  serializeCanonicalVaultEventRecord,
  serializeCanonicalVaultItemRecord,
  type VaultEventRecord,
  type VaultItem,
  type VaultItemRecord,
} from "../../ts-sdk/src/index.js";

export const sampleVaultItems: VaultItem[] = parseVaultItems(rawVaultItems);
export const sampleVaultItemRecords: VaultItemRecord[] =
  parseVaultItemRecords(rawVaultItemRecords);
export const canonicalVaultItemRecords = sampleVaultItemRecords.map(
  serializeCanonicalVaultItemRecord,
);
export const sampleVaultEventRecords: VaultEventRecord[] =
  parseVaultEventRecords(rawEventRecords);
export const canonicalVaultEventRecords = sampleVaultEventRecords.map(
  serializeCanonicalVaultEventRecord,
);
