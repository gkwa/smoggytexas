package smoggytexas

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/dustin/go-humanize"
	"github.com/taylormonacelli/lemondrop"
)

type AZs []AZPrice

type AzPrices []AZPrice

type AZPrice struct {
	AZ           string  `json:"az"`
	Price        float64 `json:"price"`
	InstanceType string
	Region       string
}

var regions lemondrop.RegionDetails

func runPriceHistoryQuery(ctx context.Context, cfg *aws.Config, input *ec2.DescribeSpotPriceHistoryInput, resultsChan chan<- AzPrices) {
	client := ec2.NewFromConfig(*cfg)

	slog.Debug("runPriceHistoryQuery1", "region", cfg.Region)

	resp, err := client.DescribeSpotPriceHistory(ctx, input)
	if err != nil {
		slog.Error(err.Error(), "regionDesc", regions[cfg.Region].RegionDesc, "region", cfg.Region)
		return
	}

	slog.Debug("runPriceHistoryQuery2", "region", cfg.Region)

	var azs AzPrices

	slog.Debug("spot history records", "count", len(resp.SpotPriceHistory), "region", cfg.Region)

	for _, price := range resp.SpotPriceHistory {
		s, err := strconv.ParseFloat(*price.SpotPrice, 64)
		if err != nil {
			slog.Error("parse spotprice", "error", err.Error())
			panic(err)
		}

		azs = append(azs, AZPrice{
			AZ:           *price.AvailabilityZone,
			Region:       cfg.Region,
			Price:        s,
			InstanceType: string(price.InstanceType),
		})

		slog.Debug("price check", "price", s, "instanceType", string(price.InstanceType), "region", cfg.Region)
	}

	resultsChan <- azs
}

func getRegions(instanceTypeSlice, ignoreRegionsPrefixes []string, dryRun bool) (lemondrop.RegionDetails, error) {
	var err error

	if dryRun {
		// Return mock regions for dry run
		mockRegions := lemondrop.RegionDetails{
			"us-east-1": lemondrop.RegionComponents{
				RegionCode: "us-east-1",
				RegionDesc: "US East (N. Virginia)",
			},
			"us-west-2": lemondrop.RegionComponents{
				RegionCode: "us-west-2",
				RegionDesc: "US West (Oregon)",
			},
		}
		regions = filterOutRegionsWithPrefix(mockRegions, ignoreRegionsPrefixes)
		slog.Info("Using mock regions for dry run", "count", len(regions))
		return regions, nil
	}

	regions, err = lemondrop.GetRegionDetails()
	if err != nil {
		slog.Error(err.Error())
		return lemondrop.RegionDetails{}, err
	}

	instTypesJsonString, err := json.Marshal(instanceTypeSlice)
	if err != nil {
		slog.Warn("error marshaling JSON", "error", err.Error())
		return lemondrop.RegionDetails{}, err
	}

	slog.Debug("debug instance types", "instance_types", string(instTypesJsonString))
	slog.Debug("regions", "count", len(regions))

	regions = filterOutRegionsWithPrefix(regions, ignoreRegionsPrefixes)
	slog.Debug("regions to search", "regions", regions)
	return regions, nil
}

func Main(commaSepInstanceTypes, ignoreCommaSepRegions string, dryRun bool) int {
	instanceTypeSlice := strings.Split(commaSepInstanceTypes, ",")

	ignoreRegionsPrefixes := strings.Split(ignoreCommaSepRegions, ",")
	regions, err := getRegions(instanceTypeSlice, ignoreRegionsPrefixes, dryRun)
	if err != nil {
		slog.Error("fetching regions", "error", err.Error())
		return 1
	}

	if dryRun {
		slog.Info("Dry run completed successfully")
		slog.Info("Configuration validated",
			"instanceTypes", instanceTypeSlice,
			"regionsToSearch", len(regions),
			"ignoredRegionPrefixes", ignoreRegionsPrefixes)
		return 0
	}

	resultsChan := make(chan AzPrices, len(regions)*len(instanceTypeSlice))

	// Define the maximum number of concurrent workers
	maxConcurrent := 10
	semaphore := make(chan struct{}, maxConcurrent)

	var wg sync.WaitGroup

	for _, regionDetail := range regions {
		wg.Add(1)
		semaphore <- struct{}{} // Acquire a slot in the semaphore

		slog.Debug("regions loop", "region", regionDetail.RegionCode)

		var cfg aws.Config
		var input ec2.DescribeSpotPriceHistoryInput

		setupPriceHistoryQuery(&cfg, &input, &regionDetail, &instanceTypeSlice)

		go func() {
			defer func() {
				<-semaphore // release the semaphore
				wg.Done()
			}()

			timeoutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			runPriceHistoryQuery(timeoutCtx, &cfg, &input, resultsChan)
		}()
	}

	go func() {
		wg.Wait()
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

	return 0
}

func setupPriceHistoryQuery(cfg *aws.Config, input *ec2.DescribeSpotPriceHistoryInput, regionDetail *lemondrop.RegionComponents, instanceTypes *[]string) {
	var err error
	*cfg, err = config.LoadDefaultConfig(context.TODO(), config.WithRegion(regionDetail.RegionCode))
	if err != nil {
		slog.Error("loading AWS configuration", "error", err.Error())
	}

	var instanceTypeFilters []types.Filter
	instanceTypeFilter := types.Filter{
		Name:   aws.String("instance-type"),
		Values: *instanceTypes,
	}
	instanceTypeFilters = append(instanceTypeFilters, instanceTypeFilter)

	*input = ec2.DescribeSpotPriceHistoryInput{
		Filters:             instanceTypeFilters,
		ProductDescriptions: []string{"Linux/UNIX"},
		StartTime:           aws.Time(time.Now()),
	}
}

func filterOutRegionsWithPrefix(allRegions lemondrop.RegionDetails, excludeRegionPrefixes []string) lemondrop.RegionDetails {
	filteredMap := make(lemondrop.RegionDetails)

	if len(excludeRegionPrefixes) == 1 && excludeRegionPrefixes[0] == "" {
		return allRegions
	}

	for key, value := range allRegions {
		found := false

		for _, prefix := range excludeRegionPrefixes {
			if strings.HasPrefix(key, prefix) {
				found = true
				break
			}
		}

		// If none of the substrings were found in the key, add it to the filtered map
		if !found {
			slog.Debug("include region", "region", key)
			filteredMap[key] = value
		}
	}

	return filteredMap
}
