package simulator

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
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

// getIntMetric safely retrieves an integer metric, returning 0 if nil
func getIntMetric(stats map[string]interface{}, key string) uint64 {
	if val, ok := stats[key]; ok && val != nil {
		if intVal, ok := val.(uint64); ok {
			return intVal
		}
	}
	return 0
}

// getFloatMetric safely retrieves a float metric, returning 0.0 if nil
func getFloatMetric(stats map[string]interface{}, key string) float64 {
	if val, ok := stats[key]; ok && val != nil {
		if floatVal, ok := val.(float64); ok {
			return floatVal
		}
	}
	return 0.0
}

func (s *Simulator) PrintStats() {
	for _, stage := range s.GetStages() {
		stats := stage.GetMetrics().GetStats()

		processedItems := getIntMetric(stats, "processed_items")
		outputItems := getIntMetric(stats, "output_items")
		droppedItems := getIntMetric(stats, "dropped_items")
		dropRate := getFloatMetric(stats, "drop_rate") * 100
		generatedItems := getIntMetric(stats, "generated_items")
		throughput := getFloatMetric(stats, "throughput")

		fmt.Printf("\n=== Stage: %s ===\n", stage.Name)
		fmt.Printf("Performance Metrics:\n")
		fmt.Printf("  • Processed Items: %d\n", processedItems)
		fmt.Printf("  • Output Items: %d\n", outputItems)
		fmt.Printf("  • Throughput: %.2f items/sec\n", throughput)
		fmt.Printf("  • Dropped Items: %d\n", droppedItems)
		fmt.Printf("  • Drop Rate: %.2f%%\n", dropRate)
		if stage.Config.IsGenerator {
			fmt.Printf("  • Generated Items: %d\n", generatedItems)
		}
		fmt.Println("===================")
	}
}

// SaveStageStats saves each stage's statistics to a separate file
func (s *Simulator) SaveStageStats() error {
	for _, stage := range s.GetStages() {
		stats := stage.GetMetrics().GetStats()

		// Create file for this stage
		filename := fmt.Sprintf("stage_info_%s.txt", stage.Name)
		file, err := os.Create(filename)
		if err != nil {
			return fmt.Errorf("failed to create stats file for stage %s: %w", stage.Name, err)
		}
		defer file.Close()

		// Write stats to file
		fmt.Fprintf(file, "=== Stage: %s ===\n", stage.Name)
		fmt.Fprintf(file, "Performance Metrics:\n")
		fmt.Fprintf(file, "  • Processed Items: %d\n", getIntMetric(stats, "processed_items"))
		fmt.Fprintf(file, "  • Output Items: %d\n", getIntMetric(stats, "output_items"))
		fmt.Fprintf(file, "  • Throughput: %.2f items/sec\n", getFloatMetric(stats, "throughput"))
		fmt.Fprintf(file, "  • Dropped Items: %d\n", getIntMetric(stats, "dropped_items"))
		fmt.Fprintf(file, "  • Drop Rate: %.2f%%\n", getFloatMetric(stats, "drop_rate")*100)
		if stage.Config.IsGenerator {
			fmt.Fprintf(file, "  • Generated Items: %d\n", getIntMetric(stats, "generated_items"))
		}
		fmt.Fprintf(file, "===================\n")
	}

	return nil
}

// SaveStats saves the statistics of each stage to a separate file
func (s *Simulator) SaveStats(outputDir string) error {
	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	for _, stage := range s.GetStages() {
		stats := stage.GetMetrics().GetStats()

		processedItems := getIntMetric(stats, "processed_items")
		outputItems := getIntMetric(stats, "output_items")
		droppedItems := getIntMetric(stats, "dropped_items")
		dropRate := getFloatMetric(stats, "drop_rate") * 100
		generatedItems := getIntMetric(stats, "generated_items")
		throughput := getFloatMetric(stats, "throughput")

		// Create or truncate the stats file
		filename := filepath.Join(outputDir, fmt.Sprintf("stage_%s_stats.txt", stage.Name))
		file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			return fmt.Errorf("failed to create/truncate stats file for stage %s: %w", stage.Name, err)
		}
		defer file.Close()

		// Write stats to file
		writer := bufio.NewWriter(file)
		fmt.Fprintf(writer, "=== Stage: %s ===\n", stage.Name)
		fmt.Fprintf(writer, "Performance Metrics:\n")
		fmt.Fprintf(writer, "  • Processed Items: %d\n", processedItems)
		fmt.Fprintf(writer, "  • Output Items: %d\n", outputItems)
		fmt.Fprintf(writer, "  • Throughput: %.2f items/sec\n", throughput)
		fmt.Fprintf(writer, "  • Dropped Items: %d\n", droppedItems)
		fmt.Fprintf(writer, "  • Drop Rate: %.2f%%\n", dropRate)
		if stage.Config.IsGenerator {
			fmt.Fprintf(writer, "  • Generated Items: %d\n", generatedItems)
		}
		fmt.Fprintf(writer, "===================\n")

		if err := writer.Flush(); err != nil {
			return fmt.Errorf("failed to write stats for stage %s: %w", stage.Name, err)
		}
	}

	return nil
}
