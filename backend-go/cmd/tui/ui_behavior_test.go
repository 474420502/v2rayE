package tui

import (
	"context"
	"testing"

	tcell "github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"v2raye/backend-go/cmd/tui/components"
)

func TestFieldLabelWithChoicesAdaptive_NarrowUsesShortLabel(t *testing.T) {
	a := newTUI(context.Background(), nil)
	a.viewportCols = 120
	got := a.fieldLabelWithChoicesAdaptive("field.settings.logLevel", "field.choices.settings.logLevel")
	want := a.fieldLabel("field.settings.logLevel")
	if got != want {
		t.Fatalf("expected short label %q, got %q", want, got)
	}
}

func TestFieldLabelWithChoicesAdaptive_WideUsesChoicesLabel(t *testing.T) {
	a := newTUI(context.Background(), nil)
	a.viewportCols = 180
	got := a.fieldLabelWithChoicesAdaptive("field.settings.logLevel", "field.choices.settings.logLevel")
	want := a.fieldLabelWithChoices("field.settings.logLevel", "field.choices.settings.logLevel")
	if got != want {
		t.Fatalf("expected choices label %q, got %q", want, got)
	}
}

func TestFocusedDropDownFromPrimitive_SupportsWrappedWidget(t *testing.T) {
	widget := newDropdownWidget("", []selectOption{{Label: "One", Value: "1"}}, nil)
	dropdown, ok := focusedDropDownFromPrimitive(widget)
	if !ok {
		t.Fatal("expected wrapped dropdown to be recognized")
	}
	if dropdown != widget.DropDown {
		t.Fatal("expected extracted dropdown pointer to match wrapped DropDown")
	}
}

func TestFocusedDropDownFromPrimitive_SupportsNativeDropdown(t *testing.T) {
	native := tview.NewDropDown()
	dropdown, ok := focusedDropDownFromPrimitive(native)
	if !ok {
		t.Fatal("expected native dropdown to be recognized")
	}
	if dropdown != native {
		t.Fatal("expected extracted dropdown pointer to match native DropDown")
	}
}

func TestFocusSidebarSelected_MovesFocusToSelectedSidebarButton(t *testing.T) {
	a := newTUI(context.Background(), nil)
	a.viewportCols = 120 // 设置合理的视口宽度，避免触发极窄模式
	a.viewportRows = 30
	app := tview.NewApplication()
	a.attachApp(app)

	sidebar := components.NewSidebar([]components.NavItem{
		{Key: pageDashboard, Label: "Dashboard", Shortcut: '1'},
		{Key: pageSettings, Label: "Settings", Shortcut: '5'},
	}, nil, nil)
	sidebar.Select(1)
	a.sidebar = sidebar
	a.sidebar.SetVisible(true) // 确保侧边栏可见

	content := tview.NewButton("content")
	app.SetFocus(content)

	if !a.focusSidebarSelected() {
		t.Fatal("expected focusSidebarSelected to succeed")
	}
	if app.GetFocus() != sidebar.GetAllButtons()[1] {
		t.Fatal("expected focus to move to selected sidebar button")
	}
}

func TestHandlerEsc_BackToSidebarFromContent(t *testing.T) {
	a := newTUI(context.Background(), nil)
	a.viewportCols = 120 // 设置合理的视口宽度，避免触发极窄模式
	a.viewportRows = 30
	app := tview.NewApplication()
	a.attachApp(app)

	sidebar := components.NewSidebar([]components.NavItem{
		{Key: pageDashboard, Label: "Dashboard", Shortcut: '1'},
		{Key: pageNetwork, Label: "Network", Shortcut: '4'},
	}, nil, nil)
	sidebar.Select(0)
	a.sidebar = sidebar
	a.sidebar.SetVisible(true) // 确保侧边栏可见

	content := tview.NewInputField()
	app.SetFocus(content)

	result := a.handler(tcell.NewEventKey(tcell.KeyEsc, 0, 0))
	if result != nil {
		t.Fatal("expected Esc to be consumed by handler")
	}
	if app.GetFocus() != sidebar.GetAllButtons()[0] {
		t.Fatal("expected Esc to move focus back to sidebar")
	}
}

