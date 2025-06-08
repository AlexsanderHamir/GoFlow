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
	stages   []*Stage
	mu       sync.RWMutex
	ctx      context.Context
	cancel   context.CancelFunc
	done     chan struct{} // Channel to signal simulation completion
}

// NewSimulator creates a new simulator instance
func NewSimulator(ctx context.Context, cancel context.CancelFunc) *Simulator {
	return &Simulator{
		stages: make([]*Stage, 0),
		ctx:    ctx,
		cancel: cancel,
		done:   make(chan struct{}),
	}
}

// AddStage adds a new stage to the pipeline
func (s *Simulator) AddStage(stage *Stage) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if stage == nil {
		return fmt.Errorf("stage cannot be nil")
	}

	if stage.Name == "" {
		return fmt.Errorf("stage name cannot be empty")
	}

	for _, existingStage := range s.stages {
		if existingStage.Name == stage.Name {
			return fmt.Errorf("stage with name %s already exists", stage.Name)
		}
	}

	s.stages = append(s.stages, stage)
	return nil
}

// Start begins the simulation
func (s *Simulator) Start() error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.stages) == 0 {
		return fmt.Errorf("no stages to simulate")
	}

	for i, stage := range s.stages {
		if err := stage.Start(s.ctx); err != nil {
			return fmt.Errorf("failed to start stage %s: %w", stage.Name, err)
		}

		if i == len(s.stages)-1 {
			stage.Config.IsFinal = true
		}

		if i < len(s.stages)-1 {
			s.stages[i+1].Input = stage.Output
		}
	}

	go func() {
		time.Sleep(s.Duration)
		s.Stop()
		close(s.done) // Signal that the simulation is complete
	}()

	return nil
}

// Stop gracefully stops the simulation
func (s *Simulator) Stop() {
	s.cancel()
}

// Done returns a channel that will be closed when the simulation is complete
func (s *Simulator) Done() <-chan struct{} {
	return s.done
}

// GetStages returns a copy of the stages slice
func (s *Simulator) GetStages() []*Stage {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.stages
}
