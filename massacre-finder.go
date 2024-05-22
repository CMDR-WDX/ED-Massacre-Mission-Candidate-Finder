package main

import (
	"encoding/json"
	"fmt"
	"github.com/xxjwxc/gowp/workpool"
	"io/ioutil"
	"massacre-finder/args"
	"massacre-finder/dataBuilder"
	"massacre-finder/evaluation"
	"os"
	"sort"
	"strconv"
)

func main() {
	fmt.Println("Hello World!")

	config := args.Args{
		FilterOnlyRingedSource:                   true,
		MinSourceSystemCount:                     3,
		MaxOtherDestSystemsForSource:             0,
		MaxOtherDestSystemsForSourceAnarchyCount: 0,
		MinSourceStationCount:                    6,
		MaxDistanceInLsForStationToBeConsidered:  1000,
		ConsiderGroundBases:                      false,
		ConsiderOdysseySettlements:               false,
	}

	systemList := dataBuilder.GetOrCreateSystemData("./system_cache.json", "./galaxy_populated.json", false)
	sectoredData := dataBuilder.BuildSectoredData(systemList, config)

	var semaphore = make(chan int, 1)

	workerPool := workpool.New(10)

	var results = make([]evaluation.SystemEvaluationResult, 100)

	for _, sectorSystems := range sectoredData {
		for _, system := range sectorSystems {

			_system := system
			workerPool.Do(func() error {

				response, relevant := evaluation.EvaluateSystem(_system, sectoredData, config)
				semaphore <- 1

				if relevant {
					results = append(results, response)
				}
				<-semaphore
				return nil
			})
		}
	}

	workerPool.Wait() // Wait for all Systems to be parsed
	fmt.Println("Found " + strconv.Itoa(len(results)) + " Results.")
	// Sort results
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	countToDisplay := len(results)
	if countToDisplay > 10 {
		countToDisplay = 10
	}

	for i, entry := range results[:countToDisplay] {
		fmt.Println("[", i+1, "]: ", entry.SystemName, " @ ", entry.Score)
	}

	buildAndWriteResult(config, results)

}

type Result struct {
	Config       args.Args                           `json:"config"`
	SortedResult []evaluation.SystemEvaluationResult `json:"sortedResult,omitempty"`
}

func buildAndWriteResult(args args.Args, result []evaluation.SystemEvaluationResult) Result {

	returnVal := Result{
		Config:       args,
		SortedResult: result,
	}

	jsonString, _ := json.MarshalIndent(returnVal, "", "\t")

	ioutil.WriteFile("./result.json", jsonString, os.ModePerm)
	return returnVal
}
