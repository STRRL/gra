package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "gractl",
	Short: "Gractl - A CLI control tool",
	Long:  `Gractl is a command-line control interface tool.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("gractl called")
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