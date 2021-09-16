package rhosus

import (
	"fmt"
	"github.com/spf13/cobra"
	"parasource/rhosus/rhosus/util"
	"runtime"
)

func init() {
	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use: "version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Rhosus DFS - version %v %v %v \n", util.Version(), runtime.GOOS, runtime.GOARCH)
	},
}
