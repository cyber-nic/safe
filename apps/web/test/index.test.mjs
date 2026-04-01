import test from "node:test";
import assert from "node:assert/strict";

import { createDeleteItemEventRecord } from "../../../packages/ts-sdk/src/index.ts";
import {
  sampleAccountConfigRecord,
  sampleCollectionHeadRecord,
  sampleVaultEventRecords,
  sampleVaultItemRecords,
} from "../../../packages/test-vectors/src/index.ts";
import { createVaultWorkspace, webBootstrap } from "../src/index.ts";

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
