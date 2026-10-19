// Command gospan is the entrypoint for the Gospan text expansion engine. It
// keeps startup minimal and delegates all behavior to the cli package.
package main

import (
	"os"

	"github.com/snzpet/gospan/internal/cli"
)

// version is overridden at build time with:
//
//	-ldflags "-X main.version=1.2.3"
var version = "dev"

func main() {
	os.Exit(cli.Run(os.Args[1:], os.Stdout, os.Stderr, version))
}
