package ui

import (
	"context"
	"errors"
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

func apiError(status int, code string, details ...api.FieldError) *api.APIError {
	return &api.APIError{Status: status, Code: code, Details: details}
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

func TestCreateDialogSendsOnlyOneRequestForADoubleEnter(t *testing.T) {
	d := newTestCreateDialog(fakeAppCreator{})
	d = typeText(d, "Storefront")
	d, first := pressEnter(d)

	_, second := pressEnter(d)

	if first == nil || second != nil {
		t.Fatalf("first = %v, second = %v", first != nil, second != nil)
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

func TestCreateDialogTreatsQAndJAsText(t *testing.T) {
	d := newTestCreateDialog(fakeAppCreator{})

	d = typeText(d, "queue")

	if view := d.view(); !strings.Contains(view, "queue") || !strings.Contains(view, "5 / 100") {
		t.Fatalf("view = %q", view)
	}
}

func TestCreateDialogEscapeAndCtrlCKeepTheirMeaning(t *testing.T) {
	d := newTestCreateDialog(fakeAppCreator{})

	_, escape := d.update(tea.KeyPressMsg{Code: tea.KeyEscape})
	_, quit := d.update(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})

	if _, ok := escape().(closeDialogMsg); !ok {
		t.Fatal("esc must close the dialog")
	}
	if _, ok := quit().(tea.QuitMsg); !ok {
		t.Fatal("ctrl+c must quit")
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

func TestDeleteDialogSendsOnlyOneRequestForADoubleEnter(t *testing.T) {
	d, first := pressEnter(typeText(newTestDeleteDialog(fakeAppDeleter{}), "Web App"))

	_, second := pressEnter(d)

	if first == nil || second != nil {
		t.Fatalf("first = %v, second = %v", first != nil, second != nil)
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

func TestComponentsText(t *testing.T) {
	if componentsText(1) != "1 component" || componentsText(0) != "0 components" || componentsText(12) != "12 components" {
		t.Fatal("wrong pluralisation")
	}
}

func TestDialogsIgnoreEscapeWhileARequestIsInFlight(t *testing.T) {
	cases := []struct {
		name      string
		open      func() dialog
		validText string
	}{
		{"create", func() dialog { return newTestCreateDialog(fakeAppCreator{}) }, "Storefront"},
		{"delete", func() dialog { return newTestDeleteDialog(fakeAppDeleter{}) }, "Web App"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d, _ := pressEnter(typeText(tc.open(), tc.validText))

			_, cmd := d.update(tea.KeyPressMsg{Code: tea.KeyEscape})

			if cmd != nil {
				t.Fatal("closing now would leave the result with no dialog to report to")
			}
		})
	}
}
