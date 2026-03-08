package tui

import (
	"github.com/gcla/gowid"
	"github.com/gcla/gowid/widgets/columns"
	"github.com/gcla/gowid/widgets/divider"
	"github.com/gcla/gowid/widgets/fill"
	"github.com/gcla/gowid/widgets/framed"
	"github.com/gcla/gowid/widgets/pile"
)

func (a *tuiApp) buildSubscriptionsPage() gowid.IWidget {
	actions := columns.New([]gowid.IContainerWidget{
		buttonCell(a.actionButton("Update All", a.updateAllSubscriptionsAction)),
		spacerCell(),
		buttonCell(a.actionButton("Update Selected", a.updateSelectedSubscriptionAction)),
	})

	body := columns.New([]gowid.IContainerWidget{
		&gowid.ContainerWidget{IWidget: framed.NewUnicode(a.subscriptionsHolder), D: gowid.RenderWithWeight{W: 2}},
		&gowid.ContainerWidget{IWidget: fill.New(' '), D: gowid.RenderWithUnits{U: 1}},
		&gowid.ContainerWidget{IWidget: framed.NewUnicode(a.subscriptionDetail), D: gowid.RenderWithWeight{W: 3}},
	})

	return pile.New([]gowid.IContainerWidget{
		&gowid.ContainerWidget{IWidget: actions, D: gowid.RenderFlow{}},
		&gowid.ContainerWidget{IWidget: divider.NewBlank(), D: gowid.RenderFlow{}},
		&gowid.ContainerWidget{IWidget: body, D: gowid.RenderWithWeight{W: 1}},
	})
}
