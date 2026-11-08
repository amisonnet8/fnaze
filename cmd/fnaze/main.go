// Command fnaze writes, reads and checks the description files of the sets of
// changes of a project (docs/reference/commands.md). Everything is in
// internal/cli.
package main

import (
	"os"
	"runtime/debug"

	"github.com/amisonnet8/fnaze/internal/cli"
)

// version is set by the release build (-ldflags "-X main.version=…").
var version string

func main() {
	os.Exit(cli.Run(os.Args[1:], cli.Env{
		Getenv:  os.Getenv,
		Stdout:  os.Stdout,
		Stderr:  os.Stderr,
		Version: cli.BuildVersion(version, debug.ReadBuildInfo),
	}))
}
