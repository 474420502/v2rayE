package main

import (
	"github.com/gcla/gowid"
	"github.com/gcla/gowid/widgets/columns"
	"github.com/gcla/gowid/widgets/divider"
	"github.com/gcla/gowid/widgets/fill"
	"github.com/gcla/gowid/widgets/framed"
	"github.com/gcla/gowid/widgets/pile"
	"github.com/gcla/gowid/widgets/styled"
)

func (a *tuiApp) buildProfilesPage() gowid.IWidget {
	importRow := columns.New([]gowid.IContainerWidget{
		&gowid.ContainerWidget{IWidget: a.profileImport, D: gowid.RenderWithWeight{W: 1}},
		spacerCell(),
		buttonCell(a.actionButton("Import URI", a.importProfileAction)),
	})

	actions := columns.New([]gowid.IContainerWidget{
		buttonCell(a.actionButton("Activate", a.activateProfileAction)),
		spacerCell(),
		buttonCell(a.actionButton("Batch Delay", a.batchDelayProfilesAction)),
		spacerCell(),
		buttonCell(a.actionButton("Delay Test", a.testSelectedProfileDelayAction)),
	})

	body := columns.New([]gowid.IContainerWidget{
		&gowid.ContainerWidget{IWidget: framed.NewUnicode(a.profilesListHolder), D: gowid.RenderWithWeight{W: 2}},
		&gowid.ContainerWidget{IWidget: fill.New(' '), D: gowid.RenderWithUnits{U: 1}},
		&gowid.ContainerWidget{IWidget: framed.NewUnicode(a.profileDetail), D: gowid.RenderWithWeight{W: 3}},
	})

	return pile.New([]gowid.IContainerWidget{
		&gowid.ContainerWidget{IWidget: importRow, D: gowid.RenderFlow{}},
		&gowid.ContainerWidget{IWidget: divider.NewBlank(), D: gowid.RenderFlow{}},
		&gowid.ContainerWidget{IWidget: styled.New(a.profileBatchStatus, gowid.MakePaletteRef("muted")), D: gowid.RenderFlow{}},
		&gowid.ContainerWidget{IWidget: divider.NewBlank(), D: gowid.RenderFlow{}},
		&gowid.ContainerWidget{IWidget: actions, D: gowid.RenderFlow{}},
		&gowid.ContainerWidget{IWidget: divider.NewBlank(), D: gowid.RenderFlow{}},
		&gowid.ContainerWidget{IWidget: body, D: gowid.RenderWithWeight{W: 1}},
	})
}
