package main

import (
	"github.com/neul-labs/stratafs/cmd/stratafs/cmd"
)

var (
	version   = "dev"
	buildTime = "unknown"
)

func main() {
	cmd.Execute(version, buildTime)
}
