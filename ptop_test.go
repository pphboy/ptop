package main

import (
	"fmt"
	"testing"
)

func TestMem(t *testing.T) {
	memTotal, memAvail := getMemInfos()

	fmt.Printf("Total: %f, Avail: %f\n", memTotal, memAvail)
}

func TestProcList(t *testing.T) {
	t.Logf("%+v", procInfoList())
}

func TestProcMem(t *testing.T) {
	t.Log(procMem(419))
}
