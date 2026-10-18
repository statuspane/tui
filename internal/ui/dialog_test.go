package ui

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/statuspane/tui/internal/api"
)

type fakeAppCreator struct {
	appDetail api.AppDetail
	err       error
	names     *[]string
}

func (f fakeAppCreator) CreateApp(_ context.Context, name string, _ []api.ComponentRef) (api.AppDetail, error) {
	if f.names != nil {
		*f.names = append(*f.names, name)
	}
	return f.appDetail, f.err
}

type fakeAppRenamer struct {
	appDetail api.AppDetail
	err       error
	requests  *[]string
}

func (f fakeAppRenamer) RenameApp(_ context.Context, appSlug, name string) (api.AppDetail, error) {
	if f.requests != nil {
		*f.requests = append(*f.requests, appSlug+"="+name)
	}
	return f.appDetail, f.err
}

type fakeAppDeleter struct {
	err   error
	slugs *[]string
}

func (f fakeAppDeleter) DeleteApp(_ context.Context, appSlug string) error {
	if f.slugs != nil {
		*f.slugs = append(*f.slugs, appSlug)
	}
	return f.err
}

func newTestCreateDialog(appCreator AppCreator) dialog {
	d, _ := newCreateAppDialog(appCreator)
	return d
}

func typeText(d dialog, text string) dialog {
	for _, r := range text {
		d, _ = d.update(tea.KeyPressMsg{Code: r, Text: string(r)})
	}
	return d
}

func pressEnter(d dialog) (dialog, tea.Cmd) {
	return d.update(tea.KeyPressMsg{Code: tea.KeyEnter})
}

func TestCreateDialogDoesNothingOnEnterWhileTheNameIsBlank(t *testing.T) {
	d := newTestCreateDialog(fakeAppCreator{})

	_, cmd := pressEnter(typeText(d, "   "))

	if cmd != nil {
		t.Fatal("blank name must not send a request")
	}
}

func TestCreateDialogSendsTheTrimmedNameAndReportsTheNewApp(t *testing.T) {
	var names []string
	created := api.AppDetail{App: api.App{Name: "Storefront", Slug: "storefront"}}
	d := newTestCreateDialog(fakeAppCreator{appDetail: created, names: &names})

	_, cmd := pressEnter(typeText(d, " Storefront "))
	msg := cmd()

	appCreated, ok := msg.(appCreatedMsg)
	if !ok || appCreated.appDetail.Slug != "storefront" || len(names) != 1 || names[0] != "Storefront" {
		t.Fatalf("msg = %#v, names = %v", msg, names)
	}
}

func TestCreateDialogShowsTheAPIsMessageAboutTheNameAndKeepsTheText(t *testing.T) {
	failure := apiError(400, api.CodeValidationFailed, api.FieldError{Field: "name", Code: "too_long", Message: "Name is too long"})
	d := newTestCreateDialog(fakeAppCreator{err: failure})
	d, cmd := pressEnter(typeText(d, "Storefront"))
	d, _ = d.update(cmd())

	view := d.view()
	if !strings.Contains(view, "Name is too long") || !strings.Contains(view, "Storefront") {
		t.Fatalf("view = %q", view)
	}
	if _, retry := pressEnter(d); retry == nil {
		t.Fatal("the dialog must accept another try after an error")
	}
}

func TestCreateDialogExplainsOtherFailures(t *testing.T) {
	d := newTestCreateDialog(fakeAppCreator{err: errors.New("connection refused")})
	d, cmd := pressEnter(typeText(d, "Storefront"))
	d, _ = d.update(cmd())

	if view := d.view(); !strings.Contains(view, "connection refused") {
		t.Fatalf("view = %q", view)
	}
}

func newTestDeleteDialog(deleter AppDeleter) dialog {
	d, _ := newDeleteAppDialog(deleter, openDeleteAppDialogMsg{slug: "web-app", name: "Web App", componentCount: 12})
	return d
}

func TestDeleteDialogSaysWhatWillHappen(t *testing.T) {
	view := newTestDeleteDialog(fakeAppDeleter{}).view()

	for _, want := range []string{"Delete Web App?", "12 components", "can't be undone"} {
		if !strings.Contains(view, want) {
			t.Errorf("view missing %q:\n%s", want, view)
		}
	}
}

