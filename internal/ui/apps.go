// Package ui holds the Bubble Tea models for the TUI's screens.
package ui

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/statuspane/tui/internal/api"
	"github.com/statuspane/tui/internal/config"
)

const (
	refreshInterval = 30 * time.Second
	fetchTimeout    = 15 * time.Second

	selectionMarker   = "▸ "
	noSelectionMarker = "  "
	columnGap         = "  "
)

// AppLister is the one thing the screen needs from the API client; tests pass a fake.
type AppLister interface {
	ListApps(ctx context.Context) ([]api.App, error)
}

// openAppMsg asks the root model to show one app's detail screen.
type openAppMsg struct{ slug, name string }

// openCreateAppDialogMsg asks the root model to show the dialog that creates an app.
type openCreateAppDialogMsg struct{}

// appsChangedMsg tells the list its data is stale: an app was created or deleted.
type appsChangedMsg struct{}

type appsLoadedMsg struct{ apps []api.App }

type appsFailedMsg struct{ err error }

// refreshTickMsg carries the refresh sequence it was scheduled under, so a tick
// left over from before a manual refresh is ignored instead of starting a second loop.
type refreshTickMsg struct{ sequence int }

type AppsModel struct {
	appLister       AppLister
	apps            []api.App
	err             error
	hasLoaded       bool
	isLoading       bool
	selectedIndex   int
	refreshSequence int
	reloadAfterLoad bool
	spinner         spinner.Model
}

func NewAppsModel(appLister AppLister) AppsModel {
	return AppsModel{appLister: appLister, isLoading: true, spinner: spinner.New()}
}

func (m AppsModel) Init() tea.Cmd {
	return m.fetchCmd()
}

func (m AppsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		return m.handleKey(msg)

	case appsLoadedMsg:
		m.apps, m.err, m.hasLoaded, m.isLoading = msg.apps, nil, true, false
		m.selectedIndex = min(m.selectedIndex, max(len(m.apps)-1, 0))
		return m.reloadOrScheduleRefresh()

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
		m.selectedIndex = max(m.selectedIndex-1, 0)
	case "down", "j":
		m.selectedIndex = min(m.selectedIndex+1, max(len(m.apps)-1, 0))
	case "enter":
		if len(m.apps) == 0 {
			return m, nil
		}
		app := m.apps[m.selectedIndex]
		return m, func() tea.Msg { return openAppMsg{slug: app.Slug, name: app.Name} }
	case "n":
		return m, func() tea.Msg { return openCreateAppDialogMsg{} }
	case "d":
		if len(m.apps) == 0 {
			return m, nil
		}
		app := m.apps[m.selectedIndex]
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

func (m AppsModel) startRefresh() (tea.Model, tea.Cmd) {
	m.isLoading = true
	return m, m.fetchCmd()
}

func (m AppsModel) fetchCmd() tea.Cmd {
	return tea.Batch(m.spinner.Tick, fetchApps(m.appLister))
}

func (m AppsModel) reloadOrScheduleRefresh() (tea.Model, tea.Cmd) {
	if m.reloadAfterLoad {
		m.reloadAfterLoad = false
		m.refreshSequence++
		return m.startRefresh()
	}
	return m.scheduleRefresh()
}

func (m AppsModel) scheduleRefresh() (tea.Model, tea.Cmd) {
	m.refreshSequence++
	sequence := m.refreshSequence
	return m, tea.Tick(refreshInterval, func(time.Time) tea.Msg {
		return refreshTickMsg{sequence: sequence}
	})
}

func (m AppsModel) View() tea.View {
	screen := titleStyle.Render("StatusPane") + "\n\n" +
		m.body() +
		m.statusLines() +
		"\n" + mutedStyle.Render("↑/↓ move · enter open · n new · d delete · r refresh · q quit") + "\n"
	return tea.NewView(screen)
}

func (m AppsModel) body() string {
	switch {
	case !m.hasLoaded && m.err != nil:
		return errorStyle.Render(errorMessage(m.err)) + "\n"
	case !m.hasLoaded:
		return m.spinner.View() + " Loading apps…\n"
	case len(m.apps) == 0:
		return "No apps yet. Press n to create one.\n"
	}
	return m.appRows()
}

// statusLines reports a failed or running refresh of apps that are already on screen.
func (m AppsModel) statusLines() string {
	var lines strings.Builder
	if m.hasLoaded && m.err != nil {
		lines.WriteString("\n" + errorStyle.Render("Refresh failed: "+errorMessage(m.err)) + "\n")
	}
	if m.hasLoaded && m.isLoading {
		lines.WriteString("\n" + m.spinner.View() + " Refreshing…\n")
	}
	return lines.String()
}

func (m AppsModel) appRows() string {
	nameWidth, statusWidth := 0, 0
	for _, app := range m.apps {
		nameWidth = max(nameWidth, lipgloss.Width(app.Name))
		statusWidth = max(statusWidth, lipgloss.Width(statusLabel(app.Status)))
	}

	var rows strings.Builder
	for index, app := range m.apps {
		marker, name := noSelectionMarker, fmt.Sprintf("%-*s", nameWidth, app.Name)
		if index == m.selectedIndex {
			marker, name = selectionMarker, selectedStyle.Render(name)
		}
		components := mutedStyle.Render(fmt.Sprintf("%d components", app.WatchedComponentsCount))
		rows.WriteString(marker + name + columnGap + renderStatus(app.Status, statusWidth) + columnGap + components + "\n")
	}
	return rows.String()
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

func errorMessage(err error) string {
	var apiErr *api.APIError
	if errors.As(err, &apiErr) {
		switch apiErr.Code {
		case api.CodeUnauthorized:
			return "The API key was rejected. Check " + config.EnvAPIKey + "."
		case api.CodeRateLimited:
			return "Rate limited. Try again in " + secondsText(apiErr.RetryAfter) + "."
		case api.CodeNotFound:
			return "Not found. It may have been deleted or renamed."
		}
	}
	return err.Error()
}

// secondsText rounds up, so a sub-second wait never reads as "0 seconds".
func secondsText(duration time.Duration) string {
	seconds := int(math.Ceil(duration.Seconds()))
	if seconds == 1 {
		return "1 second"
	}
	return fmt.Sprintf("%d seconds", seconds)
}
