# Content Distribution API

LightBridge's distribution center lets administrators deliver text, messages, regular files, and upstream-account export packages to a snapshot of selected users.

## Security model

- Every administrator write endpoint uses the existing admin authentication middleware.
- Recipient membership is resolved when a distribution is created and stored as a snapshot.
- Users can only read or download distributions whose recipient snapshot contains their user ID.
- Attachments are encrypted before database storage and are returned with `Content-Disposition: attachment` and `X-Content-Type-Options: nosniff`.
- A single decoded attachment is limited to 10 MiB. JSON requests using `file_base64` account for Base64 expansion while enforcing the same decoded limit.
- `account_export` attachments must use a `.json` or `.zip` filename.
- One distribution can contain at most 50,000 recipients. One batch request can create at most 100 distributions.
- An audience must use exactly one mode: explicit `user_ids`/`emails`, non-empty `filters`, multiline `lines`, or `all: true`. Empty filters and mixed modes are rejected. Use `all: true` explicitly for a platform-wide distribution.

## Content kinds

| Value | Meaning |
| --- | --- |
| `text` | Long-form text content |
| `message` | A normal inbox message |
| `file` | A regular downloadable file, optionally with explanatory text |
| `account_export` | A downloadable upstream-account export JSON or ZIP package |

## Create a distribution

`POST /api/v1/admin/distributions`

```json
{
  "title": "Maintenance instructions",
  "kind": "message",
  "content": "Please rotate your key before Friday.",
  "audience": {
    "user_ids": [12, 18],
    "emails": ["customer@example.com"]
  }
}
```

Files can be submitted either as JSON with `file_base64`, or as `multipart/form-data` with these fields:

- `title`
- `kind`
- `content`
- `audience`: JSON-encoded audience object
- `metadata`: optional JSON object
- `file`: uploaded attachment

## Advanced-filter distribution

The `filters` object uses the same user-list semantics as `GET /api/v1/admin/users`. At least one filter field must be non-empty; use `all: true` when the intended audience is every user.

```json
{
  "title": "Getting started guide",
  "kind": "file",
  "content": "This guide is for accounts that have never used the service.",
  "file_name": "getting-started.pdf",
  "content_type": "application/pdf",
  "file_base64": "...",
  "audience": {
    "filters": {
      "status": "active",
      "role": "user",
      "search": "",
      "group_name": "Starter",
      "activity": "none",
      "attributes": {
        "1": "education"
      }
    }
  }
}
```

`activity` accepts the canonical values implemented by the advanced user filter:

- `any`
- `usage`
- `balance_change`
- `none`

## Multiline import

Set `audience.lines` to one recipient per line. Blank lines and lines beginning with `#` are ignored.

```text
10001
user@example.com
10002 | Personalized title | Personalized content for user 10002
second@example.com | Account package | Please import this package before Monday
```

The first column is a user ID or email. The second and third optional columns override the title and content for that recipient. Do not combine multiline imports with another audience mode. If the same recipient appears more than once, a later non-empty override replaces that field, while an omitted or empty field preserves the existing personalized value.

## All users

```json
{
  "title": "Service notice",
  "kind": "message",
  "content": "A platform-wide notice.",
  "audience": {
    "all": true
  }
}
```

## Audience preview

`POST /api/v1/admin/distributions/audience-preview`

Request body is an audience object. The response contains the resolved recipient count and up to 50 preview users. The same single-mode audience validation used by distribution creation applies.

## Batch distribution

`POST /api/v1/admin/distributions/batch`

```json
{
  "items": [
    {
      "title": "Message A",
      "kind": "message",
      "content": "First delivery",
      "audience": { "emails": ["a@example.com"] }
    },
    {
      "title": "Message B",
      "kind": "text",
      "content": "Second delivery",
      "audience": { "filters": { "activity": "none" } }
    }
  ]
}
```

Each item is processed independently and the response reports successes and failures by input index. Every item must contain one valid audience mode.

## Administrator endpoints

| Method | Path | Purpose |
| --- | --- | --- |
| `GET` | `/api/v1/admin/distributions` | List distribution history and delivery statistics |
| `POST` | `/api/v1/admin/distributions` | Create one distribution |
| `POST` | `/api/v1/admin/distributions/batch` | Create multiple distributions |
| `POST` | `/api/v1/admin/distributions/audience-preview` | Resolve and preview an audience |
| `GET` | `/api/v1/admin/distributions/:id` | Get one distribution |
| `GET` | `/api/v1/admin/distributions/:id/download` | Download its attachment as an administrator |
| `DELETE` | `/api/v1/admin/distributions/:id` | Delete content and all recipient records |

## User endpoints

| Method | Path | Purpose |
| --- | --- | --- |
| `GET` | `/api/v1/distributions` | List the current user's deliveries; supports `unread_only=1` |
| `GET` | `/api/v1/distributions/:id` | Get a delivery assigned to the current user |
| `POST` | `/api/v1/distributions/:id/read` | Mark it as read |
| `GET` | `/api/v1/distributions/:id/download` | Download an assigned attachment |
