# GoFlow

[![Go Version](https://img.shields.io/badge/Go-1.24%2B-blue)](https://golang.org)
[![Go Report Card](https://goreportcard.com/badge/github.com/AlexsanderHamir/GoFlow)](https://goreportcard.com/report/github.com/AlexsanderHamir/GoFlow)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![GoDoc](https://godoc.org/github.com/AlexsanderHamir/GoFlow?status.svg)](https://godoc.org/github.com/AlexsanderHamir/GoFlow/pkg/simulator)
![Issues](https://img.shields.io/github/issues/AlexsanderHamir/GoFlow)
![Last Commit](https://img.shields.io/github/last-commit/AlexsanderHamir/GoFlow)
![Code Size](https://img.shields.io/github/languages/code-size/AlexsanderHamir/GoFlow)
![Version](https://img.shields.io/github/v/tag/AlexsanderHamir/GoFlow?sort=semver)

**GoFlow** is a tool for visualizing and tuning the performance of your concurrent systems. It lets you easily experiment with buffer sizes, goroutine counts, and stage depth to find the optimal configuration.

![Example Pipeline Visualization](example.png)

## Table of Contents

- [Features](#features)
- [Installation](#installation)
- [Usage Example](#usage-example)
- [Stats Explanation](#stats-explanation)
- [Simulation Output](#output)
- [Contributions](#contributions)
- [Why GoFlow Was Created?](#why-goflow-was-created)

## Features

- Simulate realistic workloads flowing through your concurrent system.
- Visualize the performance of each stage to identify bottlenecks and optimize throughput.
- Configure each stage independently to improve the results.

## Installation

```sh
go get github.com/AlexsanderHamir/GoFlow@latest
```

## Usage Example

For a detailed example, see [example.go](code_example/example.go).

### Information

1. The first stage acts as the **generator**, feeding data into the pipeline. Configure it accordingly.
2. The last stage acts as the **sink**, consuming and discarding data.
3. Stats are printed to the terminal by default.
4. Multiple stages can share the same configuration.

## Stats Explanation

- **Processed**: Number of processed but not yet sent items.
- **Output**: Number of items sent to the next stage successfully.
- **Throughput**: Number of output items divided by the duration of the stage in the simulation.
- **Dropped**: Number of items dropped during cancelation when the simulation ends.
- **Δ%**: Percentage difference in comparison with the stage before the current one.
- **Goroutines**: The total amount of time the spent blocked per goroutine, represented by a histogram.

## Simulation Output

The stats will be printed to the terminal, if you want to save to a file you can do the following:

```bash
go run main.go > output.txt
```

## Contributions

We welcome contributions! Before you start contributing, please ensure you have:

- **Go 1.24.3 or later** installed
- **Git** for version control
- Basic understanding of Go testing and benchmarking

### Quick Setup

```bash
# Fork and clone the repository
git clone https://github.com/AlexsanderHamir/GoFlow.git
cd GoFlow

# Run tests to verify setup
 go test -v ./test

# Check for linter errors
 go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
 golangci-lint run
```

### Development Guidelines

- Write tests for new functionality
- Follow Go code style guidelines
- Update documentation for user-facing changes
- Ensure all medium / high priority tests pass before submitting PRs

## Why GoFlow Was Created?

I was building a query engine where each query plan was turned into a pipeline of stages created at runtime. Each stage processed batches of data, and performance depended on the buffer sizes and number of goroutines per stage.

I built GoFlow to experiment with these settings and easily visualize how they affect throughput and performance, in a way that other tools didn't allow me to do, to the same granularity.
