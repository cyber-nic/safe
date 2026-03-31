import { sampleVaultItems } from "../../../packages/test-vectors/src/index.js";
import { describeVaultItem } from "../../../packages/ts-sdk/src/index.js";

export const webBootstrap = {
  app: "safe-web",
  surfaces: ["vault", "authenticator", "sharing"] as const,
  starterItems: sampleVaultItems,
  starterSummaries: sampleVaultItems.map(describeVaultItem),
};
