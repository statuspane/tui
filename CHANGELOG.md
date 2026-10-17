# Changelog

All notable user-facing changes are listed here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and the project follows
[Semantic Versioning](https://semver.org/); while it is `0.x`, anything may change.

## [Unreleased]

### Added

- Choose the components an app watches: press `w` on the app screen, pick a provider, and mark
  components with `space`. Search a provider's components by name or group with `/`; the search
  covers one provider at a time. Nothing changes until you save with `s`, and `esc` asks before
  throwing changes away.
- Create an app with `n`: a dialog asks for its name (up to 100 characters) and opens the new app.
- Delete an app with `d`, from the list or from the app screen. The dialog says how many
  components it stops watching, and you type the app's name to confirm.
- An empty apps list now says how to create the first app.

### Changed

- The program is now called `statuspane-tui`. Install it with
  `go install github.com/statuspane/tui/cmd/statuspane-tui@latest`; the old install path
  produced a binary named `tui`.

## [0.1.0] - 2026-10-15

The first release: a read-only prototype. Creating and deleting apps and choosing which
components an app watches are not here yet.

### Added

- The apps screen: your organization's apps with their overall status and how many components
  each watches. It refreshes every 30 seconds, `r` refreshes now, and a failed refresh keeps the
  list on screen with an error line.
- The app screen: press `enter` on an app to see its active incidents (impact, status, when it
  started, the latest update and the components affected) and the components it watches,
  grouped by provider and group. `esc` goes back to the list with the selection kept.
- Configuration through `STATUSPANE_API_KEY` or `~/.config/statuspane/config.toml`. The server
  defaults to `https://statuspane.io`; `STATUSPANE_URL` or `base_url` points it elsewhere,
  such as a local server.
- Plain explanations for the common failures: a missing API key prints an example config file,
  a rejected key names the setting to check, and a rate limit says how long to wait.

[Unreleased]: https://github.com/statuspane/tui/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/statuspane/tui/releases/tag/v0.1.0
