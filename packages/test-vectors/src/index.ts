import rawVaultItems from "./vault-items.json";

import { parseVaultItems, type VaultItem } from "../../ts-sdk/src/index.js";

export const sampleVaultItems: VaultItem[] = parseVaultItems(rawVaultItems);
