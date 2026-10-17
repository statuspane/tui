package ui

import tea "charm.land/bubbletea/v2"

// Backend is everything the screens need from the API client.
type Backend interface {
	AppLister
	AppGetter
	AppCreator
	AppDeleter
}

// RootModel shows the apps list, or an app's detail on top of it, or a dialog instead of
// either. The list stays alive underneath, so going back keeps the selection and its refresh timer.
type RootModel struct {
	backend Backend
	apps    AppsModel
	detail  *AppDetailModel
	dialog  dialog
	width   int
	height  int
}

func NewRootModel(backend Backend) RootModel {
	return RootModel{backend: backend, apps: NewAppsModel(backend)}
}

func (m RootModel) Init() tea.Cmd {
	return m.apps.Init()
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
	case openCreateAppDialogMsg:
		createDialog, focus := newCreateAppDialog(m.backend)
		m.dialog = createDialog
		return m, focus
	case openDeleteAppDialogMsg:
		deleteDialog, focus := newDeleteAppDialog(m.backend, msg)
		m.dialog = deleteDialog
		return m, focus
	case closeDialogMsg:
		m.dialog = nil
		return m, nil
	case appCreatedMsg:
		m.dialog = nil
		opened, openCmd := m.openDetail(msg.appDetail.Slug, msg.appDetail.Name)
		return opened, tea.Batch(openCmd, appsChanged)
	case appDeletedMsg:
		m.dialog = nil
		if m.detail != nil && m.detail.slug == msg.slug {
			m.detail = nil
		}
		return m, appsChanged
	case tea.KeyPressMsg:
		return m.updateActiveScreen(msg)
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
	if m.dialog != nil {
		updatedDialog, dialogCmd := m.dialog.update(msg)
		m.dialog = updatedDialog
		cmds = append(cmds, dialogCmd)
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

func (m RootModel) View() tea.View {
	if m.dialog != nil {
		return tea.NewView(titleStyle.Render("StatusPane") + "\n\n" + m.dialog.view() + "\n")
	}
	if m.detail != nil {
		return m.detail.View()
	}
	return m.apps.View()
}
