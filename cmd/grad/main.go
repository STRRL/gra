package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "grad",
	Short: "Grad - A CLI tool",
	Long:  `Grad is a command-line interface tool.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("grad called")
	},
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