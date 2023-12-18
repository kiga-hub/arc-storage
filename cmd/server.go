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
	// compression middleware
	s.GzipSkipper = func(uri string) bool {
		return strings.Contains(uri, "/arc")
	}

	// rate limiting
	s.APIRateSkipper = func(uri string) bool {
		return !strings.Contains(uri, "/history")
	}

	// post content size
	s.APIBodySkipper = func(uri string) bool {
		return !strings.Contains(uri, "/arc")
	}

	// timeout middleware
	s.APITimeOutSkipper = func(uri string) bool {
		return !strings.Contains(uri, "/history")
	}
}
