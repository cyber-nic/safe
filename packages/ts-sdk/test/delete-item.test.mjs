import test from "node:test";
import assert from "node:assert/strict";

import {
  createDeleteItemEventRecord,
  createPutItemEventRecord,
  createVaultItemRecord,
  parseVaultEventRecord,
  serializeCanonicalVaultEventRecord,
} from "../src/index.ts";

test("createDeleteItemEventRecord emits delete event records", () => {
  const record = createDeleteItemEventRecord({
    eventId: "evt-login-gmail-primary-delete-v3",
    accountId: "acct-dev-001",
    deviceId: "dev-web-001",
    collectionId: "vault-personal",
    sequence: 3,
    occurredAt: "2026-03-31T10:04:00Z",
    itemId: "login-gmail-primary",
  });

  assert.deepEqual(record, {
    schemaVersion: 1,
    eventId: "evt-login-gmail-primary-delete-v3",
    accountId: "acct-dev-001",
    deviceId: "dev-web-001",
    collectionId: "vault-personal",
    sequence: 3,
    occurredAt: "2026-03-31T10:04:00Z",
    action: "delete_item",
    itemId: "login-gmail-primary",
  });
});

test("parseVaultEventRecord accepts delete_item payloads", () => {
  const record = parseVaultEventRecord({
    schemaVersion: 1,
    eventId: "evt-login-gmail-primary-delete-v3",
    accountId: "acct-dev-001",
    deviceId: "dev-web-001",
    collectionId: "vault-personal",
    sequence: 3,
    occurredAt: "2026-03-31T10:04:00Z",
    action: "delete_item",
    itemId: "login-gmail-primary",
  });

  assert.equal(record.action, "delete_item");
  assert.equal(record.itemId, "login-gmail-primary");
});

test("serializeCanonicalVaultEventRecord preserves canonical delete-item order", () => {
  const record = createDeleteItemEventRecord({
    eventId: "evt-login-gmail-primary-delete-v3",
    accountId: "acct-dev-001",
    deviceId: "dev-web-001",
    collectionId: "vault-personal",
    sequence: 3,
    occurredAt: "2026-03-31T10:04:00Z",
    itemId: "login-gmail-primary",
  });

  assert.equal(
    serializeCanonicalVaultEventRecord(record),
    '{"schemaVersion":1,"eventId":"evt-login-gmail-primary-delete-v3","accountId":"acct-dev-001","deviceId":"dev-web-001","collectionId":"vault-personal","sequence":3,"occurredAt":"2026-03-31T10:04:00Z","action":"delete_item","itemId":"login-gmail-primary"}',
  );
});

test("serializeCanonicalVaultEventRecord still supports put_item records", () => {
  const record = createPutItemEventRecord({
    eventId: "evt-login-github-primary-v3",
    accountId: "acct-dev-001",
    deviceId: "dev-web-001",
    collectionId: "vault-personal",
    sequence: 3,
    occurredAt: "2026-03-31T10:02:00Z",
    itemRecord: createVaultItemRecord({
      id: "login-github-primary",
      kind: "login",
      title: "GitHub",
      tags: ["manual", "password"],
      username: "alice",
      urls: ["https://github.com/login"],
    }),
  });

  assert.equal(
    serializeCanonicalVaultEventRecord(record),
    '{"schemaVersion":1,"eventId":"evt-login-github-primary-v3","accountId":"acct-dev-001","deviceId":"dev-web-001","collectionId":"vault-personal","sequence":3,"occurredAt":"2026-03-31T10:02:00Z","action":"put_item","itemRecord":{"schemaVersion":1,"item":{"id":"login-github-primary","kind":"login","title":"GitHub","tags":["manual","password"],"username":"alice","urls":["https://github.com/login"]}}}',
  );
});
