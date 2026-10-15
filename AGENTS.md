# StatusPane TUI

A Go terminal client for StatusPane, which aggregates third-party status pages (GitHub,
Cloudflare, OpenAI, …) into one dashboard per organization. This repo is only the client: the
data and the API live in the Rails app (`statuspane/statuspane`). It authenticates with an
organization API key and uses only what the public API exposes.

## Stack

- Go (version pinned in `mise.toml`), one module `github.com/statuspane/tui`, a single binary
- Bubble Tea v2 (model/update/view), Lip Gloss v2, `charm.land/bubbles/v2`
- Standard library for HTTP, JSON and tests; `BurntSushi/toml` for the config file
- golangci-lint v2 (curated set in `.golangci.yml`), govulncheck, GitHub Actions

## Local environment

- Tools come from mise. Run Go through it: `mise exec -- go test ./...`, or use the tasks:
  `mise run ci` (everything CI runs), `mise run test`, `mise run lint`, `mise run fmt`.
- A non-mise shell (like an agent's) may not have `go` on its PATH, or the wrong version.
- Local server: set `STATUSPANE_URL=http://localhost:3000` and an API key. Don't write keys into
  files you commit or into chat.

## Working conventions

- One GitHub issue per vertical slice of work, one branch per issue (`<issue-number>-<slug>`),
  one PR per issue. Use the issue forms; a slice says what, why, done-when and out of scope.
- PRs follow `.github/pull_request_template.md`: `## Summary` (+ `Closes #N`), `## Changes`,
  `## Testing`, optional `## Notes`. Concise and human-readable; say what was not verified.
- Commits are small and independently reviewable, each passing its own checks. When an AI agent
  is doing the committing, it shows the diff and waits for explicit go-ahead before running
  `git commit` — one commit at a time, never batched.
- Comments only for a non-obvious why, usually one line. Never reference issue numbers in code.
- A PR that changes what a user of the binary sees adds a line under `## [Unreleased]` in
  `CHANGELOG.md` (Added, Changed, Fixed, Removed), written for users, not contributors.
  Refactors, CI and docs-only changes don't. A release renames `Unreleased` to the version and
  date, then tags it.
- Before considering a change done, run `mise run ci`.

## Go conventions here

- Tests use the standard `testing` package, `httptest` for the API, and small hand-written
  fakes. Test behaviour through the public surface; no mocking libraries, no assertions on
  private state unless that state is the behaviour (a cursor index).
- Accept small interfaces defined where they are consumed (`ui.AppLister`), not where they are
  implemented.
- Inject what makes code hard to test: `getenv`, the clock, the home directory.
- Name things for what they are: `appDetail`, not `app`, when it holds an `AppDetail`.
  Named constants for durations, widths and status codes.
- Errors are values: wrap with `%w`, match with `errors.As`/`errors.Is`.

## Where knowledge goes

There is no docs folder or decision log. Put what you learn where the next person will trip
over it:

- **Why it's this way** (history, alternatives): the commit message body.
- **One line of code needs it**: a comment on that line.
- **A package's patterns or gotchas**: that package's `AGENTS.md` (`internal/api`,
  `internal/config`, `internal/ui`). Create one if a package needs it, with a one-line
  `CLAUDE.md` containing `@AGENTS.md` beside it.
- **Every session needs it**: this file, kept short.

`AGENTS.md` files describe how things are now. When code changes, update or delete the lines it
makes wrong; never append dated entries.
