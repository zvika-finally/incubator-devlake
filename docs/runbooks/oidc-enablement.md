# Runbook: Enabling OIDC login

How to turn on OIDC single-sign-on for DevLake. OIDC support is fully implemented
in the backend (`backend/helpers/oidchelper`, `backend/server/api/auth`); enabling
it is **configuration only** — no code changes.

This runbook is written generically and applies to any OIDC-compliant IdP
(Okta, Auth0, Keycloak, Ping, Entra, Google, …). Where a step is provider-specific
it is called out; everything else is the same regardless of IdP.

---

## How auth works

Two independent switches:

- `AUTH_ENABLED` — the master gate. When `false` (default), DevLake is open and
  no login is required. When `true`, requests must carry either an API key/proxy
  header **or** a valid OIDC session cookie.
- `OIDC_ENABLED` — adds interactive OIDC login on top. Requires `AUTH_ENABLED=true`.
  With `AUTH_ENABLED=true` but `OIDC_ENABLED=false`, only API-key/proxy auth works.

Login endpoints (backend, redirect-based — the config-ui needs no changes):

| Route | Method | Purpose |
|---|---|---|
| `/auth/login?provider=<name>` | GET | Start login with a named provider |
| `/auth/callback` | GET | IdP redirect target (register this URL with the IdP) |
| `/auth/logout` | POST | Clear the session |

> ⚠️ Flipping `AUTH_ENABLED=true` changes access **for every user at once**. Roll it
> out on a non-production instance first (see Rollout).

---

## Prerequisites

- Your DevLake external base URL, e.g. `https://devlake.example.com`.
- Admin access to your IdP to create an application/client.
- Ability to set environment variables on the DevLake **backend** (`lake`) service
  (`.env` for local/Compose, or Secret/ConfigMap env for k8s).

---

## Step 1 — Create an OIDC application in your IdP

In your IdP, create a new **OIDC / OpenID Connect web application** (confidential
client, authorization-code flow). Configure:

- **Sign-in redirect URI:** `https://<your-devlake-url>/auth/callback`
  (must match `OIDC_<NAME>_REDIRECT_URL` exactly, including scheme and no trailing slash).
- **Grant type:** Authorization Code.
- **Scopes:** `openid`, `profile`, `email`.

Collect from the IdP:

- **Issuer URL** — the base OIDC issuer that serves `/.well-known/openid-configuration`.
  - Okta: `https://<org>.okta.com/oauth2/default` (or a custom authorization server).
  - Auth0: `https://<tenant>.auth0.com/`.
  - Keycloak: `https://<host>/realms/<realm>`.
  - Entra: `https://login.microsoftonline.com/<tenant-id>/v2.0`.
  - Google: `https://accounts.google.com`.
- **Client ID**
- **Client secret**

> The issuer must be reachable from the DevLake backend at startup — the provider is
> validated via OIDC discovery when the server boots.

---

## Step 2 — Generate a session secret

`SESSION_SECRET` signs/encrypts the session and OIDC state cookies. It must be at
least **32 bytes** of high-entropy data. Rotating it invalidates all active sessions.

```bash
openssl rand -base64 48
```

Store it as a secret (Vault, k8s Secret, etc.) — never commit it.

---

## Step 3 — Set the environment variables

Pick a short provider name (used as the `OIDC_PROVIDERS` entry and the env prefix).
This runbook uses `okta` as the example name — replace throughout with yours.

```dotenv
# Master gate + OIDC
AUTH_ENABLED=true
OIDC_ENABLED=true
SESSION_SECRET=<from Step 2, >=32 bytes>

# One or more provider names (comma-separated)
OIDC_PROVIDERS=okta

# Per-provider block: prefix is OIDC_<UPPERCASE_NAME>_
OIDC_OKTA_ISSUER_URL=https://<org>.okta.com/oauth2/default
OIDC_OKTA_CLIENT_ID=<client id>
OIDC_OKTA_CLIENT_SECRET=<client secret>
OIDC_OKTA_REDIRECT_URL=https://<your-devlake-url>/auth/callback
OIDC_OKTA_SCOPES=openid,profile,email
OIDC_OKTA_DISPLAY_NAME=Okta
```

Add more providers by listing more names in `OIDC_PROVIDERS` and replicating the
block under each prefix (e.g. `OIDC_GOOGLE_*`).

### Optional: session & cookie tuning

```dotenv
SESSION_TTL=24h            # default session lifetime
COOKIE_SECURE=true         # default true; set false ONLY for local http testing
COOKIE_DOMAIN=example.com  # set if the UI and API share a parent domain
OIDC_LOGOUT_REDIRECT=false # true = also redirect to the IdP's end-session endpoint
```

### Optional: workload identity (k8s, no static secret)

If running in a cloud k8s with federated credentials, you can omit the client
secret and use a projected identity token instead:

