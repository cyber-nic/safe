import {
  sampleVaultEventRecords,
  sampleVaultItemRecords,
} from "../../../packages/test-vectors/src/index.js";
import { describeVaultItem } from "../../../packages/ts-sdk/src/index.js";

export const webBootstrap = {
  app: "safe-web",
  surfaces: ["vault", "authenticator", "sharing"] as const,
  starterEvents: sampleVaultEventRecords,
  starterRecords: sampleVaultItemRecords,
  starterItems: sampleVaultItemRecords.map((record) => record.item),
  starterSummaries: sampleVaultItemRecords.map((record) =>
    describeVaultItem(record.item),
  ),
};
