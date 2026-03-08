package main

import (
	"github.com/gcla/gowid"
	"github.com/gcla/gowid/widgets/columns"
	"github.com/gcla/gowid/widgets/divider"
	"github.com/gcla/gowid/widgets/fill"
	"github.com/gcla/gowid/widgets/framed"
	"github.com/gcla/gowid/widgets/pile"
)

func (a *tuiApp) buildSettingsPage() gowid.IWidget {
	saveButton := a.actionButton("Save Config", a.saveConfigAction)

	controls := columns.New([]gowid.IContainerWidget{
		buttonCell(saveButton),
		spacerCell(),
		buttonCell(a.actionButton("Clear Core Error", a.clearCoreErrorAction)),
		spacerCell(),
		buttonCell(a.actionButton("Exit Cleanup", a.exitCleanupAction)),
	})

	form := pile.New([]gowid.IContainerWidget{
		&gowid.ContainerWidget{IWidget: a.settingsListenAddr, D: gowid.RenderFlow{}},
		&gowid.ContainerWidget{IWidget: a.settingsSocksPort, D: gowid.RenderFlow{}},
		&gowid.ContainerWidget{IWidget: a.settingsHTTPPort, D: gowid.RenderFlow{}},
		&gowid.ContainerWidget{IWidget: a.settingsTunName, D: gowid.RenderFlow{}},
		&gowid.ContainerWidget{IWidget: a.settingsProxyMode, D: gowid.RenderFlow{}},
		&gowid.ContainerWidget{IWidget: a.settingsProxyExcept, D: gowid.RenderFlow{}},
	})

	body := columns.New([]gowid.IContainerWidget{
		&gowid.ContainerWidget{IWidget: framed.NewUnicode(form), D: gowid.RenderWithWeight{W: 2}},
		&gowid.ContainerWidget{IWidget: fill.New(' '), D: gowid.RenderWithUnits{U: 1}},
		&gowid.ContainerWidget{IWidget: framed.NewUnicode(a.settingsSummary), D: gowid.RenderWithWeight{W: 3}},
	})

	return pile.New([]gowid.IContainerWidget{
		&gowid.ContainerWidget{IWidget: controls, D: gowid.RenderFlow{}},
		&gowid.ContainerWidget{IWidget: divider.NewBlank(), D: gowid.RenderFlow{}},
		&gowid.ContainerWidget{IWidget: body, D: gowid.RenderWithWeight{W: 1}},
	})
}
