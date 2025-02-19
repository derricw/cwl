package cmd

import (
	"fmt"
	"log"

	"github.com/derricw/cwl/fetch"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(groupsCmd)
}

var groupsCmd = &cobra.Command{
	Use:   "groups",
	Short: "list groups",
	Long:  `Lists all available log groups`,
	Run: func(cmd *cobra.Command, args []string) {

		client, err := fetch.CreateClient()
		if err != nil {
			log.Fatal(err)
		}
		groups, err := fetch.FetchLogGroups(client)
		if err != nil {
			log.Fatal(err)
		}
		for _, group := range groups {
			fmt.Printf("%s\n", *group.LogGroupName)
		}
	},
}
