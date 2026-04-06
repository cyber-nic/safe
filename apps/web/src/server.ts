import type { IncomingMessage, ServerResponse } from "node:http";
import http from "node:http";
import path from "node:path";
import { randomUUID } from "node:crypto";

import {
  addLoginToVaultWorkspace,
  getVaultItemDetail,
  getVaultLoginCredentialDetail,
} from "./index.ts";
import {
  createLocalRuntimeStore,
  type ClientIdentity,
} from "./local-runtime.ts";

type SessionState = {
  identity: ClientIdentity | null;
  remoteAccess: AccountAccessResponse | null;
  accountKey: Buffer | null;
};

type ControlPlaneSession = {
  accountId: string;
  env: string;
  bucket: string;
  endpoint: string;
  region: string;
};

type AccountAccessCapability = {
  version: number;
  accountId: string;
  deviceId: string;
  bucket: string;
  prefix: string;
  allowedActions: string[];
  issuedAt: string;
  expiresAt: string;
};

type AccountAccessResponse = {
  bucket: string;
  endpoint: string;
  region: string;
  keyId: string;
  token: string;
  capability: AccountAccessCapability;
};

export type WebClientServerOptions = {
  dataDir?: string;
  now?: () => Date;
  resolveIdentity?: () => Promise<ClientIdentity>;
  resolveRemoteAccess?: (
    identity: ClientIdentity,
  ) => Promise<AccountAccessResponse | null>;
};

