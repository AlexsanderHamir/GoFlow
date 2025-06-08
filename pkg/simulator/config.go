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

	// Handles load spikes and burst patterns
	// Generates input bursts at intervals
	InputBurst func() []any
	// Total number of bursts to inject
	BurstCountTotal int
	// Interval between bursts
	BurstInterval time.Duration

	// Number of goroutines per stage
	RoutineNum int
	// Channel buffer size per stage
	BufferSize int
	// Simulated delay per item
	WorkerDelay time.Duration
	// Probability of operations to fail
	ErrorRate float64
	// Number of times to retry on error
	RetryCount int

	// Drop input if channel is full
	DropOnBackpressure bool
	// Whether to propagate errors to the next stage
	PropagateErrors bool
	// Whether the stage is a generator
	IsGenerator bool

	// Core processing function
	// Worker function that processes each item
	WorkerFunc func(item any) (any, error)
	// Context for cancellation and deadlines
	Ctx context.Context
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
