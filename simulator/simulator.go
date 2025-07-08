package simulator

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/AlexsanderHamir/IdleSpy/tracker"
)

// Simulator represents a concurrent pipeline simulator that orchestrates
// multiple processing stages in a data flow pipeline.
type Simulator struct {
	Duration time.Duration
	stages   []*Stage
	mu       sync.RWMutex
	ctx      context.Context
	cancel   context.CancelFunc
	quit     chan struct{}
	wg       sync.WaitGroup
}

// NewSimulator creates a new simulator for a specific pipeline.
func NewSimulator() *Simulator {
	ctx, cancel := context.WithCancel(context.Background())
	return &Simulator{
		ctx:    ctx,
		cancel: cancel,
		quit:   make(chan struct{}),
	}
}

// AddStage adds a new stage to the pipeline with validation.
//
// Validation rules:
//   - Stage cannot be nil
//   - Stage name cannot be empty
//   - Stage name must be unique within the pipeline
//   - Stage config cannot be nil
func (s *Simulator) AddStage(stage *Stage) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if stage == nil {
		return errors.New("stage cannot be nil")
	}

	if stage.Name == "" {
		return errors.New("stage name cannot be empty")
	}

	for _, existingStage := range s.stages {
		if existingStage.Name == stage.Name {
			return fmt.Errorf("repeated name not allowed: %s", stage.Name)
		}
	}

	if stage.Config == nil {
		return errors.New("must provide configuration")
	}

	s.stages = append(s.stages, stage)
	return nil
}

// Start begins the simulation and blocks until completion.
//
// Validation rules:
//   - At least 3 stages if you want to collect stats
//   - Only one stage will result in creating a generator and nothing else.
//   - Only two stages will result in having one generator and one dummy.
func (s *Simulator) Start(printStats bool) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.stages) < 3 {
		return fmt.Errorf("no stages to run")
	}

	if err := s.initializeStages(); err != nil {
		return fmt.Errorf("failed to initialize stages: %w", err)
	}

	go func() {
		if s.Duration > 0 {
			time.Sleep(s.Duration)
			s.stop()
		}

		s.wg.Wait()
		close(s.quit)
	}()

	s.waitForStats(printStats)

	return nil
}

// GetStages returns a copy of all stages in the pipeline.
// used by test package
func (s *Simulator) GetStages() []*Stage {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.stages
}

func (s *Simulator) stop() {
	s.cancel()
}

func (s *Simulator) done() <-chan struct{} {
	return s.quit
}

func (s *Simulator) waitForStats(printStats bool) {
	<-s.done()

	if printStats {
		s.printStats()
	}
}

type stateEntry struct {
	Stats map[tracker.GoroutineId]*tracker.GoroutineStats
	Label string
}

func (s *Simulator) printStats() {
	stages := s.GetStages()
	printHeader()

	var prev *stageStats
	allStages := []*stateEntry{}

	for _, stage := range stages {
		current := collectStageStats(stage)
		procDiff, thruDiff := computeDiffs(prev, &current)
		printStageRow(&current, procDiff, thruDiff)
		prev = &current
		entry := &stateEntry{
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

func (s *Simulator) initializeStages() error {
	generator := s.stages[0]
	generator.stop = s.stop
	generator.isGenerator = true

	lastStage := s.stages[len(s.stages)-1]
	lastStage.isFinal = true

	for i, stage := range s.stages {
		stage.Config.ctx = s.ctx

		s.wg.Add(stage.Config.RoutineNum)

		beforeLastStage := i < len(s.stages)-1
		if beforeLastStage {
			s.stages[i+1].input = stage.output
		}

		if err := stage.validateConfig(); err != nil {
			return err
		}

		stage.initializeStage(&s.wg)
	}

	return nil
}