export function createWebClientServer(
  options: WebClientServerOptions = {},
): http.RequestListener {
  const dataDir =
    options.dataDir ??
    process.env.SAFE_WEB_DATA_DIR ??
    path.join(process.cwd(), "apps/web/.safe-client-data");
  const store = createLocalRuntimeStore({ baseDir: dataDir });
  const now = options.now ?? (() => new Date());
  const resolveIdentity = options.resolveIdentity ?? defaultResolveIdentity;
  const resolveRemoteAccess =
    options.resolveRemoteAccess ?? defaultResolveRemoteAccess;
  const sessions = new Map<string, SessionState>();

  return (request, response) => {
    void handleRequest(request, response).catch((error: unknown) => {
      const message =
        error instanceof Error ? error.message : "unexpected server error";
      writeHTML(
        response,
        500,
        renderPage({
          title: "Safe",
          body: `
            <section class="panel stack">
              <p class="eyebrow">Server Error</p>
              <h1>Request failed</h1>
              <p>${escapeHTML(message)}</p>
              <p><a class="link" href="/">Return home</a></p>
            </section>
          `,
        }),
      );
    });
  };

  async function handleRequest(
    request: IncomingMessage,
    response: ServerResponse,
  ): Promise<void> {
    const url = new URL(request.url ?? "/", "http://safe.local");
    const session = getOrCreateSession(request, response);

    if (request.method === "GET" && url.pathname === "/") {
      if (session.identity) {
        redirect(response, session.accountKey ? "/vault" : "/unlock");
        return;
      }

      writeHTML(response, 200, renderHomePage());
      return;
    }

    if (request.method === "POST" && url.pathname === "/identify") {
      session.identity = await resolveIdentity();
      session.remoteAccess = await resolveRemoteAccess(session.identity);
      session.accountKey = null;
      redirect(response, "/unlock");
      return;
    }

    if (request.method === "GET" && url.pathname === "/unlock") {
      if (!session.identity) {
        redirect(response, "/");
        return;
      }

      const firstUse = !(await store.hasUnlockRecord(session.identity.accountId));
      writeHTML(
        response,
        200,
        renderUnlockPage({
          identity: session.identity,
          firstUse,
        }),
      );
      return;
    }

    if (request.method === "POST" && url.pathname === "/unlock") {
      if (!session.identity) {
        redirect(response, "/");
        return;
      }

      const form = await readForm(request);
      const password = form.get("password")?.trim() ?? "";
      const firstUse = !(await store.hasUnlockRecord(session.identity.accountId));

      try {
        const unlocked = await store.unlock(session.identity, password);
        session.accountKey = unlocked.accountKey;
        redirect(response, "/vault");
        return;
      } catch {
        writeHTML(
          response,
          200,
          renderUnlockPage({
            identity: session.identity,
            firstUse,
            error:
              firstUse
                ? "Safe could not create local unlock state with that password."
                : "Unlock failed. Check the password and try again.",
          }),
        );
        return;
      }
    }

    if (request.method === "POST" && url.pathname === "/lock") {
      session.accountKey = null;
      redirect(response, "/unlock");
      return;
    }

    if (request.method === "POST" && url.pathname === "/logout") {
      session.identity = null;
      session.accountKey = null;
      redirect(response, "/");
      return;
    }

    if (request.method === "GET" && url.pathname === "/vault") {
      if (!session.identity || !session.accountKey) {
        redirect(response, "/unlock");
        return;
      }

      const unlocked = await store.loadUnlockedWithAccountKey(
        session.identity,
        session.accountKey,
      );
      writeHTML(
        response,
        200,
        renderVaultPage({
          identity: session.identity,
          remoteAccess: session.remoteAccess,
          itemId: url.searchParams.get("item"),
          unlocked,
        }),
      );
      return;
    }

    if (request.method === "POST" && url.pathname === "/vault/login") {
      if (!session.identity || !session.accountKey) {
        redirect(response, "/unlock");
        return;
      }

      const form = await readForm(request);
      const unlocked = await store.loadUnlockedWithAccountKey(
        session.identity,
        session.accountKey,
      );

      try {
        const result = await addLoginToVaultWorkspace({
          workspace: unlocked.workspace,
          secretMaterial: unlocked.secretMaterial,
          deviceId: session.identity.deviceId,
          title: form.get("title") ?? "",
          username: form.get("username") ?? "",
          url: form.get("url") ?? "",
          password: form.get("password") ?? "",
          tags: ["manual", "m1"],
          at: now(),
        });

        await store.persistUnlockedVault(session.identity, {
          workspace: result.workspace,
          secretMaterial: result.secretMaterial,
          accountKey: session.accountKey,
        });
        redirect(response, `/vault?item=${encodeURIComponent(result.itemId)}`);
        return;
      } catch (error) {
        writeHTML(
          response,
          200,
          renderVaultPage({
            identity: session.identity,
            remoteAccess: session.remoteAccess,
            unlocked,
            itemId: null,
            error:
              error instanceof Error
                ? error.message
                : "Safe could not save that secret.",
          }),
        );
        return;
      }
    }

    writeHTML(
      response,
      404,
      renderPage({
        title: "Safe",
        body: `
          <section class="panel stack">
            <p class="eyebrow">Missing</p>
            <h1>Nothing here</h1>
            <p>The route <code>${escapeHTML(url.pathname)}</code> does not exist.</p>
            <p><a class="link" href="/">Return home</a></p>
          </section>
        `,
      }),
    );
  }

  function getOrCreateSession(
    request: IncomingMessage,
    response: ServerResponse,
  ): SessionState {
    const existing = parseCookies(request.headers.cookie ?? "").safe_web_session;
    if (existing && sessions.has(existing)) {
      return sessions.get(existing)!;
    }

    const sessionId = randomUUID();
    const session = {
      identity: null,
      remoteAccess: null,
      accountKey: null,
    };
    sessions.set(sessionId, session);
    response.setHeader("Set-Cookie", serializeCookie("safe_web_session", sessionId));
    return session;
  }
}

export function createServer(options: WebClientServerOptions = {}): http.Server {
  return http.createServer(createWebClientServer(options));
}

function renderHomePage(): string {
  return renderPage({
    title: "Safe",
    body: `
      <section class="hero">
        <div class="hero-copy stack">
          <p class="eyebrow">M3 Client Surface</p>
          <h1>Sign in with OAuth, unlock locally, and keep the sync path real.</h1>
          <p class="lede">
            This local web client resolves identity through the control plane's OAuth-backed session endpoint
            before unlocking the durable local runtime and requesting account-scoped remote access.
          </p>
          <form method="post" action="/identify">
            <button class="button button-primary" type="submit">Sign in with OAuth</button>
          </form>
        </div>
        <aside class="hero-panel stack">
          <p class="eyebrow">Flow</p>
          <ol class="steps">
            <li>Resolve the OAuth-backed account session.</li>
            <li>Create or reuse the account password unlock path.</li>
            <li>Save one login secret into the durable local runtime.</li>
            <li>Lock or restart, unlock again, and read the same secret back.</li>
          </ol>
        </aside>
      </section>
    `,
  });
}

