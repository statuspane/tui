package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/statuspane/tui/internal/api"
)

const (
	// The incidents pane gets this many parts of incidentsAndComponentsParts, the components the rest.
	incidentsPaneParts          = 11
	incidentsAndComponentsParts = 21
	paneGapWidth                = 2
	moreHintLines               = 1
)

// detailFocus is the pane of the app screen that has the focus.
type detailFocus int

const (
	focusIncidents detailFocus = iota
	focusComponents
)

var detailHints = []keyHint{
	{"w", "watch components"}, {"e", "rename"}, {"d", "delete"}, {"r", "refresh"}, {"↑↓", "scroll"}, {"tab", "switch pane"},
}

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
	now             func() time.Time

	incidentsViewport  viewport.Model
	componentsViewport viewport.Model
	focus              detailFocus
	width, height      int
}

// NewAppDetailModel shows name until the app has loaded. A zero width or height
// (no window size received yet) falls back to a classic 80x24 terminal.
func NewAppDetailModel(appGetter AppGetter, slug, name string, width, height int) AppDetailModel {
	m := AppDetailModel{
		appGetter: appGetter,
		slug:      slug,
		name:      name,
		isLoading: true,
		spinner:   spinner.New(),
		now:       time.Now,
		focus:     focusComponents,
		width:     width,
		height:    height,
	}
	m.incidentsViewport.SoftWrap = true
	return m.resized()
}

func (m AppDetailModel) Init() tea.Cmd {
	return m.fetchCmd()
}

func (m AppDetailModel) Update(msg tea.Msg) (AppDetailModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		return m.handleKey(msg)

	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m.resized(), nil

	case appDetailLoadedMsg:
		// The API answers with the app's current slug, which may differ after a rename.
		m.appDetail, m.slug, m.name = msg.appDetail, msg.appDetail.Slug, msg.appDetail.Name
		m.err, m.hasLoaded, m.isLoading = nil, true, false
		m = m.withContent()
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
	case "tab":
		m.focus = otherFocus(m.focus)
	case "up", "k":
		m.focusedViewport().ScrollUp(1)
	case "down", "j":
		m.focusedViewport().ScrollDown(1)
	case "pgup":
		m.focusedViewport().PageUp()
	case "pgdown":
		m.focusedViewport().PageDown()
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
	screenWidth, screenHeight := withFallbackSize(m.width, m.height)
	width, panelHeight := layoutWidth(screenWidth), bodyHeight(screenHeight)
	incidentsWidth, componentsWidth := paneWidths(width)

	header := headerBar(m.headerLeft(), m.headerRight(), width)
	panes := lipgloss.JoinHorizontal(lipgloss.Top,
		m.componentsPanel(componentsWidth, panelHeight), strings.Repeat(" ", paneGapWidth), m.incidentsPanel(incidentsWidth, panelHeight))
	return tea.NewView(stackedScreen(header, panes, footerBar(detailHints, m.footerTrailing(), width)))
}

func (m AppDetailModel) footerTrailing() string {
	return renderHint(keyHint{"esc", "back"}) + mutedStyle.Render(" · ") + renderHint(keyHint{"q", "quit"})
}

// headerLeft is the breadcrumb, with the app's status once it is known.
func (m AppDetailModel) headerLeft() string {
	breadcrumb := accentStyle.Render(brandText) + mutedStyle.Render(" › ") + titleStyle.Render(m.name)
	if !m.hasLoaded {
		return breadcrumb
	}
	appearance := appearanceOf(m.appDetail.Status)
	return breadcrumb + "  " + appearance.style.Render(bullet+appearance.label)
}

// headerRight is how fresh the data is, or what the screen is doing about it.
func (m AppDetailModel) headerRight() string {
	if !m.hasLoaded {
		return ""
	}
	if line := m.statusLine(); line != "" {
		return line
	}
	return mutedStyle.Render(checkedText(m.appDetail.LastCheckedAt, m.now()))
}

func (m AppDetailModel) incidentsPanel(width, height int) string {
	title := "Active incidents"
	if m.hasLoaded {
		title = fmt.Sprintf("Active incidents (%d)", len(m.appDetail.ActiveIncidents))
	}
	look := incidentsPanelLook(m.appDetail.ActiveIncidents, m.focus == focusIncidents)
	return panelWithLook(look, title, m.paneBody(m.incidentsViewport, m.statusLine()), width, height)
}

func (m AppDetailModel) componentsPanel(width, height int) string {
	title := "Watched components"
	if m.hasLoaded {
		title = fmt.Sprintf("Watched components (%d)", len(m.appDetail.WatchedComponents))
	}
	return panelWithLook(focusLook(m.focus == focusComponents), title, m.paneBody(m.componentsViewport, ""), width, height)
}

// paneBody is a pane's scrolling content and, under it, how much more there is below. Until the
// app has loaded the pane shows beforeLoad instead.
func (m AppDetailModel) paneBody(content viewport.Model, beforeLoad string) string {
	if !m.hasLoaded {
		return beforeLoad
	}
	return content.View() + "\n" + moreHint(content)
}

// moreHint is "↓ N more" while there are lines below the visible ones, and blank otherwise, so
// the pane doesn't change height as it scrolls.
func moreHint(content viewport.Model) string {
	linesBelow := content.TotalLineCount() - content.YOffset() - content.VisibleLineCount()
	if linesBelow <= 0 {
		return ""
	}
	return mutedStyle.Render(fmt.Sprintf("↓ %d more", linesBelow))
}

// paneWidths splits the screen width between the panes, leaving the gap between them.
func paneWidths(width int) (incidentsWidth, componentsWidth int) {
	paneSpace := width - paneGapWidth
	incidentsWidth = paneSpace * incidentsPaneParts / incidentsAndComponentsParts
	return incidentsWidth, paneSpace - incidentsWidth
}

func otherFocus(focus detailFocus) detailFocus {
	if focus == focusIncidents {
		return focusComponents
	}
	return focusIncidents
}

func (m *AppDetailModel) focusedViewport() *viewport.Model {
	if m.focus == focusIncidents {
		return &m.incidentsViewport
	}
	return &m.componentsViewport
}

// resized sizes both viewports to fit their panes and fills them again, because the components
// are laid out to their pane's width.
func (m AppDetailModel) resized() AppDetailModel {
	screenWidth, screenHeight := withFallbackSize(m.width, m.height)
	incidentsWidth, componentsWidth := paneWidths(layoutWidth(screenWidth))
	panelHeight := bodyHeight(screenHeight)

	for _, pane := range []struct {
		content *viewport.Model
		width   int
	}{{&m.incidentsViewport, incidentsWidth}, {&m.componentsViewport, componentsWidth}} {
		contentWidth, contentHeight := panelContentSize(pane.width, panelHeight)
		pane.content.SetWidth(contentWidth)
		pane.content.SetHeight(max(contentHeight-moreHintLines, 1))
	}
	return m.withContent()
}

// withContent fills the viewports from the app. A viewport keeps its scroll position when its
// content changes, so a refresh doesn't send either pane back to the top.
func (m AppDetailModel) withContent() AppDetailModel {
	m.incidentsViewport.SetContent(renderIncidents(m.appDetail.ActiveIncidents, m.now()))
	m.componentsViewport.SetContent(renderComponents(m.appDetail.WatchedComponents, m.componentsViewport.Width()))
	return m
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
