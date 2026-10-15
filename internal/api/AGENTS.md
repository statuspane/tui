# internal/api

Client for the StatusPane public API (Grape, mounted at `/api`, v1). The server-side contract
is in the Rails repo's `app/api/CLAUDE.md` and the live docs at `<server>/api/docs`.

## Gotchas

- Every non-2xx response becomes `*APIError` (`Status`, `Code`, `Message`, `RetryAfter`),
  including bodies that aren't the API's `{ "error": { code, message } }` shape (then `Code` is
  empty). Branch on `Code`, never on `Message`.
- Fields the API documents as nullable are pointers (`*string`, `*time.Time`). Don't "simplify"
  them to values: `""` and a missing status mean different things.
- There are no retries. The UI refreshes on a timer anyway, and creating an app isn't
  idempotent, so a blind retry could create two.
- Write requests (when added): the API rejects a JSON `null` for `watched_components` because
  it would unwatch everything. A nil Go slice marshals to `null`; an empty one to `[]`. Build
  the body as a `map[string]any` holding only the fields being changed, and never use
  `omitempty` for the list: it would drop the `[]` that clears it.
- The base URL is the site root; the client appends `/api/v1`.
