import rawVaultItems from "./vault-items.json" with { type: "json" };
import rawEventRecords from "./event-records.json" with { type: "json" };
import rawVaultItemRecords from "./vault-item-records.json" with { type: "json" };
import rawDeleteEventRecord from "./delete-event-record.json" with { type: "json" };
import rawCollectionHeadRecord from "./collection-head-record.json" with { type: "json" };
import rawAccountConfigRecord from "./account-config-record.json" with { type: "json" };
import rawPutItemRecord from "./put-item-record.json" with { type: "json" };
import rawPutEventRecord from "./put-event-record.json" with { type: "json" };
import rawPutCollectionHeadRecord from "./put-collection-head-record.json" with { type: "json" };
import rawDeleteCollectionHeadRecord from "./delete-collection-head-record.json" with { type: "json" };

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
export const samplePutVaultItemRecord: VaultItemRecord =
  parseVaultItemRecords([rawPutItemRecord])[0];
export const canonicalPutVaultItemRecord =
  serializeCanonicalVaultItemRecord(samplePutVaultItemRecord);
export const samplePutVaultEventRecord: VaultEventRecord =
  parseVaultEventRecord(rawPutEventRecord);
export const canonicalPutVaultEventRecord =
  serializeCanonicalVaultEventRecord(samplePutVaultEventRecord);
export const samplePutCollectionHeadRecord: CollectionHeadRecord =
  parseCollectionHeadRecord(rawPutCollectionHeadRecord);
export const canonicalPutCollectionHeadRecord =
  serializeCanonicalCollectionHeadRecord(samplePutCollectionHeadRecord);
export const sampleDeleteCollectionHeadRecord: CollectionHeadRecord =
  parseCollectionHeadRecord(rawDeleteCollectionHeadRecord);
export const canonicalDeleteCollectionHeadRecord =
  serializeCanonicalCollectionHeadRecord(sampleDeleteCollectionHeadRecord);
