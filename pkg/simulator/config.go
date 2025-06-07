package simulator

import (
	"context"
	"time"
)

// SimulationConfig holds the configuration for a stage
type StageConfig struct {	
	// Runtime control
	Duration      time.Duration // Total duration of simulation
	MaxItems      int           // Optional max number of items to process
	InputRate     time.Duration // Rate at which items are generated (first stage only)
	ItemGenerator func() any // Custom item generator function

	// Bursts & load spikes
	InputBurst    func() []any  // Generates input bursts at intervals
	BurstCount    int           // Total number of bursts to inject
	BurstInterval time.Duration // Minimum interval between bursts

	// Worker control
	RoutineNum         int           // Number of goroutines per stage
	BufferSize         int           // Channel buffer size
	WorkerDelay        time.Duration // Simulated delay per item
	ErrorRate          float64       // Probability of simulated processing error
	RetryCount         int           // Number of times to retry on error
	DropOnBackpressure bool          // Drop input if channel is full
	IsGenerator        bool          // Whether the stage is a generator

	// System integration
	OnError func(error, any) // Custom error handler
	Ctx     context.Context  // Optional cancellation control
}

// DefaultConfig returns a new SimulationConfig with sensible defaults
func DefaultConfig() *StageConfig {
	return &StageConfig{
		RoutineNum:         1,
		BufferSize:         1,
		RetryCount:         0,
		DropOnBackpressure: true,
	}
}
