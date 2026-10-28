// Package ui holds the Bubble Tea models for the TUI's screens.
package ui

import (
	"context"
	"fmt"
	"time"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"

	"github.com/statuspane/tui/internal/api"
)

const (
	refreshInterval = 30 * time.Second
	fetchTimeout    = 15 * time.Second
)

var appsHints = []keyHint{
	{"↑↓", "move"}, {"enter", "open"}, {"n", "new app"}, {"e", "rename"}, {"d", "delete"}, {"r", "refresh"},
}

// AppLister is the one thing the screen needs from the API client; tests pass a fake.
type AppLister interface {
	ListApps(ctx context.Context) ([]api.App, error)
}

// AppHistoryGetter is what the apps screen needs to draw the selected app's history.
type AppHistoryGetter interface {
	GetAppHistory(ctx context.Context, slug string, days int) (api.AppHistory, error)
}

// AppsBackend is what the apps screen needs from the API client: the list, and the selected
// app's detail and history.
type AppsBackend interface {
	AppLister
	AppGetter
	AppHistoryGetter
}

// openAppMsg asks the root model to show one app's detail screen.
type openAppMsg struct{ slug, name string }

// appsChangedMsg tells the list its data is stale: an app was created or deleted.
type appsChangedMsg struct{}

type appsLoadedMsg struct{ apps []api.App }

type appsFailedMsg struct{ err error }

// refreshTickMsg carries the refresh sequence it was scheduled under, so a tick
// left over from before a manual refresh is ignored instead of starting a second loop.
type refreshTickMsg struct{ sequence int }

type AppsModel struct {
	backend         AppsBackend
	apps            []api.App
	selectedDetail  selectedDetail
	now             func() time.Time
	err             error
	hasLoaded       bool
	isLoading       bool
	selectedIndex   int
	refreshSequence int
	reloadAfterLoad bool
	spinner         spinner.Model
	width, height   int
	// hasDarkBackground picks the selected row's tint; dark until the terminal says otherwise.
	hasDarkBackground bool
}

func NewAppsModel(backend AppsBackend) AppsModel {
	return AppsModel{backend: backend, selectedDetail: newSelectedDetail(), now: time.Now, isLoading: true, spinner: spinner.New(), hasDarkBackground: true}
}

func (m AppsModel) Init() tea.Cmd {
	return tea.Batch(m.fetchCmd(), clockTickCmd())
}

func (m AppsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil

	case tea.BackgroundColorMsg:
		m.hasDarkBackground = msg.IsDark()
		return m, nil

	case tea.KeyPressMsg:
		return m.handleKey(msg)

	case appsLoadedMsg:
		m.apps, m.err, m.hasLoaded, m.isLoading = msg.apps, nil, true, false
		m.selectedIndex = min(m.selectedIndex, max(len(m.apps)-1, 0))
		next, cmd := m.reloadOrScheduleRefresh()
		return next, tea.Batch(cmd, m.fetchSelectedAppCmd())

	case clockTickMsg:
		return m, clockTickCmd()

	case detailSettledMsg:
		if !m.selectedDetail.hasRested(msg) {
			return m, nil
		}
		return m, m.fetchSelectedAppCmd()

	case selectedDetailLoadedMsg:
		m.selectedDetail = m.selectedDetail.withLoaded(msg)
		return m, nil

	case selectedHistoryLoadedMsg:
		m.selectedDetail = m.selectedDetail.withHistoryLoaded(msg)
		return m, nil

	case selectedHistoryFailedMsg:
		m.selectedDetail = m.selectedDetail.withHistoryFailed(msg)
		return m, nil

	case appsFailedMsg:
		m.err, m.isLoading = msg.err, false
		return m.reloadOrScheduleRefresh()

	case appsChangedMsg:
		if m.isLoading {
			// The fetch in flight may predate the change; load again as soon as it lands.
			m.reloadAfterLoad = true
			return m, nil
		}
		m.refreshSequence++
		return m.startRefresh()

	case refreshTickMsg:
		if msg.sequence != m.refreshSequence || m.isLoading {
			return m, nil
		}
		return m.startRefresh()

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m AppsModel) handleKey(keyPress tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch keyPress.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "up", "k":
		return m.movedTo(max(m.selectedIndex-1, 0))
	case "down", "j":
		return m.movedTo(min(m.selectedIndex+1, max(len(m.apps)-1, 0)))
	case "enter":
		app, ok := m.selectedApp()
		if !ok {
			return m, nil
		}
		return m, func() tea.Msg { return openAppMsg{slug: app.Slug, name: app.Name} }
	case "n":
		return m, func() tea.Msg { return openCreateAppDialogMsg{} }
	case "e":
		app, ok := m.selectedApp()
		if !ok {
			return m, nil
		}
		return m, func() tea.Msg { return openRenameAppDialogMsg{slug: app.Slug, name: app.Name} }
	case "d":
		app, ok := m.selectedApp()
		if !ok {
			return m, nil
		}
		return m, func() tea.Msg {
			return openDeleteAppDialogMsg{slug: app.Slug, name: app.Name, componentCount: app.WatchedComponentsCount}
		}
	case "r":
		if m.isLoading {
			return m, nil
		}
		m.refreshSequence++
		return m.startRefresh()
	}
	return m, nil
}

