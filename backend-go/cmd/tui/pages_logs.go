package tui

import "github.com/rivo/tview"

func (a *tuiApp) buildLogsPage() builtPage {
	allBtn := a.actionButton("All", a.logLevelAction("all"))
	errorBtn := a.actionButton("Error", a.logLevelAction("error"))
	warnBtn := a.actionButton("Warn", a.logLevelAction("warning"))
	infoBtn := a.actionButton("Info", a.logLevelAction("info"))
	debugBtn := a.actionButton("Debug", a.logLevelAction("debug"))
	srcAllBtn := a.actionButton("Src All", a.logSourceAction("all"))
	appBtn := a.actionButton("App", a.logSourceAction("app"))
	xrayBtn := a.actionButton("Xray", a.logSourceAction("xray-core"))
	applyBtn := a.actionButton("Apply Search", a.applyLogSearchAction)
	clearBtn := a.actionButton("Clear Search", a.clearLogSearchAction)

	toolbar := buttonRow(allBtn, errorBtn, warnBtn, infoBtn, debugBtn)
	sourceToolbar := buttonRow(srcAllBtn, appBtn, xrayBtn)
	if a.useStackedLayout() {
		toolbar = buttonColumn(allBtn, errorBtn, warnBtn, infoBtn, debugBtn)
		sourceToolbar = buttonColumn(srcAllBtn, appBtn, xrayBtn)
	}
	searchRow := inputRow(a.logsSearchInput, buttonRow(applyBtn, clearBtn), a.useStackedLayout(), 6, 4)
	toolbarHeight := actionBlockHeight(a.useStackedLayout(), 5)
	sourceToolbarHeight := actionBlockHeight(a.useStackedLayout(), 3)
	searchRowHeight := dualItemRowHeight(a.useStackedLayout())
	filtersContentHeight := 1 + 1 + toolbarHeight + 1 + sourceToolbarHeight + 1 + searchRowHeight + 1 + 1
	filters := tview.NewFlex().SetDirection(tview.FlexRow)
	filters.AddItem(newMutedText("Filter by level/source and narrow with search to isolate faults quickly"), 1, 0, false)
	filters.AddItem(verticalSpacer(1), 1, 0, false)
	filters.AddItem(toolbar, toolbarHeight, 0, false)
	filters.AddItem(verticalSpacer(1), 1, 0, false)
	filters.AddItem(sourceToolbar, sourceToolbarHeight, 0, false)
	filters.AddItem(verticalSpacer(1), 1, 0, false)
	filters.AddItem(searchRow, searchRowHeight, 0, false)
	filters.AddItem(verticalSpacer(1), 1, 0, false)
	filters.AddItem(a.logsStatus, 1, 0, false)
	body := wrapPanel("Logs", a.logsView)
	root := buildPageLayout("Filters", filters, filtersContentHeight, body)
	return builtPage{
		root:       root,
		focusables: joinFocusables(buttonsToFocusables(allBtn, errorBtn, warnBtn, infoBtn, debugBtn, srcAllBtn, appBtn, xrayBtn, applyBtn, clearBtn), primitivesToFocusables(a.logsSearchInput, a.logsView)),
	}
}
