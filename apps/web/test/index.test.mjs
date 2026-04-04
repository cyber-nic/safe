import test from "node:test";
import assert from "node:assert/strict";

import { createDeleteItemEventRecord } from "../../../packages/ts-sdk/src/index.ts";
import {
  sampleVaultSecretMaterial,
  sampleAccountConfigRecord,
  sampleCollectionHeadRecord,
  sampleVaultEventRecords,
  sampleVaultItemRecords,
} from "../../../packages/test-vectors/src/index.ts";
import {
  addApiKeyToVaultWorkspace,
  addLoginToVaultWorkspace,
  addNoteToVaultWorkspace,
  addSshKeyToVaultWorkspace,
  addTotpToVaultWorkspace,
  clearPersistedVaultWorkspace,
  createPersistedVaultWorkspaceSnapshot,
  createUnlockedVaultWorkspace,
  createVaultWorkspace,
  deleteItemFromVaultWorkspace,
  exportVaultWorkspace,
  getVaultItemDetail,
  getVaultLoginCredentialDetail,
  importVaultWorkspace,
  listVaultLoginCredentials,
  loadPersistedVaultWorkspace,
  listDeletedVaultItems,
  persistVaultWorkspaceSnapshot,
  revealLoginPassword,
  restoreItemToVaultWorkspace,
  serializePersistedVaultWorkspaceSnapshot,
  serializeVaultExportPayload,
  updateApiKeyInVaultWorkspace,
  updateLoginInVaultWorkspace,
  updateNoteInVaultWorkspace,
  updateSshKeyInVaultWorkspace,
  updateTotpInVaultWorkspace,
  unlockVaultWorkspace,
  webBootstrap,
} from "../src/index.ts";

class MemoryStorage {
  #values = new Map();

  getItem(key) {
    return this.#values.has(key) ? this.#values.get(key) : null;
  }

  setItem(key, value) {
    this.#values.set(key, value);
  }

  removeItem(key) {
    this.#values.delete(key);
  }
}

test("web bootstrap exposes a real starter workspace", () => {
  assert.equal(webBootstrap.overview.itemCount, 2);
  assert.equal(webBootstrap.overview.itemCountByKind.login, 1);
  assert.equal(webBootstrap.overview.itemCountByKind.totp, 1);
  assert.equal(webBootstrap.authenticators.length, 1);
  assert.equal(webBootstrap.insights.length, 0);
  assert.equal(webBootstrap.authenticators[0].relatedLoginTitle, "Gmail");
  assert.equal(webBootstrap.activity[0].eventId, "evt-totp-gmail-primary-v1");
  assert.deepEqual(webBootstrap.availableTags, [
    "2fa",
    "authenticator",
    "email",
    "personal",
  ]);
});

test("workspace insights flag missing 2FA and orphan authenticators", async () => {
  const withOrphanAuthenticator = await addTotpToVaultWorkspace({
    workspace: webBootstrap,
    secretMaterial: sampleVaultSecretMaterial,
    deviceId: "dev-web-001",
    title: "AWS Root 2FA",
    issuer: "AWS",
    accountName: "root@example.com",
    secretBase32: "JBSWY3DPEHPK3PXP",
  });

  const withUnprotectedLogin = await addLoginToVaultWorkspace({
    workspace: withOrphanAuthenticator.workspace,
    secretMaterial: withOrphanAuthenticator.secretMaterial,
    deviceId: "dev-web-001",
    title: "Linear",
    username: "alice@linear.example",
    url: "https://linear.app/login",
  });

  assert.equal(withUnprotectedLogin.workspace.insights.length, 2);
  assert.equal(
    withUnprotectedLogin.workspace.insights.some(
      (insight) =>
        insight.id === "logins-missing-totp" &&
        insight.itemIds.includes("login-linear-primary"),
    ),
    true,
  );
  assert.equal(
    withUnprotectedLogin.workspace.insights.some(
      (insight) =>
        insight.id === "orphan-authenticators" &&
        insight.itemIds.includes("totp-aws-primary"),
    ),
    true,
  );
});

