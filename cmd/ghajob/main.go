package main

import (
	"fmt"
	"os"

	"github.com/upsidr/check-other-job-status/internal/cli"
)

const (
	version = "v0.0.1"
)

func main() {
	if err := cli.Run(version, os.Args...); err != nil {
		fmt.Fprintf(os.Stderr, "failed to execute command: %v", err)
		os.Exit(1)
	}
}
