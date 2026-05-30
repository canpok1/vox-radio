package main

import (
	"fmt"
	"os"

	"github.com/canpok1/vox-radio/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
