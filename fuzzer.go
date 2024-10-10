package main

import (
	"os"
	"strconv"
	"time"
)

type FuzzerConfig struct {
	TimeBudget			int
	IterTimeBudget		int
	NumNodes			int
	Guider				Guider
	Mutator				Mutator
	RecordPath			string
	LogLevel			string
	BaseNetworkPort 	int
	BaseWorkingDir		string

	MutationsPerTrace	int
	SeedPopulation		int
	NumRequests			int
	NumCrashes			int
	MaxMessages			int
	ReseedFrequency		int

	ClusterConfig		*ClusterConfig
}

type Fuzzer struct {
	config			FuzzerConfig
	logger			*Logger

	cluster 		*Cluster
	fuzzerType 		string
	scheduleQueue	[]Trace
	stats			*Stats
}

func NewFuzzer(config FuzzerConfig, fuzzerType string) (*Fuzzer, error) {
	f := &Fuzzer{
		config:  config,
		logger:	 NewLogger(),
		fuzzerType: fuzzerType,
		scheduleQueue: make([]Trace, 0),
		cluster: NewCluster(config.ClusterConfig, NewLogger()),
		stats: &Stats{
			Coverages: make([]int, 0),
			RandomTraces: 0,
			MutatedTraces: 0,
		},
	}
	f.logger.SetLevel(config.LogLevel)

	// TODO
	if _, err := os.Stat(config.RecordPath); err == nil {
		os.RemoveAll(config.RecordPath)
	}
	os.MkdirAll(config.RecordPath, 0777)
	if _, err := os.Stat(config.BaseWorkingDir); err == nil {
		os.RemoveAll(config.BaseWorkingDir)
	}
	os.MkdirAll(config.BaseWorkingDir, 0777)

	return f, nil
}

func (f *Fuzzer) Reset(p string, guider Guider, mutator Mutator) {
	guider.Reset()
	f.config.Guider = guider
	f.config.Mutator = mutator
}

func (f *Fuzzer) Run() {
	// Setup

	fuzzing := true
	for fuzzing {
		select {
		case <-time.After(time.Duration(f.config.TimeBudget) * time.Hour):
			fuzzing = false
		default:
		}

		nodes := make([]string, 0)
		for i := 1; i <= f.config.NumNodes; i++ {
			nodes = append(nodes, strconv.Itoa(i))
		}

		// Get schedule
		var schedule Trace
		if len(f.scheduleQueue) > 0 {
			schedule = f.scheduleQueue[0]
			f.scheduleQueue = f.scheduleQueue[1:]
		} else {
			schedule = Trace{Choices: make([]Choice, 0)}
		}

		// Start network

		// Start cluster

		running := true
		mutated := false
		crashedNodes := make(map[string]bool)
		step := 0
		for running {
			select {
			case <-time.After(time.Duration(f.config.IterTimeBudget) * time.Second):
				running = false
			default:
			}
			
			var choice Choice
			// Random if schedule is empty
			if len(schedule.Choices) == 0 || step >= len(schedule.Choices){
				// Choose between nodes (from, to), client request, fault
				// Add the choice to schedule
			} else {
				choice = schedule.Choices[step]
			}

			// Execute choice

			step++
			time.Sleep(1 * time.Millisecond)
		}

		// Stop cluster

		// Stop and reset network

		// Get event trace

		// Get and write logs

		// Get coverage

		// Mutate and update schedule queue

		// Update stats
	}

}

func (f *Fuzzer) Cleanup() {}

