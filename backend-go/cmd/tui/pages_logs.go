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

	// 优化日志级别按钮布局：将5个按钮分成两行，减少单行按钮数量
	levelToolbarRow1 := buttonRow(allBtn, errorBtn, warnBtn)
	levelToolbarRow2 := buttonRow(infoBtn, debugBtn)
	sourceToolbar := buttonRow(srcAllBtn, appBtn, xrayBtn)
	searchActions := buttonRow(applyBtn, clearBtn)
	searchRow := inputRow(a.logsSearchInput, searchActions, a.useStackedLayout(), 6, 4)
	searchRowHeight := dualItemRowHeight(a.useStackedLayout())
	if a.useStackedLayout() {
		levelToolbarRow1 = buttonColumn(allBtn, errorBtn, warnBtn)
		levelToolbarRow2 = buttonColumn(infoBtn, debugBtn)
		sourceToolbar = buttonColumn(srcAllBtn, appBtn, xrayBtn)
		searchActions = buttonColumn(applyBtn, clearBtn)
		stackedSearchRow := tview.NewFlex().SetDirection(tview.FlexRow)
		stackedSearchRow.AddItem(a.logsSearchInput, 1, 0, false)
		stackedSearchRow.AddItem(verticalSpacer(1), 1, 0, false)
		stackedSearchRow.AddItem(searchActions, actionBlockHeight(true, 2), 0, false)
		searchRow = stackedSearchRow
		searchRowHeight = 1 + 1 + actionBlockHeight(true, 2)
	}

	// 优化高度计算
	levelRow1Height := actionBlockHeight(a.useStackedLayout(), 3)
	levelRow2Height := actionBlockHeight(a.useStackedLayout(), 2)
	sourceToolbarHeight := actionBlockHeight(a.useStackedLayout(), 3)
	filtersContentHeight := 1 + 1 + levelRow1Height + 1 + levelRow2Height + 1 + sourceToolbarHeight + 1 + searchRowHeight + 1 + 1

	filters := tview.NewFlex().SetDirection(tview.FlexRow)
	filters.AddItem(newMutedText(a.t("logs.desc")), 1, 0, false)
	filters.AddItem(verticalSpacer(1), 1, 0, false)
	filters.AddItem(levelToolbarRow1, levelRow1Height, 0, false)
	filters.AddItem(verticalSpacer(1), 1, 0, false)
	filters.AddItem(levelToolbarRow2, levelRow2Height, 0, false)
	filters.AddItem(verticalSpacer(1), 1, 0, false)
	filters.AddItem(sourceToolbar, sourceToolbarHeight, 0, false)
	filters.AddItem(verticalSpacer(1), 1, 0, false)
	filters.AddItem(searchRow, searchRowHeight, 0, false)
	filters.AddItem(verticalSpacer(1), 1, 0, false)
	filters.AddItem(a.logsStatus, 1, 0, false)
	body := wrapPanel(a.t("logs.panel.logs"), a.logsView)
	root := buildPageLayout(a.t("logs.panel.filters"), filters, filtersContentHeight, body)
	// 优化焦点组：按功能分区
	return builtPage{
		root:       root,
		focusables: joinFocusables(
			buttonsToFocusables(allBtn, errorBtn, warnBtn),
			buttonsToFocusables(infoBtn, debugBtn),
			buttonsToFocusables(srcAllBtn, appBtn, xrayBtn),
			primitivesToFocusables(a.logsSearchInput),
			buttonsToFocusables(applyBtn, clearBtn),
			primitivesToFocusables(a.logsView),
		),
		focusGroups: [][]tview.Primitive{
			buttonsToFocusables(allBtn, errorBtn, warnBtn),
			buttonsToFocusables(infoBtn, debugBtn),
			buttonsToFocusables(srcAllBtn, appBtn, xrayBtn),
			primitivesToFocusables(a.logsSearchInput),
			buttonsToFocusables(applyBtn, clearBtn),
			primitivesToFocusables(a.logsView),
		},
	}
}
