# Auth flow

PayRam has a single root-account onboarding flow plus standard signin /
forgot-password flows. There is **no public signup** — the only signup that
exists is the **first-run** wizard that creates the root account, after which
the `/signup` route redirects to `/login` for everyone.

## Screens

| # | Route | Spec |
|---|---|---|
| 1 | `/signup` (first run only) | [signup.md](../SCREENS/signup.md) |
| 2 | `/login` | [login.md](../SCREENS/login.md) |
| 3 | `/forgotPassword` | [forgotPassword.md](../SCREENS/forgotPassword.md) |
| 4 | `/createPassword` (token link) | [createPassword.md](../SCREENS/createPassword.md) |
| 5 | `/setPassword` (admin reset) | [setPassword.md](../SCREENS/setPassword.md) |

## First-run signup wizard

PayRam guards `/signup` behind `GET /api/v1/member/root/exist`. If the
response is truthy the page hard-redirects to `/login` and the localStorage
key `payram_root_exists=true` is cached so subsequent loads skip the probe.

The wizard has **three steps** rendered as a single page with progressive
disclosure (no URL changes between steps):

1. **Intro** — copy + a single "Next" button. No fields.
2. **Email** — single email input + "Continue".
3. **Password** — `password` + `confirmPassword` inputs with a four-rule
   strength meter (uppercase / lowercase / number / special). Submit posts
   to `POST /api/v1/signup` (not directly observed in the safe crawl —
   write-mutating, see `_summary.json`).

A successful response writes the JWT bundle to localStorage (see signin
below) and hard-redirects to `/project/all/dashboard`.

## Signin

```
POST /api/v1/signin
Content-Type: application/json
{ "email": string, "password": string }

200 OK
{
  "id": number,
  "accessToken": string,         // JWT, ~15min TTL
  "refreshToken": string,        // JWT, ~7d TTL, includes jti
  "memberID": number,
  "member": { id, email, name, customer_id, group, state, memberType, ... },
  "roleID": number,
  "role": { id, name, displayName, description },
  "expiresIn": 900,
  "tokenType": "Bearer",
  "resetPasswordRequired": boolean
}
```

The SPA persists into `localStorage`:

- `payram_access_token` — the JWT verbatim
- `payram_refresh_token` — the refresh JWT verbatim
- `payram_token_expiry` — epoch ms
- `payram_user` — the entire `member` object as JSON
- `payram_root_exists` — `"true"` once verified

Then the SPA navigates to `/project/all/dashboard` (note the `all` meta-project)
and immediately fires:

- `POST /api/v1/websocket-token/create?clientID=<random>` — opens the live
  events channel
- `GET /api/v1/external-platform/details` and `/api/v1/external-platform/all`
- `GET /api/v1/blockchains`
- ~10 `POST /api/v1/external-platform/all/analytics/groups/{id}/graph/{id}/data`
  for the dashboard widgets

If signin fails the page renders the toast text **"Incorrect email or
password"** and stays on `/login`. There's a **3-attempt-then-lockout** rate
limiter on the backend (mentioned in the demo research; not directly tested).

### "Reset password required" branch

If `resetPasswordRequired === true` the SPA redirects to `/setPassword?token=…`
instead of the dashboard. The user enters a new password and the same JWT
bundle is returned.

## Forgot password

`/forgotPassword` is a single-input page (`email`). Submitting calls
`POST /api/v1/forgot-password` (not in our crawl — mutating). The backend
emails a `reset_password_token` link of the form
`<dashboard>/createPassword?token=<jwt>`. `/createPassword` validates the
token via `GET /api/v1/member/reset-password/verify?token=…`, then takes a
`password` + `confirmPassword`, and submits to
`POST /api/v1/member/reset-password`.

## 401 handling

Every authenticated XHR carries `Authorization: Bearer <accessToken>`. On a
401 the SPA pre-emptively tries the refresh-token endpoint; on a second 401
it clears localStorage and redirects to `/login`. We did not see this branch
fire in the crawl because the JWT TTL (15 min) is longer than the crawl run.

## Logout

`/logout` is a no-render route that clears the four `payram_*` localStorage
keys, fires `POST /api/v1/signout` (best-effort), and replaces the URL with
`/login`. We deliberately skip this in `crawl.mjs` so subsequent route
captures keep their session.

## Notes

- **Activity-log middleware** writes to a `SQL_ASCII` Postgres in the demo
  image and crashes on non-UTF-8 bytes. The signin handler returns the JWT
  successfully, then the post-handler middleware logs an error to stderr.
  In our setup the response was 200; on stricter installs you may see a 500
  with the JWT still set in the body.
- The bcrypt prefix issue (`$2b$10$…`) bites if you ever reset passwords by
  hand via psql — see the [`README.md`](../README.md) "Login bootstrap"
  section.
