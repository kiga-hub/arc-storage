package main

import (
	"fmt"
	"os"

	"github.com/kiga-hub/arc-storage/tools/devicedata"
)

func main() {
	if err := devicedata.Cmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
