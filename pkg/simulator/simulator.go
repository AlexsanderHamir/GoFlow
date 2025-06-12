package simulator

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Simulator represents a concurrent pipeline simulator
type Simulator struct {
	// Duration of the simulation
	Duration time.Duration
	// MaxGeneratedItems is the maximum number of items to generate.
	// If set, the simulation will run until the number of generated items is reached instead of the duration.
	MaxGeneratedItems int

	Stages []*Stage
	Mu     sync.RWMutex
	Ctx    context.Context
	Cancel context.CancelFunc
	Quit   chan struct{}
	Wg     sync.WaitGroup
}

// NewSimulator creates a new simulator instance with a context and a cancel function
func NewSimulator() *Simulator {
	ctx, cancel := context.WithCancel(context.Background())
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

func (s *Simulator) Stop() {
	s.Cancel()
}

func (s *Simulator) Done() <-chan struct{} {
	return s.Quit
}

func (s *Simulator) WaitForStats() {
	<-s.Done()
	s.PrintStats()
}

func (s *Simulator) GetStages() []*Stage {
	s.Mu.RLock()
	defer s.Mu.RUnlock()

	return s.Stages
}

func (s *Simulator) PrintStats() {
	stages := s.GetStages()
	printHeader()

	var prev *StageStats
	for _, stage := range stages {
		current := collectStageStats(stage)
		procDiff, thruDiff := computeDiffs(prev, &current)
		printStageRow(&current, procDiff, thruDiff)
		prev = &current
	}

	fmt.Println()
}
