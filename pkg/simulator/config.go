package simulator

import (
	"context"
	"time"
)

// SimulationConfig holds the configuration for a stage
type StageConfig struct {
	// Runtime control
	InputRate     time.Duration // Rate at which items are generated (generator only) // x
	ItemGenerator func() any    // Custom item generator function // x

	// Bursts & load spikes
	InputBurst    func() []any  // Generates input bursts at intervals // x
	BurstCount    int           // Total number of bursts to inject // x
	BurstInterval time.Duration // Minimum interval between bursts // x

	// Worker control
	RoutineNum         int           // Number of goroutines per stage // x
	BufferSize         int           // Channel buffer size // x
	WorkerDelay        time.Duration // Simulated delay per item // x
	ErrorRate          float64       // Probability of operations to fail // x
	RetryCount         int           // Number of times to retry on error // x
	DropOnBackpressure bool          // Drop input if channel is full // x
	IsGenerator        bool          // Whether the stage is a generator // x
	IsFinal            bool          // Whether the stage is the final stage // x
	PropagateErrors    bool          // Whether to propagate errors to the next stage // x

	WorkerFunc func(item any) (any, error) // x
	Ctx    context.Context
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
