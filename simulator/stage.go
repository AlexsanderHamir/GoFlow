package simulator

import (
	"context"
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

	maxGeneratedItems int
	stop              func()
	stopOnce          sync.Once

	gm *tracker.GoroutineManager
}

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

// generatorWorker is the worker for generators
func (s *Stage) generatorWorker(wg *sync.WaitGroup) {
	defer s.stageTermination(wg)

	burstCount := 0
	lastBurstTime := time.Now()

	for {
		select {
		case <-s.Config.ctx.Done():
			return
		default:
			if s.maxGeneratedItems > 0 && s.metrics.generatedItems >= uint64(s.maxGeneratedItems) {
				s.StopOnce()
				continue
			}

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

// Ensures only one goroutine calls the cancel function
// once MaxGeneratedItems has been achieved.
func (s *Stage) StopOnce() {
	s.stopOnce.Do(func() {
		s.stop()
	})
}
