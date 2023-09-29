package main

import (
	"flag"
	"os"

	"github.com/taylormonacelli/smoggytexas"
)

var (
	verbose       bool
	instanceTypes string
)

func init() {
	const (
		defaultVerbosEnabled = false
		usage                = "Enable verbose mode"
	)
	flag.StringVar(&instanceTypes, "instanceTypes", "", "Comma-separated list of instance types to query")

	flag.BoolVar(&verbose, "verbose", defaultVerbosEnabled, usage)
	flag.BoolVar(&verbose, "v", defaultVerbosEnabled, usage+" (shorthand)")
}

func main() {
	status := smoggytexas.Main()
	os.Exit(status)
}
