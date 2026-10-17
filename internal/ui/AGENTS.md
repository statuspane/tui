# internal/ui

Bubble Tea v2 screens: `RootModel` (which screen is showing) → `AppsModel` (list) and
`AppDetailModel` (one app), `WatchPickerModel` (choose an app's components) on top of the
detail, or a dialog (`createAppDialog`, `deleteAppDialog`) instead of any of them.

## Patterns

- A screen is a value type with `Init`/`Update`/`View`. `Update` returns the changed copy, so
  state changes look like `m.x = …; return m, cmd`.
- Work happens in `tea.Cmd`s that return messages (`appsLoadedMsg`, `appsFailedMsg`); nothing
  blocks `Update`. Fetches use a timeout context created inside the command.
- Navigation is a message too: the list emits `openAppMsg`, the detail emits `closeDetailMsg`,
  and `RootModel` reacts. Key presses go to the visible screen only; other messages go to both
  and each ignores what isn't its own.
- Screens accept small interfaces (`AppLister`, `AppGetter`, `AppCreator`, `AppDeleter`) so
  tests pass fakes; `Backend` is their union, which `*api.Client` satisfies.
- A dialog implements `dialog` (`update`, `view`) and is owned by `RootModel`, because it can open
  over either screen. A screen asks for one with a message (`openCreateAppDialogMsg`,
  `openDeleteAppDialogMsg`); the dialog answers with `appCreatedMsg`, `appDeletedMsg` or
  `closeDialogMsg`, and the root reacts (open the new app, close the deleted app's detail, tell
  the list with `appsChangedMsg`).

## Gotchas

- While a dialog is open, `RootModel` sends it every key press and the screens none, so typing
  `q` or `j` into a name can't quit or move the list. Only `ctrl+c` quits.
- Writes are not idempotent: a dialog ignores `enter` while its request is in flight
  (`isSubmitting`), and a failed request leaves the dialog open with the typed text. It also
  ignores `esc` then: the success message acts on the root (opens the new app, closes the
  deleted one's detail) and assumes the dialog is still the one that sent it.
- `appsChangedMsg` during a fetch can't start a second one, because the first may predate the
  change: `AppsModel` sets `reloadAfterLoad` and fetches again right after it lands.
- Deleting an app that is already gone (404) counts as success.
- The picker keeps a pure `watchSelection`: the components the app watches now (baseline) and
  the chosen set. The API replaces the whole watched set on save, so what is sent is the whole
  chosen set, including components of providers that were never opened; never build the request
  from what is on screen. `componentRefs()` is never nil, because an empty selection must reach
  the API as `[]`.
- The API allows 60 requests a minute per key and the catalog has 87 providers, so the picker
  must not ask on every cursor move. Moving over a provider only schedules a
  `providerSettledMsg` (`providerSettleDelay`, with a sequence number like the refresh ticks);
  the fetch happens when the cursor has rested there. Search applies on `enter` for the same
  reason.
- Answers live in a `componentCache` owned by `RootModel` and shared by every picker opened
  during the run: keyed by `componentListing` (provider and search), trusted for
  `cacheLifetime`, and also holding the provider list. It records requests in flight (so going
  back to a provider before its answer lands asks nothing) and failures (a refused request, such
  as a 429, is not repeated by visiting again; `r` asks again). The root files answers that
  arrive after the picker closed (`keepAnswerWithoutPicker`); otherwise a listing would stay
  pending for good. A response carries its listing, so one for a provider the user already left
  is cached but not shown.
- Picker search applies on `enter`, not per keystroke, to spare the API's rate limit. While the
  search box has focus every key is text; `esc` clears the search.
- The picker's `esc` asks first when there are unsaved changes (a second `esc` discards), and
  is ignored while a save is in flight, for the same reason as the dialogs.
- Refresh ticks carry a sequence number. A manual refresh bumps it so a pending tick from
  before is ignored; without that you get two refresh loops.
- Pad before styling. `fmt` counts colour escape codes, so `%-16s` on a styled string
  misaligns columns. Pad the plain label, then render it (`renderStatus`).
- Don't read the real clock in rendering code; the detail model holds `now` so tests can fix it.
- `refreshInterval` and `fetchTimeout` are named constants; keep durations out of call sites.

## Testing

Drive `Update` with messages and assert on `View().Content`: no terminal, no goroutines.
Substring checks survive styling because escape codes wrap, not split, the text.
