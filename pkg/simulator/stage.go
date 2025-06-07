package simulator

import (
	"context"
	"fmt"
	"math/rand/v2"
	"sync"
	"time"
)

// Stage represents a processing stage in the pipeline
type Stage struct {
	Name       string
	Input      chan any
	Output     chan any
	WorkerFunc func(item any) (any, error)
	Config     *StageConfig
	wg         sync.WaitGroup

	metrics *StageMetrics
}

// NewStage creates a new stage with the given configuration
func NewStage(name string, config *StageConfig) *Stage {
	if config == nil {
		config = DefaultConfig()
	}

	return &Stage{
		Name:    name,
		Output:  make(chan any, config.BufferSize),
		Config:  config,
		metrics: NewStageMetrics(),
	}
}

// Start begins processing items in the stage
func (s *Stage) Start(ctx context.Context) error {
	if s.WorkerFunc == nil {
		return fmt.Errorf("worker function not set")
	}

	if s.Config.IsGenerator && s.Config.ItemGenerator == nil {
		return fmt.Errorf("generator function not set")
	}

	s.wg.Add(s.Config.RoutineNum)
	if s.Config.IsGenerator {
		s.initializeGenerators(ctx)
	} else {
		s.initializeWorkers(ctx)
	}

	return nil
}

func (s *Stage) initializeGenerators(ctx context.Context) {
	s.wg.Add(s.Config.RoutineNum)
	for range s.Config.RoutineNum {
		go s.generatorWorker(ctx)
	}
}

func (s *Stage) initializeWorkers(ctx context.Context) {
	s.wg.Add(s.Config.RoutineNum)
	for range s.Config.RoutineNum {
		go s.worker(ctx)
	}
}

func (s *Stage) generatorWorker(ctx context.Context) {
	defer s.wg.Done()

	burstCount := 0
	lastBurstTime := time.Now()

	for {
		select {
		case <-ctx.Done():
			return
		default:
			if s.shouldProcessBurst(burstCount, lastBurstTime) {
				items := s.Config.InputBurst()
				s.processBurst(items)
				burstCount++
				lastBurstTime = time.Now()
				continue
			}

			s.processRegularGeneration()
		}
	}
}

// worker processes items from the input channel
func (s *Stage) worker(ctx context.Context) {
	defer s.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case item, ok := <-s.Input:
			if !ok {
				return
			}

			if s.Config.ErrorRate > 0 && rand.Float64() < s.Config.ErrorRate {
				s.metrics.RecordError(true)

				if s.Config.PropagateErrors {
					s.Output <- fmt.Errorf("error propagated from stage %s", s.Name)
				}

				continue
			}

			result, err := s.processWorkerItem(item)
			if err != nil {
				s.metrics.RecordError(false)
				continue
			}

			s.handleWorkerOutput(result)
		}
	}
}

// processItem handles a single item with retries if configured
func (s *Stage) processItem(item any) (any, error) {
	var lastErr error
	for attempt := 0; attempt <= s.Config.RetryCount; attempt++ {
		if s.Config.WorkerDelay > 0 {
			time.Sleep(s.Config.WorkerDelay)
		}

		result, err := s.WorkerFunc(item)
		if err == nil {
			return result, nil
		}

		lastErr = err
	}

	return nil, lastErr
}

// GetMetrics returns a copy of the current stage metrics
func (s *Stage) GetMetrics() *StageMetrics {
	return s.metrics.Clone()
}