function renderUnlockPage(input: {
  identity: ClientIdentity;
  firstUse: boolean;
  error?: string;
}): string {
  const title = input.firstUse ? "Create local vault password" : "Unlock local vault";
  const detail = input.firstUse
    ? "First use creates the account-local unlock record and an empty durable runtime."
    : "Unlock uses the persisted account-scoped Argon2id record and reopens the durable runtime.";

  return renderPage({
    title: "Safe Unlock",
    body: `
      <section class="panel stack">
        <p class="eyebrow">${input.firstUse ? "First Use" : "Unlock"}</p>
        <h1>${title}</h1>
        <p>${detail}</p>
        ${input.error ? `<p class="error">${escapeHTML(input.error)}</p>` : ""}
        <dl class="meta-grid">
          <div><dt>Account</dt><dd>${escapeHTML(input.identity.accountId)}</dd></div>
          <div><dt>Device</dt><dd>${escapeHTML(input.identity.deviceId)}</dd></div>
          <div><dt>Env</dt><dd>${escapeHTML(input.identity.env)}</dd></div>
        </dl>
        <form class="stack" method="post" action="/unlock">
          <label class="field">
            <span>Password</span>
            <input name="password" type="password" autocomplete="current-password" required />
          </label>
          <div class="actions">
            <button class="button button-primary" type="submit">
              ${input.firstUse ? "Create vault" : "Unlock vault"}
            </button>
            <a class="button button-secondary" href="/">Start over</a>
          </div>
        </form>
      </section>
    `,
  });
}

function renderVaultPage(input: {
  identity: ClientIdentity;
  remoteAccess: AccountAccessResponse | null;
  unlocked: Awaited<ReturnType<ReturnType<typeof createLocalRuntimeStore>["unlock"]>>;
  itemId: string | null;
  error?: string;
}): string {
  const selectedItemId =
    input.itemId ??
    input.unlocked.workspace.itemRecords.find((record) => record.item.kind === "login")?.item.id ??
    null;
  const selectedLogin =
    selectedItemId === null
      ? null
      : getSafeLoginDetail(
          input.unlocked.workspace,
          input.unlocked.secretMaterial,
          selectedItemId,
        );
  const itemsMarkup =
    input.unlocked.workspace.itemRecords.length === 0
      ? `<p class="empty">No secrets yet. Save one login below to complete the M1 loop.</p>`
      : `
        <ul class="item-list">
          ${input.unlocked.workspace.itemRecords
            .map((record) => {
              const detail = getVaultItemDetail(input.unlocked.workspace, record.item.id);
              return `
                <li>
                  <a class="item-link" href="/vault?item=${encodeURIComponent(record.item.id)}">
                    <span class="item-title">${escapeHTML(detail.title)}</span>
                    <span class="item-summary">${escapeHTML(detail.summary)}</span>
                  </a>
                </li>
              `;
            })
            .join("")}
        </ul>
      `;
  const detailMarkup =
    selectedLogin === null
      ? `
        <div class="detail-empty">
          <p class="eyebrow">Read Path</p>
          <h2>No login selected</h2>
          <p>Save a login, then lock or restart and unlock again to confirm the durable read path.</p>
        </div>
      `
      : `
        <div class="detail-card stack">
          <p class="eyebrow">Selected Secret</p>
          <h2>${escapeHTML(selectedLogin.title)}</h2>
          <dl class="detail-grid">
            <div><dt>Username</dt><dd>${escapeHTML(selectedLogin.username)}</dd></div>
            <div><dt>URL</dt><dd>${escapeHTML(selectedLogin.primaryURL ?? "n/a")}</dd></div>
            <div><dt>Password</dt><dd><code>${escapeHTML(selectedLogin.password ?? "locked")}</code></dd></div>
            <div><dt>Status</dt><dd>${escapeHTML(selectedLogin.passwordStatus)}</dd></div>
          </dl>
        </div>
      `;

  return renderPage({
    title: "Safe Vault",
    body: `
      <header class="vault-header">
        <div>
          <p class="eyebrow">Unlocked Vault</p>
          <h1>${escapeHTML(input.identity.accountId)}</h1>
        </div>
        <div class="actions">
          <form method="post" action="/lock"><button class="button button-secondary" type="submit">Lock</button></form>
          <form method="post" action="/logout"><button class="button button-secondary" type="submit">Sign out</button></form>
        </div>
      </header>

      <section class="stats-grid">
        <article class="stat-card">
          <span class="stat-label">Items</span>
          <strong>${input.unlocked.workspace.overview.itemCount}</strong>
        </article>
        <article class="stat-card">
          <span class="stat-label">Latest seq</span>
          <strong>${input.unlocked.workspace.overview.latestSeq}</strong>
        </article>
        <article class="stat-card">
          <span class="stat-label">Events</span>
          <strong>${input.unlocked.workspace.events.length}</strong>
        </article>
      </section>

      ${input.remoteAccess
        ? `
      <section class="panel stack">
        <p class="eyebrow">Remote Sync</p>
        <h2>Access ready</h2>
        <p>Prefix <code>${escapeHTML(input.remoteAccess.capability.prefix)}</code></p>
        <p>Actions <code>${escapeHTML(input.remoteAccess.capability.allowedActions.join(", "))}</code></p>
        <p>Expires ${escapeHTML(input.remoteAccess.capability.expiresAt)}</p>
      </section>
      `
        : ""}

      ${input.error ? `<p class="error">${escapeHTML(input.error)}</p>` : ""}

      <section class="vault-grid">
        <section class="panel stack">
          <p class="eyebrow">Save Secret</p>
          <h2>Add one login</h2>
          <form class="stack" method="post" action="/vault/login">
            <label class="field"><span>Title</span><input name="title" type="text" value="GitHub" required /></label>
            <label class="field"><span>Username</span><input name="username" type="text" value="alice" required /></label>
            <label class="field"><span>URL</span><input name="url" type="url" value="https://github.com/login" required /></label>
            <label class="field"><span>Password</span><input name="password" type="text" value="ghp-secret-123" required /></label>
            <button class="button button-primary" type="submit">Save secret</button>
          </form>
        </section>

        <section class="panel stack">
          <p class="eyebrow">Vault Items</p>
          <h2>Replay-backed records</h2>
          ${itemsMarkup}
        </section>
      </section>

      <section class="panel">
        ${detailMarkup}
      </section>
    `,
  });
}