test("createVaultWorkspace supports text search and kind filtering", () => {
  const searchWorkspace = createVaultWorkspace({
    accountConfig: sampleAccountConfigRecord,
    head: sampleCollectionHeadRecord,
    events: sampleVaultEventRecords,
    starterRecords: sampleVaultItemRecords,
    query: {
      text: "google",
      kind: "totp",
    },
  });

  assert.equal(searchWorkspace.items.length, 1);
  assert.equal(searchWorkspace.items[0].id, "totp-gmail-primary");
  assert.deepEqual(searchWorkspace.items[0].matchedFields, ["issuer"]);
});

test("createVaultWorkspace supports tag filtering", () => {
  const tagWorkspace = createVaultWorkspace({
    accountConfig: sampleAccountConfigRecord,
    head: sampleCollectionHeadRecord,
    events: sampleVaultEventRecords,
    query: {
      tag: "personal",
    },
  });

  assert.equal(tagWorkspace.items.length, 1);
  assert.equal(tagWorkspace.items[0].id, "login-gmail-primary");
});

test("createVaultWorkspace keeps recent activity when items are deleted", () => {
  const deleteEvent = createDeleteItemEventRecord({
    eventId: "evt-login-gmail-primary-delete-v3",
    accountId: "acct-dev-001",
    deviceId: "dev-web-001",
    collectionId: "vault-personal",
    sequence: 3,
    occurredAt: "2026-03-31T10:04:00Z",
    itemId: "login-gmail-primary",
  });

  const workspace = createVaultWorkspace({
    accountConfig: sampleAccountConfigRecord,
    head: {
      ...sampleCollectionHeadRecord,
      latestEventId: deleteEvent.eventId,
      latestSeq: deleteEvent.sequence,
    },
    events: [...sampleVaultEventRecords, deleteEvent],
  });

  assert.equal(workspace.overview.itemCount, 1);
  assert.equal(workspace.activity[0].action, "delete_item");
  assert.equal(workspace.activity[0].itemTitle, "Gmail");
  assert.equal(workspace.activity[0].itemKind, "login");
});

test("createVaultWorkspace rejects mismatched default collection heads", () => {
  assert.throws(
    () =>
      createVaultWorkspace({
        accountConfig: {
          ...sampleAccountConfigRecord,
          defaultCollectionId: "vault-shared",
        },
        head: sampleCollectionHeadRecord,
        events: sampleVaultEventRecords,
      }),
    /default collection mismatch: expected vault-shared got vault-personal/,
  );
});

test("unlockVaultWorkspace resolves local authenticator codes", async () => {
  const workspace = await unlockVaultWorkspace({
    workspace: webBootstrap,
    secretMaterial: sampleVaultSecretMaterial,
    at: new Date("1970-01-01T00:00:59Z"),
  });

  assert.equal(workspace.authenticators[0].status, "ready");
  assert.equal(workspace.authenticators[0].code, "287082");
  assert.equal(workspace.authenticators[0].secondsRemaining, 1);
  assert.equal(
    workspace.authenticators[0].validUntil,
    "1970-01-01T00:01:00.000Z",
  );
});

test("createUnlockedVaultWorkspace preserves locked state without secret material", async () => {
  const workspace = await createUnlockedVaultWorkspace({
    accountConfig: sampleAccountConfigRecord,
    head: sampleCollectionHeadRecord,
    events: sampleVaultEventRecords,
    starterRecords: sampleVaultItemRecords,
    secretMaterial: {},
    at: new Date("1970-01-01T00:00:59Z"),
  });

  assert.equal(workspace.authenticators[0].status, "locked");
  assert.equal(workspace.authenticators[0].code, null);
});

