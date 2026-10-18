package ui

import (
	"time"

	tea "charm.land/bubbletea/v2"
)

// Backend is everything the screens need from the API client.
type Backend interface {
	AppLister
	AppGetter
	AppCreator
	AppRenamer
	AppDeleter
	WatchPickerBackend
}

// RootModel shows the apps list, or an app's detail on top of it, or the component picker on
// top of that, or a dialog instead of any of them. The screens below stay alive, so going back
// keeps the selection and the refresh timers.
type RootModel struct {
	backend Backend
	apps    AppsModel
	detail  *AppDetailModel
	picker  *WatchPickerModel
	dialog  dialog
	cache   *componentCache
	width   int
	height  int
}

func NewRootModel(backend Backend) RootModel {
	return RootModel{backend: backend, apps: NewAppsModel(backend), cache: newComponentCache(time.Now)}
}

func (m RootModel) Init() tea.Cmd {
	return tea.Batch(m.apps.Init(), tea.RequestBackgroundColor)
}

func (m RootModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case openAppMsg:
		return m.openDetail(msg.slug, msg.name)
	case closeDetailMsg:
		m.detail = nil
		return m, nil
	case openWatchPickerMsg:
		picker := NewWatchPickerModel(m.backend, m.cache, msg, m.width, m.height)
		m.picker = &picker
		return m, picker.Init()
	case closeWatchPickerMsg:
		m.picker = nil
		return m, nil
	case watchedComponentsSavedMsg:
		return m.closePickerAfterSave(msg)
	case openCreateAppDialogMsg:
		createDialog, focusCmd := newCreateAppDialog(m.backend)
		m.dialog = createDialog
		return m, focusCmd
	case openRenameAppDialogMsg:
		renameDialog, focusCmd := newRenameAppDialog(m.backend, msg)
		m.dialog = renameDialog
		return m, focusCmd
	case openDeleteAppDialogMsg:
		deleteDialog, focusCmd := newDeleteAppDialog(m.backend, msg)
		m.dialog = deleteDialog
		return m, focusCmd
	case closeDialogMsg:
		m.dialog = nil
		return m, nil
	case appCreatedMsg:
		m.dialog = nil
		opened, openCmd := m.openDetail(msg.appDetail.Slug, msg.appDetail.Name)
		return opened, tea.Batch(openCmd, appsChanged)
	case appRenamedMsg:
		return m.applyRename(msg)
	case appDeletedMsg:
		m.dialog = nil
		if m.detail != nil && m.detail.shows(msg.slug) {
			m.detail = nil
		}
		return m, appsChanged
	case tea.KeyPressMsg:
		return m.updateActiveScreen(msg)
	case providersLoadedMsg:
		m.keepAnswerWithoutPicker(msg)
	case providerComponentsLoadedMsg:
		m.keepAnswerWithoutPicker(msg)
	case providerComponentsFailedMsg:
		m.keepAnswerWithoutPicker(msg)
	}

	// Everything else (fetch results, ticks) goes to every screen; each ignores what isn't its own.
	updatedApps, appsCmd := m.apps.Update(msg)
	m.apps = updatedApps.(AppsModel)
	cmds := []tea.Cmd{appsCmd}
	if m.detail != nil {
		updatedDetail, detailCmd := m.detail.Update(msg)
		m.detail = &updatedDetail
		cmds = append(cmds, detailCmd)
	}
	if m.picker != nil {
		updatedPicker, pickerCmd := m.picker.Update(msg)
		m.picker = &updatedPicker
		cmds = append(cmds, pickerCmd)
	}
	if m.dialog != nil {
		updatedDialog, dialogCmd := m.dialog.update(msg)
		m.dialog = updatedDialog
		cmds = append(cmds, dialogCmd)
	}
	return m, tea.Batch(cmds...)
}

// keepAnswerWithoutPicker files an answer that arrives after the picker was closed. Without it a
// request that was in flight at that moment would stay "pending" in the shared cache for good,
// and its answer would be thrown away.
func (m RootModel) keepAnswerWithoutPicker(msg tea.Msg) {
	if m.picker != nil {
		return
	}
	switch msg := msg.(type) {
	case providersLoadedMsg:
		m.cache.storeProviders(msg.providers)
	case providerComponentsLoadedMsg:
		m.cache.store(msg.providerSlug, msg.components)
	case providerComponentsFailedMsg:
		m.cache.fail(msg.providerSlug, msg.err)
	}
}

// closePickerAfterSave gives the app screen the app as the API now has it, and tells the list,
// whose component count changed.
func (m RootModel) closePickerAfterSave(msg watchedComponentsSavedMsg) (tea.Model, tea.Cmd) {
	m.picker = nil
	cmds := []tea.Cmd{appsChanged}
	if m.detail != nil {
		// The saved app reaches the screen the way a freshly loaded one does; both messages carry just the app.
		updatedDetail, detailCmd := m.detail.Update(appDetailLoadedMsg(msg))
		m.detail = &updatedDetail
		cmds = append(cmds, detailCmd)
	}
	return m, tea.Batch(cmds...)
}

// applyRename gives the app screen, if it is on the renamed app, the app as the API now
// has it, and tells the list, whose row changed.
func (m RootModel) applyRename(msg appRenamedMsg) (tea.Model, tea.Cmd) {
	m.dialog = nil
	cmds := []tea.Cmd{appsChanged}
	if m.detail != nil && m.detail.shows(msg.previousSlug) {
		updatedDetail, detailCmd := m.detail.Update(appDetailLoadedMsg{appDetail: msg.appDetail})
		m.detail = &updatedDetail
		cmds = append(cmds, detailCmd)
	}
	return m, tea.Batch(cmds...)
}

func (m RootModel) openDetail(appSlug, name string) (RootModel, tea.Cmd) {
	detail := NewAppDetailModel(m.backend, appSlug, name, m.width, m.height)
	m.detail = &detail
	return m, detail.Init()
}

func appsChanged() tea.Msg { return appsChangedMsg{} }

func (m RootModel) updateActiveScreen(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.picker != nil {
		updatedPicker, cmd := m.picker.Update(msg)
		m.picker = &updatedPicker
		return m, cmd
	}
	if m.dialog != nil {
		updatedDialog, cmd := m.dialog.update(msg)
		m.dialog = updatedDialog
		return m, cmd
	}
	if m.detail != nil {
		updatedDetail, cmd := m.detail.Update(msg)
		m.detail = &updatedDetail
		return m, cmd
	}
	updatedApps, cmd := m.apps.Update(msg)
	m.apps = updatedApps.(AppsModel)
	return m, cmd
}

// View takes the whole terminal: the screens are laid out to the window size, and a frame as
// tall as the window would otherwise scroll the shell's history.
func (m RootModel) View() tea.View {
	view := m.currentView()
	view.AltScreen = true
	return view
}

func (m RootModel) currentView() tea.View {
	if m.picker != nil {
		return m.picker.View()
	}
	if m.dialog != nil {
		return tea.NewView(titleStyle.Render("StatusPane") + "\n\n" + m.dialog.view() + "\n")
	}
	if m.detail != nil {
		return m.detail.View()
	}
	return m.apps.View()
}
