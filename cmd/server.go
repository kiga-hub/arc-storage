package cmd

import (
	"os"
	"runtime/debug"
	"strings"
	"time"

	"github.com/davecgh/go-spew/spew"
	kafkaComponent "github.com/kiga-hub/arc/kafka"
	"github.com/kiga-hub/arc/micro"
	basicComponent "github.com/kiga-hub/arc/micro/component"
	tracing "github.com/kiga-hub/arc/tracing"
	"github.com/spf13/cobra"

	"github.com/kiga-hub/arc-storage/pkg"
	"github.com/kiga-hub/arc-storage/pkg/component"
)

func init() {
	spew.Config = *spew.NewDefaultConfig()
	spew.Config.ContinueOnMethod = true
}

var serverCmd = &cobra.Command{
	Use:   "run",
	Short: "run arc-storage server",
	Run:   run,
}

func run(cmd *cobra.Command, args []string) {
	// recover
	defer func() {
		if r := recover(); r != nil {
			debug.PrintStack()
			time.Sleep(time.Second * 5)
			os.Exit(1)
		}
	}()

	server, err := micro.NewServer(
		AppName,
		AppVersion,
		[]micro.IComponent{
			&basicComponent.LoggingComponent{},
			&tracing.Component{},
			&basicComponent.GossipKVCacheComponent{
				ClusterName:   "platform-global",
				Port:          6666,
				InMachineMode: false,
			},
			&kafkaComponent.Component{},
			&component.ArcStorageComponent{},
		},
	)
	pkg.ArcStorageVersion = AppVersion
	if err != nil {
		panic(err)
	}

	err = server.Init()
	if err != nil {
		panic(err)
	}

	setMiddleWareSkipper(server)

	err = server.Run()
	if err != nil {
		panic(err)
	}
}

func setMiddleWareSkipper(s *micro.Server) {
	// 压缩中间件Skipper
	s.GzipSkipper = func(uri string) bool {
		return strings.Contains(uri, "/arc")
	}

	// 自定义限流Skipper
	s.APIRateSkipper = func(uri string) bool {
		return !strings.Contains(uri, "/history")
	}

	// 自定义POST Content大小Skipper
	s.APIBodySkipper = func(uri string) bool {
		return !strings.Contains(uri, "/arc")
	}

	// 自定义超时Skipper
	s.APITimeOutSkipper = func(uri string) bool {
		return !strings.Contains(uri, "/history")
	}
}