test("persisted vault workspace snapshots exclude unlocked codes and survive reload", async () => {
  const unlockedWorkspace = await createUnlockedVaultWorkspace({
    accountConfig: sampleAccountConfigRecord,
    head: sampleCollectionHeadRecord,
    events: sampleVaultEventRecords,
    starterRecords: sampleVaultItemRecords,
    secretMaterial: sampleVaultSecretMaterial,
    query: {
      text: "gmail",
      kind: "all",
    },
    at: new Date("1970-01-01T00:00:59Z"),
  });

  const storage = new MemoryStorage();
  const snapshot = persistVaultWorkspaceSnapshot({
    storage,
    storageKey: "safe:vault",
    workspace: unlockedWorkspace,
    savedAt: new Date("2026-04-01T11:25:00Z"),
  });

  assert.equal(snapshot.savedAt, "2026-04-01T11:25:00.000Z");
  assert.equal(snapshot.events.length, 2);
  const reloadedWorkspace = loadPersistedVaultWorkspace({
    storage,
    storageKey: "safe:vault",
  });
  assert.ok(reloadedWorkspace);
  assert.equal(reloadedWorkspace.overview.itemCount, 2);
  assert.equal(reloadedWorkspace.query.text, "gmail");
  assert.equal(reloadedWorkspace.authenticators[0].status, "locked");
  assert.equal(reloadedWorkspace.authenticators[0].code, null);
});

test("persisted vault workspace helpers serialize snapshots without secret material and can clear storage", () => {
  const storage = new MemoryStorage();
  const serializedSnapshot = serializePersistedVaultWorkspaceSnapshot({
    workspace: webBootstrap,
    savedAt: new Date("2026-04-01T11:26:00Z"),
  });
  const parsedSnapshot = JSON.parse(serializedSnapshot);

  assert.equal(parsedSnapshot.savedAt, "2026-04-01T11:26:00.000Z");
  assert.equal("secretMaterial" in parsedSnapshot, false);

  persistVaultWorkspaceSnapshot({
    storage,
    storageKey: "safe:vault",
    workspace: webBootstrap,
  });
  assert.ok(loadPersistedVaultWorkspace({ storage, storageKey: "safe:vault" }));

  clearPersistedVaultWorkspace({
    storage,
    storageKey: "safe:vault",
  });
  assert.equal(loadPersistedVaultWorkspace({ storage, storageKey: "safe:vault" }), null);

  const snapshot = createPersistedVaultWorkspaceSnapshot({
    workspace: webBootstrap,
    savedAt: new Date("2026-04-01T11:27:00Z"),
  });
  assert.equal(snapshot.savedAt, "2026-04-01T11:27:00.000Z");
});

test("addLoginToVaultWorkspace appends a replay-backed login mutation", async () => {
  const result = await addLoginToVaultWorkspace({
    workspace: webBootstrap,
    secretMaterial: sampleVaultSecretMaterial,
    deviceId: "dev-web-001",
    title: "GitHub",
    username: "alice",
    url: "https://github.com/login",
    password: "ghp-secret-123",
    tags: ["dev"],
    at: new Date("2026-04-01T10:20:00Z"),
  });

  assert.equal(result.itemId, "login-github-primary");
  assert.equal(result.workspace.overview.itemCount, 3);
  assert.equal(result.workspace.overview.latestSeq, 3);
  assert.equal(result.workspace.items.some((item) => item.id === "login-github-primary"), true);
  assert.equal(result.workspace.activity[0].eventId, "evt-login-github-primary-v3");
  assert.equal(
    result.secretMaterial["vault-secret://login/github-primary"],
    "ghp-secret-123",
  );
  assert.equal(
    revealLoginPassword({
      workspace: result.workspace,
      secretMaterial: result.secretMaterial,
      itemId: "login-github-primary",
    }),
    "ghp-secret-123",
  );
});

test("addTotpToVaultWorkspace stores secret material and unlocks the new authenticator", async () => {
  const result = await addTotpToVaultWorkspace({
    workspace: webBootstrap,
    secretMaterial: sampleVaultSecretMaterial,
    deviceId: "dev-web-001",
    title: "GitHub 2FA",
    issuer: "GitHub",
    accountName: "alice",
    secretBase32: "JBSWY3DPEHPK3PXP",
    at: new Date("1970-01-01T00:00:59Z"),
  });

  assert.equal(result.itemId, "totp-github-primary");
  assert.equal(
    result.secretMaterial["vault-secret://totp/github-primary"],
    "JBSWY3DPEHPK3PXP",
  );
  const authenticator = result.workspace.authenticators.find(
    (card) => card.id === "totp-github-primary",
  );
  assert.equal(authenticator?.status, "ready");
  assert.equal(authenticator?.code, "996554");
});

