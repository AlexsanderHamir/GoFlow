package simulator

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/AlexsanderHamir/IdleSpy/tracker"
)

const graphFileName = "pipeline.dot"

// DataPresentationChoices are the current choices that the library offers for its output.
type DataPresentationChoices int

const (
	// DotFiles outputs DOT files for the entire pipeline, including a blocked time histogram (in DOT format) for each stage's goroutines.
	DotFiles DataPresentationChoices = iota
	// PrintToConsole will print the whole data to the console.
	PrintToConsole
	// Nothing is for test purposes, removes the log.
	Nothing
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
// [DataPresentationChoices]
//
// Validation rules:
//   - At least 3 stages if you want to collect stats
//   - The first stage will be interpreted as the generator.
//   - The last stage will be interpreted as the sink.
func (s *Simulator) Start(choice DataPresentationChoices) error {
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

	s.waitForStats(choice)

	return nil
}

// GetStages returns a copy of all stages in the pipeline.
// Getter used by test package
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

func (s *Simulator) waitForStats(choice DataPresentationChoices) {
	<-s.done()

	switch choice {
	case DotFiles:
		err := s.WritePipelineDot(graphFileName)
		if err != nil {
			panic(err)
		}
	case PrintToConsole:
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

	println()
	fmt.Println("================================")
	fmt.Println("Goroutine Blocked Time Histogram")
	fmt.Println("================================")

	first := 0
	last := len(stages) - 1
	for i, item := range allStages {
		if i == first || i == last {
			continue
		}
		tracker.PrintBlockedTimeHistogram(item.Stats, item.Label)
	}
}

// WritePipelineDot generates a Graphviz DOT representation of the pipeline
// and writes it to the given file path.
func (s *Simulator) WritePipelineDot(filename string) error {
	var b strings.Builder

	s.writeDotHeader(&b)

	if err := s.writeDotNodes(&b); err != nil {
		return err
	}

	s.writeDotEdges(&b)
	s.writeDotFooter(&b)

	return os.WriteFile(filename, []byte(b.String()), 0o644)
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
