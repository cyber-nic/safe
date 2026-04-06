import type { IncomingMessage, ServerResponse } from "node:http";
import http from "node:http";
import path from "node:path";
import { randomUUID } from "node:crypto";
import { execFile } from "node:child_process";
import { promisify } from "node:util";

import {
  addApiKeyToVaultWorkspace,
  addLoginToVaultWorkspace,
  addNoteToVaultWorkspace,
  addSshKeyToVaultWorkspace,
  addTotpToVaultWorkspace,
  createUnlockedVaultWorkspace,
  deleteItemFromVaultWorkspace,
  getVaultItemDetail,
  getVaultLoginCredentialDetail,
  listDeletedVaultItems,
  restoreItemToVaultWorkspace,
  updateApiKeyInVaultWorkspace,
  updateLoginInVaultWorkspace,
  updateNoteInVaultWorkspace,
  updateSshKeyInVaultWorkspace,
  updateTotpInVaultWorkspace,
} from "./index.ts";
import {
  createLocalRuntimeStore,
  type ClientIdentity,
  type PreparedFirstUse,
  type UnlockedLocalVault,
} from "./local-runtime.ts";

type SessionState = {
  identity: ClientIdentity | null;
  remoteAccess: AccountAccessResponse | null;
  accountKey: Buffer | null;
  pendingOnboarding: PreparedFirstUse | null;
  pendingPassword: string | null;
  vaultPassword: string | null;
  flash: string | null;
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

type SafeCommandResult = {
  stdout: string;
  stderr: string;
};

type SafeCommandRunner = (
  args: string[],
  env: NodeJS.ProcessEnv,
) => Promise<SafeCommandResult>;

type DeviceRecord = {
  schemaVersion: number;
  accountId: string;
  deviceId: string;
  label: string;
  deviceType: string;
  signingPublicKey: string;
  encryptionPublicKey: string;
  createdAt: string;
  status: string;
};

type EnrollmentRequest = {
  schemaVersion: number;
  accountId: string;
  deviceId: string;
  label: string;
  deviceType: string;
  encryptionPublicKey: string;
  requestedAt: string;
};

export type WebClientServerOptions = {
  dataDir?: string;
  now?: () => Date;
  resolveIdentity?: () => Promise<ClientIdentity>;
  resolveRemoteAccess?: (
    identity: ClientIdentity,
  ) => Promise<AccountAccessResponse | null>;
  runSafeCommand?: SafeCommandRunner;
};

export function createWebClientServer(
  options: WebClientServerOptions = {},
): http.RequestListener {
  const runSafeCommand = options.runSafeCommand ?? defaultRunSafeCommand;
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
        if (session.accountKey) {
          redirect(response, "/vault");
          return;
        }
        if (session.pendingOnboarding) {
          redirect(response, "/onboarding/recovery");
          return;
        }
        redirect(response, "/unlock");
        return;
      }

      writeHTML(response, 200, renderHomePage());
      return;
    }

    if (request.method === "POST" && url.pathname === "/identify") {
      session.identity = await resolveIdentity();
      session.remoteAccess = await resolveRemoteAccess(session.identity);
      session.accountKey = null;
      session.pendingOnboarding = null;
      session.pendingPassword = null;
      session.vaultPassword = null;
      session.flash = null;
      redirect(response, "/unlock");
      return;
    }

    if (request.method === "GET" && url.pathname === "/unlock") {
      if (!session.identity) {
        redirect(response, "/");
        return;
      }
      if (session.pendingOnboarding) {
        redirect(response, "/onboarding/recovery");
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
      if (session.pendingOnboarding) {
        redirect(response, "/onboarding/recovery");
        return;
      }

      const form = await readForm(request);
      const password = form.get("password")?.trim() ?? "";
      const firstUse = !(await store.hasUnlockRecord(session.identity.accountId));

      try {
        if (firstUse) {
          session.pendingOnboarding = store.prepareFirstUse(session.identity, password);
          session.pendingPassword = password;
          session.accountKey = null;
          redirect(response, "/onboarding/recovery");
          return;
        }

        const unlocked = await store.unlock(session.identity, password);
        session.pendingOnboarding = null;
        session.pendingPassword = null;
        session.accountKey = unlocked.accountKey;
        session.vaultPassword = password;
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

    if (request.method === "GET" && url.pathname === "/onboarding/recovery") {
      if (!session.identity || !session.pendingOnboarding) {
        redirect(response, session.identity ? "/unlock" : "/");
        return;
      }

      writeHTML(
        response,
        200,
        renderRecoveryPage({
          identity: session.identity,
          recoveryMnemonic: session.pendingOnboarding.recoveryMnemonic,
        }),
      );
      return;
    }

    if (request.method === "POST" && url.pathname === "/onboarding/recovery") {
      if (!session.identity || !session.pendingOnboarding) {
        redirect(response, session.identity ? "/unlock" : "/");
        return;
      }

      const form = await readForm(request);
      if (form.get("confirmed") !== "yes") {
        writeHTML(
          response,
          200,
          renderRecoveryPage({
            identity: session.identity,
            recoveryMnemonic: session.pendingOnboarding.recoveryMnemonic,
            error: "You must confirm that the recovery key is stored before continuing.",
          }),
        );
        return;
      }

      const unlocked = await store.finalizeFirstUse(
        session.identity,
        session.pendingOnboarding,
      );
      session.accountKey = unlocked.accountKey;
      session.vaultPassword = session.pendingPassword;
      session.pendingOnboarding = null;
      session.pendingPassword = null;
      redirect(response, "/vault");
      return;
    }

    if (request.method === "POST" && url.pathname === "/lock") {
      session.accountKey = null;
      session.vaultPassword = null;
      redirect(response, "/unlock");
      return;
    }

    if (request.method === "POST" && url.pathname === "/logout") {
      session.identity = null;
      session.accountKey = null;
      session.pendingOnboarding = null;
      session.pendingPassword = null;
      session.vaultPassword = null;
      session.flash = null;
      redirect(response, "/");
      return;
    }

    if (request.method === "GET" && url.pathname === "/vault") {
      if (!session.identity || !session.accountKey) {
        redirect(response, "/unlock");
        return;
      }

      const unlocked = await loadUnlockedVaultForQuery(
        session,
        readVaultQuery(url),
      );
      const flash = session.flash;
      session.flash = null;
      const remoteState = await loadRemoteState(session);
      writeHTML(
        response,
        200,
        renderVaultPage({
          identity: session.identity,
          remoteAccess: session.remoteAccess,
          itemId: url.searchParams.get("item"),
          unlocked,
          devices: remoteState.devices,
          pendingEnrollments: remoteState.pendingEnrollments,
          syncError: remoteState.error,
          flash,
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
      const unlocked = await loadUnlockedVaultForQuery(
        session,
        readVaultQueryFromForm(form),
      );

      try {
        const result = await addLoginToVaultWorkspace(
          buildLoginMutationInput(form, {
            workspace: unlocked.workspace,
            secretMaterial: unlocked.secretMaterial,
            deviceId: session.identity.deviceId,
            at: now(),
          }),
        );
        await persistVaultMutation(session, result);
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
            devices: [],
            pendingEnrollments: [],
            syncError: null,
            flash: null,
            error:
              error instanceof Error
                ? error.message
                : "Safe could not save that secret.",
          }),
        );
        return;
      }
    }

    if (request.method === "POST" && url.pathname === "/vault/items") {
      if (!session.identity || !session.accountKey) {
        redirect(response, "/unlock");
        return;
      }

      const form = await readForm(request);
      const unlocked = await loadUnlockedVaultForQuery(
        session,
        readVaultQueryFromForm(form),
      );

      try {
        const result = await createVaultItemFromForm(
          form,
          unlocked,
          session.identity.deviceId,
          now(),
        );
        await persistVaultMutation(session, result);
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
            itemId: form.get("itemId")?.trim() ?? null,
            devices: [],
            pendingEnrollments: [],
            syncError: null,
            flash: null,
            error:
              error instanceof Error
                ? error.message
                : "Safe could not save that vault item.",
          }),
        );
        return;
      }
    }

    if (request.method === "POST" && url.pathname === "/vault/items/update") {
      if (!session.identity || !session.accountKey) {
        redirect(response, "/unlock");
        return;
      }

      const form = await readForm(request);
      const itemId = form.get("itemId")?.trim() ?? "";
      const unlocked = await loadUnlockedVaultForQuery(
        session,
        readVaultQueryFromForm(form),
      );

      try {
        const result = await updateVaultItemFromForm(
          form,
          unlocked,
          session.identity.deviceId,
          now(),
        );
        await persistVaultMutation(session, result);
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
            itemId: itemId || null,
            devices: [],
            pendingEnrollments: [],
            syncError: null,
            flash: null,
            error:
              error instanceof Error
                ? error.message
                : "Safe could not update that vault item.",
          }),
        );
        return;
      }
    }

    if (request.method === "POST" && url.pathname === "/vault/items/delete") {
      if (!session.identity || !session.accountKey) {
        redirect(response, "/unlock");
        return;
      }

      const form = await readForm(request);
      const itemId = form.get("itemId")?.trim() ?? "";
      const unlocked = await loadUnlockedVaultForQuery(
        session,
        readVaultQueryFromForm(form),
      );

      try {
        const result = await deleteItemFromVaultWorkspace({
          workspace: unlocked.workspace,
          secretMaterial: unlocked.secretMaterial,
          deviceId: session.identity.deviceId,
          itemId,
          at: now(),
        });
        await persistVaultMutation(session, result);
        redirect(response, "/vault?deleted=1");
        return;
      } catch (error) {
        writeHTML(
          response,
          200,
          renderVaultPage({
            identity: session.identity,
            remoteAccess: session.remoteAccess,
            unlocked,
            itemId: itemId || null,
            devices: [],
            pendingEnrollments: [],
            syncError: null,
            flash: null,
            error:
              error instanceof Error
                ? error.message
                : "Safe could not delete that vault item.",
          }),
        );
        return;
      }
    }

    if (request.method === "POST" && url.pathname === "/vault/items/restore") {
      if (!session.identity || !session.accountKey) {
        redirect(response, "/unlock");
        return;
      }

      const form = await readForm(request);
      const itemId = form.get("itemId")?.trim() ?? "";
      const unlocked = await loadUnlockedVaultForQuery(
        session,
        readVaultQueryFromForm(form),
      );

      try {
        const result = await restoreItemToVaultWorkspace({
          workspace: unlocked.workspace,
          secretMaterial: unlocked.secretMaterial,
          deviceId: session.identity.deviceId,
          itemId,
          at: now(),
        });
        await persistVaultMutation(session, result);
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
            itemId: itemId || null,
            devices: [],
            pendingEnrollments: [],
            syncError: null,
            flash: null,
            error:
              error instanceof Error
                ? error.message
                : "Safe could not restore that vault item.",
          }),
        );
        return;
      }
    }

    if (request.method === "POST" && url.pathname === "/vault/sync/push") {
      if (!session.identity || !session.accountKey || !session.vaultPassword) {
        redirect(response, "/unlock");
        return;
      }

      try {
        await runSafeVaultCommand(session, ["sync", "push"]);
        session.flash = "Sync push completed against the account remote path.";
        redirect(response, "/vault");
        return;
      } catch (error) {
        await renderVaultErrorPage(
          response,
          session,
          null,
          error instanceof Error ? error.message : "Sync push failed.",
        );
        return;
      }
    }

    if (request.method === "POST" && url.pathname === "/vault/sync/pull") {
      if (!session.identity || !session.accountKey || !session.vaultPassword) {
        redirect(response, "/unlock");
        return;
      }

      try {
        await runSafeVaultCommand(session, ["sync", "pull"]);
        session.flash = "Sync pull completed and reloaded the local vault snapshot.";
        redirect(response, "/vault");
        return;
      } catch (error) {
        await renderVaultErrorPage(
          response,
          session,
          null,
          error instanceof Error ? error.message : "Sync pull failed.",
        );
        return;
      }
    }

    if (request.method === "POST" && url.pathname === "/vault/devices/approve") {
      if (!session.identity || !session.accountKey || !session.vaultPassword) {
        redirect(response, "/unlock");
        return;
      }

      const form = await readForm(request);
      const deviceId = form.get("deviceId")?.trim() ?? "";
      if (deviceId === "") {
        writeHTML(
          response,
          400,
          renderPage({
            title: "Safe",
            body: `
              <section class="panel stack">
                <p class="eyebrow">Bad Request</p>
                <h1>Missing device ID</h1>
                <p><a class="link" href="/vault">Return to vault</a></p>
              </section>
            `,
          }),
        );
        return;
      }

      try {
        await runSafeVaultCommand(session, ["device", "approve", deviceId]);
        session.flash = `Approved enrollment request for ${deviceId}.`;
        redirect(response, "/vault");
        return;
      } catch (error) {
        await renderVaultErrorPage(
          response,
          session,
          null,
          error instanceof Error ? error.message : "Device approval failed.",
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
      pendingOnboarding: null,
      pendingPassword: null,
      vaultPassword: null,
      flash: null,
    };
    sessions.set(sessionId, session);
    response.setHeader("Set-Cookie", serializeCookie("safe_web_session", sessionId));
    return session;
  }

  async function loadUnlockedVaultForQuery(
    session: SessionState,
    query: ReturnType<typeof readVaultQuery>,
  ): Promise<UnlockedLocalVault> {
    const unlocked = await store.loadUnlockedWithAccountKey(
      session.identity!,
      session.accountKey!,
    );
    const workspace = await createUnlockedVaultWorkspace({
      accountConfig: unlocked.workspace.accountConfig,
      head: unlocked.workspace.head,
      events: unlocked.workspace.events,
      starterRecords: unlocked.workspace.starterRecords,
      query,
      secretMaterial: unlocked.secretMaterial,
      at: now(),
    });

    return {
      ...unlocked,
      workspace,
    };
  }

  async function persistVaultMutation(
    session: SessionState,
    result: {
      workspace: UnlockedLocalVault["workspace"];
      secretMaterial: UnlockedLocalVault["secretMaterial"];
    },
  ): Promise<void> {
    await store.persistUnlockedVault(session.identity!, {
      workspace: result.workspace,
      secretMaterial: result.secretMaterial,
      accountKey: session.accountKey!,
    });
  }

  async function runSafeVaultCommand(
    session: SessionState,
    args: string[],
  ): Promise<string> {
    const { stdout, stderr } = await runSafeCommand(
      ["--json", ...args],
      buildSafeCommandEnv(dataDir, session),
    );
    if (stderr.trim() !== "") {
      return stdout;
    }
    return stdout;
  }

  async function loadRemoteState(session: SessionState): Promise<{
    devices: DeviceRecord[];
    pendingEnrollments: EnrollmentRequest[];
    error: string | null;
  }> {
    if (!session.identity || !session.accountKey || !session.vaultPassword || !session.remoteAccess) {
      return {
        devices: [],
        pendingEnrollments: [],
        error: null,
      };
    }

    try {
      const [devicesRaw, pendingRaw] = await Promise.all([
        runSafeVaultCommand(session, ["device", "list"]),
        runSafeVaultCommand(session, ["device", "pending"]),
      ]);
      return {
        devices: parseDeviceRecords(devicesRaw),
        pendingEnrollments: parseEnrollmentRequests(pendingRaw),
        error: null,
      };
    } catch (error) {
      return {
        devices: [],
        pendingEnrollments: [],
        error: error instanceof Error ? error.message : "Remote sync state is unavailable.",
      };
    }
  }

  async function renderVaultErrorPage(
    response: ServerResponse,
    session: SessionState,
    itemId: string | null,
    error: string,
  ): Promise<void> {
    const unlocked = await loadUnlockedVaultForQuery(session, {});
    const remoteState = await loadRemoteState(session);
    writeHTML(
      response,
      200,
      renderVaultPage({
        identity: session.identity!,
        remoteAccess: session.remoteAccess,
        unlocked,
        itemId,
        devices: remoteState.devices,
        pendingEnrollments: remoteState.pendingEnrollments,
        syncError: error,
        flash: null,
      }),
    );
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

function renderRecoveryPage(input: {
  identity: ClientIdentity;
  recoveryMnemonic: string;
  error?: string;
}): string {
  return renderPage({
    title: "Safe Recovery Key",
    body: `
      <section class="panel stack">
        <p class="eyebrow">Recovery Key</p>
        <h1>Store this recovery key before entering the vault.</h1>
        <p>
          Safe will only show this mnemonic during account creation. Store it offline before you continue.
        </p>
        ${input.error ? `<p class="error">${escapeHTML(input.error)}</p>` : ""}
        <dl class="meta-grid">
          <div><dt>Account</dt><dd>${escapeHTML(input.identity.accountId)}</dd></div>
          <div><dt>Device</dt><dd>${escapeHTML(input.identity.deviceId)}</dd></div>
        </dl>
        <section class="panel stack">
          <p class="eyebrow">24-word mnemonic</p>
          <p class="recovery-phrase">${escapeHTML(input.recoveryMnemonic)}</p>
        </section>
        <form class="stack" method="post" action="/onboarding/recovery">
          <label class="checkbox">
            <input name="confirmed" type="checkbox" value="yes" />
            <span>I stored this recovery key somewhere durable and offline.</span>
          </label>
          <div class="actions">
            <button class="button button-primary" type="submit">Continue to vault</button>
          </div>
        </form>
      </section>
    `,
  });
}

function renderVaultPage(input: {
  identity: ClientIdentity;
  remoteAccess: AccountAccessResponse | null;
  unlocked: UnlockedLocalVault;
  itemId: string | null;
  devices: DeviceRecord[];
  pendingEnrollments: EnrollmentRequest[];
  syncError: string | null;
  flash: string | null;
  error?: string;
}): string {
  const selectedItemId =
    input.itemId ?? input.unlocked.workspace.items[0]?.id ?? null;
  const selectedDetail =
    selectedItemId === null
      ? null
      : getVaultItemDetail(input.unlocked.workspace, selectedItemId);
  const selectedRecord =
    selectedItemId === null
      ? null
      : input.unlocked.workspace.itemRecords.find((record) => record.item.id === selectedItemId) ??
        null;
  const selectedLogin =
    selectedRecord?.item.kind !== "login"
      ? null
      : getSafeLoginDetail(
          input.unlocked.workspace,
          input.unlocked.secretMaterial,
          selectedRecord.item.id,
        );
  const selectedAuthenticator =
    selectedRecord?.item.kind === "totp"
      ? input.unlocked.workspace.authenticators.find(
          (card) => card.id === selectedRecord.item.id,
        ) ?? null
      : null;
  const deletedItems = listDeletedVaultItems(input.unlocked.workspace);
  const itemsMarkup =
    input.unlocked.workspace.items.length === 0
      ? `<p class="empty">No secrets yet. No vault items match the current filter.</p>`
      : `
        <ul class="item-list">
          ${input.unlocked.workspace.items
            .map((item) => `
                <li>
                  <a class="item-link${item.id === selectedItemId ? " is-selected" : ""}" href="/vault?${buildVaultLocation({
                    item: item.id,
                    q: input.unlocked.workspace.query.text,
                    kind: input.unlocked.workspace.query.kind,
                    tag: input.unlocked.workspace.query.tag,
                  })}">
                    <span class="item-kicker">${escapeHTML(item.kind)}</span>
                    <span class="item-title">${escapeHTML(item.title)}</span>
                    <span class="item-summary">${escapeHTML(item.summary)}</span>
                    ${item.matchedFields.length > 0
                      ? `<span class="item-meta">Matched ${escapeHTML(item.matchedFields.join(", "))}</span>`
                      : ""}
                  </a>
                </li>
              `)
            .join("")}
        </ul>
      `;
  const detailMarkup =
    selectedDetail === null
      ? `
        <div class="detail-empty">
          <p class="eyebrow">Vault Detail</p>
          <h2>No item selected</h2>
          <p>Create or filter vault items, then select one to inspect and edit it.</p>
        </div>
      `
      : renderVaultDetail({
          detail: selectedDetail,
          record: selectedRecord,
          login: selectedLogin,
          authenticator: selectedAuthenticator,
          query: input.unlocked.workspace.query,
        });

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
        <div class="actions">
          <form method="post" action="/vault/sync/push">
            <button class="button button-primary" type="submit">Sync push</button>
          </form>
          <form method="post" action="/vault/sync/pull">
            <button class="button button-secondary" type="submit">Sync pull</button>
          </form>
        </div>
      </section>
      `
        : ""}

      ${input.flash ? `<p class="success">${escapeHTML(input.flash)}</p>` : ""}
      ${input.syncError ? `<p class="error">${escapeHTML(input.syncError)}</p>` : ""}
      ${input.error ? `<p class="error">${escapeHTML(input.error)}</p>` : ""}

      <section class="panel stack">
        <p class="eyebrow">Search</p>
        <h2>Filter the vault</h2>
        <form class="filter-grid" method="get" action="/vault">
          <label class="field">
            <span>Search</span>
            <input name="q" type="search" value="${escapeHTML(input.unlocked.workspace.query.text ?? "")}" />
          </label>
          <label class="field">
            <span>Kind</span>
            <select name="kind">
              ${input.unlocked.workspace.availableKinds
                .map(
                  (kind) =>
                    `<option value="${escapeHTML(kind)}"${(input.unlocked.workspace.query.kind ?? "all") === kind ? " selected" : ""}>${escapeHTML(kind === "all" ? "all kinds" : kind)}</option>`,
                )
                .join("")}
            </select>
          </label>
          <label class="field">
            <span>Tag</span>
            <select name="tag">
              <option value="all">all tags</option>
              ${input.unlocked.workspace.availableTags
                .map(
                  (tag) =>
                    `<option value="${escapeHTML(tag)}"${(input.unlocked.workspace.query.tag ?? "all") === tag ? " selected" : ""}>${escapeHTML(tag)}</option>`,
                )
                .join("")}
            </select>
          </label>
          <div class="actions">
            <button class="button button-primary" type="submit">Apply filters</button>
            <a class="button button-secondary" href="/vault">Clear</a>
          </div>
        </form>
      </section>

      <section class="vault-grid">
        <section class="panel stack">
          <p class="eyebrow">Create</p>
          <h2>Add vault item</h2>
          ${renderCreateForm(input.unlocked.workspace.query)}
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

      <section class="panel stack">
        <p class="eyebrow">Deleted Items</p>
        <h2>Recently removed</h2>
        ${deletedItems.length === 0
          ? `<p class="empty">No deleted items yet.</p>`
          : `
            <ul class="item-list">
              ${deletedItems
                .map(
                  (item) => `
                    <li class="deleted-row">
                      <a class="item-link" href="/vault?${buildVaultLocation({
                        item: item.id,
                        q: input.unlocked.workspace.query.text,
                        kind: input.unlocked.workspace.query.kind,
                        tag: input.unlocked.workspace.query.tag,
                      })}">
                        <span class="item-kicker">${escapeHTML(item.kind)}</span>
                        <span class="item-title">${escapeHTML(item.title)}</span>
                        <span class="item-summary">${escapeHTML(item.summary)}</span>
                      </a>
                      <form method="post" action="/vault/items/restore">
                        ${renderVaultQueryHiddenFields(input.unlocked.workspace.query)}
                        <input name="itemId" type="hidden" value="${escapeHTML(item.id)}" />
                        <button class="button button-secondary" type="submit">Restore</button>
                      </form>
                    </li>
                  `,
                )
                .join("")}
            </ul>
          `}
      </section>

      <section class="vault-grid">
        <section class="panel stack">
          <p class="eyebrow">Devices</p>
          <h2>Enrolled devices</h2>
          ${input.devices.length === 0
            ? `<p class="empty">No remote device records visible yet.</p>`
            : `
              <ul class="item-list">
                ${input.devices
                  .map(
                    (device) => `
                      <li class="device-row">
                        <div class="item-link">
                          <span class="item-kicker">${escapeHTML(device.deviceType)}</span>
                          <span class="item-title">${escapeHTML(device.label)}</span>
                          <span class="item-summary">${escapeHTML(device.deviceId)}</span>
                          <span class="item-meta">Status ${escapeHTML(device.status)} • Created ${escapeHTML(device.createdAt)}</span>
                        </div>
                      </li>
                    `,
                  )
                  .join("")}
              </ul>
            `}
        </section>

        <section class="panel stack">
          <p class="eyebrow">Approvals</p>
          <h2>Pending enrollment requests</h2>
          ${input.pendingEnrollments.length === 0
            ? `<p class="empty">No pending device enrollment requests.</p>`
            : `
              <ul class="item-list">
                ${input.pendingEnrollments
                  .map(
                    (request) => `
                      <li class="deleted-row">
                        <div class="item-link">
                          <span class="item-kicker">${escapeHTML(request.deviceType)}</span>
                          <span class="item-title">${escapeHTML(request.label)}</span>
                          <span class="item-summary">${escapeHTML(request.deviceId)}</span>
                          <span class="item-meta">Requested ${escapeHTML(request.requestedAt)}</span>
                        </div>
                        <form method="post" action="/vault/devices/approve">
                          <input name="deviceId" type="hidden" value="${escapeHTML(request.deviceId)}" />
                          <button class="button button-secondary" type="submit">Approve</button>
                        </form>
                      </li>
                    `,
                  )
                  .join("")}
              </ul>
            `}
        </section>
      </section>
    `,
  });
}

