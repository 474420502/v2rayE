package tui

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

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
	a.viewportCols = 120
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
	a.viewportCols = 120
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

func TestSetActivePage_EnteringNetworkClearsStagedRoutingState(t *testing.T) {
	a := newTUI(context.Background(), nil)
	_ = a.build()
	a.page = pageDashboard
	a.networkRoutingDirty = true

	a.setActivePage(pageNetwork)

	if a.networkRoutingDirty {
		t.Fatal("expected entering network page to clear staged routing state")
	}
}

func TestSetActivePage_StayingOnNetworkKeepsStagedRoutingState(t *testing.T) {
	a := newTUI(context.Background(), nil)
	_ = a.build()
	a.page = pageNetwork
	a.networkRoutingDirty = true

	a.setActivePage(pageNetwork)

	if !a.networkRoutingDirty {
		t.Fatal("expected reselecting current network page to preserve staged routing state")
	}
}

func TestRoutingPresetKey_RequiresExactPresetShape(t *testing.T) {
	trueValue := true
	tests := []struct {
		name    string
		routing RoutingConfig
		want    string
	}{
		{
			name: "global preset exact match",
			routing: RoutingConfig{
				Mode:               "global",
				DomainStrategy:     "IPIfNonMatch",
				LocalBypassEnabled: &trueValue,
			},
			want: "global",
		},
		{
			name: "custom rules prevent preset match",
			routing: RoutingConfig{
				Mode:               "global",
				DomainStrategy:     "IPIfNonMatch",
				LocalBypassEnabled: &trueValue,
				Rules: []RoutingRule{{
					ID:       "rule-1",
					Type:     "domain",
					Values:   []string{"example.com"},
					Outbound: "direct",
				}},
			},
			want: "",
		},
		{
			name: "direct preset exact match",
			routing: RoutingConfig{
				Mode:               "direct",
				DomainStrategy:     "AsIs",
				LocalBypassEnabled: &trueValue,
			},
			want: "direct",
		},
		{
			name: "wrong strategy does not match preset",
			routing: RoutingConfig{
				Mode:               "direct",
				DomainStrategy:     "IPIfNonMatch",
				LocalBypassEnabled: &trueValue,
			},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := routingPresetKey(tt.routing); got != tt.want {
				t.Fatalf("expected preset %q, got %q", tt.want, got)
			}
		})
	}
}

func TestTargetRoutingPreset_UsesPendingPresetEvenWithExistingCustomRules(t *testing.T) {
	falseValue := false
	a := newTUI(context.Background(), nil)
	_ = a.build()
	a.routing = RoutingConfig{
		Mode:               "custom",
		DomainStrategy:     "IPOnDemand",
		LocalBypassEnabled: &falseValue,
		Rules: []RoutingRule{{
			ID:       "rule-1",
			Type:     "domain",
			Values:   []string{"example.com"},
			Outbound: "direct",
		}},
	}

	a.applyRoutingPresetToForm("global")

	if got := a.targetRoutingPreset(); got != "global" {
		t.Fatalf("expected staged preset to be global, got %q", got)
	}
}

func TestSaveRoutingModeAction_PresetClearsExistingCustomRules(t *testing.T) {
	falseValue := false
	var saved RoutingConfig
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPut && r.URL.Path == "/api/routing":
			if err := json.NewDecoder(r.Body).Decode(&saved); err != nil {
				t.Fatalf("decode request body: %v", err)
			}
			writeTestEnvelope(t, w, saved)
		case r.Method == http.MethodGet && r.URL.Path == "/api/routing":
			writeTestEnvelope(t, w, saved)
		case r.Method == http.MethodGet && r.URL.Path == "/api/routing/diagnostics":
			writeTestEnvelope(t, w, RoutingDiagnostics{})
		case r.Method == http.MethodGet && r.URL.Path == "/api/routing/hits":
			writeTestEnvelope(t, w, RoutingHitStats{})
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	a := newTUI(context.Background(), newAPIClient(server.URL, ""))
	_ = a.build()
	a.routing = RoutingConfig{
		Mode:               "custom",
		DomainStrategy:     "IPOnDemand",
		LocalBypassEnabled: &falseValue,
		Rules: []RoutingRule{{
			ID:       "rule-1",
			Type:     "domain",
			Values:   []string{"example.com"},
			Outbound: "direct",
		}},
	}

	a.applyRoutingPresetToForm("global")

	if err := a.saveRoutingModeAction(context.Background()); err != nil {
		t.Fatalf("save routing mode: %v", err)
	}
	if len(saved.Rules) != 0 {
		t.Fatalf("expected preset save to clear custom rules, got %d rule(s)", len(saved.Rules))
	}
	if saved.Mode != "global" {
		t.Fatalf("expected saved mode global, got %q", saved.Mode)
	}
	if saved.DomainStrategy != "IPIfNonMatch" {
		t.Fatalf("expected saved domain strategy IPIfNonMatch, got %q", saved.DomainStrategy)
	}
	if saved.LocalBypassEnabled == nil || !*saved.LocalBypassEnabled {
		t.Fatal("expected saved local bypass to be true")
	}
}

