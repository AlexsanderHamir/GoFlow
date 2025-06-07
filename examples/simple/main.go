package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/AlexsanderHamir/GoFlow/pkg/simulator"
)

func main() {
	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Create simulator
	sim := simulator.NewSimulator(ctx)

	// Create configuration for stages
	config := simulator.DefaultConfig()
	config.RoutineNum = 2                       // 2 workers per stage
	config.BufferSize = 10                      // Buffer size of 10
	config.WorkerDelay = 100 * time.Millisecond // Simulate some work
	config.ErrorRate = 0.1                      // 10% error rate
	config.RetryCount = 1                       // Retry once on error

	// Create stages
	stage1 := simulator.NewStage("processor", config)
	stage2 := simulator.NewStage("validator", config)

	// Set worker functions
	stage1.WorkerFunc = func(item any) (any, error) {
		// Simulate processing
		num, ok := item.(int)
		if !ok {
			return nil, fmt.Errorf("invalid item type")
		}
		return num * 2, nil
	}

	stage2.WorkerFunc = func(item any) (any, error) {
		// Simulate validation
		num, ok := item.(int)
		if !ok {
			return nil, fmt.Errorf("invalid item type")
		}
		if num > 100 {
			return nil, fmt.Errorf("value too large: %d", num)
		}
		return num, nil
	}

	// Add stages to simulator
	if err := sim.AddStage(stage1); err != nil {
		log.Fatalf("Failed to add stage1: %v", err)
	}
	if err := sim.AddStage(stage2); err != nil {
		log.Fatalf("Failed to add stage2: %v", err)
	}

	// Start the simulation
	if err := sim.Start(); err != nil {
		log.Fatalf("Failed to start simulation: %v", err)
	}

	// Generate some input
	go func() {
		for i := 1; i <= 20; i++ {
			select {
			case <-ctx.Done():
				return
			case stage1.Input <- i:
				time.Sleep(50 * time.Millisecond) // Rate limit input
			}
		}
	}()

	// Collect results
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case result, ok := <-stage2.Output:
				if !ok {
					return
				}
				fmt.Printf("Result: %v\n", result)
			}
		}
	}()

	// Wait for simulation to complete
	<-ctx.Done()

	// Print metrics
	fmt.Println("\nStage 1 Metrics:")
	printMetrics(stage1.GetMetrics())
	fmt.Println("\nStage 2 Metrics:")
	printMetrics(stage2.GetMetrics())
}

func printMetrics(metrics *simulator.StageMetrics) {
	stats := metrics.GetStats()
	fmt.Printf("Processed Items: %d\n", stats["processed_items"])
	fmt.Printf("Error Rate: %.2f%%\n", stats["error_rate"].(float64)*100)
	fmt.Printf("Retry Rate: %.2f%%\n", stats["retry_rate"].(float64)*100)
	fmt.Printf("Drop Rate: %.2f%%\n", stats["drop_rate"].(float64)*100)
	fmt.Printf("Avg Processing Time: %v\n", stats["avg_processing"])
	fmt.Printf("Min Processing Time: %v\n", stats["min_processing"])
	fmt.Printf("Max Processing Time: %v\n", stats["max_processing"])
	fmt.Printf("Throughput: %.2f items/sec\n", stats["throughput"].(float64))
}
