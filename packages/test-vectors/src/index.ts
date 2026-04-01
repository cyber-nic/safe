import rawVaultItems from "./vault-items.json" with { type: "json" };
import rawEventRecords from "./event-records.json" with { type: "json" };
import rawVaultItemRecords from "./vault-item-records.json" with { type: "json" };
import rawDeleteEventRecord from "./delete-event-record.json" with { type: "json" };
import rawCollectionHeadRecord from "./collection-head-record.json" with { type: "json" };
import rawAccountConfigRecord from "./account-config-record.json" with { type: "json" };

import {
  parseAccountConfigRecord,
  parseCollectionHeadRecord,
  parseVaultEventRecord,
  parseVaultEventRecords,
  parseVaultItemRecords,
  parseVaultItems,
  serializeCanonicalAccountConfigRecord,
  serializeCanonicalCollectionHeadRecord,
  serializeCanonicalVaultEventRecord,
  serializeCanonicalVaultItemRecord,
  type AccountConfigRecord,
  type CollectionHeadRecord,
  type VaultEventRecord,
  type VaultItem,
  type VaultItemRecord,
} from "../../ts-sdk/src/index.ts";

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
export const sampleDeleteVaultEventRecord: VaultEventRecord =
  parseVaultEventRecord(rawDeleteEventRecord);
export const canonicalDeleteVaultEventRecord =
  serializeCanonicalVaultEventRecord(sampleDeleteVaultEventRecord);
export const sampleCollectionHeadRecord: CollectionHeadRecord =
  parseCollectionHeadRecord(rawCollectionHeadRecord);
export const canonicalCollectionHeadRecord =
  serializeCanonicalCollectionHeadRecord(sampleCollectionHeadRecord);
export const sampleAccountConfigRecord: AccountConfigRecord =
  parseAccountConfigRecord(rawAccountConfigRecord);
export const canonicalAccountConfigRecord =
  serializeCanonicalAccountConfigRecord(sampleAccountConfigRecord);
