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
	Name string

	Input  chan any
	Output chan any
	Sem    chan struct{}

	Config  *StageConfig
	Metrics *StageMetrics
	IdleSpy *tracker.GoroutineManager

	IsFinal           bool
	MaxGeneratedItems int
	Stop              func()
	stopOnce          sync.Once
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
		IdleSpy: tracker.NewGoroutineManager(),
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

func (s *Stage) generatorWorker(wg *sync.WaitGroup) {
	burstCount := 0
	lastBurstTime := time.Now()

	id := s.IdleSpy.TrackGoroutineStart()
	defer s.IdleSpy.TrackGoroutineEnd(id)

	for {
		startTime := time.Now()
		select {
		case <-s.Config.Ctx.Done():
			s.IdleSpy.TrackSelectCase(fmt.Sprintf("generator_%d_ctx_done", id), time.Since(startTime), id)
			s.stageTermination(wg)
			return
		default:
			if s.MaxGeneratedItems > 0 && s.Metrics.GeneratedItems >= uint64(s.MaxGeneratedItems) {
				s.StopOnce()
				continue
			}

			s.IdleSpy.TrackSelectCase(fmt.Sprintf("generator_%d_default", id), time.Since(startTime), id)
			if s.shouldExecuteBurst(burstCount, lastBurstTime) {
				s.executeBurst(&burstCount, &lastBurstTime)
				continue
			}

			s.processRegularGeneration(id, startTime)
		}
	}
}

// worker processes items from the input channel
func (s *Stage) worker(wg *sync.WaitGroup) {
	var id tracker.GoroutineId

	defer func() {
		s.stageTermination(wg)
		s.IdleSpy.TrackGoroutineEnd(id)
	}()

	id = s.IdleSpy.TrackGoroutineStart()

	for {
		startTime := time.Now()
		select {
		case <-s.Config.Ctx.Done():
			s.IdleSpy.TrackSelectCase(fmt.Sprintf("worker_%d_ctx_done", id), time.Since(startTime), id)
			return
		case item, ok := <-s.Input:
			if !ok {
				return
			}

			s.IdleSpy.TrackSelectCase(fmt.Sprintf("worker_%d_input_select", id), time.Since(startTime), id)
			result, err := s.processWorkerItem(item)
			if err != nil {
				s.Metrics.RecordDropped()
				continue
			}

			if !s.IsFinal {
				s.handleWorkerOutput(result, id, startTime)
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
