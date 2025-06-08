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
	metrics *StageMetrics
	sem     chan struct{}
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
		sem:     make(chan struct{}, 1),
		metrics: NewStageMetrics(),
	}
}

// Start begins processing items in the stage
func (s *Stage) Start(ctx context.Context, wg *sync.WaitGroup) error {
	if s.Config.WorkerFunc == nil && !s.Config.IsGenerator {
		return fmt.Errorf("worker function not set")
	}

	if s.Config.IsGenerator && s.Config.ItemGenerator == nil {
		return fmt.Errorf("generator function not set")
	}

	if s.Config.IsGenerator {
		s.initializeGenerators(wg)
	} else {
		s.initializeWorkers(wg)
	}

	return nil
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

func (s *Stage) generatorWorker(wg *sync.WaitGroup) {
	defer func() {
		select {
		case s.sem <- struct{}{}:
			close(s.Output)
		default:
		}
		s.metrics.Stop()
		wg.Done()
	}()

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
				s.metrics.RecordGenerated()
				continue
			}

			s.processRegularGeneration()
			s.metrics.RecordGenerated()
		}
	}
}

// worker processes items from the input channel
func (s *Stage) worker(wg *sync.WaitGroup) {
	defer func() {
		select {
		case s.sem <- struct{}{}:
			close(s.Output)
		default:
		}
		s.metrics.Stop()
		wg.Done()
	}()

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
					s.Output <- fmt.Errorf("error propagated from stage %s", s.Name)
				} else {
					s.Output <- fmt.Errorf("error from stage %s", s.Name)
				}

				continue
			}

			result, err := s.processWorkerItem(item)
			if err != nil {
				continue
			}

			if s.Config.IsFinal {
				fmt.Println("Stage", s.Name, "is final")
				log.Println("Final stage, item processed and dropped")
			} else {
				s.handleWorkerOutput(result)
				log.Println("Non-final stage, item processed and sent to output")
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
	return s.metrics
}