function renderVaultDetail(input: {
  detail: ReturnType<typeof getVaultItemDetail>;
  record: UnlockedLocalVault["workspace"]["itemRecords"][number] | null;
  login: ReturnType<typeof getSafeLoginDetail>;
  authenticator: UnlockedLocalVault["workspace"]["authenticators"][number] | null;
  query: {
    text?: string;
    kind?: string;
    tag?: string;
  };
}): string {
  const tagValue = input.detail.tags.length > 0 ? input.detail.tags.join(", ") : "none";

  if (input.detail.status === "deleted") {
    return `
      <div class="detail-card stack">
        <p class="eyebrow">Deleted Item</p>
        <h2>${escapeHTML(input.detail.title)}</h2>
        <p>${escapeHTML(input.detail.summary)}</p>
        <dl class="detail-grid">
          <div><dt>Kind</dt><dd>${escapeHTML(input.detail.kind)}</dd></div>
          <div><dt>Status</dt><dd>${escapeHTML(input.detail.status)}</dd></div>
          <div><dt>Tags</dt><dd>${escapeHTML(tagValue)}</dd></div>
          <div><dt>History</dt><dd>${input.detail.history.length}</dd></div>
        </dl>
        <form method="post" action="/vault/items/restore">
          ${renderVaultQueryHiddenFields(input.query)}
          <input name="itemId" type="hidden" value="${escapeHTML(input.detail.id)}" />
          <button class="button button-primary" type="submit">Restore item</button>
        </form>
      </div>
    `;
  }

  const deleteForm = `
    <form method="post" action="/vault/items/delete">
      ${renderVaultQueryHiddenFields(input.query)}
      <input name="itemId" type="hidden" value="${escapeHTML(input.detail.id)}" />
      <button class="button button-secondary" type="submit">Delete item</button>
    </form>
  `;

  if (input.record?.item.kind === "login" && input.login) {
    return `
      <div class="detail-card stack">
        <p class="eyebrow">Selected Login</p>
        <h2>${escapeHTML(input.login.title)}</h2>
        <dl class="detail-grid">
          <div><dt>Kind</dt><dd>${escapeHTML(input.detail.kind)}</dd></div>
          <div><dt>Username</dt><dd>${escapeHTML(input.login.username)}</dd></div>
          <div><dt>URL</dt><dd>${escapeHTML(input.login.primaryURL ?? "n/a")}</dd></div>
          <div><dt>Password</dt><dd><code>${escapeHTML(input.login.password ?? "locked")}</code></dd></div>
          <div><dt>Status</dt><dd>${escapeHTML(input.login.passwordStatus)}</dd></div>
          <div><dt>Authenticator</dt><dd>${escapeHTML(input.login.relatedAuthenticatorTitle ?? "none")}</dd></div>
          <div><dt>TOTP Code</dt><dd><code>${escapeHTML(input.login.relatedAuthenticatorCode ?? "n/a")}</code></dd></div>
          <div><dt>Tags</dt><dd>${escapeHTML(tagValue)}</dd></div>
        </dl>
        ${renderEditForm({
          itemId: input.record.item.id,
          item: input.record.item,
          query: input.query,
        })}
        ${deleteForm}
      </div>
    `;
  }

  if (input.record?.item.kind === "totp" && input.authenticator) {
    return `
      <div class="detail-card stack">
        <p class="eyebrow">Selected Authenticator</p>
        <h2>${escapeHTML(input.authenticator.title)}</h2>
        <dl class="detail-grid">
          <div><dt>Kind</dt><dd>${escapeHTML(input.detail.kind)}</dd></div>
          <div><dt>Issuer</dt><dd>${escapeHTML(input.authenticator.issuer)}</dd></div>
          <div><dt>Account</dt><dd>${escapeHTML(input.authenticator.accountName)}</dd></div>
          <div><dt>Code</dt><dd><code>${escapeHTML(input.authenticator.code ?? "locked")}</code></dd></div>
          <div><dt>Status</dt><dd>${escapeHTML(input.authenticator.status)}</dd></div>
          <div><dt>Related Login</dt><dd>${escapeHTML(input.authenticator.relatedLoginTitle ?? "none")}</dd></div>
          <div><dt>Tags</dt><dd>${escapeHTML(tagValue)}</dd></div>
        </dl>
        ${renderEditForm({
          itemId: input.record.item.id,
          item: input.record.item,
          query: input.query,
        })}
        ${deleteForm}
      </div>
    `;
  }

  return `
    <div class="detail-card stack">
      <p class="eyebrow">Selected Item</p>
      <h2>${escapeHTML(input.detail.title)}</h2>
      <p>${escapeHTML(input.detail.summary)}</p>
      <dl class="detail-grid">
        <div><dt>Kind</dt><dd>${escapeHTML(input.detail.kind)}</dd></div>
        <div><dt>Status</dt><dd>${escapeHTML(input.detail.status)}</dd></div>
        <div><dt>Tags</dt><dd>${escapeHTML(tagValue)}</dd></div>
        <div><dt>History</dt><dd>${input.detail.history.length}</dd></div>
      </dl>
      ${input.record
        ? renderEditForm({
            itemId: input.record.item.id,
            item: input.record.item,
            query: input.query,
          })
        : ""}
      ${deleteForm}
    </div>
  `;
}

