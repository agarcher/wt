package main

import (
	"os"

	"github.com/agarcher/wt/internal/commands"
)

func main() {
	if err := commands.Execute(); err != nil {
		os.Exit(1)
	}
}