func TestHandlerEsc_DropdownOverlayHasPriorityOverSidebarBack(t *testing.T) {
	a := newTUI(context.Background(), nil)
	app := tview.NewApplication()
	a.attachApp(app)

	sidebar := components.NewSidebar([]components.NavItem{
		{Key: pageDashboard, Label: "Dashboard", Shortcut: '1'},
		{Key: pageNetwork, Label: "Network", Shortcut: '4'},
	}, nil, nil)
	a.sidebar = sidebar

	content := tview.NewInputField()
	app.SetFocus(content)
	a.dropdownSelectVisible.Store(true)

	result := a.handler(tcell.NewEventKey(tcell.KeyEsc, 0, 0))
	if result != nil {
		t.Fatal("expected Esc to be consumed when dropdown overlay is visible")
	}
	if app.GetFocus() != content {
		t.Fatal("expected Esc to handle overlay first without jumping focus to sidebar")
	}
}

func TestHandlerEsc_CommandPaletteHasPriorityOverSidebarBack(t *testing.T) {
	a := newTUI(context.Background(), nil)
	app := tview.NewApplication()
	a.attachApp(app)

	sidebar := components.NewSidebar([]components.NavItem{
		{Key: pageDashboard, Label: "Dashboard", Shortcut: '1'},
		{Key: pageSettings, Label: "Settings", Shortcut: '5'},
	}, nil, nil)
	a.sidebar = sidebar

	content := tview.NewInputField()
	app.SetFocus(content)
	a.commandPaletteVisible.Store(true)

	result := a.handler(tcell.NewEventKey(tcell.KeyEsc, 0, 0))
	if result != nil {
		t.Fatal("expected Esc to be consumed when command palette is visible")
	}
	if app.GetFocus() != content {
		t.Fatal("expected Esc to close command palette layer first without immediate sidebar jump")
	}
}

func TestDropdownDialogHeight_MinimumRows(t *testing.T) {
	if got := dropdownDialogHeight(1, 0); got != 10 {
		t.Fatalf("expected minimum height 10 for 1 option, got %d", got)
	}
	if got := dropdownDialogHeight(4, 0); got != 10 {
		t.Fatalf("expected height 10 for 4 options, got %d", got)
	}
}

func TestDropdownDialogHeight_GrowsWithOptions(t *testing.T) {
	if got := dropdownDialogHeight(7, 0); got != 11 {
		t.Fatalf("expected height 11 for 7 options, got %d", got)
	}
}

func TestDropdownDialogHeight_UsesViewportRatioFloor(t *testing.T) {
	if got := dropdownDialogHeight(2, 40); got != 16 {
		t.Fatalf("expected height 16 for viewport ratio floor, got %d", got)
	}
}

func TestDropdownDialogWidth_UsesLongestOption(t *testing.T) {
	width := dropdownDialogWidth("Select", "Options: 3", []string{"short", "this-is-a-very-long-option-label-that-should-expand-dialog-width"}, 200)
	if width <= 40 {
		t.Fatalf("expected width to expand for long option, got %d", width)
	}
}

func TestDropdownDialogWidth_RespectsViewportLimit(t *testing.T) {
	width := dropdownDialogWidth("Select", "Options: 3", []string{"this-is-a-very-long-option-label"}, 50)
	if width > 44 {
		t.Fatalf("expected width to stay within viewport limit, got %d", width)
	}
}

func TestDropdownDialogWidth_UsesViewportRatioFloor(t *testing.T) {
	width := dropdownDialogWidth("Select", "Options: 2", []string{"a", "b"}, 120)
	if width < 66 {
		t.Fatalf("expected width to be at least 55%% of viewport, got %d", width)
	}
}

func TestDropdownDialogWidth_HasReasonableMinimum(t *testing.T) {
	width := dropdownDialogWidth("S", "H", []string{"x"}, 0)
	if width < 40 {
		t.Fatalf("expected minimum width >= 40, got %d", width)
	}
}
