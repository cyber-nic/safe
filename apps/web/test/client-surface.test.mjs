import assert from "node:assert/strict";
import { PassThrough } from "node:stream";
import test from "node:test";
import os from "node:os";
import path from "node:path";
import { mkdtemp, readFile, rm } from "node:fs/promises";

import { createWebClientServer } from "../src/server.ts";

function getSetCookie(response) {
  const raw = response.headers["set-cookie"];
  if (Array.isArray(raw)) {
    return raw[0]?.split(";")[0] ?? null;
  }
  return raw ? raw.split(";")[0] : null;
}

function createMockResponse() {
  const headers = {};
  const chunks = [];
  let finished;
  const done = new Promise((resolve) => {
    finished = resolve;
  });

  return {
    statusCode: 200,
    headers,
    setHeader(name, value) {
      headers[name.toLowerCase()] = value;
    },
    getHeader(name) {
      return headers[name.toLowerCase()];
    },
    end(chunk = "") {
      if (chunk) {
        chunks.push(Buffer.isBuffer(chunk) ? chunk : Buffer.from(chunk));
      }
      finished();
    },
    async asResult() {
      await done;
      return {
        status: this.statusCode,
        headers,
        text: Buffer.concat(chunks).toString("utf8"),
      };
    },
  };
}

async function invoke(app, input) {
  const request = new PassThrough();
  request.method = input.method ?? "GET";
  request.url = input.url;
  request.headers = input.headers ?? {};

  const response = createMockResponse();
  app(request, response);
  if (input.body) {
    request.end(input.body.toString());
  } else {
    request.end();
  }

  return response.asResult();
}

