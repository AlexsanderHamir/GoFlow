package simulator

import (
	"context"
	"fmt"
	"sync"
)

// Simulator represents a concurrent pipeline simulator
type Simulator struct {
	stages []*Stage
	mu     sync.RWMutex
	ctx    context.Context
	cancel context.CancelFunc
}

// NewSimulator creates a new simulator instance
func NewSimulator(ctx context.Context) *Simulator {
	ctx, cancel := context.WithCancel(ctx)
	return &Simulator{
		stages: make([]*Stage, 0),
		ctx:    ctx,
		cancel: cancel,
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

		if i < len(s.stages)-1 {
			s.stages[i+1].Input = stage.Output
		}
	}

	return nil
}

// Stop gracefully stops the simulation
func (s *Simulator) Stop() {
	s.cancel()
}

// GetStages returns a copy of the stages slice
func (s *Simulator) GetStages() []*Stage {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stages := make([]*Stage, len(s.stages))
	copy(stages, s.stages)
	return stages
}
