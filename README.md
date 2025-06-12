# GoFlow

GoFlow is a pipeline visualizer that helps developers fine-tune their concurrent systems by providing statistics for each stage of the pipeline, making it easy to identify bottlenecks and performance issues.

![Example Pipeline Visualization](example.png)

## Table of Contents

- [Features](#features)
- [Installation](#installation)
- [Quick Start Example](#quick-start-example)
- [Concepts](#concepts)
- [Customization](#customization)
- [Running the Example](#running-the-example)
- [IMPORTANT](#important)

## Features

- Visualize concurrent pipeline stages stats.
- Configure each stage independently (buffer size, worker count, etc.)
- Identify bottlenecks and optimize throughput
- Plug in your own worker functions for realistic simulation

## Installation

```sh
go get github.com/AlexsanderHamir/GoFlow
```

## Quick Start Example

Below is a minimal example to get you started. Copy this into a file (e.g., `main.go`) in your project:

```go
package main

import (
	"log"
	"math/rand"
	"time"

	"github.com/AlexsanderHamir/GoFlow/pkg/simulator"
)

func main() {
	sim := simulator.NewSimulator()
	sim.Duration = 10 * time.Second // Or use sim.MaxGeneratedItems = 10000

	generatorConfig := &simulator.StageConfig{
		InputRate:   100 * time.Millisecond,
		RoutineNum:  100,
		BufferSize:  5000,
		IsGenerator: true,
		ItemGenerator: func() any {
			return rand.Intn(100)
		},
	}

	globalConfig := &simulator.StageConfig{
		RoutineNum: 100,
		BufferSize: 5000,
	}

	newStage := func(name string, sleep time.Duration) *simulator.Stage {
		stage := simulator.NewStage(name, globalConfig)
		stage.Config.WorkerFunc = func(item any) (any, error) {
			time.Sleep(sleep)
			return item.(int) + rand.Intn(100), nil
		}
		return stage
	}

	sim.AddStage(simulator.NewStage("Generator", generatorConfig))
	sim.AddStage(newStage("Stage-1", 10*time.Millisecond))
	sim.AddStage(newStage("Stage-2", 100*time.Millisecond))
	sim.AddStage(newStage("DummyStage", 200*time.Millisecond)) // DummyStage must be last

	if err := sim.Start(); err != nil {
		log.Fatalf("Failed to start simulator: %v", err)
	}
}
```

## Concepts

- **Simulator**: Orchestrates the pipeline and manages stages.
- **Stage**: Represents a processing step. Each stage can have its own configuration and worker function.
- **Generator**: The first stage, responsible for generating items into the pipeline.
- **DummyStage**: The last stage, responsible for consuming items and removing them from the pipeline.
- **StageConfig**: Configuration for each stage (buffer size, worker count, etc.).

## Customization

- You can define your own worker functions for each stage by setting `WorkerFunc` in the stage's config.
- You can control the number of items generated or the simulation duration (set only one: `Duration` or `MaxGeneratedItems`).
- Adjust `RoutineNum` and `BufferSize` for each stage to simulate different concurrency and buffering scenarios.

## Running the Example

1. Save the example code to `main.go`.
2. Run:
   ```sh
   go run main.go
   ```
3. Observe the output and statistics for each stage.

## IMPORTANT

- Ensure you are using Go 1.18 or newer.
- Only set one of `Duration` or `MaxGeneratedItems`.
- The `Generator` stage must be the first, and `DummyStage` must be the last, both stages must be named that way if you don't want to include their statistics.
- In case your function could error, configure the `RetryCount` field.