function renderCreateForm(query: {
  text?: string;
  kind?: string;
  tag?: string;
}): string {
  return `
    <form class="stack" method="post" action="/vault/items">
      ${renderVaultQueryHiddenFields(query)}
      <label class="field">
        <span>Kind</span>
        <select name="itemKind">
          <option value="login">login</option>
          <option value="totp">totp</option>
          <option value="note">note</option>
          <option value="apiKey">apiKey</option>
          <option value="sshKey">sshKey</option>
        </select>
      </label>
      <label class="field"><span>Title</span><input name="title" type="text" value="GitHub" required /></label>
      <label class="field"><span>Username</span><input name="username" type="text" value="alice" /></label>
      <label class="field"><span>URL</span><input name="url" type="url" value="https://github.com/login" /></label>
      <label class="field"><span>Password</span><input name="password" type="text" value="ghp-secret-123" /></label>
      <label class="field"><span>Issuer</span><input name="issuer" type="text" value="GitHub" /></label>
      <label class="field"><span>Account Name</span><input name="accountName" type="text" value="alice@example.com" /></label>
      <label class="field"><span>TOTP Secret</span><input name="secretBase32" type="text" value="JBSWY3DPEHPK3PXP" /></label>
      <label class="field"><span>Note Preview</span><input name="bodyPreview" type="text" value="Backup codes stored offline." /></label>
      <label class="field"><span>API Service</span><input name="service" type="text" value="GitHub API" /></label>
      <label class="field"><span>SSH Host</span><input name="host" type="text" value="github.com" /></label>
      <label class="field"><span>Tags</span><input name="tags" type="text" value="manual,m3" /></label>
      <button class="button button-primary" type="submit">Save item</button>
    </form>
  `;
}

