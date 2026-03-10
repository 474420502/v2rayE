package tui

import "github.com/rivo/tview"

func (a *tuiApp) buildLogsPage() builtPage {
	buildGroupPanel := func(title string, rows ...struct {
		primitive tview.Primitive
		height    int
	}) tview.Primitive {
		content := tview.NewFlex().SetDirection(tview.FlexRow)
		for idx, row := range rows {
			content.AddItem(row.primitive, row.height, 0, false)
			if idx != len(rows)-1 {
				content.AddItem(verticalSpacer(1), 1, 0, false)
			}
		}
		return wrapPanel(title, content)
	}

	applyBtn := a.actionButton(a.t("logs.btn.applySearch"), a.applyLogSearchAction)
	clearBtn := a.actionButton(a.t("logs.btn.clearSearch"), a.clearLogSearchAction)

	searchActions := buttonRow(applyBtn, clearBtn)
	searchRow := inputRow(a.logsSearchInput, searchActions, false, 6, 4)
	searchRowHeight := dualItemRowHeight(false)

	filtersContentHeight := 1 + 1 + 1

	levelPanel := buildGroupPanel(
		a.t("logs.group.level"),
		struct {
			primitive tview.Primitive
			height    int
		}{primitive: a.logsLevelSelect, height: 1},
	)

	sourcePanel := buildGroupPanel(
		a.t("logs.group.source"),
		struct {
			primitive tview.Primitive
			height    int
		}{primitive: a.logsSourceSelect, height: 1},
	)

	searchPanel := buildGroupPanel(
		a.t("logs.group.search"),
		struct {
			primitive tview.Primitive
			height    int
		}{primitive: searchRow, height: searchRowHeight},
		struct {
			primitive tview.Primitive
			height    int
		}{primitive: a.logsStatus, height: 1},
	)

	filters := tview.NewFlex().SetDirection(tview.FlexRow)
	filters.AddItem(newMutedText(a.t("logs.desc")), 1, 0, false)
	filters.AddItem(verticalSpacer(1), 1, 0, false)
	grid := tview.NewGrid().SetBorders(false).SetGap(1, 1)
	grid.SetRows(0, 0).SetColumns(0, 0)
	grid.AddItem(levelPanel, 0, 0, 1, 1, 0, 0, false)
	grid.AddItem(sourcePanel, 0, 1, 1, 1, 0, 0, false)
	grid.AddItem(searchPanel, 1, 0, 1, 2, 0, 0, false)
	filters.AddItem(grid, 0, 1, false)
	body := wrapPanel(a.t("logs.panel.logs"), a.logsView)
	root := buildPageLayout(a.t("logs.panel.filters"), filters, filtersContentHeight, body)

	return builtPage{
		root: root,
		focusables: joinFocusables(
			primitivesToFocusables(a.logsLevelSelect, a.logsSourceSelect),
			primitivesToFocusables(a.logsSearchInput),
			buttonsToFocusables(applyBtn, clearBtn),
			primitivesToFocusables(a.logsView),
		),
		focusGroups: [][]tview.Primitive{
			primitivesToFocusables(a.logsLevelSelect, a.logsSourceSelect),
			primitivesToFocusables(a.logsSearchInput),
			buttonsToFocusables(applyBtn, clearBtn),
			primitivesToFocusables(a.logsView),
		},
	}
}
