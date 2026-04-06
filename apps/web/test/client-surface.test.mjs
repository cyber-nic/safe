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
