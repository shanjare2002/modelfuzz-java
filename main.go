package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

func main() {
	argsWithoutProg := os.Args[1:]
	seed, _ := strconv.Atoi(argsWithoutProg[0])

	logLevel := "debug"
	numNodes := 3
	fuzzerType := ModelFuzz
	strategy := CodeAndStateCoverage

	// Setup JaCoCo
	baseWorkingDir := "./output/" + fuzzerType.String()
	jacocoDestFile := baseWorkingDir + "/jacoco/jacocoRun.exec"
	jacocoAgent := fmt.Sprintf("-javaagent:./jacocoagent.jar=output=file,destfile=%s,append=true,dumponexit=true", jacocoDestFile)
	os.Setenv("JAVA_TOOL_OPTIONS", jacocoAgent)

	var wg sync.WaitGroup

	// Parse JaCoCo file
	javaToolOptions := os.Getenv("JAVA_TOOL_OPTIONS")
	var jacocoFile string
	for _, opt := range strings.Split(javaToolOptions, ",") {
		if strings.HasPrefix(opt, "destfile=") {
			jacocoFile = strings.TrimPrefix(opt, "destfile=")
			break
		}
	}

	// Configure both fuzzer and cluster
	var BaseWorkingDir = "./output/" + fuzzerType.String()
	jacocoOutput := BaseWorkingDir + "/jacoco/jacocoOutput.xml"
	config := FuzzerConfig{
		Horizon:           200,
		Iterations:        5,
		NumNodes:          numNodes,
		LogLevel:          logLevel,
		NetworkPort:       7074,
		BaseWorkingDir:    BaseWorkingDir,
		RatisDataDir:      "./data",
		jacocoFile:        jacocoFile,
		jacocoOutput:      jacocoOutput,
		MutationsPerTrace: 5,
		SeedPopulation:    20,
		NumRequests:       20,
		NumCrashes:        5,
		MaxMessages:       20,
		ReseedFrequency:   250,
		RandomSeed:        seed,

		ClusterConfig: &ClusterConfig{
			FuzzerType:          fuzzerType,
			NumNodes:            numNodes,
			ServerType:          Xraft,
			XraftServerPath:     "../xraft-controlled/xraft-kvstore/target/xraft-kvstore-0.1.0-SNAPSHOT-bin/xraft-kvstore-0.1.0-SNAPSHOT/bin/xraft-kvstore",
			XraftClientPath:     "../xraft-controlled/xraft-kvstore/target/xraft-kvstore-0.1.0-SNAPSHOT-bin/xraft-kvstore-0.1.0-SNAPSHOT/bin/xraft-kvstore-cli",
			RatisServerPath:     "../ratis-fuzzing/ratis-examples/target/ratis-examples-2.5.1.jar",
			RatisClientPath:     "../ratis-fuzzing/ratis-examples/target/ratis-examples-2.5.1.jar",
			RatisLog4jConfig:    "-Dlog4j.configuration=file:../ratis-fuzzing/ratis-examples/src/main/resources/log4j.properties",
			BaseGroupPort:       2330 + ((numNodes + 1) * 100), //(i * (numNodes + 1) * 100),
			BaseServicePort:     3330 + ((numNodes + 1) * 100), //(i * (numNodes + 1) * 100),
			BaseInterceptorPort: 7000 + ((numNodes + 1) * 100), //(i * (numNodes + 1) * 100),
			LogLevel:            logLevel,
		},
		TLCPort: 2023,
	}

	if _, err := os.Stat(config.BaseWorkingDir); err == nil {
		os.RemoveAll(config.BaseWorkingDir)
	}
	os.MkdirAll(config.BaseWorkingDir, 0777)

	fuzzer, err := NewFuzzer(config, config.ClusterConfig.FuzzerType, strategy)
	if err != nil {
		fmt.Errorf("Could not create fuzzer %e", err)
		return
	}

	wg.Add(1)
	go func() {
		fuzzer.Run()
		wg.Done()
	}()

	wg.Wait()

	// Ask user to move to corresponding directory
	fmt.Print("Move output to finalOutputs/" + strategy.String() + "? [y/n]: ")
	var input string
	fmt.Scanln(&input)
	if strings.ToLower(input) == "y" {

		// Create the strategy directory if it doesn't exist
		finalOutputsDir := "./finalOutputs/" + strategy.String()
		if _, err := os.Stat(finalOutputsDir); os.IsNotExist(err) {
			os.MkdirAll(finalOutputsDir, 0777)
		}

		// Get the output directory name from the user
		fmt.Print("How to name the output directory? ")
		var outputDir string
		fmt.Scanln(&outputDir)

		// Move the output directory to finalOutputs
		finalDir := "./finalOutputs/" + strategy.String() + "/" + outputDir
		os.RemoveAll(finalDir)
		baseDir := filepath.Dir(config.BaseWorkingDir)
		err := os.Rename(baseDir, finalDir)
		if err != nil {
			fmt.Printf("Failed to move output: %v\n", err)
		} else {
			fmt.Println("Output moved to", finalDir)
		}
	}
}