func TestDeleteDialogEnterDoesNothingUntilTheNameMatches(t *testing.T) {
	var slugs []string
	deleter := fakeAppDeleter{slugs: &slugs}

	_, partial := pressEnter(typeText(newTestDeleteDialog(deleter), "Web Ap"))
	_, wrongCase := pressEnter(typeText(newTestDeleteDialog(deleter), "web app"))
	_, exact := pressEnter(typeText(newTestDeleteDialog(deleter), "Web App"))

	if partial != nil || wrongCase != nil || exact == nil {
		t.Fatalf("partial = %v, wrongCase = %v, exact = %v", partial != nil, wrongCase != nil, exact != nil)
	}
	if _, ok := exact().(appDeletedMsg); !ok || len(slugs) != 1 || slugs[0] != "web-app" {
		t.Fatalf("slugs = %v", slugs)
	}
}

func TestDeleteDialogTreatsAnAppThatIsAlreadyGoneAsDeleted(t *testing.T) {
	d := typeText(newTestDeleteDialog(fakeAppDeleter{err: apiError(404, api.CodeNotFound)}), "Web App")
	_, cmd := pressEnter(d)

	if deleted, ok := cmd().(appDeletedMsg); !ok || deleted.slug != "web-app" {
		t.Fatalf("got %#v", cmd())
	}
}

func TestDeleteDialogKeepsOpenAndExplainsOtherFailures(t *testing.T) {
	d := typeText(newTestDeleteDialog(fakeAppDeleter{err: apiError(401, api.CodeUnauthorized)}), "Web App")
	d, cmd := pressEnter(d)
	d, _ = d.update(cmd())

	if view := d.view(); !strings.Contains(view, "STATUSPANE_API_KEY") {
		t.Fatalf("view = %q", view)
	}
}

func TestCreateDialogShowsHowLongTheNameIs(t *testing.T) {
	d := typeText(newTestCreateDialog(fakeAppCreator{}), "Storefront")

	if view := d.view(); !strings.Contains(view, "10 / 100") {
		t.Fatalf("view = %q", view)
	}
}

func newTestRenameDialog(appRenamer AppRenamer) dialog {
	d, _ := newRenameAppDialog(appRenamer, openRenameAppDialogMsg{slug: "web-app", name: "Web App"})
	return d
}

func TestRenameDialogStartsWithTheCurrentName(t *testing.T) {
	view := newTestRenameDialog(fakeAppRenamer{}).view()

	if !strings.Contains(view, "Rename app") || !strings.Contains(view, "Web App") || !strings.Contains(view, "7 / 100") {
		t.Fatalf("view = %q", view)
	}
}

func backspaces(d dialog, count int) dialog {
	for range count {
		d, _ = d.update(tea.KeyPressMsg{Code: tea.KeyBackspace})
	}
	return d
}

func TestRenameDialogSendsTheTrimmedNewNameForTheAppsSlug(t *testing.T) {
	var renames []string
	d := newTestRenameDialog(fakeAppRenamer{appDetail: api.AppDetail{App: api.App{Slug: "web-shop"}}, requests: &renames})

	_, cmd := pressEnter(typeText(backspaces(d, len("App")), "Shop "))
	msg := cmd()

	if !reflect.DeepEqual(renames, []string{"web-app=Web Shop"}) {
		t.Fatalf("renames = %v", renames)
	}
	renamedMsg, ok := msg.(appRenamedMsg)
	if !ok || renamedMsg.previousSlug != "web-app" {
		t.Fatalf("msg = %#v", msg)
	}
}

func TestRenameDialogClosesWithoutARequestWhenTheNameIsUnchanged(t *testing.T) {
	for name, typed := range map[string]string{"exactly": "", "with spaces around it": "  "} {
		t.Run(name, func(t *testing.T) {
			var renames []string
			d := typeText(newTestRenameDialog(fakeAppRenamer{requests: &renames}), typed)

			_, cmd := pressEnter(d)

			if _, ok := cmd().(closeDialogMsg); !ok || len(renames) != 0 {
				t.Fatalf("want the dialog to close without a request, renames = %v", renames)
			}
		})
	}
}

