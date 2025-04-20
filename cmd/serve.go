package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	serveCmd.Run = runServe
}

func runServe(cmd *cobra.Command, args []string) {
	fmt.Println("Serve command called (implementation pending)...")
}
