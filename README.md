# StatusPane TUI

A terminal client for [StatusPane](https://statuspane.io): see your organization's apps and the
status of every component they watch, without leaving the terminal. It talks to the StatusPane
public API with an organization API key.

Contributors and coding agents: see [AGENTS.md](AGENTS.md).

## Build

Go and the dev tools are pinned in `mise.toml`.

```sh
mise install
mise run build      # vets, then builds ./statuspane-tui
```

Or install it straight from Go:

```sh
go install github.com/statuspane/tui/cmd/statuspane-tui@latest
```

## Configure

Create an organization API key in StatusPane, then either export it:

```sh
export STATUSPANE_API_KEY=sp_...
```

or put it in `~/.config/statuspane/config.toml` (`$XDG_CONFIG_HOME` is respected):

```toml
api_key = "sp_..."
```

The server defaults to `https://statuspane.io`. Override it with `STATUSPANE_URL` or `base_url`
in the file, for example to use a local server. Environment variables win over the file.

## Run

```sh
./statuspane-tui
```

`↑/↓` move, `enter` opens an app, `esc` goes back, `n` creates an app, `e` renames the selected
app, `d` deletes it (you type its name to confirm), `r` refreshes, `q` quits. On an app's screen,
`w` opens the component picker: choose a provider, `space` watches or unwatches a component, `g`
watches or unwatches the whole group under the cursor (the components you can see), `r` asks again
for the provider on screen, and `s` saves. `/` filters the pane that has the focus as you type
(`enter` keeps the filter, `esc` clears it): on the providers by name, slug or category, on the
components by name or group. Filtering makes no request, and a component filter belongs to the
provider it was typed for; the API cannot search across providers. The API allows 60 requests a
minute per key, so the picker loads a provider only once the cursor rests on it and remembers
what it loaded for five minutes.

## Develop

```sh
mise run ci         # everything CI runs: lint, vet and build, tests with -race, go mod tidy, govulncheck
mise run test       # just the tests
mise run fmt        # format (gofmt + goimports)
```

`mise tasks` lists the rest. CI runs the same tasks on every pull request.

```
cmd/statuspane-tui  the binary: wires config, API client and UI together
internal/config    API key and server URL from env or file
internal/api       client for the StatusPane public API
internal/ui        Bubble Tea screens
```
