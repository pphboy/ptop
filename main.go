package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/signal"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/rivo/tview"
)

type cpuStat struct {
	total float64
	used  float64
}

func main() {
	app := tview.NewApplication()

	tvcpuInfos := tview.NewTextView().SetText("cpuInfos")
	tvmemInfos := tview.NewTextView().SetText("memInfos")
	tvprocListInfos := tview.NewTextView().SetText("procList")

	flex := tview.NewFlex().
		AddItem(
			tview.NewFlex().
				SetDirection(tview.FlexRow).
				AddItem(tvcpuInfos, 4, 1, false).
				AddItem(tvmemInfos, 1, 1, false).
				AddItem(tvprocListInfos, 20, 1, false), 0, 1, false)

	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, os.Interrupt)

	var wg sync.WaitGroup
	ctx := context.TODO()
	go func() {
		sig := <-sigchan
		fmt.Println("sig: ", sig)
		ctx.Done()
		app.Stop()
		fmt.Println("exit 0")
		os.Exit(0)
	}()

	// cpu信息
	wg.Add(1)
	go func() {
		defer wg.Done()

		cpuInfoBuf := bytes.NewBuffer(nil)
		s1 := cpuRecords()
		refershCpuInfo := func() {
			cpuRecords := cpuRecords()
			for cpuindex, cpuRecord := range cpuRecords {
				fmt.Fprintf(cpuInfoBuf, "cpu%d:%.2f%%\t\t",
					cpuindex,
					(cpuRecord.used-s1[cpuindex].used)/(cpuRecord.total-s1[cpuindex].total)*100)
			}
			app.QueueUpdateDraw(func() {
				tvcpuInfos.SetText(cpuInfoBuf.String())
				cpuInfoBuf.Reset()
			})
			s1 = cpuRecords
			time.Sleep(1 * time.Second)
		}
		for {
			select {
			case <-ctx.Done():
				return
			default:
				refershCpuInfo()
			}
		}
	}()

	// 内存信息
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			default:
				app.QueueUpdateDraw(func() {
					total, avail := getMemInfos()
					tvmemInfos.SetText(fmt.Sprintf("Used: %.0f\tAvail: %.0f\tTotal: %.0f ", (total - avail), avail, total))
				})
				time.Sleep(1 * time.Second)
			}
		}
	}()

	// 进程信息
	wg.Add(1)
	go func() {
		defer wg.Done()

		p1 := procInfoList()
		c1 := procCpuTotal()
		procBufs := bytes.NewBuffer(nil)
		totalMem, _ := getMemInfos()
		for {
			select {
			case <-ctx.Done():
				return
			default:
				procBufs.WriteByte('\n')
				p2 := procInfoList()
				c2 := procCpuTotal()
				var ps []procInfo
				for k, p := range p2 {
					p.str = fmt.Sprintf("Pid: %d\tCpu:%.4f%%\tRes: %.3fMiB(%.4f%%)\tCmd: %s",
						p.pid,
						(p.utime-p1[k].utime)/(c2-c1)*100, p.mem/1024/1024, (p.mem/totalMem)*100, p.cmdline[:min(len(p.cmdline), 30)])
					ps = append(ps, p)
				}

				slices.SortFunc(ps, func(p1, p2 procInfo) int {
					return p2.pid - p1.pid
				})
				for _, p := range ps {
					procBufs.WriteString(p.str)
					procBufs.WriteByte('\n')
				}

				p1 = p2
				c1 = c2
				app.QueueUpdateDraw(func() {
					tvprocListInfos.SetText(procBufs.String())
					procBufs.Reset()
				})
				time.Sleep(1 * time.Second)
			}
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := app.SetRoot(flex, true).EnableMouse(true).Run(); err != nil {
			panic(err)
		}
	}()
	wg.Wait()
}

func getMemInfos() (total, avail float64) {
	info, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		panic(err)
	}
	memLines := strings.Split(string(info), "\n")
	var memTotal float64
	var memAvail float64
	for _, l := range memLines {
		if !strings.HasPrefix(l, "Mem") {
			continue
		}
		a := strings.Fields(l)
		// fmt.Printf("%v, %v, %v\n", a, len(a), a[1])
		mem, _ := strconv.ParseFloat(a[1], 64)
		switch a[0] {
		case "MemTotal:":
			memTotal = mem
		case "MemAvailable:":
			memAvail = mem
		}
	}
	return memTotal * 1000, memAvail * 1000
}

func getCpuLines() []string {
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

func cpuRecords() []cpuStat {
	cpuRecord := make([]cpuStat, len(getCpuLines()))
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
		cpuRecord[cpuindex].total = total
		cpuRecord[cpuindex].used = total - idle
	}
	return cpuRecord
}

func procCpuTotal() float64 {
	var total float64
	for _, v := range cpuRecords() {
		total += v.total
	}
	return total
}
func procCmdLine(pid int) string {
	cmdline, _ := os.ReadFile(fmt.Sprintf("/proc/%d/cmdline", pid))
	return string(cmdline)
}

func procTimes(pid int) float64 {
	times, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
	if err != nil {
		return 0
	}
	pci := strings.Fields(string(times))
	// fmt.Printf("%+v %+v ", pci[13], pci[14])
	p13, _ := strconv.ParseFloat(pci[13], 64)
	p14, _ := strconv.ParseFloat(pci[14], 64)
	return p13 + p14
}

func procMem(pid int) float64 {
	statmem, err := os.ReadFile(fmt.Sprintf("/proc/%d/status", pid))
	if err != nil {
		return 0
	}
	for line := range strings.SplitSeq(string(statmem), "\n") {
		if strings.HasPrefix(line, "VmRSS:") {
			statmems := strings.Fields(string(line))
			rs, _ := strconv.ParseFloat(statmems[1], 64)
			return rs * 1000
		}
	}
	return 0
}

type procInfo struct {
	utime   float64
	mem     float64
	cmdline string
	pid     int
	str     string
}

func procInfoList() map[int]procInfo {
	procs, err := os.ReadDir("/proc")
	if err != nil {
		return nil
	}

	var pids []int
	for _, p := range procs {
		pid, err := strconv.ParseInt(p.Name(), 10, 32)
		if err != nil {
			continue
		}
		pids = append(pids, int(pid))
	}

	res := make(map[int]procInfo)
	for _, p := range pids {
		res[p] = procInfo{
			utime:   procTimes(p),
			cmdline: procCmdLine(p),
			mem:     procMem(p),
			pid:     p,
		}
	}
	return res
}
