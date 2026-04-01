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
  addLoginToVaultWorkspace,
  addTotpToVaultWorkspace,
  createUnlockedVaultWorkspace,
  createVaultWorkspace,
  deleteItemFromVaultWorkspace,
  getVaultItemDetail,
  listDeletedVaultItems,
  restoreItemToVaultWorkspace,
  unlockVaultWorkspace,
  webBootstrap,
} from "../src/index.ts";

test("web bootstrap exposes a real starter workspace", () => {
  assert.equal(webBootstrap.overview.itemCount, 2);
  assert.equal(webBootstrap.overview.itemCountByKind.login, 1);
  assert.equal(webBootstrap.overview.itemCountByKind.totp, 1);
  assert.equal(webBootstrap.authenticators.length, 1);
  assert.equal(webBootstrap.authenticators[0].relatedLoginTitle, "Gmail");
  assert.equal(webBootstrap.activity[0].eventId, "evt-totp-gmail-primary-v1");
  assert.deepEqual(webBootstrap.availableTags, [
    "2fa",
    "authenticator",
    "email",
    "personal",
  ]);
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

test("addLoginToVaultWorkspace appends a replay-backed login mutation", async () => {
  const result = await addLoginToVaultWorkspace({
    workspace: webBootstrap,
    secretMaterial: sampleVaultSecretMaterial,
    deviceId: "dev-web-001",
    title: "GitHub",
    username: "alice",
    url: "https://github.com/login",
    tags: ["dev"],
    at: new Date("2026-04-01T10:20:00Z"),
  });

  assert.equal(result.itemId, "login-github-primary");
  assert.equal(result.workspace.overview.itemCount, 3);
  assert.equal(result.workspace.overview.latestSeq, 3);
  assert.equal(result.workspace.items.some((item) => item.id === "login-github-primary"), true);
  assert.equal(result.workspace.activity[0].eventId, "evt-login-github-primary-v3");
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

test("getVaultItemDetail returns active item history", () => {
  const detail = getVaultItemDetail(webBootstrap, "login-gmail-primary");

  assert.equal(detail.status, "active");
  assert.equal(detail.title, "Gmail");
  assert.equal(detail.history.length, 1);
  assert.equal(detail.history[0].action, "put_item");
  assert.equal(detail.canRestore, false);
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
