package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"math/big"
	"net/http"
	"os"
	"sort"
	"time"
)


type ExecutionBlockResponse struct {
	Status string `json:"status"`
	Data   []struct {
		ProducerReward int64 `json:"producerReward"`
		TxCount        int   `json:"txCount"`
	} `json:"data"`
}

type Reward struct {
	BlockNumber    int64 `json:"blockNumber"`
	ProducerReward int64 `json:"reward"`
	TxCount        int   `json:"txCount"`
}

const resultsFile = "rewards.json"
const maxRetries = 10

func main() {
	BEACONCHAIN_API_KEY := os.Getenv("BEACONCHAIN_API_KEY")
	resultsFlag := flag.Bool("results", false, "Read rewards from rewards.json and produce math output")
	flag.Parse()

	if *resultsFlag {
		rewards, err := readRewardsFromFile()
		if err != nil {
			log.Fatal("Failed to read rewards:", err)
		}
		printMathResults(rewards)
		return
	}

	startBlock := 17000000
	endBlock := 17050000

	// Start from cached block if existing rewards file already exists.
	rewards, err := readRewardsFromFile()
	if err != nil {
		fmt.Println("Failed to read rewards:", err)
	} else if len(rewards) > 0 {
		startBlock = int(rewards[len(rewards)-1].BlockNumber) + 1
	}

	for blockNumber := startBlock; blockNumber <= endBlock; blockNumber++ {
		var blockReward int64
		var blockTxCount int
		var success bool

		for attempt := 0; attempt < maxRetries; attempt++ {
			url := fmt.Sprintf("https://beaconcha.in/api/v1/execution/block/%d?apikey=%s", blockNumber, BEACONCHAIN_API_KEY)
			resp, err := http.Get(url)
			if err != nil {
				fmt.Println("Error fetching block:", err)
				time.Sleep(time.Duration(attempt*attempt) * time.Second)
				continue
			}

			var blockResponse ExecutionBlockResponse
			err = json.NewDecoder(resp.Body).Decode(&blockResponse)
			resp.Body.Close()
			if err != nil {
				fmt.Println("Error decoding response:", err)
				time.Sleep(time.Duration(attempt*attempt) * time.Second)
				continue
			}

			if len(blockResponse.Data) == 0 {
				fmt.Printf("Unexpected response for block %d. Full response: %+v\n", blockNumber, blockResponse)
				time.Sleep(time.Duration(attempt*attempt) * time.Second)
				continue
			}

			blockReward = blockResponse.Data[0].ProducerReward
			blockTxCount = blockResponse.Data[0].TxCount
			success = true
			break
		}

		if success {
			blockRewardInEth := float64(blockReward) / 1e18
			reward := Reward{BlockNumber: int64(blockNumber), ProducerReward: blockReward, TxCount: blockTxCount}
			rewards = append(rewards, reward)
			writeRewardToFile(rewards)
			fmt.Println("Block", blockNumber, "\tTxCount:", blockTxCount, "\tReward:", blockRewardInEth, "ETH")
		} else {
			fmt.Println("Failed to fetch valid Reward for block", blockNumber)
		}

	}
}

func readRewardsFromFile() ([]Reward, error) {
	file, err := os.Open(resultsFile)
	if err != nil {
		return []Reward{}, err
	}
	defer file.Close()

	var rewards []Reward
	err = json.NewDecoder(file).Decode(&rewards)
	if err != nil {
		return []Reward{}, err
	}

	return rewards, nil
}

func writeRewardToFile(rewards []Reward) {
	data, err := json.Marshal(rewards)
	if err != nil {
		fmt.Println("Error marshaling rewards:", err)
		return
	}

	err = ioutil.WriteFile(resultsFile, data, 0644)
	if err != nil {
		fmt.Println("Error writing rewards to file:", err)
	}
}

func printMathResults(rewards []Reward) {
	values := []int64{}
	zeroRewardCount := 0

	for _, reward := range rewards {
		values = append(values, reward.ProducerReward)
		if reward.ProducerReward == 0 {
			zeroRewardCount++
		}
	}

	emptyBlockRewards := []int64{}
	for i, reward := range rewards {
		if reward.TxCount == 0 && i < len(rewards)-1 {
			// If the block is empty and not the last block
			emptyBlockRewards = append(emptyBlockRewards, rewards[i+1].ProducerReward)
		}
	}

	zeroRewardPercentage := float64(zeroRewardCount) / float64(len(rewards)) * 100
	fmt.Println("\nNumber of Zero Rewards:", zeroRewardCount)
	fmt.Println("Percentage of Blocks with Zero Rewards:", zeroRewardPercentage, "%")

	printStatistics("\nStatistics for all blocks:", values)
	printStatistics("\nStatistics for blocks following empty blocks:", emptyBlockRewards)
}

func printStatistics(title string, rewards []int64) {
    meanBigFloat := calculateMean(rewards)
    mean, _ := meanBigFloat.Float64()
    median := calculateMedian(rewards) / 1e18
    stdDev := calculateStdDev(rewards, mean * 1e18)
    min, max := calculateRange(rewards)

    fmt.Println(title)
    fmt.Println("Mean:", mean, "ETH")
    fmt.Println("Median:", median, "ETH")
    fmt.Println("Standard Deviation:", stdDev / 1e18, "ETH")
    fmt.Println("Range:", min, "-", max, "ETH")
}

func calculateMean(rewards []int64) *big.Float {
	sum := big.NewInt(0)
	for _, reward := range rewards {
		sum.Add(sum, big.NewInt(reward))
	}
	mean := new(big.Float).SetInt(sum)
	mean.Quo(mean, big.NewFloat(float64(len(rewards))))
	mean.Quo(mean, big.NewFloat(1e18))
	return mean
}

func calculateMedian(values []int64) float64 {
	sort.Slice(values, func(i, j int) bool { return values[i] < values[j] })
	middle := len(values) / 2
	if len(values)%2 == 0 {
		return float64(values[middle-1]+values[middle]) / 2
	}
	return float64(values[middle])
}

func calculateStdDev(values []int64, mean float64) float64 {
    var sum float64
    for _, value := range values {
        sum += math.Pow(float64(value)/1e18-mean, 2)  // Convert value from wei to ETH
    }
    return math.Sqrt(sum / float64(len(values)))
}


func calculateRange(values []int64) (float64, float64) {
	min := float64(values[0]) / 1e18
	max := float64(values[0]) / 1e18
	for _, value := range values[1:] {
		valueInEth := float64(value) / 1e18
		if valueInEth < min {
			min = valueInEth
		}
		if valueInEth > max {
			max = valueInEth
		}
	}
	return min, max
}
