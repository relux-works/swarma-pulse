// pulse reads one explicitly admitted snapshot and emits the caller's own view.
package main

import (
	"github.com/relux-works/swarma-pulse/internal/pulse"
	"os"
)

func main() { os.Exit(pulse.Run(os.Args[1:], os.Stdout, os.Stderr)) }
