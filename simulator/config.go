package simulator

import (
	"context"
	"time"
)

// StageConfig holds the configuration for a pipeline stage
type StageConfig struct {
	// Rate at which items are generated (generator only)
	InputRate time.Duration
	// Custom item generator function
	ItemGenerator func() any

	// Number of goroutines per stage
	RoutineNum int
	// Channel buffer size per stage
	BufferSize int
	// Simulated delay per item
	WorkerDelay time.Duration
	// Number of times to retry on error
	RetryCount int

	// Drop input if channel is full
	DropOnBackpressure bool

	// Core processing function
	// Worker function that processes each item
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