func TestRenameDialogStaysOpenWithoutARequestWhenTheNameIsBlank(t *testing.T) {
	var renames []string
	d := backspaces(newTestRenameDialog(fakeAppRenamer{requests: &renames}), len("Web App"))

	_, cmd := pressEnter(d)

	if cmd != nil || len(renames) != 0 {
		t.Fatalf("blank name must neither close the dialog nor send a request, renames = %v", renames)
	}
}

func TestRenameDialogShowsTheAPIsMessageAboutTheNameAndKeepsTheText(t *testing.T) {
	failure := apiError(400, api.CodeValidationFailed, api.FieldError{Field: "name", Code: "taken", Message: "Name is already taken"})
	d := newTestRenameDialog(fakeAppRenamer{err: failure})
	d, cmd := pressEnter(typeText(d, " 2"))
	d, _ = d.update(cmd())

	view := d.view()
	if !strings.Contains(view, "Name is already taken") || !strings.Contains(view, "Web App 2") {
		t.Fatalf("view = %q", view)
	}
	if _, retry := pressEnter(d); retry == nil {
		t.Fatal("the dialog must accept another try after an error")
	}
}

// The dialogs share these rules, so one table covers them. validText is typed after whatever the
// dialog starts with: the rename dialog starts with "Web App", so " 2" makes a new name.
var dialogCases = []struct {
	name      string
	open      func() dialog
	validText string
}{
	{"create", func() dialog { return newTestCreateDialog(fakeAppCreator{}) }, "Storefront"},
	{"rename", func() dialog { return newTestRenameDialog(fakeAppRenamer{}) }, " 2"},
	{"delete", func() dialog { return newTestDeleteDialog(fakeAppDeleter{}) }, "Web App"},
}

func TestDialogsSendOnlyOneRequestForADoubleEnter(t *testing.T) {
	for _, tc := range dialogCases {
		t.Run(tc.name, func(t *testing.T) {
			d, first := pressEnter(typeText(tc.open(), tc.validText))

			_, second := pressEnter(d)

			if first == nil || second != nil {
				t.Fatalf("first = %v, second = %v", first != nil, second != nil)
			}
		})
	}
}

func TestDialogsCloseOnEscape(t *testing.T) {
	for _, tc := range dialogCases {
		t.Run(tc.name, func(t *testing.T) {
			_, cmd := tc.open().update(tea.KeyPressMsg{Code: tea.KeyEscape})

			if _, ok := cmd().(closeDialogMsg); !ok {
				t.Fatal("esc must close the dialog")
			}
		})
	}
}

func TestDialogsIgnoreEscapeWhileARequestIsInFlight(t *testing.T) {
	for _, tc := range dialogCases {
		t.Run(tc.name, func(t *testing.T) {
			d, _ := pressEnter(typeText(tc.open(), tc.validText))

			_, cmd := d.update(tea.KeyPressMsg{Code: tea.KeyEscape})

			if cmd != nil {
				t.Fatal("closing now would leave the result with no dialog to report to")
			}
		})
	}
}

func TestDialogsQuitOnCtrlC(t *testing.T) {
	for _, tc := range dialogCases {
		t.Run(tc.name, func(t *testing.T) {
			_, cmd := tc.open().update(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})

			if _, ok := cmd().(tea.QuitMsg); !ok {
				t.Fatal("ctrl+c must quit")
			}
		})
	}
}

func TestDialogsTypeScreenShortcutLettersAsText(t *testing.T) {
	for _, tc := range dialogCases {
		t.Run(tc.name, func(t *testing.T) {
			d := typeText(tc.open(), "qjdn")

			if view := d.view(); !strings.Contains(view, "qjdn") {
				t.Fatalf("view = %q", view)
			}
		})
	}
}

func TestComponentsText(t *testing.T) {
	cases := []struct {
		count int
		want  string
	}{
		{0, "0 components"},
		{1, "1 component"},
		{12, "12 components"},
	}
	for _, tc := range cases {
		t.Run(tc.want, func(t *testing.T) {
			if got := componentsText(tc.count); got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}
