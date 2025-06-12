package code_example

import (
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/AlexsanderHamir/GoFlow/pkg/simulator"
)

func Example() {
	// Create simulator with a fixed duration
	sim := simulator.NewSimulator()
	sim.Duration = 10 * time.Second
	// Either duration or max generated items must be set, not both.
	// sim.MaxGeneratedItems = 10000

	// Generator stage configuration must be set
	generatorConfig := &simulator.StageConfig{
		InputRate:   100 * time.Millisecond,
		RoutineNum:  100,
		BufferSize:  5000,
		IsGenerator: true,
		ItemGenerator: func() any {
			return rand.Intn(100)
		},
	}

	// Common configuration for processing stages must be set
	globalConfig := &simulator.StageConfig{
		RoutineNum: 100,
		BufferSize: 5000,
	}

	// Helper to create a stage with a given name and sleep duration
	newStage := func(name string, sleep time.Duration) *simulator.Stage {
		stage := simulator.NewStage(name, globalConfig)
		stage.Config.WorkerFunc = func(item any) (any, error) {
			time.Sleep(sleep)
			return item.(int) + rand.Intn(100), nil
		}
		return stage
	}

	// Add stages to the simulator
	sim.AddStage(simulator.NewStage("Generator", generatorConfig)) // Generator must be first

	// Example pipeline with different sleep durations per stage
	sleepDurations := []time.Duration{
		10 * time.Millisecond,
		100 * time.Millisecond,
		300 * time.Millisecond,
		500 * time.Millisecond,
		80 * time.Millisecond,
		160 * time.Millisecond,
		320 * time.Millisecond,
	}
	for i, d := range sleepDurations {
		stageName := fmt.Sprintf("Stage-%d", i+1)
		sim.AddStage(newStage(stageName, d))
	}

	// DummyStage must be last
	dummy := newStage("DummyStage", 640*time.Millisecond)
	sim.AddStage(dummy)

	// Start the simulation
	if err := sim.Start(); err != nil {
		log.Fatalf("Failed to start simulator: %v", err)
	}
}
