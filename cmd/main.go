package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"sort"
	"sync"
	"text/template"
	"time"

	"github.com/taylormonacelli/lemondrop"
	"gopkg.in/alessio/shellescape.v1"
)

type AZs []ZonePrice

type ZonePrice struct {
	AZ    string `json:"az"`
	Price string `json:"price"`
}

type DescribePriceCommand struct {
	Region       string
	InstanceType string
	Epoch        int64
	CmdStr       string
	Command      exec.Cmd
	Description  string
	Query        string
}

func runCommand(cmd DescribePriceCommand, wg *sync.WaitGroup, semaphore chan struct{}, resultsChan chan<- AZs) {
	defer wg.Done()

	semaphore <- struct{}{}

	// Release the semaphore slot when the function exits
	defer func() {
		<-semaphore
	}()

	out, err := cmd.Command.Output()
	if err != nil {
		log.Printf("Error running AWS CLI command: %v\n", err)
	}

	log.Println(cmd.Command.String())

	var azs AZs
	err = json.Unmarshal(out, &azs)
	if err != nil {
		panic(nil)
	}

	resultsChan <- azs
}

func main() {
	var instanceType string

	flag.StringVar(&instanceType, "instanceType", "", "The instance type to use")

	flag.Parse()

	// Check if "instanceType" is empty and exit with an error if it is
	if instanceType == "" {
		fmt.Println("Error: The 'instanceType' flag is required.")
		flag.Usage()
		os.Exit(1)
	}

	// Access the value of the "instanceType" flag
	fmt.Printf("Instance Type: %s\n", instanceType)
	regions, err := lemondrop.GetAllAwsRegions()
	if err != nil {
		log.Fatal(err)
	}

	var cmds []DescribePriceCommand

	cmdTemplate := "aws ec2 describe-spot-price-history --region={{ .Region }} --instance-types={{ .InstanceType }} --start-time={{ .Epoch }} --product-descriptions={{ .Description }} --query={{ .Query }}"
	tmpl := template.Must(template.New("simple").Parse(cmdTemplate))

	for _, region := range regions {
		data := DescribePriceCommand{
			Region:       *region.RegionName,
			InstanceType: instanceType,
			Epoch:        time.Now().Unix(),
			Query:        shellescape.Quote("SpotPriceHistory[*].{az:AvailabilityZone, price:SpotPrice}"),
			Description:  shellescape.Quote("Linux/UNIX"),
		}

		var cmdTplRendered bytes.Buffer

		err := tmpl.Execute(&cmdTplRendered, data)
		if err != nil {
			fmt.Println("Error executing template:", err)
			return
		}

		cmd := exec.Command("bash", "-c", cmdTplRendered.String())
		data.Command = *cmd
		cmds = append(cmds, data)
	}

	var wg sync.WaitGroup

	// Limit the number of concurrently running commands to maxConcurrent
	maxConcurrent := 20
	semaphore := make(chan struct{}, maxConcurrent)
	resultsChan := make(chan AZs, len(cmds))

	for i := 0; i < len(cmds); i++ {
		cmd := cmds[i]

		wg.Add(1)
		go runCommand(cmd, &wg, semaphore, resultsChan)
	}

	// Wait for all goroutines to finish and collect results
	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	var y AZs
	// Collect results from the channel
	for azs := range resultsChan {
		y = append(y, azs...)
	}

	// Define a custom sorting function
	sortByPrice := func(i, j int) bool {
		price1 := y[i].Price
		price2 := y[j].Price
		return price1 > price2
	}

	// Sort the y slice by price
	sort.Slice(y, sortByPrice)

	jsonBytes, err := json.MarshalIndent(y, "", "  ")
	if err != nil {
		fmt.Println("Error marshaling JSON:", err)
		return
	}

	log.Println("Message:", string(jsonBytes))

	for _, item := range y {
		fmt.Printf("AZ: %s, Price: %s\n", item.AZ, item.Price)
	}
}