test("web client can identify, save, lock, restart, unlock, and read a secret", async () => {
  const dataDir = await mkdtemp(path.join(os.tmpdir(), "safe-web-"));
  const identity = {
    accountId: "acct-dev-001",
    deviceId: "dev-web-001",
    env: "test",
  };

  try {
    let app = createWebClientServer({
      dataDir,
      resolveIdentity: async () => identity,
      now: () => new Date("2026-04-05T09:30:00Z"),
    });

    let response = await invoke(app, { url: "/" });
    assert.equal(response.status, 200);
    assert.match(response.text, /Sign in with OAuth/);

    response = await invoke(app, {
      method: "POST",
      url: "/identify",
    });
    assert.equal(response.status, 303);
    assert.equal(response.headers.location, "/unlock");
    let cookie = getSetCookie(response);
    assert.ok(cookie);

    response = await invoke(app, {
      url: "/unlock",
      headers: { cookie },
    });
    assert.equal(response.status, 200);
    assert.match(response.text, /Create local vault password/);

    response = await invoke(app, {
      method: "POST",
      url: "/unlock",
      headers: {
        cookie,
        "content-type": "application/x-www-form-urlencoded",
      },
      body: new URLSearchParams({
        password: "correct horse battery staple",
      }),
    });
    assert.equal(response.status, 303);
    assert.equal(response.headers.location, "/onboarding/recovery");

    response = await invoke(app, {
      url: "/vault",
      headers: { cookie },
    });
    assert.equal(response.status, 303);
    assert.equal(response.headers.location, "/unlock");

    response = await invoke(app, {
      url: "/onboarding/recovery",
      headers: { cookie },
    });
    assert.equal(response.status, 200);
    assert.match(response.text, /Store this recovery key/);
    assert.match(response.text, /24-word mnemonic/);

    response = await invoke(app, {
      method: "POST",
      url: "/onboarding/recovery",
      headers: {
        cookie,
        "content-type": "application/x-www-form-urlencoded",
      },
      body: new URLSearchParams({}),
    });
    assert.equal(response.status, 200);
    assert.match(response.text, /You must confirm/);

    response = await invoke(app, {
      method: "POST",
      url: "/onboarding/recovery",
      headers: {
        cookie,
        "content-type": "application/x-www-form-urlencoded",
      },
      body: new URLSearchParams({
        confirmed: "yes",
      }),
    });
    assert.equal(response.status, 303);
    assert.equal(response.headers.location, "/vault");

    response = await invoke(app, {
      url: "/vault",
      headers: { cookie },
    });
    assert.equal(response.status, 200);
    assert.match(response.text, /No secrets yet/);

    response = await invoke(app, {
      method: "POST",
      url: "/vault/login",
      headers: {
        cookie,
        "content-type": "application/x-www-form-urlencoded",
      },
      body: new URLSearchParams({
        title: "GitHub",
        username: "alice",
        url: "https://github.com/login",
        password: "ghp-secret-123",
      }),
    });
    assert.equal(response.status, 303);
    assert.equal(response.headers.location, "/vault?item=login-github-primary");

    response = await invoke(app, {
      url: "/vault?item=login-github-primary",
      headers: { cookie },
    });
    assert.equal(response.status, 200);
    assert.match(response.text, /ghp-secret-123/);
    assert.match(response.text, /login-github-primary/);

    response = await invoke(app, {
      method: "POST",
      url: "/vault/items",
      headers: {
        cookie,
        "content-type": "application/x-www-form-urlencoded",
      },
      body: new URLSearchParams({
        itemKind: "totp",
        title: "GitHub OTP",
        issuer: "GitHub",
        accountName: "alice@example.com",
        secretBase32: "JBSWY3DPEHPK3PXP",
        tags: "2fa,authenticator",
      }),
    });
    assert.equal(response.status, 303);
    assert.equal(response.headers.location, "/vault?item=totp-github-primary");

    response = await invoke(app, {
      url: "/vault?item=totp-github-primary",
      headers: { cookie },
    });
    assert.equal(response.status, 200);
    assert.match(response.text, /Selected Authenticator/);
    assert.match(response.text, /GitHub OTP/);
    assert.match(response.text, /<code>\d{6}<\/code>/);

    response = await invoke(app, {
      method: "POST",
      url: "/vault/items",
      headers: {
        cookie,
        "content-type": "application/x-www-form-urlencoded",
      },
      body: new URLSearchParams({
        itemKind: "note",
        title: "Deployment Runbook",
        bodyPreview: "Backup codes stored offline.",
        tags: "ops,note",
      }),
    });
    assert.equal(response.status, 303);
    assert.equal(response.headers.location, "/vault?item=note-deployment-runbook-primary");

    response = await invoke(app, {
      url: "/vault?q=backup&kind=note",
      headers: { cookie },
    });
    assert.equal(response.status, 200);
    assert.match(response.text, /Deployment Runbook/);
    assert.doesNotMatch(response.text, /login-github-primary/);

    response = await invoke(app, {
      method: "POST",
      url: "/vault/items/update",
      headers: {
        cookie,
        "content-type": "application/x-www-form-urlencoded",
      },
      body: new URLSearchParams({
        itemId: "login-github-primary",
        itemKind: "login",
        title: "GitHub",
        username: "alice-updated",
        url: "https://github.com/login",
        tags: "manual,m3",
      }),
    });
    assert.equal(response.status, 303);
    assert.equal(response.headers.location, "/vault?item=login-github-primary");

    response = await invoke(app, {
      url: "/vault?item=login-github-primary",
      headers: { cookie },
    });
    assert.equal(response.status, 200);
    assert.match(response.text, /alice-updated/);

    response = await invoke(app, {
      method: "POST",
      url: "/vault/items/delete",
      headers: {
        cookie,
        "content-type": "application/x-www-form-urlencoded",
      },
      body: new URLSearchParams({
        itemId: "note-deployment-runbook-primary",
      }),
    });
    assert.equal(response.status, 303);
    assert.equal(response.headers.location, "/vault?deleted=1");

    response = await invoke(app, {
      url: "/vault?item=note-deployment-runbook-primary",
      headers: { cookie },
    });
    assert.equal(response.status, 200);
    assert.match(response.text, /Deleted Item/);
    assert.match(response.text, /Restore item/);

    response = await invoke(app, {
      method: "POST",
      url: "/vault/items/restore",
      headers: {
        cookie,
        "content-type": "application/x-www-form-urlencoded",
      },
      body: new URLSearchParams({
        itemId: "note-deployment-runbook-primary",
      }),
    });
    assert.equal(response.status, 303);
    assert.equal(
      response.headers.location,
      "/vault?item=note-deployment-runbook-primary",
    );

    response = await invoke(app, {
      url: "/vault?item=note-deployment-runbook-primary",
      headers: { cookie },
    });
    assert.equal(response.status, 200);
    assert.match(response.text, /Deployment Runbook/);
    assert.match(response.text, /Backup codes stored offline/);

    const recoveryRecord = JSON.parse(
      await readFile(
        path.join(dataDir, "accounts", "acct-dev-001", "recovery.json"),
        "utf8",
      ),
    );
    assert.equal(recoveryRecord.accountId, "acct-dev-001");
    assert.equal(recoveryRecord.schemaVersion, 1);

    response = await invoke(app, {
      method: "POST",
      url: "/lock",
      headers: { cookie },
    });
    assert.equal(response.status, 303);
    assert.equal(response.headers.location, "/unlock");

    app = createWebClientServer({
      dataDir,
      resolveIdentity: async () => identity,
      now: () => new Date("2026-04-05T09:35:00Z"),
    });

    response = await invoke(app, {
      method: "POST",
      url: "/identify",
    });
    assert.equal(response.status, 303);
    cookie = getSetCookie(response);
    assert.ok(cookie);

    response = await invoke(app, {
      url: "/unlock",
      headers: { cookie },
    });
    assert.equal(response.status, 200);
    assert.match(response.text, /Unlock local vault/);

    response = await invoke(app, {
      method: "POST",
      url: "/unlock",
      headers: {
        cookie,
        "content-type": "application/x-www-form-urlencoded",
      },
      body: new URLSearchParams({
        password: "wrong password",
      }),
    });
    assert.equal(response.status, 200);
    assert.match(response.text, /Unlock failed/);

    response = await invoke(app, {
      method: "POST",
      url: "/unlock",
      headers: {
        cookie,
        "content-type": "application/x-www-form-urlencoded",
      },
      body: new URLSearchParams({
        password: "correct horse battery staple",
      }),
    });
    assert.equal(response.status, 303);
    assert.equal(response.headers.location, "/vault");

    response = await invoke(app, {
      url: "/vault?item=login-github-primary",
      headers: { cookie },
    });
    assert.equal(response.status, 200);
    assert.match(response.text, /ghp-secret-123/);
    assert.match(response.text, /GitHub/);
  } finally {
    await rm(dataDir, { recursive: true, force: true });
  }
});