test("addNoteToVaultWorkspace, addApiKeyToVaultWorkspace, and addSshKeyToVaultWorkspace add the remaining item kinds", async () => {
  const withNote = await addNoteToVaultWorkspace({
    workspace: webBootstrap,
    secretMaterial: sampleVaultSecretMaterial,
    deviceId: "dev-web-001",
    title: "Server Notes",
    bodyPreview: "Rotate staging credentials before Friday",
    at: new Date("2026-04-01T10:55:00Z"),
  });
  assert.equal(withNote.itemId, "note-server-notes-primary");
  assert.equal(
    withNote.workspace.items.some((item) => item.id === "note-server-notes-primary"),
    true,
  );

  const withApiKey = await addApiKeyToVaultWorkspace({
    workspace: webBootstrap,
    secretMaterial: sampleVaultSecretMaterial,
    deviceId: "dev-web-001",
    title: "Stripe Prod",
    service: "Stripe",
    at: new Date("2026-04-01T10:56:00Z"),
  });
  assert.equal(withApiKey.itemId, "api-key-stripe-prod-primary");
  assert.equal(
    withApiKey.workspace.items.some((item) => item.id === "api-key-stripe-prod-primary"),
    true,
  );

  const withSshKey = await addSshKeyToVaultWorkspace({
    workspace: webBootstrap,
    secretMaterial: sampleVaultSecretMaterial,
    deviceId: "dev-web-001",
    title: "Prod Root",
    username: "root",
    host: "prod-01.internal",
    at: new Date("2026-04-01T10:57:00Z"),
  });
  assert.equal(withSshKey.itemId, "ssh-key-prod-root-primary");
  assert.equal(
    withSshKey.workspace.items.some((item) => item.id === "ssh-key-prod-root-primary"),
    true,
  );
});

test("deleteItemFromVaultWorkspace removes items and totp secret material", async () => {
  const unlocked = await createUnlockedVaultWorkspace({
    accountConfig: sampleAccountConfigRecord,
    head: sampleCollectionHeadRecord,
    events: sampleVaultEventRecords,
    starterRecords: sampleVaultItemRecords,
    secretMaterial: sampleVaultSecretMaterial,
    at: new Date("1970-01-01T00:00:59Z"),
  });

  const result = await deleteItemFromVaultWorkspace({
    workspace: unlocked,
    secretMaterial: sampleVaultSecretMaterial,
    deviceId: "dev-web-001",
    itemId: "totp-gmail-primary",
    at: new Date("2026-04-01T10:21:00Z"),
  });

  assert.equal(result.workspace.overview.itemCount, 1);
  assert.equal(result.workspace.authenticators.length, 0);
  assert.equal(
    "vault-secret://totp/gmail-primary" in result.secretMaterial,
    false,
  );
  assert.equal(result.workspace.activity[0].action, "delete_item");
});

test("deleteItemFromVaultWorkspace removes login password secret material", async () => {
  const withPassword = await addLoginToVaultWorkspace({
    workspace: webBootstrap,
    secretMaterial: sampleVaultSecretMaterial,
    deviceId: "dev-web-001",
    title: "GitHub",
    username: "alice",
    url: "https://github.com/login",
    password: "ghp-secret-123",
  });

  const deleted = await deleteItemFromVaultWorkspace({
    workspace: withPassword.workspace,
    secretMaterial: withPassword.secretMaterial,
    deviceId: "dev-web-001",
    itemId: withPassword.itemId,
  });

  assert.equal(
    "vault-secret://login/github-primary" in deleted.secretMaterial,
    false,
  );
});

test("getVaultItemDetail returns active item history", () => {
  const detail = getVaultItemDetail(webBootstrap, "login-gmail-primary");

  assert.equal(detail.status, "active");
  assert.equal(detail.title, "Gmail");
  assert.equal(detail.history.length, 1);
  assert.equal(detail.history[0].action, "put_item");
  assert.equal(detail.canRestore, false);
});

