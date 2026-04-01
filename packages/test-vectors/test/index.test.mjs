import test from "node:test";
import assert from "node:assert/strict";

import {
  canonicalAccountConfigRecord,
  canonicalCollectionHeadRecord,
  canonicalDeleteCollectionHeadRecord,
  canonicalDeleteVaultEventRecord,
  canonicalPutCollectionHeadRecord,
  canonicalPutVaultEventRecord,
  canonicalPutVaultItemRecord,
  canonicalVaultEventRecords,
  canonicalVaultItemRecords,
  sampleAccountConfigRecord,
  sampleCollectionHeadRecord,
  sampleDeleteCollectionHeadRecord,
  sampleDeleteVaultEventRecord,
  samplePutCollectionHeadRecord,
  samplePutVaultEventRecord,
  samplePutVaultItemRecord,
  sampleVaultEventRecords,
  sampleVaultItemRecords,
  sampleVaultItems,
} from "../src/index.ts";

test("sample vault items and records stay aligned", () => {
  assert.equal(sampleVaultItems.length, sampleVaultItemRecords.length);
  assert.deepEqual(
    sampleVaultItems.map((item) => item.id),
    sampleVaultItemRecords.map((record) => record.item.id),
  );
});

test("canonical vault item records preserve starter ordering", () => {
  assert.equal(canonicalVaultItemRecords.length, sampleVaultItemRecords.length);
  assert.equal(
    canonicalVaultItemRecords[0],
    '{"schemaVersion":1,"item":{"id":"login-gmail-primary","kind":"login","title":"Gmail","tags":["email","personal"],"username":"alice@example.com","urls":["https://accounts.google.com"]}}',
  );
});

test("sample vault event records preserve starter ordering", () => {
  assert.equal(sampleVaultEventRecords.length, 2);
  assert.equal(sampleVaultEventRecords[0].eventId, "evt-login-gmail-primary-v1");
  assert.equal(sampleVaultEventRecords[1].eventId, "evt-totp-gmail-primary-v1");
  assert.equal(canonicalVaultEventRecords.length, sampleVaultEventRecords.length);
});

test("delete event vector exports parsed and canonical forms", () => {
  assert.equal(sampleDeleteVaultEventRecord.action, "delete_item");
  assert.equal(sampleDeleteVaultEventRecord.itemId, "login-gmail-primary");
  assert.equal(
    canonicalDeleteVaultEventRecord,
    '{"schemaVersion":1,"eventId":"evt-login-gmail-primary-delete-v3","accountId":"acct-dev-001","deviceId":"dev-web-001","collectionId":"vault-personal","sequence":3,"occurredAt":"2026-03-31T10:04:00Z","action":"delete_item","itemId":"login-gmail-primary"}',
  );
});

test("collection head vector exports parsed and canonical forms", () => {
  assert.equal(sampleCollectionHeadRecord.latestEventId, "evt-totp-gmail-primary-v1");
  assert.equal(sampleCollectionHeadRecord.latestSeq, 2);
  assert.equal(
    canonicalCollectionHeadRecord,
    '{"schemaVersion":1,"accountId":"acct-dev-001","collectionId":"vault-personal","latestEventId":"evt-totp-gmail-primary-v1","latestSeq":2}',
  );
});

test("account config vector exports parsed and canonical forms", () => {
  assert.equal(sampleAccountConfigRecord.defaultCollectionId, "vault-personal");
  assert.deepEqual(sampleAccountConfigRecord.collectionIds, ["vault-personal"]);
  assert.deepEqual(sampleAccountConfigRecord.deviceIds, ["dev-web-001"]);
  assert.equal(
    canonicalAccountConfigRecord,
    '{"schemaVersion":1,"accountId":"acct-dev-001","defaultCollectionId":"vault-personal","collectionIds":["vault-personal"],"deviceIds":["dev-web-001"]}',
  );
});

test("put mutation vectors export parsed and canonical forms", () => {
  assert.equal(samplePutVaultItemRecord.item.id, "login-github-primary");
  assert.equal(
    canonicalPutVaultItemRecord,
    '{"schemaVersion":1,"item":{"id":"login-github-primary","kind":"login","title":"GitHub","tags":["dev"],"username":"alice","urls":["https://github.com/login"]}}',
  );
  assert.equal(samplePutVaultEventRecord.eventId, "evt-login-github-primary-v3");
  assert.equal(
    canonicalPutVaultEventRecord,
    '{"schemaVersion":1,"eventId":"evt-login-github-primary-v3","accountId":"acct-dev-001","deviceId":"dev-web-001","collectionId":"vault-personal","sequence":3,"occurredAt":"2026-03-31T10:02:00Z","action":"put_item","itemRecord":{"schemaVersion":1,"item":{"id":"login-github-primary","kind":"login","title":"GitHub","tags":["dev"],"username":"alice","urls":["https://github.com/login"]}}}',
  );
  assert.equal(samplePutCollectionHeadRecord.latestEventId, "evt-login-github-primary-v3");
  assert.equal(
    canonicalPutCollectionHeadRecord,
    '{"schemaVersion":1,"accountId":"acct-dev-001","collectionId":"vault-personal","latestEventId":"evt-login-github-primary-v3","latestSeq":3}',
  );
});

test("delete mutation head vector exports parsed and canonical forms", () => {
  assert.equal(sampleDeleteCollectionHeadRecord.latestEventId, "evt-login-gmail-primary-delete-v3");
  assert.equal(sampleDeleteCollectionHeadRecord.latestSeq, 3);
  assert.equal(
    canonicalDeleteCollectionHeadRecord,
    '{"schemaVersion":1,"accountId":"acct-dev-001","collectionId":"vault-personal","latestEventId":"evt-login-gmail-primary-delete-v3","latestSeq":3}',
  );
});
