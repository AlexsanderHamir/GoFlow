package simulator

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Stage represents a processing stage in the pipeline
type Stage struct {
	Name          string
	Input         chan any
	Output        chan any
	WorkerFunc    func(item any) (any, error)
	Config        *StageConfig
	wg            sync.WaitGroup

	metrics   		*StageMetrics
}

// NewStage creates a new stage with the given configuration
func NewStage(name string, config *StageConfig) *Stage {
	if config == nil {
		config = DefaultConfig()
	}

	return &Stage{
		Name:          name,
		Output:        make(chan any, config.BufferSize),
		Config:        config,
		metrics:       NewStageMetrics(),
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

func (s *Stage) generatorWorker(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			s.Output <- s.Config.ItemGenerator()
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

			startTime := time.Now()

			result, err := s.processItem(item)
			processingTime := time.Since(startTime)
			s.metrics.RecordProcessing(processingTime, err)
			if err != nil {
				if s.Config.OnError != nil {
					s.Config.OnError(err, item)
				}
				continue
			}

			// Try to send the result, respecting backpressure settings
			select {
			case s.Output <- result:
				s.metrics.RecordOutput()
			default:
				if s.Config.DropOnBackpressure {
					s.metrics.RecordDropped()
				}
			}
		}
	}
}

// processItem handles a single item with retries if configured
func (s *Stage) processItem(item any) (any, error) {
	var lastErr error
	for attempt := 0; attempt <= s.Config.RetryCount; attempt++ {
		// Simulate worker delay if configured
		if s.Config.WorkerDelay > 0 {
			time.Sleep(s.Config.WorkerDelay)
		}

		// Simulate random errors if configured
		if s.Config.ErrorRate > 0 && shouldSimulateError(s.Config.ErrorRate) {
			lastErr = fmt.Errorf("simulated error on attempt %d", attempt+1)
			continue
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

// shouldSimulateError determines if we should simulate an error based on the error rate
func shouldSimulateError(rate float64) bool {
	// TODO: Implement proper random error simulation
	return false
}