test("getVaultLoginCredentialDetail reports locked and unlocked password state with linked authenticator context", async () => {
  const lockedDetail = getVaultLoginCredentialDetail({
    workspace: webBootstrap,
    itemId: "login-gmail-primary",
  });
  assert.equal(lockedDetail.username, "alice@example.com");
  assert.equal(lockedDetail.primaryURL, "https://accounts.google.com");
  assert.equal(lockedDetail.passwordStatus, "locked");
  assert.equal(lockedDetail.password, null);
  assert.equal(lockedDetail.relatedAuthenticatorId, "totp-gmail-primary");
  assert.equal(lockedDetail.relatedAuthenticatorStatus, "locked");

  const unlockedWorkspace = await createUnlockedVaultWorkspace({
    accountConfig: sampleAccountConfigRecord,
    head: sampleCollectionHeadRecord,
    events: sampleVaultEventRecords,
    starterRecords: sampleVaultItemRecords,
    secretMaterial: sampleVaultSecretMaterial,
    at: new Date("1970-01-01T00:00:59Z"),
  });
  const unlockedDetail = getVaultLoginCredentialDetail({
    workspace: unlockedWorkspace,
    itemId: "login-gmail-primary",
    secretMaterial: sampleVaultSecretMaterial,
  });
  assert.equal(unlockedDetail.passwordStatus, "ready");
  assert.equal(unlockedDetail.password, "correct-horse-battery-staple");
  assert.equal(unlockedDetail.relatedAuthenticatorStatus, "ready");
  assert.equal(unlockedDetail.relatedAuthenticatorCode, "287082");
});

test("listVaultLoginCredentials summarizes password and authenticator readiness across logins", async () => {
  const withPasswordlessLogin = await addLoginToVaultWorkspace({
    workspace: webBootstrap,
    secretMaterial: sampleVaultSecretMaterial,
    deviceId: "dev-web-001",
    title: "Linear",
    username: "alice@linear.example",
    url: "https://linear.app/login",
  });

  const credentials = listVaultLoginCredentials({
    workspace: withPasswordlessLogin.workspace,
    secretMaterial: withPasswordlessLogin.secretMaterial,
  });

  assert.equal(credentials.length, 2);
  assert.equal(credentials[0].id, "login-gmail-primary");
  assert.equal(credentials[0].passwordStatus, "ready");
  assert.equal(credentials[0].relatedAuthenticatorId, "totp-gmail-primary");
  assert.equal(credentials[1].id, "login-linear-primary");
  assert.equal(credentials[1].passwordStatus, "missing");
  assert.equal(credentials[1].relatedAuthenticatorId, null);
});

test("listDeletedVaultItems surfaces deleted records and restoreItemToVaultWorkspace replays them back", async () => {
  const deleted = await deleteItemFromVaultWorkspace({
    workspace: webBootstrap,
    secretMaterial: sampleVaultSecretMaterial,
    deviceId: "dev-web-001",
    itemId: "login-gmail-primary",
    at: new Date("2026-04-01T10:30:00Z"),
  });

  const deletedItems = listDeletedVaultItems(deleted.workspace);
  assert.equal(deletedItems.length, 1);
  assert.equal(deletedItems[0].id, "login-gmail-primary");
  assert.equal(deletedItems[0].title, "Gmail");

  const deletedDetail = getVaultItemDetail(
    deleted.workspace,
    "login-gmail-primary",
  );
  assert.equal(deletedDetail.status, "deleted");
  assert.equal(deletedDetail.canRestore, true);
  assert.equal(deletedDetail.history[0].action, "delete_item");

  const restored = await restoreItemToVaultWorkspace({
    workspace: deleted.workspace,
    secretMaterial: deleted.secretMaterial,
    deviceId: "dev-web-001",
    itemId: "login-gmail-primary",
    at: new Date("2026-04-01T10:31:00Z"),
  });

  assert.equal(restored.workspace.overview.itemCount, 2);
  assert.equal(
    restored.workspace.items.some((item) => item.id === "login-gmail-primary"),
    true,
  );
  assert.equal(restored.workspace.activity[0].action, "put_item");
});

