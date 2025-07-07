package simulator

import (
	"context"
	"sync"
	"time"

	"github.com/AlexsanderHamir/IdleSpy/tracker"
)

// Stage represents a processing stage in the pipeline
type Stage struct {
	Name string

	Input  chan any
	Output chan any
	Sem    chan struct{}

	Config  *StageConfig
	Metrics *StageMetrics

	IsFinal           bool
	MaxGeneratedItems int
	Stop              func()
	stopOnce          sync.Once

	gm *tracker.GoroutineManager
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
		Sem:     make(chan struct{}, 1),
		Metrics: NewStageMetrics(),
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
		case <-s.Config.Ctx.Done():
			return
		default:
			if s.MaxGeneratedItems > 0 && s.Metrics.GeneratedItems >= uint64(s.MaxGeneratedItems) {
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
		case <-s.Config.Ctx.Done():
			return
		case item, ok := <-s.Input:
			s.gm.TrackSelectCase(s.Name, time.Since(startTime), id)
			if !ok {
				return
			}

			result, err := s.processWorkerItem(item)
			if err != nil {
				s.Metrics.RecordDropped()
				continue
			}

			if !s.IsFinal {
				s.handleWorkerOutput(result)
			} else {
				s.Metrics.RecordDropped()
			}
		}
	}
}

func (s *Stage) StopOnce() {
	s.stopOnce.Do(func() {
		s.Stop()
	})
}
