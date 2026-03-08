package tui

import (
	"github.com/gcla/gowid"
	"github.com/gcla/gowid/widgets/columns"
	"github.com/gcla/gowid/widgets/divider"
	"github.com/gcla/gowid/widgets/fill"
	"github.com/gcla/gowid/widgets/framed"
	"github.com/gcla/gowid/widgets/pile"
)

func (a *tuiApp) buildNetworkPage() gowid.IWidget {
	actions := columns.New([]gowid.IContainerWidget{
		buttonCell(a.actionButton("Check Network", a.reloadOverviewAction)),
		spacerCell(),
		buttonCell(a.actionButton("Apply Proxy", a.applySystemProxyAction)),
		spacerCell(),
		buttonCell(a.actionButton("Clear Proxy", a.clearSystemProxyAction)),
		spacerCell(),
		buttonCell(a.actionButton("Geo Update", a.updateGeoDataAction)),
		spacerCell(),
		buttonCell(a.actionButton("Repair TUN", a.repairTunAction)),
		spacerCell(),
		buttonCell(a.actionButton("Route Test", a.routeTestAction)),
	})

	testRow := columns.New([]gowid.IContainerWidget{
		&gowid.ContainerWidget{IWidget: a.networkTestTarget, D: gowid.RenderWithWeight{W: 4}},
		&gowid.ContainerWidget{IWidget: fill.New(' '), D: gowid.RenderWithUnits{U: 1}},
		&gowid.ContainerWidget{IWidget: a.networkTestPort, D: gowid.RenderWithWeight{W: 1}},
	})

	body := columns.New([]gowid.IContainerWidget{
		&gowid.ContainerWidget{IWidget: framed.NewUnicode(a.networkSummary), D: gowid.RenderWithWeight{W: 3}},
		&gowid.ContainerWidget{IWidget: fill.New(' '), D: gowid.RenderWithUnits{U: 1}},
		&gowid.ContainerWidget{IWidget: framed.NewUnicode(a.networkTestResult), D: gowid.RenderWithWeight{W: 2}},
	})

	return pile.New([]gowid.IContainerWidget{
		&gowid.ContainerWidget{IWidget: actions, D: gowid.RenderFlow{}},
		&gowid.ContainerWidget{IWidget: divider.NewBlank(), D: gowid.RenderFlow{}},
		&gowid.ContainerWidget{IWidget: testRow, D: gowid.RenderFlow{}},
		&gowid.ContainerWidget{IWidget: divider.NewBlank(), D: gowid.RenderFlow{}},
		&gowid.ContainerWidget{IWidget: body, D: gowid.RenderWithWeight{W: 1}},
	})
}
