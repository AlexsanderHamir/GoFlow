package simulator

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Simulator represents a concurrent pipeline simulator
type Simulator struct {
	Duration time.Duration
	Stages   []*Stage
	Mu       sync.RWMutex
	Ctx      context.Context
	Cancel   context.CancelFunc
	Quit     chan struct{} // Channel to signal simulation completion
	Wg       sync.WaitGroup
}

// NewSimulator creates a new simulator instance
func NewSimulator(ctx context.Context, cancel context.CancelFunc) *Simulator {
	return &Simulator{
		Ctx:    ctx,
		Cancel: cancel,
		Quit:   make(chan struct{}),
	}
}

// AddStage adds a new stage to the pipeline
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

// Start begins the simulation
func (s *Simulator) Start() error {
	s.Mu.RLock()
	defer s.Mu.RUnlock()

	if len(s.Stages) == 0 {
		return fmt.Errorf("no stages to run")
	}

	for i, stage := range s.Stages {
		s.Wg.Add(stage.Config.RoutineNum)

		beforeLastStage := i < len(s.Stages)-1
		if beforeLastStage {
			s.Stages[i+1].Input = stage.Output
		}

		lastStage := i == len(s.Stages)-1
		if lastStage {
			stage.IsFinal = true
		}

		if err := stage.Start(s.Ctx, &s.Wg); err != nil {
			return fmt.Errorf("failed to start stage %s: %w", stage.Name, err)
		}
	}

	go func() {
		time.Sleep(s.Duration)
		s.Stop()
		s.Wg.Wait()
		close(s.Quit)
	}()

	return nil
}

// Stop gracefully stops the simulation
func (s *Simulator) Stop() {
	s.Cancel()
}

// Done returns a channel that will be closed when the simulation is complete
func (s *Simulator) Done() <-chan struct{} {
	return s.Quit
}

// GetStages returns a copy of the stages slice
func (s *Simulator) GetStages() []*Stage {
	s.Mu.RLock()
	defer s.Mu.RUnlock()

	return s.Stages
}

func (s *Simulator) PrintStats() {
	for _, stage := range s.GetStages() {
		fmt.Printf("\n=== Stage: %s ===\n", stage.Name)
		stats := stage.GetMetrics().GetStats()
		if !stage.Config.IsGenerator {
			fmt.Printf("Performance Metrics:\n")
			fmt.Printf("  • Processed Items: %d\n", stats["processed_items"])
			fmt.Printf("  • Throughput: %.2f items/sec\n", stats["throughput"])
			fmt.Printf("  • Drop Rate: %.2f%%\n", stats["drop_rate"].(float64)*100)
			fmt.Printf("  • Dropped Items: %d\n", stats["dropped_items"])
		} else {
			fmt.Printf("  • Generated Items: %d\n", stats["generated_items"])
			fmt.Printf("  • Dropped Items: %d\n", stats["dropped_items"])
			fmt.Printf("  • Drop Rate: %.2f%%\n", stats["drop_rate"].(float64)*100)
		}
		fmt.Println("===================")
	}
}
