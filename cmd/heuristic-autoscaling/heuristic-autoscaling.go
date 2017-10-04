package main

import (
	ha "github.com/punytan/heuristic-autoscaling"
	"os"
)

const (
	Version  = "v0.0.1"
	ProgName = "heuristic-autoscaling"
)

func main() {
	cli := &ha.CLI{
		OutStream: os.Stdout,
		ErrStream: os.Stderr,
		Version:   Version,
		Name:      ProgName,
	}
	os.Exit(cli.Run(os.Args))
}