test("restoreItemToVaultWorkspace rejects active and unknown items", async () => {
  await assert.rejects(
    () =>
      restoreItemToVaultWorkspace({
        workspace: webBootstrap,
        secretMaterial: sampleVaultSecretMaterial,
        deviceId: "dev-web-001",
        itemId: "login-gmail-primary",
      }),
    /vault item already active: login-gmail-primary/,
  );

  await assert.rejects(
    () =>
      restoreItemToVaultWorkspace({
        workspace: webBootstrap,
        secretMaterial: sampleVaultSecretMaterial,
        deviceId: "dev-web-001",
        itemId: "missing-item",
      }),
    /vault item version not found: missing-item/,
  );
});

test("updateLoginInVaultWorkspace replays login edits into the workspace", async () => {
  const result = await updateLoginInVaultWorkspace({
    workspace: webBootstrap,
    secretMaterial: sampleVaultSecretMaterial,
    deviceId: "dev-web-001",
    itemId: "login-gmail-primary",
    title: "Gmail Personal",
    username: "alice+safe@example.com",
    url: "https://mail.google.com",
    password: "updated-gmail-password",
    tags: ["email", "personal", "updated"],
    at: new Date("2026-04-01T10:40:00Z"),
  });

  const detail = getVaultItemDetail(result.workspace, "login-gmail-primary");
  assert.equal(detail.title, "Gmail Personal");
  assert.equal(detail.history[0].action, "put_item");
  assert.equal(result.workspace.overview.latestSeq, 3);
  assert.equal(
    result.workspace.items.find((item) => item.id === "login-gmail-primary")?.title,
    "Gmail Personal",
  );
  assert.equal(
    revealLoginPassword({
      workspace: result.workspace,
      secretMaterial: result.secretMaterial,
      itemId: "login-gmail-primary",
    }),
    "updated-gmail-password",
  );
});

test("updateTotpInVaultWorkspace rotates authenticator metadata and secret material", async () => {
  const unlocked = await createUnlockedVaultWorkspace({
    accountConfig: sampleAccountConfigRecord,
    head: sampleCollectionHeadRecord,
    events: sampleVaultEventRecords,
    starterRecords: sampleVaultItemRecords,
    secretMaterial: sampleVaultSecretMaterial,
    at: new Date("1970-01-01T00:00:59Z"),
  });

  const result = await updateTotpInVaultWorkspace({
    workspace: unlocked,
    secretMaterial: sampleVaultSecretMaterial,
    deviceId: "dev-web-001",
    itemId: "totp-gmail-primary",
    title: "Gmail Primary 2FA",
    issuer: "Google Workspace",
    accountName: "alice@example.com",
    secretBase32: "JBSWY3DPEHPK3PXP",
    tags: ["2fa", "workspace"],
    at: new Date("1970-01-01T00:00:59Z"),
  });

  const authenticator = result.workspace.authenticators.find(
    (card) => card.id === "totp-gmail-primary",
  );
  assert.equal(authenticator?.title, "Gmail Primary 2FA");
  assert.equal(authenticator?.issuer, "Google Workspace");
  assert.equal(authenticator?.status, "ready");
  assert.equal(authenticator?.code, "996554");
  assert.equal(
    result.secretMaterial["vault-secret://totp/gmail-primary"],
    "JBSWY3DPEHPK3PXP",
  );
});

test("update helpers reject the wrong item kinds", async () => {
  await assert.rejects(
    () =>
      updateLoginInVaultWorkspace({
        workspace: webBootstrap,
        secretMaterial: sampleVaultSecretMaterial,
        deviceId: "dev-web-001",
        itemId: "totp-gmail-primary",
        title: "Invalid",
        username: "alice",
      }),
    /vault login update only supports login items: totp-gmail-primary/,
  );

  await assert.rejects(
    () =>
      updateTotpInVaultWorkspace({
        workspace: webBootstrap,
        secretMaterial: sampleVaultSecretMaterial,
        deviceId: "dev-web-001",
        itemId: "login-gmail-primary",
        title: "Invalid",
        issuer: "Google",
        accountName: "alice@example.com",
      }),
    /vault totp update only supports totp items: login-gmail-primary/,
  );
});

