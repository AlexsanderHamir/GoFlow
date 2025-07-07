package simulator

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/AlexsanderHamir/IdleSpy/tracker"
)

// Simulator represents a concurrent pipeline simulator that orchestrates
// multiple processing stages in a data flow pipeline.
//
// The simulator manages the lifecycle of all stages, coordinates data flow
// between them, and collects comprehensive performance metrics. It supports
// both time-based and item-count-based termination conditions.
type Simulator struct {
	// Duration specifies how long the simulation should run.
	// If set to a positive value, the simulation will automatically stop
	// after this duration. Mutually exclusive with MaxGeneratedItems.
	Duration time.Duration

	// MaxGeneratedItems is the maximum number of items to generate before stopping.
	// If set to a positive value, the simulation will stop once this many items
	// have been generated. Mutually exclusive with Duration.
	MaxGeneratedItems int

	// Stages contains all the processing stages in the pipeline, ordered
	// from first (generator) to last (final stage).
	Stages []*Stage

	// Mu protects access to the Stages slice and other shared state
	Mu sync.RWMutex

	// Ctx provides cancellation context for all stages
	Ctx context.Context

	// Cancel function to stop all stages gracefully
	Cancel context.CancelFunc

	// Quit channel is closed when the simulation completes
	Quit chan struct{}

	// Wg tracks all running goroutines for proper cleanup
	Wg sync.WaitGroup
}

func NewSimulator() *Simulator {
	ctx, cancel := context.WithCancel(context.Background())
	return &Simulator{
		Ctx:    ctx,
		Cancel: cancel,
		Quit:   make(chan struct{}),
	}
}

// AddStage adds a new stage to the pipeline with validation.
//
// The stage is added to the end of the pipeline. The first stage added
// should be a generator, the last stage should be the dummy, while subsequent stages
// should be processors.
//
// Args:
//   - stage: The stage to add to the pipeline
//
// Returns:
//   - error: nil if successful, or an error describing the validation failure
//
// Validation rules:
//   - Stage cannot be nil
//   - Stage name cannot be empty
//   - Stage name must be unique within the pipeline
func (s *Simulator) AddStage(stage *Stage) error {
	s.Mu.Lock()
	defer s.Mu.Unlock()

	if stage == nil {
		return fmt.Errorf("stage cannot be nil")
	}

	if stage.Name == "" {
		return fmt.Errorf("stage name cannot be empty")
	}

	for _, existingStage := range s.Stages {
		if existingStage.Name == stage.Name {
			return fmt.Errorf("stage with name %s already exists", stage.Name)
		}
	}

	s.Stages = append(s.Stages, stage)
	return nil
}

// Start begins the simulation and blocks until completion.
//
// This method initializes all stages, starts their goroutines, and waits
// for the simulation to complete based on the configured termination
// condition (Duration or MaxGeneratedItems).
//
// The simulation will automatically stop when:
//   - The configured Duration has elapsed (if Duration > 0)
//   - The configured MaxGeneratedItems have been generated (if MaxGeneratedItems > 0)
//   - Stop() is called explicitly
//
// Returns:
//   - error: nil if successful, or an error describing the failure
//
// Panics:
//   - If both Duration and MaxGeneratedItems are set to positive values
func (s *Simulator) Start() error {
	s.Mu.RLock()
	defer s.Mu.RUnlock()

	if len(s.Stages) == 0 {
		return fmt.Errorf("no stages to run")
	}

	if err := s.initializeStages(); err != nil {
		return fmt.Errorf("failed to initialize stages: %w", err)
	}

	go func() {
		if s.MaxGeneratedItems > 0 && s.Duration > 0 {
			panic("either duration or max generated items must be set, not both")
		}

		durationActive := s.MaxGeneratedItems <= 0 && s.Duration > 0
		if durationActive {
			time.Sleep(s.Duration)
			s.Stop()
		}

		s.Wg.Wait()
		close(s.Quit)
	}()

	s.WaitForStats()

	return nil
}

// Stop terminates the simulation by canceling the context.
func (s *Simulator) Stop() {
	s.Cancel()
}

// Done returns a channel that is closed when the simulation completes.
func (s *Simulator) Done() <-chan struct{} {
	return s.Quit
}

// WaitForStats blocks until the simulation completes and then prints statistics.
func (s *Simulator) WaitForStats() {
	<-s.Done()
	s.PrintStats()
}

// GetStages returns a copy of all stages in the pipeline.
func (s *Simulator) GetStages() []*Stage {
	s.Mu.RLock()
	defer s.Mu.RUnlock()

	return s.Stages
}

type StateEntry struct {
	Stats map[tracker.GoroutineId]*tracker.GoroutineStats
	Label string
}

// PrintStats displays comprehensive statistics for all stages in the pipeline.
//
// The statistics include:
//   - Processed items count
//   - Output items count
//   - Throughput (items per second)
//   - Dropped items count
//   - Drop rate percentage
//   - Generated items (for generator stages)
//   - Percentage changes between stages
//   - Histogram accounting for the total blocked time per goroutine
//
// The output is formatted as a table for easy reading and analysis.
func (s *Simulator) PrintStats() {
	stages := s.GetStages()
	printHeader()

	var prev *StageStats
	allStages := []*StateEntry{}
	for _, stage := range stages {
		current := collectStageStats(stage)
		procDiff, thruDiff := computeDiffs(prev, &current)
		printStageRow(&current, procDiff, thruDiff)
		prev = &current
		entry := &StateEntry{
			Stats: stage.gm.GetAllStats(),
			Label: stage.Name,
		}
		allStages = append(allStages, entry)
	}

	fmt.Println()

	for _, item := range allStages {
		tracker.PrintBlockedTimeHistogram(item.Stats, item.Label)
	}
}
