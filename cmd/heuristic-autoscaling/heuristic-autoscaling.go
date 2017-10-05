package main

import (
	ha "github.com/punytan/heuristic-autoscaling"
	"os"
)

var version string

const (
	ProgName = "heuristic-autoscaling"
)

func main() {
	cli := &ha.CLI{
		OutStream: os.Stdout,
		ErrStream: os.Stderr,
		Version:   version,
		Name:      ProgName,
	}
	os.Exit(cli.Run(os.Args))
}
