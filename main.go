package main

import (
	"fmt"
	"os"
	"strconv"
	"sync"
)

func main() {
	logLevel := "INFO"
	numNodes := 5

	argsWithoutProg := os.Args[1:]
	seed, _ := strconv.Atoi(argsWithoutProg[0])
	fmt.Println("Random seed: " + argsWithoutProg[0])

	var wg sync.WaitGroup
	for i := 0; i <= 2; i++ {
		config := FuzzerConfig{
			// TimeBudget:			60,
			Horizon:			200,
			Iterations:			20000,
			NumNodes:			numNodes,
			LogLevel:			logLevel,
			NetworkPort: 		7074 + i,
			BaseWorkingDir:		"./output/" + FuzzerType(i).String(),
			MutationsPerTrace:	3,
			SeedPopulation:		20,
			NumRequests:		3,
			NumCrashes:			5,
			MaxMessages:		5,
			ReseedFrequency:	200,
			RandomSeed:			seed,

			ClusterConfig:		&ClusterConfig{
					FuzzerType:				FuzzerType(i),
					NumNodes:				numNodes,
					XraftBinaryPath:		"../xraft-controlled/xraft-kvstore/target/xraft-kvstore-0.1.0-SNAPSHOT-bin/xraft-kvstore-0.1.0-SNAPSHOT/bin/xraft-kvstore",
					XraftClientPath:		"../xraft-controlled/xraft-kvstore/target/xraft-kvstore-0.1.0-SNAPSHOT-bin/xraft-kvstore-0.1.0-SNAPSHOT/bin/xraft-kvstore-cli",
					BaseGroupPort:			2330 + (i * (numNodes + 1) * 100),
					BaseServicePort:	 	3330 + (i * (numNodes + 1) * 100),
					BaseInterceptorPort: 	7000 + (i * (numNodes + 1) * 100),
					LogLevel:				logLevel,
			},
			TLCPort:			2023,
		}

		if _, err := os.Stat(config.BaseWorkingDir); err == nil {
			os.RemoveAll(config.BaseWorkingDir)
		}
		os.MkdirAll(config.BaseWorkingDir, 0777)

		fuzzer, err := NewFuzzer(config, FuzzerType(i))
		if err != nil {
			fmt.Errorf("Could not create fuzzer %e", err)
			return
		}

		wg.Add(1)
		go func ()  {
			fuzzer.Run()
			wg.Done()
		} ()
	}
	wg.Wait()
	
}