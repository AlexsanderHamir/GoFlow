package simulator

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// StageStats represents the statistics for a single stage
type StageStats struct { // CODE REVIEW
	StageName      string
	ProcessedItems uint64
	OutputItems    uint64
	Throughput     float64
	DroppedItems   uint64
	DropRate       float64
	GeneratedItems uint64
	ThruDiffPct    float64
	ProcDiffPct    float64
	isGenerator    bool
	IsFinal        bool
}

// processBurst handles sending a burst of items to the output channel
func (s *Stage) processBurst(items []any) {
	var processedItems int

	defer func() {
		if r := recover(); r != nil {
			s.metrics.RecordDroppedBurst(len(items) - processedItems)
		}
	}()

	for _, item := range items {
		select {
		case <-s.Config.ctx.Done():
			s.metrics.RecordDroppedBurst(len(items) - processedItems)
			return
		case s.output <- item:
			processedItems++
			s.metrics.RecordOutput()
		default:
			if s.Config.DropOnBackpressure {
				s.metrics.RecordDropped()
			} else {
				s.output <- item
				processedItems++
				s.metrics.RecordOutput()
			}
		}
	}
}

// shouldExecuteBurst determines if it's time to process a burst based on configuration and timing
func (s *Stage) shouldExecuteBurst(burstCount int, lastBurstTime time.Time) bool {
	if s.Config.InputBurst == nil || s.Config.BurstCountTotal <= 0 {
		return false
	}

	now := time.Now()
	return burstCount < s.Config.BurstCountTotal && now.Sub(lastBurstTime) >= s.Config.BurstInterval
}

// processRegularGeneration handles the regular item generation flow
func (s *Stage) processRegularGeneration() {
	defer func() {
		if r := recover(); r != nil {
			s.metrics.RecordDropped()
		}
	}()

	if s.Config.ItemGenerator == nil {
		return
	}

	if s.Config.InputRate > 0 {
		time.Sleep(s.Config.InputRate)
	}

	item := s.Config.ItemGenerator()
	s.metrics.RecordGenerated()
	select {
	case <-s.Config.ctx.Done():
		s.metrics.RecordDropped()
		return
	case s.output <- item:
		s.metrics.RecordOutput()
	default:
		if s.Config.DropOnBackpressure {
			s.metrics.RecordDropped()
		} else {
			s.output <- item
			s.metrics.RecordOutput()
		}
	}
}

// processWorkerItem handles the processing of a single item in the worker loop
func (s *Stage) processWorkerItem(item any) (any, error) {
	result, err := s.processItem(item)
	if result != nil {
		s.metrics.RecordProcessing()
	}

	return result, err
}

// handleWorkerOutput manages sending the processed item to the output channel with backpressure handling
func (s *Stage) handleWorkerOutput(result any) {
	defer func() {
		if r := recover(); r != nil {
			s.metrics.RecordDropped()
		}
	}()

	select {
	case <-s.Config.ctx.Done():
		s.metrics.RecordDropped()
		return
	case s.output <- result:
		s.metrics.RecordOutput()
	default:
		if s.Config.DropOnBackpressure {
			s.metrics.RecordDropped()
		} else {
			s.output <- result
			s.metrics.RecordOutput()
		}
	}
}

func (s *Stage) validateConfig() error {
	if s.Config.WorkerFunc == nil && !s.Config.isGenerator {
		return fmt.Errorf("worker function not set")
	}

	if s.Config.isGenerator && s.Config.ItemGenerator == nil {
		return fmt.Errorf("generator function not set")
	}

	return nil
}

func (s *Stage) initializeStages(wg *sync.WaitGroup) {
	if s.Config.isGenerator {
		s.initializeGenerators(wg)
	} else {
		s.initializeWorkers(wg)
	}
}

func (s *Stage) initializeGenerators(wg *sync.WaitGroup) {
	for range s.Config.RoutineNum {
		go s.generatorWorker(wg)
	}
}

func (s *Stage) initializeWorkers(wg *sync.WaitGroup) {
	for range s.Config.RoutineNum {
		go s.worker(wg)
	}
}

