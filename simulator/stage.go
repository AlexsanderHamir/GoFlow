package simulator

import (
	"errors"
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

	metrics *stageMetrics

	isFinal     bool
	isGenerator bool

	stop func()

	gm *tracker.GoroutineManager
}

// GetIsGenerator is a getter.
func (s *Stage) GetIsGenerator() bool {
	return s.isGenerator
}

// NewStage creates a new stage with the provided config or creates a default one.
func NewStage(name string, config *StageConfig) *Stage {
	if config == nil {
		config = DefaultConfig()
	}

	return &Stage{
		Name:    name,
		output:  make(chan any, config.BufferSize),
		Config:  config,
		sem:     make(chan struct{}, 1),
		metrics: newStageMetrics(),
		gm:      tracker.NewGoroutineManager(),
	}
}

// generatorWorker is the worker for the generators
func (s *Stage) generatorWorker(wg *sync.WaitGroup) {
	defer s.stageTermination(wg)

	for {
		select {
		case <-s.Config.ctx.Done():
			return
		default:
			s.handleGeneration()
		}
	}
}

// worker is the worker for normal stages
func (s *Stage) worker(wg *sync.WaitGroup) {
	id := s.gm.TrackGoroutineStart()

	defer func() {
		s.stageTermination(wg)
		s.gm.TrackGoroutineEnd(id)
	}()

	for {
		startTime := time.Now()
		select {
		case <-s.Config.ctx.Done():
			return
		case item, ok := <-s.input:
			latency := time.Since(startTime)
			s.gm.TrackSelectCase(s.Name, latency, id)
			if !ok {
				return
			}

			if !s.isFinal {
				result, err := s.processItem(item)
				if err != nil {
					s.metrics.recordDropped()
					break
				}
				s.metrics.recordProcessed()

				s.sendOutput(result)
				break
			}

			s.metrics.recordDropped()
		}
	}
}

// processRegularGeneration handles the regular item generation flow
func (s *Stage) handleGeneration() {
	defer func() {
		if r := recover(); r != nil {
			s.metrics.recordDropped()
		}
	}()

	if s.Config.ItemGenerator == nil {
		return
	}

	if s.Config.InputRate > 0 {
		time.Sleep(s.Config.InputRate)
	}

	item := s.Config.ItemGenerator()
	s.metrics.recordGenerated()

	select {
	case <-s.Config.ctx.Done():
		s.metrics.recordDropped()
	case s.output <- item: // blocks
		s.metrics.recordOutput()
	default:
		if s.Config.DropOnBackpressure {
			s.metrics.recordDropped()
		} else {
			s.output <- item
			s.metrics.recordOutput()
		}
	}
}

// handleWorkerOutput manages sending the processed item to the output channel with backpressure.
func (s *Stage) sendOutput(result any) {
	defer func() {
		if r := recover(); r != nil {
			s.metrics.recordDropped()
		}
	}()

	select {
	case <-s.Config.ctx.Done():
		s.metrics.recordDropped()
		return
	case s.output <- result:
		s.metrics.recordOutput()
	default:
		if s.Config.DropOnBackpressure {
			s.metrics.recordDropped()
		} else {
			s.output <- result // blocks
			s.metrics.recordOutput()
		}
	}
}

func (s *Stage) validateConfig() error {
	cfg := s.Config

	if (!s.isGenerator && !s.isFinal) && cfg.WorkerFunc == nil {
		return errors.New("worker function must be set for non-generator stages")
	}

	if s.isGenerator && cfg.ItemGenerator == nil {
		return errors.New("ItemGenerator must be set for generator stage")
	}

	if cfg.RoutineNum <= 0 {
		return errors.New("routine number must be greater than 0")
	}

	if cfg.BufferSize < 0 {
		return errors.New("buffer size cannot be negative")
	}

	if s.isGenerator && cfg.InputRate < 0 {
		return errors.New("input rate cannot be negative for generator stages")
	}

	if cfg.RetryCount < 0 {
		return errors.New("retry count cannot be negative")
	}

	if cfg.ctx == nil {
		return errors.New("context must not be nil")
	}

	if s.Name == "" {
		return errors.New("stage name cannot be empty")
	}

	return nil
}

func (s *Stage) initializeStage(wg *sync.WaitGroup) {
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

		if attempt == s.Config.RetryCount {
			break
		}
	}

	return nil, lastErr
}

// GetMetrics is a getting.
// Used by the test package
func (s *Stage) GetMetrics() *stageMetrics {
	return s.metrics
}

// Only one worker will be able to close the channel and to
// stop the metric, all other workers will just decrement the counter.
func (s *Stage) stageTermination(wg *sync.WaitGroup) {
	// Instead of calling wg.Done() inside case and default, it was best
	// to do it outside the select.

	select {
	case s.sem <- struct{}{}:
		close(s.output)
		s.metrics.stop()
	default:
	}
	wg.Done()
}
