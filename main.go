package main

import (
	"bytes"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func main() {
	app := tview.NewApplication()

	cpuInfo := tview.NewTextView().SetText("123")
	frame := tview.NewFrame(cpuInfo).
		SetBorders(2, 2, 2, 2, 4, 4).
		AddText("Ptop", true, tview.AlignLeft, tcell.ColorWhite).
		AddText("powered by pphboy", false, tview.AlignCenter, tcell.ColorGreen)

	getCpuLines := func() []string {
		a, err := os.ReadFile("/proc/stat")
		if err != nil {
			panic(err)
		}
		var cpulines []string
		stat := strings.SplitSeq(string(a), "\n")
		for l := range stat {
			if !strings.HasPrefix(l, "cpu") {
				continue
			}
			cpulines = append(cpulines, l)
		}
		return cpulines
	}

	type cpuStat struct {
		total float64
		used  float64
	}

	go func() {
		cpuRecord := make([]cpuStat, len(getCpuLines()))
		cpuInfoBuf := bytes.NewBuffer(nil)
		for {
			for cpuindex, cpuline := range getCpuLines() {
				var total float64
				var idle float64
				for i, f := range strings.Fields(cpuline) {
					if i == 0 || i == 10 || i == 9 {
						continue
					}
					flt, err := strconv.ParseFloat(f, 32)
					if err == nil {
						if i == 4 || i == 5 {
							idle += flt
						}
						total += flt
					}
				}

				used := total - idle
				if cpuRecord[cpuindex].total == 0 {
					cpuRecord[cpuindex].total = total
					cpuRecord[cpuindex].used = used
					continue
				}

				fmt.Fprintf(cpuInfoBuf, "cpu%d:%.2f%%\t", cpuindex, (used-cpuRecord[cpuindex].used)/(total-cpuRecord[cpuindex].total)*100)
				cpuRecord[cpuindex].total = total
				cpuRecord[cpuindex].used = used
			}
			app.QueueUpdateDraw(func() {
				cpuInfo.SetText(cpuInfoBuf.String())
			})
			time.Sleep(1 * time.Second)
			cpuInfoBuf.Reset()
		}
	}()

	if err := app.SetRoot(frame, true).EnableMouse(true).Run(); err != nil {
		panic(err)
	}
}
