package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/taylormonacelli/goldbug"
	"github.com/taylormonacelli/smoggytexas"
)

var (
	instanceTypes, ignoreCommaSepRegions string
	verbose                              bool
)

func init() {
	flag.StringVar(&instanceTypes, "instanceTypes", "t3a.xlarge", "Comma-separated list of instance types to query, eg. t3a.xlarge,t3.small")
	flag.StringVar(&ignoreCommaSepRegions, "ignoreRegions", "", "Exclude regions that start with, eg: cn-north-1,cn-")
	flag.BoolVar(&verbose, "verbose", false, "Enable verbose output")
	flag.BoolVar(&verbose, "v", false, "Enable verbose output (shorthand)")
	flag.Parse()
}

func main() {
	flag.Parse()

	// Check if "instanceTypes" is empty and exit with an error if it is
	if instanceTypes == "" {
		fmt.Println("Error: The 'instanceTypes' flag is required.")
		flag.Usage()
		os.Exit(1)
	}

	if verbose {
		goldbug.SetDefaultLoggerText(slog.LevelDebug)
	}

	status := smoggytexas.Main(instanceTypes, ignoreCommaSepRegions)
	os.Exit(status)
}