function renderEditForm(input: {
  itemId: string;
  item: UnlockedLocalVault["workspace"]["itemRecords"][number]["item"];
  query: {
    text?: string;
    kind?: string;
    tag?: string;
  };
}): string {
  return `
    <form class="stack" method="post" action="/vault/items/update">
      ${renderVaultQueryHiddenFields(input.query)}
      <input name="itemId" type="hidden" value="${escapeHTML(input.itemId)}" />
      <input name="itemKind" type="hidden" value="${escapeHTML(input.item.kind)}" />
      <label class="field"><span>Title</span><input name="title" type="text" value="${escapeHTML(input.item.title)}" required /></label>
      ${renderEditFields(input.item)}
      <label class="field"><span>Tags</span><input name="tags" type="text" value="${escapeHTML(input.item.tags.join(", "))}" /></label>
      <button class="button button-primary" type="submit">Update item</button>
    </form>
  `;
}

function renderEditFields(
  item: UnlockedLocalVault["workspace"]["itemRecords"][number]["item"],
): string {
  switch (item.kind) {
    case "login":
      return `
        <label class="field"><span>Username</span><input name="username" type="text" value="${escapeHTML(item.username)}" required /></label>
        <label class="field"><span>URL</span><input name="url" type="url" value="${escapeHTML(item.urls[0] ?? "")}" required /></label>
        <label class="field"><span>Password</span><input name="password" type="text" value="" placeholder="Leave blank to keep current password" /></label>
      `;
    case "totp":
      return `
        <label class="field"><span>Issuer</span><input name="issuer" type="text" value="${escapeHTML(item.issuer)}" required /></label>
        <label class="field"><span>Account Name</span><input name="accountName" type="text" value="${escapeHTML(item.accountName)}" required /></label>
        <label class="field"><span>TOTP Secret</span><input name="secretBase32" type="text" value="" placeholder="Leave blank to keep current secret" /></label>
      `;
    case "note":
      return `<label class="field"><span>Note Preview</span><input name="bodyPreview" type="text" value="${escapeHTML(item.bodyPreview)}" required /></label>`;
    case "apiKey":
      return `<label class="field"><span>Service</span><input name="service" type="text" value="${escapeHTML(item.service)}" required /></label>`;
    case "sshKey":
      return `
        <label class="field"><span>Username</span><input name="username" type="text" value="${escapeHTML(item.username)}" required /></label>
        <label class="field"><span>Host</span><input name="host" type="text" value="${escapeHTML(item.host)}" required /></label>
      `;
  }
}

