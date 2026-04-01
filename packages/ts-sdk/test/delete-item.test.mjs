import test from "node:test";
import assert from "node:assert/strict";

import {
  createAccountConfigRecord,
  createCollectionHeadRecord,
  createDeleteItemEventRecord,
  createPutItemEventRecord,
  createVaultItemRecord,
  ensureMonotonicHead,
  parseAccountConfigRecord,
  parseCollectionHeadRecord,
  parseVaultEventRecord,
  serializeCanonicalAccountConfigRecord,
  serializeCanonicalCollectionHeadRecord,
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

test("parse and serialize collection head records canonically", () => {
  const record = parseCollectionHeadRecord({
    schemaVersion: 1,
    accountId: "acct-dev-001",
    collectionId: "vault-personal",
    latestEventId: "evt-totp-gmail-primary-v1",
    latestSeq: 2,
  });

  assert.deepEqual(record, createCollectionHeadRecord({
    accountId: "acct-dev-001",
    collectionId: "vault-personal",
    latestEventId: "evt-totp-gmail-primary-v1",
    latestSeq: 2,
  }));

  assert.equal(
    serializeCanonicalCollectionHeadRecord(record),
    '{"schemaVersion":1,"accountId":"acct-dev-001","collectionId":"vault-personal","latestEventId":"evt-totp-gmail-primary-v1","latestSeq":2}',
  );
});

test("parse and serialize account config records canonically", () => {
  const record = parseAccountConfigRecord({
    schemaVersion: 1,
    accountId: "acct-dev-001",
    defaultCollectionId: "vault-personal",
    collectionIds: ["vault-personal"],
    deviceIds: ["dev-web-001"],
  });

  assert.deepEqual(record, createAccountConfigRecord({
    accountId: "acct-dev-001",
    defaultCollectionId: "vault-personal",
    collectionIds: ["vault-personal"],
    deviceIds: ["dev-web-001"],
  }));

  assert.equal(
    serializeCanonicalAccountConfigRecord(record),
    '{"schemaVersion":1,"accountId":"acct-dev-001","defaultCollectionId":"vault-personal","collectionIds":["vault-personal"],"deviceIds":["dev-web-001"]}',
  );
});

test("ensureMonotonicHead rejects rollback and equal-sequence divergence", () => {
  const trusted = createCollectionHeadRecord({
    accountId: "acct-dev-001",
    collectionId: "vault-personal",
    latestEventId: "evt-login-github-primary-v3",
    latestSeq: 3,
  });

  assert.throws(
    () =>
      ensureMonotonicHead(
        trusted,
        createCollectionHeadRecord({
          accountId: "acct-dev-001",
          collectionId: "vault-personal",
          latestEventId: "evt-totp-gmail-primary-v1",
          latestSeq: 2,
        }),
      ),
    /sync stale head rejected: trusted 3 candidate 2/,
  );

  assert.throws(
    () =>
      ensureMonotonicHead(
        createCollectionHeadRecord({
          accountId: "acct-dev-001",
          collectionId: "vault-personal",
          latestEventId: "evt-totp-gmail-primary-v1",
          latestSeq: 2,
        }),
        createCollectionHeadRecord({
          accountId: "acct-dev-001",
          collectionId: "vault-personal",
          latestEventId: "evt-different",
          latestSeq: 2,
        }),
      ),
    /sync head mismatch: latestEventId expected evt-totp-gmail-primary-v1 got evt-different/,
  );
});

test("ensureMonotonicHead accepts forward progress", () => {
  assert.doesNotThrow(() =>
    ensureMonotonicHead(
      createCollectionHeadRecord({
        accountId: "acct-dev-001",
        collectionId: "vault-personal",
        latestEventId: "evt-totp-gmail-primary-v1",
        latestSeq: 2,
      }),
      createCollectionHeadRecord({
        accountId: "acct-dev-001",
        collectionId: "vault-personal",
        latestEventId: "evt-login-github-primary-v3",
        latestSeq: 3,
      }),
    ),
  );
});
