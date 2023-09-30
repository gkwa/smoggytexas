package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/taylormonacelli/smoggytexas"
)

var instanceTypes, ignoreCommaSepRegions string

func init() {
	flag.StringVar(&instanceTypes, "instanceTypes", "", "Comma-separated list of instance types to query, eg. t3a.xlarge,t3.small")
	flag.StringVar(&ignoreCommaSepRegions, "ignoreRegions", "", "Exclude regions that start with, eg: cn-north-1,cn-")
}

func main() {
	flag.Parse()

	// Check if "instanceTypes" is empty and exit with an error if it is
	if instanceTypes == "" {
		fmt.Println("Error: The 'instanceTypes' flag is required.")
		flag.Usage()
		os.Exit(1)
	}

	status := smoggytexas.Main(instanceTypes, ignoreCommaSepRegions)
	os.Exit(status)
}
