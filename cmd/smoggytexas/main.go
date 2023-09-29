package main

import (
	"flag"
	"fmt"
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
	flag.Parse()

	// Check if "instanceTypes" is empty and exit with an error if it is
	if instanceTypes == "" {
		fmt.Println("Error: The 'instanceTypes' flag is required.")
		flag.Usage()
		os.Exit(1)
	}

	status := smoggytexas.Main(instanceTypes, verbose)
	os.Exit(status)
}
