package main

import (
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func main() {
	app := tview.NewApplication()

	cpuInfo := tview.NewTextView().SetText("123")
	frame := tview.NewFrame(cpuInfo).
		SetBorders(2, 2, 2, 2, 4, 4).
		AddText("Header left", true, tview.AlignLeft, tcell.ColorWhite).
		AddText("Header middle", true, tview.AlignCenter, tcell.ColorWhite).
		AddText("Header right", true, tview.AlignRight, tcell.ColorWhite).
		AddText("Header second middle", true, tview.AlignCenter, tcell.ColorRed).
		AddText("Footer middle", false, tview.AlignCenter, tcell.ColorGreen).
		AddText("Footer second middle", false, tview.AlignCenter, tcell.ColorGreen)

	go func() {
		for {
			app.QueueUpdateDraw(func() {
				cpuInfo.SetText(time.Now().String())
			})
			time.Sleep(100 * time.Millisecond)
		}
	}()
	if err := app.SetRoot(frame, true).EnableMouse(true).Run(); err != nil {
		panic(err)
	}

}
