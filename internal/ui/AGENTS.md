# internal/ui

Bubble Tea v2 screens: `RootModel` (which screen is showing) → `AppsModel` (list) and
`AppDetailModel` (one app).

## Patterns

- A screen is a value type with `Init`/`Update`/`View`. `Update` returns the changed copy, so
  state changes look like `m.x = …; return m, cmd`.
- Work happens in `tea.Cmd`s that return messages (`appsLoadedMsg`, `appsFailedMsg`); nothing
  blocks `Update`. Fetches use a timeout context created inside the command.
- Navigation is a message too: the list emits `openAppMsg`, the detail emits `closeDetailMsg`,
  and `RootModel` reacts. Key presses go to the visible screen only; other messages go to both
  and each ignores what isn't its own.
- Screens accept small interfaces (`AppLister`, `AppGetter`) so tests pass fakes.

## Gotchas

- Refresh ticks carry a sequence number. A manual refresh bumps it so a pending tick from
  before is ignored; without that you get two refresh loops.
- Pad before styling. `fmt` counts colour escape codes, so `%-16s` on a styled string
  misaligns columns. Pad the plain label, then render it (`renderStatus`).
- Don't read the real clock in rendering code; the detail model holds `now` so tests can fix it.
- `refreshInterval` and `fetchTimeout` are named constants; keep durations out of call sites.

## Testing

Drive `Update` with messages and assert on `View().Content`: no terminal, no goroutines.
Substring checks survive styling because escape codes wrap, not split, the text.