test("updateNoteInVaultWorkspace, updateApiKeyInVaultWorkspace, and updateSshKeyInVaultWorkspace replay edits for the remaining item kinds", async () => {
  const noteAdded = await addNoteToVaultWorkspace({
    workspace: webBootstrap,
    secretMaterial: sampleVaultSecretMaterial,
    deviceId: "dev-web-001",
    title: "Server Notes",
    bodyPreview: "Initial preview",
  });
  const noteUpdated = await updateNoteInVaultWorkspace({
    workspace: noteAdded.workspace,
    secretMaterial: noteAdded.secretMaterial,
    deviceId: "dev-web-001",
    itemId: noteAdded.itemId,
    title: "Server Notes Updated",
    bodyPreview: "Rotated on-call credentials",
    tags: ["note", "ops"],
  });
  assert.equal(
    getVaultItemDetail(noteUpdated.workspace, noteAdded.itemId).title,
    "Server Notes Updated",
  );

  const apiKeyAdded = await addApiKeyToVaultWorkspace({
    workspace: webBootstrap,
    secretMaterial: sampleVaultSecretMaterial,
    deviceId: "dev-web-001",
    title: "Stripe Prod",
    service: "Stripe",
  });
  const apiKeyUpdated = await updateApiKeyInVaultWorkspace({
    workspace: apiKeyAdded.workspace,
    secretMaterial: apiKeyAdded.secretMaterial,
    deviceId: "dev-web-001",
    itemId: apiKeyAdded.itemId,
    title: "Stripe Prod Primary",
    service: "Stripe Billing",
    tags: ["api", "billing"],
  });
  assert.equal(
    getVaultItemDetail(apiKeyUpdated.workspace, apiKeyAdded.itemId).title,
    "Stripe Prod Primary",
  );

  const sshAdded = await addSshKeyToVaultWorkspace({
    workspace: webBootstrap,
    secretMaterial: sampleVaultSecretMaterial,
    deviceId: "dev-web-001",
    title: "Prod Root",
    username: "root",
    host: "prod-01.internal",
  });
  const sshUpdated = await updateSshKeyInVaultWorkspace({
    workspace: sshAdded.workspace,
    secretMaterial: sshAdded.secretMaterial,
    deviceId: "dev-web-001",
    itemId: sshAdded.itemId,
    title: "Prod Root Bastion",
    username: "deploy",
    host: "bastion-01.internal",
    tags: ["ssh", "ops"],
  });
  assert.equal(
    getVaultItemDetail(sshUpdated.workspace, sshAdded.itemId).title,
    "Prod Root Bastion",
  );
});

test("remaining item-kind update helpers reject the wrong item kinds", async () => {
  await assert.rejects(
    () =>
      updateNoteInVaultWorkspace({
        workspace: webBootstrap,
        secretMaterial: sampleVaultSecretMaterial,
        deviceId: "dev-web-001",
        itemId: "login-gmail-primary",
        title: "Invalid",
        bodyPreview: "Invalid",
      }),
    /vault note update only supports note items: login-gmail-primary/,
  );

  await assert.rejects(
    () =>
      updateApiKeyInVaultWorkspace({
        workspace: webBootstrap,
        secretMaterial: sampleVaultSecretMaterial,
        deviceId: "dev-web-001",
        itemId: "login-gmail-primary",
        title: "Invalid",
        service: "Invalid",
      }),
    /vault api key update only supports api key items: login-gmail-primary/,
  );

  await assert.rejects(
    () =>
      updateSshKeyInVaultWorkspace({
        workspace: webBootstrap,
        secretMaterial: sampleVaultSecretMaterial,
        deviceId: "dev-web-001",
        itemId: "login-gmail-primary",
        title: "Invalid",
        username: "root",
        host: "invalid",
      }),
    /vault ssh key update only supports ssh key items: login-gmail-primary/,
  );
});

