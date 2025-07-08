package simulator

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/AlexsanderHamir/IdleSpy/tracker"
)

// Stage represents a processing stage in the pipeline
type Stage struct {
	Name   string
	Config *StageConfig

	input  chan any
	output chan any
	sem    chan struct{}

	metrics *StageMetrics

	isFinal     bool
	isGenerator bool

	stop     func()
	stopOnce sync.Once

	gm *tracker.GoroutineManager
}

// getter
func (s *Stage) GetisGenerator() bool {
	return s.isGenerator
}

// NewStage creates a new stage with the given configuration
func NewStage(name string, config *StageConfig) *Stage {
	if config == nil {
		config = DefaultConfig()
	}

	return &Stage{
		Name:    name,
		output:  make(chan any, config.BufferSize),
		Config:  config,
		sem:     make(chan struct{}, 1),
		metrics: NewStageMetrics(),
		gm:      tracker.NewGoroutineManager(),
	}
}

// Start initializes the workers and generators for all stages
func (s *Stage) Start(ctx context.Context, wg *sync.WaitGroup) error {
	if err := s.validateConfig(); err != nil {
		return err
	}

	s.initializeStages(wg)

	return nil
}

// generatorWorker is the worker for the generators
func (s *Stage) generatorWorker(wg *sync.WaitGroup) {
	defer s.stageTermination(wg)

	burstCount := 0
	lastBurstTime := time.Now()

	for {
		select {
		case <-s.Config.ctx.Done():
			return
		default:
			if s.shouldExecuteBurst(burstCount, lastBurstTime) {
				s.executeBurst(&burstCount, &lastBurstTime)
				continue
			}

			s.processRegularGeneration()
		}
	}
}

// worker is the worker for normal stages
func (s *Stage) worker(wg *sync.WaitGroup) {
	defer s.stageTermination(wg)

	id := s.gm.TrackGoroutineStart()
	defer s.gm.TrackGoroutineEnd(id)

	for {
		startTime := time.Now()
		select {
		case <-s.Config.ctx.Done():
			return
		case item, ok := <-s.input:
			s.gm.TrackSelectCase(s.Name, time.Since(startTime), id)
			if !ok {
				return
			}

			result, err := s.processWorkerItem(item)
			if err != nil {
				s.metrics.RecordDropped()
				continue
			}

			if !s.isFinal {
				s.handleWorkerOutput(result)
			} else {
				s.metrics.RecordDropped()
			}
		}
	}
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

// shouldExecuteBurst determines if it's time to process a burst based on configuration.
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

// handleWorkerOutput manages sending the processed item to the output channel with backpressure.
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
	if s.Config.WorkerFunc == nil && !s.isGenerator {
		return fmt.Errorf("worker function not set")
	}

	if s.isGenerator && s.Config.ItemGenerator == nil {
		return fmt.Errorf("generator function not set")
	}

	return nil
}

func (s *Stage) initializeStages(wg *sync.WaitGroup) {
	if s.isGenerator {
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