async function defaultResolveIdentity(): Promise<ClientIdentity> {
  const controlPlaneURL =
    process.env.SAFE_WEB_CONTROL_PLANE_URL ??
    process.env.SAFE_CONTROL_PLANE_URL;
  const deviceId =
    process.env.SAFE_WEB_DEVICE_ID ??
    process.env.SAFE_DEVICE_ID ??
    process.env.SAFE_DEV_DEVICE_ID ??
    "dev-web-001";
  const oauthToken =
    process.env.SAFE_WEB_OAUTH_ACCESS_TOKEN ??
    process.env.SAFE_OAUTH_ACCESS_TOKEN;

  if (controlPlaneURL && oauthToken) {
    try {
      const response = await fetch(new URL("/v1/session", controlPlaneURL), {
        headers: {
          authorization: `Bearer ${oauthToken}`,
        },
      });
      if (response.ok) {
        const payload = (await response.json()) as ControlPlaneSession;
        if (payload.accountId && payload.env) {
          return {
            accountId: payload.accountId,
            deviceId,
            env: payload.env,
          };
        }
      }
    } catch {
      // Fall back to local defaults when the control plane is not running.
    }
  }

  return {
    accountId:
      process.env.SAFE_OAUTH_ACCOUNT_ID ??
      process.env.SAFE_DEV_ACCOUNT_ID ??
      "acct-dev-001",
    deviceId,
    env: process.env.SAFE_ENV ?? "development",
  };
}

