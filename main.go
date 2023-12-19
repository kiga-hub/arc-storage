package main

import (
	"fmt"
	"runtime"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/kiga-hub/arc-storage/cmd"
	_ "go.uber.org/automaxprocs"
)

func main() {
	Version()
	go DoMonitor(15, func(stat *MonitorStat) {
		fmt.Printf("==== stat %s === \nalloc:\t%d\ttotal:\t%d\tsys:\t%d\tmallocs: %d\tfrees:\t%d\npause:\t%d\tgc:\t%d\tgoroutine: %d\nheap: %d / %d / %d\tstack: %d\n",
			time.Now().String(),
			stat.Alloc, stat.TotalAlloc, stat.Sys, stat.Mallocs, stat.Frees,
			stat.PauseTotalNs, stat.NumGC, stat.NumGoroutine,
			stat.HeapIdle, stat.HeapInuse, stat.HeapReleased, stat.StackInuse,
		)
	})
	spew.Config = *spew.NewDefaultConfig()
	spew.Config.ContinueOnMethod = true
	cmd.AppName = AppName
	cmd.AppVersion = AppVersion
	cmd.Execute()
}

var (
	// AppName - 应用名称
	AppName string
	// AppVersion - 应用版本
	AppVersion string
	// BuildVersion - 编译版本
	BuildVersion string
	// BuildTime - 编译时间
	BuildTime string
	// GitRevision - Git版本
	GitRevision string
	// GitBranch - Git分支
	GitBranch string
	// GoVersion - Golang信息
	GoVersion string
)

// Version 版本信息
func Version() {
	fmt.Printf("App Name:\t%s\n", AppName)
	fmt.Printf("App Version:\t%s\n", AppVersion)
	fmt.Printf("Build version:\t%s\n", BuildVersion)
	fmt.Printf("Build time:\t%s\n", BuildTime)
	fmt.Printf("Git revision:\t%s\n", GitRevision)
	fmt.Printf("Git branch:\t%s\n", GitBranch)
	fmt.Printf("Golang Version: %s\n", GoVersion)
	fmt.Printf("Runtime CPUs:\t%d\n", runtime.NumCPU())
	fmt.Printf("Runtime GoRoot:\t%s\n", runtime.GOROOT())
	fmt.Printf("Runtime GoOS:\t%s\n", runtime.GOOS)
	fmt.Printf("Runtime GoArch:\t%s\n", runtime.GOARCH)
	fmt.Printf("Compiler:\t%s\n", runtime.Compiler)
}

// MonitorStat is the state of the runtime
type MonitorStat struct {
	runtime.MemStats
	LiveObjects  uint64 `json:"live_objects,omitempty"`  // Live objects = Mallocs - Frees
	NumGoroutine int    `json:"num_goroutine,omitempty"` // Number of goroutines
}

// DoMonitor start a loop for monitor
func DoMonitor(duration int, callback func(*MonitorStat)) {
	interval := time.Duration(duration) * time.Second
	timer := time.Tick(interval)
	for range timer {
		var rtm runtime.MemStats
		runtime.ReadMemStats(&rtm)
		callback(&MonitorStat{
			MemStats:     rtm,
			NumGoroutine: runtime.NumGoroutine(),
			LiveObjects:  rtm.Mallocs - rtm.Frees,
		})
	}
}