func TestMarkNetworkRoutingDirtyFromManualEdit_ClearsPendingPreset(t *testing.T) {
	a := newTUI(context.Background(), nil)
	_ = a.build()
	a.applyRoutingPresetToForm("global")

	a.markNetworkRoutingDirtyFromManualEdit()

	if got := a.pendingNetworkPreset(); got != "" {
		t.Fatalf("expected pending preset to be cleared after manual edit, got %q", got)
	}
}

func TestBuildNetworkPage_FocusGroupsFollowSeparatedVisualFlow(t *testing.T) {
	a := newTUI(context.Background(), nil)
	_ = a.build()
	built := a.buildNetworkPage()

	if len(built.focusGroups) != 7 {
		t.Fatalf("expected 7 focus groups, got %d", len(built.focusGroups))
	}
	if len(built.focusGroups[0]) != 1 {
		t.Fatalf("expected check action group size 1, got %d", len(built.focusGroups[0]))
	}
	if built.focusGroups[1][0] != a.networkPresetSelect {
		t.Fatal("expected preset dropdown to be second focus group")
	}
	if len(built.focusGroups[2]) != 3 {
		t.Fatalf("expected routing form group size 3, got %d", len(built.focusGroups[2]))
	}
	if built.focusGroups[2][0] != a.networkRoutingMode || built.focusGroups[2][1] != a.networkDomainStrategy || built.focusGroups[2][2] != a.networkLocalBypass {
		t.Fatal("expected routing form fields to stay together before system proxy actions")
	}
	if len(built.focusGroups[3]) != 2 {
		t.Fatalf("expected system proxy group size 2, got %d", len(built.focusGroups[3]))
	}
	applyBtn, ok := built.focusGroups[3][0].(*tview.Button)
	if !ok || applyBtn.GetLabel() != a.t("network.btn.applyProxy") {
		t.Fatal("expected system proxy apply button after routing form group")
	}
	clearBtn, ok := built.focusGroups[3][1].(*tview.Button)
	if !ok || clearBtn.GetLabel() != a.t("network.btn.clearProxy") {
		t.Fatal("expected system proxy clear button in same group")
	}
	if built.focusGroups[6][0] != a.networkTestTarget || built.focusGroups[6][1] != a.networkTestPort {
		t.Fatal("expected route test inputs to stay in final focus group")
	}
}

func TestMarkBackgroundWorkStarted_OnlyOnce(t *testing.T) {
	a := newTUI(context.Background(), nil)

	if !a.markBackgroundWorkStarted() {
		t.Fatal("expected first background-work mark to succeed")
	}
	if a.markBackgroundWorkStarted() {
		t.Fatal("expected second background-work mark to be ignored")
	}
}

func TestFormatNetworkSummary_DoesNotDeadlockWhileHoldingMutex(t *testing.T) {
	trueValue := true
	a := newTUI(context.Background(), nil)
	_ = a.build()
	a.routing = RoutingConfig{
		Mode:               "global",
		DomainStrategy:     "IPIfNonMatch",
		LocalBypassEnabled: &trueValue,
	}
	a.networkPresetApplied = "global"

	done := make(chan struct{})
	go func() {
		a.mu.Lock()
		defer a.mu.Unlock()
		_ = a.formatNetworkSummary()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("formatNetworkSummary deadlocked while app mutex was held")
	}
}

func writeTestEnvelope(t *testing.T, w http.ResponseWriter, data any) {
	t.Helper()
	payload, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("marshal envelope payload: %v", err)
	}
	if err := json.NewEncoder(w).Encode(apiEnvelope{Code: 0, Message: "ok", Data: payload}); err != nil {
		t.Fatalf("encode envelope: %v", err)
	}
}
