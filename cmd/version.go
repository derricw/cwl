package cmd

import (
	"fmt"
	"runtime/debug"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of cwl",
	Long:  `All software has versions. This is cwl's`,
	Run: func(cmd *cobra.Command, args []string) {
		info, ok := debug.ReadBuildInfo()
		if ok {
			// installed with `go install`
			fmt.Println(info.Main.Version)
		} else {
			// built manually
			fmt.Println("build from source")
		}
	},
}
