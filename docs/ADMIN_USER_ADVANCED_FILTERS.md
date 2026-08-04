# Admin user advanced filters

The existing admin user list endpoint supports activity-based filtering without changing its pagination or response format.

## Endpoint

```http
GET /api/v1/admin/users
```

Authentication is the same as for the other admin endpoints.

## Activity filter

Pass the `activity` query parameter with one of the canonical values below:

| Value | Meaning |
| --- | --- |
| `any` | The user has at least one API usage record or balance-change record. |
| `usage` | The user has at least one API usage record. |
| `balance_change` | The user has at least one balance-change record. This includes balance redemption, admin balance adjustment, and affiliate balance transfer records. |
| `none` | The user has neither API usage nor any balance-change record. |

Aliases intended for automation clients are also accepted: `active` / `has_activity`, `used`, `balance` / `balance_changed`, and `unused` / `no_activity`.

An unsupported value returns HTTP 400.

## Examples

Users who have used the service or whose balance has changed:

```http
GET /api/v1/admin/users?activity=any&page=1&page_size=50
```

Users with no usage or balance history:

```http
GET /api/v1/admin/users?activity=none&page=1&page_size=50
```

Users with API usage only:

```http
GET /api/v1/admin/users?activity=usage
```

Users with a balance change only:

```http
GET /api/v1/admin/users?activity=balance_change
```

The filter composes with existing parameters such as `status`, `role`, `group_name`, `search`, `sort_by`, and `sort_order`:

```http
GET /api/v1/admin/users?activity=any&status=active&role=user&search=example.com&sort_by=last_used_at&sort_order=desc
```

## Reserved search expression

For internal clients that already construct only the `search` parameter, the same filter can be expressed with a reserved token:

```http
GET /api/v1/admin/users?search=example.com%20%40activity%3Aany
```

The public `activity` parameter is preferred. Recognized activity tokens are removed before ordinary email, username, notes, and API-key searching is applied.
