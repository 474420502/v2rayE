package main

import (
	"github.com/gcla/gowid"
	"github.com/gcla/gowid/widgets/button"
	"github.com/gcla/gowid/widgets/columns"
	"github.com/gcla/gowid/widgets/divider"
	"github.com/gcla/gowid/widgets/edit"
	"github.com/gcla/gowid/widgets/holder"
	"github.com/gcla/gowid/widgets/pile"
	"github.com/gcla/gowid/widgets/styled"
	"github.com/gcla/gowid/widgets/text"
)

func (a *tuiApp) build() gowid.IWidget {
	a.footer = text.New("Connecting...")
	a.pageHolder = holder.New(text.New("Loading..."))
	a.dashboardSummary = readOnlyEditor("")
	a.dashboardEvents = readOnlyEditor("")
	a.logsStatus = text.New("Logs: all levels | all sources")
	a.logsView = readOnlyEditor("")
	a.logsSearchInput = edit.New(edit.Options{Caption: "search: "})
	a.profileDetail = readOnlyEditor("Select a profile to inspect.")
	a.profileBatchStatus = text.New("Batch delay test idle.")
	a.profileImport = edit.New(edit.Options{Caption: "URI: "})
	a.profilesListHolder = holder.New(text.New("Loading profiles..."))
	a.subscriptionDetail = readOnlyEditor("Select a subscription to inspect.")
	a.subscriptionsHolder = holder.New(text.New("Loading subscriptions..."))
	a.networkSummary = readOnlyEditor("")
	a.networkTestTarget = edit.New(edit.Options{Caption: "target: "})
	a.networkTestPort = edit.New(edit.Options{Caption: "port: "})
	a.networkTestResult = readOnlyEditor("No routing test executed.")
	a.settingsSummary = readOnlyEditor("")
	a.settingsListenAddr = edit.New(edit.Options{Caption: "listenAddr: "})
	a.settingsSocksPort = edit.New(edit.Options{Caption: "socksPort: "})
	a.settingsHTTPPort = edit.New(edit.Options{Caption: "httpPort: "})
	a.settingsTunName = edit.New(edit.Options{Caption: "tunName: "})
	a.settingsProxyMode = edit.New(edit.Options{Caption: "proxyMode(forced_change|forced_clear|pac): "})
	a.settingsProxyExcept = edit.New(edit.Options{Caption: "proxyExceptions: "})

	root := pile.New([]gowid.IContainerWidget{
		&gowid.ContainerWidget{IWidget: styled.New(text.New(" v2rayE Terminal "), gowid.MakePaletteRef("title")), D: gowid.RenderFlow{}},
		&gowid.ContainerWidget{IWidget: a.buildTabs(), D: gowid.RenderFlow{}},
		&gowid.ContainerWidget{IWidget: divider.NewBlank(), D: gowid.RenderFlow{}},
		&gowid.ContainerWidget{IWidget: a.pageHolder, D: gowid.RenderWithWeight{W: 1}},
		&gowid.ContainerWidget{IWidget: divider.NewBlank(), D: gowid.RenderFlow{}},
		&gowid.ContainerWidget{IWidget: styled.New(a.footer, gowid.MakePaletteRef("footer")), D: gowid.RenderFlow{}},
	})

	a.syncPages(nil)
	return styled.New(root, gowid.MakePaletteRef("body"))
}

func (a *tuiApp) buildTabs() gowid.IWidget {
	tabs := tuiPageTabs()
	children := make([]gowid.IContainerWidget, 0, len(tabs)*2)
	for idx, tab := range tabs {
		tab := tab
		btn := button.New(text.New(tab.label))
		btn.OnClick(gowid.MakeWidgetCallback("tab-"+tab.key, func(app gowid.IApp, _ gowid.IWidget) {
			a.setActivePage(tab.key, app)
		}))
		style := "button"
		if tab.key == a.page {
			style = "button-selected"
		}
		children = append(children, &gowid.ContainerWidget{IWidget: styled.NewExt(btn, gowid.MakePaletteRef(style), gowid.MakePaletteRef("button-focus")), D: gowid.RenderFixed{}})
		if idx != len(tabs)-1 {
			children = append(children, &gowid.ContainerWidget{IWidget: text.New(" "), D: gowid.RenderFixed{}})
		}
	}
	return columns.New(children)
}

func (a *tuiApp) syncPages(app gowid.IApp) {
	if a.pageHolder == nil {
		return
	}

	var page gowid.IWidget
	switch a.page {
	case "profiles":
		page = a.buildProfilesPage()
	case "subscriptions":
		page = a.buildSubscriptionsPage()
	case "network":
		page = a.buildNetworkPage()
	case "settings":
		page = a.buildSettingsPage()
	case "logs":
		page = a.buildLogsPage()
	default:
		page = a.buildDashboardPage()
	}

	if app != nil {
		a.pageHolder.SetSubWidget(page, app)
		return
	}
	if a.app != nil {
		a.runUI(func(app gowid.IApp) {
			a.pageHolder.SetSubWidget(page, app)
		})
	}
}