test("web client requests remote account access during identify when control plane is configured", async () => {
  const dataDir = await mkdtemp(path.join(os.tmpdir(), "safe-web-"));
  const previousURL = process.env.SAFE_WEB_CONTROL_PLANE_URL;
  const previousToken = process.env.SAFE_WEB_OAUTH_ACCESS_TOKEN;
  const previousDeviceID = process.env.SAFE_WEB_DEVICE_ID;
  const previousFetch = globalThis.fetch;
  const requests = [];

  process.env.SAFE_WEB_CONTROL_PLANE_URL = "http://control-plane.test";
  process.env.SAFE_WEB_OAUTH_ACCESS_TOKEN = "oauth-token";
  process.env.SAFE_WEB_DEVICE_ID = "dev-web-001";
  globalThis.fetch = async (input, init = {}) => {
    const url = input instanceof URL ? input : new URL(input);
    requests.push({
      path: url.pathname,
      method: init.method ?? "GET",
      body: init.body ?? null,
    });

    if (url.pathname === "/v1/session") {
      assert.equal(init.headers.authorization, "Bearer oauth-token");
      return new Response(
        JSON.stringify({
          accountId: "acct-dev-001",
          env: "test",
          bucket: "safe-test",
          endpoint: "http://localstack:4566",
          region: "us-east-1",
        }),
        {
          status: 200,
          headers: { "content-type": "application/json" },
        },
      );
    }

    if (url.pathname === "/v1/access/account") {
      assert.equal(init.headers.authorization, "Bearer oauth-token");
      return new Response(
        JSON.stringify({
          bucket: "safe-test",
          endpoint: "http://localstack:4566",
          region: "us-east-1",
          keyId: "dev-hmac-v1",
          token: "signed-token",
          capability: {
            version: 1,
            accountId: "acct-dev-001",
            deviceId: "dev-web-001",
            bucket: "safe-test",
            prefix: "accounts/acct-dev-001/",
            allowedActions: ["get", "put"],
            issuedAt: "2026-04-06T08:00:00Z",
            expiresAt: "2026-04-06T08:05:00Z",
          },
        }),
        {
          status: 200,
          headers: { "content-type": "application/json" },
        },
      );
    }

    throw new Error(`unexpected fetch ${url.pathname}`);
  };

  try {
    const app = createWebClientServer({ dataDir });

    let response = await invoke(app, {
      method: "POST",
      url: "/identify",
    });
    assert.equal(response.status, 303);
    const cookie = getSetCookie(response);
    assert.ok(cookie);

    response = await invoke(app, {
      method: "POST",
      url: "/unlock",
      headers: {
        cookie,
        "content-type": "application/x-www-form-urlencoded",
      },
      body: new URLSearchParams({
        password: "correct horse battery staple",
      }),
    });
    assert.equal(response.status, 303);
    assert.equal(response.headers.location, "/onboarding/recovery");

    response = await invoke(app, {
      method: "POST",
      url: "/onboarding/recovery",
      headers: {
        cookie,
        "content-type": "application/x-www-form-urlencoded",
      },
      body: new URLSearchParams({
        confirmed: "yes",
      }),
    });
    assert.equal(response.status, 303);
    assert.equal(response.headers.location, "/vault");

    response = await invoke(app, {
      url: "/vault",
      headers: { cookie },
    });
    assert.equal(response.status, 200);
    assert.match(response.text, /Remote Sync/);
    assert.match(response.text, /accounts\/acct-dev-001\//);

    assert.equal(requests.length, 2);
    assert.equal(requests[0].path, "/v1/session");
    assert.equal(requests[1].path, "/v1/access/account");
    assert.equal(requests[1].method, "POST");
    assert.match(String(requests[1].body), /"accountId":"acct-dev-001"/);
    assert.match(String(requests[1].body), /"deviceId":"dev-web-001"/);
  } finally {
    if (previousURL === undefined) {
      delete process.env.SAFE_WEB_CONTROL_PLANE_URL;
    } else {
      process.env.SAFE_WEB_CONTROL_PLANE_URL = previousURL;
    }
    if (previousToken === undefined) {
      delete process.env.SAFE_WEB_OAUTH_ACCESS_TOKEN;
    } else {
      process.env.SAFE_WEB_OAUTH_ACCESS_TOKEN = previousToken;
    }
    if (previousDeviceID === undefined) {
      delete process.env.SAFE_WEB_DEVICE_ID;
    } else {
      process.env.SAFE_WEB_DEVICE_ID = previousDeviceID;
    }
    globalThis.fetch = previousFetch;
    await rm(dataDir, { recursive: true, force: true });
  }
});

test("web client redirects through provider auth and completes oauth callback", async () => {
  const dataDir = await mkdtemp(path.join(os.tmpdir(), "safe-web-"));
  const previousEnv = {
    SAFE_OAUTH_DEV_MODE: process.env.SAFE_OAUTH_DEV_MODE,
    SAFE_OAUTH_PROVIDER: process.env.SAFE_OAUTH_PROVIDER,
    SAFE_OAUTH_CLIENT_ID: process.env.SAFE_OAUTH_CLIENT_ID,
    SAFE_OAUTH_CLIENT_SECRET: process.env.SAFE_OAUTH_CLIENT_SECRET,
    SAFE_OAUTH_REDIRECT_URL: process.env.SAFE_OAUTH_REDIRECT_URL,
    SAFE_OAUTH_AUTHORIZE_URL: process.env.SAFE_OAUTH_AUTHORIZE_URL,
    SAFE_OAUTH_TOKEN_URL: process.env.SAFE_OAUTH_TOKEN_URL,
    SAFE_WEB_CONTROL_PLANE_URL: process.env.SAFE_WEB_CONTROL_PLANE_URL,
  };
  const previousFetch = globalThis.fetch;
  const requests = [];

  process.env.SAFE_OAUTH_DEV_MODE = "false";
  process.env.SAFE_OAUTH_PROVIDER = "Google";
  process.env.SAFE_OAUTH_CLIENT_ID = "client-test-123.apps.googleusercontent.com";
  process.env.SAFE_OAUTH_CLIENT_SECRET = "secret-test-123";
  process.env.SAFE_OAUTH_REDIRECT_URL = "http://localhost:3000/auth/callback";
  process.env.SAFE_OAUTH_AUTHORIZE_URL = "https://accounts.google.com/o/oauth2/v2/auth";
  process.env.SAFE_OAUTH_TOKEN_URL = "https://oauth2.googleapis.com/token";
  process.env.SAFE_WEB_CONTROL_PLANE_URL = "http://control-plane.test";
  globalThis.fetch = async (input, init = {}) => {
    const url = input instanceof URL ? input : new URL(input);
    requests.push({
      url: url.toString(),
      method: init.method ?? "GET",
      body: init.body ? String(init.body) : null,
    });

    if (url.toString() === "https://oauth2.googleapis.com/token") {
      assert.equal(init.method, "POST");
      assert.match(String(init.body), /grant_type=authorization_code/);
      assert.match(String(init.body), /code=oauth-code-123/);
      assert.match(
        String(init.body),
        /client_id=client-test-123.apps.googleusercontent.com/,
      );
      return new Response(
        JSON.stringify({
          id_token: "google-id-token",
        }),
        {
          status: 200,
          headers: { "content-type": "application/json" },
        },
      );
    }

    if (url.pathname === "/v1/session") {
      assert.equal(init.headers.authorization, "Bearer google-id-token");
      return new Response(
        JSON.stringify({
          accountId: "acct-oauth-001",
          env: "test",
          bucket: "safe-test",
          endpoint: "http://localstack:4566",
          region: "us-east-1",
        }),
        {
          status: 200,
          headers: { "content-type": "application/json" },
        },
      );
    }

    if (url.pathname === "/v1/access/account") {
      assert.equal(init.headers.authorization, "Bearer google-id-token");
      return new Response(
        JSON.stringify({
          bucket: "safe-test",
          endpoint: "http://localstack:4566",
          region: "us-east-1",
          keyId: "dev-hmac-v1",
          token: "signed-token",
          capability: {
            version: 1,
            accountId: "acct-oauth-001",
            deviceId: "dev-web-001",
            bucket: "safe-test",
            prefix: "accounts/acct-oauth-001/",
            allowedActions: ["get", "put"],
            issuedAt: "2026-04-08T08:00:00Z",
            expiresAt: "2026-04-08T08:05:00Z",
          },
        }),
        {
          status: 200,
          headers: { "content-type": "application/json" },
        },
      );
    }

    throw new Error(`unexpected fetch ${url.toString()}`);
  };

  try {
    const app = createWebClientServer({ dataDir });

    let response = await invoke(app, {
      method: "GET",
      url: "/auth/login",
    });
    assert.equal(response.status, 303);
    assert.match(
      response.headers.location,
      /^https:\/\/accounts\.google\.com\/o\/oauth2\/v2\/auth\?/,
    );
    const redirectURL = new URL(response.headers.location);
    assert.equal(redirectURL.searchParams.get("client_id"), process.env.SAFE_OAUTH_CLIENT_ID);
    assert.equal(
      redirectURL.searchParams.get("redirect_uri"),
      process.env.SAFE_OAUTH_REDIRECT_URL,
    );
    assert.equal(redirectURL.searchParams.get("response_type"), "code");
    assert.equal(redirectURL.searchParams.get("scope"), "openid email profile");
    assert.equal(redirectURL.searchParams.get("code_challenge_method"), "S256");
    assert.ok(redirectURL.searchParams.get("state"));
    assert.ok(redirectURL.searchParams.get("code_challenge"));
    const cookie = getSetCookie(response);
    assert.ok(cookie);

    response = await invoke(app, {
      method: "GET",
      url: `/auth/callback?code=oauth-code-123&state=${encodeURIComponent(
        redirectURL.searchParams.get("state"),
      )}`,
      headers: { cookie },
    });
    assert.equal(response.status, 303);
    assert.equal(response.headers.location, "/unlock");

    response = await invoke(app, {
      url: "/unlock",
      headers: { cookie },
    });
    assert.equal(response.status, 200);
    assert.match(response.text, /acct-oauth-001/);
    assert.match(response.text, /Create local vault password/);

    assert.equal(requests[0].url, "https://oauth2.googleapis.com/token");
    assert.equal(requests[1].url, "http://control-plane.test/v1/session");
    assert.equal(requests[2].url, "http://control-plane.test/v1/access/account");
  } finally {
    for (const [key, value] of Object.entries(previousEnv)) {
      if (value === undefined) {
        delete process.env[key];
      } else {
        process.env[key] = value;
      }
    }
    globalThis.fetch = previousFetch;
    await rm(dataDir, { recursive: true, force: true });
  }
});

test("web client surfaces sync actions, enrolled devices, and enrollment approval", async () => {
  const dataDir = await mkdtemp(path.join(os.tmpdir(), "safe-web-"));
  const identity = {
    accountId: "acct-dev-001",
    deviceId: "dev-web-001",
    env: "test",
  };
  const remoteAccess = {
    bucket: "safe-test",
    endpoint: "http://127.0.0.1:4566",
    region: "us-east-1",
    keyId: "dev-hmac-v1",
    token: "signed-token",
    capability: {
      version: 1,
      accountId: "acct-dev-001",
      deviceId: "dev-web-001",
      bucket: "safe-test",
      prefix: "accounts/acct-dev-001/",
      allowedActions: ["get", "put"],
      issuedAt: "2026-04-06T08:00:00Z",
      expiresAt: "2026-04-06T08:05:00Z",
    },
  };

  const devices = [
    {
      schemaVersion: 1,
      accountId: "acct-dev-001",
      deviceId: "dev-cli-001",
      label: "Safe CLI dev-cli-001",
      deviceType: "cli",
      signingPublicKey: "c2ln",
      encryptionPublicKey: "ZW5j",
      createdAt: "2026-04-06T07:55:00Z",
      status: "active",
    },
    {
      schemaVersion: 1,
      accountId: "acct-dev-001",
      deviceId: "dev-web-001",
      label: "Safe Web dev-web-001",
      deviceType: "web",
      signingPublicKey: "c2lnMg",
      encryptionPublicKey: "ZW5jMg",
      createdAt: "2026-04-06T08:00:00Z",
      status: "active",
    },
  ];
  const pendingEnrollments = [
    {
      schemaVersion: 1,
      accountId: "acct-dev-001",
      deviceId: "dev-web-099",
      label: "Safe Web dev-web-099",
      deviceType: "web",
      encryptionPublicKey: "ZW5jLTA5OQ",
      requestedAt: "2026-04-06T08:10:00Z",
    },
  ];
  const calls = [];

  try {
    const app = createWebClientServer({
      dataDir,
      resolveIdentity: async () => identity,
      resolveRemoteAccess: async () => remoteAccess,
      now: () => new Date("2026-04-06T08:12:00Z"),
      runSafeCommand: async (args, env) => {
        calls.push({ args, env });
        assert.match(env.SAFE_LOCAL_RUNTIME_DIR, /accounts$/);
        assert.equal(env.SAFE_LOCAL_PASSWORD, "correct horse battery staple");
        assert.equal(env.SAFE_DEVICE_ID, "dev-web-001");

        if (args[0] === "--json" && args[1] === "device" && args[2] === "list") {
          return {
            stdout: JSON.stringify(devices),
            stderr: "",
          };
        }
        if (args[0] === "--json" && args[1] === "device" && args[2] === "pending") {
          return {
            stdout: JSON.stringify(pendingEnrollments),
            stderr: "",
          };
        }
        if (
          args[0] === "--json" &&
          args[1] === "device" &&
          args[2] === "approve" &&
          args[3] === "dev-web-099"
        ) {
          pendingEnrollments.splice(0, pendingEnrollments.length);
          return {
            stdout: JSON.stringify({
              approvedDeviceId: "dev-web-099",
              accountId: "acct-dev-001",
              bundleAlgorithm: "x25519-hkdf-aes-256-gcm",
            }),
            stderr: "",
          };
        }
        if (args[0] === "--json" && args[1] === "sync" && args[2] === "push") {
          return {
            stdout: JSON.stringify({
              pushed: 1,
              latestSeq: 3,
              deviceId: "dev-web-001",
            }),
            stderr: "",
          };
        }
        if (args[0] === "--json" && args[1] === "sync" && args[2] === "pull") {
          return {
            stdout: JSON.stringify({
              latestSeq: 3,
              itemCount: 2,
            }),
            stderr: "",
          };
        }

        throw new Error(`unexpected safe command ${args.join(" ")}`);
      },
    });

    let response = await invoke(app, {
      method: "POST",
      url: "/identify",
    });
    assert.equal(response.status, 303);
    const cookie = getSetCookie(response);
    assert.ok(cookie);

    response = await invoke(app, {
      method: "POST",
      url: "/unlock",
      headers: {
        cookie,
        "content-type": "application/x-www-form-urlencoded",
      },
      body: new URLSearchParams({
        password: "correct horse battery staple",
      }),
    });
    assert.equal(response.status, 303);
    assert.equal(response.headers.location, "/onboarding/recovery");

    response = await invoke(app, {
      method: "POST",
      url: "/onboarding/recovery",
      headers: {
        cookie,
        "content-type": "application/x-www-form-urlencoded",
      },
      body: new URLSearchParams({
        confirmed: "yes",
      }),
    });
    assert.equal(response.status, 303);
    assert.equal(response.headers.location, "/vault");

    response = await invoke(app, {
      url: "/vault",
      headers: { cookie },
    });
    assert.equal(response.status, 200);
    assert.match(response.text, /Sync push/);
    assert.match(response.text, /Sync pull/);
    assert.match(response.text, /Safe CLI dev-cli-001/);
    assert.match(response.text, /Safe Web dev-web-099/);
    assert.match(response.text, /Approve/);

    response = await invoke(app, {
      method: "POST",
      url: "/vault/devices/approve",
      headers: {
        cookie,
        "content-type": "application/x-www-form-urlencoded",
      },
      body: new URLSearchParams({
        deviceId: "dev-web-099",
      }),
    });
    assert.equal(response.status, 303);
    assert.equal(response.headers.location, "/vault");

    response = await invoke(app, {
      url: "/vault",
      headers: { cookie },
    });
    assert.equal(response.status, 200);
    assert.match(response.text, /Approved enrollment request for dev-web-099/);
    assert.doesNotMatch(response.text, /Safe Web dev-web-099/);

    response = await invoke(app, {
      method: "POST",
      url: "/vault/sync/push",
      headers: { cookie },
    });
    assert.equal(response.status, 303);
    assert.equal(response.headers.location, "/vault");

    response = await invoke(app, {
      url: "/vault",
      headers: { cookie },
    });
    assert.equal(response.status, 200);
    assert.match(response.text, /Sync push completed against the account remote path/);

    response = await invoke(app, {
      method: "POST",
      url: "/vault/sync/pull",
      headers: { cookie },
    });
    assert.equal(response.status, 303);
    assert.equal(response.headers.location, "/vault");

    response = await invoke(app, {
      url: "/vault",
      headers: { cookie },
    });
    assert.equal(response.status, 200);
    assert.match(response.text, /Sync pull completed and reloaded the local vault snapshot/);

    assert.ok(
      calls.some((call) => call.args[1] === "device" && call.args[2] === "approve"),
    );
    assert.ok(
      calls.some((call) => call.args[1] === "sync" && call.args[2] === "push"),
    );
    assert.ok(
      calls.some((call) => call.args[1] === "sync" && call.args[2] === "pull"),
    );
  } finally {
    await rm(dataDir, { recursive: true, force: true });
  }
});
