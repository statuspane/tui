package ui

import tea "charm.land/bubbletea/v2"

// Backend is everything the screens need from the API client.
type Backend interface {
	AppLister
	AppGetter
}

// RootModel shows the apps list, or an app's detail on top of it. The list stays
// alive underneath, so going back keeps the selection and its refresh timer.
type RootModel struct {
	backend Backend
	apps    AppsModel
	detail  *AppDetailModel
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
		detail := NewAppDetailModel(m.backend, msg.slug, msg.name, m.width, m.height)
		m.detail = &detail
		return m, detail.Init()
	case closeDetailMsg:
		m.detail = nil
		return m, nil
	case tea.KeyPressMsg:
		return m.updateActiveScreen(msg)
	}

	// Everything else (fetch results, ticks) goes to both; each ignores what isn't its own.
	updatedApps, appsCmd := m.apps.Update(msg)
	m.apps = updatedApps.(AppsModel)
	if m.detail == nil {
		return m, appsCmd
	}
	updatedDetail, detailCmd := m.detail.Update(msg)
	m.detail = &updatedDetail
	return m, tea.Batch(appsCmd, detailCmd)
}

func (m RootModel) updateActiveScreen(msg tea.Msg) (tea.Model, tea.Cmd) {
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
	if m.detail != nil {
		return m.detail.View()
	}
	return m.apps.View()
}
