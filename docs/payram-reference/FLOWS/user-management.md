# User & role management

PayRam supports multiple internal members per merchant account, each assigned
to one of six roles. The flow lives under `/settings/{userManagement,roleManagement,policyManagement,activityLog}`.

## Screens

| # | Route | Spec |
|---|---|---|
| 1 | `/settings/userManagement` | [settings-userManagement.md](../SCREENS/settings-userManagement.md) |
| 2 | `/settings/roleManagement` | [settings-roleManagement.md](../SCREENS/settings-roleManagement.md) |
| 3 | `/settings/roleManagement/role/{id}` | [settings-roleManagement-role-roleId.md](../SCREENS/settings-roleManagement-role-roleId.md) |
| 4 | `/settings/policyManagement` | [settings-policyManagement.md](../SCREENS/settings-policyManagement.md) |
| 5 | `/settings/activityLog` | [settings-activityLog.md](../SCREENS/settings-activityLog.md) |
| 6 | `/settings/adminControl` | [settings-adminControl.md](../SCREENS/settings-adminControl.md) |

## Read endpoints observed

- `GET /api/v1/external-platform/all/members` — `{ members: [...] }`
- `POST /api/v1/internalMembers` — internal member list (POST is used because
  it accepts a complex filter body)
- `GET /api/v1/roles` — `{ roles: [...] }` — returns the six built-in roles
- `GET /api/v1/activity-log` — paginated audit trail
- `GET /api/v1/activity-log/event-categories` — filter chip source

## Built-in roles

From `GET /api/v1/roles` and the JWT we minted (`"roles":["root"]`,
`"perms":["write_withdrawal_approve"]`), PayRam ships with **six** built-in
roles in this hierarchy:

1. **Root** — single seat, all permissions, can never be deleted
2. **Admin** — almost everything except seat-management of root
3. **Manager** — payments + payouts + customers, no settings
4. **Operator** — read + sweep approval, no payouts
5. **Developer** — API keys, webhooks, docs only
6. **Viewer** — read-only across the dashboard

The `/settings/roleManagement` page shows them as cards. Clicking a role
opens `/settings/roleManagement/role/{id}` with a **permission matrix** —
rows are 30+ permissions grouped by domain (`payments.read`, `payments.write`,
`payouts.approve`, `wallets.create`, `wallets.export`, `users.invite`,
`webhooks.create`, `settings.write`, …), columns are toggle switches.

Built-in roles are read-only; the merchant can clone one to create a custom
role.

## User management sequence

1. Land on `/settings/userManagement`. Loads `internalMembers` list.
2. Table columns: **Name**, **Email**, **Role**, **Status**, **Last active**, **Actions**.
3. **"Invite User"** lime CTA → modal:
   - Email input
   - Role dropdown (single-select, 6 options)
   - Optional message text area
   - Submit → `POST /api/v1/internalMembers/invite` (mutating). The invitee
     receives an email link to `/createPassword?token=…` to set their
     password and activate the account.
4. Row actions menu:
   - **Edit role** → role-change modal, calls `PUT /api/v1/internalMembers/{id}`
   - **Reset password** → forces `reset_password_required=true` and emails
     a fresh link
   - **Ban** / **Unban** → toggles `state` between `active` and `banned`
   - **Delete** → soft-delete via `deleted_at`
5. Clicking a row anywhere else opens a **detail drawer** with the user's
   **last 50 activity log entries** (drilled from `/api/v1/activity-log`
   filtered by `member_id`).

## Activity log

`/settings/activityLog` is a global audit table. Schema (from
`/api/v1/activity-log`):

```
{
  data: [{
    id, created_at,
    member_id, member_email, member_name,
    event_category, event_action,        // e.g. "payments", "create_payment"
    target_type, target_id,
    request_method, request_path,
    response_status,
    metadata: { ... }                    // JSON, varies by event
  }],
  total_count: number
}
```

Filter chips along the top come from `/api/v1/activity-log/event-categories`
and let you scope by category (Auth, Payments, Wallets, Withdrawals, Settings,
Webhooks, Members, System).

Clicking a row opens a **detail modal** that pretty-prints the full
`metadata` JSON in a `JetBrains Mono` block.

## Policy management

`/settings/policyManagement` is a sparser cousin of role management — it's
where the merchant configures **non-permission policies** like:

- IP allow-listing for the dashboard
- Required 2FA for specific roles
- Session timeout per role
- Failed-login lockout threshold

(This page mostly renders empty cards in the demo build; the underlying
endpoints are stubbed.)

## Gaps vs. Payminto clone

- `settings/userManagement` exists as a stub — no invite flow, no role
  change, no ban toggle.
- `settings/roleManagement` exists as an empty state — no permission matrix.
- `settings/activityLog` exists with mock data only — no event-category
  chips, no detail modal, no filter persistence.
- `settings/policyManagement`, `settings/adminControl` are missing.
