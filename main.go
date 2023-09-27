package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/exp/slog"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/dustin/go-humanize"
	"github.com/taylormonacelli/lemondrop"
)

type AZs []AZPrice

type AzPrices []AZPrice

var logger *slog.Logger

type AZPrice struct {
	AZ           string  `json:"az"`
	Price        float64 `json:"price"`
	InstanceType string
	Region       string
}

var regions lemondrop.RegionDetails

func runCommand(logger *slog.Logger, ctx context.Context, cfg aws.Config, input *ec2.DescribeSpotPriceHistoryInput, resultsChan chan<- AzPrices) {
	client := ec2.NewFromConfig(cfg)

	resp, err := client.DescribeSpotPriceHistory(ctx, input)
	if err != nil {
		logger.Error(err.Error(), "region", cfg.Region, "regionDesc", regions[cfg.Region].RegionDesc)
		return
	}
	var azs AzPrices

	for _, price := range resp.SpotPriceHistory {
		s, err := strconv.ParseFloat(*price.SpotPrice, 64)
		if err != nil {
			panic(err)
		}

		azs = append(azs, AZPrice{
			AZ:           *price.AvailabilityZone,
			Region:       cfg.Region,
			Price:        s,
			InstanceType: string(price.InstanceType),
		})
	}

	resultsChan <- azs
}

func main() {
	var instanceTypes string

	flag.StringVar(&instanceTypes, "instanceTypes", "", "Comma-separated list of instance types to query")

	flag.Parse()

	logLevel := &slog.LevelVar{} // INFO
	logLevel.Set(slog.LevelInfo)
	opts := slog.HandlerOptions{
		AddSource: true,
		Level:     logLevel,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey && len(groups) == 0 {
				return slog.Attr{}
			}
			return a
		},
	}
	handler1 := slog.NewTextHandler(os.Stdout, &opts)

	// default logger customized
	slog.SetDefault(slog.New(handler1))

	handler2 := slog.NewTextHandler(os.Stderr, &opts)

	logger = slog.New(handler2)

	// Check if "instanceTypes" is empty and exit with an error if it is
	if instanceTypes == "" {
		fmt.Println("Error: The 'instanceTypes' flag is required.")
		flag.Usage()
		os.Exit(1)
	}

	instanceTypeSlice := strings.Split(instanceTypes, ",")

	var err error

	regions, err = lemondrop.GetRegionDetails()
	if err != nil {
		logger.Error(err.Error())
	}

	instTypesJsonString, err := json.Marshal(instanceTypeSlice)
	if err != nil {
		logger.Warn("Error marshaling JSON: ", err)
		return
	}

	// Print the JSON string
	logger.Debug(string(instTypesJsonString))
	logger.Debug("regions", "count", len(regions))

	// List of substrings to search for
	substrings := []string{"us-gov", "cn-"}
	logger.Debug("regions", "exclude prefix", substrings)

	// Create a filtered map
	filteredMap := make(lemondrop.RegionDetails)

	for key, value := range regions {
		found := false

		for _, prefix := range substrings {
			if strings.HasPrefix(key, prefix) {
				found = true
				break
			}
		}

		// If none of the substrings were found in the key, add it to the filtered map
		if !found {
			filteredMap[key] = value
		}
	}

	regions = filteredMap

	resultsChan := make(chan AzPrices, len(regions)*len(instanceTypeSlice))

	// Define the maximum number of concurrent workers
	maxConcurrent := 10

	// Use a semaphore pattern to limit concurrent goroutines
	semaphore := make(chan struct{}, maxConcurrent)

	var wg sync.WaitGroup

	for _, regionDetail := range regions {
		logger.Debug("regions loop", "region", regionDetail.RegionCode)
		cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(regionDetail.RegionCode))
		if err != nil {
			logger.Error("Error loading AWS configuration:", err)
		}

		var instanceTypeFilters []types.Filter

		instanceTypeFilter := types.Filter{
			Name:   aws.String("instance-type"),
			Values: instanceTypeSlice,
		}
		instanceTypeFilters = append(instanceTypeFilters, instanceTypeFilter)

		input := ec2.DescribeSpotPriceHistoryInput{
			Filters:             instanceTypeFilters,
			ProductDescriptions: []string{"Linux/UNIX"},
			StartTime:           aws.Time(time.Now()),
		}

		wg.Add(1)

		// Acquire a slot in the semaphore
		semaphore <- struct{}{}

		go func() {
			defer wg.Done()

			// Ensure that we release the semaphore slot even in case of a panic
			defer func() {
				<-semaphore
			}()

			timeoutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			runCommand(logger, timeoutCtx, cfg, &input, resultsChan)
		}()
	}

	go func() {
		// Wait for all goroutines to finish
		wg.Wait()

		// Close the resultsChan when all goroutines are done
		close(resultsChan)
	}()

	var y AZs
	for azs := range resultsChan {
		y = append(y, azs...)
	}

	sortByPrice := func(i, j int) bool {
		price1 := y[i].Price
		price2 := y[j].Price
		return price1 > price2
	}

	sort.Slice(y, sortByPrice)

	for _, item := range y {
		r := humanize.FormatFloat("#,###.###", item.Price)
		regionDetail := regions[item.Region]
		fmt.Printf("$%s [%s] %s %s %s %s\n", r, regionDetail.RegionDesc, item.Region, item.AZ, item.InstanceType, time.Now().Format(time.RFC3339))
	}
}
