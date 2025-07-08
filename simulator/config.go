package simulator

import (
	"context"
	"time"
)

// StageConfig holds the configuration for a pipeline stage,
// it can be shared among all pipelines.
type StageConfig struct {

	// Rate at which items are generated (generator only)
	InputRate time.Duration

	// Custom item generator function  (generator only)
	ItemGenerator func() any

	// Number of goroutines per stage
	RoutineNum int

	// Channel buffer size per stage
	BufferSize int

	// Simulated delay per item
	WorkerDelay time.Duration

	// Number of times to retry on error, since your custom function
	// could fail.
	RetryCount int

	// Drop input if channel is full, when not set to drop it will block
	// in case the channels are full.
	DropOnBackpressure bool

	// Custom worker function that processes each item
	WorkerFunc func(item any) (any, error)

	// Context for cancellation and deadlines
	ctx context.Context
}

// DefaultConfig returns a new SimulationConfig with sensible defaults
func DefaultConfig() *StageConfig {
	return &StageConfig{
		RoutineNum:         1,
		BufferSize:         1,
		RetryCount:         0,
		DropOnBackpressure: false,
	}
}
