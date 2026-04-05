import assert from "node:assert/strict";
import { PassThrough } from "node:stream";
import test from "node:test";
import os from "node:os";
import path from "node:path";
import { mkdtemp, rm } from "node:fs/promises";

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
    assert.match(response.text, /Start dev session/);

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
