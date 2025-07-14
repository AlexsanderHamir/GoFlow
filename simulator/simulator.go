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

const GRAPH_FILE_NAME = "pipeline.dot"

// DataPresentationChoices are the current choices that the library offers for its output.
type DataPresentationChoices int

const (
	DotFiles DataPresentationChoices = iota
	Console
	Nothing // Test purposes
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
//   - The last stage will be interpreted as the dummy.
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
// getter used by test package
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
		err := s.WritePipelineDot(GRAPH_FILE_NAME)
		if err != nil {
			panic(err)
		}
	case Console:
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

// WritePipelineDot generates a Graphviz DOT representation of the pipeline
// and writes it to the given file path.
func (s *Simulator) WritePipelineDot(filename string) error {
	var prevStats *stageStats
	var b strings.Builder

	stages := s.GetStages()

	b.WriteString("digraph Pipeline {\n")
	b.WriteString("  rankdir=LR;\n")
	b.WriteString("  node [shape=box, style=filled, fontname=\"Arial\", fontsize=10];\n")
	b.WriteString("  edge [fontname=\"Arial\", fontsize=8];\n\n")

	for i, stage := range stages {
		currentStats := collectStageStats(stage)
		procDiffStr, thruDiffStr := computeDiffs(prevStats, &currentStats)
		prevStats = &currentStats

		var nodeColor string
		switch {
		case stage.isGenerator:
			nodeColor = "lightgreen"
		case stage.isFinal:
			nodeColor = "lightcoral"
		default:
			nodeColor = "lightblue"
		}

		label := fmt.Sprintf(`"%s\nRoutines: %d\nBuffer: %d\nProcessed: %d (%s)\nDroppedItems: %d\nOutput: %d\nThroughput: %.2f (%s)"`,
			stage.Name,
			stage.Config.RoutineNum,
			stage.Config.BufferSize,
			currentStats.ProcessedItems, procDiffStr,
			currentStats.DroppedItems,
			currentStats.OutputItems,
			currentStats.Throughput, thruDiffStr,
		)

		fmt.Fprintf(&b, "  stage_%d [label=%s, style=filled, fillcolor=%s];\n",
			i, label, nodeColor)

		goroutineStats := stage.gm.GetAllStats()
		err := tracker.WriteBlockedTimeHistogramDot(goroutineStats, stage.Name)
		if err != nil {
			return fmt.Errorf("goroutine tracker failed: %w", err)
		}
	}

	b.WriteString("\n")

	// Define edges
	for i := 0; i < len(stages)-1; i++ {
		fmt.Fprintf(&b, "  stage_%d -> stage_%d;\n", i, i+1)
	}

	b.WriteString("}\n")

	// Write to file
	return os.WriteFile(filename, []byte(b.String()), 0644)
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