async function defaultResolveRemoteAccess(
  identity: ClientIdentity,
): Promise<AccountAccessResponse | null> {
  const controlPlaneURL =
    process.env.SAFE_WEB_CONTROL_PLANE_URL ??
    process.env.SAFE_CONTROL_PLANE_URL;
  const oauthToken =
    process.env.SAFE_WEB_OAUTH_ACCESS_TOKEN ??
    process.env.SAFE_OAUTH_ACCESS_TOKEN;

  if (!controlPlaneURL || !oauthToken) {
    return null;
  }

  try {
    const response = await fetch(new URL("/v1/access/account", controlPlaneURL), {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        authorization: `Bearer ${oauthToken}`,
      },
      body: JSON.stringify({
        accountId: identity.accountId,
        deviceId: identity.deviceId,
      }),
    });
    if (!response.ok) {
      return null;
    }

    const payload = (await response.json()) as AccountAccessResponse;
    if (
      payload.token &&
      payload.capability?.prefix &&
      Array.isArray(payload.capability.allowedActions) &&
      payload.capability.allowedActions.length > 0 &&
      payload.capability.expiresAt
    ) {
      return payload;
    }
  } catch {
    // Fall back to local-only mode when remote access is unavailable.
  }

  return null;
}

async function readForm(request: IncomingMessage): Promise<URLSearchParams> {
  const chunks: Buffer[] = [];
  for await (const chunk of request) {
    chunks.push(Buffer.isBuffer(chunk) ? chunk : Buffer.from(chunk));
  }

  return new URLSearchParams(Buffer.concat(chunks).toString("utf8"));
}

function writeHTML(
  response: ServerResponse,
  statusCode: number,
  body: string,
): void {
  response.statusCode = statusCode;
  response.setHeader("Content-Type", "text/html; charset=utf-8");
  response.end(body);
}

function redirect(response: ServerResponse, location: string): void {
  response.statusCode = 303;
  response.setHeader("Location", location);
  response.end();
}

function parseCookies(cookieHeader: string): Record<string, string> {
  return Object.fromEntries(
    cookieHeader
      .split(";")
      .map((part) => part.trim())
      .filter((part) => part.includes("="))
      .map((part) => {
        const separator = part.indexOf("=");
        return [part.slice(0, separator), decodeURIComponent(part.slice(separator + 1))];
      }),
  );
}

function serializeCookie(name: string, value: string): string {
  return `${name}=${encodeURIComponent(value)}; Path=/; HttpOnly; SameSite=Lax`;
}

function getSafeLoginDetail(
  workspace: Parameters<typeof getVaultLoginCredentialDetail>[0]["workspace"],
  secretMaterial: Parameters<typeof getVaultLoginCredentialDetail>[0]["secretMaterial"],
  itemId: string,
) {
  try {
    return getVaultLoginCredentialDetail({
      workspace,
      itemId,
      secretMaterial,
    });
  } catch {
    return null;
  }
}

