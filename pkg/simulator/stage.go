package simulator

import (
	"context"
	"fmt"
	"log"
	"math/rand/v2"
	"sync"
	"time"
)

// Stage represents a processing stage in the pipeline
type Stage struct {
	Name    string
	Input   chan any
	Output  chan any
	Config  *StageConfig
	wg      sync.WaitGroup
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
	if s.Config.WorkerFunc == nil && !s.Config.IsGenerator {
		return fmt.Errorf("worker function not set")
	}

	if s.Config.IsGenerator && s.Config.ItemGenerator == nil {
		return fmt.Errorf("generator function not set")
	}

	s.wg.Add(s.Config.RoutineNum)
	if s.Config.IsGenerator {
		s.initializeGenerators()
	} else {
		s.initializeWorkers()
	}

	return nil
}

func (s *Stage) initializeGenerators() {
	s.wg.Add(s.Config.RoutineNum)
	for range s.Config.RoutineNum {
		go s.generatorWorker()
	}
}

func (s *Stage) initializeWorkers() {
	s.wg.Add(s.Config.RoutineNum)
	for range s.Config.RoutineNum {
		go s.worker()
	}
}

func (s *Stage) generatorWorker() {
	defer s.wg.Done()

	burstCount := 0
	lastBurstTime := time.Now()

	for {
		select {
		case <-s.Config.Ctx.Done():
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
func (s *Stage) worker() {
	defer s.wg.Done()

	for {
		select {
		case <-s.Config.Ctx.Done():
			return
		case item, ok := <-s.Input:
			if !ok {
				return
			}

			log.Println("Item received from input")

			if s.Config.ErrorRate > 0 && rand.Float64() < s.Config.ErrorRate {
				if s.Config.PropagateErrors {
					s.metrics.RecordError(true)
					s.Output <- fmt.Errorf("error propagated from stage %s", s.Name)
				} else {
					s.metrics.RecordError(false)
					s.Output <- fmt.Errorf("error from stage %s", s.Name)
				}

				continue
			}

			result, err := s.processWorkerItem(item)
			if err != nil {
				s.metrics.RecordError(false)
				continue
			}

			if s.Config.IsFinal {
				log.Println("Final stage, item processed and dropped")
			} else {
				log.Println("Non-final stage, item processed and sent to output")
				s.handleWorkerOutput(result)
			}
		}
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

		// Only continue if we haven't exceeded retry count
		if attempt > s.Config.RetryCount {
			break
		}
	}

	return nil, lastErr
}

// GetMetrics returns a copy of the current stage metrics
func (s *Stage) GetMetrics() *StageMetrics {
	return s.metrics.Clone()
}
