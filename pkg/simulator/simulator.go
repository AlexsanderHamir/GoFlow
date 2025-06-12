package simulator

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
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

// cleanStaticDirectory removes all directories inside the static directory
func (s *Simulator) cleanStaticDirectory() error {
	// Get the absolute path of the current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current working directory: %w", err)
	}

	// Create the absolute path for the static directory
	staticDir := filepath.Join(cwd, "static")

	// Ensure the static directory exists
	if _, err := os.Stat(staticDir); os.IsNotExist(err) {
		return fmt.Errorf("static directory %s does not exist", staticDir)
	}

	// Read all entries in the static directory
	entries, err := os.ReadDir(staticDir)
	if err != nil {
		return fmt.Errorf("failed to read static directory: %w", err)
	}

	// Remove each directory
	for _, entry := range entries {
		if entry.IsDir() {
			dirPath := filepath.Join(staticDir, entry.Name())
			if err := os.RemoveAll(dirPath); err != nil {
				return fmt.Errorf("failed to remove directory %s: %w", dirPath, err)
			}
		}
	}

	return nil
}

// Start begins the simulation
func (s *Simulator) Start() error {
	// Clean the static directory before starting
	if err := s.cleanStaticDirectory(); err != nil {
		return fmt.Errorf("failed to clean static directory: %w", err)
	}

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

	err := s.WaitForStats()
	if err != nil {
		log.Fatalf("failed to wait for stats: %v", err)
	}

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

func (s *Simulator) WaitForStats() error {
	<-s.Done()

	err := s.SaveStats()
	if err != nil {
		return fmt.Errorf("failed to save stats: %w", err)
	}

	return nil
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

// StageStats represents the statistics for a single stage
type StageStats struct {
	StageName      string  `json:"stage_name"`
	ProcessedItems uint64  `json:"processed_items"`
	OutputItems    uint64  `json:"output_items"`
	Throughput     float64 `json:"throughput"`
	DroppedItems   uint64  `json:"dropped_items"`
	DropRate       float64 `json:"drop_rate"`
	GeneratedItems uint64  `json:"generated_items,omitempty"`
}

// SaveStats saves the statistics of each stage to a separate JSON file
func (s *Simulator) SaveStats() error {
	// Create static directory if it doesn't exist
	staticDir := "static"
	if err := os.MkdirAll(staticDir, 0755); err != nil {
		return fmt.Errorf("failed to create static directory: %w", err)
	}

	// Create stage directory under static if it doesn't exist
	stageDir := filepath.Join(staticDir, "stages")
	if err := os.MkdirAll(stageDir, 0755); err != nil {
		return fmt.Errorf("failed to create stages directory: %w", err)
	}

	for _, stage := range s.GetStages() {
		stats := stage.GetMetrics().GetStats()

		stageStats := StageStats{
			StageName:      stage.Name,
			ProcessedItems: getIntMetric(stats, "processed_items"),
			OutputItems:    getIntMetric(stats, "output_items"),
			Throughput:     getFloatMetric(stats, "throughput"),
			DroppedItems:   getIntMetric(stats, "dropped_items"),
			DropRate:       getFloatMetric(stats, "drop_rate") * 100,
		}

		if stage.Config.IsGenerator {
			stageStats.GeneratedItems = getIntMetric(stats, "generated_items")
		}

		// Marshal stats to JSON
		jsonData, err := json.MarshalIndent(stageStats, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal stats for stage %s: %w", stage.Name, err)
		}

		// Create or truncate the stats file in the stages directory
		filename := filepath.Join(stageDir, fmt.Sprintf("stage_%s_stats.json", stage.Name))
		if err := os.WriteFile(filename, jsonData, 0644); err != nil {
			return fmt.Errorf("failed to write stats file for stage %s: %w", stage.Name, err)
		}
	}

	return nil
}