function renderPage(input: {
  title: string;
  body: string;
}): string {
  return `<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
    <title>${escapeHTML(input.title)}</title>
    <style>
      :root {
        color-scheme: light;
        --bg: #f4efe5;
        --bg-strong: #ece2d3;
        --panel: rgba(255, 252, 246, 0.88);
        --border: rgba(93, 70, 44, 0.16);
        --text: #1f1a14;
        --muted: #6c5a48;
        --accent: #a04d1f;
        --accent-strong: #7f3814;
        --shadow: 0 24px 60px rgba(77, 55, 36, 0.12);
      }

      * { box-sizing: border-box; }
      body {
        margin: 0;
        min-height: 100vh;
        background:
          radial-gradient(circle at top left, rgba(209, 151, 83, 0.22), transparent 35%),
          radial-gradient(circle at top right, rgba(92, 128, 109, 0.15), transparent 32%),
          linear-gradient(180deg, var(--bg), #f8f4ee);
        color: var(--text);
        font-family: "Avenir Next", "Segoe UI", sans-serif;
      }

      h1, h2 {
        margin: 0;
        font-family: "Iowan Old Style", "Palatino Linotype", "Book Antiqua", Georgia, serif;
        font-weight: 700;
        line-height: 1.05;
      }

      p, li, dd, dt, span, a, button, input, code { font-size: 1rem; }

      main {
        max-width: 1120px;
        margin: 0 auto;
        padding: 48px 20px 72px;
      }

      .stack > * + * { margin-top: 1rem; }
      .hero, .vault-grid, .stats-grid, .meta-grid, .detail-grid {
        display: grid;
        gap: 18px;
      }

      .hero { grid-template-columns: 1.25fr 0.95fr; align-items: stretch; }
      .hero-copy, .hero-panel, .panel, .stat-card, .detail-card {
        background: var(--panel);
        border: 1px solid var(--border);
        border-radius: 24px;
        box-shadow: var(--shadow);
      }

      .hero-copy, .hero-panel, .panel, .detail-card { padding: 28px; }
      .hero-copy h1 { font-size: clamp(2.8rem, 6vw, 4.7rem); max-width: 12ch; }
      .hero-panel h2, .panel h2 { font-size: 1.8rem; }
      .lede { color: var(--muted); max-width: 58ch; line-height: 1.6; }
      .eyebrow {
        margin: 0;
        color: var(--accent);
        font-size: 0.78rem;
        font-weight: 700;
        letter-spacing: 0.16em;
        text-transform: uppercase;
      }

      .steps {
        margin: 0;
        padding-left: 1.25rem;
        color: var(--muted);
        line-height: 1.6;
      }

      .button, .link {
        display: inline-flex;
        align-items: center;
        justify-content: center;
        text-decoration: none;
        border-radius: 999px;
        border: 1px solid transparent;
        min-height: 44px;
        padding: 0 18px;
        font-weight: 700;
      }

      .button-primary {
        background: var(--accent);
        color: white;
      }

      .button-primary:hover { background: var(--accent-strong); }

      .button-secondary {
        background: transparent;
        border-color: var(--border);
        color: var(--text);
      }

      .vault-header, .actions {
        display: flex;
        gap: 12px;
        align-items: center;
        justify-content: space-between;
        flex-wrap: wrap;
      }

      .stats-grid { grid-template-columns: repeat(3, minmax(0, 1fr)); margin: 22px 0; }
      .stat-card { padding: 20px; }
      .stat-label {
        display: block;
        color: var(--muted);
        margin-bottom: 8px;
        text-transform: uppercase;
        letter-spacing: 0.12em;
        font-size: 0.72rem;
      }
      .stat-card strong { font-size: 2rem; }

      .vault-grid { grid-template-columns: 1fr 1fr; }
      .field { display: block; }
      .field span {
        display: block;
        margin-bottom: 8px;
        color: var(--muted);
        font-weight: 700;
      }

      input {
        width: 100%;
        border-radius: 16px;
        border: 1px solid rgba(93, 70, 44, 0.22);
        padding: 14px 16px;
        background: rgba(255, 255, 255, 0.9);
      }

      .item-list {
        list-style: none;
        margin: 0;
        padding: 0;
        display: grid;
        gap: 10px;
      }

      .item-link {
        display: block;
        padding: 14px 16px;
        border-radius: 18px;
        color: inherit;
        text-decoration: none;
        background: rgba(247, 239, 228, 0.76);
        border: 1px solid rgba(93, 70, 44, 0.1);
      }

      .item-title { display: block; font-weight: 700; }
      .item-summary, .empty, .error, dt { color: var(--muted); }
      .meta-grid, .detail-grid {
        grid-template-columns: repeat(2, minmax(0, 1fr));
      }

      dt {
        font-size: 0.76rem;
        text-transform: uppercase;
        letter-spacing: 0.08em;
        margin-bottom: 6px;
      }

      dd { margin: 0; font-weight: 700; }
      code {
        display: inline-block;
        padding: 4px 8px;
        border-radius: 10px;
        background: rgba(93, 70, 44, 0.08);
      }

      .error {
        margin: 0;
        padding: 12px 14px;
        border-radius: 16px;
        background: rgba(194, 73, 31, 0.08);
        border: 1px solid rgba(194, 73, 31, 0.16);
      }

      @media (max-width: 820px) {
        .hero, .vault-grid, .stats-grid, .meta-grid, .detail-grid {
          grid-template-columns: 1fr;
        }

        main { padding: 28px 16px 48px; }
        .hero-copy h1 { font-size: clamp(2.4rem, 11vw, 3.3rem); }
      }
    </style>
  </head>
  <body>
    <main>${input.body}</main>
  </body>
</html>`;
}

function escapeHTML(value: string): string {
  return value
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#39;");
}
