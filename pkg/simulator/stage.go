package simulator

import (
	"context"
	"sync"
	"time"
)

// Stage represents a processing stage in the pipeline
type Stage struct {
	Name string

	Input  chan any
	Output chan any
	Sem    chan struct{}

	Config  *StageConfig
	Metrics *StageMetrics

	IsFinal bool
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

	for {
		select {
		case <-s.Config.Ctx.Done():
			s.stageTermination(wg)
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

// worker processes items from the input channel
func (s *Stage) worker(wg *sync.WaitGroup) {
	defer s.stageTermination(wg)

	for {
		select {
		case <-s.Config.Ctx.Done():
			return
		case item, ok := <-s.Input:
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
