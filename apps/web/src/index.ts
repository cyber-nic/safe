import { sampleVaultItemRecords } from "../../../packages/test-vectors/src/index.js";
import { describeVaultItem } from "../../../packages/ts-sdk/src/index.js";

export const webBootstrap = {
  app: "safe-web",
  surfaces: ["vault", "authenticator", "sharing"] as const,
  starterRecords: sampleVaultItemRecords,
  starterItems: sampleVaultItemRecords.map((record) => record.item),
  starterSummaries: sampleVaultItemRecords.map((record) =>
    describeVaultItem(record.item),
  ),
};
