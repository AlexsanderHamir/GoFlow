package simulator

import (
	"context"
	"fmt"
	"strings"
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
	Quit   chan struct{} // Channel to signal simulation completion
	Wg     sync.WaitGroup
}

// NewSimulator creates a new simulator instance
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

	stage.Config.Ctx = s.Ctx

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
		if i == 0 {
			stage.MaxGeneratedItems = s.MaxGeneratedItems
			stage.Stop = s.Stop
		}

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

// Stop gracefully stops the simulation
func (s *Simulator) Stop() {
	s.Cancel()
}

// Done returns a channel that will be closed when the simulation is complete
func (s *Simulator) Done() <-chan struct{} {
	return s.Quit
}

func (s *Simulator) WaitForStats() {
	<-s.Done()

	s.PrintStats()
}

// GetStages returns a copy of the stages slice
func (s *Simulator) GetStages() []*Stage {
	s.Mu.RLock()
	defer s.Mu.RUnlock()

	return s.Stages
}

// getIntMetric safely retrieves an integer metric, returning 0 if nil
func getIntMetric(stats map[string]any, key string) uint64 {
	if val, ok := stats[key]; ok && val != nil {
		if intVal, ok := val.(uint64); ok {
			return intVal
		}
	}
	return 0
}

// getFloatMetric safely retrieves a float metric, returning 0.0 if nil
func getFloatMetric(stats map[string]any, key string) float64 {
	if val, ok := stats[key]; ok && val != nil {
		if floatVal, ok := val.(float64); ok {
			return floatVal
		}
	}
	return 0.0
}

func (s *Simulator) PrintStats() {
	stages := s.GetStages()
	stats := make([]StageStats, len(stages))

	// Collect stats for each stage
	for i, stage := range stages {
		stageStats := stage.GetMetrics().GetStats()
		stats[i] = StageStats{
			StageName:      stage.Name,
			ProcessedItems: getIntMetric(stageStats, "processed_items"),
			OutputItems:    getIntMetric(stageStats, "output_items"),
			Throughput:     getFloatMetric(stageStats, "throughput"),
			DroppedItems:   getIntMetric(stageStats, "dropped_items"),
			DropRate:       getFloatMetric(stageStats, "drop_rate") * 100,
			GeneratedItems: getIntMetric(stageStats, "generated_items"),
		}
	}

	// Calculate percentage differences for throughput and processed items
	for i := 1; i < len(stats); i++ {
		if stats[i].StageName == "Generator" || stats[i].StageName == "DummyStage" ||
			stats[i-1].StageName == "Generator" || stats[i-1].StageName == "DummyStage" {
			continue
		}
		if stats[i-1].Throughput > 0 {
			stats[i].ThruDiffPct = ((stats[i].Throughput - stats[i-1].Throughput) / stats[i-1].Throughput) * 100
		}
		if stats[i-1].ProcessedItems > 0 {
			stats[i].ProcDiffPct = ((float64(stats[i].ProcessedItems) - float64(stats[i-1].ProcessedItems)) / float64(stats[i-1].ProcessedItems)) * 100
		}
	}

	fmt.Printf("\n%-20s %12s %12s %12s %12s %12s %12s %12s\n",
		"Stage", "Processed", "Output", "Throughput", "Dropped", "Drop Rate %", "Proc Δ%", "Thru Δ%")
	fmt.Println(strings.Repeat("-", 114))

	for _, stat := range stats {
		thruDiffStr := "-"
		if stat.ThruDiffPct != 0 {
			thruDiffStr = fmt.Sprintf("%+.2f", stat.ThruDiffPct)
		}
		procDiffStr := "-"
		if stat.ProcDiffPct != 0 {
			procDiffStr = fmt.Sprintf("%+.2f", stat.ProcDiffPct)
		}
		fmt.Printf("%-20s %12d %12d %12.2f %12d %12.2f %12s %12s\n",
			stat.StageName,
			stat.ProcessedItems,
			stat.OutputItems,
			stat.Throughput,
			stat.DroppedItems,
			stat.DropRate,
			procDiffStr,
			thruDiffStr)
	}
	fmt.Println()
}

// StageStats represents the statistics for a single stage
type StageStats struct {
	StageName      string  `json:"stage_name"`
	ProcessedItems uint64  `json:"processed_items"`
	OutputItems    uint64  `json:"output_items"`
	Throughput     float64 `json:"throughput"`
	DroppedItems   uint64  `json:"dropped_items"`
	DropRate       float64 `json:"drop_rate"`
	GeneratedItems uint64  `json:"generated_items,omitempty"`
	ThruDiffPct    float64 `json:"-"`
	ProcDiffPct    float64 `json:"-"`
}