function renderVaultQueryHiddenFields(query: {
  text?: string;
  kind?: string;
  tag?: string;
}): string {
  return [
    `<input name="q" type="hidden" value="${escapeHTML(query.text ?? "")}" />`,
    `<input name="kind" type="hidden" value="${escapeHTML(query.kind ?? "all")}" />`,
    `<input name="tag" type="hidden" value="${escapeHTML(query.tag ?? "all")}" />`,
  ].join("");
}

function buildVaultLocation(input: {
  item?: string | null;
  q?: string;
  kind?: string;
  tag?: string;
}): string {
  const search = new URLSearchParams();
  if (input.item) {
    search.set("item", input.item);
  }
  if (input.q) {
    search.set("q", input.q);
  }
  if (input.kind && input.kind !== "all") {
    search.set("kind", input.kind);
  }
  if (input.tag && input.tag !== "all") {
    search.set("tag", input.tag);
  }
  return search.toString();
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

function readVaultQuery(url: URL) {
  return normalizeVaultQuery({
    q: url.searchParams.get("q"),
    kind: url.searchParams.get("kind"),
    tag: url.searchParams.get("tag"),
  });
}

function readVaultQueryFromForm(form: URLSearchParams) {
  return normalizeVaultQuery({
    q: form.get("q"),
    kind: form.get("kind"),
    tag: form.get("tag"),
  });
}

function normalizeVaultQuery(input: {
  q: string | null;
  kind: string | null;
  tag: string | null;
}) {
  const query: {
    text?: string;
    kind?: "all" | "login" | "note" | "apiKey" | "sshKey" | "totp";
    tag?: string;
  } = {};
  const text = input.q?.trim() ?? "";
  const tag = input.tag?.trim() ?? "";
  const kind = normalizeKind(input.kind);

  if (text !== "") {
    query.text = text;
  }
  if (kind) {
    query.kind = kind;
  }
  if (tag !== "" && tag !== "all") {
    query.tag = tag;
  }

  return query;
}

function normalizeKind(
  value: string | null,
): "all" | "login" | "note" | "apiKey" | "sshKey" | "totp" | undefined {
  switch (value) {
    case "all":
    case "login":
    case "note":
    case "apiKey":
    case "sshKey":
    case "totp":
      return value;
    default:
      return undefined;
  }
}

function readTags(form: URLSearchParams, fallback: string[]): string[] {
  const raw = form.get("tags")?.trim() ?? "";
  if (raw === "") {
    return fallback;
  }

  const tags = raw
    .split(",")
    .map((tag) => tag.trim())
    .filter(Boolean);
  return tags.length > 0 ? tags : fallback;
}

function buildLoginMutationInput(
  form: URLSearchParams,
  base: {
    workspace: UnlockedLocalVault["workspace"];
    secretMaterial: UnlockedLocalVault["secretMaterial"];
    deviceId: string;
    at: Date;
  },
) {
  return {
    ...base,
    title: form.get("title") ?? "",
    username: form.get("username") ?? "",
    url: form.get("url") ?? "",
    password: form.get("password") ?? "",
    tags: readTags(form, ["manual", "m3"]),
  };
}

async function createVaultItemFromForm(
  form: URLSearchParams,
  unlocked: UnlockedLocalVault,
  deviceId: string,
  at: Date,
) {
  const base = {
    workspace: unlocked.workspace,
    secretMaterial: unlocked.secretMaterial,
    deviceId,
    at,
  };

  switch (form.get("itemKind")) {
    case "login":
      return addLoginToVaultWorkspace(buildLoginMutationInput(form, base));
    case "totp":
      return addTotpToVaultWorkspace({
        ...base,
        title: form.get("title") ?? "",
        issuer: form.get("issuer") ?? "",
        accountName: form.get("accountName") ?? "",
        secretBase32: form.get("secretBase32") ?? "",
        tags: readTags(form, ["2fa", "authenticator"]),
      });
    case "note":
      return addNoteToVaultWorkspace({
        ...base,
        title: form.get("title") ?? "",
        bodyPreview: form.get("bodyPreview") ?? "",
        tags: readTags(form, ["note"]),
      });
    case "apiKey":
      return addApiKeyToVaultWorkspace({
        ...base,
        title: form.get("title") ?? "",
        service: form.get("service") ?? "",
        tags: readTags(form, ["api", "key"]),
      });
    case "sshKey":
      return addSshKeyToVaultWorkspace({
        ...base,
        title: form.get("title") ?? "",
        username: form.get("username") ?? "",
        host: form.get("host") ?? "",
        tags: readTags(form, ["ssh"]),
      });
    default:
      throw new Error("invalid vault item kind");
  }
}

async function updateVaultItemFromForm(
  form: URLSearchParams,
  unlocked: UnlockedLocalVault,
  deviceId: string,
  at: Date,
) {
  const base = {
    workspace: unlocked.workspace,
    secretMaterial: unlocked.secretMaterial,
    deviceId,
    itemId: form.get("itemId")?.trim() ?? "",
    at,
  };

  switch (form.get("itemKind")) {
    case "login":
      return updateLoginInVaultWorkspace({
        ...base,
        title: form.get("title") ?? "",
        username: form.get("username") ?? "",
        url: form.get("url") ?? "",
        password: form.get("password")?.trim() ? form.get("password")! : undefined,
        tags: readTags(form, []),
      });
    case "totp":
      return updateTotpInVaultWorkspace({
        ...base,
        title: form.get("title") ?? "",
        issuer: form.get("issuer") ?? "",
        accountName: form.get("accountName") ?? "",
        secretBase32: form.get("secretBase32")?.trim()
          ? form.get("secretBase32")!
          : undefined,
        tags: readTags(form, []),
      });
    case "note":
      return updateNoteInVaultWorkspace({
        ...base,
        title: form.get("title") ?? "",
        bodyPreview: form.get("bodyPreview") ?? "",
        tags: readTags(form, []),
      });
    case "apiKey":
      return updateApiKeyInVaultWorkspace({
        ...base,
        title: form.get("title") ?? "",
        service: form.get("service") ?? "",
        tags: readTags(form, []),
      });
    case "sshKey":
      return updateSshKeyInVaultWorkspace({
        ...base,
        title: form.get("title") ?? "",
        username: form.get("username") ?? "",
        host: form.get("host") ?? "",
        tags: readTags(form, []),
      });
    default:
      throw new Error("invalid vault item kind");
  }
}

const execFileAsync = promisify(execFile);

async function defaultRunSafeCommand(
  args: string[],
  env: NodeJS.ProcessEnv,
): Promise<SafeCommandResult> {
  const result = await execFileAsync("go", ["run", "./cmd/safe", ...args], {
    cwd: process.cwd(),
    env,
  });

  return {
    stdout: result.stdout,
    stderr: result.stderr,
  };
}

function buildSafeCommandEnv(
  dataDir: string,
  session: SessionState,
): NodeJS.ProcessEnv {
  const env: NodeJS.ProcessEnv = {
    ...process.env,
    SAFE_LOCAL_RUNTIME_DIR: path.join(dataDir, "accounts"),
    SAFE_LOCAL_PASSWORD: session.vaultPassword ?? "",
  };

  if (session.identity) {
    env.SAFE_DEVICE_ID = session.identity.deviceId;
  }

  if (!env.SAFE_CONTROL_PLANE_URL && process.env.SAFE_WEB_CONTROL_PLANE_URL) {
    env.SAFE_CONTROL_PLANE_URL = process.env.SAFE_WEB_CONTROL_PLANE_URL;
  }
  if (!env.SAFE_OAUTH_ACCESS_TOKEN && process.env.SAFE_WEB_OAUTH_ACCESS_TOKEN) {
    env.SAFE_OAUTH_ACCESS_TOKEN = process.env.SAFE_WEB_OAUTH_ACCESS_TOKEN;
  }

  return env;
}

function parseDeviceRecords(raw: string): DeviceRecord[] {
  const payload = JSON.parse(raw);
  return Array.isArray(payload) ? payload as DeviceRecord[] : [];
}

function parseEnrollmentRequests(raw: string): EnrollmentRequest[] {
  const payload = JSON.parse(raw);
  return Array.isArray(payload) ? payload as EnrollmentRequest[] : [];
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

      p, li, dd, dt, span, a, button, input, select, code { font-size: 1rem; }

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

      .vault-grid, .filter-grid { grid-template-columns: 1fr 1fr; }
      .field { display: block; }
      .field span {
        display: block;
        margin-bottom: 8px;
        color: var(--muted);
        font-weight: 700;
      }

      input, select {
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

      .item-link.is-selected {
        border-color: rgba(160, 77, 31, 0.45);
        background: rgba(255, 245, 236, 0.98);
      }

      .item-title { display: block; font-weight: 700; }
      .item-kicker, .item-meta {
        display: block;
        color: var(--muted);
        font-size: 0.82rem;
        text-transform: uppercase;
        letter-spacing: 0.08em;
      }
      .item-summary, .empty, .error, .success, dt { color: var(--muted); }
      .deleted-row {
        display: grid;
        grid-template-columns: 1fr auto;
        gap: 12px;
        align-items: center;
      }
      .device-row { display: block; }
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

      .success {
        margin: 0;
        padding: 12px 14px;
        border-radius: 16px;
        background: rgba(70, 126, 84, 0.09);
        border: 1px solid rgba(70, 126, 84, 0.18);
      }

      @media (max-width: 820px) {
        .hero, .vault-grid, .filter-grid, .stats-grid, .meta-grid, .detail-grid {
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