test("exportVaultWorkspace returns deterministic full-vault payloads", () => {
  const payload = exportVaultWorkspace(webBootstrap, sampleVaultSecretMaterial);

  assert.equal(payload.accountId, "acct-dev-001");
  assert.equal(payload.collectionId, "vault-personal");
  assert.equal(payload.latestSeq, 2);
  assert.equal(payload.items?.length, 2);
  assert.deepEqual(
    payload.items?.map((record) => record.item.id),
    ["login-gmail-primary", "totp-gmail-primary"],
  );
  assert.equal(
    payload.secretMaterial?.["vault-secret://login/gmail-primary"],
    "correct-horse-battery-staple",
  );
  assert.equal(
    payload.secretMaterial?.["vault-secret://totp/gmail-primary"],
    "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ",
  );

  const serialized = serializeVaultExportPayload(payload);
  assert.equal(serialized.includes('"items"'), true);
  assert.equal(serialized.includes('"secretMaterial"'), true);
});

test("workspace insights flag duplicate logins and API keys", async () => {
  const withDuplicateLogin = await addLoginToVaultWorkspace({
    workspace: webBootstrap,
    secretMaterial: sampleVaultSecretMaterial,
    deviceId: "dev-web-001",
    title: "Gmail Backup",
    username: "alice@example.com",
    url: "https://accounts.google.com",
  });

  const withDuplicateApiKeys = await addApiKeyToVaultWorkspace({
    workspace: withDuplicateLogin.workspace,
    secretMaterial: withDuplicateLogin.secretMaterial,
    deviceId: "dev-web-001",
    title: "Stripe Primary",
    service: "Stripe",
  });
  const withSecondApiKey = await addApiKeyToVaultWorkspace({
    workspace: withDuplicateApiKeys.workspace,
    secretMaterial: withDuplicateApiKeys.secretMaterial,
    deviceId: "dev-web-001",
    title: "Stripe Backup",
    service: "Stripe",
  });

  assert.equal(
    withSecondApiKey.workspace.insights.some(
      (insight) =>
        insight.title === "Duplicate Login Candidates" &&
        insight.itemIds.includes("login-gmail-primary") &&
        insight.itemIds.includes("login-gmail-backup-primary"),
    ),
    true,
  );
  assert.equal(
    withSecondApiKey.workspace.insights.some(
      (insight) =>
        insight.title === "Multiple API Keys For One Service" &&
        insight.itemIds.includes("api-key-stripe-primary-primary") &&
        insight.itemIds.includes("api-key-stripe-backup-primary"),
    ),
    true,
  );
});

test("exportVaultWorkspace supports single-item exports", () => {
  const payload = exportVaultWorkspace(
    webBootstrap,
    sampleVaultSecretMaterial,
    "login-gmail-primary",
  );

  assert.equal(payload.item?.item.id, "login-gmail-primary");
  assert.equal(payload.items, undefined);
  assert.equal(
    payload.secretMaterial?.["vault-secret://login/gmail-primary"],
    "correct-horse-battery-staple",
  );
});

test("importVaultWorkspace replays exported payloads back through put-item mutations", async () => {
  const added = await addLoginToVaultWorkspace({
    workspace: webBootstrap,
    secretMaterial: sampleVaultSecretMaterial,
    deviceId: "dev-web-001",
    title: "GitHub",
    username: "alice",
    url: "https://github.com/login",
    password: "ghp-secret-123",
    at: new Date("2026-04-01T10:50:00Z"),
  });

  const exportPayload = exportVaultWorkspace(
    added.workspace,
    added.secretMaterial,
    "login-github-primary",
  );
  const imported = await importVaultWorkspace({
    workspace: webBootstrap,
    secretMaterial: sampleVaultSecretMaterial,
    deviceId: "dev-web-001",
    payload: serializeVaultExportPayload(exportPayload),
    at: new Date("2026-04-01T10:51:00Z"),
  });

  assert.deepEqual(imported.importedItemIds, ["login-github-primary"]);
  assert.equal(imported.workspace.overview.itemCount, 3);
  assert.equal(imported.workspace.overview.latestSeq, 3);
  assert.equal(
    imported.workspace.items.some((item) => item.id === "login-github-primary"),
    true,
  );
  assert.equal(
    revealLoginPassword({
      workspace: imported.workspace,
      secretMaterial: imported.secretMaterial,
      itemId: "login-github-primary",
    }),
    "ghp-secret-123",
  );
});
