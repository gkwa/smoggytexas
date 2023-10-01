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
	"github.com/taylormonacelli/goldbug"
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

	resp, err := client.DescribeSpotPriceHistory(ctx, input)
	if err != nil {
		slog.Error(err.Error(), "region", cfg.Region, "regionDesc", regions[cfg.Region].RegionDesc)
		return
	}
	var azs AzPrices

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
	}

	resultsChan <- azs
}

func getRegions(instanceTypeSlice, ignoreRegionsPrefixes []string) (lemondrop.RegionDetails, error) {
	var err error

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

func Main(commaSepInstanceTypes, ignoreCommaSepRegions string) int {
	goldbug.SetDefaultLogger()

	instanceTypeSlice := strings.Split(commaSepInstanceTypes, ",")

	ignoreRegionsPrefixes := strings.Split(ignoreCommaSepRegions, ",")
	regions, err := getRegions(instanceTypeSlice, ignoreRegionsPrefixes)
	if err != nil {
		slog.Error("fetching regions", "error", err.Error())
		return 1
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

		go func(regionDetail lemondrop.RegionComponents) {
			defer func() {
				<-semaphore // release the semaphore
				wg.Done()
			}()

			timeoutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			runPriceHistoryQuery(timeoutCtx, &cfg, &input, resultsChan)
		}(regionDetail)
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
		slog.Error("Error loading AWS configuration:", err)
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
