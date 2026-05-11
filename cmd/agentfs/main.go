package main

import (
	"agentfs/cmd/agentfs/cmd"
)

var (
	version   = "dev"
	buildTime = "unknown"
)

func main() {
	cmd.Execute(version, buildTime)
}
