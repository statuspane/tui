# internal/api

Client for the StatusPane public API (Grape, mounted at `/api`, v1). The server-side contract
is in the Rails repo's `app/api/CLAUDE.md` and the live docs at `<server>/api/docs`.

## Gotchas

- Every non-2xx response becomes `*APIError` (`Status`, `Code`, `Message`, `Details`,
  `RetryAfter`), including bodies that aren't the API's `{ "error": { code, message } }` shape
  (then `Code` is empty). Branch on `Code`, never on `Message`.
- A `validation_failed` error carries `Details`: one `FieldError{Field, Code, Message}` per
  rejected request field, such as `name`/`too_long` or `watched_components[3]`/`unknown_component`
  (the index is the position that was sent). Show the problem against that field; absent on
  other errors.
- Fields the API documents as nullable are pointers (`*string`, `*time.Time`). Don't "simplify"
  them to values: `""` and a missing status mean different things.
- There are no retries. The UI refreshes on a timer anyway, and creating an app isn't
  idempotent, so a blind retry could create two.
- Write bodies are a `map[string]any` holding only the fields being changed ("leave it out to
  keep it" is the API's rule), never a struct with `omitempty`: `omitempty` would drop the `[]`
  that clears the watched set. The API rejects a JSON `null` for `watched_components` because
  it would unwatch everything, and a nil Go slice marshals to `null`, so `SetWatchedComponents`
  turns nil into `[]`. `CreateApp` is the reverse: nil sends no list at all.
- `watched_components` replaces the whole set. To add one component, send the current set plus
  it; sending only the new one stops watching everything else.
- Component search (`q`) filters one provider's components by name or group name. There is no
  search across providers.
- `DeleteApp` returns nothing: a 204 has no body, and `do` skips decoding when `out` is nil. A
  404 means the app is already gone.
- The base URL is the site root; the client appends `/api/v1`.
