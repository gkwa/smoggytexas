package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/taylormonacelli/smoggytexas"
)

var instanceTypes, ignoreCommaSepRegions string

func init() {
	flag.StringVar(&instanceTypes, "instanceTypes", "", "Comma-separated list of instance types to query")
	flag.StringVar(&ignoreCommaSepRegions, "ignoreRegions", "", "")
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
