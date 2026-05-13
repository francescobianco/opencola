package main

import (
	"fmt"
	"os"

	"github.com/opencola/opencola/cmd"
)

func main() {
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