```dotenv
OIDC_OKTA_USE_WORKLOAD_IDENTITY=true
# then OIDC_OKTA_CLIENT_SECRET may be left empty
```

---

## Step 4 — Restrict who can log in (recommended)

Without an allowlist, **any** account the IdP authenticates is accepted. Restrict to
your org (comma-separated, case-insensitive):

```dotenv
OIDC_ALLOW_DOMAINS=example.com          # only @example.com emails
OIDC_ALLOW_EMAILS=alice@example.com,bob@example.com   # or specific users
```

If both are set, a login is allowed if it matches **either**. Leave both empty only
if the IdP application itself already restricts assignment to the right users.

---

## Step 5 — Roll out

1. **Staging first.** Apply the config to a non-production DevLake, restart the
   `lake` backend, and confirm it boots — the log prints
   `OIDC provider "okta" enabled (issuer=…, client=…)` for each provider. A
   misconfigured issuer or a `SESSION_SECRET` under 32 bytes fails startup with a
   clear error.
2. **Verify the flow:** open the DevLake UI (or hit `/auth/login?provider=okta`),
   complete IdP login, land back via `/auth/callback`, and confirm you reach the app
   with a session. The startup log prints one line per successful login
   (`oidc login: provider=… sub=… email=… jti=…`).
3. **Verify allowlist:** attempt a login from a disallowed email/domain and confirm
   it's rejected.
4. **Verify API keys still work** (automation/pipelines): existing API-key auth
   continues to function alongside OIDC.
5. **Production:** apply the same config, restart, re-verify.

---

## Rollback

OIDC/auth is controlled entirely by env vars, so rollback is a config revert:

- Disable **just** OIDC (keep API-key auth): set `OIDC_ENABLED=false`, restart.
- Disable **all** auth (fully open again): set `AUTH_ENABLED=false`, restart.

No data or schema changes are involved.

---

## Troubleshooting

| Symptom | Likely cause |
|---|---|
| Backend fails to start: `SESSION_SECRET is not set` / `must be at least 32 bytes` | `OIDC_ENABLED=true` without a valid `SESSION_SECRET`. |
| Startup error: `OIDC_PROVIDERS is empty` | `OIDC_ENABLED=true` but no provider names listed. |
| Startup error mentioning `oidc discovery (<issuer>)` | Issuer URL wrong or unreachable from the backend; must serve `/.well-known/openid-configuration`. |
| IdP shows "redirect URI mismatch" | `OIDC_<NAME>_REDIRECT_URL` ≠ the URI registered in the IdP (scheme/host/path must match exactly, no trailing slash). |
| Login succeeds at IdP but DevLake rejects the user | Allowlist (`OIDC_ALLOW_DOMAINS`/`OIDC_ALLOW_EMAILS`) excludes them, or the IdP omits the `email` claim (ensure the `email` scope). |
| Cookie not set / immediately logged out | `COOKIE_SECURE=true` over plain HTTP, or a `COOKIE_DOMAIN` that doesn't match the UI host. |

---

## Environment variable reference

| Variable | Required | Notes |
|---|---|---|
| `AUTH_ENABLED` | yes | Master gate. `true` to require auth. |
| `OIDC_ENABLED` | yes (for SSO) | Requires `AUTH_ENABLED=true`. |
| `SESSION_SECRET` | yes (when OIDC on) | ≥32 bytes. Rotating invalidates sessions. |
| `OIDC_PROVIDERS` | yes (when OIDC on) | Comma-separated provider names. |
| `OIDC_<NAME>_ISSUER_URL` | yes | Per provider. |
| `OIDC_<NAME>_CLIENT_ID` | yes | Per provider. |
| `OIDC_<NAME>_CLIENT_SECRET` | yes* | *Unless `USE_WORKLOAD_IDENTITY=true`. |
| `OIDC_<NAME>_REDIRECT_URL` | yes | Must match IdP registration; `…/auth/callback`. |
| `OIDC_<NAME>_SCOPES` | no | Default `openid,profile,email`. |
| `OIDC_<NAME>_DISPLAY_NAME` | no | Defaults to the provider name. |
| `OIDC_<NAME>_USE_WORKLOAD_IDENTITY` | no | k8s federated creds; secret optional. |
| `OIDC_ALLOW_DOMAINS` | no | Allowlist by email domain. |
| `OIDC_ALLOW_EMAILS` | no | Allowlist by exact email. |
| `OIDC_LOGOUT_REDIRECT` | no | Also hit the IdP end-session endpoint on logout. |
| `SESSION_TTL` | no | e.g. `24h`. |
| `COOKIE_SECURE` | no | Default `true`. |
| `COOKIE_DOMAIN` | no | Set when UI/API share a parent domain. |

See `env.example` (the `# OIDC / Authentication` section) for inline templates,
including ready-to-copy Entra and Google blocks.
