import test from "node:test";
import assert from "node:assert/strict";

import {
  buildDeleteItemMutation,
  buildPutItemMutation,
  createAccountConfigRecord,
  createCollectionHeadRecord,
  createDeleteItemEventRecord,
  createPutItemEventRecord,
  generateTOTP,
  generateTotpCodeForItem,
  createTotpItem,
  createVaultItemRecord,
  ensureMonotonicHead,
  parseAccountConfigRecord,
  parseCollectionHeadRecord,
  parseVaultEventRecord,
  replayCollection,
  replayCollectionAgainstHead,
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

test("generateTOTP matches the RFC SHA1 vector", async () => {
  const code = await generateTOTP(
    "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ",
    new Date("1970-01-01T00:00:59Z"),
    8,
    30,
    "SHA1",
  );

  assert.equal(code, "94287082");
});

test("generateTotpCodeForItem returns code window metadata", async () => {
  const snapshot = await generateTotpCodeForItem(
    createTotpItem({
      id: "totp-gmail-primary",
      title: "Gmail 2FA",
      issuer: "Google",
      accountName: "alice@example.com",
      secretRef: "vault-secret://totp/gmail-primary",
    }),
    "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ",
    new Date("1970-01-01T00:00:59Z"),
  );

  assert.equal(snapshot.code, "287082");
  assert.equal(snapshot.secondsRemaining, 1);
  assert.equal(snapshot.validFrom, "1970-01-01T00:00:30.000Z");
  assert.equal(snapshot.validUntil, "1970-01-01T00:01:00.000Z");
});

test("generateTOTP rejects unsupported algorithms", async () => {
  await assert.rejects(
    () =>
      generateTOTP(
        "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ",
        new Date("1970-01-01T00:00:59Z"),
        6,
        30,
        "SHA256",
      ),
    /unsupported totp algorithm: SHA256/,
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

test("replayCollection sorts input and builds the latest state", () => {
  const loginEvent = createPutItemEventRecord({
    eventId: "evt-login-gmail-primary-v1",
    accountId: "acct-dev-001",
    deviceId: "dev-web-001",
    collectionId: "vault-personal",
    sequence: 1,
    occurredAt: "2026-03-31T10:00:00Z",
    itemRecord: createVaultItemRecord({
      id: "login-gmail-primary",
      kind: "login",
      title: "Gmail",
      tags: ["email", "personal"],
      username: "alice@example.com",
      urls: ["https://accounts.google.com"],
    }),
  });
  const totpEvent = createPutItemEventRecord({
    eventId: "evt-totp-gmail-primary-v1",
    accountId: "acct-dev-001",
    deviceId: "dev-web-001",
    collectionId: "vault-personal",
    sequence: 2,
    occurredAt: "2026-03-31T10:01:00Z",
    itemRecord: createVaultItemRecord(createTotpItem({
      id: "totp-gmail-primary",
      title: "Gmail OTP",
      issuer: "Google",
      accountName: "alice@example.com",
      secretRef: "secret://gmail/totp",
      tags: ["2fa", "email"],
    })),
  });

  const projection = replayCollection([totpEvent, loginEvent]);

  assert.equal(projection.accountId, "acct-dev-001");
  assert.equal(projection.collectionId, "vault-personal");
  assert.equal(projection.latestSeq, 2);
  assert.equal(projection.latestEventId, "evt-totp-gmail-primary-v1");
  assert.equal(projection.items.size, 2);
  assert.equal(projection.items.get("login-gmail-primary")?.item.title, "Gmail");
});

test("replayCollection deletes items and rejects sequence gaps", () => {
  const loginEvent = createPutItemEventRecord({
    eventId: "evt-login-gmail-primary-v1",
    accountId: "acct-dev-001",
    deviceId: "dev-web-001",
    collectionId: "vault-personal",
    sequence: 1,
    occurredAt: "2026-03-31T10:00:00Z",
    itemRecord: createVaultItemRecord({
      id: "login-gmail-primary",
      kind: "login",
      title: "Gmail",
      tags: ["email", "personal"],
      username: "alice@example.com",
      urls: ["https://accounts.google.com"],
    }),
  });
  const deleteEvent = createDeleteItemEventRecord({
    eventId: "evt-login-gmail-primary-delete-v2",
    accountId: "acct-dev-001",
    deviceId: "dev-web-001",
    collectionId: "vault-personal",
    sequence: 2,
    occurredAt: "2026-03-31T10:04:00Z",
    itemId: "login-gmail-primary",
  });

  const deletedProjection = replayCollection([loginEvent, deleteEvent]);
  assert.equal(deletedProjection.latestSeq, 2);
  assert.equal(deletedProjection.items.has("login-gmail-primary"), false);

  assert.throws(
    () =>
      replayCollection([
        loginEvent,
        createDeleteItemEventRecord({
          ...deleteEvent,
          eventId: "evt-login-gmail-primary-delete-v3",
          sequence: 3,
        }),
      ]),
    /sync replay sequence gap: expected 2 got 3/,
  );
});

test("replayCollectionAgainstHead enforces latest seq and event alignment", () => {
  const events = [
    createPutItemEventRecord({
      eventId: "evt-login-gmail-primary-v1",
      accountId: "acct-dev-001",
      deviceId: "dev-web-001",
      collectionId: "vault-personal",
      sequence: 1,
      occurredAt: "2026-03-31T10:00:00Z",
      itemRecord: createVaultItemRecord({
        id: "login-gmail-primary",
        kind: "login",
        title: "Gmail",
        tags: ["email", "personal"],
        username: "alice@example.com",
        urls: ["https://accounts.google.com"],
      }),
    }),
    createPutItemEventRecord({
      eventId: "evt-totp-gmail-primary-v1",
      accountId: "acct-dev-001",
      deviceId: "dev-web-001",
      collectionId: "vault-personal",
      sequence: 2,
      occurredAt: "2026-03-31T10:01:00Z",
      itemRecord: createVaultItemRecord(createTotpItem({
        id: "totp-gmail-primary",
        title: "Gmail OTP",
        issuer: "Google",
        accountName: "alice@example.com",
        secretRef: "secret://gmail/totp",
        tags: ["2fa", "email"],
      })),
    }),
  ];
  const head = createCollectionHeadRecord({
    accountId: "acct-dev-001",
    collectionId: "vault-personal",
    latestEventId: "evt-totp-gmail-primary-v1",
    latestSeq: 2,
  });

  const projection = replayCollectionAgainstHead(events, head);
  assert.equal(projection.latestSeq, 2);

  assert.throws(
    () =>
      replayCollectionAgainstHead(
        events,
        createCollectionHeadRecord({
          accountId: "acct-dev-001",
          collectionId: "vault-personal",
          latestEventId: "evt-totp-gmail-primary-v1",
          latestSeq: 3,
        }),
      ),
    /sync head mismatch: latestSeq expected 3 got 2/,
  );
});

test("buildPutItemMutation and buildDeleteItemMutation advance the head", () => {
  const head = createCollectionHeadRecord({
    accountId: "acct-dev-001",
    collectionId: "vault-personal",
    latestEventId: "evt-totp-gmail-primary-v1",
    latestSeq: 2,
  });

  const putMutation = buildPutItemMutation(
    head,
    "dev-web-001",
    createVaultItemRecord({
      id: "login-github-primary",
      kind: "login",
      title: "GitHub",
      tags: ["dev"],
      username: "alice",
      urls: ["https://github.com/login"],
    }),
    "2026-03-31T10:02:00Z",
  );
  assert.equal(putMutation.event.eventId, "evt-login-github-primary-v3");
  assert.equal(putMutation.event.sequence, 3);
  assert.equal(putMutation.newHead.latestEventId, putMutation.event.eventId);
  assert.equal(putMutation.newHead.latestSeq, 3);

  const deleteMutation = buildDeleteItemMutation(
    head,
    "dev-web-001",
    "login-gmail-primary",
    "2026-03-31T10:04:00Z",
  );
  assert.equal(deleteMutation.event.eventId, "evt-login-gmail-primary-delete-v3");
  assert.equal(deleteMutation.event.sequence, 3);
  assert.equal(deleteMutation.event.action, "delete_item");
  assert.equal(deleteMutation.newHead.latestEventId, deleteMutation.event.eventId);
  assert.equal(deleteMutation.newHead.latestSeq, 3);
});
