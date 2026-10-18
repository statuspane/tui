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
- A rename can change the app's slug. `appRenamedMsg` carries the slug it was sent for
  (`previousSlug`) so the root can find the open app screen, which then takes the response's
  slug, name and content like a freshly loaded app. An unchanged name closes the rename dialog
  without a request; a blank one leaves it open, as in the create dialog.
- The picker keeps a pure `watchSelection`: the components the app watches now (baseline) and
  the chosen set. The API replaces the whole watched set on save, so what is sent is the whole
  chosen set, including components of providers that were never opened; never build the request
  from what is on screen. `componentRefs()` is never nil, because an empty selection must reach
  the API as `[]`.
- `/` filters the pane that has the focus, live and in memory, so typing makes no request. Each
  pane has a `paneFilter` (`providersFilter`, `componentsFilter`): the text box, the query kept in
  it and whether it has the focus. While a box has the focus every key is text; `enter` keeps the
  query, `esc` clears it. Providers are narrowed by `filterProviders` (name, slug, category) and
  the cursor moves over `visibleProviders()`, never over `providers` directly; the pane's width
  comes from the full list so it does not jump while the user types. Components are narrowed by
  `filterComponents` (name or group, the rule the API applies to its own `q`) over the provider's
  full cached list, and the cursor moves over `visibleComponents()`. The TUI never sends `q`. The
  filters trim and lower-case the query themselves, so the box and the match always agree. A
  component filter belongs to the provider it was typed for: `landOnProvider` clears it. What a
  save sends is the chosen set, not what the filters show, so a hidden watched component stays.
- Components are shown under group headings, but the list the cursor, `space`, `g`, the filter
  and the save work on stays flat: `visibleComponents()` is the filtered list put in group order
  by `orderedByGroup` (groups alphabetical, no group last, API order inside a group), and the
  headings are only drawn by `linesWithHeadings`, which has no say in what the cursor can reach. A
  provider where nothing has a group gets no headings. `g` acts on the visible members of the
  group (`groupBounds`), never on components a filter hides. Never sort or reorder the cached
  slice: `filterComponents` can hand back the cache's own slice, and `orderedByGroup` copies it.
- A component row shows its state twice: the marker text (`[x]`, `[+]`, `[-]`, `[ ]`, which still
  reads without colour) and `nameStyleFor` across the name. The row under the cursor in the
  focused pane is one `cursorRowStyle` bar drawn from plain text: styles nested inside a bar end
  it early, so it carries no state colours. In tests, read rows with `plain()` or `rawRow()`:
  lipgloss applies strikethrough to each character on its own, so a struck-through name is not
  contiguous in the raw view.
- The API allows 60 requests a minute per key and the catalog has 87 providers, so the picker
  must not ask on every cursor move. Moving over a provider only schedules a
  `providerSettledMsg` (`providerSettleDelay`, with a sequence number like the refresh ticks);
  the fetch happens when the cursor has rested there.
- Answers live in a `componentCache` owned by `RootModel` and shared by every picker opened
  during the run: keyed by the provider's slug, trusted for
  `cacheLifetime`, and also holding the provider list. It records requests in flight (so going
  back to a provider before its answer lands asks nothing) and failures (a refused request, such
  as a 429, is not repeated by visiting again; `r` asks again). The root files answers that
  arrive after the picker closed (`keepAnswerWithoutPicker`); otherwise a listing would stay
  pending for good. A response names its provider, so one for a provider the user already left
  is cached but not shown.
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
