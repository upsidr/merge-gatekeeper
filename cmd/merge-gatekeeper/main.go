package main

import (
	_ "embed"
	"fmt"
	"os"
	"strings"

	"github.com/upsidr/merge-gatekeeper/internal/cli"
)

var (
	//go:embed version.txt
	version string
)

func main() {
	if err := cli.Run(strings.TrimSuffix(version, "\n"), os.Args...); err != nil {
		fmt.Fprintf(os.Stderr, "failed to execute command: %v", err)
		os.Exit(1)
	}
}
