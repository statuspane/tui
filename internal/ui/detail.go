package ui

import (
	"context"
	"time"

	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"

	"github.com/statuspane/tui/internal/api"
)

const (
	// Rows around the scrolling content: title, status line, blank, key help.
	detailChromeHeight = 4

	fallbackWidth  = 80
	fallbackHeight = 24
)

// AppGetter is the one thing the detail screen needs from the API client; tests pass a fake.
type AppGetter interface {
	GetApp(ctx context.Context, slug string) (api.AppDetail, error)
}

// closeDetailMsg asks the root model to go back to the list.
type closeDetailMsg struct{}

type appDetailLoadedMsg struct{ appDetail api.AppDetail }

type appDetailFailedMsg struct{ err error }

// detailRefreshTickMsg is guarded by a sequence number like refreshTickMsg.
type detailRefreshTickMsg struct{ sequence int }

type AppDetailModel struct {
	appGetter       AppGetter
	slug            string
	name            string
	appDetail       api.AppDetail
	err             error
	hasLoaded       bool
	isLoading       bool
	refreshSequence int
	spinner         spinner.Model
	viewport        viewport.Model
	now             func() time.Time
}

// NewAppDetailModel shows name until the app has loaded. A zero width or height
// (no window size received yet) falls back to a classic 80x24 terminal.
func NewAppDetailModel(appGetter AppGetter, slug, name string, width, height int) AppDetailModel {
	if width <= 0 {
		width = fallbackWidth
	}
	if height <= 0 {
		height = fallbackHeight
	}
	content := viewport.New(viewport.WithWidth(width), viewport.WithHeight(max(height-detailChromeHeight, 1)))
	content.SoftWrap = true

	return AppDetailModel{
		appGetter: appGetter,
		slug:      slug,
		name:      name,
		isLoading: true,
		spinner:   spinner.New(),
		viewport:  content,
		now:       time.Now,
	}
}

func (m AppDetailModel) Init() tea.Cmd {
	return m.fetchCmd()
}

func (m AppDetailModel) Update(msg tea.Msg) (AppDetailModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		return m.handleKey(msg)

	case tea.WindowSizeMsg:
		m.viewport.SetWidth(msg.Width)
		m.viewport.SetHeight(max(msg.Height-detailChromeHeight, 1))
		return m, nil

	case appDetailLoadedMsg:
		// The API answers with the app's current slug, which may differ after a rename.
		m.appDetail, m.slug, m.name = msg.appDetail, msg.appDetail.Slug, msg.appDetail.Name
		m.err, m.hasLoaded, m.isLoading = nil, true, false
		m.viewport.SetContent(renderAppDetail(m.appDetail, m.now()))
		return m.scheduleRefresh()

	case appDetailFailedMsg:
		m.err, m.isLoading = msg.err, false
		return m.scheduleRefresh()

	case detailRefreshTickMsg:
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

func (m AppDetailModel) handleKey(keyPress tea.KeyPressMsg) (AppDetailModel, tea.Cmd) {
	switch keyPress.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "esc":
		return m, func() tea.Msg { return closeDetailMsg{} }
	case "up", "k":
		m.viewport.ScrollUp(1)
	case "down", "j":
		m.viewport.ScrollDown(1)
	case "pgup":
		m.viewport.PageUp()
	case "pgdown":
		m.viewport.PageDown()
	case "w":
		if !m.hasLoaded {
			return m, nil
		}
		msg := openWatchPickerMsg{appSlug: m.slug, appName: m.name, watchedComponents: m.appDetail.WatchedComponents}
		return m, func() tea.Msg { return msg }
	case "e":
		if !m.hasLoaded {
			return m, nil
		}
		return m, func() tea.Msg { return openRenameAppDialogMsg{slug: m.slug, name: m.name} }
	case "d":
		if !m.hasLoaded {
			return m, nil
		}
		msg := openDeleteAppDialogMsg{slug: m.slug, name: m.name, componentCount: len(m.appDetail.WatchedComponents)}
		return m, func() tea.Msg { return msg }
	case "r":
		if m.isLoading {
			return m, nil
		}
		m.refreshSequence++
		return m.startRefresh()
	}
	return m, nil
}

// shows reports whether the screen is on that app, by the slug it currently knows.
func (m AppDetailModel) shows(appSlug string) bool {
	return m.slug == appSlug
}

func (m AppDetailModel) startRefresh() (AppDetailModel, tea.Cmd) {
	m.isLoading = true
	return m, m.fetchCmd()
}

func (m AppDetailModel) fetchCmd() tea.Cmd {
	return tea.Batch(m.spinner.Tick, fetchAppDetail(m.appGetter, m.slug))
}

func (m AppDetailModel) scheduleRefresh() (AppDetailModel, tea.Cmd) {
	m.refreshSequence++
	sequence := m.refreshSequence
	return m, tea.Tick(refreshInterval, func(time.Time) tea.Msg {
		return detailRefreshTickMsg{sequence: sequence}
	})
}

func (m AppDetailModel) View() tea.View {
	title := titleStyle.Render("StatusPane") + mutedStyle.Render(" › ") + titleStyle.Render(m.name)
	help := mutedStyle.Render("↑/↓ scroll · w watch components · e rename · d delete · r refresh · esc back · q quit")
	return tea.NewView(title + "\n" + m.statusLine() + "\n" + m.viewport.View() + "\n\n" + help + "\n")
}

func (m AppDetailModel) statusLine() string {
	switch {
	case m.isLoading && m.hasLoaded:
		return m.spinner.View() + " Refreshing…"
	case m.isLoading:
		return m.spinner.View() + " Loading…"
	case m.err != nil && m.hasLoaded:
		return errorStyle.Render("Refresh failed: " + errorMessage(m.err))
	case m.err != nil:
		return errorStyle.Render(errorMessage(m.err))
	}
	return ""
}

func fetchAppDetail(appGetter AppGetter, slug string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
		defer cancel()

		appDetail, err := appGetter.GetApp(ctx, slug)
		if err != nil {
			return appDetailFailedMsg{err: err}
		}
		return appDetailLoadedMsg{appDetail: appDetail}
	}
}
