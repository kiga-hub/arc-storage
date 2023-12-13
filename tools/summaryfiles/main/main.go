package main

import (
	"fmt"
	"os"

	"github.com/kiga-hub/arc-storage/tools/summaryfiles"
)

func main() {
	if err := summaryfiles.Cmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
