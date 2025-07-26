package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/strrl/gra/cmd/gractl/cmd"
)

var rootCmd = &cobra.Command{
	Use:   "gractl",
	Short: "Gractl - A CLI control tool for grad",
	Long:  `Gractl is a command-line control interface tool for managing grad runners and executing remote commands.`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func init() {
	// Register subcommands
	rootCmd.AddCommand(cmd.RunnersCmd)
	rootCmd.AddCommand(cmd.ExecuteCmd)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func main() {
	Execute()
}
