import rawVaultItems from "./vault-items.json";
import rawVaultItemRecords from "./vault-item-records.json";

import {
  parseVaultItemRecords,
  parseVaultItems,
  serializeCanonicalVaultItemRecord,
  type VaultItem,
  type VaultItemRecord,
} from "../../ts-sdk/src/index.js";

export const sampleVaultItems: VaultItem[] = parseVaultItems(rawVaultItems);
export const sampleVaultItemRecords: VaultItemRecord[] =
  parseVaultItemRecords(rawVaultItemRecords);
export const canonicalVaultItemRecords = sampleVaultItemRecords.map(
  serializeCanonicalVaultItemRecord,
);
