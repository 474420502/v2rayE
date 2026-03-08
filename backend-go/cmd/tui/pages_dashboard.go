package tui

import (
	"github.com/gcla/gowid"
	"github.com/gcla/gowid/widgets/columns"
	"github.com/gcla/gowid/widgets/divider"
	"github.com/gcla/gowid/widgets/fill"
	"github.com/gcla/gowid/widgets/framed"
	"github.com/gcla/gowid/widgets/pile"
)

func (a *tuiApp) buildDashboardPage() gowid.IWidget {
	actions := columns.New([]gowid.IContainerWidget{
		buttonCell(a.actionButton("Start", a.startCoreAction)),
		spacerCell(),
		buttonCell(a.actionButton("Stop", a.stopCoreAction)),
		spacerCell(),
		buttonCell(a.actionButton("Restart", a.restartCoreAction)),
		spacerCell(),
		buttonCell(a.actionButton("Refresh", a.refreshAllAction)),
	})

	left := framed.NewUnicode(a.dashboardSummary)
	right := framed.NewUnicode(a.dashboardEvents)
	mainCols := columns.New([]gowid.IContainerWidget{
		&gowid.ContainerWidget{IWidget: left, D: gowid.RenderWithWeight{W: 3}},
		&gowid.ContainerWidget{IWidget: fill.New(' '), D: gowid.RenderWithUnits{U: 1}},
		&gowid.ContainerWidget{IWidget: right, D: gowid.RenderWithWeight{W: 2}},
	})

	return pile.New([]gowid.IContainerWidget{
		&gowid.ContainerWidget{IWidget: actions, D: gowid.RenderFlow{}},
		&gowid.ContainerWidget{IWidget: divider.NewBlank(), D: gowid.RenderFlow{}},
		&gowid.ContainerWidget{IWidget: mainCols, D: gowid.RenderWithWeight{W: 1}},
	})
}
