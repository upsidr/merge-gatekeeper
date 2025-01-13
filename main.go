// Package main is the entry point for the gatekeeper application.
package main

import (
	_ "embed"
	"fmt"
	"os"
	"strings"

	"github.com/argandtech/gatekeeper/internal/cli"
)

var (
	version string
)

// main is the entry point for the application.
func main() {
	if err := cli.Run(strings.TrimSuffix(version, "\n"), os.Args...); err != nil {
		fmt.Fprintf(os.Stderr, "failed to execute command: %v", err)
		os.Exit(1)
	}
}
