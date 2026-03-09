package tui

import "github.com/rivo/tview"

func (a *tuiApp) buildLogsPage() builtPage {
	allBtn := a.actionButton(a.t("logs.btn.all"), a.logLevelAction("all"))
	errorBtn := a.actionButton(a.t("logs.btn.error"), a.logLevelAction("error"))
	warnBtn := a.actionButton(a.t("logs.btn.warn"), a.logLevelAction("warning"))
	infoBtn := a.actionButton(a.t("logs.btn.info"), a.logLevelAction("info"))
	debugBtn := a.actionButton(a.t("logs.btn.debug"), a.logLevelAction("debug"))
	srcAllBtn := a.actionButton(a.t("logs.btn.srcAll"), a.logSourceAction("all"))
	appBtn := a.actionButton(a.t("logs.btn.app"), a.logSourceAction("app"))
	xrayBtn := a.actionButton(a.t("logs.btn.xray"), a.logSourceAction("xray-core"))
	applyBtn := a.actionButton(a.t("logs.btn.applySearch"), a.applyLogSearchAction)
	clearBtn := a.actionButton(a.t("logs.btn.clearSearch"), a.clearLogSearchAction)

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
	filters.AddItem(newMutedText(a.t("logs.desc")), 1, 0, false)
	filters.AddItem(verticalSpacer(1), 1, 0, false)
	filters.AddItem(toolbar, toolbarHeight, 0, false)
	filters.AddItem(verticalSpacer(1), 1, 0, false)
	filters.AddItem(sourceToolbar, sourceToolbarHeight, 0, false)
	filters.AddItem(verticalSpacer(1), 1, 0, false)
	filters.AddItem(searchRow, searchRowHeight, 0, false)
	filters.AddItem(verticalSpacer(1), 1, 0, false)
	filters.AddItem(a.logsStatus, 1, 0, false)
	body := wrapPanel(a.t("logs.panel.logs"), a.logsView)
	root := buildPageLayout(a.t("logs.panel.filters"), filters, filtersContentHeight, body)
	return builtPage{
		root:       root,
		focusables: joinFocusables(buttonsToFocusables(allBtn, errorBtn, warnBtn, infoBtn, debugBtn, srcAllBtn, appBtn, xrayBtn, applyBtn, clearBtn), primitivesToFocusables(a.logsSearchInput, a.logsView)),
		focusGroups: [][]tview.Primitive{
			buttonsToFocusables(allBtn, errorBtn, warnBtn, infoBtn, debugBtn),
			buttonsToFocusables(srcAllBtn, appBtn, xrayBtn),
			primitivesToFocusables(a.logsSearchInput),
			buttonsToFocusables(applyBtn, clearBtn),
			primitivesToFocusables(a.logsView),
		},
	}
}
