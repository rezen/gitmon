package gitmon

import (
	"runtime"
)

type Monitor struct {
	Alloc,
	TotalAlloc,
	Sys,
	Mallocs,
	Frees,
	LiveObjects,
	PauseTotalNs uint64
	NumCgoCall int64
	NumGC        uint32
	NumGoroutine int
}

func GetMonitorStats() Monitor {
	var m Monitor
	var rtm runtime.MemStats
	runtime.ReadMemStats(&rtm)
	m.NumGoroutine = runtime.NumGoroutine()
	m.NumCgoCall = runtime.NumCgoCall()
	m.Alloc = rtm.Alloc
	m.TotalAlloc = rtm.TotalAlloc
	m.Sys = rtm.Sys
	m.Mallocs = rtm.Mallocs
	m.Frees = rtm.Frees
	m.LiveObjects = m.Mallocs - m.Frees
	m.PauseTotalNs = rtm.PauseTotalNs
	m.NumGC = rtm.NumGC
	return m
}
