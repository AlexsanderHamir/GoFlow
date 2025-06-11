package simulator

import (
	"fmt"
	"sync"
	"time"

	"github.com/AlexsanderHamir/IdleSpy/tracker"
)

// processBurst handles sending a burst of items to the output channel
func (s *Stage) processBurst(items []any, id tracker.GoroutineId) {
	var processedItems int

	defer func() {
		if r := recover(); r != nil {
			s.Metrics.RecordDroppedBurst(len(items) - processedItems)
		}
	}()

	for _, item := range items {
		startTime := time.Now()
		select {
		case <-s.Config.Ctx.Done():
			s.Metrics.RecordDroppedBurst(len(items) - processedItems)
			s.IdleSpy.TrackSelectCase("burst_ctx_done", time.Since(startTime), id)
			return
		case s.Output <- item:
			processedItems++
			s.Metrics.RecordOutput()
			s.IdleSpy.TrackSelectCase("burst_output_select", time.Since(startTime), id)
		default:
			if s.Config.DropOnBackpressure {
				s.Metrics.RecordDropped()
				s.IdleSpy.TrackSelectCase("burst_output_backpressure_default", time.Since(startTime), id)
			} else {
				s.Output <- item
				processedItems++
				s.Metrics.RecordOutput()
				s.IdleSpy.TrackSelectCase("burst_output_default", time.Since(startTime), id)
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
func (s *Stage) processRegularGeneration(id tracker.GoroutineId, startTime time.Time) {
	defer func() {
		if r := recover(); r != nil {
			s.Metrics.RecordDropped()
		}
	}()

	if s.Config.ItemGenerator == nil {
		return
	}

	if s.Config.InputRate > 0 {
		time.Sleep(s.Config.InputRate)
	}

	item := s.Config.ItemGenerator()
	s.Metrics.RecordGenerated()
	select {
	case <-s.Config.Ctx.Done():
		s.Metrics.RecordDropped()
		s.IdleSpy.TrackSelectCase("generation_ctx_done", time.Since(startTime), id)
		return
	case s.Output <- item:
		s.Metrics.RecordOutput()
		s.IdleSpy.TrackSelectCase("generation_output_select", time.Since(startTime), id)
	default:
		if s.Config.DropOnBackpressure {
			s.Metrics.RecordDropped()
			s.IdleSpy.TrackSelectCase("generation_backpressure_default", time.Since(startTime), id)
		} else {
			s.Output <- item
			s.IdleSpy.TrackSelectCase("generation_output_default", time.Since(startTime), id)
			s.Metrics.RecordOutput()
		}
	}
}

// processWorkerItem handles the processing of a single item in the worker loop
func (s *Stage) processWorkerItem(item any) (any, error) {
	result, err := s.processItem(item)
	if result != nil {
		s.Metrics.RecordProcessing()
	}

	return result, err
}

// handleWorkerOutput manages sending the processed item to the output channel with backpressure handling
func (s *Stage) handleWorkerOutput(result any, id tracker.GoroutineId, startTime time.Time) {
	defer func() {
		if r := recover(); r != nil {
			s.Metrics.RecordDropped()
		}
	}()

	select {
	case <-s.Config.Ctx.Done():
		s.Metrics.RecordDropped()
		s.IdleSpy.TrackSelectCase("worker_ctx_done", time.Since(startTime), id)
		return
	case s.Output <- result:
		s.Metrics.RecordOutput()
		s.IdleSpy.TrackSelectCase("worker_output_select", time.Since(startTime), id)
	default:
		if s.Config.DropOnBackpressure {
			s.Metrics.RecordDropped()
			s.IdleSpy.TrackSelectCase("worker_backpressure_default", time.Since(startTime), id)
		} else {
			s.Output <- result
			s.Metrics.RecordOutput()
			s.IdleSpy.TrackSelectCase("worker_output_default", time.Since(startTime), id)
		}
	}
}

// validateConfig validates the stage configuration
func (s *Stage) validateConfig() error {
	if s.Config.WorkerFunc == nil && !s.Config.IsGenerator {
		return fmt.Errorf("worker function not set")
	}

	if s.Config.IsGenerator && s.Config.ItemGenerator == nil {
		return fmt.Errorf("generator function not set")
	}

	return nil
}

// initialize initializes the stages, both generators and workers
func (s *Stage) initializeStages(wg *sync.WaitGroup) {
	if s.Config.IsGenerator {
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

// processItem handles a single item with retries if configured
func (s *Stage) processItem(item any) (any, error) {
	var lastErr error
	attempt := 0

	// Always process at least once
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
	return s.Metrics
}

func (s *Stage) stageTermination(wg *sync.WaitGroup) {
	select {
	case s.Sem <- struct{}{}:
		close(s.Output)
		s.Metrics.Stop()
		tracker.SaveStats(s.IdleSpy.Stats, fmt.Sprintf("goroutine_info_%s", s.Name))
	default:
	}

	wg.Done()
}

func (s *Stage) executeBurst(burstCount *int, lastBurstTime *time.Time, id tracker.GoroutineId) {
	items := s.Config.InputBurst()
	s.Metrics.RecordGeneratedBurst(len(items))
	s.processBurst(items, id)
	*burstCount++
	*lastBurstTime = time.Now()
}
