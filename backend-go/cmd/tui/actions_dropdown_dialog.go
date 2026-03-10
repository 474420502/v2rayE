package tui

import (
	"strings"

	tcell "github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

const dropdownSelectDialogPage = "dropdown-select-dialog"

func (a *tuiApp) openDropdownSelectDialog(widget *dropdownWidget) {
	a.runUI(func(app *tview.Application) {
		if widget == nil || a.pageHolder == nil || a.dropdownSelectVisible.Load() {
			return
		}
		if a.commandPaletteVisible.Load() || a.profileActionsVisible.Load() || a.profileDeleteVisible.Load() || a.profileImportVisible.Load() || a.profileEditVisible.Load() || a.proxyUserSelectVisible.Load() {
			return
		}
		if len(widget.options) == 0 {
			return
		}

		list := newListWidget()
		list.ShowSecondaryText(false)
		list.SetBorder(true)
		optionCount := len(widget.options)
		listTitle := a.tf("dialog.dropdown.select.listTitle", optionCount)
		list.SetTitle(" " + listTitle + " ")
 
		optionLabels := make([]string, 0, optionCount)

		for index, option := range widget.options {
			idx := index
			label := option.Label
			if strings.TrimSpace(label) == "" {
				label = option.Value
			}
			optionLabels = append(optionLabels, label)
			list.AddItem(label, "", 0, func() {
				widget.SetCurrentOption(idx)
				a.closeDropdownSelectDialog(true)
			})
		}

		current, _ := widget.GetCurrentOption()
		if current >= 0 && current < list.GetItemCount() {
			list.SetCurrentItem(current)
		}

		list.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
			switch event.Key() {
			case tcell.KeyEsc:
				a.closeDropdownSelectDialog(false)
				return nil
			case tcell.KeyLeft, tcell.KeyRight, tcell.KeyTAB, tcell.KeyBacktab:
				// Overlay list only supports up/down movement and Enter confirm.
				return nil
			}
			return event
		})

		height := dropdownDialogHeight(optionCount, a.viewportRows)

		title := strings.TrimSpace(widget.GetLabel())
		if title == "" {
			title = a.t("dialog.dropdown.select.defaultTitle")
		}
		containerTitle := a.tf("dialog.dropdown.select.containerTitle", title, optionCount)
		hintText := a.tf("dialog.dropdown.select.hint", optionCount)
		width := dropdownDialogWidth(containerTitle, hintText, optionLabels, a.viewportCols)

		container := tview.NewFlex().SetDirection(tview.FlexRow)
		container.SetBorder(true)
		container.SetTitle(" " + containerTitle + " ")
		container.AddItem(newMutedText(hintText), 1, 0, false)
		container.AddItem(verticalSpacer(1), 1, 0, false)
		container.AddItem(list, 0, 1, true)

		a.dropdownSelectMenu = list
		a.dropdownSelectPrev = app.GetFocus()
		a.dropdownSelectTarget = widget
		a.pageHolder.AddPage(dropdownSelectDialogPage, centeredPrimitive(container, width, height), true, true)
		a.dropdownSelectVisible.Store(true)
		app.SetFocus(list)
	})
}

func dropdownDialogWidth(title, hint string, optionLabels []string, viewportCols int) int {
	width := tview.TaggedStringWidth(title) + 6

	if hintWidth := tview.TaggedStringWidth(hint) + 4; hintWidth > width {
		width = hintWidth
	}
	for _, label := range optionLabels {
		if labelWidth := tview.TaggedStringWidth(label) + 6; labelWidth > width {
			width = labelWidth
		}
	}

	maxWidth := 96
	if viewportCols > 0 {
		// Keep modal visually balanced on wide screens: at least ~55% viewport width.
		ratioMin := viewportCols * 55 / 100
		if ratioMin > width {
			width = ratioMin
		}

		allowed := viewportCols - 6
		if allowed < 24 {
			allowed = 24
		}
		if allowed < maxWidth {
			maxWidth = allowed
		}
	}

	minWidth := 40
	if maxWidth < minWidth {
		minWidth = maxWidth
	}
	if width < minWidth {
		width = minWidth
	}
	if width > maxWidth {
		width = maxWidth
	}

	return width
}

func dropdownDialogHeight(optionCount, viewportRows int) int {
	visibleRows := optionCount
	if visibleRows < 6 {
		visibleRows = 6
	}
	height := visibleRows + 4

	if viewportRows > 0 {
		// Keep modal height at a stable fraction of viewport for better visual balance.
		ratioMin := viewportRows * 40 / 100
		if ratioMin > height {
			height = ratioMin
		}

		ratioMax := viewportRows * 70 / 100
		if ratioMax < 8 {
			ratioMax = 8
		}
		if height > ratioMax {
			height = ratioMax
		}
	}

	if height < 10 {
		height = 10
	}
	if height > 28 {
		height = 28
	}

	return height
}

func (a *tuiApp) closeDropdownSelectDialog(restoreFocus bool) {
	a.runUI(func(app *tview.Application) {
		if a.pageHolder == nil || !a.dropdownSelectVisible.Load() {
			return
		}
		a.pageHolder.RemovePage(dropdownSelectDialogPage)
		a.dropdownSelectVisible.Store(false)
		a.dropdownSelectMenu = nil
		target := a.dropdownSelectTarget
		a.dropdownSelectTarget = nil
		if restoreFocus {
			if target != nil {
				app.SetFocus(target)
			} else if a.dropdownSelectPrev != nil {
				app.SetFocus(a.dropdownSelectPrev)
			} else if len(a.focusables) > 0 {
				app.SetFocus(a.focusables[0])
			}
		}
		a.dropdownSelectPrev = nil
	})
}
