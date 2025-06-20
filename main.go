package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
)

func main() {
	logLevel := "debug"
	numNodes := 3

	argsWithoutProg := os.Args[1:]
	seed, _ := strconv.Atoi(argsWithoutProg[0])

	os.Setenv("JAVA_TOOL_OPTIONS", "-javaagent:./jacocoagent.jar=output=file,destfile=./output/modelfuzz/jacoco/jacocoRun.exec,append=true,dumponexit=true")

	var wg sync.WaitGroup
	javaToolOptions := os.Getenv("JAVA_TOOL_OPTIONS")
	var destFile = ""
	var codeCoverage = false
	if javaToolOptions != "" {
		codeCoverage = true
		options := strings.Split(javaToolOptions, ",")
		for _, opt := range options {
			if strings.HasPrefix(opt, "destfile=") {
				destFile = strings.TrimPrefix(opt, "destfile=")
				break
			}
		}
	}

	var BaseWorkingDir = "./output/" + ModelFuzz.String()
	var jacocoFile = ""
	var jacocoOutput = ""
	if codeCoverage {
		jacocoFile = destFile
		jacocoOutput = BaseWorkingDir + "/jacoco/" + "jacocoOutput.xml"
	}

	var horizon = 200
	config := FuzzerConfig{
		// TimeBudget:			60,
		Horizon:           horizon,
		Iterations:        200,
		NumNodes:          numNodes,
		LogLevel:          logLevel,
		NetworkPort:       7074,           // + i,
		BaseWorkingDir:    BaseWorkingDir, // FuzzerType(i).String(),
		RatisDataDir:      "./data",
		jacocoFile:        jacocoFile,
		jacocoOutput:      jacocoOutput,
		MutationsPerTrace: 5, // 5 and 10
		SeedPopulation:    20,
		NumRequests:       20, // 20 and 30
		NumCrashes:        10, // 5 and 10
		MaxMessages:       20, // 20 and 30
		ReseedFrequency:   100,
		RandomSeed:        seed,

		ClusterConfig: &ClusterConfig{
			FuzzerType:          ModelFuzz, // FuzzerType(i),
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

	if codeCoverage {
		var baseJacoco = config.BaseWorkingDir + "/jacoco/"

		if _, err := os.Stat(baseJacoco); err == nil {
			os.RemoveAll(baseJacoco)
		}
		os.MkdirAll(baseJacoco, 0777)

		if _, err := os.Stat(config.jacocoFile); err == nil {
			os.Remove(config.jacocoFile)
		}
		os.Create(config.jacocoFile)
		if _, err := os.Stat(config.jacocoOutput); err == nil {
			os.Remove(config.jacocoOutput)
		}
		os.Create(config.jacocoOutput)
	}

	fuzzer, err := NewFuzzer(config, config.ClusterConfig.FuzzerType, stateCoverage)
	if err != nil {
		fmt.Errorf("Could not create fuzzer %e", err)
		return
	}

	wg.Add(1)
	go func() {
		fuzzer.Run()
		wg.Done()
	}()
	// }
	wg.Wait()

}
