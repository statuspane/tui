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
	// The panes share the width in these proportions.
	incidentsPaneParts  = 11
	componentsPaneParts = 10
	paneGapWidth        = 2
	moreHintLines       = 1
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

	incidentsViewport         viewport.Model
	componentsViewport        viewport.Model
	focus                     detailFocus
	windowWidth, windowHeight int
}

// NewAppDetailModel shows name until the app has loaded. A zero width or height (no window
// size received yet) is drawn as a classic 80x24 terminal.
func NewAppDetailModel(appGetter AppGetter, slug, name string, width, height int) AppDetailModel {
	m := AppDetailModel{
		appGetter:    appGetter,
		slug:         slug,
		name:         name,
		isLoading:    true,
		spinner:      spinner.New(),
		now:          time.Now,
		focus:        focusComponents,
		windowWidth:  width,
		windowHeight: height,
	}
	m.incidentsViewport.SoftWrap = true
	return m.sizedToWindow()
}

func (m AppDetailModel) Init() tea.Cmd {
	return m.fetchCmd()
}

func (m AppDetailModel) Update(msg tea.Msg) (AppDetailModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		return m.handleKey(msg)

	case tea.WindowSizeMsg:
		m.windowWidth, m.windowHeight = msg.Width, msg.Height
		return m.sizedToWindow(), nil

	case appDetailLoadedMsg:
		// The API answers with the app's current slug, which may differ after a rename.
		m.appDetail, m.slug, m.name = msg.appDetail, msg.appDetail.Slug, msg.appDetail.Name
		m.err, m.hasLoaded, m.isLoading = nil, true, false
		m = m.withViewportContents()
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
		return m.scrolled(func(paneViewport *viewport.Model) { paneViewport.ScrollUp(1) }), nil
	case "down", "j":
		return m.scrolled(func(paneViewport *viewport.Model) { paneViewport.ScrollDown(1) }), nil
	case "pgup":
		return m.scrolled(func(paneViewport *viewport.Model) { paneViewport.PageUp() }), nil
	case "pgdown":
		return m.scrolled(func(paneViewport *viewport.Model) { paneViewport.PageDown() }), nil
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
	layout := m.paneLayout()

	header := headerBar(m.headerLeft(), m.headerRight(), layout.contentWidth)
	panes := lipgloss.JoinHorizontal(lipgloss.Top,
		m.componentsPanel(layout.componentsWidth, layout.paneHeight), strings.Repeat(" ", paneGapWidth), m.incidentsPanel(layout.incidentsWidth, layout.paneHeight))
	return tea.NewView(stackedScreen(header, panes, footerBar(detailHints, m.footerTrailing(), layout.contentWidth)))
}

// paneLayout is how the window is shared out: the width inside the margin, each pane's width and
// the panes' height.
type paneLayout struct{ contentWidth, incidentsWidth, componentsWidth, paneHeight int }

func (m AppDetailModel) paneLayout() paneLayout {
	windowWidth, windowHeight := withFallbackSize(m.windowWidth, m.windowHeight)
	contentWidth := layoutWidth(windowWidth)
	paneSpace := contentWidth - paneGapWidth
	incidentsWidth := paneSpace * incidentsPaneParts / (incidentsPaneParts + componentsPaneParts)
	return paneLayout{contentWidth, incidentsWidth, paneSpace - incidentsWidth, bodyHeight(windowHeight)}
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
	title, body := "Active incidents", m.statusLine()
	if m.hasLoaded {
		title = fmt.Sprintf("Active incidents (%d)", len(m.appDetail.ActiveIncidents))
		body = scrollingBody(m.incidentsViewport)
	}
	look := incidentsPanelLook(m.appDetail.ActiveIncidents, m.focus == focusIncidents)
	return panelWithLook(look, title, body, width, height)
}

func (m AppDetailModel) componentsPanel(width, height int) string {
	title, body := "Watched components", ""
	if m.hasLoaded {
		title = fmt.Sprintf("Watched components (%d)", len(m.appDetail.WatchedComponents))
		body = scrollingBody(m.componentsViewport)
	}
	return panelWithLook(untintedPanelLook(m.focus == focusComponents), title, body, width, height)
}

// scrollingBody is a pane's scrolling content and, under it, how much more there is below.
func scrollingBody(paneViewport viewport.Model) string {
	return paneViewport.View() + "\n" + moreHint(paneViewport)
}

// moreHint is "↓ N more" while there are lines below the visible ones, and blank otherwise, so
// the pane doesn't change height as it scrolls.
func moreHint(paneViewport viewport.Model) string {
	linesBelow := paneViewport.TotalLineCount() - paneViewport.YOffset() - paneViewport.VisibleLineCount()
	if linesBelow <= 0 {
		return ""
	}
	return mutedStyle.Render(fmt.Sprintf("↓ %d more", linesBelow))
}

func otherFocus(focus detailFocus) detailFocus {
	if focus == focusIncidents {
		return focusComponents
	}
	return focusIncidents
}

// scrolled applies scroll to the viewport of the pane that has the focus.
func (m AppDetailModel) scrolled(scroll func(*viewport.Model)) AppDetailModel {
	if m.focus == focusIncidents {
		scroll(&m.incidentsViewport)
	} else {
		scroll(&m.componentsViewport)
	}
	return m
}

// sizedToWindow sizes both viewports to fit their panes and fills them again, because the
// components are laid out to their pane's width.
func (m AppDetailModel) sizedToWindow() AppDetailModel {
	layout := m.paneLayout()
	m.incidentsViewport = sizedViewport(m.incidentsViewport, layout.incidentsWidth, layout.paneHeight)
	m.componentsViewport = sizedViewport(m.componentsViewport, layout.componentsWidth, layout.paneHeight)
	return m.withViewportContents()
}

// sizedViewport fits a viewport inside a pane, leaving the last line for the "more" hint.
func sizedViewport(paneViewport viewport.Model, paneWidth, paneHeight int) viewport.Model {
	contentWidth, contentHeight := panelContentSize(paneWidth, paneHeight)
	paneViewport.SetWidth(contentWidth)
	paneViewport.SetHeight(max(contentHeight-moreHintLines, 1))
	return paneViewport
}

// withViewportContents fills the viewports from the app. A viewport keeps its scroll position
// when its content changes, so a refresh doesn't send either pane back to the top.
func (m AppDetailModel) withViewportContents() AppDetailModel {
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