// processItem handles a single item with retries and delay if configured
func (s *Stage) processItem(item any) (any, error) {
	var lastErr error
	attempt := 0

	for {
		if s.Config.WorkerDelay > 0 {
			time.Sleep(s.Config.WorkerDelay)
		}

		result, err := s.Config.WorkerFunc(item)
		if err == nil {
			return result, nil
		}

		lastErr = err
		attempt++

		if attempt > s.Config.RetryCount {
			break
		}
	}

	return nil, lastErr
}

func (s *Stage) GetMetrics() *StageMetrics {
	return s.metrics
}

func (s *Stage) stageTermination(wg *sync.WaitGroup) {
	select {
	case s.sem <- struct{}{}:
		close(s.output)
		s.metrics.Stop()
	default:
	}

	wg.Done()
}

func (s *Stage) executeBurst(burstCount *int, lastBurstTime *time.Time) {
	items := s.Config.InputBurst()
	s.metrics.RecordGeneratedBurst(len(items))
	s.processBurst(items)
	*burstCount++
	*lastBurstTime = time.Now()
}

func collectStageStats(stage *Stage) StageStats {
	stats := stage.GetMetrics().GetStats()
	return StageStats{
		StageName:      stage.Name,
		ProcessedItems: getIntMetric(stats, "processed_items"),
		OutputItems:    getIntMetric(stats, "output_items"),
		Throughput:     getFloatMetric(stats, "throughput"),
		DroppedItems:   getIntMetric(stats, "dropped_items"),
		DropRate:       getFloatMetric(stats, "drop_rate") * 100,
		GeneratedItems: getIntMetric(stats, "generated_items"),
		isGenerator:    stage.Config.isGenerator,
		IsFinal:        stage.isFinal,
	}
}

func (s *Simulator) initializeStages() error {
	generator := s.stages[0]
	generator.maxGeneratedItems = s.MaxGeneratedItems
	generator.stop = s.Stop
	generator.Config.isGenerator = true

	lastStage := s.stages[len(s.stages)-1]
	lastStage.isFinal = true

	for i, stage := range s.stages {
		stage.Config.ctx = s.ctx

		s.wg.Add(stage.Config.RoutineNum)

		beforeLastStage := i < len(s.stages)-1
		if beforeLastStage {
			s.stages[i+1].input = stage.output
		}

		if err := stage.Start(s.ctx, &s.wg); err != nil {
			return fmt.Errorf("failed to start stage %s: %w", stage.Name, err)
		}

	}

	return nil
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

func computeDiffs(prev, curr *StageStats) (procDiffStr, thruDiffStr string) {
	procDiffStr = "-"
	thruDiffStr = "-"
	if prev == nil {
		return
	}

	// Skip Generator and DummyStage
	if curr.isGenerator || curr.IsFinal ||
		prev.isGenerator || prev.IsFinal {
		return
	}

	if prev.Throughput > 0 {
		diff := ((curr.Throughput - prev.Throughput) / prev.Throughput) * 100
		thruDiffStr = fmt.Sprintf("%+.2f", diff)
	}
	if prev.ProcessedItems > 0 {
		diff := ((float64(curr.ProcessedItems) - float64(prev.ProcessedItems)) / float64(prev.ProcessedItems)) * 100
		procDiffStr = fmt.Sprintf("%+.2f", diff)
	}

	return
}

func printHeader() {
	fmt.Printf("\n%-20s %12s %12s %12s %12s %12s %12s %12s\n",
		"Stage", "Processed", "Output", "Throughput", "Dropped", "Drop Rate %", "Proc Δ%", "Thru Δ%")
	fmt.Println(strings.Repeat("-", 114))
}

func printStageRow(stat *StageStats, procDiff, thruDiff string) {
	fmt.Printf("%-20s %12d %12d %12.2f %12d %12.2f %12s %12s\n",
		stat.StageName,
		stat.ProcessedItems,
		stat.OutputItems,
		stat.Throughput,
		stat.DroppedItems,
		stat.DropRate,
		procDiff,
		thruDiff,
	)
}
