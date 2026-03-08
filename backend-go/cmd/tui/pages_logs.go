package tui

import (
	"github.com/gcla/gowid"
	"github.com/gcla/gowid/widgets/columns"
	"github.com/gcla/gowid/widgets/divider"
	"github.com/gcla/gowid/widgets/fill"
	"github.com/gcla/gowid/widgets/framed"
	"github.com/gcla/gowid/widgets/pile"
	"github.com/gcla/gowid/widgets/styled"
	"github.com/gcla/gowid/widgets/text"
)

func (a *tuiApp) buildLogsPage() gowid.IWidget {
	toolbar := columns.New([]gowid.IContainerWidget{
		buttonCell(a.actionButton("All", a.logLevelAction("all"))),
		spacerCell(),
		buttonCell(a.actionButton("Error", a.logLevelAction("error"))),
		spacerCell(),
		buttonCell(a.actionButton("Warn", a.logLevelAction("warning"))),
		spacerCell(),
		buttonCell(a.actionButton("Info", a.logLevelAction("info"))),
		spacerCell(),
		buttonCell(a.actionButton("Debug", a.logLevelAction("debug"))),
	})

	sourceToolbar := columns.New([]gowid.IContainerWidget{
		buttonCell(a.actionButton("Src All", a.logSourceAction("all"))),
		spacerCell(),
		buttonCell(a.actionButton("App", a.logSourceAction("app"))),
		spacerCell(),
		buttonCell(a.actionButton("Xray", a.logSourceAction("xray-core"))),
	})

	searchRow := columns.New([]gowid.IContainerWidget{
		&gowid.ContainerWidget{IWidget: a.logsSearchInput, D: gowid.RenderWithWeight{W: 1}},
		&gowid.ContainerWidget{IWidget: fill.New(' '), D: gowid.RenderWithUnits{U: 1}},
		buttonCell(a.actionButton("Apply Search", a.applyLogSearchAction)),
		spacerCell(),
		buttonCell(a.actionButton("Clear Search", a.clearLogSearchAction)),
	})

	return pile.New([]gowid.IContainerWidget{
		&gowid.ContainerWidget{IWidget: styled.New(text.New("Live logs from /api/logs/stream"), gowid.MakePaletteRef("muted")), D: gowid.RenderFlow{}},
		&gowid.ContainerWidget{IWidget: divider.NewBlank(), D: gowid.RenderFlow{}},
		&gowid.ContainerWidget{IWidget: toolbar, D: gowid.RenderFlow{}},
		&gowid.ContainerWidget{IWidget: divider.NewBlank(), D: gowid.RenderFlow{}},
		&gowid.ContainerWidget{IWidget: sourceToolbar, D: gowid.RenderFlow{}},
		&gowid.ContainerWidget{IWidget: divider.NewBlank(), D: gowid.RenderFlow{}},
		&gowid.ContainerWidget{IWidget: searchRow, D: gowid.RenderFlow{}},
		&gowid.ContainerWidget{IWidget: divider.NewBlank(), D: gowid.RenderFlow{}},
		&gowid.ContainerWidget{IWidget: styled.New(a.logsStatus, gowid.MakePaletteRef("muted")), D: gowid.RenderFlow{}},
		&gowid.ContainerWidget{IWidget: divider.NewBlank(), D: gowid.RenderFlow{}},
		&gowid.ContainerWidget{IWidget: framed.NewUnicode(a.logsView), D: gowid.RenderWithWeight{W: 1}},
	})
}
