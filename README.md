# GoFlow

[![Go Version](https://img.shields.io/badge/Go-1.24%2B-blue)](https://golang.org)
[![Go Report Card](https://goreportcard.com/badge/github.com/AlexsanderHamir/GoFlow)](https://goreportcard.com/report/github.com/AlexsanderHamir/GoFlow)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![GoDoc](https://godoc.org/github.com/AlexsanderHamir/GoFlow?status.svg)](https://godoc.org/github.com/AlexsanderHamir/GoFlow/pkg/simulator)
![Issues](https://img.shields.io/github/issues/AlexsanderHamir/GoFlow)
![Last Commit](https://img.shields.io/github/last-commit/AlexsanderHamir/GoFlow)
![Code Size](https://img.shields.io/github/languages/code-size/AlexsanderHamir/GoFlow)
![Version](https://img.shields.io/github/v/tag/AlexsanderHamir/GoFlow?sort=semver)

**GoFlow** is a tool for visualizing and tuning pipeline performance in Go. Easily experiment with buffer sizes, goroutine counts, and stage depth to find the optimal configuration for your system.

![Example Pipeline Visualization](example.png)

## Table of Contents

- [Features](#features)
- [Installation](#installation)
- [Quick Start Example](#quick-start-example)
- [Stats Explanation](#stats-explanation)
- [Output](#output)

## Features

- Visualize pipeline performance stats for each stage.
- Configure each stage independently (buffer size, goroutine count, input rate, etc.).
- Plug in your own main worker functions for accurate stats collection.

## Installation

```sh
go get github.com/AlexsanderHamir/GoFlow@latest
```

## Quick Start Example

Below is a minimal example to give you a basic idea of how to use the library, if you want a more robust example go to: [example.go](code_example/example.go)

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

	// Generator must be first, feeds items into the pipeline.
	sim.AddStage(simulator.NewStage("Generator", generatorConfig))

	// Your pipeline stages
	sim.AddStage(newStage("Stage-1", 10*time.Millisecond))
	sim.AddStage(newStage("Stage-2", 100*time.Millisecond))

	// DummyStage must be last, removes items from the pipeline.
	sim.AddStage(newStage("DummyStage", 200*time.Millisecond))

	// Generator and Dummy are essential for the simulation,
	// both have their stats ignore because their sole purpose
	// is to allow the pipeline to be tested.
	if err := sim.Start(); err != nil {
		log.Fatalf("Failed to start simulator: %v", err)
	}
}
```

## Stats Explanation

- **Processed**: Number of processed but not yet sent items.
- **Output**: Number of items sent to the next stage successfully.
- **Throughput**: Number of output items divided by the duration of the stage in the simulation.
- **Dropped**: Number of items dropped during cancelation when the simulation ends.
- **Δ%**: Percentage difference in comparison with the stage before the current one.
- **Goroutines**: The total amount of time the goroutine spent blocked, represented by a histogram.

## Output

The stats will be printed to the terminal, if you want to save to a file you can do the following:

```bash
go run main.go > output.txt
```

## Contributing

We welcome contributions! Before you start contributing, please ensure you have:

- **Go 1.24.3 or later** installed
- **Git** for version control
- Basic understanding of Go testing and benchmarking

### Quick Setup

```bash
# Fork and clone the repository
git clone https://github.com/AlexsanderHamir/GenPool.git
cd GenPool

# Install dependencies
go mod download
go mod tidy

# Run tests to verify setup
go test -v ./...
go test -bench=. ./...
```

### Development Guidelines

- Write tests for new functionality
- Run benchmarks to ensure no performance regressions
- Follow Go code style guidelines
- Update documentation for user-facing changes
- Ensure all tests pass before submitting PRs