// movedTo selects another app and waits for the selection to rest before asking for its detail.
func (m AppsModel) movedTo(index int) (tea.Model, tea.Cmd) {
	if index == m.selectedIndex {
		return m, nil
	}
	m.selectedIndex = index
	var restCmd tea.Cmd
	m.selectedDetail, restCmd = m.selectedDetail.afterSelectionMoved()
	return m, restCmd
}

func (m AppsModel) selectedApp() (api.App, bool) {
	if len(m.apps) == 0 {
		return api.App{}, false
	}
	return m.apps[m.selectedIndex], true
}

func (m AppsModel) startRefresh() (AppsModel, tea.Cmd) {
	m.isLoading = true
	return m, m.fetchCmd()
}

func (m AppsModel) fetchCmd() tea.Cmd {
	return tea.Batch(m.spinner.Tick, fetchApps(m.backend))
}

func (m AppsModel) reloadOrScheduleRefresh() (AppsModel, tea.Cmd) {
	if m.reloadAfterLoad {
		m.reloadAfterLoad = false
		m.refreshSequence++
		return m.startRefresh()
	}
	return m.scheduleRefresh()
}

func (m AppsModel) scheduleRefresh() (AppsModel, tea.Cmd) {
	m.refreshSequence++
	sequence := m.refreshSequence
	return m, tea.Tick(refreshInterval, func(time.Time) tea.Msg {
		return refreshTickMsg{sequence: sequence}
	})
}

func (m AppsModel) View() tea.View {
	screenWidth, screenHeight := m.windowSize()
	width, panelHeight := layoutWidth(screenWidth), bodyHeight(screenHeight)
	header := headerBar(accentStyle.Render(brandText)+summaryGap+statusSummary(m.apps), m.clockText(), width)
	return tea.NewView(stackedScreen(header, m.missionControl(width, panelHeight), footerBar(appsHints, m.footerTrailing(), width)))
}

func (m AppsModel) windowSize() (width, height int) {
	return withFallbackSize(m.width, m.height)
}

func (m AppsModel) footerTrailing() string {
	return mutedStyle.Render(fmt.Sprintf("refreshes every %ds · ", int(refreshInterval.Seconds()))) + renderHint(keyHint{"q", "quit"})
}

// refreshNotice reports a failed or running refresh of apps that are already on screen.
func (m AppsModel) refreshNotice() string {
	switch {
	case m.err != nil:
		return errorStyle.Render("Refresh failed: " + errorMessage(m.err))
	case m.isLoading:
		return m.spinner.View() + " Refreshing…"
	}
	return ""
}

// fetchSelectedAppCmd asks for the selected app's detail and its history, as two requests.
func (m AppsModel) fetchSelectedAppCmd() tea.Cmd {
	app, ok := m.selectedApp()
	if !ok {
		return nil
	}
	return tea.Batch(fetchSelectedAppDetail(m.backend, app), fetchSelectedAppHistory(m.backend, app))
}

func fetchApps(appLister AppLister) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
		defer cancel()

		apps, err := appLister.ListApps(ctx)
		if err != nil {
			return appsFailedMsg{err: err}
		}
		return appsLoadedMsg{apps: apps}
	}
}
